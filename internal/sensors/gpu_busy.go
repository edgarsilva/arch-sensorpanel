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

type GPUBusySampler struct {
	mu      sync.RWMutex
	utilPct float64
	path    string
}

func NewGPUBusySampler(interval time.Duration) *GPUBusySampler {
	path := detectGPUBusyPath()
	s := &GPUBusySampler{path: path}
	if path != "" {
		go s.run(interval)
	}

	return s
}

func (s *GPUBusySampler) run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		util, err := readGPUBusy(s.path)
		if err != nil {
			continue
		}

		s.mu.Lock()
		s.utilPct = util
		s.mu.Unlock()
	}
}

func (s *GPUBusySampler) Utilization() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.utilPct
}

func detectGPUBusyPath() string {
	matches, err := filepath.Glob("/sys/class/drm/card*/device/gpu_busy_percent")
	if err != nil || len(matches) == 0 {
		return ""
	}

	return matches[0]
}

func readGPUBusy(path string) (float64, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	value := strings.TrimSpace(string(raw))
	utilPct, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}

	return utilPct, nil
}
