package settings

import (
	"errors"
	"io/fs"
	"strconv"

	"sensorpanel/internal/models"

	"github.com/gofiber/fiber/v3"
)

type writeSettingsRequest struct {
	Config models.SettingsConfig `json:"config"`
}

func (s *Service) IndexPage(c fiber.Ctx) error {
	settingsHTML, err := fs.ReadFile(s.PublicFS, "settings.html")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "settings.html not found")
	}

	return c.Type("html").Send(settingsHTML)
}

func (s *Service) Index(c fiber.Ctx) error {
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
	var req writeSettingsRequest
	if err := c.Bind().JSON(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	created, err := s.CreateVersion(c.Context(), req.Config)
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

	var req writeSettingsRequest
	if err := c.Bind().JSON(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	created, err := s.CreateVersionFromID(c.Context(), uint(id), req.Config)
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

	return c.JSON(fiber.Map{
		"id":         created.ID,
		"version":    created.Version,
		"is_current": created.IsCurrent,
		"config":     cfg,
	})
}
