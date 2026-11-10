//go:build cgoprobe && !cgo

package core

// This file only builds when the cgoprobe tag is set but cgo is unavailable
// for the target (for example GOOS=js GOARCH=wasm, or CGO_ENABLED=0). It
// fails the build on purpose with a descriptive undefined name.
var _ = cgoprobe_requires_cgo_but_cgo_is_unavailable_for_this_target
