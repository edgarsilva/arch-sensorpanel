-- +goose Up
CREATE TABLE IF NOT EXISTS media_sources (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kind TEXT NOT NULL CHECK (kind IN ('video', 'playlist', 'image')),
  url TEXT NOT NULL,
  label TEXT NOT NULL DEFAULT '',
  is_active INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_media_sources_is_active ON media_sources (is_active);

-- +goose Down
DROP TABLE IF EXISTS media_sources;
