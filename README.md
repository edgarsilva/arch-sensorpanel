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
- GPU VRAM usage (used/total)
- GPU power draw (watts)
- Normalized JSON API
- DaisyUI-based HUD overlay
- Ultrawide / 8.8-inch display friendly
- Kiosk-safe (no focus stealing, non-interactive)
- Metrics are sampled on an interval and served from cached snapshots

---------------------------------------------------------------------

ARCHITECTURE

lm-sensors + sysfs
        |
        v
Go samplers
        |
        v
Go / Fiber API  (/metrics)
        |
        v
Browser HUD overlay (DaisyUI)

Design rationale:

- Hardware access stays in Go samplers for fewer dependencies
- Go server returns normalized JSON
- UI consumes stable, normalized JSON
- Any layer can be replaced independently

---------------------------------------------------------------------

PROJECT LAYOUT

.
├── main.go
├── public/
│   └── index.html
├── README.md
└── LICENSE

---------------------------------------------------------------------

REQUIREMENTS

System:

- lm-sensors
- AMD GPU using the amdgpu driver
- /sys/class/drm available

Go:

- Go 1.21 or newer
- Fiber v2

---------------------------------------------------------------------

SETUP

1) Install dependencies

sudo pacman -S lm_sensors
sudo sensors-detect

1) Run the server

From the project root:

go run .

The API will be available at:

<http://localhost:9070/metrics>

---------------------------------------------------------------------

API

GET /metrics

Example response:

{
  "cpu": {
    "temp_c": 38.6,
    "util_pct": 12.4,
    "power_w": 22.1
  },
  "ram": {
    "total_gb": 31.9,
    "used_gb": 11.4,
    "avail_gb": 20.5,
    "used_pct": 35.7
  },
  "gpu": {
    "edge_c": 48,
    "hotspot_c": 57,
    "vram_c": 74,
    "power_w": 42,
    "util_pct": 18
  }
}

---------------------------------------------------------------------

HUD OVERLAY

- Built with DaisyUI
- Designed to float over fullscreen video
- Uses stat, progress, and radial-progress components
- Fully themeable

Recommended theme for kiosk use:

```html

<html data-theme="winter">

```

---------------------------------------------------------------------

GPU UTILIZATION (OPTIONAL)

GPU utilization is not provided by lm-sensors.

On AMD systems it is read from:

/sys/class/drm/card0/device/gpu_busy_percent

---------------------------------------------------------------------

SECURITY NOTES

- Intended for local or localhost use
- Do not expose publicly without authentication
- Sensor collection is performed in-process by Go samplers

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
