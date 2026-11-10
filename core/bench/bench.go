// Package bench runs the GGUIX bench script against a Go UI.
//
// Start it from a goroutine (not the UI thread) once the window exists,
// when core.BenchEnabled() is true:
//
//	if core.BenchEnabled() {
//	    go bench.Run(context.Background(), svc, myDriver)
//	}
//
// Run executes these steps, emitting one GGUIX_METRIC line per step:
//
//  1. ready: ms from core.SinceStart() start (ProcessStart natively, navigation start in a browser) until Total() > 0 and VisibleLoaded().
//  2. jump: GoToRow(500000), ms until VisibleLoaded().
//  3. scroll: 200 steps, every 16 ms ScrollToRow(500000 + k*25); total_ms
//     until loaded after the last step, max_gap_ms, p95_gap_ms,
//     placeholder_steps.
//  4. cards_scroll: SetView(true), GoToRow(200000), switch_ms until loaded,
//     then 200 steps of ScrollToRow(200000 + k*10) every 16 ms; total_ms,
//     max_gap_ms, p95_gap_ms, placeholder_steps; then SetView(false).
//  5. sort: Sort(name, desc), ms until loaded.
//  6. search: Search("steel"), ms until loaded and Total() equals the
//     service's total for that query.
//  7. export: StartExport(ExportPath()), rows, ms until done, max_gap_ms,
//     p95_gap_ms.
//  8. done, then Quit().
//
// During steps 3, 4 and 7 a UI tick probe schedules a no-op on the UI thread
// every 16 ms through RunOnUI and records how long each took to run.
// If a step times out (60 s), an "error" metric is emitted, the remaining
// steps are skipped, and "done" and Quit still happen.
package bench

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"gguix/core"
)

// Driver is implemented by the app. RunOnUI may be called from any
// goroutine; every other method is only ever called by Run from inside
// RunOnUI, that is, on the UI thread.
type Driver interface {
	// RunOnUI schedules f to run on the UI thread and returns immediately.
	RunOnUI(f func())
	// SetView switches between the table (false) and cards (true) views.
	SetView(cards bool)
	// GoToRow makes row index (0-based) the top visible row or card and
	// selects it, exactly like the Go-to-row control.
	GoToRow(index int)
	// ScrollToRow makes row index the top visible row or card, without
	// changing the selection.
	ScrollToRow(index int)
	// Sort applies a sort, like clicking a column header.
	Sort(field core.SortField, desc bool)
	// Search applies the search text immediately (no debounce).
	Search(text string)
	// VisibleLoaded reports whether Total is known and every row or card in
	// the current visible range has data. It must reflect state changes made
	// by the methods above immediately (for example by checking
	// core.Pager.RangeLoaded for the range just passed to SetVisible),
	// never stale widget content.
	VisibleLoaded() bool
	// Total returns the item count the UI currently shows.
	Total() int
	// StartExport starts an export of the current query to path without a
	// dialog ("" on web: discard the CSV) and calls done when it finishes,
	// from any goroutine.
	StartExport(path string, done func(rows int, err error))
	// Quit closes the app (no-op on web).
	Quit()
}

const (
	stepTimeout = 60 * time.Second
	tick        = 16 * time.Millisecond
	steps       = 200
	searchText  = "steel"
)

// ExportPath returns $GGUIX_BENCH_EXPORT, or a file in the temp dir.
func ExportPath() string {
	if p := os.Getenv("GGUIX_BENCH_EXPORT"); p != "" {
		return p
	}
	return filepath.Join(os.TempDir(), "gguix-bench-export.csv")
}

var errTimeout = errors.New("timeout")

type runner struct {
	ctx context.Context
	d   Driver
}

func (r runner) ui(f func()) error {
	done := make(chan struct{})
	r.d.RunOnUI(func() { f(); close(done) })
	t := time.NewTimer(stepTimeout)
	defer t.Stop()
	select {
	case <-done:
		return nil
	case <-t.C:
		return errTimeout
	case <-r.ctx.Done():
		return r.ctx.Err()
	}
}

func (r runner) loaded() (bool, error) {
	var ok bool
	err := r.ui(func() { ok = r.d.VisibleLoaded() })
	return ok, err
}

// waitUntil polls cond on the UI thread every 4 ms.
func (r runner) waitUntil(cond func() bool) error {
	deadline := time.Now().Add(stepTimeout)
	for {
		var ok bool
		if err := r.ui(func() { ok = cond() }); err != nil {
			return err
		}
		if ok {
			return nil
		}
		if time.Now().After(deadline) {
			return errTimeout
		}
		select {
		case <-time.After(4 * time.Millisecond):
		case <-r.ctx.Done():
			return r.ctx.Err()
		}
	}
}

func (r runner) waitLoaded() error { return r.waitUntil(r.d.VisibleLoaded) }

// probe measures UI-thread scheduling delay.
type probe struct {
	mu      sync.Mutex
	pending map[int]time.Time
	delays  []time.Duration
	stop    chan struct{}
	wg      sync.WaitGroup
}

func startProbe(d Driver) *probe {
	p := &probe{pending: map[int]time.Time{}, stop: make(chan struct{})}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		t := time.NewTicker(tick)
		defer t.Stop()
		for id := 0; ; id++ {
			select {
			case <-p.stop:
				return
			case <-t.C:
			}
			p.mu.Lock()
			p.pending[id] = time.Now()
			p.mu.Unlock()
			d.RunOnUI(func() {
				p.mu.Lock()
				if t0, ok := p.pending[id]; ok {
					delete(p.pending, id)
					p.delays = append(p.delays, time.Since(t0))
				}
				p.mu.Unlock()
			})
		}
	}()
	return p
}

// finish stops the probe; callbacks that never ran count as delayed until now.
func (p *probe) finish() (maxMs, p95Ms float64) {
	close(p.stop)
	p.wg.Wait()
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	for _, t0 := range p.pending {
		p.delays = append(p.delays, now.Sub(t0))
	}
	clear(p.pending)
	if len(p.delays) == 0 {
		return 0, 0
	}
	slices.Sort(p.delays)
	i95 := min(len(p.delays)-1, int(float64(len(p.delays))*0.95))
	return core.Ms(p.delays[len(p.delays)-1]), core.Ms(p.delays[i95])
}

// Run executes the bench script. svc is used only to learn the expected
// total of the search step. It returns the first error, after emitting
// "done" and calling Quit.
func Run(ctx context.Context, svc core.DataService, d Driver) error {
	r := runner{ctx: ctx, d: d}
	err := r.run(svc)
	if err != nil {
		core.EmitMetric("error", map[string]any{"message": err.Error()})
	}
	core.EmitMetric("done", nil)
	_ = r.ui(d.Quit)
	return err
}

func (r runner) run(svc core.DataService) error {
	d := r.d
	// 1. ready
	if err := r.waitUntil(func() bool { return d.Total() > 0 && d.VisibleLoaded() }); err != nil {
		return fmt.Errorf("ready: %w", err)
	}
	core.EmitMetric("ready", map[string]any{"ms": core.Ms(core.SinceStart())})

	// 2. jump
	t0 := time.Now()
	if err := r.ui(func() { d.GoToRow(500_000) }); err != nil {
		return fmt.Errorf("jump: %w", err)
	}
	if err := r.waitLoaded(); err != nil {
		return fmt.Errorf("jump: %w", err)
	}
	core.EmitMetric("jump", map[string]any{"ms": core.Ms(time.Since(t0))})

	// 3. scroll
	fields, err := r.scroll(500_000, 25)
	if err != nil {
		return fmt.Errorf("scroll: %w", err)
	}
	core.EmitMetric("scroll", fields)

	// 4. cards_scroll
	t0 = time.Now()
	if err := r.ui(func() { d.SetView(true); d.GoToRow(200_000) }); err != nil {
		return fmt.Errorf("cards_scroll: %w", err)
	}
	if err := r.waitLoaded(); err != nil {
		return fmt.Errorf("cards_scroll: %w", err)
	}
	switchMs := core.Ms(time.Since(t0))
	fields, err = r.scroll(200_000, 10)
	if err != nil {
		return fmt.Errorf("cards_scroll: %w", err)
	}
	fields["switch_ms"] = switchMs
	core.EmitMetric("cards_scroll", fields)
	if err := r.ui(func() { d.SetView(false) }); err != nil {
		return fmt.Errorf("cards_scroll: %w", err)
	}
	if err := r.waitLoaded(); err != nil {
		return fmt.Errorf("cards_scroll: %w", err)
	}

	// 5. sort
	t0 = time.Now()
	if err := r.ui(func() { d.Sort(core.SortName, true) }); err != nil {
		return fmt.Errorf("sort: %w", err)
	}
	if err := r.waitLoaded(); err != nil {
		return fmt.Errorf("sort: %w", err)
	}
	core.EmitMetric("sort", map[string]any{"ms": core.Ms(time.Since(t0))})

	// 6. search
	exp, err := svc.Page(r.ctx, core.PageRequest{Query: core.Query{Search: searchText, SortBy: core.SortName, Desc: true}, Limit: 1})
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}
	t0 = time.Now()
	if err := r.ui(func() { d.Search(searchText) }); err != nil {
		return fmt.Errorf("search: %w", err)
	}
	var total int
	if err := r.waitUntil(func() bool { total = d.Total(); return total == exp.Total && d.VisibleLoaded() }); err != nil {
		return fmt.Errorf("search: %w", err)
	}
	core.EmitMetric("search", map[string]any{"ms": core.Ms(time.Since(t0)), "total": total})

	// 7. export
	type res struct {
		rows int
		err  error
	}
	ch := make(chan res, 1)
	p := startProbe(d)
	t0 = time.Now()
	if err := r.ui(func() { d.StartExport(ExportPath(), func(rows int, err error) { ch <- res{rows, err} }) }); err != nil {
		p.finish()
		return fmt.Errorf("export: %w", err)
	}
	var out res
	select {
	case out = <-ch:
	case <-time.After(5 * stepTimeout):
		out.err = errTimeout
	case <-r.ctx.Done():
		out.err = r.ctx.Err()
	}
	elapsed := time.Since(t0)
	maxGap, p95 := p.finish()
	if out.err != nil {
		return fmt.Errorf("export: %w", out.err)
	}
	core.EmitMetric("export", map[string]any{"rows": out.rows, "ms": core.Ms(elapsed), "max_gap_ms": maxGap, "p95_gap_ms": p95})
	return nil
}

func (r runner) scroll(base, step int) (map[string]any, error) {
	p := startProbe(r.d)
	start := time.Now()
	next := start
	placeholders := 0
	for k := 1; k <= steps; k++ {
		next = next.Add(tick)
		if d := time.Until(next); d > 0 {
			time.Sleep(d)
		}
		ok, err := r.loaded()
		if err != nil {
			p.finish()
			return nil, err
		}
		if !ok {
			placeholders++
		}
		row := base + k*step
		if err := r.ui(func() { r.d.ScrollToRow(row) }); err != nil {
			p.finish()
			return nil, err
		}
	}
	err := r.waitLoaded()
	total := time.Since(start)
	maxGap, p95 := p.finish()
	if err != nil {
		return nil, err
	}
	return map[string]any{"total_ms": core.Ms(total), "max_gap_ms": maxGap, "p95_gap_ms": p95, "placeholder_steps": placeholders}, nil
}
