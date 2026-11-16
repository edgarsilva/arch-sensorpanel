// Package sensors contains sensor metrics,
// like cpu/gpu utilization, temperatures, power draw, etc.
package sensors

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SystemRAMSnapshot struct {
	TotalGB float64
	UsedGB  float64
	AvailGB float64
	UsedPct float64
}

type SystemRAMSampler struct {
	mu       sync.RWMutex
	snapshot SystemRAMSnapshot
}

func NewSystemRAMSampler(interval time.Duration) *SystemRAMSampler {
	s := &SystemRAMSampler{}
	go s.run(interval)

	return s
}

func (s *SystemRAMSampler) run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		snapshot, err := readMemorySnapshot()
		if err != nil {
			continue
		}

		s.mu.Lock()
		s.snapshot = snapshot
		s.mu.Unlock()
	}
}

func (s *SystemRAMSampler) Snapshot() (SystemRAMSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot, nil
}

func readMemorySnapshot() (SystemRAMSnapshot, error) {
	totalKB, availKB, err := readMemInfo()
	if err != nil {
		return SystemRAMSnapshot{}, err
	}

	availKB = addGPUReclaim(totalKB, availKB, readGPUReclaimKB("/proc/vmstat"))

	totalGB := float64(totalKB) / (1024.0 * 1024.0)
	availGB := float64(availKB) / (1024.0 * 1024.0)
	usedGB := totalGB - availGB

	usedPct := 0.0
	if totalGB > 0 {
		usedPct = 100.0 * usedGB / totalGB
	}

	return SystemRAMSnapshot{
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

// addGPUReclaim adds the amdgpu reclaim pool to available memory, capped at
// total. MemAvailable does not count it, so without this the pool reads as used.
func addGPUReclaim(totalKB, availKB, gpuReclaimKB uint64) uint64 {
	if availKB+gpuReclaimKB > totalKB {
		return totalKB
	}

	return availKB + gpuReclaimKB
}

// readGPUReclaimKB returns the size of the amdgpu/TTM freed-page pool
// (nr_gpu_reclaim in /proc/vmstat, kernel 7.2+) in KiB. The pool is
// reclaimable, but the kernel leaves it out of MemAvailable. It returns 0 when
// the file or the counter is missing, such as on older kernels.
func readGPUReclaimKB(path string) uint64 {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 || fields[0] != "nr_gpu_reclaim" {
			continue
		}

		pages, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0
		}

		return pages * 4 // vmstat counts 4 KiB pages
	}

	return 0
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
