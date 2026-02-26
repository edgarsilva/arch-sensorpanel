package settings

import (
	"errors"
	"io/fs"
	"strconv"
	"strings"

	"sensorpanel/internal/models"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
)

type createSettingsInput struct {
	Config                models.SettingsConfig `json:"config"`
	ConfigName            string                `form:"config_name"`
	LayoutName            string                `form:"layout_name"`
	OverlayLayout         string                `form:"overlay_layout"`
	Theme                 string                `form:"theme"`
	VideoFit              string                `form:"video_fit"`
	VideoAlign            string                `form:"video_align"`
	VideoOffsetXPct       string                `form:"video_offset_x_pct"`
	VideoOffsetYPct       string                `form:"video_offset_y_pct"`
	InfiniteVideoPlayback string                `form:"infinite_video_playback"`
	OverlayPaddingTop     string                `form:"overlay_padding_top"`
	OverlayPaddingRight   string                `form:"overlay_padding_right"`
	OverlayPaddingBottom  string                `form:"overlay_padding_bottom"`
	OverlayPaddingLeft    string                `form:"overlay_padding_left"`
	MetricsScale          string                `form:"metrics_scale_pct"`
	MetricsOffsetX        string                `form:"metrics_offset_x"`
	MetricsOffsetY        string                `form:"metrics_offset_y"`
	MediaKind             string                `form:"media_kind"`
	MediaURL              string                `form:"media_url"`
	MediaLabel            string                `form:"media_label"`
}

func (s *Service) IndexPage(c fiber.Ctx) error {
	settingsHTML, err := fs.ReadFile(s.PublicFS, "settings.html")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "settings.html not found")
	}

	return c.Type("html").Send(settingsHTML)
}

func (s *Service) Index(c fiber.Ctx) error {
	if !wantsJSON(c) {
		return s.IndexPage(c)
	}

	rows, err := s.List(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	type settingsItem struct {
		ID        uint                  `json:"id"`
		Version   int64                 `json:"version"`
		IsCurrent bool                  `json:"is_current"`
		Config    models.SettingsConfig `json:"config"`
	}

	items := make([]settingsItem, 0, len(rows))
	for _, row := range rows {
		cfg, err := s.DecodeConfig(&row)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		items = append(items, settingsItem{
			ID:        row.ID,
			Version:   row.Version,
			IsCurrent: row.IsCurrent,
			Config:    cfg,
		})
	}

	return c.JSON(fiber.Map{"items": items})
}

func (s *Service) Get(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	if !wantsJSON(c) {
		return c.Redirect().To("/settings/" + strconv.FormatUint(id, 10) + "/edit")
	}

	row, err := s.GetByID(c.Context(), uint(id))
	if err != nil {
		if errors.Is(err, ErrSettingsNotFound) {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	cfg, err := s.DecodeConfig(row)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"id":         row.ID,
		"version":    row.Version,
		"is_current": row.IsCurrent,
		"config":     cfg,
	})
}

func (s *Service) GetCurrent(c fiber.Ctx) error {
	row, err := s.GetCurrentRow(c.Context())
	if err != nil {
		if errors.Is(err, ErrSettingsNotFound) {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	cfg, err := s.DecodeConfig(row)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"id":         row.ID,
		"version":    row.Version,
		"is_current": row.IsCurrent,
		"config":     cfg,
	})
}

func (s *Service) Create(c fiber.Ctx) error {
	in, err := parseCreateInput(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	created, err := s.CreateVersion(c.Context(), in.Config)
	if err != nil {
		if errors.Is(err, ErrInvalidConfig) {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	cfg, err := s.DecodeConfig(created)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if s.Server != nil && s.Server.WSHub != nil {
		s.Server.WSHub.BroadcastSettingsUpdated(created.Version)
	}

	if !wantsJSON(c) {
		return c.Redirect().To("/settings/" + strconv.FormatUint(uint64(created.ID), 10) + "/edit")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":         created.ID,
		"version":    created.Version,
		"is_current": created.IsCurrent,
		"config":     cfg,
	})
}

func (s *Service) Patch(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	in, err := parseUpdateInput(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	created, err := s.CreateVersionFromID(c.Context(), uint(id), in.Config)
	if err != nil {
		if errors.Is(err, ErrSettingsNotFound) {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
		if errors.Is(err, ErrInvalidConfig) {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	cfg, err := s.DecodeConfig(created)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if s.Server != nil && s.Server.WSHub != nil {
		s.Server.WSHub.BroadcastSettingsUpdated(created.Version)
	}

	if !wantsJSON(c) {
		return c.Redirect().To("/settings/" + strconv.FormatUint(uint64(created.ID), 10) + "/edit")
	}

	return c.JSON(fiber.Map{
		"id":         created.ID,
		"version":    created.Version,
		"is_current": created.IsCurrent,
		"config":     cfg,
	})
}

func (s *Service) Put(c fiber.Ctx) error {
	return s.Patch(c)
}

func (s *Service) Delete(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	if err := s.DeleteByID(c.Context(), uint(id)); err != nil {
		if errors.Is(err, ErrSettingsNotFound) {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if !wantsJSON(c) {
		return c.Redirect().To("/settings")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Service) PostWithMethodOverride(c fiber.Ctx) error {
	method := strings.ToUpper(strings.TrimSpace(c.FormValue("_method")))
	if method == "PATCH" || method == "PUT" {
		return s.Patch(c)
	}
	if method == "DELETE" {
		return s.Delete(c)
	}

	return s.Create(c)
}

func (s *Service) NewSettingsWS() fiber.Handler {
	return websocket.New(func(conn *websocket.Conn) {
		if s.Server != nil {
			_ = s.Server.AddSettingsWSConn(conn)
		}
		defer func() {
			if s.Server != nil {
				_ = s.Server.DelSettingsWSConn(conn)
			}
			_ = conn.Close()
		}()

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})
}

func wantsJSON(c fiber.Ctx) bool {
	if strings.HasPrefix(strings.ToLower(c.Path()), "/api/") {
		return true
	}

	if strings.EqualFold(c.Query("format"), "json") {
		return true
	}

	accept := strings.ToLower(c.Get("Accept"))
	return strings.Contains(accept, "application/json")
}

func parseCreateInput(c fiber.Ctx) (createSettingsInput, error) {
	return parseSettingsInput(c)
}

func parseUpdateInput(c fiber.Ctx) (createSettingsInput, error) {
	return parseSettingsInput(c)
}

func parseSettingsInput(c fiber.Ctx) (createSettingsInput, error) {
	contentType := strings.ToLower(c.Get("Content-Type"))
	if strings.Contains(contentType, "application/json") {
		var in createSettingsInput
		if err := c.Bind().JSON(&in); err != nil {
			return createSettingsInput{}, err
		}
		return in, nil
	}

	var in createSettingsInput
	if err := c.Bind().Form(&in); err != nil {
		return createSettingsInput{}, err
	}

	in.Config = models.SettingsConfig{
		Name: strings.TrimSpace(in.ConfigName),
		Layout: models.SettingsLayout{
			Name:                  strings.TrimSpace(in.LayoutName),
			OverlayLayout:         strings.TrimSpace(in.OverlayLayout),
			Theme:                 strings.TrimSpace(in.Theme),
			VideoFit:              strings.TrimSpace(in.VideoFit),
			VideoAlign:            strings.TrimSpace(in.VideoAlign),
			VideoOffsetXPct:       parseIntOrZero(in.VideoOffsetXPct),
			VideoOffsetYPct:       parseIntOrZero(in.VideoOffsetYPct),
			InfiniteVideoPlayback: parseBoolForm(in.InfiniteVideoPlayback),
			OverlayPaddingTop:     parseIntOrZero(in.OverlayPaddingTop),
			OverlayPaddingRight:   parseIntOrZero(in.OverlayPaddingRight),
			OverlayPaddingBottom:  parseIntOrZero(in.OverlayPaddingBottom),
			OverlayPaddingLeft:    parseIntOrZero(in.OverlayPaddingLeft),
			MetricsScale:          parseIntOrZero(in.MetricsScale),
			MetricsOffsetX:        parseIntOrZero(in.MetricsOffsetX),
			MetricsOffsetY:        parseIntOrZero(in.MetricsOffsetY),
		},
		MediaSources: []models.SettingsMediaSource{{
			Kind:  strings.TrimSpace(in.MediaKind),
			URL:   strings.TrimSpace(in.MediaURL),
			Label: strings.TrimSpace(in.MediaLabel),
		}},
	}

	return in, nil
}

func parseIntOrZero(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return value
}

func parseBoolForm(raw string) bool {
	value := strings.ToLower(strings.TrimSpace(raw))
	return value == "true" || value == "1" || value == "on" || value == "yes"
}
