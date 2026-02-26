package routes

import (
	"io/fs"
	"time"

	"sensorpanel/internal/server"
	"sensorpanel/internal/services/metrics"
	"sensorpanel/internal/services/settings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

func RegisterServices(s *server.Server) {
	if s == nil || s.App == nil {
		return
	}
	PublicRoutes(s)
	MetricsRoutes(s)
	SettingsRoutes(s)
}

func PublicRoutes(s *server.Server) {
	if s == nil || s.App == nil || s.PublicFS == nil {
		return
	}

	s.Use("/public", static.New("", static.Config{FS: s.PublicFS}))

	s.Get("/", func(c fiber.Ctx) error {
		indexHTML, err := fs.ReadFile(s.PublicFS, "index.html")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "index.html not found")
		}
		return c.Type("html").Send(indexHTML)
	})

	s.Get("/telemetry", func(c fiber.Ctx) error {
		telemetryHTML, err := fs.ReadFile(s.PublicFS, "telemetry.html")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "telemetry.html not found")
		}
		return c.Type("html").Send(telemetryHTML)
	})

	s.Get("/favicon.ico", func(c fiber.Ctx) error {
		return c.Redirect().To("/public/favicon.png")
	})
}

func SettingsRoutes(s *server.Server) {
	if s == nil || s.App == nil {
		return
	}

	settingsHandler := settings.New(s)

	s.Get("/settings", settingsHandler.IndexPage)
	s.Get("/settings/new", settingsHandler.IndexPage)
	s.Get("/settings/:id/edit", settingsHandler.IndexPage)
	s.Get("/settings/ws", settingsHandler.NewSettingsWS())
	s.Post("/settings", settingsHandler.Create)
	s.Post("/settings/:id", settingsHandler.PostWithMethodOverride)

	s.Get("/api/settings", settingsHandler.Index)
	s.Get("/api/settings/current", settingsHandler.GetCurrent)
	s.Get("/api/settings/:id", settingsHandler.Get)
	s.Post("/api/settings", settingsHandler.Create)
	s.Put("/api/settings/:id", settingsHandler.Put)
	s.Patch("/api/settings/:id", settingsHandler.Patch)
	s.Delete("/api/settings/:id", settingsHandler.Delete)
}

func MetricsRoutes(s *server.Server) {
	if s == nil || s.App == nil {
		return
	}

	metricsHandler := metrics.New(s, metrics.WithSampleInterval(time.Second))

	s.Get("/metrics", metricsHandler.GetMetrics)
	s.Get("/metrics/ws", metricsHandler.NewMetricsWS())
}
