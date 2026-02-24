package settings

import (
	"context"
	"errors"
	"time"

	"sensorpanel/internal/db"
	"sensorpanel/internal/models"
	"sensorpanel/internal/server"

	"gorm.io/gorm"
)

var ErrMediaSourceNotFound = errors.New("media source not found")

type Service struct {
	*server.Server
}

type CreateMediaSourceInput struct {
	Kind  models.MediaSourceKind
	URL   string
	Label string
}

func New(s *server.Server) *Service {
	return &Service{Server: s}
}

func (s *Service) CreateMediaSource(ctx context.Context, in CreateMediaSourceInput) (models.MediaSource, error) {
	mediaSource := models.MediaSource{
		Kind:  in.Kind,
		URL:   in.URL,
		Label: in.Label,
	}

	if err := gorm.G[models.MediaSource](s.DB.Gorm).Create(ctx, &mediaSource); err != nil {
		return models.MediaSource{}, db.WrapWithOp("create media source", err)
	}

	return mediaSource, nil
}

func (s *Service) ListMediaSources(ctx context.Context) ([]models.MediaSource, error) {
	rows, err := gorm.G[models.MediaSource](s.DB.Gorm).Order("created_at DESC").Find(ctx)
	if err != nil {
		return nil, db.WrapWithOp("list media sources", err)
	}

	return rows, nil
}

func (s *Service) GetActiveMediaSource(ctx context.Context) (*models.MediaSource, error) {
	row, err := gorm.G[models.MediaSource](s.DB.Gorm).Where("is_active = ?", true).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, db.WrapWithOp("get active media source", err)
	}

	return &row, nil
}

func (s *Service) SetActiveMediaSource(ctx context.Context, id uint) (*models.MediaSource, error) {
	err := s.DB.Gorm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()

		if err := tx.Model(&models.MediaSource{}).
			Where("is_active = ?", true).
			Updates(map[string]any{"is_active": false, "updated_at": now}).Error; err != nil {
			return db.WrapWithOp("clear active media source", err)
		}

		result := tx.Model(&models.MediaSource{}).
			Where("id = ?", id).
			Updates(map[string]any{"is_active": true, "updated_at": now})
		if result.Error != nil {
			return db.WrapWithOp("set active media source", result.Error)
		}

		if result.RowsAffected == 0 {
			return ErrMediaSourceNotFound
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	active, err := gorm.G[models.MediaSource](s.DB.Gorm.WithContext(ctx)).Where("id = ?", id).First(ctx)
	if err != nil {
		return nil, db.WrapWithOp("fetch active media source", err)
	}

	return &active, nil
}
