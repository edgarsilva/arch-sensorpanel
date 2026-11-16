package sensors

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMemInfoKB(t *testing.T) {
	tests := []struct {
		name string
		line string
		want uint64
	}{
		{name: "valid", line: "MemTotal:       32767584 kB", want: 32767584},
		{name: "missing value", line: "MemTotal:", want: 0},
		{name: "invalid value", line: "MemTotal:       nope kB", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseMemInfoKB(tt.line)
			if got != tt.want {
				t.Fatalf("parseMemInfoKB(%q)=%d, want %d", tt.line, got, tt.want)
			}
		})
	}
}

func TestReadGPUReclaimKB(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write temp file: %v", err)
		}

		return path
	}

	tests := []struct {
		name string
		path string
		want uint64
	}{
		{name: "pool present", path: write("present", "nr_free_pages 100\nnr_gpu_reclaim 4718592\nnr_dirty 3\n"), want: 4718592 * 4},
		{name: "counter missing", path: write("missing", "nr_free_pages 100\nnr_dirty 3\n"), want: 0},
		{name: "invalid value", path: write("invalid", "nr_gpu_reclaim nope\n"), want: 0},
		{name: "file unreadable", path: filepath.Join(dir, "does-not-exist"), want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readGPUReclaimKB(tt.path); got != tt.want {
				t.Fatalf("readGPUReclaimKB()=%d, want %d", got, tt.want)
			}
		})
	}
}

func TestAddGPUReclaim(t *testing.T) {
	tests := []struct {
		name                     string
		total, avail, gpuReclaim uint64
		want                     uint64
	}{
		{name: "adds pool", total: 100, avail: 50, gpuReclaim: 20, want: 70},
		{name: "no pool", total: 100, avail: 50, gpuReclaim: 0, want: 50},
		{name: "capped at total", total: 100, avail: 90, gpuReclaim: 30, want: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := addGPUReclaim(tt.total, tt.avail, tt.gpuReclaim); got != tt.want {
				t.Fatalf("addGPUReclaim()=%d, want %d", got, tt.want)
			}
		})
	}
}
