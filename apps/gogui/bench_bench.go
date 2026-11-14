package main

import (
	"runtime"

	"github.com/go-gui-org/go-gui/gui"

	"gguix/core"
)

// benchDriver lets core/bench drive the app. Every method except RunOnUI
// runs on the frame thread inside RunOnUI.
type benchDriver struct{ a *app }

func newDriver(a *app) *benchDriver { return &benchDriver{a} }

// RunOnUI uses Window.QueueCommand, go-gui's documented way to run code
// on the frame thread from another goroutine.
func (d *benchDriver) RunOnUI(f func()) {
	d.a.win.QueueCommand(func(*gui.Window) { f() })
}

func (d *benchDriver) SetView(cards bool) {
	v := 0
	if cards {
		v = 1
	}
	d.a.setView(v)
	d.a.wake()
}

func (d *benchDriver) GoToRow(i int) {
	d.a.requestScroll(i)
	d.a.selectRow(i)
}

func (d *benchDriver) ScrollToRow(i int) { d.a.requestScroll(i) }

func (d *benchDriver) Sort(field core.SortField, desc bool) {
	a := d.a
	a.sortField, a.sortDesc = field, desc
	a.rebuildQuery(true)
}

func (d *benchDriver) Search(text string) {
	a := d.a
	a.search = text
	a.applySearchNow()
}

func (d *benchDriver) VisibleLoaded() bool {
	a := d.a
	t := a.pager.Total()
	if t < 0 {
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
