// Package handlers contains HTTP handlers for the sensor panel.
package handlers

import (
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
		EdgeC       float64 `json:"edge_c"`
		HotspotC    float64 `json:"hotspot_c"`
		VramC       float64 `json:"vram_c"`
		VramUsedGB  float64 `json:"vram_used_gb"`
		VramTotalGB float64 `json:"vram_total_gb"`
		VramUsedPct float64 `json:"vram_used_pct"`
		PowerW      float64 `json:"power_w"`
		UtilPct     float64 `json:"util_pct"`
	} `json:"gpu"`
}

type MetricsHandler struct {
	cpuSampler     *sensors.CPUBusySampler
	cpuPower       *sensors.CPUPowerSampler
	ramSampler     *sensors.SystemRAMSampler
	sensorsSampler *sensors.LmSensorsSampler
	gpuBusySampler *sensors.GPUBusySampler
	gpuVRAMSampler *sensors.GPUVRAMSampler
}

func NewMetricsHandler(
	cpuSampler *sensors.CPUBusySampler,
	cpuPower *sensors.CPUPowerSampler,
	ramSampler *sensors.SystemRAMSampler,
	sensorsSampler *sensors.LmSensorsSampler,
	gpuBusySampler *sensors.GPUBusySampler,
	gpuVRAMSampler *sensors.GPUVRAMSampler,
) *MetricsHandler {
	return &MetricsHandler{
		cpuSampler:     cpuSampler,
		cpuPower:       cpuPower,
		ramSampler:     ramSampler,
		sensorsSampler: sensorsSampler,
		gpuBusySampler: gpuBusySampler,
		gpuVRAMSampler: gpuVRAMSampler,
	}
}

func (h *MetricsHandler) GetMetrics(c *fiber.Ctx) error {
	var resp MetricsResponse

	sensorSnapshot := h.sensorsSampler.Snapshot()
	resp.CPU.TempC = sensorSnapshot.CPUTempC
	resp.GPU.EdgeC = sensorSnapshot.GPUEdgeC
	resp.GPU.HotspotC = sensorSnapshot.GPUHotspotC
	resp.GPU.VramC = sensorSnapshot.GPUVramC
	resp.GPU.PowerW = sensorSnapshot.GPUPowerW

	// fmt.Println("util_pct ->", h.cpuSampler.Utilization())
	resp.CPU.UtilPct = h.cpuSampler.Utilization()
	resp.CPU.PowerW = h.cpuPower.PowerW()
	resp.GPU.UtilPct = h.gpuBusySampler.Utilization()
	gpuVRAMSnapshot := h.gpuVRAMSampler.Snapshot()
	resp.GPU.VramUsedGB = gpuVRAMSnapshot.UsedGB
	resp.GPU.VramTotalGB = gpuVRAMSnapshot.TotalGB
	resp.GPU.VramUsedPct = gpuVRAMSnapshot.UsedPct

	ramSnapshot, err := h.ramSampler.Snapshot()
	if err == nil {
		resp.RAM.TotalGB = ramSnapshot.TotalGB
		resp.RAM.UsedGB = ramSnapshot.UsedGB
		resp.RAM.AvailGB = ramSnapshot.AvailGB
		resp.RAM.UsedPct = ramSnapshot.UsedPct
	}

	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	return c.JSON(resp)
}
