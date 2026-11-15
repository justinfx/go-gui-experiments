//go:build !js

package main

import (
	"context"
	"log"
	"runtime"

	"github.com/gen2brain/iup-go/iup"

	"gguix/core"
	corebackend "gguix/core/backend"
	"gguix/core/bench"
	"gguix/core/host"
)

// IUP must run on the process's main thread (Cocoa requires it).
func init() { runtime.LockOSThread() }

func main() {
	svc, store, err := corebackend.Open(appID)
	if err != nil {
		log.Fatal(err)
	}
	iup.Open()
	defer iup.Close()
	openControls()

	a := newApp(svc, store, host.Current(appID), core.BenchEnabled())
	a.plat = desktopHooks(a)
	dlg := a.buildUI()
	ww, wh := a.prefs.WindowWidth, a.prefs.WindowHeight
	if ww < 1024 {
		ww = 1280
	}
	if wh < 640 {
		wh = 800
	}
	set(dlg, "RASTERSIZE", sizeStr(ww, wh))
	iup.Map(dlg)
	a.uiInit()
	iup.ShowXY(dlg, iup.CENTER, iup.CENTER)
	// A radio loses its initial value when the window is first shown.
	a.uiViewControls()
	a.restoreSelection()
	if a.bench {
		go bench.Run(context.Background(), svc, newDriver(a))
	}
	iup.MainLoop()
	a.flushPrefs()
	a.pager.Close()
}
