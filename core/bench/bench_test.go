package bench_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"gguix/core"
	"gguix/core/bench"
	"gguix/core/memory"
)

// fakeUI is a minimal Driver with a single "UI thread" goroutine.
type fakeUI struct {
	svc    core.DataService
	tasks  chan func()
	pager  *core.Pager
	top    int
	height int
	q      core.Query
	quit   chan struct{}
	once   sync.Once
}

func newFake(svc core.DataService) *fakeUI {
	f := &fakeUI{svc: svc, tasks: make(chan func(), 1024), height: 30, quit: make(chan struct{})}
	f.pager = core.NewPager(svc, core.PagerOptions{}, nil)
	go func() {
		for {
			select {
			case fn := <-f.tasks:
				fn()
			case <-f.quit:
				return
			}
		}
	}()
	f.RunOnUI(func() { f.pager.SetQuery(f.q); f.pager.SetVisible(0, f.height) })
	return f
}

func (f *fakeUI) RunOnUI(fn func()) { f.tasks <- fn }
func (f *fakeUI) SetView(bool)      {}
func (f *fakeUI) GoToRow(i int)     { f.ScrollToRow(i) }
func (f *fakeUI) ScrollToRow(i int) { f.top = i; f.pager.SetVisible(i, i+f.height) }
func (f *fakeUI) Sort(field core.SortField, desc bool) {
	f.q.SortBy, f.q.Desc = field, desc
	f.top = 0
	f.pager.SetQuery(f.q)
}
func (f *fakeUI) Search(s string) { f.q.Search = s; f.top = 0; f.pager.SetQuery(f.q) }
func (f *fakeUI) VisibleLoaded() bool {
	return f.pager.RangeLoaded(f.top, f.top+f.height)
}
func (f *fakeUI) Total() int { return f.pager.Total() }
func (f *fakeUI) StartExport(path string, done func(int, error)) {
	q := f.q
	go func() {
		n, err := f.svc.Export(context.Background(), q, io.Discard, nil)
		done(n, err)
	}()
}
func (f *fakeUI) Quit() { f.once.Do(func() { close(f.quit) }) }

func TestRunEmitsAllEvents(t *testing.T) {
	svc := memory.New(memory.Options{Rows: 600_000, Latency: 2 * time.Millisecond, ExportRowsPerSec: -1})
	f := newFake(svc)

	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w
	err := bench.Run(context.Background(), svc, f)
	w.Close()
	os.Stdout = stdout
	var out bytes.Buffer
	io.Copy(&out, r)
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out.String())
	}
	for _, ev := range []string{"ready", "jump", "scroll", "cards_scroll", "sort", "search", "export", "done"} {
		if !strings.Contains(out.String(), `{"event":"`+ev+`"`) {
			t.Errorf("missing %s in:\n%s", ev, out.String())
		}
	}
	t.Log(out.String())
}
