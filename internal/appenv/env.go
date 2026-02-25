package appenv

import (
	"fmt"
	"os"
	"strings"
)

type Env struct {
	AppEnv      string
	AppPort     string
	DatabaseURI string
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

	return &Env{
		AppEnv:      appEnv,
		AppPort:     appPort,
		DatabaseURI: databaseURI,
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
