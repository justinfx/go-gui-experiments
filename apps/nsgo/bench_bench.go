package main

import (
	"context"
	"runtime"
	"time"

	"github.com/nsgo-dev/nsgo/core"

	gx "gguix/core"
	"gguix/core/bench"
)

// benchDriver lets core/bench drive the app. Every method except RunOnUI
// runs on the engine goroutine inside RunOnUI.
type benchDriver struct{ a *app }

// startBench runs the script once the first frame has been drawn.
func startBench(cx *core.Scope, a *app) {
	if !a.bench {
		return
	}
	cx.RequestAnimationFrame(func(time.Time) {
		go bench.Run(context.Background(), a.svc, &benchDriver{a})
	})
}

// RunOnUI uses Scope.Post, NSGo's documented way to run code on the engine
// goroutine from another goroutine.
func (d *benchDriver) RunOnUI(f func()) { d.a.post(f) }

func (d *benchDriver) SetView(cards bool) {
	v := 0
	if cards {
		v = 1
	}
	d.a.setView(v)
}

func (d *benchDriver) GoToRow(i int) {
	d.a.requestScroll(i)
	d.a.selectRow(i)
}

func (d *benchDriver) ScrollToRow(i int) { d.a.requestScroll(i) }

func (d *benchDriver) Sort(field gx.SortField, desc bool) {
	d.a.sortField, d.a.sortDesc = field, desc
	d.a.rebuildQuery(true)
}

func (d *benchDriver) Search(text string) {
	d.a.search = text
	d.a.rQuery.Bump()
	d.a.applySearchNow()
}

// VisibleLoaded is true when the total is known, every row of the requested
// visible range is in the pager and the refresh the pager asked for has run.
func (d *benchDriver) VisibleLoaded() bool {
	a := d.a
	t := a.pager.Total()
	if t < 0 || a.rowsDirty.Load() {
		return false
	}
	if t == 0 {
		return true
	}
	return a.pager.RangeLoaded(a.visFirst, min(a.visLast, t-1))
}

func (d *benchDriver) Total() int { return max(d.a.pager.Total(), 0) }

func (d *benchDriver) StartExport(path string, done func(rows int, err error)) {
	if runtime.GOOS == "js" {
		path = "" // web: the CSV is discarded
	}
	d.a.runExport(path, done)
}

func (d *benchDriver) Quit() {
	if d.a.plat.Quit != nil {
		d.a.plat.Quit()
	}
}
