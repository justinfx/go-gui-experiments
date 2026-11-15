//go:build !js

package main

import (
	"context"
	"log"

	"github.com/nsgo-dev/nsgo"
	"github.com/nsgo-dev/nsgo/core"

	gx "gguix/core"
	corebackend "gguix/core/backend"
	"gguix/core/host"
)

func main() {
	svc, store, err := corebackend.Open(appID)
	if err != nil {
		log.Fatal(err)
	}
	a := newApp(svc, store, host.Current(appID), gx.BenchEnabled())
	ww, wh := a.prefs.WindowWidth, a.prefs.WindowHeight
	if ww < 1024 {
		ww = 1280
	}
	if wh < 640 {
		wh = 800
	}
	nsgo.Run(rootView(a),
		nsgo.WindowSize(core.Points(ww), core.Points(wh)),
		nsgo.WindowConstraints(core.WindowConstraints{MinWidth: 1024, MinHeight: 640}),
	)
	a.flushPrefs()
	_ = context.Background
}
