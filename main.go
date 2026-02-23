package main

import (
	"embed"
	"io/fs"
	"log"
	"time"

	"sensorpanel/internal/sensors"
	"sensorpanel/services/metrics"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/static"
)

//go:embed public/**
var publicFS embed.FS

func main() {
	app := fiber.New()
	app.Use(logger.New())

	cpuSampler := sensors.NewCPUBusySampler(time.Second)
	cpuPowerSampler := sensors.NewCPUPowerSampler(time.Second)
	ramSampler := sensors.NewSystemRAMSampler(time.Second)
	sensorsSampler := sensors.NewLmSensorsSampler(time.Second)
	gpuBusySampler := sensors.NewGPUBusySampler(time.Second)
	gpuVRAMSampler := sensors.NewGPUVRAMSampler(time.Second)

	// create haldler with cpu sampler dep injected
	metricsHandler := metrics.New(
		cpuSampler,
		cpuPowerSampler,
		ramSampler,
		sensorsSampler,
		gpuBusySampler,
		gpuVRAMSampler,
	)

	publicSub, err := fs.Sub(publicFS, "public")
	if err != nil {
		log.Fatal(err)
	}

	app.Use("/public", static.New("", static.Config{
		FS: publicSub,
	}))

	// Serve the sensor panel HTML
	app.Get("/", func(c fiber.Ctx) error {
		indexHTML, err := fs.ReadFile(publicSub, "index.html")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "index.html not found")
		}
		return c.Type("html").Send(indexHTML)
	})

	app.Get("/telemetry", func(c fiber.Ctx) error {
		telemetryHTML, err := fs.ReadFile(publicSub, "telemetry.html")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "telemetry.html not found")
		}
		return c.Type("html").Send(telemetryHTML)
	})

	// app.Use("/metrics/ws", func(c fiber.Ctx) error {
	// 	if websocket.IsWebSocketUpgrade(c) {
	// 		return c.Next()
	// 	}
	// 	return fiber.ErrUpgradeRequired
	// })
	app.Get("/metrics", metricsHandler.GetMetrics)
	app.Get("/metrics/ws", metricsHandler.NewMetricsWS())

	log.Println("Listening on http://localhost:9070")
	log.Fatal(app.Listen(":9070"))
}
