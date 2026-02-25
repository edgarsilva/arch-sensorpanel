package appenv

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Env struct {
	AppEnv             string
	AppPort            string
	DatabaseURI        string
	AppShutdownTimeout time.Duration
}

func Load() (*Env, error) {
	appEnv, err := require("APP_ENV")
	if err != nil {
		return nil, err
	}

	appPort, err := require("APP_PORT")
	if err != nil {
		return nil, err
	}

	databaseURI, err := require("DATABASE_URI")
	if err != nil {
		return nil, err
	}

	shutdownTimeout, err := durationOrDefault("APP_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, err
	}

	return &Env{
		AppEnv:             appEnv,
		AppPort:            appPort,
		DatabaseURI:        databaseURI,
		AppShutdownTimeout: shutdownTimeout,
	}, nil
}

func (e *Env) ListenAddr() string {
	if e == nil {
		return ":9070"
	}

	port := strings.TrimSpace(e.AppPort)
	if port == "" {
		return ":9070"
	}

	if strings.HasPrefix(port, ":") {
		return port
	}

	return ":" + port
}

func require(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("missing required environment variable: %s", key)
	}

	return value, nil
}

func durationOrDefault(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s: %w", key, err)
	}

	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than 0", key)
	}

	return parsed, nil
}
