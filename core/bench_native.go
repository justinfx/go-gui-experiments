//go:build !js

package core

import "os"

// BenchEnabled reports bench mode: env GGUIX_BENCH=1 (native) or ?bench=1 in
// the page URL (js/wasm).
func BenchEnabled() bool { return os.Getenv("GGUIX_BENCH") == "1" }
