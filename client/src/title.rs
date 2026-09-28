//! Strip YouTube wallpaper-video boilerplate ("1 Hour Loop", "4K Resolution", "Live Wallpaper",
//! hashtags, …) from titles so the panel shows just the name of the video.

use regex_lite::Regex;
use std::sync::LazyLock;

static NOISE: LazyLock<Vec<Regex>> = LazyLock::new(|| {
    [
        r"#\S+",
        r"\(download link\)",
        r"now has \d+ views!?",
        r"\b\d+\s*hours?(\s*(loop|video))?\b",
        r"\[?\b[48]k\b\]?(\s*(resolution|ultra\s*hd))?",
        r"\bultra\s*hd\b\.?",
        r"\b\d+\s*fps\b",
        r"\byour live wallpaper for pc\b",
        r"&?\s*\bscreensa[vw]er\b",
        r"\b(silent\s+)?live wallpapers?\b",
        r"\b(anime\s+)?wallpapers?\b",
        r"\b(loop\s+)?background\b",
        r"\b(for\s+)?desktop\b",
        r"\bno sound\b",
        r"\bno copyright video\b",
    ]
    .iter()
    .map(|p| Regex::new(&format!("(?i){p}")).expect("valid title pattern"))
    .collect()
});

static SEPARATORS: LazyLock<Regex> = LazyLock::new(|| Regex::new(r"\s+[|\-–—]\s+|\s*\|\s*").unwrap());
static SPACES: LazyLock<Regex> = LazyLock::new(|| Regex::new(r"\s{2,}").unwrap());

fn clean_segment(segment: &str) -> String {
    let mut s = segment.to_string();
    for re in NOISE.iter() {
        s = re.replace_all(&s, " ").into_owned();
    }
    let s = SPACES.replace_all(&s, " ");
    s.trim_matches(|c: char| c.is_whitespace() || "&,-–—|:.".contains(c)).to_string()
}

/// First `|`/` - ` separated part of the title that still has content once cleaned;
/// falls back to the raw title if cleaning leaves nothing.
pub fn clean(raw: &str) -> String {
    SEPARATORS
        .split(raw)
        .map(clean_segment)
        .find(|s| !s.is_empty())
        .unwrap_or_else(|| raw.trim().to_string())
}

#[cfg(test)]
mod tests {
    use super::clean;

    #[test]
    fn strips_wallpaper_boilerplate() {
        let cases = [
            ("1 Hour Loop Girl On The Car Window 4K Resolution Your Live Wallpaper For PC #livewallpaper", "Girl On The Car Window"),
            ("1 Hour loop 4k Gray haired Girl In Dark Leaves Live Wallpaper & Screensaver #anime #livewallpaper", "Gray haired Girl In Dark Leaves"),
            ("Convenience Store In The Rain Live Wallpaper 4K | Cozy Rainy Night Desktop Background", "Convenience Store In The Rain"),
            ("Girl Behind Curtains | [4K] | Anime Mystery Live Wallpaper now has 1124155 views!", "Girl Behind Curtains"),
            ("Gaze of the Blade 4K Live Wallpaper | Crystal Eye Girl | 1 Hour Loop (Anime Aesthetic)", "Gaze of the Blade"),
            ("4K Live Wallpaper | BMW F30 Drift Animation for Desktop", "BMW F30 Drift Animation"),
            ("City Skyline Screensaver Wallpaper - 12 Hours - 4K Ultra HD. No Sound", "City Skyline"),
            ("The Drive - 12 Hours - 4K Ultra HD 60fps", "The Drive"),
            ("Loop Background | Live Wallpaper | Chilling Cat | No Sound", "Chilling Cat"),
            ("Samurai TV Screensaver | Silent Live Wallpaper 1 Hour | 4K Ambient Loop", "Samurai TV"),
            ("2 Hours Loop Lone Samurai Ronin 4K Resolution Your Live Wallpaper For PC", "Lone Samurai Ronin"),
            ("4K Live Wallpaper GOKU Falling Stars Dragon Ball Background (DOWNLOAD LINK)", "GOKU Falling Stars Dragon Ball"),
            ("Esdeath akame ga kill 4K anime wallpaper", "Esdeath akame ga kill"),
            ("ANIME GIRL - BEST PURPLE WALLPAPER - 1 HOUR VIDEO", "ANIME GIRL"),
            ("the chillest lofi comes from tomorrow", "the chillest lofi comes from tomorrow"),
            ("Superman and Krypto Live Wallpaper", "Superman and Krypto"),
        ];
        for (raw, want) in cases {
            assert_eq!(clean(raw), want, "raw: {raw}");
        }
    }

    #[test]
    fn falls_back_to_raw() {
        assert_eq!(clean("4K Live Wallpaper"), "4K Live Wallpaper");
        assert_eq!(clean(""), "");
    }
}
