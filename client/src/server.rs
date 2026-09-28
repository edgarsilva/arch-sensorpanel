//! HTTP access to the sensorpanel Go server (settings bootstrap + progress persistence).

use anyhow::{Context, Result};
use serde::Deserialize;
use serde_json::json;
use std::time::Duration;

#[derive(Debug, Clone, Default, Deserialize)]
pub struct CurrentSettings {
    #[serde(default)]
    pub config: Config,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct Config {
    #[serde(default)]
    pub media_type: String,
    #[serde(default)]
    pub media_sources: Vec<MediaSource>,
    #[serde(default)]
    pub layout: Layout,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct MediaSource {
    #[serde(default)]
    pub url: String,
    #[serde(default)]
    pub label: String,
}

/// Mirrors `models.SettingsLayout`; missing fields fall back to the same defaults as panel.js.
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(default)]
pub struct Layout {
    pub name: String,
    pub overlay_layout: String,
    pub theme: String,
    pub video_fit: String,
    pub video_align: String,
    pub video_offset_x_pct: f64,
    pub video_offset_y_pct: f64,
    pub infinite_video_playback: bool,
    pub overlay_disable_backdrop: bool,
    pub overlay_padding_top: f64,
    pub overlay_padding_right: f64,
    pub overlay_padding_bottom: f64,
    pub overlay_padding_left: f64,
    pub metrics_scale_pct: f64,
    pub metrics_offset_x: f64,
    pub metrics_offset_y: f64,
}

#[derive(Clone)]
pub struct Server {
    base: String,
    agent: ureq::Agent,
}

impl Server {
    pub fn new(base: &str) -> Self {
        let agent = ureq::Agent::config_builder()
            .timeout_global(Some(Duration::from_secs(5)))
            .build()
            .into();
        Self { base: base.trim_end_matches('/').to_string(), agent }
    }

    pub fn ws_url(&self, path: &str) -> String {
        let base = self
            .base
            .replacen("https://", "wss://", 1)
            .replacen("http://", "ws://", 1);
        format!("{base}{path}")
    }

    pub fn current_settings(&self) -> Result<CurrentSettings> {
        self.agent
            .get(format!("{}/api/settings/current", self.base))
            .call()
            .context("GET /api/settings/current")?
            .body_mut()
            .read_json()
            .context("decode current settings")
    }

    /// Same contract as `persistCurrentPlaylistVideo()` in panel.js: saves the playing video
    /// without broadcasting a settings change (which would make panels reload).
    pub fn persist_media_url(&self, url: &str) -> Result<()> {
        self.agent
            .patch(format!("{}/api/settings/current/field", self.base))
            .send_json(json!({ "field": "media_url", "value": url, "broadcast": false }))
            .context("PATCH /api/settings/current/field")?;
        Ok(())
    }
}
