package appenv

import (
	"os"
	"testing"
	"time"

	"github.com/edgarsilva/simpleenv"
)

func loadForTest() (*Env, error) {
	env := &Env{AppShutdownTimeout: 10 * time.Second}
	if err := simpleenv.Load(env); err != nil {
		return nil, err
	}

	return env, nil
}

func TestLoadSuccess(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("APP_PORT", "9070")
	t.Setenv("DATABASE_URI", "data/sensorpanel.db.sqlite3")
	t.Setenv("APP_SHUTDOWN_TIMEOUT", "12s")

	env, err := loadForTest()
	if err != nil {
		t.Fatalf("loadForTest returned error: %v", err)
	}

	if env.Environment != "development" {
		t.Fatalf("expected AppEnv development, got %q", env.Environment)
	}
	if env.AppPort != 9070 {
		t.Fatalf("expected AppPort 9070, got %d", env.AppPort)
	}
	if env.DatabaseURI != "data/sensorpanel.db.sqlite3" {
		t.Fatalf("expected DatabaseURI data/sensorpanel.db.sqlite3, got %q", env.DatabaseURI)
	}
	if env.AppShutdownTimeout != 12*time.Second {
		t.Fatalf("expected AppShutdownTimeout 12s, got %s", env.AppShutdownTimeout)
	}
}

func TestLoadMissingOptionalEnv(t *testing.T) {
	os.Unsetenv("APP_ENV")
	os.Unsetenv("APP_PORT")
	os.Unsetenv("DATABASE_URI")
	os.Unsetenv("APP_SHUTDOWN_TIMEOUT")

	env, err := loadForTest()
	if err != nil {
		t.Fatalf("expected loadForTest to succeed when optional env vars are missing: %v", err)
	}

	if env.Environment != "" {
		t.Fatalf("expected default Environment to remain empty, got %q", env.Environment)
	}
	if env.AppPort != 0 {
		t.Fatalf("expected default AppPort to remain 0, got %d", env.AppPort)
	}
	if env.DatabaseURI != "" {
		t.Fatalf("expected default DatabaseURI to remain empty, got %q", env.DatabaseURI)
	}
}

func TestLoadDefaultsShutdownTimeout(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("APP_PORT", "9070")
	t.Setenv("DATABASE_URI", "data/sensorpanel.db.sqlite3")
	os.Unsetenv("APP_SHUTDOWN_TIMEOUT")

	env, err := loadForTest()
	if err != nil {
		t.Fatalf("loadForTest returned error: %v", err)
	}

	if env.AppShutdownTimeout != 10*time.Second {
		t.Fatalf("expected default AppShutdownTimeout 10s, got %s", env.AppShutdownTimeout)
	}
}

func TestLoadRejectsInvalidShutdownTimeout(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("APP_PORT", "9070")
	t.Setenv("DATABASE_URI", "data/sensorpanel.db.sqlite3")
	t.Setenv("APP_SHUTDOWN_TIMEOUT", "abc")

	if _, err := loadForTest(); err == nil {
		t.Fatal("expected loadForTest to fail when APP_SHUTDOWN_TIMEOUT is invalid")
	}
}

func TestLoadRejectsNonPositiveShutdownTimeout(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("APP_PORT", "9070")
	t.Setenv("DATABASE_URI", "data/sensorpanel.db.sqlite3")
	t.Setenv("APP_SHUTDOWN_TIMEOUT", "0s")

	if _, err := loadForTest(); err == nil {
		t.Fatal("expected loadForTest to fail when APP_SHUTDOWN_TIMEOUT is not positive")
	}
}
