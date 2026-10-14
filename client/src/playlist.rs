//! Playlist picker: titles and thumbnails for the list opened beside the prev/next buttons.

use crate::player::Entry;
use crate::server::Server;
use crate::{status, Panel, PlaylistItem};
use slint::{ComponentHandle, Image, Model, ModelRc, VecModel};
use std::rc::Rc;
use std::sync::atomic::{AtomicU64, Ordering};
use std::thread;

/// Bumped on every open and close, so a thumbnail fetch for an older listing stops early
/// and its results are dropped.
static GENERATION: AtomicU64 = AtomicU64::new(0);

/// Fill the list from `entries` and show it; thumbnails arrive as they are cached.
pub fn open(ui: &Panel, server: &Server, entries: Vec<Entry>) {
    let generation = GENERATION.fetch_add(1, Ordering::Relaxed) + 1;
    let items: Vec<PlaylistItem> =
        entries.iter().map(|e| PlaylistItem { title: status::title(e).into(), thumb: Image::default() }).collect();
    ui.set_playlist_items(ModelRc::from(Rc::new(VecModel::from(items))));
    ui.set_playlist_open(true);

    let (server, weak) = (server.clone(), ui.as_weak());
    thread::spawn(move || {
        for (row, entry) in entries.into_iter().enumerate() {
            if GENERATION.load(Ordering::Relaxed) != generation {
                return;
            }
            // 320×180 (16:9, ~230 KB decoded): the full-size posters would cost ~3.7 MB each.
            let file = format!("{}-mq.jpg", entry.video_id);
            let Some(path) = status::thumbnail(&server, &entry.video_id, &["mqdefault", "hqdefault"], &file) else {
                eprintln!("playlist: no thumbnail for {}", entry.video_id);
                continue;
            };
            let _ = weak.upgrade_in_event_loop(move |ui| {
                if GENERATION.load(Ordering::Relaxed) != generation {
                    return;
                }
                let image = match Image::load_from_path(&path) {
                    Ok(image) => image,
                    Err(e) => {
                        eprintln!("playlist: {}: {e:?}", path.display());
                        let _ = std::fs::remove_file(&path);
                        return;
                    }
                };
                let items = ui.get_playlist_items();
                if let Some(mut item) = items.row_data(row) {
                    item.thumb = image;
                    items.set_row_data(row, item);
                }
            });
        }
    });
}

/// Hide the list and drop its rows (Slint's 5 MB image cache may keep some thumbnails).
pub fn close(ui: &Panel) {
    GENERATION.fetch_add(1, Ordering::Relaxed);
    ui.set_playlist_open(false);
    ui.set_playlist_items(ModelRc::default());
}
