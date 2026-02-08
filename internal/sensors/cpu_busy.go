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

type CPUBusySampler struct {
	mu        sync.RWMutex
	lastIdle  uint64
	lastTotal uint64
	utilPct   float64
}

func NewCPUBusySampler(interval time.Duration) *CPUBusySampler {
	s := &CPUBusySampler{}
	go s.run(interval)

	return s
}

func (s *CPUBusySampler) run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		idle, total, err := readProcStat()
		if err != nil {
			continue
		}

		s.mu.Lock()
		if s.lastTotal != 0 {
			idleDelta := idle - s.lastIdle
			totalDelta := total - s.lastTotal
			if totalDelta > 0 {
				s.utilPct = 100.0 * float64(totalDelta-idleDelta) / float64(totalDelta)
			}
		}
		s.lastIdle = idle
		s.lastTotal = total
		s.mu.Unlock()
	}
}

func (s *CPUBusySampler) Utilization() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.utilPct
}

func readProcStat() (idle uint64, total uint64, err error) {
	f, err := os.Open("/proc/stat")
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
	if !scanner.Scan() {
		return 0, 0, scanner.Err()
	}

	fields := strings.Fields(scanner.Text())
	if len(fields) < 8 {
		return 0, 0, nil
	}

	var values []uint64
	for _, v := range fields[1:8] {
		n, _ := strconv.ParseUint(v, 10, 64)
		values = append(values, n)
	}

	idle = values[3]
	for _, v := range values {
		total += v
	}

	return idle, total, nil
}
