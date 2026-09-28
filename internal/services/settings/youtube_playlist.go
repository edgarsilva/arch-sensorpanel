package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"sensorpanel/internal/models"
)

const youtubePlaylistAPIBaseURL = "https://www.googleapis.com/youtube/v3/playlistItems"
const youtubePlaylistMaxResults = 50
const youtubePlaylistMaxPages = 20

type playlistVideo struct {
	VideoID string
	Title   string
}

type youtubePlaylistItemsResponse struct {
	NextPageToken string `json:"nextPageToken"`
	Items         []struct {
		Snippet struct {
			Title      string `json:"title"`
			ResourceID struct {
				VideoID string `json:"videoId"`
			} `json:"resourceId"`
		} `json:"snippet"`
	} `json:"items"`
}

type youtubeErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (s *Service) expandPlaylistMediaSources(ctx context.Context, config *models.SettingsConfig) error {
	if config == nil {
		return nil
	}

	if normalizeMediaTypeValue(config.MediaType, config.MediaSources) != "playlist" {
		return nil
	}

	if len(config.MediaSources) == 0 || len(config.MediaSources) > 1 {
		return nil
	}

	seed := config.MediaSources[0]
	playlistID := extractPlaylistIDFromRaw(seed.URL)
	if playlistID == "" {
		return nil
	}
	apiKey := ""
	if s != nil && s.Server != nil && s.Server.Env != nil {
		apiKey = strings.TrimSpace(s.Server.Env.YouTubeAPIKey)
	}
	if apiKey == "" {
		return fmt.Errorf("%w: YOUTUBE_API_KEY is required to expand playlist media sources", ErrInvalidConfig)
	}

	videos, err := fetchPlaylistVideos(ctx, playlistID, apiKey)
	if err != nil {
		return fmt.Errorf("%w: failed to expand playlist %q: %v", ErrInvalidConfig, playlistID, err)
	}
	if len(videos) == 0 {
		return fmt.Errorf("%w: playlist %q returned no videos", ErrInvalidConfig, playlistID)
	}

	selectedID := extractVideoIDFromRaw(seed.URL)
	videos = rotateVideosToFront(videos, selectedID)

	expanded := make([]models.SettingsMediaSource, 0, len(videos))
	for i, video := range videos {
		expanded = append(expanded, models.SettingsMediaSource{
			Kind:  "video",
			URL:   buildPlaylistWatchURL(video.VideoID, playlistID, i+1),
			Label: strings.TrimSpace(video.Title),
		})
	}

	config.MediaSources = expanded
	return nil
}

func fetchPlaylistVideos(ctx context.Context, playlistID string, apiKey string) ([]playlistVideo, error) {
	client := &http.Client{Timeout: 12 * time.Second}
	videos := make([]playlistVideo, 0, youtubePlaylistMaxResults)
	pageToken := ""

	for page := 0; page < youtubePlaylistMaxPages; page++ {
		requestURL, err := buildYouTubePlaylistAPIURL(playlistID, apiKey, pageToken)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "sensorpanel/playlist-expander")

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("youtube api returned %s: %s", resp.Status, summarizeYouTubeAPIError(body))
		}

		var payload youtubePlaylistItemsResponse
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, err
		}

		for _, item := range payload.Items {
			videoID := strings.TrimSpace(item.Snippet.ResourceID.VideoID)
			if !isLikelyYouTubeVideoID(videoID) {
				continue
			}
			videos = append(videos, playlistVideo{
				VideoID: videoID,
				Title:   strings.TrimSpace(item.Snippet.Title),
			})
		}

		nextToken := strings.TrimSpace(payload.NextPageToken)
		if nextToken == "" || nextToken == pageToken {
			break
		}
		pageToken = nextToken
	}

	if len(videos) == 0 {
		return nil, fmt.Errorf("playlist has no accessible items")
	}

	return dedupePlaylistVideos(videos), nil
}

func buildYouTubePlaylistAPIURL(playlistID string, apiKey string, pageToken string) (string, error) {
	parsed, err := url.Parse(youtubePlaylistAPIBaseURL)
	if err != nil {
		return "", err
	}

	query := parsed.Query()
	query.Set("part", "snippet")
	query.Set("maxResults", fmt.Sprintf("%d", youtubePlaylistMaxResults))
	query.Set("playlistId", strings.TrimSpace(playlistID))
	query.Set("key", strings.TrimSpace(apiKey))
	if strings.TrimSpace(pageToken) != "" {
		query.Set("pageToken", strings.TrimSpace(pageToken))
	}
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

func summarizeYouTubeAPIError(body []byte) string {
	var payload youtubeErrorResponse
	if err := json.Unmarshal(body, &payload); err == nil {
		message := strings.TrimSpace(payload.Error.Message)
		if message != "" {
			return message
		}
	}

	text := strings.TrimSpace(string(body))
	if text == "" {
		return "unknown error"
	}
	if len(text) > 200 {
		return text[:200] + "..."
	}
	return text
}

func dedupePlaylistVideos(videos []playlistVideo) []playlistVideo {
	seen := make(map[string]struct{}, len(videos))
	out := make([]playlistVideo, 0, len(videos))
	for _, video := range videos {
		if !isLikelyYouTubeVideoID(video.VideoID) {
			continue
		}
		if _, ok := seen[video.VideoID]; ok {
			continue
		}
		seen[video.VideoID] = struct{}{}
		out = append(out, video)
	}
	return out
}

func rotateVideosToFront(videos []playlistVideo, selectedID string) []playlistVideo {
	selected := strings.TrimSpace(selectedID)
	if selected == "" || len(videos) < 2 {
		return videos
	}

	index := -1
	for i, video := range videos {
		if strings.TrimSpace(video.VideoID) == selected {
			index = i
			break
		}
	}
	if index <= 0 {
		return videos
	}

	rotated := make([]playlistVideo, 0, len(videos))
	rotated = append(rotated, videos[index:]...)
	rotated = append(rotated, videos[:index]...)
	return rotated
}

func extractPlaylistIDFromRaw(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	if strings.HasPrefix(trimmed, "PL") || strings.HasPrefix(trimmed, "UU") || strings.HasPrefix(trimmed, "OLAK5uy") {
		return trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(parsed.Query().Get("list"))
}

func extractVideoIDFromRaw(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	if len(trimmed) == 11 && isLikelyYouTubeVideoID(trimmed) {
		return trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}

	host := strings.ToLower(parsed.Hostname())
	if strings.Contains(host, "youtu.be") {
		candidate := strings.TrimPrefix(parsed.Path, "/")
		if isLikelyYouTubeVideoID(candidate) {
			return candidate
		}
	}

	if strings.Contains(host, "youtube.com") {
		fromQuery := strings.TrimSpace(parsed.Query().Get("v"))
		if isLikelyYouTubeVideoID(fromQuery) {
			return fromQuery
		}

		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		for i := 0; i < len(parts)-1; i++ {
			if parts[i] == "embed" || parts[i] == "shorts" || parts[i] == "live" {
				candidate := strings.TrimSpace(parts[i+1])
				if isLikelyYouTubeVideoID(candidate) {
					return candidate
				}
			}
		}
	}

	return ""
}

func isLikelyYouTubeVideoID(value string) bool {
	if len(value) != 11 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func buildPlaylistWatchURL(videoID string, playlistID string, index int) string {
	query := url.Values{}
	query.Set("v", strings.TrimSpace(videoID))
	query.Set("list", strings.TrimSpace(playlistID))
	if index > 0 {
		query.Set("index", fmt.Sprintf("%d", index))
	}
	return "https://www.youtube.com/watch?" + query.Encode()
}
