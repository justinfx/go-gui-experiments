package main

import (
	"embed"
	"log"
	"net/http"

	"gguix/core"
	"gguix/core/backend"
	"gguix/core/memory"
	"gguix/core/prefs"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Wails embeds the built frontend into the binary.
//
//go:embed all:frontend/dist
var assets embed.FS

// emitEvent sends an event to every frontend (the window, or all browsers in
// server mode).
func emitEvent(name string, data any) {
	app := application.Get()
	if app == nil {
		return
	}
	if data == nil {
		app.Event.Emit(name)
		return
	}
	app.Event.Emit(name, data)
}

// openBackend returns the core services. Bench mode ignores the GGUIX_ROWS
// style overrides and always uses the default 1,000,000 row dataset.
func openBackend(bench bool) (core.DataService, core.PrefsStore, error) {
	if bench {
		return memory.New(memory.DefaultOptions()), prefs.Default(appID), nil
	}
	return backend.Open(appID)
}

func main() {
	bench := core.BenchEnabled()
	ds, ps, err := openBackend(bench)
	if err != nil {
		log.Fatal(err)
	}
	svc := newItemService(ds, ps, emitEvent)
	svc.bench = bench

	opts := application.Options{
		Name:        "GGUIX - Wails",
		Description: "GGUIX reference application",
		Services:    append([]application.Service{application.NewService(svc)}, targetServices()...),
		Assets: application.AssetOptions{
			Handler:    application.AssetFileServerFS(assets),
			Middleware: assetMiddleware(svc),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	}
	configureTarget(&opts, svc)
	app := application.New(opts)
	setupTarget(app, svc)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

var _ = http.StatusOK
