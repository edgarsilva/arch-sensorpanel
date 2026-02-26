package settings

import (
	"errors"
	"io/fs"
	"strconv"
	"strings"

	"sensorpanel/internal/models"

	"github.com/gofiber/fiber/v3"
)

type createSettingsInput struct {
	Config     models.SettingsConfig `json:"config"`
	LayoutName string                `form:"layout_name"`
	LayoutPath string                `form:"layout_path"`
	MediaKind  string                `form:"media_kind"`
	MediaURL   string                `form:"media_url"`
	MediaLabel string                `form:"media_label"`
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

func wantsJSON(c fiber.Ctx) bool {
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
		Layout: models.SettingsLayout{
			Name: strings.TrimSpace(in.LayoutName),
			Path: strings.TrimSpace(in.LayoutPath),
		},
		MediaSources: []models.SettingsMediaSource{{
			Kind:  strings.TrimSpace(in.MediaKind),
			URL:   strings.TrimSpace(in.MediaURL),
			Label: strings.TrimSpace(in.MediaLabel),
		}},
	}

	return in, nil
}
