let player
let loopTimer
let watchdogTimer
let recoveryGraceTimer
let mediaMode = { kind: "youtube", videoId: "", playlistId: "" }

const LOOP_GUARD_INTERVAL_MS = 400
const RESTART_THRESHOLD = 0.800
const DEFAULT_VIDEO_ID = "AKfsikEXZHM"
const WS_URL = `${window.location.protocol === "https:" ? "wss" : "ws"}://${window.location.host}/metrics/ws`
const SETTINGS_WS_URL = `${window.location.protocol === "https:" ? "wss" : "ws"}://${window.location.host}/settings/ws`
const SETTINGS_RELOAD_DELAY_MS = 350
const PLAYER_RECOVERY_RELOAD_DELAY_MS = 1500
const PLAYER_RECOVERY_MIN_INTERVAL_MS = 30000
const PLAYER_RECOVERY_LAST_RELOAD_KEY = "sensorpanel.playerRecoveryLastReloadAt"
const PLAYER_WATCHDOG_INTERVAL_MS = 3000
const PLAYER_STALL_THRESHOLD_MS = 30000
const PLAYER_STATE_SILENCE_THRESHOLD_MS = 45000
const PLAYER_API_ERROR_THRESHOLD = 3
const PLAYER_RECOVERY_RECREATE_GRACE_MS = 15000

let playerRecoveryInProgress = false
let playerReady = false
let expectedPlayerToBePlaying = false
let lastPlayerProgressAt = 0
let lastPlayerStateAt = 0
let lastObservedPlayerTime = 0
let consecutivePlayerAPIErrors = 0
let playerRecreateAttempts = 0

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
		overlay_disable_backdrop: false,
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
let lastObservedPlaylistVideoId = ""

function clamp(value, min, max) {
	return Math.min(Math.max(value, min), max)
}

function normalizeTheme(theme) {
	const value = String(theme || "lofi").toLowerCase()
	const supported = new Set([
		"cool",
		"light",
		"dark",
		"cupcake",
		"bumblebee",
		"emerald",
		"corporate",
		"synthwave",
		"retro",
		"cyberpunk",
		"valentine",
		"halloween",
		"garden",
		"forest",
		"aqua",
		"lofi",
		"pastel",
		"fantasy",
		"wireframe",
		"black",
		"luxury",
		"dracula",
		"cmyk",
		"autumn",
		"business",
		"acid",
		"lemonade",
		"night",
		"coffee",
		"winter",
		"dim",
		"nord",
		"sunset",
		"caramellatte",
		"abyss",
		"silk",
	])
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

function resizeYouTubePlayer() {
	const playerEl = document.getElementById("player")
	if (!player || !playerEl || typeof player.setSize !== "function") return

	const rect = playerEl.getBoundingClientRect()
	const width = Math.max(1, Math.round(rect.width))
	const height = Math.max(1, Math.round(rect.height))
	player.setSize(width, height)
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
	const hasBackdrop = !(layoutConfig && layoutConfig.overlay_disable_backdrop)
	panel.className = "fixed inset-0 flex items-center z-50 pointer-events-none"

	if (overlayLayout === "row") {
		slot.className = `flex flex-row items-center justify-center gap-6 h-full rounded-2xl px-8 py-2 ${hasBackdrop ? "bg-white/35" : "bg-transparent"}`
	} else {
		slot.className = `flex flex-col items-start justify-center gap-6 h-full rounded-2xl px-8 py-2 ${hasBackdrop ? "bg-white/35" : "bg-transparent"}`
	}

	if (layout === "cover") {
		panel.classList.add("justify-center")
		slot.className = `flex ${overlayLayout === "row" ? "flex-row items-center" : "flex-col items-start"} justify-center gap-6 h-full w-full px-10 py-6 ${hasBackdrop ? "bg-white/25" : "bg-transparent"}`
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
	const offsetX = clamp(Number(layoutConfig && layoutConfig.metrics_offset_x) || 0, -1000, 1000)
	const offsetY = clamp(Number(layoutConfig && layoutConfig.metrics_offset_y) || 0, -1000, 1000)

	slot.style.transform = `translate(${offsetX}px, ${offsetY}px) scale(${scalePct / 100})`
	slot.style.transformOrigin = "center"
}

function applyOverlayPadding(layoutConfig) {
	const slot = document.getElementById("overlay_slot")
	if (!slot) return

	const layout = normalizeOverlayPosition(layoutConfig && layoutConfig.name)
	const extraTop = clamp(Number(layoutConfig && layoutConfig.overlay_padding_top) || 0, 0, 500)
	const extraRight = clamp(Number(layoutConfig && layoutConfig.overlay_padding_right) || 0, 0, 500)
	const extraBottom = clamp(Number(layoutConfig && layoutConfig.overlay_padding_bottom) || 0, 0, 500)
	const extraLeft = clamp(Number(layoutConfig && layoutConfig.overlay_padding_left) || 0, 0, 500)

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

function currentMediaURLFromConfig() {
	const first = Array.isArray(bootConfig.media_sources) ? bootConfig.media_sources[0] : null
	return String((first && first.url) || "").trim()
}

function buildPlaylistMediaURL(videoId, playlistId) {
	const v = String(videoId || "").trim()
	const list = String(playlistId || "").trim()
	if (!v || !list) return ""
	return `https://www.youtube.com/watch?v=${encodeURIComponent(v)}&list=${encodeURIComponent(list)}`
}

function videoIdFromURL(rawURL) {
	if (!rawURL) return ""
	try {
		const url = new URL(rawURL)
		return String(url.searchParams.get("v") || "").trim()
	} catch (_) {
		return ""
	}
}

function getCurrentPlayerVideoId() {
	if (!player) return ""

	if (typeof player.getVideoData === "function") {
		const videoData = player.getVideoData() || {}
		const fromData = String(videoData.video_id || "").trim()
		if (fromData) return fromData
	}

	if (typeof player.getVideoUrl === "function") {
		const fromURL = videoIdFromURL(player.getVideoUrl())
		if (fromURL) return fromURL
	}

	return ""
}

async function persistCurrentPlaylistVideo() {
	if (!player || mediaMode.kind !== "playlist" || !mediaMode.playlistId) return

	const currentVideoId = getCurrentPlayerVideoId()
	if (!currentVideoId) return
	if (currentVideoId === lastObservedPlaylistVideoId) return
	lastObservedPlaylistVideoId = currentVideoId

	const nextURL = buildPlaylistMediaURL(currentVideoId, mediaMode.playlistId)
	if (!nextURL) return

	if (nextURL === currentMediaURLFromConfig()) return

	try {
		const res = await fetch("/api/settings/current/field", {
			method: "PATCH",
			headers: { "Content-Type": "application/json", Accept: "application/json" },
			body: JSON.stringify({ field: "media_url", value: nextURL, broadcast: false }),
		})
		if (!res.ok) {
			throw new Error(`playlist progress save failed: ${res.status}`)
		}
	} catch (err) {
		console.warn("failed to persist current playlist video", err)
	}
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

function schedulePlayerRecoveryReload(reason) {
	if (settingsReloadScheduled) return

	const now = Date.now()
	const lastReloadRaw = window.sessionStorage.getItem(PLAYER_RECOVERY_LAST_RELOAD_KEY)
	const lastReloadAt = Number(lastReloadRaw) || 0
	if (lastReloadAt > 0 && now - lastReloadAt < PLAYER_RECOVERY_MIN_INTERVAL_MS) {
		console.warn("player recovery reload suppressed due to cooldown", { reason })
		return
	}

	settingsReloadScheduled = true
	console.warn("player failure detected, reloading page", { reason })
	window.sessionStorage.setItem(PLAYER_RECOVERY_LAST_RELOAD_KEY, String(now))

	setTimeout(() => {
		window.location.reload()
	}, PLAYER_RECOVERY_RELOAD_DELAY_MS)
}

function isExpectedYouTubeSrc(src) {
	if (!src) return false
	try {
		const url = new URL(src)
		return url.hostname === "www.youtube.com" || url.hostname === "youtube.com" || url.hostname === "www.youtube-nocookie.com" || url.hostname === "youtube-nocookie.com"
	} catch (_) {
		return false
	}
}

function safePlayerCall(methodName) {
	if (!player || typeof player[methodName] !== "function") return { ok: false, missing: true }
	try {
		const value = player[methodName]()
		consecutivePlayerAPIErrors = 0
		return { ok: true, value }
	} catch (err) {
		consecutivePlayerAPIErrors += 1
		console.warn("player api exception", { methodName, consecutivePlayerAPIErrors, err })
		if (consecutivePlayerAPIErrors >= PLAYER_API_ERROR_THRESHOLD) {
			attemptPlayerRecovery(`api_exception_${methodName}`)
		}
		return { ok: false, error: err }
	}
}

function stopPlayerWatchdog() {
	if (watchdogTimer) {
		clearInterval(watchdogTimer)
		watchdogTimer = null
	}
}

function markPlayerRecovered(reason) {
	if (!playerRecoveryInProgress) return
	playerRecoveryInProgress = false
	playerRecreateAttempts = 0
	if (recoveryGraceTimer) {
		clearTimeout(recoveryGraceTimer)
		recoveryGraceTimer = null
	}
	console.info("player recovery succeeded", { reason })
}

function startPlayerWatchdog() {
	stopPlayerWatchdog()
	watchdogTimer = setInterval(() => {
		if (!player || !playerReady) return

		const now = Date.now()

		if (lastPlayerStateAt > 0 && now - lastPlayerStateAt > PLAYER_STATE_SILENCE_THRESHOLD_MS) {
			attemptPlayerRecovery("no_state_events")
			return
		}

		const iframeResult = safePlayerCall("getIframe")
		if (iframeResult.ok) {
			const iframe = iframeResult.value
			const iframeSrc = iframe && typeof iframe.src === "string" ? iframe.src : ""
			if (!isExpectedYouTubeSrc(iframeSrc)) {
				console.warn("player watchdog detected unexpected iframe src", { iframeSrc })
				attemptPlayerRecovery("iframe_src_invalid")
				return
			}
		}

		const stateResult = safePlayerCall("getPlayerState")
		if (stateResult.ok && typeof stateResult.value === "number") {
			const state = stateResult.value
			if (window.YT && YT.PlayerState) {
				expectedPlayerToBePlaying = state === YT.PlayerState.PLAYING || state === YT.PlayerState.BUFFERING || state === YT.PlayerState.CUED
			}
		}

		const timeResult = safePlayerCall("getCurrentTime")
		if (timeResult.ok) {
			const currentTime = Number(timeResult.value) || 0
			if (currentTime > lastObservedPlayerTime + 0.05) {
				lastObservedPlayerTime = currentTime
				lastPlayerProgressAt = now
				markPlayerRecovered("time_progress")
			}
		}

		if (expectedPlayerToBePlaying && lastPlayerProgressAt > 0 && now - lastPlayerProgressAt > PLAYER_STALL_THRESHOLD_MS) {
			attemptPlayerRecovery("stalled_time")
		}
	}, PLAYER_WATCHDOG_INTERVAL_MS)
}

function destroyPlayerInstance() {
	if (!player) return
	try {
		if (typeof player.destroy === "function") {
			player.destroy()
		}
	} catch (err) {
		console.warn("failed to destroy player instance", { err })
	}
	player = null
	playerReady = false
}

function rebuildPlayerMountNode() {
	const currentNode = document.getElementById("player")
	if (!currentNode || !currentNode.parentNode) return false
	const nextNode = document.createElement("div")
	nextNode.id = "player"
	currentNode.parentNode.replaceChild(nextNode, currentNode)
	return true
}

function createYouTubePlayer() {
	if (!window.YT || typeof YT.Player !== "function") {
		console.warn("youtube iframe api not ready during player creation")
		return false
	}

	const playerVars = {
		autoplay: 1,
		mute: 1,
		controls: 0,
		disablekb: 1,
		fs: 0,
		iv_load_policy: 3,
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
			onError: onPlayerError,
		},
	})

	return true
}

function attemptPlayerRecovery(reason) {
	if (settingsReloadScheduled) return
	if (playerRecoveryInProgress) {
		console.warn("player recovery already in progress", { reason })
		return
	}

	playerRecoveryInProgress = true
	playerRecreateAttempts += 1
	console.warn("player watchdog recovery starting", {
		reason,
		attempt: playerRecreateAttempts,
		lastPlayerProgressAt,
		lastPlayerStateAt,
		lastObservedPlayerTime,
		consecutivePlayerAPIErrors,
	})

	stopLoopGuard()
	stopPlayerWatchdog()
	expectedPlayerToBePlaying = false

	destroyPlayerInstance()
	if (!rebuildPlayerMountNode()) {
		schedulePlayerRecoveryReload(`rebuild_mount_failed_${reason}`)
		return
	}

	if (!createYouTubePlayer()) {
		schedulePlayerRecoveryReload(`recreate_player_failed_${reason}`)
		return
	}

	if (recoveryGraceTimer) {
		clearTimeout(recoveryGraceTimer)
	}
	recoveryGraceTimer = setTimeout(() => {
		console.warn("player watchdog recovery timed out; escalating to reload", { reason, attempt: playerRecreateAttempts })
		schedulePlayerRecoveryReload(`recreate_timeout_${reason}`)
	}, PLAYER_RECOVERY_RECREATE_GRACE_MS)
}

function connectSettingsSocket() {
	const ws = new WebSocket(SETTINGS_WS_URL)

	ws.onmessage = (event) => {
		try {
			const payload = JSON.parse(event.data)
			if (!payload || payload.type !== "settings.updated") return

			const incomingVersion = Number(payload.version) || 0
			if (incomingVersion === 0 || incomingVersion >= bootSettingsVersion) {
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
	createYouTubePlayer()
}

function onPlayerError(e) {
	const errorCode = e && typeof e.data !== "undefined" ? e.data : "unknown"
	attemptPlayerRecovery(`youtube_error_${errorCode}`)
}

function onPlayerReady(e) {
	playerReady = true
	const now = Date.now()
	lastPlayerStateAt = now
	lastPlayerProgressAt = now
	lastObservedPlayerTime = 0
	consecutivePlayerAPIErrors = 0
	expectedPlayerToBePlaying = true
	resizeYouTubePlayer()
	e.target.playVideo()
	lastObservedPlaylistVideoId = getCurrentPlayerVideoId() || resolveVideoId()
	applyVideoOffset(bootConfig.layout)
	startPlayerWatchdog()
	if (isInfiniteVideoPlaybackEnabled()) {
		startLoopGuard()
	}
}

function onPlayerStateChange(e) {
	lastPlayerStateAt = Date.now()

	if (e.data === YT.PlayerState.PLAYING || e.data === YT.PlayerState.CUED) {
		persistCurrentPlaylistVideo()
	}

	if (e.data === YT.PlayerState.PLAYING) {
		expectedPlayerToBePlaying = true
		lastPlayerProgressAt = Date.now()
		markPlayerRecovered("state_playing")

		if (isInfiniteVideoPlaybackEnabled()) {
			startLoopGuard()
		} else {
			stopLoopGuard()
		}
	}

	if (e.data === YT.PlayerState.BUFFERING || e.data === YT.PlayerState.CUED) {
		expectedPlayerToBePlaying = true
	}

	if (e.data === YT.PlayerState.PAUSED || e.data === YT.PlayerState.UNSTARTED) {
		expectedPlayerToBePlaying = false
	}

	if (e.data === YT.PlayerState.ENDED) {
		expectedPlayerToBePlaying = false
		if (isInfiniteVideoPlaybackEnabled()) {
			restart()
		}
	}
}

function startLoopGuard() {
	stopLoopGuard()
	loopTimer = setInterval(() => {
		const durationResult = safePlayerCall("getDuration")
		const currentResult = safePlayerCall("getCurrentTime")
		if (!durationResult.ok || !currentResult.ok) return

		const duration = Number(durationResult.value) || 0
		const current = Number(currentResult.value) || 0

		if (duration > 0 && duration - current <= RESTART_THRESHOLD) {
			restart()
		}
	}, LOOP_GUARD_INTERVAL_MS)
}

function stopLoopGuard() {
	if (loopTimer) {
		clearInterval(loopTimer)
		loopTimer = null
	}
}

function restart() {
	stopLoopGuard()
	try {
		if (!player || typeof player.seekTo !== "function" || typeof player.playVideo !== "function") {
			return
		}
		player.seekTo(0.25, true)
		player.playVideo()
		consecutivePlayerAPIErrors = 0
	} catch (err) {
		consecutivePlayerAPIErrors += 1
		console.warn("player api exception", { methodName: "restart", consecutivePlayerAPIErrors, err })
		if (consecutivePlayerAPIErrors >= PLAYER_API_ERROR_THRESHOLD) {
			attemptPlayerRecovery("api_exception_restart")
		}
	}
}

function updateUI(data) {
	if (!data) return

	const cpuTemp = Math.round(data.cpu.temp_c)
	const cpuPackageTempRaw = data.cpu.package_temp_c
	const cpuPackageTemp = Number.isFinite(cpuPackageTempRaw) ? Math.round(cpuPackageTempRaw) : null
	const cpuUtil = Math.round(data.cpu.util_pct)
	const cpuPower = Math.round(data.cpu.power_w)
	document.getElementById("cpu_temp").textContent = cpuTemp
	document.getElementById("cpu_power").textContent = `CPU (${cpuPower}W)`
	const cpuPackageTempEl = document.getElementById("cpu_package_temp")
	if (cpuPackageTempEl) {
		cpuPackageTempEl.textContent = cpuPackageTemp !== null && cpuPackageTemp > 0 ? `(${cpuPackageTemp})` : ""
	}

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
	resizeYouTubePlayer()
	applyVideoOffset(bootConfig.layout)
})

window.addEventListener("beforeunload", () => {
	stopLoopGuard()
	stopPlayerWatchdog()
	if (recoveryGraceTimer) {
		clearTimeout(recoveryGraceTimer)
		recoveryGraceTimer = null
	}
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
