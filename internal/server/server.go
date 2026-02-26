package server

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"strconv"

	"sensorpanel/internal/db"
	"sensorpanel/internal/lib/appenv"
	"sensorpanel/internal/lib/wshub"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	fiberLogger "github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

type Server struct {
	*fiber.App
	DB       *db.Database
	PublicFS fs.FS
	Env      *appenv.Env
	WSHub    *wshub.Hub
	port     int
	fiberCfg *fiber.Config
}

type ServerOption func(*Server) error

var ErrWSHubNotConfigured = errors.New("ws hub is not configured")

func New(opts ...ServerOption) (*Server, error) {
	s := &Server{}
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}

	if s.fiberCfg != nil {
		s.App = fiber.New(*s.fiberCfg)
	} else {
		s.App = fiber.New()
	}

	s.App.Use(recover.New())
	s.App.Use(fiberLogger.New())

	return s, nil
}

func (s *Server) Listen(portOverride ...int) error {
	port := s.port

	if s.Env != nil && s.Env.AppPort > 0 {
		port = s.Env.AppPort
	}

	if len(portOverride) > 0 {
		port = portOverride[0]
	}

	if port <= 0 {
		port = 9070
	}

	return s.App.Listen(":" + strconv.Itoa(port))
}

func (s *Server) Shutdown() error {
	if s == nil || s.App == nil {
		return nil
	}

	s.CloseWSHubConnections()

	return s.App.Shutdown()
}

func (s *Server) AddSettingsWSConn(conn *websocket.Conn) error {
	if s == nil || s.WSHub == nil {
		log.Printf("warning: cannot add settings ws conn: %v", ErrWSHubNotConfigured)
		return ErrWSHubNotConfigured
	}
	s.WSHub.AddSettingsWSConn(conn)
	return nil
}

func (s *Server) DelSettingsWSConn(conn *websocket.Conn) error {
	if s == nil || s.WSHub == nil {
		log.Printf("warning: cannot delete settings ws conn: %v", ErrWSHubNotConfigured)
		return ErrWSHubNotConfigured
	}
	s.WSHub.DelSettingsWSConn(conn)
	return nil
}

func (s *Server) CloseWSHubConnections() {
	if s == nil || s.WSHub == nil {
		return
	}

	s.WSHub.Close()
}

func WithDatabase(database *db.Database) ServerOption {
	return func(s *Server) error {
		if database == nil {
			return fmt.Errorf("database is required")
		}
		s.DB = database
		return nil
	}
}

func WithPublicFS(publicFS fs.FS) ServerOption {
	return func(s *Server) error {
		if publicFS == nil {
			return fmt.Errorf("public fs is required")
		}
		s.PublicFS = publicFS
		return nil
	}
}

func WithAppEnv(env *appenv.Env) ServerOption {
	return func(s *Server) error {
		if env == nil {
			return fmt.Errorf("app environment is required")
		}

		s.Env = env
		if env.AppPort > 0 {
			s.port = env.AppPort
		}

		return nil
	}
}

func WithPort(port int) ServerOption {
	return func(s *Server) error {
		if port <= 0 {
			return fmt.Errorf("port must be greater than 0")
		}

		s.port = port
		return nil
	}
}

func WithFiberConfig(cfg fiber.Config) ServerOption {
	return func(s *Server) error {
		s.fiberCfg = &cfg
		return nil
	}
}

func WithWSHub(hub *wshub.Hub) ServerOption {
	return func(s *Server) error {
		if hub == nil {
			return fmt.Errorf("ws hub is required")
		}

		s.WSHub = hub
		return nil
	}
}
