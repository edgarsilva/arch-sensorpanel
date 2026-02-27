# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project follows Semantic Versioning.

## [0.1.0] - 2026-02-27

### Added
- Initial public release of Sensor Panel HUD.
- Versioned settings UI for media, overlay, and metrics tuning.
- Live metrics overlay with REST and WebSocket updates.
- Hyprland-focused setup documentation, launcher script examples, and workspace rules.
- Hardware context and display compatibility notes for small HDMI in-case screens.

### Changed
- Metrics scale and offsets now use slider controls in the settings UI.
- Metrics offset tuning range expanded to `-1000..1000` px in UI, runtime clamping, and server validation.
- `/settings` now preloads current settings when available; defaults remain for first-run draft flows.
- Settings page now highlights the current history entry and shows a context badge for editing state.
