package routes

import (
	"io/fs"
	"time"

	"sensorpanel/internal/server"
	"sensorpanel/internal/services/metrics"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

func RegisterServices(s *server.Server) {
	if s == nil || s.App == nil {
		return
	}
	PublicRoutes(s)
	MetricsRoutes(s)
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
}

func MetricsRoutes(s *server.Server) {
	if s == nil || s.App == nil {
		return
	}

	metricsHandler := metrics.New(s, metrics.WithSampleInterval(time.Second))

	s.Get("/metrics", metricsHandler.GetMetrics)
	s.Get("/metrics/ws", metricsHandler.NewMetricsWS())
}
