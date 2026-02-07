// Package sensors contains sensor metrics,
// like cpu/gpu utilization, temperatures, power draw, etc.
package sensors

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CPUPowerSampler struct {
	mu         sync.RWMutex
	lastEnergy uint64
	powerW     float64
	energyPath string
	maxPath    string
}

func NewCPUPowerSampler(interval time.Duration) *CPUPowerSampler {
	path := detectRAPLPackagePath()
	s := &CPUPowerSampler{}
	if path != "" {
		s.energyPath = filepath.Join(path, "energy_uj")
		s.maxPath = filepath.Join(path, "max_energy_range_uj")
		go s.run(interval)
	}

	return s
}

func (s *CPUPowerSampler) run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		energy, max, err := readEnergy(s.energyPath, s.maxPath)
		if err != nil {
			continue
		}

		s.mu.Lock()
		if s.lastEnergy != 0 {
			delta := energy - s.lastEnergy
			if energy < s.lastEnergy && max > 0 {
				delta = (max - s.lastEnergy) + energy
			}
			s.powerW = float64(delta) / interval.Seconds() / 1_000_000.0
		}
		s.lastEnergy = energy
		s.mu.Unlock()
	}
}

func (s *CPUPowerSampler) PowerW() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.powerW
}

func detectRAPLPackagePath() string {
	name := "intel-rapl:0"
	return filepath.Join("/sys/class/powercap", name)
}

func readEnergy(energyPath string, maxPath string) (uint64, uint64, error) {
	energyRaw, err := os.ReadFile(energyPath)
	if err != nil {
		return 0, 0, err
	}
	maxRaw, err := os.ReadFile(maxPath)
	if err != nil {
		return 0, 0, err
	}

	fmt.Println("energyRaw ->", string(energyRaw))
	fmt.Println("maxRaw ->", string(maxRaw))

	energy, err := strconv.ParseUint(strings.TrimSpace(string(energyRaw)), 10, 64)
	if err != nil {
		return 0, 0, err
	}
	maxEnergy, err := strconv.ParseUint(strings.TrimSpace(string(maxRaw)), 10, 64)
	if err != nil {
		return 0, 0, err
	}

	return energy, maxEnergy, nil
}
