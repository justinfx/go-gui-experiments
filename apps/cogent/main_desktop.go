//go:build !js

package main

import (
	"image"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/system"
	"cogentcore.org/core/system/driver/desktop"
	"github.com/go-gl/glfw/v3.4/glfw"
)

const (
	minWinW = 1024
	minWinH = 640
)

// applyMinSize sets the glfw minimum window size on the desktop window.
// Returns false when the system window is not a glfw desktop window.
func applyMinSize(sw system.Window) bool {
	dw, ok := sw.(*desktop.Window)
	if !ok || dw.Glw == nil {
		return false
	}
	dw.Glw.SetSizeLimits(minWinW, minWinH, glfw.DontCare, glfw.DontCare)
	return true
}

// platformInit restores and saves the window size, and saves prefs on quit.
func platformInit(a *App, b *core.Body) {
	system.TheApp.AddQuitCleanFunc(func() { a.saveOnQuit() })
	b.OnShow(func(e events.Event) {
		w, h := a.prefs.WindowWidth, a.prefs.WindowHeight
		if w < 1024 {
			w = 1280
		}
		if h < 640 {
			h = 800
		}
		if rw := b.Scene.RenderWindow(); rw != nil {
			applyMinSize(rw.SystemWindow)
			rw.SystemWindow.SetWinSize(image.Pt(w, h))
		}
	})
	b.OnFinal(events.WindowResize, func(e events.Event) {
		if rw := b.Scene.RenderWindow(); rw != nil {
			sz := rw.SystemWindow.WinSize()
			if sz.X > 0 && sz.Y > 0 && (sz.X != a.prefs.WindowWidth || sz.Y != a.prefs.WindowHeight) {
				a.prefs.WindowWidth, a.prefs.WindowHeight = sz.X, sz.Y
				a.savePrefsSoon()
			}
		}
	})
}
