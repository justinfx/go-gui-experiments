// Command web runs the GoTTH server-mode web target on http://localhost:8090.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gguix/apps/gotth"
	"gguix/core/backend"
)

func main() {
	svc, store, err := backend.Open(gotth.AppID)
	if err != nil {
		log.Fatal(err)
	}
	srv, err := gotth.New(gotth.Options{Svc: svc, Prefs: store})
	if err != nil {
		log.Fatal(err)
	}
	ln, url := listen()
	// The CSV download and the SSE progress stream are not compressed (B2).
	noGzip := func(r *http.Request) bool { return r.URL.Path == "/export" || strings.HasPrefix(r.URL.Path, "/export/") }
	hs := &http.Server{Handler: gzipMiddleware(srv.Handler(), noGzip)}
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		_ = srv.Flush()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = hs.Shutdown(ctx)
	}()
	fmt.Printf("GoTTH web target listening on %s\n", url)
	if err := hs.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

// listen binds port 8090 (or $GOTTH_PORT, else $PORT) on the loopback
// interface. When the IPv4 loopback port is taken by another program it tries
// the IPv6 loopback, which http://localhost:8090 also resolves to, and then
// the next ports. A port set explicitly is never swapped for another one: the
// harness must not bench a stale listener on the port it asked for (C3).
func listen() (net.Listener, string) {
	port, explicit := 8090, false
	for _, name := range []string{"GOTTH_PORT", "PORT"} {
		if p, err := strconv.Atoi(os.Getenv(name)); err == nil && p > 0 {
			port, explicit = p, true
			break
		}
	}
	tries := 10
	if explicit {
		tries = 1
	}
	for try := 0; try < tries; try++ {
		for _, host := range []string{"127.0.0.1", "[::1]"} {
			ln, err := net.Listen("tcp", host+":"+strconv.Itoa(port+try))
			if err == nil {
				if try > 0 {
					fmt.Fprintf(os.Stderr, "WARNING: port %d is busy, using %d\n", port, port+try)
				} else if host != "127.0.0.1" {
					fmt.Fprintf(os.Stderr, "NOTE: 127.0.0.1:%d is busy, listening on %s:%d (http://localhost:%d still works)\n", port, host, port, port)
				}
				return ln, "http://localhost:" + strconv.Itoa(port+try)
			}
		}
	}
	if explicit {
		log.Fatalf("port %d is busy", port)
	}
	log.Fatal("no free port found")
	return nil, ""
}
