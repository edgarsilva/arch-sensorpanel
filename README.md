# Sensor Panel HUD

A lightweight Linux hardware sensor panel and HUD overlay designed for kiosk and
dashboard setups. It exposes normalized CPU and GPU metrics via a small Go
(Fiber) API and renders a minimal, themeable DaisyUI overlay on top of fullscreen
video (for example, YouTube).

Tested on:

- Arch Linux
- Hyprland (Wayland)
- AMD Ryzen CPU
- AMD GPU (amdgpu)
- lm-sensors

---------------------------------------------------------------------

FEATURES

- CPU temperature (AMD Tctl)
- GPU temperatures (edge, hotspot, VRAM)
- GPU power draw (watts)
- Normalized JSON API
- DaisyUI-based HUD overlay
- Ultrawide / 8.8-inch display friendly
- Kiosk-safe (no focus stealing, non-interactive)

---------------------------------------------------------------------

ARCHITECTURE

lm-sensors + sysfs
        |
        v
sensor-panel-metrics.sh
        |
        v
Go / Fiber API  (/api/sensors)
        |
        v
Browser HUD overlay (DaisyUI)

Design rationale:

- Hardware access stays in Bash for simplicity and flexibility
- Go server is transport-only
- UI consumes stable, normalized JSON
- Any layer can be replaced independently

---------------------------------------------------------------------

PROJECT LAYOUT

.
├── main.go
├── public/
│   └── index.html
├── scripts/
│   └── metrics_collector.sh
├── README.md
└── LICENSE

---------------------------------------------------------------------

REQUIREMENTS

System:

- lm-sensors
- jq
- AMD GPU using the amdgpu driver
- /sys/class/drm available

Go:

- Go 1.21 or newer
- Fiber v2

---------------------------------------------------------------------

SETUP

1) Install dependencies

sudo pacman -S lm_sensors jq
sudo sensors-detect

1) Make the metrics script executable

chmod +x sensor-panel-metrics.sh

Test it:

./sensor-panel-metrics.sh | jq

1) Run the server

From the project root:

go run .

The API will be available at:

<http://localhost:3000/api/sensors>

---------------------------------------------------------------------

API

GET /api/sensors

Example response:

{
  "cpu": {
    "temp_c": 38.6
  },
  "gpu": {
    "edge_c": 48,
    "hotspot_c": 57,
    "vram_c": 74,
    "power_w": 42
  }
}

---------------------------------------------------------------------

HUD OVERLAY

- Built with DaisyUI
- Designed to float over fullscreen video
- Uses stat, progress, and radial-progress components
- Fully themeable

Recommended theme for kiosk use:

<html data-theme="winter">

---------------------------------------------------------------------

GPU UTILIZATION (OPTIONAL)

GPU utilization is not provided by lm-sensors.

On AMD systems it can be read from:

/sys/class/drm/card0/device/gpu_busy_percent

This can be added to the Bash script or read directly in Go.

---------------------------------------------------------------------

SECURITY NOTES

- Intended for local or localhost use
- Do not expose publicly without authentication
- Script execution is time-limited in the Go handler

---------------------------------------------------------------------

LICENSE

This project is released under the MIT License.
See the LICENSE file for details.

---------------------------------------------------------------------

FUTURE IMPROVEMENTS

- WebSocket streaming instead of polling
- EMA smoothing for sensor values
- CPU power (RAPL) integration
- Multi-GPU support
- systemd user service
- Auto-restart kiosk browser

---------------------------------------------------------------------

AUTHOR

Built for a Hyprland-based kiosk and sensor panel workflow.
