package main

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"

	gc "gguix/core"
	"gguix/core/backend"
	"gguix/core/memory"
	"gguix/core/prefs"
)

func openBackend(bench bool) (gc.DataService, gc.PrefsStore) {
	if bench && backend.Mode == "memory" {
		// bench mode always uses the default dataset
		return memory.New(memory.DefaultOptions()), prefs.Default(appID)
	}
	svc, pst, err := backend.Open(appID)
	if err != nil {
		panic(err)
	}
	return svc, pst
}

func main() {
	bench := gc.BenchEnabled()
	svc, pst := openBackend(bench)
	a := newApp(svc, pst, bench)
	b := core.NewBody(windowTitle)
	a.buildUI(b)
	platformInit(a, b)
	b.OnShow(func(e events.Event) {
		a.start()
		a.startBench()
	})
	b.RunMainWindow()
}
