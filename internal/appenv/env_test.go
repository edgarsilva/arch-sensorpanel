package appenv

import (
	"testing"
	"time"
)

func TestLoadSuccess(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("APP_PORT", "9070")
	t.Setenv("DATABASE_URI", "data/sensorpanel.db.sqlite3")
	t.Setenv("APP_SHUTDOWN_TIMEOUT", "12s")

	env, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if env.AppEnv != "development" {
		t.Fatalf("expected AppEnv development, got %q", env.AppEnv)
	}
	if env.AppPort != "9070" {
		t.Fatalf("expected AppPort 9070, got %q", env.AppPort)
	}
	if env.DatabaseURI != "data/sensorpanel.db.sqlite3" {
		t.Fatalf("expected DatabaseURI data/sensorpanel.db.sqlite3, got %q", env.DatabaseURI)
	}
	if env.AppShutdownTimeout != 12*time.Second {
		t.Fatalf("expected AppShutdownTimeout 12s, got %s", env.AppShutdownTimeout)
	}
}

func TestLoadMissingRequiredEnv(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_PORT", "")
	t.Setenv("DATABASE_URI", "")
	t.Setenv("APP_SHUTDOWN_TIMEOUT", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected Load to fail when required env vars are missing")
	}
}

func TestLoadDefaultsShutdownTimeout(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("APP_PORT", "9070")
	t.Setenv("DATABASE_URI", "data/sensorpanel.db.sqlite3")
	t.Setenv("APP_SHUTDOWN_TIMEOUT", "")

	env, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
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

	if _, err := Load(); err == nil {
		t.Fatal("expected Load to fail when APP_SHUTDOWN_TIMEOUT is invalid")
	}
}

func TestLoadRejectsNonPositiveShutdownTimeout(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("APP_PORT", "9070")
	t.Setenv("DATABASE_URI", "data/sensorpanel.db.sqlite3")
	t.Setenv("APP_SHUTDOWN_TIMEOUT", "0s")

	if _, err := Load(); err == nil {
		t.Fatal("expected Load to fail when APP_SHUTDOWN_TIMEOUT is not positive")
	}
}

func TestListenAddr(t *testing.T) {
	tests := []struct {
		name string
		env  *Env
		want string
	}{
		{name: "nil env", env: nil, want: ":9070"},
		{name: "plain port", env: &Env{AppPort: "3000"}, want: ":3000"},
		{name: "prefixed port", env: &Env{AppPort: ":3000"}, want: ":3000"},
		{name: "blank port", env: &Env{AppPort: "   "}, want: ":9070"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.env.ListenAddr(); got != tc.want {
				t.Fatalf("ListenAddr mismatch: got %q want %q", got, tc.want)
			}
		})
	}
}
