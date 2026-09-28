package settings

import (
	"testing"

	"sensorpanel/internal/models"
)

func TestNormalizeMediaTypeValueDetectsPlaylistFromSourceURL(t *testing.T) {
	t.Parallel()

	sources := []models.SettingsMediaSource{{
		Kind: "video",
		URL:  "https://www.youtube.com/watch?v=AKfsikEXZHM&list=PLfjAQvQm_beGbIqk1j0NxF4Mm11JsjGMf",
	}}

	got := normalizeMediaTypeValue("", sources)
	if got != "playlist" {
		t.Fatalf("expected media_type playlist, got %q", got)
	}
}

func TestApplyFieldPatchInfiniteVideoPlayback(t *testing.T) {
	t.Parallel()

	cfg := models.SettingsConfig{
		MediaType: "video",
		MediaSources: []models.SettingsMediaSource{{
			Kind: "video",
			URL:  "https://www.youtube.com/watch?v=AKfsikEXZHM",
		}},
	}

	if err := applyFieldPatch(&cfg, "infinite_video_playback", true); err != nil {
		t.Fatalf("applyFieldPatch(true) returned error: %v", err)
	}
	if !cfg.Layout.InfiniteVideoPlayback {
		t.Fatal("expected infinite_video_playback to be true")
	}

	if err := applyFieldPatch(&cfg, "infinite_video_playback", "false"); err != nil {
		t.Fatalf("applyFieldPatch(false) returned error: %v", err)
	}
	if cfg.Layout.InfiniteVideoPlayback {
		t.Fatal("expected infinite_video_playback to be false")
	}
}
