package metrics

import (
	"time"

	"sensorpanel/internal/lib/sensors"
	"sensorpanel/internal/server"
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

type Service struct {
	*server.Server
	sampleInterval time.Duration

	cpuSampler     cpuBusyReader
	cpuPower       cpuPowerReader
	ramSampler     ramReader
	sensorsSampler lmSensorsReader
	gpuBusySampler gpuBusyReader
	gpuVRAMSampler gpuVRAMReader
}

type Option func(*Service)

type Snapshot struct {
	CPU struct {
		TempC        float64 `json:"temp_c"`
		PackageTempC float64 `json:"package_temp_c"`
		UtilPct      float64 `json:"util_pct"`
		PowerW       float64 `json:"power_w"`
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

func New(s *server.Server, opts ...Option) *Service {
	svc := &Service{
		Server:         s,
		sampleInterval: 1 * time.Second,
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(svc)
	}

	if svc.sampleInterval <= 0 {
		svc.sampleInterval = time.Second
	}

	if svc.cpuSampler == nil {
		svc.cpuSampler = sensors.NewCPUBusySampler(svc.sampleInterval)
	}
	if svc.cpuPower == nil {
		svc.cpuPower = sensors.NewCPUPowerSampler(svc.sampleInterval)
	}
	if svc.ramSampler == nil {
		svc.ramSampler = sensors.NewSystemRAMSampler(svc.sampleInterval)
	}
	if svc.sensorsSampler == nil {
		svc.sensorsSampler = sensors.NewLmSensorsSampler(svc.sampleInterval)
	}
	if svc.gpuBusySampler == nil {
		svc.gpuBusySampler = sensors.NewGPUBusySampler(svc.sampleInterval)
	}
	if svc.gpuVRAMSampler == nil {
		svc.gpuVRAMSampler = sensors.NewGPUVRAMSampler(svc.sampleInterval)
	}

	return newWithDeps(
		s,
		svc.sampleInterval,
		svc.cpuSampler,
		svc.cpuPower,
		svc.ramSampler,
		svc.sensorsSampler,
		svc.gpuBusySampler,
		svc.gpuVRAMSampler,
	)
}

func WithSampleInterval(interval time.Duration) Option {
	return func(s *Service) {
		s.sampleInterval = interval
	}
}

func newWithDeps(
	s *server.Server,
	sampleInterval time.Duration,
	cpuSampler cpuBusyReader,
	cpuPower cpuPowerReader,
	ramSampler ramReader,
	sensorsSampler lmSensorsReader,
	gpuBusySampler gpuBusyReader,
	gpuVRAMSampler gpuVRAMReader,
) *Service {
	return &Service{
		Server:         s,
		sampleInterval: sampleInterval,
		cpuSampler:     cpuSampler,
		cpuPower:       cpuPower,
		ramSampler:     ramSampler,
		sensorsSampler: sensorsSampler,
		gpuBusySampler: gpuBusySampler,
		gpuVRAMSampler: gpuVRAMSampler,
	}
}

func (m *Service) buildSnapshot() Snapshot {
	var resp Snapshot

	sensorSnapshot := m.sensorsSampler.Snapshot()
	resp.CPU.TempC = sensorSnapshot.CPUTempC
	resp.CPU.PackageTempC = sensorSnapshot.CPUPackageTempC
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
