package gotth

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func execLookPath(name string) (string, error) { return exec.LookPath(name) }

func execCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	done := time.AfterFunc(90*time.Second, func() { _ = cmd.Process.Kill() })
	defer done.Stop()
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// TestServeDesktopMode serves the desktop-mode page (in-page menu bar, native
// features backed by a fake) on GOTTH_SERVE_DESKTOP=<addr> for manual checks in
// a browser. It is skipped unless the variable is set.
func TestServeDesktopMode(t *testing.T) {
	addr := os.Getenv("GOTTH_SERVE_DESKTOP")
	if addr == "" {
		t.Skip("set GOTTH_SERVE_DESKTOP=127.0.0.1:8095 to serve")
	}
	e := newEnv(t, 0, &fakeNative{savePath: filepath.Join(t.TempDir(), "x.csv")})
	srv := &http.Server{Addr: addr, Handler: e.srv.Handler()}
	go func() { time.Sleep(10 * time.Minute); _ = srv.Close() }()
	_ = srv.ListenAndServe()
}
