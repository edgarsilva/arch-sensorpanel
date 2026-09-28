// Package appenv provides a simple way to load/veriby/validate environment variables
// in one place.
package appenv

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Env struct {
	Environment        string
	AppPort            int
	DatabaseURI        string
	AppShutdownTimeout time.Duration
	YouTubeAPIKey      string
}

func New() *Env {
	env := &Env{
		Environment:        "development",
		AppPort:            9070,
		DatabaseURI:        "~/.config/sensorpanel/db.sqlite3",
		AppShutdownTimeout: 1 * time.Second,
		YouTubeAPIKey:      "",
	}

	if err := loadFromYAML(env); err != nil {
		log.Fatal("failed to load yaml config:", err)
	}

	if err := validate(env); err != nil {
		log.Fatal("invalid app config:", err)
	}

	return env
}

type yamlConfig struct {
	Environment        string        `yaml:"environment"`
	AppPort            int           `yaml:"app_port"`
	DatabaseURI        string        `yaml:"database_uri"`
	AppShutdownTimeout time.Duration `yaml:"app_shutdown_timeout"`
	YouTubeAPIKey      string        `yaml:"youtube_api_key"`
}

func loadFromYAML(env *Env) error {
	if env == nil {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}

	configPath := filepath.Join(home, ".config", "sensorpanel", "conf.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", configPath, err)
	}

	var cfg yamlConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse %s: %w", configPath, err)
	}

	if cfg.Environment != "" {
		env.Environment = cfg.Environment
	}
	if cfg.AppPort != 0 {
		env.AppPort = cfg.AppPort
	}
	if cfg.DatabaseURI != "" {
		env.DatabaseURI = cfg.DatabaseURI
	}
	if cfg.AppShutdownTimeout != 0 {
		env.AppShutdownTimeout = cfg.AppShutdownTimeout
	}
	if cfg.YouTubeAPIKey != "" {
		env.YouTubeAPIKey = cfg.YouTubeAPIKey
	}

	return nil
}

func validate(env *Env) error {
	if env == nil {
		return fmt.Errorf("env is required")
	}

	switch env.Environment {
	case "development", "test", "staging", "production":
	default:
		return fmt.Errorf("environment must be one of development,test,staging,production")
	}

	if env.AppPort < 1 || env.AppPort > 65535 {
		return fmt.Errorf("app_port must be between 1 and 65535")
	}

	if env.AppShutdownTimeout <= 0 {
		return fmt.Errorf("app_shutdown_timeout must be greater than 0")
	}

	return nil
}
