// Package static embeds the app's own static files (vendored htmx, app
// scripts, bench driver).
package static

import "embed"

//go:embed all:*
var FS embed.FS
