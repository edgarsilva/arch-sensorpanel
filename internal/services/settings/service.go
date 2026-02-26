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
	if layout != "left" && layout != "right" && layout != "center" {
		return fmt.Errorf("%w: unsupported layout %q", ErrInvalidConfig, config.Layout.Name)
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
