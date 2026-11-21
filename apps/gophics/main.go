// Command gophics is the GGUIX reference app on Gophics. The same main runs
// on desktop (native window) and in the browser (js/wasm, WebGPU); the shell
// is chosen by build tags inside the framework.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/doug/gophics/app"

	"gguix/apps/gophics/ui"
	"gguix/core"
	"gguix/core/backend"
	"gguix/core/host"
)

func main() {
	ui.BuildMarker = BuildMarker
	benchMode := core.BenchEnabled()
	if benchMode {
		// Bench uses the default dataset regardless of env overrides.
		os.Unsetenv("GGUIX_ROWS")
		os.Unsetenv("GGUIX_LATENCY_MS")
		os.Unsetenv("GGUIX_EXPORT_RPS")
	}
	svc, store, err := backend.Open(ui.AppID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "backend:", err)
		os.Exit(1)
	}
	p, err := store.Load()
	if err != nil {
		p = core.DefaultPrefs()
	}
	w, h := p.WindowWidth, p.WindowHeight
	if w <= 0 || h <= 0 {
		w, h = 1280, 800
	}
	w, h = max(w, 1024), max(h, 640)
	opts := &ui.Options{Svc: svc, Store: store, Host: host.Current(ui.AppID), Bench: benchMode, Prefs: p}
	if benchMode {
		benchHook(opts, svc)
	}
	err = app.Run(ui.Root{Opts: opts}, ui.Config(w, h))
	if a := opts.App; a != nil {
		a.SavePrefsSync()
	}
	if err != nil {
		log.Fatal(err)
	}
}
