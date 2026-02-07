#!/usr/bin/env bash
set -euo pipefail

# -----------------------------
# GPU UTILIZATION (AMD)
# -----------------------------
gpu_busy_file="$(ls /sys/class/drm/card*/device/gpu_busy_percent 2>/dev/null | head -n1)"
gpu_util_pct=0
if [[ -n "$gpu_busy_file" ]]; then
  gpu_util_pct="$(cat "$gpu_busy_file")"
fi

# -----------------------------
# TEMPERATURES & POWER (lm-sensors)
# -----------------------------
sensors -j | jq --arg gpu_util "$gpu_util_pct" '
{
  cpu: {
    temp_c: .["k10temp-pci-00c3"].Tctl.temp1_input
  },
  gpu: {
    edge_c: .["amdgpu-pci-0e00"].edge.temp1_input,
    hotspot_c: .["amdgpu-pci-0e00"].junction.temp2_input,
    vram_c: .["amdgpu-pci-0e00"].mem.temp3_input,
    power_w: .["amdgpu-pci-0e00"].PPT.power1_average,
    util_pct: ($gpu_util | tonumber)
  }
}'
