//go:build js

package main

import (
	"log"

	"github.com/nsgo-dev/nsgo"

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
	nsgo.Run(rootView(a))
}
