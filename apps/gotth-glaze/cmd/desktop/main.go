// Command desktop runs the GoTTH server in this process on a random free
// 127.0.0.1 port and shows it in a glaze (pure-Go, no cgo) webview window.
package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"

	"github.com/crgimenes/glaze"

	"gguix/apps/gotth"
	"gguix/core/backend"
)

func init() { runtime.LockOSThread() }

func main() {
	disableAppNap()
	svc, store, err := backend.Open(gotth.AppID)
	if err != nil {
		log.Fatal(err)
	}
	w, err := glaze.New(false)
	if err != nil {
		log.Fatal(err)
	}
	defer w.Destroy()
	nativeMenu := installNativeMenu(w)
	quit := func() { w.Dispatch(func() { w.Terminate() }) }
	srv, err := gotth.New(gotth.Options{
		Svc:         svc,
		Prefs:       store,
		Native:      newNative(w),
		NativeMenu:  nativeMenu,
		Bench:       os.Getenv("GGUIX_BENCH") == "1",
		BenchExport: os.Getenv("GGUIX_BENCH_EXPORT"),
		Quit:        quit,
	})
	if err != nil {
		log.Fatal(err)
	}
	// Listen only on the loopback interface.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	go func() { _ = http.Serve(ln, srv.Handler()) }()

	p := srv.Prefs()
	w.SetTitle("GGUIX - GoTTH")
	w.SetSize(p.WindowWidth, p.WindowHeight, glaze.HintNone)
	w.SetSize(1024, 640, glaze.HintMin)
	w.Navigate(fmt.Sprintf("http://%s/", ln.Addr()))
	w.Run()
	_ = srv.Flush()
}
