//! DaisyUI theme tokens (approximate sRGB of the v5 oklch values) and the temperature colour ramp.

use slint::Color;

pub struct Palette {
    pub primary: u32,
    pub secondary: u32,
    pub base_200: u32,
    pub base_content: u32,
    pub success: u32,
}

const fn p(primary: u32, secondary: u32, base_200: u32, base_content: u32, success: u32) -> Palette {
    Palette { primary, secondary, base_200, base_content, success }
}

/// Unknown themes fall back to `lofi`, like `normalizeTheme()` in panel.js.
pub fn palette(theme: &str) -> Palette {
    match theme.to_ascii_lowercase().as_str() {
        "light" => p(0x605dff, 0xf43098, 0xf8f8f8, 0x18181b, 0x00d390),
        "dark" => p(0x605dff, 0xf43098, 0x191e24, 0xecf9ff, 0x00d390),
        "nord" => p(0x5e81ac, 0x81a1c1, 0xe5e9f0, 0x2e3440, 0xa3be8c),
        "dracula" => p(0xff79c6, 0xbd93f9, 0x21222c, 0xf8f8f2, 0x50fa7b),
        "winter" => p(0x0069ff, 0x463aa2, 0xf2f7ff, 0x394e6a, 0x2ac3a2),
        "dim" => p(0x9fe88d, 0xff7d5c, 0x242933, 0xb2ccd6, 0x62efbd),
        "synthwave" => p(0xf861b4, 0x71d1fe, 0x140c30, 0xf9f7fd, 0x1be885),
        "business" => p(0x1c4e80, 0x7c909a, 0x1c1c1c, 0xcdcdcd, 0x6bb187),
        "night" => p(0x38bdf8, 0x818cf8, 0x0c1425, 0xb3c5ef, 0x2dd4bf),
        "black" => p(0x373737, 0x373737, 0x141414, 0xd6d6d6, 0x008000),
        "corporate" => p(0x0082ce, 0x61738c, 0xf2f3f4, 0x181a2a, 0x00a96e),
        _ => p(0x0d0d0d, 0x1a1919, 0xf2f2f2, 0x000000, 0x00c38a),
    }
}

pub fn rgb(hex: u32) -> Color {
    Color::from_rgb_u8((hex >> 16) as u8, (hex >> 8) as u8, hex as u8)
}

/// Port of `tempColorForPct()`: blue (cool) → red (hot) as `hsl(hue, 85%, 55%)`.
pub fn temp_color(temp: f64, max: f64, min: f64) -> Color {
    let t = (temp.clamp(min, max) - min) / (max - min);
    hsl(220.0 - t * 220.0, 0.85, 0.55)
}

fn hsl(h: f64, s: f64, l: f64) -> Color {
    let c = (1.0 - (2.0 * l - 1.0).abs()) * s;
    let hp = (h / 60.0) % 6.0;
    let x = c * (1.0 - (hp % 2.0 - 1.0).abs());
    let (r, g, b) = match hp as u32 {
        0 => (c, x, 0.0),
        1 => (x, c, 0.0),
        2 => (0.0, c, x),
        3 => (0.0, x, c),
        4 => (x, 0.0, c),
        _ => (c, 0.0, x),
    };
    let m = l - c / 2.0;
    let u = |v: f64| ((v + m) * 255.0).round() as u8;
    Color::from_rgb_u8(u(r), u(g), u(b))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn temp_ramp_endpoints() {
        // hsl(220,85%,55%) ≈ #3a6ff1-ish blue; hsl(0,85%,55%) ≈ #f13a3a red.
        let cold = temp_color(10.0, 95.0, 35.0);
        let hot = temp_color(120.0, 95.0, 35.0);
        assert!(cold.blue() > 200 && cold.red() < 80);
        assert!(hot.red() > 200 && hot.blue() < 80);
    }
}
