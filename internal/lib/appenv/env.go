// Package appenv provides a simple way to load/veriby/validate environment variables
// in one place.
package appenv

import (
	"log"
	"time"

	"github.com/edgarsilva/simpleenv"
)

type Env struct {
	Environment        string        `env:"APP_ENV;optional;oneof=development,test,staging,production"`
	AppPort            int           `env:"APP_PORT;optional;min=1;max=65535"`
	DatabaseURI        string        `env:"DATABASE_URI;optional"`
	AppShutdownTimeout time.Duration `env:"APP_SHUTDOWN_TIMEOUT;optional;min=1s"`
}

func New() *Env {
	env := &Env{
		Environment:        "development",
		AppPort:            9070,
		DatabaseURI:        "~/.config/sensorpanel.db.sqlite3",
		AppShutdownTimeout: 1 * time.Second,
	}
	err := simpleenv.Load(env)
	if err != nil {
		log.Fatal("failed to load environment variables:", err)
	}

	return env
}
