package web

import "embed"

// Files holds the compiled frontend assets built by `pnpm run build`.
//
//go:embed all:dist
var Files embed.FS
