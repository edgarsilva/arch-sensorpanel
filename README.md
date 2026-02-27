# Sensor Panel HUD

Sensor Panel HUD is a Linux hardware telemetry overlay for fullscreen media setups.
It combines a Go/Fiber backend with a browser HUD that sits on top of YouTube video,
then lets you tune both media framing and overlay layout from a built-in settings UI.

The project is designed for kiosk-style dashboards (Hyprland, ultrawide, and small
secondary displays), but it also works as a local observability panel while gaming,
streaming, or monitoring workloads.

It was inspired by the Lian Li Universal Screen and created to accommodate the small
format displays people mount inside a computer case.

Since those screens do not provide Linux support, I decided to build my own setup
using free and open source tools plus an inexpensive Lesown 1920x480 display from
AliExpress. Similar options from brands like Wisecoco, Waveshare, Eyoyo, UPERFECT,
and other generic HDMI mini displays should also work (for example 5in, 7in, and
13.3in panels at 1920x480, 800x480, 1024x600, and 1920x1080).

https://github.com/user-attachments/assets/6e66fe1e-e2a5-4c1c-9fa3-11435a825713

---

## What It Does

- Samples live CPU, RAM, and GPU telemetry on a fixed interval.
- Serves normalized metrics over REST and WebSocket.
- Renders a live HUD overlay with temperature, utilization, VRAM, and power.
- Plays YouTube video/playlist content under the overlay.
- Stores panel customization as versioned settings in SQLite.
- Applies new settings live (panel auto-reloads when settings are updated).

---

## Hardware + Runtime Notes

Tested on:

- Arch Linux
- Hyprland (Wayland)
- AMD Ryzen CPU
- AMD GPU (`amdgpu`)
- `lm-sensors`

Runtime strategy status:

- Currently only an Arch Linux (Hyprland) strategy is implemented.
- Additional distro/window-manager strategies can be added over time.

Requirements:

- Go 1.25+
- `lm-sensors`
- AMD GPU sysfs paths (for GPU busy/VRAM sensors)

---

## Install and Use

1. Clone/download this repository, then install:

```bash
make install
```

2. Move the binary to a directory in your `PATH` (example):

```bash
mv ./bin/sensorpanel ~/.local/bin/
```

3. Start the server, then open:

- `http://localhost:9070/` to view Sensor Panel
- `http://localhost:9070/settings` to configure layout/media/metrics

4. In Hyprland, add a keybind that launches Chrome in kiosk/fullscreen mode:

```ini
bindd = SUPER CTRL, P, Launch SensorPanel, exec, ~/.config/hypr/scripts/launch-sensor-panel.sh
```

5. Add a window rule so the kiosk window opens on your small monitor workspace.

---

## Hyprland Extras

Example scripts (`~/.config/hypr/scripts`):

`launch-sensor-panel.sh`

```bash
#!/usr/bin/env bash
set -euo pipefail

PANEL_URL="http://homeassistant.local:8123"
PROFILE_DIR="$HOME/.config/chromium-sensor-panel"
VIDEO_ID="AKfsikEXZHM"

hyprctl dispatch exec "chromium \
    --class=chromium-sensor-panel \
    --user-data-dir=$PROFILE_DIR \
    --kiosk \
    --noerrdialogs \
    --disable-extensions \
    --disable-infobars \
    --disable-session-crashed-bubble \
    --disable-translate \
    --disable-features=TranslateUI \
    --disable-sync \
    \"http://localhost:9070\"
"
```

`run-sensor-panel-server.sh`

```bash
#!/usr/bin/env bash
~/.local/bin/sensorpanel
```

Make scripts executable:

```bash
chmod +x ~/.config/hypr/scripts/launch-sensor-panel.sh
chmod +x ~/.config/hypr/scripts/run-sensor-panel-server.sh
```

Hyprland config snippets:

```ini
bindd = SUPER CTRL, P, Launch SensorPanel, exec, ~/.config/hypr/scripts/launch-sensor-panel.sh
exec-once = ~/.config/hypr/scripts/run-sensor-panel-server.sh

# Workspace dedicated to the sensor screen
workspace = 15, monitor:HDMI-A-1

# Move SensorPanel Chromium window to that workspace
windowrule = match:class ^(chromium-sensor-panel)$, workspace 15 silent
```

Adjust workspace number and monitor name to match your setup (`hyprctl monitors`).

---

## Quick Start

1. Install dependencies:

```bash
sudo pacman -S lm_sensors
sudo sensors-detect
go install github.com/air-verse/air@latest
```

2. Create local env file:

```bash
cp .env.example .env
```

3. Run in dev mode (hot reload):

```bash
make dev
```

Or run once:

```bash
make run
```

Database migrations are applied automatically at startup.

---

## App URLs

Assuming default `APP_PORT=9070`:

- Main overlay: `http://localhost:9070/`
  <img width="1890" height="541" alt="screenshot-2026-02-26_19-14-06" src="https://github.com/user-attachments/assets/d2d2304d-e4c0-4cac-befe-e2146b2d78e8" />

- Settings editor: `http://localhost:9070/settings`
  <img width="1664" height="2064" alt="screenshot-2026-02-26_19-08-33" src="https://github.com/user-attachments/assets/25080ddb-f36a-465e-a349-b0ed19b1806f" />

- Telemetry debug page: `http://localhost:9070/telemetry`
  <img width="1721" height="1748" alt="screenshot-2026-02-26_19-10-27" src="https://github.com/user-attachments/assets/1ddff268-cdee-4b48-a4c0-2a3c6f76090c" />

- Metrics JSON: `http://localhost:9070/metrics`
---

## Customization You Can Control

All of these are editable in the settings page and persisted as versioned records.

### Media

- Media source kind: `youtube`, `video`, `playlist`
- YouTube URL or video ID
- Playlist URL support (`list=` parsing)
- Video fit: `cover` or `contain`
- Video alignment (for contain): `left`, `center`, `right`
- Video offsets:
  - `video_offset_x_pct` (`-100` to `100`)
  - `video_offset_y_pct` (`-100` to `100`)
- Infinite playback loop guard (`infinite_video_playback`)

### Overlay + Visuals

- DaisyUI theme (cool, winter, business, nord, lofi, etc.)
- Overlay position: `left`, `right`, `center`, `cover`
- Overlay orientation: `column` or `row`
- Backdrop on/off (`overlay_disable_backdrop`)
- Fine-grained overlay padding (`0` to `500` px per side)

### Metrics Placement

- Metrics scale (`metrics_scale_pct`: `50` to `200`)
- Metrics offset X (`-1000` to `1000` px)
- Metrics offset Y (`-1000` to `1000` px)

### Versioning Behavior

- Each save creates a new settings version.
- The newest saved version becomes `current`.
- Version history is visible in the settings page.
- The running panel receives a settings WebSocket event and reloads.

---

## How To Update Settings

### Option A: Use the UI (recommended)

1. Open `http://localhost:9070/settings`
2. Adjust media/overlay/metrics controls
3. Click **Save version**
4. Panel reloads and applies the new current version
 
<img width="1664" height="2064" alt="screenshot-2026-02-26_19-08-33" src="https://github.com/user-attachments/assets/d8eea306-9c1c-4c99-bad0-beca60f52970" />

<img width="1416" height="1421" alt="screenshot-2026-02-26_19-08-47" src="https://github.com/user-attachments/assets/21029a06-2e93-414a-9b16-31f6891175ea" />


### Option B: Use the API

Create a new version:

```bash
curl -X POST http://localhost:9070/api/settings \
  -H 'Content-Type: application/json' \
  -d '{
    "config": {
      "name": "Main Stream Profile",
      "layout": {
        "name": "left",
        "overlay_layout": "column",
        "theme": "winter",
        "video_fit": "cover",
        "video_align": "center",
        "video_offset_x_pct": 0,
        "video_offset_y_pct": 0,
        "infinite_video_playback": true,
        "metrics_scale_pct": 100,
        "metrics_offset_x": 0,
        "metrics_offset_y": 0
      },
      "media_sources": [
        {
          "kind": "youtube",
          "url": "https://www.youtube.com/watch?v=AKfsikEXZHM",
          "label": ""
        }
      ]
    }
  }'
```

Get current settings:

```bash
curl http://localhost:9070/api/settings/current
```

List recent versions:

```bash
curl http://localhost:9070/api/settings
```

---

## Metrics API

### `GET /metrics`

Returns a normalized snapshot:

```json
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
    "vram_used_gb": 3.2,
    "vram_total_gb": 16,
    "vram_used_pct": 20,
    "power_w": 42,
    "util_pct": 18
  }
}
```

### WebSockets

- `GET /metrics/ws` streams live sensor snapshots.
- `GET /settings/ws` emits settings update events.

---

## Environment Variables

- `DATABASE_URI` SQLite path (default: `data/sensorpanel.db.sqlite3`)
- `APP_ENV` app mode (`development` enables verbose SQL logs)
- `APP_PORT` HTTP port (default in example: `9070`)
- `APP_SHUTDOWN_TIMEOUT` graceful shutdown timeout (default: `10s`)

---

## Security Notes

- Intended for localhost / trusted LAN use.
- Do not expose directly to the public internet without auth and transport security.

---

## Screenshots

Add your latest panel and settings screenshots here.

---

## License

MIT. See `LICENSE`.
