package appenv

import "testing"

func TestLoadSuccess(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("APP_PORT", "9070")
	t.Setenv("DATABASE_URI", "data/sensorpanel.db.sqlite3")

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
}

func TestLoadMissingRequiredEnv(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_PORT", "")
	t.Setenv("DATABASE_URI", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected Load to fail when required env vars are missing")
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
