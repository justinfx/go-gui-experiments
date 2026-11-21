package ui

import (
	"runtime"

	"gguix/core"
)

// BenchDriver implements bench.Driver over the App presentation layer. Every
// method except RunOnUI runs on the UI thread (inside RunOnUI).
type BenchDriver struct{ a *App }

func NewBenchDriver(a *App) *BenchDriver { return &BenchDriver{a: a} }

// RunOnUI uses the framework's Ctx.Post, the documented UI-thread mechanism.
func (d *BenchDriver) RunOnUI(f func()) { d.a.post(f) }

func (d *BenchDriver) SetView(cards bool) {
	d.a.SetState(func() {
		if cards {
			d.a.SetView(core.ViewCards)
		} else {
			d.a.SetView(core.ViewTable)
		}
	})
}

func (d *BenchDriver) GoToRow(index int)     { d.a.SetState(func() { d.a.GoToRow(index) }) }
func (d *BenchDriver) ScrollToRow(index int) { d.a.SetState(func() { d.a.scrollRowToTop(index) }) }
func (d *BenchDriver) Sort(f core.SortField, desc bool) {
	d.a.SetState(func() { d.a.SortBy(f, desc) })
}
func (d *BenchDriver) Search(text string)  { d.a.SetState(func() { d.a.ApplySearch(text) }) }
func (d *BenchDriver) VisibleLoaded() bool { return d.a.VisibleLoaded() }
func (d *BenchDriver) Total() int          { return d.a.Pager.Total() }
func (d *BenchDriver) StartExport(path string, done func(rows int, err error)) {
	// gguix:target web
	if runtime.GOOS == "js" {
		path = "" // web bench: the CSV is discarded
	}
	// gguix:end
	d.a.SetState(func() { d.a.StartExportTo(path, done) })
}
func (d *BenchDriver) Quit() { d.a.Quit() }
