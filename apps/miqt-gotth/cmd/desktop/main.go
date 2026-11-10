// Command desktop runs the GoTTH server in this process on a random free
// 127.0.0.1 port and shows it in a Qt 6 window: a QMainWindow hosting a
// QWebEngineView (MIQT bindings), with a native QMenuBar, QFileDialog,
// QSystemTrayIcon notification and QClipboard. See window.go and
// native.go.
package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"

	"gguix/apps/gotth"
	"gguix/core/backend"
)

func init() { runtime.LockOSThread() }

func main() {
	disableAppNap()
	// Qt WebEngine needs a shared GL context; this must precede QApplication.
	qt.QCoreApplication_SetAttribute(qt.AA_ShareOpenGLContexts)
	qt.NewQApplication(os.Args)
	qt.QCoreApplication_SetApplicationName("GGUIX")

	svc, store, err := backend.Open(gotth.AppID)
	if err != nil {
		log.Fatal(err)
	}
	nat := newQtNative()
	srv, err := gotth.New(gotth.Options{
		Svc:         svc,
		Prefs:       store,
		Native:      nat,
		NativeMenu:  true,
		Bench:       os.Getenv("GGUIX_BENCH") == "1",
		BenchExport: os.Getenv("GGUIX_BENCH_EXPORT"),
		Quit:        func() { mainthread.Start(qt.QCoreApplication_Quit) },
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
	win := newWindow(p.WindowWidth, p.WindowHeight)
	nat.win = win
	win.Show()
	win.Raise()
	win.view.SetUrl(qt.NewQUrl3(fmt.Sprintf("http://%s/", ln.Addr())))
	rc := qt.QApplication_Exec()
	_ = srv.Flush()
	os.Exit(rc)
}
