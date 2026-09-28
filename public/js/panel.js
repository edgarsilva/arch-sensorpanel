let player
let loopTimer
let watchdogTimer
let settingsWS
let metricsWS
let settingsReconnectTimer
let metricsReconnectTimer
let mediaMode = { kind: "youtube", videoId: "", playlistId: "" }

const LOOP_GUARD_INTERVAL_MS = 400
const RESTART_THRESHOLD = 0.800
const DEFAULT_VIDEO_ID = "AKfsikEXZHM"
const WS_URL = `${window.location.protocol === "https:" ? "wss" : "ws"}://${window.location.host}/metrics/ws`
const SETTINGS_WS_URL = `${window.location.protocol === "https:" ? "wss" : "ws"}://${window.location.host}/settings/ws`
const SETTINGS_RELOAD_DELAY_MS = 350
const RECOVERY_RELOAD_WINDOW_MS = 90_000
const MAX_RECOVERY_RELOADS_PER_WINDOW = 3
const RECOVERY_RELOAD_SESSION_KEY = "sensorpanel.recovery.reload.window"

let previousPlayerTime = null
let noProgressChecks = 0
let playerApiErrorCount = 0
let recoveryReloadScheduled = false
let pageUnloading = false

let bootConfig = {
	media_type: "video",
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
let lastKnownVideoId = ""
let playlistEntries = []
let playlistCursor = 0

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

function resizeYoutubePlayer() {
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
			const fromQuery = url.searchParams.get("v") || ""
			if (fromQuery) return fromQuery

			const parts = url.pathname.split("/").filter(Boolean)
			const markerIndex = parts.findIndex((part) => part === "embed" || part === "shorts" || part === "live")
			if (markerIndex >= 0 && parts[markerIndex + 1]) {
				return parts[markerIndex + 1]
			}
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
	const sources = Array.isArray(bootConfig.media_sources) ? bootConfig.media_sources : []
	const first = sources.find((source) => source && String(source.url || "").trim() !== "") || sources[0] || null
	const fallback = { kind: "youtube", videoId: DEFAULT_VIDEO_ID, playlistId: "" }
	if (!first) return fallback

	const configuredType = String(bootConfig.media_type || "").toLowerCase()
	const sourceKind = String(first.kind || "").toLowerCase()
	const mediaType = configuredType === "playlist" || sourceKind === "playlist" ? "playlist" : "video"
	const trimmedURL = String(first.url || "").trim()
	if (!trimmedURL) return fallback

	const videoId = (() => {
		const parsed = extractVideoIdFromURL(trimmedURL)
		if (parsed) return parsed
		if (/^[A-Za-z0-9_-]{11}$/.test(trimmedURL)) return trimmedURL
		return DEFAULT_VIDEO_ID
	})()

	if (mediaType === "playlist") {
		const playlistId = extractPlaylistIdFromURL(trimmedURL) || trimmedURL
		if (playlistId) {
			return { kind: "playlist", videoId, playlistId }
		}
	}

	return { kind: "youtube", videoId, playlistId: "" }
}

function buildPlaylistEntries() {
	if (String(bootConfig.media_type || "").toLowerCase() !== "playlist") {
		return []
	}

	const sources = Array.isArray(bootConfig.media_sources) ? bootConfig.media_sources : []
	const seen = new Set()
	const entries = []

	for (const source of sources) {
		const rawURL = String((source && source.url) || "").trim()
		if (!rawURL) continue
		const videoId = extractVideoIdFromURL(rawURL)
		if (!videoId || seen.has(videoId)) continue
		seen.add(videoId)
		entries.push({
			videoId,
			url: rawURL,
			label: String((source && source.label) || "").trim(),
		})
	}

	return entries
}

function setupPlaylistRuntime() {
	playlistEntries = buildPlaylistEntries()
	if (playlistEntries.length === 0) {
		playlistCursor = 0
		return
	}

	const activeVideoId = resolveVideoId()
	const index = playlistEntries.findIndex((entry) => entry.videoId === activeVideoId)
	playlistCursor = index >= 0 ? index : 0
}

function syncPlaylistCursorWithCurrentVideo() {
	if (playlistEntries.length === 0) return
	const currentVideoId = getCurrentPlayerVideoId()
	if (!currentVideoId) return
	const index = playlistEntries.findIndex((entry) => entry.videoId === currentVideoId)
	if (index >= 0) {
		playlistCursor = index
	}
}

function playPlaylistIndex(index, reason) {
	if (!player || playlistEntries.length === 0) return

	const total = playlistEntries.length
	const nextIndex = ((index % total) + total) % total
	const entry = playlistEntries[nextIndex]
	if (!entry || !entry.videoId) return

	try {
		playlistCursor = nextIndex
		player.loadVideoById(entry.videoId)
		playerApiErrorCount = 0
		noProgressChecks = 0
		previousPlayerTime = null
		console.info("manual playlist navigation", { reason, playlistCursor, videoId: entry.videoId })
	} catch (err) {
		playerApiErrorCount += 1
		console.warn("player api exception", { methodName: "loadVideoById", playerApiErrorCount, err })
		if (playerApiErrorCount >= 3) {
			attemptPlayerRecovery(`api_exception_${reason}`)
		}
	}
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
		if (mediaMode.kind === "playlist" && playlistEntries.length > 0) {
			playPlaylistIndex(playlistCursor - 1, "manual_prev")
		}
	})

	nextBtn.addEventListener("click", () => {
		if (mediaMode.kind === "playlist" && playlistEntries.length > 0) {
			playPlaylistIndex(playlistCursor + 1, "manual_next")
		}
	})
}

function resolveVideoId() {
	return mediaMode.videoId || DEFAULT_VIDEO_ID
}

function currentMediaURLFromConfig() {
	const sources = Array.isArray(bootConfig.media_sources) ? bootConfig.media_sources : []
	const first = sources.find((source) => source && String(source.url || "").trim() !== "") || sources[0] || null
	return String((first && first.url) || "").trim()
}

function buildPlaylistMediaURL(videoId, playlistId) {
	const v = String(videoId || "").trim()
	const list = String(playlistId || "").trim()
	if (!v || !list) return ""
	return `https://www.youtube.com/watch?v=${encodeURIComponent(v)}&list=${encodeURIComponent(list)}`
}

function videoIdFromURL(rawURL) {
	return String(extractVideoIdFromURL(rawURL) || "").trim()
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
	if (settingsReloadScheduled || recoveryReloadScheduled) return
	settingsReloadScheduled = true
	closeRealtimeConnections()
	setTimeout(() => {
		window.location.reload()
	}, SETTINGS_RELOAD_DELAY_MS)
}

function resetRecoveryReloadWindow() {
	try {
		sessionStorage.removeItem(RECOVERY_RELOAD_SESSION_KEY)
	} catch (_) {
	}
}

function canTriggerRecoveryReload() {
	const now = Date.now()
	try {
		const raw = sessionStorage.getItem(RECOVERY_RELOAD_SESSION_KEY)
		let windowStart = now
		let count = 0

		if (raw) {
			const parsed = JSON.parse(raw)
			const parsedWindowStart = Number(parsed && parsed.windowStart)
			const parsedCount = Number(parsed && parsed.count)
			if (Number.isFinite(parsedWindowStart) && Number.isFinite(parsedCount) && now - parsedWindowStart <= RECOVERY_RELOAD_WINDOW_MS) {
				windowStart = parsedWindowStart
				count = parsedCount
			}
		}

		count += 1
		sessionStorage.setItem(RECOVERY_RELOAD_SESSION_KEY, JSON.stringify({ windowStart, count }))
		return count <= MAX_RECOVERY_RELOADS_PER_WINDOW
	} catch (_) {
		return true
	}
}

function showRecoveryFailureState(reason) {
	setConnectionState("reconnecting")
	const text = document.getElementById("conn_text")
	if (text) {
		text.textContent = "player failed"
	}
	console.error("player recovery aborted to prevent reload loop", { reason })
}

function clearSettingsReconnectTimer() {
	if (!settingsReconnectTimer) return
	clearTimeout(settingsReconnectTimer)
	settingsReconnectTimer = null
}

function clearMetricsReconnectTimer() {
	if (!metricsReconnectTimer) return
	clearTimeout(metricsReconnectTimer)
	metricsReconnectTimer = null
}

function closeSettingsSocket() {
	clearSettingsReconnectTimer()
	if (!settingsWS) return
	const ws = settingsWS
	settingsWS = null
	ws.onopen = null
	ws.onmessage = null
	ws.onerror = null
	ws.onclose = null
	try {
		ws.close()
	} catch (_) {
	}
}

function closeMetricsSocket() {
	clearMetricsReconnectTimer()
	if (!metricsWS) return
	const ws = metricsWS
	metricsWS = null
	ws.onopen = null
	ws.onmessage = null
	ws.onerror = null
	ws.onclose = null
	try {
		ws.close()
	} catch (_) {
	}
}

function closeRealtimeConnections() {
	closeSettingsSocket()
	closeMetricsSocket()
}

function triggerHardReload(reason) {
	if (recoveryReloadScheduled || pageUnloading) return
	recoveryReloadScheduled = true
	console.warn("player recovery reloading page", { reason, previousPlayerTime, playerApiErrorCount })
	closeRealtimeConnections()
	setTimeout(() => {
		window.location.reload()
	}, 0)
}

function safePlayerCall(methodName) {
	if (!player || typeof player[methodName] !== "function") return { ok: false, missing: true }
	try {
		const value = player[methodName]()
		playerApiErrorCount = 0
		return { ok: true, value }
	} catch (err) {
		playerApiErrorCount += 1
		console.warn("player api exception", { methodName, playerApiErrorCount, err })
		if (playerApiErrorCount >= 3) {
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

function resetWatchdogFailureCounters() {
	noProgressChecks = 0
}

function startPlayerWatchdog() {
	stopPlayerWatchdog()
	watchdogTimer = setInterval(() => {
		if (!player || recoveryReloadScheduled || pageUnloading) return

		if (document.hidden) {
			previousPlayerTime = null
			noProgressChecks = 0
			return
		}

		const stateResult = safePlayerCall("getPlayerState")
		if (!stateResult.ok) return
		if (!(window.YT && YT.PlayerState) || stateResult.value !== YT.PlayerState.PLAYING) {
			previousPlayerTime = null
			noProgressChecks = 0
			return
		}

		const timeResult = safePlayerCall("getCurrentTime")
		if (!timeResult.ok) return

		const currentTime = Number(timeResult.value) || 0
		if (previousPlayerTime === null) {
			console.debug("player watchdog baseline set", { currentTime })
			previousPlayerTime = currentTime
			noProgressChecks = 0
			return
		}

		const delta = Math.abs(currentTime - previousPlayerTime)
		const progressed = delta > 0.5

		if (progressed) {
			previousPlayerTime = currentTime
			noProgressChecks = 0
			return
		}

		noProgressChecks += 1
		if (noProgressChecks === 1) {
			console.debug("player watchdog no video progress", { previousPlayerTime, currentTime, delta, failedChecks: noProgressChecks })
		}
		if (noProgressChecks >= 4) {
			console.warn("player watchdog triggering recovery", {
				reason: "stalled_time",
				failedChecks: noProgressChecks,
				currentTime,
				delta,
			})
			attemptPlayerRecovery("stalled_time")
		}
	}, 2000)
}

function createYoutubePlayer() {
	if (!window.YT || typeof YT.Player !== "function") {
		console.warn("youtube iframe api not ready during player creation")
		return false
	}

	const playerVars = {
		autoplay: 1,
		mute: 1,
		controls: 0,
		disablekb: 0,
		fs: 0,
		iv_load_policy: 3,
		rel: 0,
		playsinline: 1,
		origin: window.location.origin,
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

function applyYoutubeIframeAttributes() {
	if (!player || typeof player.getIframe !== "function") return
	try {
		const iframe = player.getIframe()
		if (!iframe) return
		iframe.setAttribute("allow", "accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share")
		iframe.setAttribute("referrerpolicy", "strict-origin-when-cross-origin")
		iframe.setAttribute("frameborder", "0")
		iframe.setAttribute("allowfullscreen", "")
		iframe.setAttribute("title", "Background video player")
	} catch (_) {
	}
}

function attemptPlayerRecovery(reason) {
	if (recoveryReloadScheduled || pageUnloading) {
		return
	}

	stopLoopGuard()
	stopPlayerWatchdog()
	resetWatchdogFailureCounters()
	if (!canTriggerRecoveryReload()) {
		showRecoveryFailureState(reason)
		return
	}
	triggerHardReload(reason)
}

function connectSettingsSocket() {
	if (settingsReloadScheduled || recoveryReloadScheduled || pageUnloading) return
	clearSettingsReconnectTimer()
	if (settingsWS && (settingsWS.readyState === WebSocket.CONNECTING || settingsWS.readyState === WebSocket.OPEN)) {
		return
	}
	closeSettingsSocket()

	const ws = new WebSocket(SETTINGS_WS_URL)
	settingsWS = ws

	ws.onmessage = (event) => {
		if (ws !== settingsWS) return
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
		if (ws !== settingsWS) return
		ws.close()
	}

	ws.onclose = () => {
		if (ws !== settingsWS) return
		settingsWS = null
		if (settingsReloadScheduled || recoveryReloadScheduled || pageUnloading) return
		settingsReconnectTimer = setTimeout(() => {
			settingsReconnectTimer = null
			connectSettingsSocket()
		}, 2000)
	}
}

function onYouTubeIframeAPIReady() {
	createYoutubePlayer()
}

function onPlayerError(e) {
	const errorCode = e && typeof e.data !== "undefined" ? e.data : "unknown"
	attemptPlayerRecovery(`youtube_error_${errorCode}`)
}

function onPlayerReady(e) {
	resetRecoveryReloadWindow()
	previousPlayerTime = null
	noProgressChecks = 0
	playerApiErrorCount = 0
	applyYoutubeIframeAttributes()
	resizeYoutubePlayer()
	e.target.playVideo()
	syncPlaylistCursorWithCurrentVideo()
	lastKnownVideoId = getCurrentPlayerVideoId() || resolveVideoId()
	lastObservedPlaylistVideoId = lastKnownVideoId
	applyVideoOffset(bootConfig.layout)
	startPlayerWatchdog()
	if (isInfiniteVideoPlaybackEnabled()) {
		startLoopGuard()
	}
}

function onPlayerStateChange(e) {
	if (e.data === YT.PlayerState.PLAYING || e.data === YT.PlayerState.CUED) {
		const currentVideoId = getCurrentPlayerVideoId()
		if (currentVideoId) {
			lastKnownVideoId = currentVideoId
		}
		syncPlaylistCursorWithCurrentVideo()
		persistCurrentPlaylistVideo()
	}

	if (e.data === YT.PlayerState.PLAYING) {
		resetRecoveryReloadWindow()

		if (isInfiniteVideoPlaybackEnabled()) {
			startLoopGuard()
		} else {
			stopLoopGuard()
		}
	}

	if (e.data === YT.PlayerState.ENDED) {
		if (isInfiniteVideoPlaybackEnabled()) {
			const videoId = lastKnownVideoId || getCurrentPlayerVideoId() || resolveVideoId()
			if (videoId && player && typeof player.loadVideoById === "function") {
				try {
					player.loadVideoById(videoId, 0)
					playerApiErrorCount = 0
					noProgressChecks = 0
					previousPlayerTime = null
				} catch (err) {
					playerApiErrorCount += 1
					console.warn("player api exception", { methodName: "loadVideoById(loop)", playerApiErrorCount, err })
					if (playerApiErrorCount >= 3) {
						attemptPlayerRecovery("api_exception_loop_reload")
					}
				}
			} else {
				restart()
			}
			return
		}

		if (mediaMode.kind === "playlist" && playlistEntries.length > 0) {
			playPlaylistIndex(playlistCursor + 1, "auto_next")
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
		playerApiErrorCount = 0
	} catch (err) {
		playerApiErrorCount += 1
		console.warn("player api exception", { methodName: "restart", playerApiErrorCount, err })
		if (playerApiErrorCount >= 3) {
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
	if (recoveryReloadScheduled || pageUnloading) return
	clearMetricsReconnectTimer()
	if (metricsWS && (metricsWS.readyState === WebSocket.CONNECTING || metricsWS.readyState === WebSocket.OPEN)) {
		return
	}
	closeMetricsSocket()

	setConnectionState("connecting")
	const ws = new WebSocket(WS_URL)
	metricsWS = ws

	ws.onopen = () => {
		if (ws !== metricsWS) return
		setConnectionState("live")
	}

	ws.onmessage = (event) => {
		if (ws !== metricsWS) return
		try {
			updateUI(JSON.parse(event.data))
		} catch (err) {
			console.warn("invalid ws payload", err)
		}
	}

	ws.onerror = () => {
		if (ws !== metricsWS) return
		setConnectionState("reconnecting")
		ws.close()
	}

	ws.onclose = () => {
		if (ws !== metricsWS) return
		metricsWS = null
		if (recoveryReloadScheduled || pageUnloading) return
		setConnectionState("reconnecting")
		metricsReconnectTimer = setTimeout(() => {
			metricsReconnectTimer = null
			connectMetricsSocket()
		}, 2000)
	}
}

window.onYouTubeIframeAPIReady = onYouTubeIframeAPIReady

window.addEventListener("resize", () => {
	resizeYoutubePlayer()
	applyVideoOffset(bootConfig.layout)
})

window.addEventListener("beforeunload", () => {
	pageUnloading = true
	stopLoopGuard()
	stopPlayerWatchdog()
	closeRealtimeConnections()
})

bootstrapSettings().finally(() => {
	mediaMode = resolveMediaMode()
	setupPlaylistRuntime()
	applyTheme(bootConfig.layout && bootConfig.layout.theme)
	applyVideoLayout(bootConfig.layout)
	applyVideoOffset(bootConfig.layout)

	applyLayout(bootConfig.layout)
	applyOverlayPadding(bootConfig.layout)
	applyMetricsTuning(bootConfig.layout)
	setPlaylistControlsVisible(mediaMode.kind === "playlist" && playlistEntries.length > 1)
	bindPlaylistControls()
	connectSettingsSocket()
	connectMetricsSocket()
})
