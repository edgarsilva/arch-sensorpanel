package models

import "time"

type MediaSourceKind string

const (
	MediaSourceKindVideo    MediaSourceKind = "video"
	MediaSourceKindPlaylist MediaSourceKind = "playlist"
	MediaSourceKindImage    MediaSourceKind = "image"
)

type MediaSource struct {
	ID        uint            `gorm:"primaryKey"`
	Kind      MediaSourceKind `gorm:"type:text;not null"`
	URL       string          `gorm:"type:text;not null"`
	Label     string          `gorm:"type:text;not null;default:''"`
	IsActive  bool            `gorm:"not null;default:false;index"`
	CreatedAt time.Time       `gorm:"not null"`
	UpdatedAt time.Time       `gorm:"not null"`
}

func (MediaSource) TableName() string {
	return "media_sources"
}
