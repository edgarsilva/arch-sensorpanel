//! WebSocket readers for `/metrics/ws` (telemetry snapshots) and `/settings/ws` (change notices).
//! Both reconnect every 2 s like panel.js.

use serde::Deserialize;
use std::thread;
use std::time::Duration;

const RECONNECT_DELAY: Duration = Duration::from_secs(2);

/// Mirrors `metrics.Snapshot` in internal/services/metrics/service.go.
#[derive(Debug, Clone, Copy, Default, Deserialize)]
#[serde(default)]
pub struct Snapshot {
    pub cpu: Cpu,
    pub ram: Ram,
    pub gpu: Gpu,
}

#[derive(Debug, Clone, Copy, Default, Deserialize)]
#[serde(default)]
pub struct Cpu {
    pub temp_c: f64,
    pub package_temp_c: f64,
    pub util_pct: f64,
    pub power_w: f64,
}

#[derive(Debug, Clone, Copy, Default, Deserialize)]
#[serde(default)]
pub struct Ram {
    pub total_gb: f64,
    pub used_gb: f64,
    pub avail_gb: f64,
    pub used_pct: f64,
}

#[derive(Debug, Clone, Copy, Default, Deserialize)]
#[serde(default)]
pub struct Gpu {
    pub edge_c: f64,
    pub hotspot_c: f64,
    pub vram_c: f64,
    pub vram_used_gb: f64,
    pub vram_total_gb: f64,
    pub vram_used_pct: f64,
    pub power_w: f64,
    pub util_pct: f64,
}

#[derive(Debug, Deserialize)]
struct SettingsNotice {
    #[serde(rename = "type")]
    kind: String,
    #[serde(default)]
    version: i64,
}

/// Connect to `url` forever, handing every text frame to `on_text` and connection
/// state changes to `on_connected`.
fn run_ws(url: &str, mut on_connected: impl FnMut(bool), mut on_text: impl FnMut(&str)) -> ! {
    loop {
        match tungstenite::connect(url) {
            Ok((mut socket, _)) => {
                on_connected(true);
                loop {
                    match socket.read() {
                        Ok(tungstenite::Message::Text(text)) => on_text(text.as_str()),
                        Ok(tungstenite::Message::Close(_)) | Err(_) => break,
                        Ok(_) => {}
                    }
                }
            }
            Err(e) => eprintln!("ws {url}: {e}"),
        }
        on_connected(false);
        thread::sleep(RECONNECT_DELAY);
    }
}

pub fn spawn_metrics(url: String, on_status: impl Fn(bool) + Send + 'static, on_snapshot: impl Fn(Snapshot) + Send + 'static) {
    thread::spawn(move || {
        run_ws(&url, on_status, |text| match serde_json::from_str::<Snapshot>(text) {
            Ok(snapshot) => on_snapshot(snapshot),
            Err(e) => eprintln!("metrics: bad snapshot: {e}"),
        })
    });
}

/// Calls `on_change` with the new version for every `settings.updated` notice, and once after
/// each reconnect (changes may have been missed while disconnected).
pub fn spawn_settings_watch(url: String, on_change: impl Fn(i64) + Send + 'static) {
    thread::spawn(move || {
        let mut was_connected = true;
        run_ws(
            &url,
            |connected| {
                if connected && !was_connected {
                    on_change(0);
                }
                was_connected = connected;
            },
            |text| {
                if let Ok(notice) = serde_json::from_str::<SettingsNotice>(text) {
                    if notice.kind == "settings.updated" {
                        on_change(notice.version);
                    }
                }
            },
        )
    });
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_snapshot() {
        let raw = r#"{"cpu":{"temp_c":37.3,"package_temp_c":37.3,"util_pct":1.05,"power_w":43.4},
            "ram":{"total_gb":62.7,"used_gb":21.3,"avail_gb":41.3,"used_pct":34.0},
            "gpu":{"edge_c":45,"hotspot_c":47,"vram_c":48,"vram_used_gb":3.6,"vram_total_gb":15.9,
                   "vram_used_pct":22.7,"power_w":44,"util_pct":8}}"#;
        let s: Snapshot = serde_json::from_str(raw).unwrap();
        assert_eq!(s.gpu.hotspot_c, 47.0);
        assert_eq!(s.ram.used_pct, 34.0);
        let partial: Snapshot = serde_json::from_str(r#"{"cpu":{"temp_c":50}}"#).unwrap();
        assert_eq!((partial.cpu.temp_c, partial.gpu.util_pct), (50.0, 0.0));
    }
}
