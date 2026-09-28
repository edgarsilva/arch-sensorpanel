//! Player status → loading card, poster thumbnail and toasts.

use crate::player::{Entry, Status};
use crate::server::Server;
use crate::Panel;
use slint::{ComponentHandle, Image, Timer, TimerMode};
use std::cell::RefCell;
use std::path::PathBuf;
use std::thread;
use std::time::Duration;

const TOAST_DURATION: Duration = Duration::from_millis(3500);

thread_local! {
    static TOAST_TIMER: Timer = Timer::default();
    /// Video id whose thumbnail the poster should show (UI thread only).
    static POSTER_ID: RefCell<String> = const { RefCell::new(String::new()) };
}

fn title(entry: &Entry) -> String {
    let clean = crate::title::clean(&entry.label);
    if clean.is_empty() { entry.video_id.clone() } else { clean }
}

fn position(index: usize, total: usize) -> String {
    if total > 1 { format!("{} / {total}", index + 1) } else { String::new() }
}

/// Show `text` at the bottom of the panel for a few seconds.
pub fn toast(ui: &Panel, text: impl Into<slint::SharedString>) {
    ui.set_toast_text(text.into());
    ui.set_toast_visible(true);
    let weak = ui.as_weak();
    TOAST_TIMER.with(|t| {
        t.start(TimerMode::SingleShot, TOAST_DURATION, move || {
            if let Some(ui) = weak.upgrade() {
                ui.set_toast_visible(false);
            }
        })
    });
}

/// Loading card for the time before the player has anything to open (server not reachable yet).
pub fn starting(ui: &Panel, title: &str, detail: &str) {
    if matches!(ui.get_status_kind().as_str(), "starting" | "") && ui.get_status_title() != title {
        ui.set_status_kind("starting".into());
        ui.set_status_title(title.into());
        ui.set_status_detail(detail.into());
    }
}

/// Build the player's status callback: runs off the UI thread, applies on it.
pub fn handler(server: Server, weak: slint::Weak<Panel>) -> impl Fn(Status) + Send + Sync + 'static {
    move |status| {
        if let Status::Loading { entry, .. } = &status {
            fetch_poster(server.clone(), weak.clone(), entry.video_id.clone());
        }
        let _ = weak.upgrade_in_event_loop(move |ui| apply(&ui, status));
    }
}

fn apply(ui: &Panel, status: Status) {
    match status {
        Status::Loading { index, total, entry } => {
            ui.set_status_kind("loading".into());
            ui.set_status_title(title(&entry).into());
            ui.set_status_detail(position(index, total).into());
            let changed = POSTER_ID.with(|id| id.replace(entry.video_id.clone()) != entry.video_id);
            if changed {
                ui.set_poster(Image::default());
            }
            ui.set_show_poster(true);
        }
        Status::Playing { index, total, entry } => {
            ui.set_status_kind("".into());
            ui.set_show_poster(false);
            let pos = position(index, total);
            let pos = if pos.is_empty() { pos } else { format!("{pos} · ") };
            toast(ui, format!("Now playing · {pos}{}", title(&entry)));
        }
        Status::Buffering(true) if ui.get_status_kind().is_empty() => {
            ui.set_status_kind("buffering".into());
            ui.set_status_detail("".into());
        }
        Status::Buffering(false) if ui.get_status_kind() == "buffering" => {
            ui.set_status_kind("".into());
        }
        Status::Buffering(_) => {}
        Status::Failed { entry } => {
            let name = entry.as_ref().map(title).unwrap_or_else(|| "video".into());
            toast(ui, format!("Couldn't load {name} · skipping"));
        }
        Status::Retrying { secs } => {
            ui.set_status_kind("retrying".into());
            ui.set_status_title(format!("Retrying in {secs}s").into());
            ui.set_status_detail("Check the network or update yt-dlp".into());
        }
    }
}

fn thumb_path(video_id: &str) -> Option<PathBuf> {
    let base = std::env::var_os("XDG_CACHE_HOME")
        .map(PathBuf::from)
        .or_else(|| std::env::var_os("HOME").map(|h| PathBuf::from(h).join(".cache")))?;
    let dir = base.join("sensorpanel").join("thumbs");
    std::fs::create_dir_all(&dir).ok()?;
    Some(dir.join(format!("{video_id}.jpg")))
}

/// Fetch (or reuse the cached) YouTube thumbnail and show it if that video is still loading.
fn fetch_poster(server: Server, weak: slint::Weak<Panel>, video_id: String) {
    thread::spawn(move || {
        let Some(path) = thumb_path(&video_id) else { return };
        if !path.exists() {
            // maxresdefault is missing for some uploads; hqdefault always exists.
            let ok = ["maxresdefault", "hqdefault"].iter().any(|name| {
                server.download(&format!("https://i.ytimg.com/vi/{video_id}/{name}.jpg"), &path).is_ok()
            });
            if !ok {
                return eprintln!("poster: no thumbnail for {video_id}");
            }
        }
        let _ = weak.upgrade_in_event_loop(move |ui| {
            if POSTER_ID.with(|id| *id.borrow() != video_id) {
                return;
            }
            match Image::load_from_path(&path) {
                Ok(image) => ui.set_poster(image),
                Err(e) => {
                    eprintln!("poster: {}: {e:?}", path.display());
                    let _ = std::fs::remove_file(&path);
                }
            }
        });
    });
}
