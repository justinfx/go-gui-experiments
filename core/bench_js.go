//go:build js

package core

import (
	"net/url"
	"strings"
	"syscall/js"
)

// BenchEnabled reports bench mode: env GGUIX_BENCH=1 (native) or ?bench=1 in
// the page URL (js/wasm).
func BenchEnabled() bool {
	loc := js.Global().Get("location")
	if loc.IsUndefined() || loc.IsNull() {
		return false
	}
	v, err := url.ParseQuery(strings.TrimPrefix(loc.Get("search").String(), "?"))
	return err == nil && v.Get("bench") == "1"
}
