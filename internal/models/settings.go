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
	Name         string                `json:"name,omitempty"`
	MediaSources []SettingsMediaSource `json:"media_sources"`
	Layout       SettingsLayout        `json:"layout"`
}

type SettingsMediaSource struct {
	Kind  string `json:"kind"`
	URL   string `json:"url"`
	Label string `json:"label,omitempty"`
}

type SettingsLayout struct {
	Name                  string `json:"name"`
	OverlayLayout         string `json:"overlay_layout,omitempty"`
	Theme                 string `json:"theme,omitempty"`
	VideoFit              string `json:"video_fit,omitempty"`
	VideoAlign            string `json:"video_align,omitempty"`
	VideoOffsetXPct       int    `json:"video_offset_x_pct,omitempty"`
	VideoOffsetYPct       int    `json:"video_offset_y_pct,omitempty"`
	InfiniteVideoPlayback bool   `json:"infinite_video_playback,omitempty"`
	OverlayPaddingTop     int    `json:"overlay_padding_top,omitempty"`
	OverlayPaddingRight   int    `json:"overlay_padding_right,omitempty"`
	OverlayPaddingBottom  int    `json:"overlay_padding_bottom,omitempty"`
	OverlayPaddingLeft    int    `json:"overlay_padding_left,omitempty"`
	MetricsScale          int    `json:"metrics_scale_pct,omitempty"`
	MetricsOffsetX        int    `json:"metrics_offset_x,omitempty"`
	MetricsOffsetY        int    `json:"metrics_offset_y,omitempty"`
}
