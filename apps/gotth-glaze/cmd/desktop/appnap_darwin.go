//go:build darwin

package main

import (
	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// disableAppNap keeps macOS from throttling the webview timers when the
// window is occluded or the screen is locked. It calls
// -[NSProcessInfo beginActivityWithOptions:reason:] through purego (no cgo).
func disableAppNap() {
	if _, err := purego.Dlopen("/System/Library/Frameworks/Foundation.framework/Foundation", purego.RTLD_GLOBAL|purego.RTLD_NOW); err != nil {
		return
	}
	const userInitiated = 0x00FFFFFF | (1 << 20) // NSActivityUserInitiated
	pi := objc.ID(objc.GetClass("NSProcessInfo")).Send(objc.RegisterName("processInfo"))
	reason := objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("stringWithUTF8String:"), "GGUIX timers")
	act := pi.Send(objc.RegisterName("beginActivityWithOptions:reason:"), uint64(userInitiated), reason)
	if act != 0 {
		act.Send(objc.RegisterName("retain"))
	}
}
