// Command server runs the GGUIX HTTP API (see gguix/core/server) over the
// in-memory dataset, optionally serving a built web app from -static.
//
//	cd core && go run ./cmd/server -addr :8090 -static ../apps/<id>/bin/web
//	go run gguix/core/cmd/server ...          (from an app module)
//	go run -tags cgoprobe ./cmd/server ...    (cgo-backed core)
package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"gguix/core"
	"gguix/core/memory"
	"gguix/core/server"
)

func main() {
	def := memory.DefaultOptions()
	addr := flag.String("addr", ":8090", "listen address")
	static := flag.String("static", "", "directory of static files to serve (index.html fallback)")
	rows := flag.Int("rows", def.Rows, "number of generated rows")
	latency := flag.Int("latency-ms", int(def.Latency/time.Millisecond), "latency added per items request, in ms")
	rps := flag.Int("export-rps", def.ExportRowsPerSec, "export rows per second (negative = unthrottled)")
	flag.String("app", "", "app ID (informational; prefs are keyed by the request path)")
	flag.Parse()

	svc := memory.New(memory.Options{Rows: *rows, Latency: time.Duration(*latency) * time.Millisecond, ExportRowsPerSec: *rps, Seed: def.Seed})
	log.Printf("gguix server on %s (rows=%d latency=%dms cgoprobe=%v static=%q)", *addr, svc.Rows(), *latency, core.CgoProbe, *static)
	log.Fatal(http.ListenAndServe(*addr, server.Handler(svc, *static)))
}
