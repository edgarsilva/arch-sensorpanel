package sensors

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestReadUintFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "value")
	if err := os.WriteFile(path, []byte("12345\n"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	got, err := readUintFromFile(path)
	if err != nil {
		t.Fatalf("readUintFromFile error: %v", err)
	}
	if got != 12345 {
		t.Fatalf("readUintFromFile got %d, want 12345", got)
	}
}

func TestReadGPUBusy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gpu_busy_percent")
	if err := os.WriteFile(path, []byte("77.5\n"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	got, err := readGPUBusy(path)
	if err != nil {
		t.Fatalf("readGPUBusy error: %v", err)
	}
	if got != 77.5 {
		t.Fatalf("readGPUBusy got %v, want 77.5", got)
	}
}

func TestReadEnergy(t *testing.T) {
	dir := t.TempDir()
	energyPath := filepath.Join(dir, "energy_uj")
	maxPath := filepath.Join(dir, "max_energy_range_uj")

	if err := os.WriteFile(energyPath, []byte("2500000\n"), 0o644); err != nil {
		t.Fatalf("write energy file: %v", err)
	}
	if err := os.WriteFile(maxPath, []byte("5000000\n"), 0o644); err != nil {
		t.Fatalf("write max file: %v", err)
	}

	energy, max, err := readEnergy(energyPath, maxPath)
	if err != nil {
		t.Fatalf("readEnergy error: %v", err)
	}
	if energy != 2500000 || max != 5000000 {
		t.Fatalf("readEnergy got energy=%d max=%d", energy, max)
	}
}

func TestReadVRAMSnapshot(t *testing.T) {
	dir := t.TempDir()
	usedPath := filepath.Join(dir, "used")
	totalPath := filepath.Join(dir, "total")

	if err := os.WriteFile(usedPath, []byte("2147483648\n"), 0o644); err != nil {
		t.Fatalf("write used file: %v", err)
	}
	if err := os.WriteFile(totalPath, []byte("8589934592\n"), 0o644); err != nil {
		t.Fatalf("write total file: %v", err)
	}

	snapshot, err := readVRAMSnapshot(usedPath, totalPath)
	if err != nil {
		t.Fatalf("readVRAMSnapshot error: %v", err)
	}

	if math.Abs(snapshot.UsedGB-2.0) > 1e-9 {
		t.Fatalf("UsedGB got %v, want 2", snapshot.UsedGB)
	}
	if math.Abs(snapshot.TotalGB-8.0) > 1e-9 {
		t.Fatalf("TotalGB got %v, want 8", snapshot.TotalGB)
	}
	if math.Abs(snapshot.UsedPct-25.0) > 1e-9 {
		t.Fatalf("UsedPct got %v, want 25", snapshot.UsedPct)
	}
}
