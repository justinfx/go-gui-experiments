package main

import (
	"context"
	"io"

	"fyne.io/fyne/v2"

	"gguix/core"
	"gguix/core/bench"
)

// benchDriver lets core/bench drive the app (brief section 9).
type benchDriver struct{ u *ui }

func startBench(ctx context.Context, u *ui) {
	_ = bench.Run(ctx, u.svc, benchDriver{u})
}

func (d benchDriver) RunOnUI(f func()) { fyne.Do(f) }

func (d benchDriver) SetView(cards bool) { d.u.setView(cards) }

func (d benchDriver) GoToRow(index int) { d.u.goToRow(index) }

func (d benchDriver) ScrollToRow(index int) {
	d.u.scrollToRow(index)
	d.u.refreshViews()
}

func (d benchDriver) Sort(field core.SortField, desc bool) {
	d.u.sortBy, d.u.sortDesc = field, desc
	d.u.queryChanged()
}

func (d benchDriver) Search(text string) {
	d.u.suppress = true
	d.u.searchEntry.SetText(text)
	d.u.suppress = false
	d.u.search = text
	d.u.queryChanged()
}

func (d benchDriver) VisibleLoaded() bool { return d.u.VisibleLoaded() }

func (d benchDriver) Total() int { return max(d.u.shown.total, 0) }

func (d benchDriver) StartExport(path string, done func(rows int, err error)) {
	var w io.Writer
	closeFn := func() {}
	w, closeFn = benchExportWriter(path)
	d.u.runExport(w, func(rows int, err error) {
		closeFn()
		done(rows, err)
	})
}

func (d benchDriver) Quit() { d.u.quitApp() }
