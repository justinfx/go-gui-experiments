package main

import (
	"context"
	"log"

	"fyne.io/fyne/v2/app"

	"gguix/core"
	"gguix/core/backend"
)

func main() {
	svc, store, err := backend.Open(appID)
	if err != nil {
		log.Fatal(err)
	}
	a := app.NewWithID("io.gguix.fyne")
	w := a.NewWindow("GGUIX - Fyne")
	w.SetMaster()
	u := newUI(a, w, deps{svc: svc, store: store, bench: core.BenchEnabled()})
	u.start()
	if u.bench {
		a.Lifecycle().SetOnStarted(func() {
			go startBench(context.Background(), u)
		})
	}
	w.ShowAndRun()
}
