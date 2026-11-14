//go:build js

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
	w := gui.NewWindow(gui.WindowCfg{
		State: a, Title: "GGUIX - go-gui", Width: 1280, Height: 800,
		OnInit: func(w *gui.Window) {
			a.bindWindow(w)
			a.plat = webHooks()
			w.SetView(mainView)
			a.restoreSelection()
			if a.bench {
				go bench.Run(context.Background(), svc, newDriver(a))
			}
		},
	})
	backend.Run(w)
}
