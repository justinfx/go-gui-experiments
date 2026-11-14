//go:build !js

package main

import (
	"context"
	"log"

	"github.com/go-gui-org/go-gui/gui"
	"github.com/go-gui-org/go-gui/gui/backend"

	"gguix/core"
	corebackend "gguix/core/backend"
	"gguix/core/bench"
	"gguix/core/host"
)

func main() {
	svc, store, err := corebackend.Open(appID)
	if err != nil {
		log.Fatal(err)
	}
	a := newApp(svc, store, host.Current(appID), core.BenchEnabled())
	ww, wh := a.prefs.WindowWidth, a.prefs.WindowHeight
	if ww < 1024 {
		ww = 1280
	}
	if wh < 640 {
		wh = 800
	}
	guiApp = gui.NewApp()
	w := gui.NewWindow(gui.WindowCfg{
		State:     a,
		Title:     "GGUIX - go-gui",
		Width:     ww,
		Height:    wh,
		MinWidth:  1024,
		MinHeight: 640,
		OnInit: func(w *gui.Window) {
			a.bindWindow(w)
			a.plat = desktopHooks(w, guiApp)
			w.SetView(mainView)
			a.restoreSelection()
			if a.bench {
				go bench.Run(context.Background(), svc, newDriver(a))
			}
		},
		OnEvent: func(e *gui.Event, w *gui.Window) {
			if e.Type == gui.EventResized && e.WindowWidth > 0 && e.WindowHeight > 0 {
				a.setWindowSize(e.WindowWidth, e.WindowHeight)
			}
		},
		OnCloseRequest: func(w *gui.Window) {
			a.flushPrefs()
			w.Close()
		},
	})
	backend.RunApp(guiApp, w)
	a.flushPrefs()
}
