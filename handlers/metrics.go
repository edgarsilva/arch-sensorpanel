// Package handlers contains HTTP handlers for the sensor panel.
package handlers

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"sensorpanel/internal/sensors"

	"github.com/gofiber/fiber/v2"
)

type MetricsResponse struct {
	CPU struct {
		TempC   float64 `json:"temp_c"`
		UtilPct float64 `json:"util_pct"`
		PowerW  float64 `json:"power_w"`
	} `json:"cpu"`

	RAM struct {
		TotalGB float64 `json:"total_gb"`
		UsedGB  float64 `json:"used_gb"`
		AvailGB float64 `json:"avail_gb"`
		UsedPct float64 `json:"used_pct"`
	} `json:"ram"`

	GPU struct {
		EdgeC    float64 `json:"edge_c"`
		HotspotC float64 `json:"hotspot_c"`
		VramC    float64 `json:"vram_c"`
		PowerW   float64 `json:"power_w"`
		UtilPct  float64 `json:"util_pct"`
	} `json:"gpu"`
}

type MetricsHandler struct {
	cpuSampler *sensors.CPUUtilSampler
	cpuPower   *sensors.CPUPowerSampler
	memSampler *sensors.MemorySampler
}

func NewMetricsHandler(cpuSampler *sensors.CPUUtilSampler, cpuPower *sensors.CPUPowerSampler, memSampler *sensors.MemorySampler) *MetricsHandler {
	return &MetricsHandler{cpuSampler, cpuPower, memSampler}
}

func (h *MetricsHandler) GetMetrics(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	scriptPath, err := sensorScriptPath()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, scriptPath)

	output, err := cmd.Output()
	if err != nil {
		return fiber.NewError(500, "failed to execute collector")
	}

	var resp MetricsResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return fiber.NewError(500, "invalid collector output")
	}

	// fmt.Println("util_pct ->", h.cpuSampler.Utilization())
	resp.CPU.UtilPct = h.cpuSampler.Utilization()
	resp.CPU.PowerW = h.cpuPower.PowerW()

	memSnapshot, err := h.memSampler.Snapshot()
	if err == nil {
		resp.RAM.TotalGB = memSnapshot.TotalGB
		resp.RAM.UsedGB = memSnapshot.UsedGB
		resp.RAM.AvailGB = memSnapshot.AvailGB
		resp.RAM.UsedPct = memSnapshot.UsedPct
	}

	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	return c.JSON(resp)
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
