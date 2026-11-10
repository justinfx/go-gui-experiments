//go:build darwin

package main

/*
#cgo darwin LDFLAGS: -framework Foundation
void gguixDisableAppNap(void);
*/
import "C"

// disableAppNap keeps macOS from throttling the webview timers when the
// window is occluded or the screen is locked.
func disableAppNap() { C.gguixDisableAppNap() }
