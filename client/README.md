# sensorpanel-client (spike)

A native replacement for the kiosk Chromium window on the sensor strip (HDMI-A-1, 1920×480).
It is a single process. libmpv renders the video into the window's OpenGL framebuffer. Slint then draws the overlay on top, with a transparent background.

```
Go server (unchanged) ──/api/settings/current──▶ player.rs ──▶ libmpv (+ yt-dlp, VA-API)
          ├──/metrics/ws──▶ telemetry.rs ──▶ Slint overlay (ui/panel.slint)
          └──/settings/ws─▶ re-fetch + apply live (layout/theme/playlist, no restart)
frame: Slint BeforeRendering → mpv render(fbo 0) → Slint draws overlay → swap
```

## Run

```bash
make client-run                         # windowed dev run
SENSORPANEL_URL=http://host:9070 sensorpanel-client   # fullscreen (default)
SENSORPANEL_MPV_LOG=/tmp/mpv.log sensorpanel-client   # verbose mpv log
```

**Requirements**
- `mpv` (for libmpv) and `yt-dlp` on `PATH`.
- Rust 1.92 or newer. `rust-toolchain.toml` pins 1.97.1, because Slint 1.18 needs 1.92+.
- The Wayland app id is `sensorpanel-client`.

## Controls

These work only when the panel has focus, e.g. when the pointer is over it:

| key | action |
|---|---|
| F5 | re-fetch settings and reload the current video |
| ← / → | previous / next video (same as the on-screen buttons, wrapping at both ends) |

To fully restart the client, use the Hyprland launcher (`SUPER+ALT+P`).

## What maps to what

| Web panel (`public/js/panel.js`) | Native client |
|---|---|
| YouTube IFrame, muted autoplay | libmpv + yt-dlp, `aid=no`, VP9/AV1 ≤1080p, `hwdec=vaapi` |
| `buildPlaylistEntries` / `resolveMediaMode` | `player::build_entries` (same dedupe, starts at `media_sources[0]`) |
| loop guard (`seekTo(0.25)` near end) | `loop-file=inf` (gapless) |
| auto-advance + wrap | mpv playlist + `loop-playlist=inf` |
| `persistCurrentPlaylistVideo` | PATCH `media_url` (`broadcast:false`) on `file-loaded` |
| watchdog page reloads | mpv skips failed entries. If everything fails and mpv goes idle, the playlist is reloaded with backoff (5 s doubling up to 120 s). |
| `video_fit` cover/contain, align, offsets | `panscan`, `video-align-x/y` |
| DaisyUI overlay, themes, layout, scale, padding | `ui/panel.slint` + `theme.rs` (a subset of the DaisyUI themes; unknown themes fall back to lofi) |
| settings WS → page reload | settings WS → re-fetch and apply in place. The playlist is only reloaded if its set of videos changed. |

## Spike results (2026-09-28)

Test setup:
- RX 9070 XT, Mesa 26.2.
- mpv 0.41, playing the same 1080p VP9 stream on the same panel.
- Measured over a 60 s sample, with each client in turn visible on HDMI-A-1.

| | processes | CPU (% of one core) | PSS | RSS (summed) |
|---|---|---|---|---|
| Chromium kiosk | 14 | 13.1% | 749 MB | 1571 MB |
| sensorpanel-client | 1 | 4.3% | 193 MB | 269 MB |

- **Decoding.** mpv reports `Using hardware decoding (vaapi)` and `VO: [libmpv] 1920x1080 vaapi[nv12]`. Frames stay on the GPU with no copy back to system memory.
  - This requires handing Slint's `wl_display` to mpv (`RenderParam::WaylandDisplay`).
  - Without it, mpv falls back to `vulkan-copy`.
- **Memory.** Most of the client's memory was mpv's demuxer cache, plus Mesa/GL (see tuning below).
- **GPU busy %.** `gpu_busy_percent` (~10–11% in both runs) covers the whole card, including two 4K desktops, so it doesn't separate the two clients.

### Tuning after the spike (40–60 s samples, same stream)

| change | CPU | PSS |
|---|---|---|
| baseline (numbers above) | 3.9% | 185 MB |
| `cache-rendering-hint` on the overlay slot: rasterised once, reused across video frames | 3.5% | 179 MB |
| demuxer cache 64 MiB → 32 MiB, back-buffer 16 MiB → 0 | ~3.5–3.9% | **141 MB** |

Where the CPU goes, per thread:
- About 2% is Slint's main thread: overlay drawing plus the mpv render call.
- mpv's decode, demux and output threads together use under 1%.

Also tested and left as is:
- **Disabling the "live" ping animation** (which redraws at the monitor's refresh rate): no measurable change, so it stays.

## Focus

The panel must never take focus from the main monitors. Two things make sure of that:
- winit is told `with_active(false)`, so the client doesn't send an activation request. Hyprland's `misc:focus_on_activate` would honour one.
- The Hyprland rule uses `workspace = "15 silent"`.

Don't add `no_initial_focus` to the rule: Hyprland then drops the fullscreen state and tiles the window.

## Not verified yet / next

- Long-running soak test: many hours of `loop-file=inf` on a stream, stream URL expiry, and network drops.
- Live settings edits and the prev/next buttons were implemented but not exercised during the spike.
- **Video cache.** A "Cache videos" setting where the server runs yt-dlp into `data/videos/`, and both clients play local files first, falling back to streaming.
