//go:build js

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
	a.plat = webHooks()
	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{Title: "GGUIX - MyGo", Content: ui.View(a.view)})
		a.post = win.Update
		a.wake = win.Invalidate
		a.restoreSelection()
		if a.bench {
			go bench.Run(context.Background(), svc, newDriver(a))
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
