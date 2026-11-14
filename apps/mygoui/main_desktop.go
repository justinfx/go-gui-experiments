//go:build !js

package main

import (
	"context"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"gguix/core"
	"gguix/core/backend"
	"gguix/core/bench"
	"gguix/core/host"
)

func main() {
	svc, store, err := backend.Open(appID)
	if err != nil {
		log.Fatal(err)
	}
	a := newApp(svc, store, host.Current(appID), core.BenchEnabled())
	var win *mygo.Window
	a.plat = desktopHooks(func() *mygo.Window { return win })

	mygo.App.WhenReady(func() {
		w, h := a.prefs.WindowWidth, a.prefs.WindowHeight
		if w < 1024 {
			w = 1280
		}
		if h < 640 {
			h = 800
		}
		win = mygo.NewWindow(mygo.WindowOptions{
			Title:     "GGUIX - MyGo",
			Width:     w,
			Height:    h,
			MinWidth:  1024,
			MinHeight: 640,
			Content:   ui.View(a.view),
		})
		a.post = win.Update
		a.wake = win.Invalidate
		a.restoreSelection()
		win.OnResize(func() {
			w, h := win.Size()
			a.setWindowSize(w, h)
		})
		mygo.App.SetMenu(a.buildMenu())
		mygo.App.OnBeforeQuit(func(e *mygo.QuitEvent) { a.flushPrefs() })
		if a.bench {
			go bench.Run(context.Background(), svc, newDriver(a))
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
