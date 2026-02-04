package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	// Serve the sensor panel HTML
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("./public/index.html")
	})

	app.Get("/metrics", SensorPanelMetrics)

	log.Println("Listening on http://localhost:9070")
	log.Fatal(app.Listen(":9070"))
}

func SensorPanelMetrics(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 750*time.Millisecond)
	defer cancel()

	scriptPath, err := sensorScriptPath()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, scriptPath)

	output, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
		return fiber.NewError(
			fiber.StatusInternalServerError,
			"failed to collect sensor metrics",
		)
	}

	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	return c.Send(output)
}

func sensorScriptPath() (string, error) {
	// Resolve current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fiber.NewError(
			fiber.StatusInternalServerError,
			"failed to resolve working directory",
		)
	}

	// Script assumed to live at project root
	scriptPath := filepath.Join(cwd, "scripts/metrics_collector.sh")

	// Ensure the script exists
	if _, err := os.Stat(scriptPath); err != nil {
		return "", fiber.NewError(
			fiber.StatusInternalServerError,
			"sensor metrics script not found",
		)
	}

	return scriptPath, nil
}
