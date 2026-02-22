package main

import (
	"embed"
	"io/fs"
	"log"
	"time"

	"sensorpanel/handlers"
	"sensorpanel/internal/sensors"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

//go:embed public/**
var publicFS embed.FS

func main() {
	app := fiber.New()

	cpuSampler := sensors.NewCPUBusySampler(time.Second)
	cpuPowerSampler := sensors.NewCPUPowerSampler(time.Second)
	ramSampler := sensors.NewSystemRAMSampler(time.Second)
	sensorsSampler := sensors.NewLmSensorsSampler(time.Second)
	gpuBusySampler := sensors.NewGPUBusySampler(time.Second)
	gpuVRAMSampler := sensors.NewGPUVRAMSampler(time.Second)

	// create haldler with cpu sampler dep injected
	metricsHandler := handlers.NewMetricsHandler(
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

	app.Get("/metrics", metricsHandler.GetMetrics)
	app.Get("/metrics/ws", metricsHandler.GetMetricsWS)

	log.Println("Listening on http://localhost:9070")
	log.Fatal(app.Listen(":9070"))
}
