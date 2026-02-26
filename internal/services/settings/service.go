package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"sensorpanel/internal/db"
	"sensorpanel/internal/models"
	"sensorpanel/internal/server"

	"gorm.io/gorm"
)

var (
	ErrSettingsNotFound = errors.New("settings not found")
	ErrInvalidConfig    = errors.New("invalid settings config")
	ErrDeleteCurrent    = errors.New("cannot delete current settings")
)

type Service struct {
	*server.Server
}

func New(s *server.Server) *Service {
	return &Service{Server: s}
}

func (s *Service) List(ctx context.Context) ([]models.Settings, error) {
	rows, err := gorm.G[models.Settings](s.DB.WithContext(ctx)).Order("version DESC").Find(ctx)
	if err != nil {
		return nil, db.WrapWithOp("list settings", err)
	}

	return rows, nil
}

func (s *Service) GetByID(ctx context.Context, id uint) (*models.Settings, error) {
	row, err := gorm.G[models.Settings](s.DB.WithContext(ctx)).Where("id = ?", id).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSettingsNotFound
		}
		return nil, db.WrapWithOp("get settings by id", err)
	}

	return &row, nil
}

func (s *Service) GetCurrentRow(ctx context.Context) (*models.Settings, error) {
	row, err := gorm.G[models.Settings](s.DB.WithContext(ctx)).Where("is_current = ?", true).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSettingsNotFound
		}
		return nil, db.WrapWithOp("get current settings", err)
	}

	return &row, nil
}

func (s *Service) CreateVersion(ctx context.Context, config models.SettingsConfig) (*models.Settings, error) {
	return s.createVersion(ctx, config)
}

func (s *Service) CreateVersionFromID(ctx context.Context, id uint, config models.SettingsConfig) (*models.Settings, error) {
	if _, err := s.GetByID(ctx, id); err != nil {
		return nil, err
	}

	return s.createVersion(ctx, config)
}

func (s *Service) DeleteByID(ctx context.Context, id uint) error {
	row, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if row.IsCurrent {
		return ErrDeleteCurrent
	}

	result := s.DB.WithContext(ctx).Delete(&models.Settings{}, id)
	if result.Error != nil {
		return db.WrapWithOp("delete settings", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrSettingsNotFound
	}

	return nil
}

func (s *Service) DecodeConfig(row *models.Settings) (models.SettingsConfig, error) {
	if row == nil {
		return models.SettingsConfig{}, ErrSettingsNotFound
	}

	var cfg models.SettingsConfig
	if err := json.Unmarshal([]byte(row.ConfigJSON), &cfg); err != nil {
		return models.SettingsConfig{}, db.WrapWithOp("decode settings config", err)
	}

	return cfg, nil
}

func (s *Service) createVersion(ctx context.Context, config models.SettingsConfig) (*models.Settings, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, db.WrapWithOp("marshal settings config", err)
	}

	var created models.Settings
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var maxVersion int64
		if err := tx.Model(&models.Settings{}).Select("COALESCE(MAX(version), 0)").Scan(&maxVersion).Error; err != nil {
			return db.WrapWithOp("select max settings version", err)
		}

		now := time.Now().UTC()
		if err := tx.Model(&models.Settings{}).
			Where("is_current = ?", true).
			Updates(map[string]any{"is_current": false, "updated_at": now}).Error; err != nil {
			return db.WrapWithOp("clear current settings", err)
		}

		created = models.Settings{
			Version:    maxVersion + 1,
			IsCurrent:  true,
			ConfigJSON: string(configJSON),
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		if err := gorm.G[models.Settings](tx).Create(ctx, &created); err != nil {
			return db.WrapWithOp("create settings version", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &created, nil
}

func validateConfig(config models.SettingsConfig) error {
	layout := strings.ToLower(strings.TrimSpace(config.Layout.Name))
	if layout != "left" && layout != "right" && layout != "center" && layout != "cover" {
		return fmt.Errorf("%w: unsupported layout %q", ErrInvalidConfig, config.Layout.Name)
	}

	overlayLayout := strings.ToLower(strings.TrimSpace(config.Layout.OverlayLayout))
	if overlayLayout != "" && overlayLayout != "column" && overlayLayout != "row" {
		return fmt.Errorf("%w: unsupported overlay_layout %q", ErrInvalidConfig, config.Layout.OverlayLayout)
	}

	theme := strings.ToLower(strings.TrimSpace(config.Layout.Theme))
	if theme != "" &&
		theme != "cool" &&
		theme != "winter" &&
		theme != "corporate" &&
		theme != "nord" &&
		theme != "aqua" &&
		theme != "lofi" &&
		theme != "business" &&
		theme != "dark" &&
		theme != "dim" {
		return fmt.Errorf("%w: unsupported theme %q", ErrInvalidConfig, config.Layout.Theme)
	}

	videoFit := strings.ToLower(strings.TrimSpace(config.Layout.VideoFit))
	if videoFit != "" && videoFit != "cover" && videoFit != "contain" {
		return fmt.Errorf("%w: unsupported video_fit %q", ErrInvalidConfig, config.Layout.VideoFit)
	}

	videoAlign := strings.ToLower(strings.TrimSpace(config.Layout.VideoAlign))
	if videoAlign != "" && videoAlign != "left" && videoAlign != "center" && videoAlign != "right" {
		return fmt.Errorf("%w: unsupported video_align %q", ErrInvalidConfig, config.Layout.VideoAlign)
	}

	if config.Layout.OverlayPaddingTop < 0 || config.Layout.OverlayPaddingTop > 100 {
		return fmt.Errorf("%w: overlay_padding_top must be between 0 and 100", ErrInvalidConfig)
	}

	if config.Layout.OverlayPaddingRight < 0 || config.Layout.OverlayPaddingRight > 100 {
		return fmt.Errorf("%w: overlay_padding_right must be between 0 and 100", ErrInvalidConfig)
	}

	if config.Layout.OverlayPaddingBottom < 0 || config.Layout.OverlayPaddingBottom > 100 {
		return fmt.Errorf("%w: overlay_padding_bottom must be between 0 and 100", ErrInvalidConfig)
	}

	if config.Layout.OverlayPaddingLeft < 0 || config.Layout.OverlayPaddingLeft > 100 {
		return fmt.Errorf("%w: overlay_padding_left must be between 0 and 100", ErrInvalidConfig)
	}

	if config.Layout.MetricsScale != 0 && (config.Layout.MetricsScale < 50 || config.Layout.MetricsScale > 200) {
		return fmt.Errorf("%w: metrics_scale_pct must be between 50 and 200", ErrInvalidConfig)
	}

	if config.Layout.MetricsOffsetX < -250 || config.Layout.MetricsOffsetX > 250 {
		return fmt.Errorf("%w: metrics_offset_x must be between -250 and 250", ErrInvalidConfig)
	}

	if config.Layout.MetricsOffsetY < -250 || config.Layout.MetricsOffsetY > 250 {
		return fmt.Errorf("%w: metrics_offset_y must be between -250 and 250", ErrInvalidConfig)
	}

	for i, source := range config.MediaSources {
		if strings.TrimSpace(source.URL) == "" {
			return fmt.Errorf("%w: media_sources[%d].url is required", ErrInvalidConfig, i)
		}
		if strings.TrimSpace(source.Kind) == "" {
			return fmt.Errorf("%w: media_sources[%d].kind is required", ErrInvalidConfig, i)
		}
	}

	return nil
}
