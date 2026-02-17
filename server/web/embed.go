package web

import "embed"

// Dist contains the embedded admin UI static files.
// Built from admin-ui/ via Vite, output to server/web/dist/.
//
//go:embed all:dist
var Dist embed.FS
