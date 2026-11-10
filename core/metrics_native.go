//go:build !js

package core

import "time"

// SinceStart returns the time since ProcessStart, the start used for the
// bench "ready" metric in native builds.
func SinceStart() time.Duration { return time.Since(ProcessStart) }
