package main

import (
	"context"
	"time"

	"cogentcore.org/core/core"

	gc "gguix/core"
	"gguix/core/bench"
)

// benchDriver implements bench.Driver on the UI thread via AsyncLock.
type benchDriver struct{ a *App }

func (d benchDriver) RunOnUI(f func()) {
	go func() {
		d.a.body.AsyncLock()
		defer d.a.body.AsyncUnlock()
		f()
	}()
}
func (d benchDriver) SetView(cards bool) {
	if cards {
		d.a.tools.views.SelectValue("Cards")
		d.a.setView(viewCards)
	} else {
		d.a.tools.views.SelectValue("Table")
		d.a.setView(viewTable)
	}
}
func (d benchDriver) GoToRow(i int)                  { d.a.goToRow(i + 1) }
func (d benchDriver) ScrollToRow(i int)              { d.a.activeList().scrollTop(i) }
func (d benchDriver) Sort(f gc.SortField, desc bool) { d.a.setSort(f, desc) }
func (d benchDriver) Search(s string)                { d.a.tools.search.SetText(s); d.a.setSearch(s, true) }
func (d benchDriver) VisibleLoaded() bool            { return d.a.visibleLoaded() }
func (d benchDriver) Total() int                     { return d.a.pager.Total() }
func (d benchDriver) StartExport(path string, done func(int, error)) {
	d.a.startExportTo(path, done)
}
func (d benchDriver) Quit() { quitApp() }

// startBench runs the bench script when bench mode is on.
func (a *App) startBench() {
	if !a.bench {
		return
	}
	go bench.Run(context.Background(), a.svc, benchDriver{a})
}

var _ = core.TheApp
var _ = time.Now
