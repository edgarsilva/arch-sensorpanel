package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"time"

	"sensorpanel/handlers"
	"sensorpanel/internal/sensors"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
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

	app.Use("/public", filesystem.New(filesystem.Config{
		Root: http.FS(publicSub),
	}))

	// Serve the sensor panel HTML
	app.Get("/", func(c *fiber.Ctx) error {
		indexHTML, err := fs.ReadFile(publicSub, "index.html")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "index.html not found")
		}
		return c.Type("html").Send(indexHTML)
	})

	app.Get("/metrics", metricsHandler.GetMetrics)

	log.Println("Listening on http://localhost:9070")
	log.Fatal(app.Listen(":9070"))
}
