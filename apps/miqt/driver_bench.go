//go:build !js

package main

import (
	"gguix/core"

	"github.com/mappu/miqt/qt6/mainthread"
)

// benchDriver adapts the app to bench.Driver (section 9 of the brief).
type benchDriver struct{ a *App }

func (d benchDriver) RunOnUI(f func()) { mainthread.Start(f) }

func (d benchDriver) SetView(cards bool) { d.a.setView(cards, false) }

func (d benchDriver) GoToRow(index int) { d.a.goToRow(index) }

func (d benchDriver) ScrollToRow(index int) { d.a.scrollToTopRow(index) }

func (d benchDriver) Sort(field core.SortField, desc bool) {
	d.a.sortCol = columnForSort(field)
	d.a.setQuerySort(core.Query{SortBy: field, Desc: desc})
}

func (d benchDriver) Search(text string) {
	d.a.suppress++
	d.a.search.SetText(text)
	d.a.suppress--
	d.a.applyFilters()
}

func (d benchDriver) VisibleLoaded() bool { return d.a.visibleLoaded() }

func (d benchDriver) Total() int { return d.a.tmodel.shown }

func (d benchDriver) StartExport(path string, done func(rows int, err error)) {
	d.a.startExport(path, done)
}

func (d benchDriver) Quit() { d.a.quit() }
