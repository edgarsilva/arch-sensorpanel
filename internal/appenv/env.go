// Package appenv provides a simple way to load/veriby/validate environment variables
// in one place.
package appenv

import (
	"log"
	"time"

	"github.com/edgarsilva/simpleenv"
)

type Env struct {
	AppEnv             string        `env:"APP_ENV;oneof=development,test,staging,production"`
	AppPort            int           `env:"APP_PORT;min=1;max=65535"`
	DatabaseURI        string        `env:"DATABASE_URI"`
	AppShutdownTimeout time.Duration `env:"APP_SHUTDOWN_TIMEOUT;optional;min=1s"`
}

func New() *Env {
	env := &Env{AppShutdownTimeout: 10 * time.Second}
	err := simpleenv.Load(env)
	if err != nil {
		log.Fatal("failed to load environment variables:", err)
	}

	return env
}
