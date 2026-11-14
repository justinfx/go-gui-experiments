//go:build !e2e

package main

// Desktop entry points (category: desktop): the window, the native menu and
// the quit sequence.

import (
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
)

// quitting is set once the frontend has flushed its preferences.
var quitting atomic.Bool

func setupTarget(app *mygo.Application, svc *ItemService) {
	svc.plat = &desktopPlatform{}

	// Ask the frontend to save its pending preferences first, then quit.
	app.OnBeforeQuit(func(e *mygo.QuitEvent) {
		if quitting.Load() || e.SessionEnding {
			return
		}
		e.PreventDefault()
		emitEvent("app:flush", nil)
		go func() {
			select {
			case <-svc.flushed:
			case <-time.After(400 * time.Millisecond):
			}
			quitting.Store(true)
			app.Quit()
		}()
	})
	app.OnQuit(func() { svc.CancelExport() })
	// Quit when the last window closes, on macOS too (the Wails app did).
	app.OnWindowAllClosed(func() { app.Quit() })

	app.WhenReady(func() {
		p, _ := svc.store.Load()
		w, h := p.WindowWidth, p.WindowHeight
		if w <= 0 {
			w = 1280
		}
		if h <= 0 {
			h = 800
		}
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:     "GGUIX - MyGo",
			Width:     w,
			Height:    h,
			MinWidth:  1024,
			MinHeight: 640,
			StateKey:  "main",
			URL:       "/",
		})
		svc.plat.(*desktopPlatform).window = win

		var timer *time.Timer
		win.OnResize(func() {
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(400*time.Millisecond, func() {
				cw, ch := win.Size()
				svc.noteWindowSize(cw, ch)
			})
		})

		app.SetMenu(buildMenu(svc))
		svc.onMenuState = menuStateUpdater()
	})
}
