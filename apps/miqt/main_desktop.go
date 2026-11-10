//go:build !js

package main

import (
	"fmt"
	"os"

	"gguix/core"
	"gguix/core/backend"
	"gguix/core/bench"

	qt "github.com/mappu/miqt/qt6"
)

func main() {
	qt.NewQApplication(os.Args)
	qt.QCoreApplication_SetApplicationName("GGUIX")

	svc, store, err := backend.Open(appID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "backend:", err)
		os.Exit(1)
	}
	a := newApp(options{svc: svc, store: store, bench: core.BenchEnabled()})
	a.win.Show()
	a.win.Raise()
	a.start()
	if a.bench {
		// gguix:target bench
		go bench.Run(a.ctx, svc, benchDriver{a})
		// gguix:end
	}
	os.Exit(qt.QApplication_Exec())
}
