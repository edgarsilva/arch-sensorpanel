// Package sensors contains sensor metrics,
// like cpu/gpu utilization, temperatures, power draw, etc.
package sensors

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type GPUVRAMSnapshot struct {
	TotalGB float64
	UsedGB  float64
	UsedPct float64
}

type GPUVRAMSampler struct {
	mu        sync.RWMutex
	snapshot  GPUVRAMSnapshot
	usedPath  string
	totalPath string
}

func NewGPUVRAMSampler(interval time.Duration) *GPUVRAMSampler {
	usedPath, totalPath := detectVRAMPaths()
	s := &GPUVRAMSampler{usedPath: usedPath, totalPath: totalPath}
	if usedPath != "" && totalPath != "" {
		go s.run(interval)
	}

	return s
}

func (s *GPUVRAMSampler) run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		snapshot, err := readVRAMSnapshot(s.usedPath, s.totalPath)
		if err != nil {
			continue
		}

		s.mu.Lock()
		s.snapshot = snapshot
		s.mu.Unlock()
	}
}

func (s *GPUVRAMSampler) Snapshot() GPUVRAMSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot
}

func detectVRAMPaths() (string, string) {
	usedMatches, err := filepath.Glob("/sys/class/drm/card*/device/mem_info_vram_used")
	if err != nil || len(usedMatches) == 0 {
		return "", ""
	}

	totalMatches, err := filepath.Glob("/sys/class/drm/card*/device/mem_info_vram_total")
	if err != nil || len(totalMatches) == 0 {
		return "", ""
	}

	return usedMatches[0], totalMatches[0]
}

func readVRAMSnapshot(usedPath string, totalPath string) (GPUVRAMSnapshot, error) {
	usedBytes, err := readUintFromFile(usedPath)
	if err != nil {
		return GPUVRAMSnapshot{}, err
	}

	totalBytes, err := readUintFromFile(totalPath)
	if err != nil {
		return GPUVRAMSnapshot{}, err
	}

	usedGB := float64(usedBytes) / (1024.0 * 1024.0 * 1024.0)
	totalGB := float64(totalBytes) / (1024.0 * 1024.0 * 1024.0)
	usedPct := 0.0
	if totalGB > 0 {
		usedPct = 100.0 * usedGB / totalGB
	}

	return GPUVRAMSnapshot{
		TotalGB: totalGB,
		UsedGB:  usedGB,
		UsedPct: usedPct,
	}, nil
}

func readUintFromFile(path string) (uint64, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	value := strings.TrimSpace(string(raw))
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}

	return parsed, nil
}
