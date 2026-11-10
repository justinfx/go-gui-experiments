package main

import (
	"context"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"gguix/core"
	"gguix/core/memory"
	"gguix/core/prefs"
)

// countingSvc counts Update calls.
type countingSvc struct {
	core.DataService
	updates atomic.Int32
}

func (s *countingSvc) Update(ctx context.Context, it core.Item) (core.Item, error) {
	s.updates.Add(1)
	return s.DataService.Update(ctx, it)
}

// harness is one app instance driven through Fyne's test driver. All UI
// work the app schedules is queued and run on the test goroutine.
type harness struct {
	t    *testing.T
	u    *ui
	svc  *countingSvc
	mem  *memory.Service
	q    chan func()
	dir  string
	size fyne.Size
}

func newHarness(t *testing.T, opts memory.Options) *harness {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(prefs.EnvDir, dir)
	test.NewTempApp(t)
	mem := memory.New(opts)
	svc := &countingSvc{DataService: mem}
	w := test.NewWindow(nil)
	h := &harness{t: t, svc: svc, mem: mem, q: make(chan func(), 4096), dir: dir, size: fyne.NewSize(1280, 800)}
	h.u = newUI(fyne.CurrentApp(), w, deps{svc: svc, store: prefs.NewFileStore(appID)})
	h.u.runOnUI = func(f func()) { h.q <- f }
	h.u.start()
	w.Resize(h.size)
	h.pump()
	t.Cleanup(func() { h.u.pager.Close() })
	return h
}

// pump runs the UI work queued by background goroutines and timers.
func (h *harness) pump() {
	for {
		select {
		case f := <-h.q:
			f()
		default:
			return
		}
	}
}

// wait pumps the UI queue until cond is true.
func (h *harness) wait(what string, cond func() bool) {
	h.t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for {
		h.pump()
		if cond() {
			return
		}
		if time.Now().After(deadline) {
			h.t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (h *harness) loaded() { h.t.Helper(); h.wait("visible rows to load", h.u.VisibleLoaded) }

func (h *harness) canvas() fyne.Canvas { return h.u.win.Canvas() }

// walk lists o and everything below it using the live (cached) renderers,
// skipping hidden objects.
func walk(o fyne.CanvasObject) []fyne.CanvasObject {
	if o == nil || !o.Visible() {
		return nil
	}
	out := []fyne.CanvasObject{o}
	switch c := o.(type) {
	case fyne.Widget:
		for _, ch := range test.WidgetRenderer(c).Objects() {
			out = append(out, walk(ch)...)
		}
	case *fyne.Container:
		for _, ch := range c.Objects {
			out = append(out, walk(ch)...)
		}
	}
	return out
}

func (h *harness) objects() []fyne.CanvasObject { return walk(h.u.win.Content()) }

func (h *harness) overlay() []fyne.CanvasObject {
	if top := h.canvas().Overlays().Top(); top != nil {
		return walk(top)
	}
	return nil
}

func collect[T any](objs []fyne.CanvasObject) []T {
	var out []T
	for _, o := range objs {
		if v, ok := o.(T); ok {
			out = append(out, v)
		}
	}
	return out
}

// tableCells returns the loaded table cells on screen by row and column.
func (h *harness) tableCells() map[int]map[int]*tableCell {
	out := map[int]map[int]*tableCell{}
	for _, c := range collect[*tableCell](h.objects()) {
		if !c.loaded {
			continue
		}
		if out[c.row] == nil {
			out[c.row] = map[int]*tableCell{}
		}
		out[c.row][c.col] = c
	}
	return out
}

func (h *harness) header(prefix string) *headerCell {
	for _, c := range collect[*headerCell](h.objects()) {
		if strings.HasPrefix(c.Text, prefix) {
			return c
		}
	}
	h.t.Fatalf("no header %q", prefix)
	return nil
}

func (h *harness) item(id int64) core.Item {
	it, err := h.mem.Get(context.Background(), id)
	if err != nil {
		h.t.Fatal(err)
	}
	return it
}

func (h *harness) tapAt(o fyne.CanvasObject, dx, dy float32) {
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(o).Add(fyne.NewPos(dx, dy))
	test.TapCanvas(h.canvas(), pos)
}

func (h *harness) labels(objs []fyne.CanvasObject) []string {
	var out []string
	for _, l := range collect[*widget.Label](objs) {
		out = append(out, l.Text)
	}
	return out
}

// shot saves a PNG of the window when GGUIX_SHOT_DIR is set.
func (h *harness) shot(name string) {
	dir := os.Getenv("GGUIX_SHOT_DIR")
	if dir == "" {
		return
	}
	h.pump()
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		h.t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, h.canvas().Capture()); err != nil {
		h.t.Fatal(err)
	}
}

var _ = strconv.Itoa
