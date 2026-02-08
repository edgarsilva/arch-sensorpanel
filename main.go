package main

import (
	"log"
	"time"

	"sensorpanel/handlers"
	"sensorpanel/internal/sensors"

	"github.com/gofiber/fiber/v2"
)

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

	// Serve the sensor panel HTML
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("./public/index.html")
	})

	app.Get("/metrics", metricsHandler.GetMetrics)

	log.Println("Listening on http://localhost:9070")
	log.Fatal(app.Listen(":9070"))
}
