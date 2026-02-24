package metrics

import (
	"errors"
	"sensorpanel/internal/sensors"
	"sensorpanel/internal/server"
	"testing"
	"time"
)

type fakeCPUBusy struct {
	util float64
}

func (f fakeCPUBusy) Snapshot() sensors.CPUBusySnapshot {
	return sensors.CPUBusySnapshot{UtilPct: f.util}
}

type fakeCPUPower struct {
	power float64
}

func (f fakeCPUPower) Snapshot() sensors.CPUPowerSnapshot {
	return sensors.CPUPowerSnapshot{PowerW: f.power}
}

type fakeRAM struct {
	snapshot sensors.SystemRAMSnapshot
	err      error
}

func (f fakeRAM) Snapshot() (sensors.SystemRAMSnapshot, error) {
	return f.snapshot, f.err
}

type fakeLmSensors struct {
	snapshot sensors.LmSensorsSnapshot
}

func (f fakeLmSensors) Snapshot() sensors.LmSensorsSnapshot {
	return f.snapshot
}

type fakeGPUBusy struct {
	util float64
}

func (f fakeGPUBusy) Snapshot() sensors.GPUBusySnapshot {
	return sensors.GPUBusySnapshot{UtilPct: f.util}
}

type fakeGPUVRAM struct {
	snapshot sensors.GPUVRAMSnapshot
}

func (f fakeGPUVRAM) Snapshot() sensors.GPUVRAMSnapshot {
	return f.snapshot
}

func TestBuildSnapshotMapsAllValues(t *testing.T) {
	m := newWithDeps(
		&server.Server{},
		time.Second,
		fakeCPUBusy{util: 33.3},
		fakeCPUPower{power: 45.6},
		fakeRAM{snapshot: sensors.SystemRAMSnapshot{TotalGB: 32, UsedGB: 14, AvailGB: 18, UsedPct: 43.75}},
		fakeLmSensors{snapshot: sensors.LmSensorsSnapshot{CPUTempC: 70.1, GPUEdgeC: 61.2, GPUHotspotC: 75.3, GPUVramC: 79.4, GPUPowerW: 210.5}},
		fakeGPUBusy{util: 88.8},
		fakeGPUVRAM{snapshot: sensors.GPUVRAMSnapshot{UsedGB: 7.5, TotalGB: 16, UsedPct: 46.875}},
	)

	s := m.buildSnapshot()

	if s.CPU.TempC != 70.1 {
		t.Fatalf("CPU temp mismatch: got %v", s.CPU.TempC)
	}
	if s.CPU.UtilPct != 33.3 {
		t.Fatalf("CPU util mismatch: got %v", s.CPU.UtilPct)
	}
	if s.CPU.PowerW != 45.6 {
		t.Fatalf("CPU power mismatch: got %v", s.CPU.PowerW)
	}

	if s.RAM.TotalGB != 32 || s.RAM.UsedGB != 14 || s.RAM.AvailGB != 18 || s.RAM.UsedPct != 43.75 {
		t.Fatalf("RAM snapshot mismatch: got %+v", s.RAM)
	}

	if s.GPU.EdgeC != 61.2 || s.GPU.HotspotC != 75.3 || s.GPU.VramC != 79.4 {
		t.Fatalf("GPU temps mismatch: got %+v", s.GPU)
	}
	if s.GPU.PowerW != 210.5 || s.GPU.UtilPct != 88.8 {
		t.Fatalf("GPU util/power mismatch: got %+v", s.GPU)
	}
	if s.GPU.VramUsedGB != 7.5 || s.GPU.VramTotalGB != 16 || s.GPU.VramUsedPct != 46.875 {
		t.Fatalf("GPU VRAM mismatch: got %+v", s.GPU)
	}
}

func TestBuildSnapshotKeepsRAMZeroWhenSamplerFails(t *testing.T) {
	m := newWithDeps(
		&server.Server{},
		time.Second,
		fakeCPUBusy{util: 10},
		fakeCPUPower{power: 20},
		fakeRAM{err: errors.New("ram unavailable")},
		fakeLmSensors{snapshot: sensors.LmSensorsSnapshot{CPUTempC: 50, GPUEdgeC: 55, GPUPowerW: 100}},
		fakeGPUBusy{util: 30},
		fakeGPUVRAM{snapshot: sensors.GPUVRAMSnapshot{UsedGB: 4, TotalGB: 8, UsedPct: 50}},
	)

	s := m.buildSnapshot()

	if s.RAM.TotalGB != 0 || s.RAM.UsedGB != 0 || s.RAM.AvailGB != 0 || s.RAM.UsedPct != 0 {
		t.Fatalf("expected zero RAM when sampler errors, got %+v", s.RAM)
	}
	if s.CPU.TempC != 50 || s.CPU.UtilPct != 10 || s.CPU.PowerW != 20 {
		t.Fatalf("non-RAM fields should still map, got CPU=%+v", s.CPU)
	}
	if s.GPU.UtilPct != 30 || s.GPU.PowerW != 100 || s.GPU.VramUsedPct != 50 {
		t.Fatalf("non-RAM fields should still map, got GPU=%+v", s.GPU)
	}
}
