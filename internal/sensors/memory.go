// Package sensors contains sensor metrics,
// like cpu/gpu utilization, temperatures, power draw, etc.
package sensors

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type MemorySnapshot struct {
	TotalGB float64
	UsedGB  float64
	AvailGB float64
	UsedPct float64
}

type MemorySampler struct{}

func NewMemorySampler() *MemorySampler {
	return &MemorySampler{}
}

func (s *MemorySampler) Snapshot() (MemorySnapshot, error) {
	totalKB, availKB, err := readMemInfo()
	if err != nil {
		return MemorySnapshot{}, err
	}

	totalGB := float64(totalKB) / (1024.0 * 1024.0)
	availGB := float64(availKB) / (1024.0 * 1024.0)
	usedGB := totalGB - availGB

	usedPct := 0.0
	if totalGB > 0 {
		usedPct = 100.0 * usedGB / totalGB
	}

	return MemorySnapshot{
		TotalGB: totalGB,
		UsedGB:  usedGB,
		AvailGB: availGB,
		UsedPct: usedPct,
	}, nil
}

func readMemInfo() (totalKB uint64, availKB uint64, err error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			fmt.Println(err)
		}
	}()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			totalKB = parseMemInfoKB(line)
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			availKB = parseMemInfoKB(line)
		}
		if totalKB > 0 && availKB > 0 {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}

	return totalKB, availKB, nil
}

func parseMemInfoKB(line string) uint64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}

	value, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}

	return value
}
