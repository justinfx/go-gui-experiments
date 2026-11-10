//go:build js

package core

import (
	"syscall/js"
	"time"
)

// SinceStart returns the time since the start used for the bench "ready"
// metric. In the browser that is navigation start (performance.timeOrigin),
// so wasm apps measure "ready" the same way as the JS apps, which use
// performance.now() (harness audit B3). Outside a page it falls back to
// ProcessStart.
func SinceStart() time.Duration {
	perf := js.Global().Get("performance")
	if perf.IsUndefined() || perf.IsNull() {
		return time.Since(ProcessStart)
	}
	return time.Duration(perf.Call("now").Float() * float64(time.Millisecond))
}
