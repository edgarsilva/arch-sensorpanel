let player
let loopTimer

const RESTART_THRESHOLD = 0.5
const DEFAULT_VIDEO_ID = "AKfsikEXZHM"
const WS_URL = `${window.location.protocol === "https:" ? "wss" : "ws"}://${window.location.host}/metrics/ws`

let bootConfig = {
  layout: { name: "left", path: "" },
  media_sources: [],
}

function tempColorForPct(temp, max = 95, min = 35) {
  const clamped = Math.min(Math.max(temp, min), max)
  const hue = 220 - ((clamped - min) / (max - min)) * 220
  return `hsl(${hue}, 85%, 55%)`
}

function normalizeLayout(layoutName) {
  const name = String(layoutName || "left").toLowerCase()
  if (name === "right" || name === "center") {
    return name
  }
  return "left"
}

function applyLayout(layoutName) {
  const panel = document.getElementById("overlay_panel")
  const slot = document.getElementById("overlay_slot")
  if (!panel || !slot) return

  const layout = normalizeLayout(layoutName)
  panel.className = "fixed inset-0 flex items-center z-50 pointer-events-none"

  if (layout === "right") {
    panel.classList.add("justify-end")
    slot.className = "flex flex-col items-start justify-center gap-6 h-full bg-white/35 rounded-2xl px-8 py-2 scale-150 -translate-x-28"
    return
  }

  if (layout === "center") {
    panel.classList.add("justify-center")
    slot.className = "flex flex-col items-start justify-center gap-6 h-full bg-white/35 rounded-2xl px-8 py-2 scale-150"
    return
  }

  panel.classList.add("justify-start")
  slot.className = "flex flex-col items-start justify-center gap-6 h-full bg-white/35 rounded-2xl px-8 py-2 scale-150 translate-x-28"
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

function resolveVideoId() {
  const first = Array.isArray(bootConfig.media_sources) ? bootConfig.media_sources[0] : null
  if (!first) return DEFAULT_VIDEO_ID

  const trimmedURL = String(first.url || "").trim()
  if (!trimmedURL) return DEFAULT_VIDEO_ID

  const parsed = extractVideoIdFromURL(trimmedURL)
  if (parsed) return parsed

  if (/^[A-Za-z0-9_-]{11}$/.test(trimmedURL)) {
    return trimmedURL
  }

  return DEFAULT_VIDEO_ID
}

async function bootstrapSettings() {
  try {
    const res = await fetch("/settings/current", {
      headers: { Accept: "application/json" },
    })
    if (!res.ok) return
    const payload = await res.json()
    if (!payload || !payload.config) return
    bootConfig = payload.config
  } catch (_) {
  }
}

function onYouTubeIframeAPIReady() {
  player = new YT.Player("player", {
    videoId: resolveVideoId(),
    playerVars: {
      autoplay: 1,
      mute: 1,
      controls: 1,
      rel: 0,
      playsinline: 1,
      origin: window.location.origin,
    },
    events: {
      onReady: onPlayerReady,
      onStateChange: onPlayerStateChange,
    },
  })
}

function onPlayerReady(e) {
  e.target.playVideo()
  startLoopGuard()
}

function onPlayerStateChange(e) {
  if (e.data === YT.PlayerState.PLAYING) {
    startLoopGuard()
  }

  if (e.data === YT.PlayerState.ENDED) {
    restart()
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

bootstrapSettings().finally(() => {
  applyLayout(bootConfig.layout && bootConfig.layout.name)
  connectMetricsSocket()
})
