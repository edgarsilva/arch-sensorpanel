package sensors

import "testing"

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
