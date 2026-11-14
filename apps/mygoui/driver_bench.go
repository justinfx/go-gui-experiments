package main

import (
	"runtime"
	"time"

	"github.com/egoist/mygo"

	"gguix/core"
)

// benchDriver lets core/bench drive the app: every method runs on the UI
// thread inside RunOnUI.
type benchDriver struct{ a *app }

func newDriver(a *app) *benchDriver { return &benchDriver{a} }

// RunOnUI uses mygo.RunOnMain, which draws no frame, so the tick probe
// measures only how long the main thread takes to get to it.
func (d *benchDriver) RunOnUI(f func()) { mygo.RunOnMain(f) }

func (d *benchDriver) SetView(cards bool) {
	v := 0
	if cards {
		v = 1
	}
	d.a.viewMode = v
	d.a.wake()
}

func (d *benchDriver) GoToRow(i int) {
	a := d.a
	a.requestScroll(i)
	a.selectRow(i)
}

func (d *benchDriver) ScrollToRow(i int) {
	d.a.requestScroll(i)
}

func (d *benchDriver) Sort(field core.SortField, desc bool) {
	a := d.a
	a.sortField, a.sortDesc = field, desc
	a.rebuildQuery(true)
	a.requestScroll(0)
}

func (d *benchDriver) Search(text string) {
	a := d.a
	a.search = text
	a.applySearchNow()
	a.requestScroll(0)
}

func (d *benchDriver) VisibleLoaded() bool {
	a := d.a
	if a.pager.Total() < 0 {
		return false
	}
	if a.pager.Total() == 0 {
		return true
	}
	return a.pager.RangeLoaded(a.visFirst, min(a.visLast, a.pager.Total()-1))
}

func (d *benchDriver) Total() int { return a0(d.a.pager.Total()) }

func a0(n int) int { return max(n, 0) }

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

var _ = time.Now
