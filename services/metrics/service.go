package metrics

import (
	"sensorpanel/internal/sensors"
)

type cpuBusyReader interface {
	Snapshot() sensors.CPUBusySnapshot
}

type cpuPowerReader interface {
	Snapshot() sensors.CPUPowerSnapshot
}

type ramReader interface {
	Snapshot() (sensors.SystemRAMSnapshot, error)
}

type lmSensorsReader interface {
	Snapshot() sensors.LmSensorsSnapshot
}

type gpuBusyReader interface {
	Snapshot() sensors.GPUBusySnapshot
}

type gpuVRAMReader interface {
	Snapshot() sensors.GPUVRAMSnapshot
}

type Metrics struct {
	cpuSampler     cpuBusyReader
	cpuPower       cpuPowerReader
	ramSampler     ramReader
	sensorsSampler lmSensorsReader
	gpuBusySampler gpuBusyReader
	gpuVRAMSampler gpuVRAMReader
}

type Snapshot struct {
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

func New(
	cpuSampler *sensors.CPUBusySampler,
	cpuPower *sensors.CPUPowerSampler,
	ramSampler *sensors.SystemRAMSampler,
	sensorsSampler *sensors.LmSensorsSampler,
	gpuBusySampler *sensors.GPUBusySampler,
	gpuVRAMSampler *sensors.GPUVRAMSampler,
) *Metrics {
	return newWithDeps(
		cpuSampler,
		cpuPower,
		ramSampler,
		sensorsSampler,
		gpuBusySampler,
		gpuVRAMSampler,
	)
}

func newWithDeps(
	cpuSampler cpuBusyReader,
	cpuPower cpuPowerReader,
	ramSampler ramReader,
	sensorsSampler lmSensorsReader,
	gpuBusySampler gpuBusyReader,
	gpuVRAMSampler gpuVRAMReader,
) *Metrics {
	return &Metrics{
		cpuSampler:     cpuSampler,
		cpuPower:       cpuPower,
		ramSampler:     ramSampler,
		sensorsSampler: sensorsSampler,
		gpuBusySampler: gpuBusySampler,
		gpuVRAMSampler: gpuVRAMSampler,
	}
}

func (m *Metrics) buildSnapshot() Snapshot {
	var resp Snapshot

	sensorSnapshot := m.sensorsSampler.Snapshot()
	resp.CPU.TempC = sensorSnapshot.CPUTempC
	resp.GPU.EdgeC = sensorSnapshot.GPUEdgeC
	resp.GPU.HotspotC = sensorSnapshot.GPUHotspotC
	resp.GPU.VramC = sensorSnapshot.GPUVramC
	resp.GPU.PowerW = sensorSnapshot.GPUPowerW

	resp.CPU.UtilPct = m.cpuSampler.Snapshot().UtilPct
	resp.CPU.PowerW = m.cpuPower.Snapshot().PowerW
	resp.GPU.UtilPct = m.gpuBusySampler.Snapshot().UtilPct
	gpuVRAMSnapshot := m.gpuVRAMSampler.Snapshot()
	resp.GPU.VramUsedGB = gpuVRAMSnapshot.UsedGB
	resp.GPU.VramTotalGB = gpuVRAMSnapshot.TotalGB
	resp.GPU.VramUsedPct = gpuVRAMSnapshot.UsedPct

	ramSnapshot, err := m.ramSampler.Snapshot()
	if err == nil {
		resp.RAM.TotalGB = ramSnapshot.TotalGB
		resp.RAM.UsedGB = ramSnapshot.UsedGB
		resp.RAM.AvailGB = ramSnapshot.AvailGB
		resp.RAM.UsedPct = ramSnapshot.UsedPct
	}

	return resp
}
