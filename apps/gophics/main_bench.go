package main

import (
	"context"

	"gguix/apps/gophics/ui"
	"gguix/core"
	"gguix/core/bench"
)

// benchHook starts the core bench script once the window has built.
func benchHook(opts *ui.Options, svc core.DataService) {
	opts.Ready = func(a *ui.App) {
		go bench.Run(context.Background(), svc, ui.NewBenchDriver(a))
	}
}
