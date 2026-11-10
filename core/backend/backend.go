// Package backend picks the DataService and PrefsStore for the build:
//
//   - default build: the in-process memory service (with environment
//     overrides GGUIX_ROWS, GGUIX_LATENCY_MS, GGUIX_EXPORT_RPS) and
//     prefs.Default (a file store natively, localStorage in js/wasm).
//   - build tag "remote": a remote.Client for both. In js/wasm the server
//     is the page origin; natively it is $GGUIX_REMOTE_URL (required).
package backend

import (
	"os"
	"strconv"
	"time"

	"gguix/core/memory"
)

// MemoryOptions returns memory.DefaultOptions with environment overrides
// applied: GGUIX_ROWS, GGUIX_LATENCY_MS, GGUIX_EXPORT_RPS.
func MemoryOptions() memory.Options {
	o := memory.DefaultOptions()
	if n, err := strconv.Atoi(os.Getenv("GGUIX_ROWS")); err == nil && n > 0 {
		o.Rows = n
	}
	if n, err := strconv.Atoi(os.Getenv("GGUIX_LATENCY_MS")); err == nil && n >= 0 {
		o.Latency = time.Duration(n) * time.Millisecond
	}
	if n, err := strconv.Atoi(os.Getenv("GGUIX_EXPORT_RPS")); err == nil && n != 0 {
		o.ExportRowsPerSec = n
	}
	return o
}
