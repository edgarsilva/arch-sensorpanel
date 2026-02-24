package migrations

import "embed"

// FS exposes embedded goose migrations.
//
//go:embed *.sql
var FS embed.FS
