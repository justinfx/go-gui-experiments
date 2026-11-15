//go:build js

package main

import (
	"context"
	"log"

	"github.com/gen2brain/iup-go/iup"

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
	iup.Open()
	defer iup.Close()
	a := newApp(svc, store, host.Current(appID), core.BenchEnabled())
	a.plat = webHooks(a)
	dlg := a.buildUI()
	iup.Map(dlg)
	a.uiInit()
	iup.ShowXY(dlg, iup.CENTER, iup.CENTER)
	a.uiViewControls()
	a.restoreSelection()
	if a.bench {
		go bench.Run(context.Background(), svc, newDriver(a))
	}
	iup.MainLoop()
	a.flushPrefs()
}
