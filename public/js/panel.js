let player
let loopTimer
let mediaMode = { kind: "youtube", videoId: "", playlistId: "" }

const RESTART_THRESHOLD = 0.5
const DEFAULT_VIDEO_ID = "AKfsikEXZHM"
const WS_URL = `${window.location.protocol === "https:" ? "wss" : "ws"}://${window.location.host}/metrics/ws`
const SETTINGS_WS_URL = `${window.location.protocol === "https:" ? "wss" : "ws"}://${window.location.host}/settings/ws`
const SETTINGS_RELOAD_DELAY_MS = 350

let bootConfig = {
  layout: {
    name: "left",
    overlay_layout: "column",
    theme: "lofi",
    video_fit: "cover",
    video_align: "center",
    video_offset_x_pct: 0,
    video_offset_y_pct: 0,
    infinite_video_playback: false,
    overlay_padding_top: 0,
    overlay_padding_right: 0,
    overlay_padding_bottom: 0,
    overlay_padding_left: 0,
    metrics_scale_pct: 100,
    metrics_offset_x: 0,
    metrics_offset_y: 0,
  },
  media_sources: [],
}
let bootSettingsVersion = 0
let settingsReloadScheduled = false

function clamp(value, min, max) {
  return Math.min(Math.max(value, min), max)
}

function normalizeTheme(theme) {
  const value = String(theme || "lofi").toLowerCase()
  const supported = new Set(["cool", "winter", "corporate", "nord", "aqua", "lofi", "business", "dark", "dim"])
  return supported.has(value) ? value : "lofi"
}

function applyTheme(theme) {
  document.documentElement.setAttribute("data-theme", normalizeTheme(theme))
}

function normalizeVideoFit(value) {
  const fit = String(value || "cover").toLowerCase()
  if (fit === "contain") return "contain"
  return "cover"
}

function normalizeVideoAlign(value) {
  const align = String(value || "center").toLowerCase()
  if (align === "left" || align === "right") return align
  return "center"
}

function applyVideoLayout(layoutConfig) {
  const wrap = document.querySelector(".video-cover")
  if (!wrap) return

  const fit = normalizeVideoFit(layoutConfig && layoutConfig.video_fit)
  const align = normalizeVideoAlign(layoutConfig && layoutConfig.video_align)

  wrap.classList.remove("video-fit-cover", "video-fit-contain", "video-align-left", "video-align-center", "video-align-right")
  wrap.classList.add(fit === "contain" ? "video-fit-contain" : "video-fit-cover")
  wrap.classList.add(`video-align-${align}`)
}

function applyVideoOffset(layoutConfig) {
  const wrap = document.querySelector(".video-cover")
  const playerEl = document.getElementById("player")
  if (!wrap || !playerEl) return

  const fit = normalizeVideoFit(layoutConfig && layoutConfig.video_fit)
  const offsetXPct = clamp(Number(layoutConfig && layoutConfig.video_offset_x_pct) || 0, -100, 100)
  const offsetYPct = clamp(Number(layoutConfig && layoutConfig.video_offset_y_pct) || 0, -100, 100)

  const vw = window.innerWidth || 0
  const vh = window.innerHeight || 0
  const videoWFromH = vh * (16 / 9)
  const videoHFromW = vw * (9 / 16)

  if (fit === "cover") {
    const displayedW = Math.max(vw, videoWFromH)
    const displayedH = Math.max(vh, videoHFromW)
    const availX = Math.max(0, (displayedW - vw) / 2)
    const availY = Math.max(0, (displayedH - vh) / 2)
    const shiftX = availX * (offsetXPct / 100)
    const shiftY = availY * (offsetYPct / 100)
    playerEl.style.setProperty("--video-offset-x", `${shiftX}px`)
    playerEl.style.setProperty("--video-offset-y", `${shiftY}px`)
    return
  }

  const displayedW = Math.min(vw, videoWFromH)
  const displayedH = Math.min(vh, videoHFromW)
  const availX = Math.max(0, (vw - displayedW) / 2)
  const availY = Math.max(0, (vh - displayedH) / 2)
  const shiftX = availX * (offsetXPct / 100)
  const shiftY = availY * (offsetYPct / 100)
  playerEl.style.setProperty("--video-offset-x", `${shiftX}px`)
  playerEl.style.setProperty("--video-offset-y", `${shiftY}px`)
}

function tempColorForPct(temp, max = 95, min = 35) {
  const clamped = Math.min(Math.max(temp, min), max)
  const hue = 220 - ((clamped - min) / (max - min)) * 220
  return `hsl(${hue}, 85%, 55%)`
}

function normalizeOverlayPosition(layoutName) {
  const name = String(layoutName || "left").toLowerCase()
  if (name === "right" || name === "center" || name === "cover") {
    return name
  }
  return "left"
}

function normalizeOverlayLayout(layoutValue) {
  const value = String(layoutValue || "column").toLowerCase()
  if (value === "row") {
    return "row"
  }
  return "column"
}

function applyLayout(layoutConfig) {
  const panel = document.getElementById("overlay_panel")
  const slot = document.getElementById("overlay_slot")
  if (!panel || !slot) return

  const layout = normalizeOverlayPosition(layoutConfig && layoutConfig.name)
  const overlayLayout = normalizeOverlayLayout(layoutConfig && layoutConfig.overlay_layout)
  panel.className = "fixed inset-0 flex items-center z-50 pointer-events-none"

  if (overlayLayout === "row") {
    slot.className = "flex flex-row items-center justify-center gap-6 h-full bg-white/35 rounded-2xl px-8 py-2"
  } else {
    slot.className = "flex flex-col items-start justify-center gap-6 h-full bg-white/35 rounded-2xl px-8 py-2"
  }

  if (layout === "cover") {
    panel.classList.add("justify-center")
    slot.className = `flex ${overlayLayout === "row" ? "flex-row items-center" : "flex-col items-start"} justify-center gap-6 h-full w-full bg-white/25 px-10 py-6`
    return
  }

  if (layout === "right") {
    panel.classList.add("justify-end")
    return
  }

  if (layout === "center") {
    panel.classList.add("justify-center")
    return
  }

  panel.classList.add("justify-start")
}

function applyMetricsTuning(layoutConfig) {
  const slot = document.getElementById("overlay_slot")
  if (!slot) return

  const scalePct = clamp(Number(layoutConfig && layoutConfig.metrics_scale_pct) || 100, 50, 200)
  const offsetX = clamp(Number(layoutConfig && layoutConfig.metrics_offset_x) || 0, -250, 250)
  const offsetY = clamp(Number(layoutConfig && layoutConfig.metrics_offset_y) || 0, -250, 250)

  slot.style.transform = `translate(${offsetX}px, ${offsetY}px) scale(${scalePct / 100})`
  slot.style.transformOrigin = "center"
}

function applyOverlayPadding(layoutConfig) {
  const slot = document.getElementById("overlay_slot")
  if (!slot) return

  const layout = normalizeOverlayPosition(layoutConfig && layoutConfig.name)
  const extraTop = clamp(Number(layoutConfig && layoutConfig.overlay_padding_top) || 0, 0, 100)
  const extraRight = clamp(Number(layoutConfig && layoutConfig.overlay_padding_right) || 0, 0, 100)
  const extraBottom = clamp(Number(layoutConfig && layoutConfig.overlay_padding_bottom) || 0, 0, 100)
  const extraLeft = clamp(Number(layoutConfig && layoutConfig.overlay_padding_left) || 0, 0, 100)

  const base = layout === "cover"
    ? { top: 24, right: 40, bottom: 24, left: 40 }
    : { top: 8, right: 32, bottom: 8, left: 32 }

  slot.style.paddingTop = `${base.top + extraTop}px`
  slot.style.paddingRight = `${base.right + extraRight}px`
  slot.style.paddingBottom = `${base.bottom + extraBottom}px`
  slot.style.paddingLeft = `${base.left + extraLeft}px`
}

function extractVideoIdFromURL(rawURL) {
  if (!rawURL) return ""
  try {
    const url = new URL(rawURL)
    if (url.hostname.includes("youtu.be")) {
      return url.pathname.replace("/", "")
    }
    if (url.hostname.includes("youtube.com")) {
      return url.searchParams.get("v") || ""
    }
  } catch (_) {
    return ""
  }
  return ""
}

function extractPlaylistIdFromURL(rawURL) {
  if (!rawURL) return ""
  try {
    const url = new URL(rawURL)
    return url.searchParams.get("list") || ""
  } catch (_) {
    return ""
  }
}

function resolveMediaMode() {
  const first = Array.isArray(bootConfig.media_sources) ? bootConfig.media_sources[0] : null
  const fallback = { kind: "youtube", videoId: DEFAULT_VIDEO_ID, playlistId: "" }
  if (!first) return fallback

  const kind = String(first.kind || "youtube").toLowerCase()
  const trimmedURL = String(first.url || "").trim()
  if (!trimmedURL) return fallback

  const videoId = (() => {
    const parsed = extractVideoIdFromURL(trimmedURL)
    if (parsed) return parsed
    if (/^[A-Za-z0-9_-]{11}$/.test(trimmedURL)) return trimmedURL
    return DEFAULT_VIDEO_ID
  })()

  if (kind === "playlist") {
    const playlistId = extractPlaylistIdFromURL(trimmedURL) || trimmedURL
    if (playlistId) {
      return { kind: "playlist", videoId, playlistId }
    }
  }

  return { kind: "youtube", videoId, playlistId: "" }
}

function setPlaylistControlsVisible(visible) {
  const controls = document.getElementById("playlist_controls")
  if (!controls) return
  controls.classList.toggle("hidden", !visible)
}

function bindPlaylistControls() {
  const prevBtn = document.getElementById("playlist_prev")
  const nextBtn = document.getElementById("playlist_next")
  if (!prevBtn || !nextBtn) return

  prevBtn.addEventListener("click", () => {
    if (player && typeof player.previousVideo === "function") {
      player.previousVideo()
    }
  })

  nextBtn.addEventListener("click", () => {
    if (player && typeof player.nextVideo === "function") {
      player.nextVideo()
    }
  })
}

function resolveVideoId() {
  return mediaMode.videoId || DEFAULT_VIDEO_ID
}

function isInfiniteVideoPlaybackEnabled() {
  return !!(bootConfig.layout && bootConfig.layout.infinite_video_playback)
}

async function bootstrapSettings() {
  try {
    const res = await fetch("/api/settings/current", {
      headers: { Accept: "application/json" },
    })
    if (!res.ok) return
    const payload = await res.json()
    if (!payload || !payload.config) return
    bootConfig = payload.config
    bootSettingsVersion = Number(payload.version) || 0
  } catch (_) {
  }
}

function scheduleSettingsReload() {
  if (settingsReloadScheduled) return
  settingsReloadScheduled = true
  setTimeout(() => {
    window.location.reload()
  }, SETTINGS_RELOAD_DELAY_MS)
}

function connectSettingsSocket() {
  const ws = new WebSocket(SETTINGS_WS_URL)

  ws.onmessage = (event) => {
    try {
      const payload = JSON.parse(event.data)
      if (!payload || payload.type !== "settings.updated") return

      const incomingVersion = Number(payload.version) || 0
      if (incomingVersion === 0 || incomingVersion > bootSettingsVersion) {
        scheduleSettingsReload()
      }
    } catch (_) {
    }
  }

  ws.onerror = () => {
    ws.close()
  }

  ws.onclose = () => {
    if (settingsReloadScheduled) return
    setTimeout(connectSettingsSocket, 2000)
  }
}

function onYouTubeIframeAPIReady() {
  const playerVars = {
    autoplay: 1,
    mute: 1,
    controls: 1,
    rel: 0,
    playsinline: 1,
    origin: window.location.origin,
  }

  if (mediaMode.kind === "playlist" && mediaMode.playlistId) {
    playerVars.listType = "playlist"
    playerVars.list = mediaMode.playlistId
  }

  player = new YT.Player("player", {
    videoId: resolveVideoId(),
    playerVars,
    events: {
      onReady: onPlayerReady,
      onStateChange: onPlayerStateChange,
    },
  })
}

function onPlayerReady(e) {
  e.target.playVideo()
  applyVideoOffset(bootConfig.layout)
  if (isInfiniteVideoPlaybackEnabled()) {
    startLoopGuard()
  }
}

function onPlayerStateChange(e) {
  if (e.data === YT.PlayerState.PLAYING) {
    if (isInfiniteVideoPlaybackEnabled()) {
      startLoopGuard()
    } else {
      stopLoopGuard()
    }
  }

  if (e.data === YT.PlayerState.ENDED) {
    if (isInfiniteVideoPlaybackEnabled()) {
      restart()
    }
  }
}

function startLoopGuard() {
  stopLoopGuard()
  loopTimer = setInterval(() => {
    const duration = player.getDuration()
    const current = player.getCurrentTime()

    if (duration > 0 && duration - current <= RESTART_THRESHOLD) {
      restart()
    }
  }, 100)
}

function stopLoopGuard() {
  if (loopTimer) {
    clearInterval(loopTimer)
    loopTimer = null
  }
}

function restart() {
  stopLoopGuard()
  player.seekTo(0.25, true)
  player.playVideo()
}

function updateUI(data) {
  if (!data) return

  const cpuTemp = Math.round(data.cpu.temp_c)
  const cpuUtil = Math.round(data.cpu.util_pct)
  const cpuPower = Math.round(data.cpu.power_w)
  document.getElementById("cpu_temp").textContent = cpuTemp
  document.getElementById("cpu_power").textContent = `CPU (${cpuPower}W)`

  const cpuTempProgress = document.getElementById("cpu_temp_progress")
  if (cpuTempProgress) {
    cpuTempProgress.value = cpuTemp
    cpuTempProgress.style.setProperty("--cpu-temp-color", tempColorForPct(cpuTemp))
  }

  const cpuRadial = document.getElementById("cpu_util")
  cpuRadial.style.setProperty("--value", cpuUtil)
  document.getElementById("cpu_util_text").textContent = `${cpuUtil}%`

  const gpuHotspot = Math.round(data.gpu.hotspot_c)
  const gpuUtil = Math.round(data.gpu.util_pct)
  const gpuVramTemp = Math.round(data.gpu.vram_c)
  const gpuVramTotal = Math.round(data.gpu.vram_total_gb * 10) / 10
  const gpuVramUsed = Math.round(data.gpu.vram_used_gb * 10) / 10
  const gpuVramUsedPct = Math.round(data.gpu.vram_used_pct)
  const gpuPower = Math.round(data.gpu.power_w)

  document.getElementById("gpu_hotspot").textContent = gpuHotspot
  document.getElementById("vram_temp").textContent = `VRAM ${gpuVramTemp}°C`
  document.getElementById("gpu_desc").textContent = `VRAM ${gpuVramUsed}/${gpuVramTotal}GB (${gpuVramUsedPct}%)`
  document.getElementById("gpu_progress").value = gpuVramUsedPct
  document.getElementById("gpu_power").textContent = `GPU (${gpuPower}W)`

  const gpuTempProgress = document.getElementById("gpu_temp_progress")
  if (gpuTempProgress) {
    gpuTempProgress.value = gpuHotspot
    gpuTempProgress.style.setProperty("--gpu-temp-color", tempColorForPct(gpuHotspot, 110, 45))
  }

  const radial = document.getElementById("gpu_util")
  radial.style.setProperty("--value", gpuUtil)
  document.getElementById("gpu_util_text").textContent = `${gpuUtil}%`

  const ramTotal = Math.round(data.ram.total_gb * 10) / 10
  const ramUsed = Math.round(data.ram.used_gb * 10) / 10
  const ramUsedPct = Math.round(data.ram.used_pct * 10) / 10
  document.getElementById("ram_progress").value = ramUsedPct
  document.getElementById("ram_desc").textContent = `RAM ${ramUsed}/${ramTotal}gb (${ramUsedPct}%)`
}

function setConnectionState(state) {
  const dot = document.getElementById("conn_dot")
  const ping = document.getElementById("conn_ping")
  const text = document.getElementById("conn_text")

  if (!dot || !ping || !text) return

  if (state === "live") {
    dot.className = "relative inline-flex size-3 rounded-full bg-success"
    ping.className = "absolute inline-flex h-full w-full rounded-full bg-success opacity-60 animate-ping"
    text.textContent = "live"
    return
  }

  if (state === "reconnecting") {
    dot.className = "relative inline-flex size-3 rounded-full bg-warning"
    ping.className = "absolute inline-flex h-full w-full rounded-full bg-warning opacity-50 animate-ping"
    text.textContent = "reconnecting"
    return
  }

  dot.className = "relative inline-flex size-3 rounded-full bg-neutral"
  ping.className = "absolute inline-flex h-full w-full rounded-full bg-neutral opacity-40"
  text.textContent = "connecting"
}

function connectMetricsSocket() {
  setConnectionState("connecting")
  const ws = new WebSocket(WS_URL)

  ws.onopen = () => {
    setConnectionState("live")
  }

  ws.onmessage = (event) => {
    try {
      updateUI(JSON.parse(event.data))
    } catch (err) {
      console.warn("invalid ws payload", err)
    }
  }

  ws.onerror = () => {
    setConnectionState("reconnecting")
    ws.close()
  }

  ws.onclose = () => {
    setConnectionState("reconnecting")
    setTimeout(connectMetricsSocket, 2000)
  }
}

window.onYouTubeIframeAPIReady = onYouTubeIframeAPIReady

window.addEventListener("resize", () => {
  applyVideoOffset(bootConfig.layout)
})

bootstrapSettings().finally(() => {
  mediaMode = resolveMediaMode()
  applyTheme(bootConfig.layout && bootConfig.layout.theme)
  applyVideoLayout(bootConfig.layout)
  applyVideoOffset(bootConfig.layout)
  applyLayout(bootConfig.layout)
  applyOverlayPadding(bootConfig.layout)
  applyMetricsTuning(bootConfig.layout)
  setPlaylistControlsVisible(mediaMode.kind === "playlist")
  bindPlaylistControls()
  connectSettingsSocket()
  connectMetricsSocket()
})
