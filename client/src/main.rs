//! sensorpanel-client: a single native window that renders the panel video with libmpv
//! (OpenGL underlay) and the telemetry overlay with Slint on top.

mod player;
mod server;
mod telemetry;
mod theme;

use anyhow::{anyhow, Result};
use libmpv2::render::{OpenGLInitParams, RenderContext, RenderParam, RenderParamApiType};
use player::Player;
use server::{Config, Server};
use raw_window_handle::{HasDisplayHandle, RawDisplayHandle};
use slint::winit_030::winit::platform::wayland::WindowAttributesExtWayland;
use slint::winit_030::winit::window::Fullscreen;
use slint::{ComponentHandle, GraphicsAPI, RenderingState};
use std::ffi::{c_char, c_void, CString};
use std::sync::Arc;
use std::thread;
use std::time::Duration;
use telemetry::Snapshot;

slint::include_modules!();

const APP_ID: &str = "sensorpanel-client";

#[link(name = "EGL")]
unsafe extern "C" {
    fn eglGetProcAddress(name: *const c_char) -> *mut c_void;
}

/// Slint's winit/femtovg renderer drives an EGL context on Wayland, so mpv can resolve GL
/// entry points straight from EGL (Mesa returns core functions too).
fn gl_proc_address(_: &(), name: &str) -> *mut c_void {
    let Ok(name) = CString::new(name) else { return std::ptr::null_mut() };
    unsafe { eglGetProcAddress(name.as_ptr()) }
}

fn main() -> Result<()> {
    let windowed = std::env::args().any(|a| a == "--windowed");
    let base_url = std::env::var("SENSORPANEL_URL").unwrap_or_else(|_| "http://127.0.0.1:9070".into());
    let server = Server::new(&base_url);

    slint::BackendSelector::new()
        .backend_name("winit".into())
        .renderer_name("femtovg".into())
        .with_winit_window_attributes_hook(move |attrs| {
            // Don't ask the compositor to focus us: the panel is display-only and must not
            // pull focus off the main monitors (Hyprland honours activation requests).
            let attrs = attrs.with_name(APP_ID, APP_ID).with_active(false);
            if windowed { attrs } else { attrs.with_fullscreen(Some(Fullscreen::Borderless(None))) }
        })
        .select()
        .map_err(|e| anyhow!("select slint backend: {e}"))?;

    let ui = Panel::new()?;
    let player = Player::new(server.clone())?;

    install_video_underlay(&ui, player.mpv)?;
    thread::spawn({
        let player = player.clone();
        move || player.run_events()
    });

    ui.on_playlist_step({
        let player = player.clone();
        move |delta| player.next(delta.into())
    });

    let weak = ui.as_weak();
    telemetry::spawn_metrics(
        server.ws_url("/metrics/ws"),
        {
            let weak = weak.clone();
            move |connected| {
                let _ = weak.upgrade_in_event_loop(move |ui| ui.set_connected(connected));
            }
        },
        {
            let weak = weak.clone();
            move |snapshot| {
                let _ = weak.upgrade_in_event_loop(move |ui| apply_snapshot(&ui, &snapshot));
            }
        },
    );

    let reload = {
        let (server, player, weak) = (server.clone(), player.clone(), weak.clone());
        move || load_settings(&server, &player, &weak)
    };
    let initial = reload.clone();
    thread::spawn(initial);
    telemetry::spawn_settings_watch(server.ws_url("/settings/ws"), move |version| {
        eprintln!("settings: changed (version {version}), re-applying");
        reload();
    });

    ui.run()?;
    Ok(())
}

/// Hook libmpv's OpenGL renderer into Slint's frame: mpv paints the video into the default
/// framebuffer before Slint draws the (transparent-background) overlay on top.
fn install_video_underlay(ui: &Panel, mpv: &'static libmpv2::Mpv) -> Result<()> {
    let weak = ui.as_weak();
    let mut render: Option<RenderContext<'static>> = None;
    ui.window()
        .set_rendering_notifier(move |state, api| match state {
            RenderingState::RenderingSetup => {
                if !matches!(api, GraphicsAPI::NativeOpenGL { .. }) {
                    eprintln!("video: renderer is not OpenGL ({api:?}); video disabled");
                    return;
                }
                let mut params = vec![
                    RenderParam::ApiType(RenderParamApiType::OpenGl),
                    RenderParam::InitParams(OpenGLInitParams { get_proc_address: gl_proc_address, ctx: () }),
                ];
                // mpv needs the wl_display for zero-copy VA-API interop (else it falls back to *-copy).
                if let Some(ui) = weak.upgrade() {
                    let handle = ui.window().window_handle();
                    if let Ok(RawDisplayHandle::Wayland(h)) = handle.display_handle().map(|d| d.as_raw()) {
                        params.push(RenderParam::WaylandDisplay(h.display.as_ptr()));
                    }
                }
                let ctx = mpv.create_render_context(params);
                match ctx {
                    Ok(mut ctx) => {
                        let weak = weak.clone();
                        ctx.set_update_callback(move || {
                            let _ = weak.upgrade_in_event_loop(|ui| ui.window().request_redraw());
                        });
                        render = Some(ctx);
                    }
                    Err(e) => eprintln!("video: create mpv render context: {e}"),
                }
            }
            RenderingState::BeforeRendering => {
                let (Some(ctx), Some(ui)) = (&render, weak.upgrade()) else { return };
                let size = ui.window().size();
                if let Err(e) = ctx.render::<()>(0, size.width as i32, size.height as i32, true) {
                    eprintln!("video: render: {e}");
                }
            }
            RenderingState::RenderingTeardown => render = None,
            _ => {}
        })
        .map_err(|e| anyhow!("set rendering notifier: {e:?}"))
}

/// Fetch current settings (retrying until the server is up) and apply them to mpv and the UI.
fn load_settings(server: &Server, player: &Arc<Player>, weak: &slint::Weak<Panel>) {
    let settings = loop {
        match server.current_settings() {
            Ok(s) => break s,
            Err(e) => {
                eprintln!("settings: {e:#}; retrying");
                thread::sleep(Duration::from_secs(2));
            }
        }
    };
    player.apply(&settings.config);
    let config = settings.config;
    let _ = weak.upgrade_in_event_loop(move |ui| apply_layout(&ui, &config));
}

fn apply_layout(ui: &Panel, config: &Config) {
    let l = &config.layout;
    let position = match l.name.to_ascii_lowercase().as_str() {
        p @ ("right" | "center" | "cover") => p.to_string(),
        _ => "left".to_string(),
    };
    ui.set_position(position.into());
    ui.set_horizontal(l.overlay_layout.eq_ignore_ascii_case("row"));
    ui.set_backdrop(!l.overlay_disable_backdrop);
    let pad = |v: f64| v.clamp(0.0, 500.0) as f32;
    ui.set_pad_top(pad(l.overlay_padding_top));
    ui.set_pad_right(pad(l.overlay_padding_right));
    ui.set_pad_bottom(pad(l.overlay_padding_bottom));
    ui.set_pad_left(pad(l.overlay_padding_left));
    let scale = if l.metrics_scale_pct == 0.0 { 100.0 } else { l.metrics_scale_pct.clamp(50.0, 200.0) };
    ui.set_scale((scale / 100.0) as f32);
    ui.set_offset_x(l.metrics_offset_x.clamp(-1000.0, 1000.0) as f32);
    ui.set_offset_y(l.metrics_offset_y.clamp(-1000.0, 1000.0) as f32);

    let is_playlist = config.media_type.eq_ignore_ascii_case("playlist");
    ui.set_show_controls(is_playlist && player::build_entries(config).0.len() > 1);

    let palette = theme::palette(&l.theme);
    let t = ui.global::<Theme>();
    t.set_primary(theme::rgb(palette.primary));
    t.set_secondary(theme::rgb(palette.secondary));
    t.set_base_200(theme::rgb(palette.base_200));
    t.set_base_content(theme::rgb(palette.base_content));
    t.set_success(theme::rgb(palette.success));
}

/// Port of `updateUI()` in panel.js.
fn apply_snapshot(ui: &Panel, s: &Snapshot) {
    let r1 = |v: f64| (v * 10.0).round() / 10.0;
    let cpu_temp = s.cpu.temp_c.round();
    ui.set_cpu_util(s.cpu.util_pct as f32);
    ui.set_cpu_title(format!("CPU ({}W)", s.cpu.power_w.round()).into());
    ui.set_cpu_temp(format!("{cpu_temp}").into());
    let pkg = s.cpu.package_temp_c.round();
    ui.set_cpu_pkg(if pkg > 0.0 { format!("({pkg})") } else { String::new() }.into());
    ui.set_cpu_temp_val(cpu_temp as f32);
    ui.set_cpu_temp_color(theme::temp_color(cpu_temp, 95.0, 35.0));
    ui.set_ram_desc(format!("RAM {}/{}gb ({}%)", r1(s.ram.used_gb), r1(s.ram.total_gb), r1(s.ram.used_pct)).into());
    ui.set_ram_pct(s.ram.used_pct as f32);

    let hot = s.gpu.hotspot_c.round();
    ui.set_gpu_util(s.gpu.util_pct as f32);
    ui.set_gpu_title(format!("GPU ({}W)", s.gpu.power_w.round()).into());
    ui.set_gpu_hot(format!("{hot}").into());
    ui.set_vram_temp(format!("VRAM {}°C", s.gpu.vram_c.round()).into());
    ui.set_gpu_temp_val(hot as f32);
    ui.set_gpu_temp_color(theme::temp_color(hot, 110.0, 45.0));
    ui.set_gpu_desc(
        format!("VRAM {}/{}GB ({}%)", r1(s.gpu.vram_used_gb), r1(s.gpu.vram_total_gb), s.gpu.vram_used_pct.round()).into(),
    );
    ui.set_vram_pct(s.gpu.vram_used_pct as f32);
}
