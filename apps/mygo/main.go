//go:build !e2e

package main

import (
	"log"

	"gguix/core"

	"github.com/egoist/mygo"
)

// emitEvent sends an event to every window. MyGo events are typed, so the
// name is mapped to its declaration (service.go).
func emitEvent(name string, data any) {
	switch name {
	case "export:progress":
		_ = evExportProgress.Broadcast(data.(ExportProgress))
	case "export:done":
		_ = evExportDone.Broadcast(data.(ExportDone))
	case "export:error":
		_ = evExportError.Broadcast(data.(ExportError))
	case "export:cancelled":
		_ = evExportCancelled.Broadcast(struct{}{})
	case "menu:action":
		_ = evMenuAction.Broadcast(data.(MenuAction))
	case "app:flush":
		_ = evAppFlush.Broadcast(struct{}{})
	}
}

func main() {
	bench := core.BenchEnabled()
	ds, ps, err := openBackend(bench)
	if err != nil {
		log.Fatal(err)
	}
	svc := newItemService(ds, ps, emitEvent)
	svc.bench = bench

	// The built frontend is embedded by `mygo build` (it calls
	// mygo.SetFrontend through a generated file); `mygo dev` serves it
	// from disk or from the Vite dev server.
	app := mygo.App
	// The name decides the user data directory and the menu bar title.
	app.SetName("GGUIX - MyGo")
	mygo.Bind(svc)
	setupTarget(app, svc)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
