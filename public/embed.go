package public

import "embed"

// FS exposes embedded web assets (HTML, JS, CSS, images).
//
//go:embed *
var FS embed.FS
