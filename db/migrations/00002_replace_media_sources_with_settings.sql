-- +goose Up
DROP TABLE IF EXISTS media_sources;

CREATE TABLE IF NOT EXISTS settings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  version INTEGER NOT NULL UNIQUE,
  is_current INTEGER NOT NULL DEFAULT 0,
  config_json TEXT NOT NULL CHECK (json_valid(config_json)),
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_settings_is_current ON settings (is_current);

INSERT INTO settings (version, is_current, config_json)
SELECT 1, 1, '{"media_sources":[],"layout":{"name":"left","path":""}}'
WHERE NOT EXISTS (SELECT 1 FROM settings);

-- +goose Down
DROP TABLE IF EXISTS settings;
