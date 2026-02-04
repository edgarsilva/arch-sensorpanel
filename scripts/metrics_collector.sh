#!/usr/bin/env bash
set -euo pipefail

# Collect and normalize lm-sensors output for the sensor panel.
# Output is stable JSON intended for HTTP consumption.

sensors -j | jq '{
  cpu: {
    temp_c: .["k10temp-pci-00c3"].Tctl.temp1_input
  },
  gpu: {
    edge_c: .["amdgpu-pci-0e00"].edge.temp1_input,
    hotspot_c: .["amdgpu-pci-0e00"].junction.temp2_input,
    vram_c: .["amdgpu-pci-0e00"].mem.temp3_input,
    power_w: .["amdgpu-pci-0e00"].PPT.power1_average
  }
}'
