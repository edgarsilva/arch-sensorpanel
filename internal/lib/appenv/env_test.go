package appenv

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadFromYAMLSuccess(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	configDir := filepath.Join(tmpHome, ".config", "sensorpanel")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "conf.yaml")
	content := []byte(`
environment: staging
app_port: 9999
database_uri: /tmp/sensorpanel.sqlite3
app_shutdown_timeout: 15s
youtube_api_key: abc123
`)
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	env := &Env{
		Environment:        "development",
		AppPort:            9070,
		DatabaseURI:        "~/.config/sensorpanel/db.sqlite3",
		AppShutdownTimeout: 1 * time.Second,
	}

	if err := loadFromYAML(env); err != nil {
		t.Fatalf("loadFromYAML failed: %v", err)
	}

	if env.Environment != "staging" {
		t.Fatalf("expected environment staging, got %q", env.Environment)
	}
	if env.AppPort != 9999 {
		t.Fatalf("expected app_port 9999, got %d", env.AppPort)
	}
	if env.DatabaseURI != "/tmp/sensorpanel.sqlite3" {
		t.Fatalf("expected database_uri /tmp/sensorpanel.sqlite3, got %q", env.DatabaseURI)
	}
	if env.AppShutdownTimeout != 15*time.Second {
		t.Fatalf("expected app_shutdown_timeout 15s, got %s", env.AppShutdownTimeout)
	}
	if env.YouTubeAPIKey != "abc123" {
		t.Fatalf("expected youtube_api_key abc123, got %q", env.YouTubeAPIKey)
	}
}

func TestLoadFromYAMLMissingFileUsesDefaults(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	env := &Env{
		Environment:        "development",
		AppPort:            9070,
		DatabaseURI:        "~/.config/sensorpanel/db.sqlite3",
		AppShutdownTimeout: 1 * time.Second,
		YouTubeAPIKey:      "",
	}

	if err := loadFromYAML(env); err != nil {
		t.Fatalf("loadFromYAML should ignore missing file: %v", err)
	}

	if env.Environment != "development" || env.AppPort != 9070 || env.DatabaseURI == "" {
		t.Fatalf("expected defaults to remain unchanged: %+v", env)
	}
}

func TestValidateRejectsInvalidValues(t *testing.T) {
	env := &Env{Environment: "qa", AppPort: 0, AppShutdownTimeout: 0}
	if err := validate(env); err == nil {
		t.Fatal("expected validate to fail for invalid values")
	}
}
