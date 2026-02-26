// Package sensors contains sensor metrics,
// like cpu/gpu utilization, temperatures, power draw, etc.
package sensors

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type LmSensorsSnapshot struct {
	CPUTempC    float64
	GPUEdgeC    float64
	GPUHotspotC float64
	GPUVramC    float64
	GPUPowerW   float64
}

type LmSensorsSampler struct {
	mu       sync.RWMutex
	snapshot LmSensorsSnapshot
}

func NewLmSensorsSampler(interval time.Duration) *LmSensorsSampler {
	s := &LmSensorsSampler{}
	go s.run(interval)

	return s
}

func (s *LmSensorsSampler) run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		snapshot, err := readLmSensors()
		if err != nil {
			continue
		}

		s.mu.Lock()
		s.snapshot = *snapshot
		s.mu.Unlock()
	}
}

func (s *LmSensorsSampler) Snapshot() LmSensorsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot
}

func readLmSensors() (*LmSensorsSnapshot, error) {
	cmd := exec.Command("sensors", "-j")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(output))
	decoder.UseNumber()

	var data map[string]any
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	snapshot := &LmSensorsSnapshot{}

	if chip, ok := findChip(data, "k10temp"); ok {
		snapshot.CPUTempC = findFirstValue(chip, []string{"Tctl", "Tdie"}, "temp1_input")
	} else if chip, ok := findChip(data, "coretemp"); ok {
		snapshot.CPUTempC = findFirstValue(chip, []string{"Package id 0", "Core 0"}, "temp1_input")
	}

	if chip, ok := findChip(data, "amdgpu"); ok {
		snapshot.GPUEdgeC = findFirstValue(chip, []string{"edge"}, "temp1_input")
		snapshot.GPUHotspotC = findFirstValue(chip, []string{"junction"}, "temp2_input")
		snapshot.GPUVramC = findFirstValue(chip, []string{"mem"}, "temp3_input")
		snapshot.GPUPowerW = findFirstValue(chip, []string{"PPT"}, "power1_average")
	}

	return snapshot, nil
}

func findChip(data map[string]any, prefixes ...string) (map[string]any, bool) {
	for key, value := range data {
		for _, prefix := range prefixes {
			if strings.HasPrefix(key, prefix) {
				if chip, ok := value.(map[string]any); ok {
					return chip, true
				}
			}
		}
	}

	return nil, false
}

func findFirstValue(chip map[string]any, sections []string, field string) float64 {
	for _, section := range sections {
		sectionData, ok := chip[section].(map[string]any)
		if !ok {
			continue
		}

		if value, ok := parseSensorValue(sectionData[field]); ok {
			return value
		}
	}

	return 0
}

func parseSensorValue(value any) (float64, bool) {
	switch v := value.(type) {
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	case float64:
		return v, true
	case string:
		f, err := json.Number(v).Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}
