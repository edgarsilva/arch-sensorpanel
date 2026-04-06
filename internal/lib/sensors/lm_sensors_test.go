package sensors

import (
	"encoding/json"
	"testing"
)

func TestParseSensorValue(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  float64
		ok    bool
	}{
		{name: "json number", input: json.Number("42.5"), want: 42.5, ok: true},
		{name: "float", input: 12.25, want: 12.25, ok: true},
		{name: "string", input: "9.75", want: 9.75, ok: true},
		{name: "bad string", input: "abc", want: 0, ok: false},
		{name: "unsupported", input: true, want: 0, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseSensorValue(tt.input)
			if ok != tt.ok {
				t.Fatalf("parseSensorValue ok=%v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("parseSensorValue value=%v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindChip(t *testing.T) {
	data := map[string]any{
		"k10temp-pci-00c3": map[string]any{"Tctl": map[string]any{"temp1_input": 68.0}},
		"amdgpu-pci-0100":  map[string]any{"edge": map[string]any{"temp1_input": 55.0}},
	}

	chip, ok := findChip(data, "k10temp")
	if !ok {
		t.Fatal("expected to find k10temp chip")
	}
	if _, hasTctl := chip["Tctl"]; !hasTctl {
		t.Fatal("expected k10temp chip to include Tctl")
	}

	_, ok = findChip(data, "nouveau")
	if ok {
		t.Fatal("did not expect to find nouveau chip")
	}
}

func TestFindFirstValue(t *testing.T) {
	chip := map[string]any{
		"Tctl": map[string]any{"temp1_input": json.Number("67.5")},
		"Tdie": map[string]any{"temp1_input": json.Number("62.0")},
	}

	got := findFirstValue(chip, []string{"Tctl", "Tdie"}, "temp1_input")
	if got != 67.5 {
		t.Fatalf("findFirstValue got %v, want 67.5", got)
	}

	missing := findFirstValue(chip, []string{"edge"}, "temp1_input")
	if missing != 0 {
		t.Fatalf("findFirstValue missing got %v, want 0", missing)
	}
}

func TestAMDPackageTempSelectionPrefersTdie(t *testing.T) {
	chip := map[string]any{
		"Tctl": map[string]any{"temp1_input": json.Number("72.0")},
		"Tdie": map[string]any{"temp1_input": json.Number("65.5")},
	}

	got := findFirstValue(chip, []string{"Tdie", "Tctl"}, "temp1_input")
	if got != 65.5 {
		t.Fatalf("AMD package temp got %v, want 65.5", got)
	}
}

func TestAMDPackageTempSelectionFallsBackToTctl(t *testing.T) {
	chip := map[string]any{
		"Tctl": map[string]any{"temp1_input": json.Number("71.25")},
	}

	got := findFirstValue(chip, []string{"Tdie", "Tctl"}, "temp1_input")
	if got != 71.25 {
		t.Fatalf("AMD package temp fallback got %v, want 71.25", got)
	}
}
