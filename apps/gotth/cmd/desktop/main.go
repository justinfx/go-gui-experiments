// Command desktop runs the GoTTH server in this process on a random free
// 127.0.0.1 port and shows it in a webview window.
package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"

	webview "github.com/webview/webview_go"

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
	w := webview.New(false)
	defer w.Destroy()
	nativeMenu := installNativeMenu(w)
	quit := func() { w.Dispatch(func() { w.Terminate() }) }
	srv, err := gotth.New(gotth.Options{
		Svc:         svc,
		Prefs:       store,
		Native:      platformNative(),
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
	w.SetSize(p.WindowWidth, p.WindowHeight, webview.HintNone)
	w.SetSize(1024, 640, webview.HintMin)
	w.Navigate(fmt.Sprintf("http://%s/", ln.Addr()))
	w.Run()
	_ = srv.Flush()
}
