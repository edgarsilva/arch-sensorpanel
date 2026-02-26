package models

import "time"

type Settings struct {
	ID         uint      `gorm:"primaryKey"`
	Version    int64     `gorm:"not null;uniqueIndex"`
	IsCurrent  bool      `gorm:"not null;default:false;index"`
	ConfigJSON string    `gorm:"type:text;not null"`
	CreatedAt  time.Time `gorm:"not null"`
	UpdatedAt  time.Time `gorm:"not null"`
}

func (Settings) TableName() string {
	return "settings"
}

type SettingsConfig struct {
	MediaSources []SettingsMediaSource `json:"media_sources"`
	Layout       SettingsLayout        `json:"layout"`
}

type SettingsMediaSource struct {
	Kind  string `json:"kind"`
	URL   string `json:"url"`
	Label string `json:"label,omitempty"`
}

type SettingsLayout struct {
	Name          string `json:"name"`
	OverlayLayout string `json:"overlay_layout,omitempty"`
	Theme         string `json:"theme,omitempty"`
	VideoFit      string `json:"video_fit,omitempty"`
	VideoAlign    string `json:"video_align,omitempty"`
}
