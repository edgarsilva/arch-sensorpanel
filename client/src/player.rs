//! libmpv playback: playlist built from settings, layout mapped onto mpv properties,
//! progress persisted back to the server, and recovery when streams fail.

use crate::server::{Config, Layout, Server};
use anyhow::{anyhow, Result};
use libmpv2::events::{Event, PropertyData};
use libmpv2::{Format, Mpv};
use std::sync::{Arc, Mutex};
use std::thread;
use std::time::Duration;

const DEFAULT_VIDEO_ID: &str = "AKfsikEXZHM";
const IDLE_PROP_ID: u64 = 1;

#[derive(Debug, Clone, PartialEq)]
pub struct Entry {
    pub video_id: String,
    /// Original settings URL (kept verbatim so the persisted value matches what panel.js writes).
    pub url: String,
    pub label: String,
}

impl Entry {
    /// Bare watch URL: dropping `list=` keeps yt-dlp from expanding the whole playlist.
    pub fn stream_url(&self) -> String {
        format!("https://www.youtube.com/watch?v={}", self.video_id)
    }
}

/// Port of `extractVideoIdFromURL()` in panel.js, plus bare 11-char ids.
pub fn extract_video_id(raw: &str) -> Option<String> {
    let raw = raw.trim();
    let is_id = |s: &str| s.len() == 11 && s.bytes().all(|b| b.is_ascii_alphanumeric() || b == b'-' || b == b'_');
    if is_id(raw) {
        return Some(raw.to_string());
    }
    let rest = raw.split_once("://").map(|(_, r)| r)?;
    let (host, path_query) = rest.split_once('/').unwrap_or((rest, ""));
    let (path, query) = path_query.split_once('?').unwrap_or((path_query, ""));
    let query = query.split('#').next().unwrap_or("");

    if host.contains("youtu.be") {
        let id = path.split('/').next().unwrap_or("");
        return (!id.is_empty()).then(|| id.to_string());
    }
    if !host.contains("youtube.com") {
        return None;
    }
    if let Some(v) = query.split('&').find_map(|kv| kv.strip_prefix("v=")) {
        if !v.is_empty() {
            return Some(v.to_string());
        }
    }
    let parts: Vec<&str> = path.split('/').filter(|p| !p.is_empty()).collect();
    parts
        .iter()
        .position(|p| matches!(*p, "embed" | "shorts" | "live"))
        .and_then(|i| parts.get(i + 1))
        .map(|id| id.to_string())
}

/// Port of `buildPlaylistEntries()` + `resolveMediaMode()`: returns the deduplicated entries and
/// the index to start from (the video saved in `media_sources[0]`).
pub fn build_entries(config: &Config) -> (Vec<Entry>, usize) {
    let is_playlist = config.media_type.eq_ignore_ascii_case("playlist");
    let first = config
        .media_sources
        .iter()
        .find(|s| !s.url.trim().is_empty())
        .and_then(|s| extract_video_id(&s.url).map(|id| (id, s)));

    if !is_playlist {
        let entry = match first {
            Some((video_id, s)) => Entry { video_id, url: s.url.clone(), label: s.label.clone() },
            None => Entry {
                video_id: DEFAULT_VIDEO_ID.into(),
                url: format!("https://www.youtube.com/watch?v={DEFAULT_VIDEO_ID}"),
                label: String::new(),
            },
        };
        return (vec![entry], 0);
    }

    let mut entries: Vec<Entry> = Vec::new();
    for source in &config.media_sources {
        let Some(video_id) = extract_video_id(&source.url) else { continue };
        if entries.iter().any(|e| e.video_id == video_id) {
            continue;
        }
        entries.push(Entry { video_id, url: source.url.trim().to_string(), label: source.label.trim().to_string() });
    }
    if entries.is_empty() {
        return build_entries(&Config { media_type: "video".into(), ..config.clone() });
    }
    let start = first
        .and_then(|(id, _)| entries.iter().position(|e| e.video_id == id))
        .unwrap_or(0);
    (entries, start)
}

/// Identity of the playlist, ignoring `media_sources[0]`: that slot is overwritten with the playing
/// video on every advance, so including it would restart playback on unrelated settings edits.
fn playlist_signature(config: &Config) -> String {
    let skip = usize::from(config.media_type.eq_ignore_ascii_case("playlist") && config.media_sources.len() > 1);
    let mut ids: Vec<String> = config.media_sources.iter().skip(skip).filter_map(|s| extract_video_id(&s.url)).collect();
    ids.sort();
    ids.dedup();
    format!("{}|{}", config.media_type.to_ascii_lowercase(), ids.join(","))
}

/// mpv `video-align-x/y` for the settings' fit/align/offset (see `applyVideoOffset` in panel.js).
/// Positive offsets move the picture right/down, i.e. reveal more of its left/top edge.
pub fn video_align(layout: &Layout) -> (f64, f64) {
    let off_x = layout.video_offset_x_pct.clamp(-100.0, 100.0) / 100.0;
    let off_y = layout.video_offset_y_pct.clamp(-100.0, 100.0) / 100.0;
    if layout.video_fit.eq_ignore_ascii_case("contain") {
        let base = match layout.video_align.to_ascii_lowercase().as_str() {
            "left" => -1.0,
            "right" => 1.0,
            _ => 0.0,
        };
        ((base + off_x).clamp(-1.0, 1.0), off_y)
    } else {
        (-off_x, -off_y)
    }
}

#[derive(Default)]
struct State {
    entries: Vec<Entry>,
    start: usize,
    signature: String,
    last_persisted: String,
}

pub struct Player {
    pub mpv: &'static Mpv,
    server: Server,
    state: Mutex<State>,
}

impl Player {
    pub fn new(server: Server) -> Result<Arc<Self>> {
        let mpv = Mpv::with_initializer(|init| {
            init.set_option("vo", "libmpv")?;
            init.set_option("hwdec", "vaapi,auto-safe")?;
            init.set_option("aid", "no")?;
            init.set_option("mute", "yes")?;
            init.set_option("ytdl", "yes")?;
            // Video only, sized for a 1920×480 strip; prefer VP9/AV1 which the GPU decodes.
            init.set_option("ytdl-format", "bv*[height<=1080][vcodec^=vp09]/bv*[height<=1080][vcodec^=av01]/bv*[height<=1080]/best")?;
            // ~1 min of 1080p VP9 read-ahead is plenty for a wallpaper stream; mpv otherwise
            // fills whatever it's given. No back-buffer: we never seek backwards (loop = reopen).
            init.set_option("demuxer-max-bytes", "32MiB")?;
            init.set_option("demuxer-max-back-bytes", "0")?;
            // Don't block the UI thread in render() waiting for the frame's display time.
            init.set_option("video-timing-offset", "0")?;
            init.set_option("idle", "yes")?;
            init.set_option("keep-open", "no")?;
            if let Ok(path) = std::env::var("SENSORPANEL_MPV_LOG") {
                init.set_option("log-file", path.as_str())?;
            }
            Ok(())
        })
        .map_err(|e| anyhow!("init mpv: {e}"))?;
        let mpv: &'static Mpv = Box::leak(Box::new(mpv));
        Ok(Arc::new(Self { mpv, server, state: Mutex::new(State::default()) }))
    }

    /// Apply settings: always re-applies layout, reloads the playlist only when it changed.
    pub fn apply(&self, config: &Config) {
        let layout = &config.layout;
        let set = |name: &str, value: &str| {
            if let Err(e) = self.mpv.set_property(name, value) {
                eprintln!("mpv: set {name}={value}: {e}");
            }
        };
        set("panscan", if layout.video_fit.eq_ignore_ascii_case("contain") { "0" } else { "1.0" });
        let (ax, ay) = video_align(layout);
        set("video-align-x", &format!("{ax:.3}"));
        set("video-align-y", &format!("{ay:.3}"));
        if layout.infinite_video_playback {
            set("loop-file", "inf");
            set("loop-playlist", "no");
        } else {
            set("loop-file", "no");
            set("loop-playlist", "inf");
        }

        let signature = playlist_signature(config);
        let mut state = self.state.lock().unwrap();
        if state.signature == signature && !state.entries.is_empty() {
            return;
        }
        let (entries, start) = build_entries(config);
        eprintln!("player: loading {} entries, starting at #{start} ({})", entries.len(), entries[start].video_id);
        state.entries = entries;
        state.start = start;
        state.signature = signature;
        state.last_persisted = config.media_sources.first().map(|s| s.url.clone()).unwrap_or_default();
        self.load_playlist(&state);
    }

    fn load_playlist(&self, state: &State) {
        // `stop` also clears the playlist, so indices below match `state.entries`.
        let mut result = self.mpv.command("stop", &[]);
        for entry in &state.entries {
            result = result.and(self.mpv.command("loadfile", &[&entry.stream_url(), "append"]));
        }
        result = result.and(self.mpv.set_property("playlist-pos", state.start as i64));
        if let Err(e) = result {
            eprintln!("mpv: load playlist: {e}");
        }
    }

    pub fn next(&self, delta: i64) {
        let cmd = if delta < 0 { "playlist-prev" } else { "playlist-next" };
        if let Err(e) = self.mpv.command(cmd, &["force"]) {
            eprintln!("mpv: {cmd}: {e}");
        }
    }

    fn on_file_loaded(&self) {
        let pos = self.mpv.get_property::<i64>("playlist-pos").unwrap_or(-1);
        let hwdec = self.mpv.get_property::<String>("hwdec-current").unwrap_or_default();
        let entry = {
            let mut state = self.state.lock().unwrap();
            let Some(entry) = usize::try_from(pos).ok().and_then(|p| state.entries.get(p)).cloned() else { return };
            state.start = pos as usize;
            if state.last_persisted == entry.url {
                None
            } else {
                state.last_persisted = entry.url.clone();
                Some(entry)
            }
        };
        eprintln!("player: playing #{pos} hwdec={hwdec:?}");
        if let Some(entry) = entry {
            let server = self.server.clone();
            thread::spawn(move || {
                if let Err(e) = server.persist_media_url(&entry.url) {
                    eprintln!("player: persist progress: {e:#}");
                }
            });
        }
    }

    /// Blocking mpv event loop on its own client handle; run on a dedicated thread.
    pub fn run_events(self: Arc<Self>) {
        let client = match self.mpv.create_client(Some("sensorpanel-events")) {
            Ok(c) => c,
            Err(e) => return eprintln!("mpv: create event client: {e}"),
        };
        let _ = client.disable_deprecated_events();
        let _ = client.observe_property("idle-active", Format::Flag, IDLE_PROP_ID);

        let mut backoff = Duration::from_secs(5);
        loop {
            match client.wait_event(-1.0) {
                Some(Ok(Event::FileLoaded)) => {
                    backoff = Duration::from_secs(5);
                    self.on_file_loaded();
                }
                Some(Ok(Event::VideoReconfig)) => {
                    let hwdec = self.mpv.get_property::<String>("hwdec-current").unwrap_or_default();
                    eprintln!("player: video reconfig hwdec={hwdec:?}");
                }
                // Everything failed (yt-dlp/network): mpv went idle. Retry with backoff.
                Some(Ok(Event::PropertyChange { reply_userdata: IDLE_PROP_ID, change: PropertyData::Flag(true), .. })) => {
                    let state = self.state.lock().unwrap();
                    if state.entries.is_empty() {
                        continue;
                    }
                    drop(state);
                    eprintln!("player: idle, retrying in {backoff:?}");
                    thread::sleep(backoff);
                    backoff = (backoff * 2).min(Duration::from_secs(120));
                    self.load_playlist(&self.state.lock().unwrap());
                }
                Some(Ok(Event::Shutdown)) => return,
                Some(Err(e)) => eprintln!("mpv: event error: {e}"),
                _ => {}
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::server::MediaSource;

    fn src(url: &str) -> MediaSource {
        MediaSource { url: url.into(), label: String::new() }
    }

    #[test]
    fn extracts_ids() {
        assert_eq!(extract_video_id("https://www.youtube.com/watch?v=ZRZSJDN4R18&list=PLx").as_deref(), Some("ZRZSJDN4R18"));
        assert_eq!(extract_video_id("https://www.youtube.com/watch?index=2&list=PLx&v=QEnpyfSWX5o").as_deref(), Some("QEnpyfSWX5o"));
        assert_eq!(extract_video_id("https://youtu.be/AKfsikEXZHM?t=3").as_deref(), Some("AKfsikEXZHM"));
        assert_eq!(extract_video_id("https://www.youtube.com/embed/PuEPbC2mOQI").as_deref(), Some("PuEPbC2mOQI"));
        assert_eq!(extract_video_id("AKfsikEXZHM").as_deref(), Some("AKfsikEXZHM"));
        assert_eq!(extract_video_id("https://example.com/watch?v=x"), None);
    }

    #[test]
    fn playlist_dedupes_and_starts_at_saved_video() {
        let config = Config {
            media_type: "playlist".into(),
            media_sources: vec![
                src("https://www.youtube.com/watch?v=CCCCCCCCCCC&list=PL"),
                src("https://www.youtube.com/watch?v=AAAAAAAAAAA&list=PL"),
                src("https://www.youtube.com/watch?v=BBBBBBBBBBB&list=PL"),
                src("https://www.youtube.com/watch?v=CCCCCCCCCCC&list=PL"),
            ],
            ..Default::default()
        };
        let (entries, start) = build_entries(&config);
        let ids: Vec<_> = entries.iter().map(|e| e.video_id.as_str()).collect();
        assert_eq!(ids, ["CCCCCCCCCCC", "AAAAAAAAAAA", "BBBBBBBBBBB"]);
        assert_eq!(start, 0);
        assert_eq!(entries[1].stream_url(), "https://www.youtube.com/watch?v=AAAAAAAAAAA");
    }

    #[test]
    fn signature_ignores_progress_slot() {
        let mut config = Config {
            media_type: "playlist".into(),
            media_sources: vec![src("AAAAAAAAAAA"), src("AAAAAAAAAAA"), src("BBBBBBBBBBB")],
            ..Default::default()
        };
        let before = playlist_signature(&config);
        config.media_sources[0] = src("BBBBBBBBBBB");
        assert_eq!(before, playlist_signature(&config));
    }

    #[test]
    fn video_mode_falls_back_to_default() {
        let (entries, start) = build_entries(&Config::default());
        assert_eq!((entries[0].video_id.as_str(), start), (DEFAULT_VIDEO_ID, 0));
    }

    #[test]
    fn align_maps_offsets() {
        let cover = Layout { video_fit: "cover".into(), video_offset_y_pct: 20.0, ..Default::default() };
        assert_eq!(video_align(&cover), (-0.0, -0.2));
        let contain = Layout { video_fit: "contain".into(), video_align: "right".into(), video_offset_x_pct: -50.0, ..Default::default() };
        assert_eq!(video_align(&contain), (0.5, 0.0));
    }
}
