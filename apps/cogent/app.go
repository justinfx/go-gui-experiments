package main

// Application state and logic shared by the desktop and web targets:
// query, selection, preferences, theme, view switching and the UI thread
// bridge. The widgets live in table.go, cards.go, form.go and ui.go.

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cogentcore.org/core/colors/matcolor"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"

	gc "gguix/core"
	"gguix/core/host"
)

const (
	appID       = "cogent"
	windowTitle = "GGUIX - Cogent Core"
)

type viewKind int

const (
	viewTable viewKind = iota
	viewCards
)

// App is the whole application.
type App struct {
	svc   gc.DataService
	pst   gc.PrefsStore
	prefs gc.Prefs
	pager *gc.Pager
	query gc.Query

	// bench mode: no debounce, no selection restore, bench export.
	bench bool

	body    *core.Body
	table   *rowsTable
	cards   *cardsList
	stack   *core.Frame
	empty   *core.Text
	form    *detailForm
	status  *statusBar
	tools   *toolbarParts
	sortCol int

	view     viewKind
	selRow   int   // selected row index in the current result, -1 for none
	selID    int64 // selected item id, 0 for none
	selWait  bool  // a row was selected before its item was loaded
	selCheck bool  // after a query change, keep the selection only if in page 0

	confirming bool

	gen          atomic.Int64 // bumped by every pager change
	refreshedGen int64
	queued       atomic.Bool
	lastTotal    int

	searchTimer *time.Timer

	prefsMu    sync.Mutex
	prefsSnap  gc.Prefs
	prefsTimer *time.Timer

	hostInfo host.Info

	flashMsg   string
	flashTimer *time.Timer

	exp      exportState
	lastText core.Widget
}

// newApp makes the application model on a data service and prefs store.
func newApp(svc gc.DataService, pst gc.PrefsStore, bench bool) *App {
	a := &App{svc: svc, pst: pst, bench: bench, selRow: -1, lastTotal: -2}
	p, err := pst.Load()
	if err != nil {
		p = gc.DefaultPrefs()
	}
	a.prefs = p
	if a.prefs.View != gc.ViewCards {
		a.prefs.View = gc.ViewTable
	}
	if a.prefs.Theme == "" {
		a.prefs.Theme = gc.ThemeSystem
	}
	if a.prefs.SortBy == "" {
		a.prefs.SortBy = gc.SortID
	}
	if bench {
		a.prefs.SelectedID = 0
	}
	if a.prefs.View == gc.ViewCards {
		a.view = viewCards
	}
	a.query = a.buildQuery()
	a.sortCol = colForField(a.prefs.SortBy)
	a.pager = gc.NewPager(svc, gc.DefaultPagerOptions(), a.pagerChanged)
	return a
}

func colForField(f gc.SortField) int {
	for i, c := range columns {
		if c.sort == f {
			return i
		}
	}
	return colID
}

func (a *App) buildQuery() gc.Query {
	cats := append([]gc.Category(nil), a.prefs.Categories...)
	return gc.Query{
		Search:     strings.TrimSpace(a.prefs.Search),
		Categories: cats,
		Status:     a.prefs.Status,
		SortBy:     a.prefs.SortBy,
		Desc:       a.prefs.SortDesc,
	}
}

func (a *App) currentView() viewKind { return a.view }

// listBase returns the active list view.
func (a *App) activeList() interface {
	scrollTop(int)
	setSelected(int)
	visibleRange() (int, int)
} {
	if a.view == viewCards {
		return a.cards
	}
	return a.table
}

//////// UI thread bridge

// async runs f on the UI thread (the framework's AsyncLock/AsyncUnlock
// mechanism for updates from other goroutines).
func (a *App) async(f func()) {
	a.body.AsyncLock()
	defer a.body.AsyncUnlock()
	f()
}

// pagerChanged is the pager's onChange, called from background goroutines.
func (a *App) pagerChanged() {
	a.gen.Add(1)
	if a.queued.Swap(true) {
		return
	}
	go func() {
		time.Sleep(6 * time.Millisecond) // coalesce bursts
		a.queued.Store(false)
		a.async(a.refresh)
	}()
}

// refresh redraws the views after the pager's data changed (UI thread).
func (a *App) refresh() {
	a.refreshedGen = a.gen.Load()
	total := a.pager.Total()
	a.resolveSelection()
	a.updateStack()
	changed := total != a.lastTotal
	a.lastTotal = total
	if a.view == viewCards {
		if changed {
			a.cards.Update()
		} else {
			a.cards.UpdateRender()
		}
	} else {
		if changed {
			a.table.Update()
		} else {
			a.table.UpdateRender()
		}
	}
	a.updateStatus()
}

// updateStack shows the active view, or the empty message.
func (a *App) updateStack() {
	top := int(a.view)
	if a.pager.Total() == 0 {
		top = 2
	}
	if a.stack.StackTop != top {
		a.stack.StackTop = top
		a.stack.Update()
	}
}

// visibleLoaded reports whether the visible range is loaded and refreshed.
func (a *App) visibleLoaded() bool {
	if a.pager.Total() < 0 {
		return false
	}
	first, last := a.activeList().visibleRange()
	if a.pager.Total() == 0 {
		return a.refreshedGen == a.gen.Load()
	}
	return a.pager.RangeLoaded(first, last) && a.refreshedGen == a.gen.Load()
}

// visibleMoved is called when the visible row range of a view changes.
func (a *App) visibleMoved() {
	if a.status != nil {
		a.updateStatus()
	}
}

//////// Query

// applyQuery installs the query derived from the preferences (UI thread).
func (a *App) applyQuery() {
	a.query = a.buildQuery()
	a.selCheck = a.selID != 0
	a.selWait = false
	a.pager.SetQuery(a.query)
	a.lastTotal = -2
	a.table.StartIndex = 0
	a.cards.StartIndex = 0
	a.table.scrollTop(0)
	a.cards.scrollTop(0)
	a.updateStack()
	a.updateStatus()
	if a.tools != nil {
		a.tools.sync()
	}
	a.savePrefsSoon()
}

// setSearch applies a search text, with the 250 ms debounce unless now.
func (a *App) setSearch(s string, now bool) {
	if a.searchTimer != nil {
		a.searchTimer.Stop()
	}
	apply := func() {
		if a.prefs.Search == s {
			return
		}
		a.prefs.Search = s
		a.applyQuery()
	}
	if now || a.bench {
		apply()
		return
	}
	a.searchTimer = time.AfterFunc(250*time.Millisecond, func() { a.async(apply) })
}

func (a *App) setCategories(cats []gc.Category) {
	a.prefs.Categories = cats
	a.applyQuery()
}

func (a *App) setStatus(st *gc.Status) {
	a.prefs.Status = st
	a.applyQuery()
}

// sortByColumn is the header click: ascending first, then toggle.
func (a *App) sortByColumn(c int) {
	if a.sortCol == c {
		a.prefs.SortDesc = !a.prefs.SortDesc
	} else {
		a.sortCol = c
		a.prefs.SortBy = columns[c].sort
		a.prefs.SortDesc = false
	}
	a.afterSort()
}

// setSort applies a sort by field (bench driver, prefs restore).
func (a *App) setSort(f gc.SortField, desc bool) {
	a.sortCol = colForField(f)
	a.prefs.SortBy = f
	a.prefs.SortDesc = desc
	a.afterSort()
}

func (a *App) afterSort() {
	a.applyQuery()
	if a.table.header != nil {
		a.table.header.UpdateTree()
		a.table.header.NeedsRender()
	}
}

// reload reloads the current query from the first page.
func (a *App) reload() {
	a.applyQuery()
}

//////// Selection

// currentItemLoaded returns the item for the selection, if known.
func (a *App) syncViewSelection() {
	a.table.setSelected(a.selRow)
	a.cards.setSelected(a.selRow)
}

// userSelect is called when the user selects row idx in either view.
func (a *App) userSelect(idx int) {
	if a.confirming {
		a.syncViewSelection()
		return
	}
	if idx < 0 || idx == a.selRow {
		a.syncViewSelection()
		return
	}
	if a.form.dirty() {
		a.syncViewSelection()
		a.confirmDiscard(func() { a.selectRow(idx) })
		return
	}
	a.selectRow(idx)
}

// selectRow selects row idx without the dirty guard.
func (a *App) selectRow(idx int) {
	a.selRow = idx
	a.selCheck = false
	if it, ok := a.pager.Row(idx); ok {
		a.selID = it.ID
		a.selWait = false
		a.form.load(it)
	} else {
		a.selID = 0
		a.selWait = true
		a.form.clear()
	}
	a.syncViewSelection()
	a.savePrefsSoon()
}

// selectItem selects an item whose row is known (prefs restore).
func (a *App) selectItem(row int, id int64) {
	a.selRow = row
	a.selID = id
	a.selCheck = false
	a.selWait = false
	if it, ok := a.pager.Row(row); ok && it.ID == id {
		a.form.load(it)
	} else {
		a.selWait = true
		a.selID = id
	}
	a.syncViewSelection()
}

func (a *App) clearSelection() {
	a.selRow = -1
	a.selID = 0
	a.selWait = false
	a.selCheck = false
	a.form.clear()
	a.syncViewSelection()
	a.savePrefsSoon()
}

// resolveSelection completes a pending selection once data has arrived.
func (a *App) resolveSelection() {
	total := a.pager.Total()
	if total < 0 {
		return
	}
	if a.selWait && a.selRow >= 0 {
		if it, ok := a.pager.Row(a.selRow); ok && (a.selID == 0 || it.ID == a.selID) {
			a.selID = it.ID
			a.selWait = false
			a.form.load(it)
			a.savePrefsSoon()
		}
	}
	if a.selCheck {
		n := min(total, 200)
		if n > 0 && !a.pager.RangeLoaded(0, n-1) {
			return
		}
		a.selCheck = false
		for i := 0; i < n; i++ {
			if it, ok := a.pager.Row(i); ok && it.ID == a.selID {
				a.selRow = i
				a.syncViewSelection()
				return
			}
		}
		a.clearSelection()
	}
}

// goToRow scrolls the active view so row n (1-based) is on top and selects it.
func (a *App) goToRow(n int) {
	total := a.pager.Total()
	if total <= 0 {
		return
	}
	n = max(1, min(n, total))
	idx := n - 1
	a.activeList().scrollTop(idx)
	a.userSelect(idx)
}

// goToRowText parses the Go-to-row text.
func (a *App) goToRowText(s string) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return
	}
	a.goToRow(n)
}

// setView switches between table and cards.
func (a *App) setView(v viewKind) {
	if v == a.view && a.stack.StackTop == int(v) {
		return
	}
	a.view = v
	if v == viewCards {
		a.prefs.View = gc.ViewCards
	} else {
		a.prefs.View = gc.ViewTable
	}
	a.updateStack()
	top := 0
	if a.selRow >= 0 {
		top = a.selRow
	}
	l := a.activeList()
	l.scrollTop(top)
	l.setSelected(a.selRow)
	a.refresh()
	a.tools.sync()
	a.savePrefsSoon()
}

//////// Dialogs

// confirmDiscard asks "Discard unsaved changes?" and runs onDiscard on Discard.
func (a *App) confirmDiscard(onDiscard func()) {
	a.confirming = true
	d := core.NewBody("Discard unsaved changes?")
	core.NewText(d).SetType(core.TextSupporting).SetText("Your edits to this item will be lost.")
	d.OnClose(func(e events.Event) { a.confirming = false })
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Discard").OnClick(func(e events.Event) {
			a.confirming = false
			onDiscard()
		})
	})
	d.RunDialog(a.body)
}

func (a *App) showAbout() {
	d := core.NewBody("About GGUIX")
	for _, line := range aboutLines() {
		core.NewText(d).SetType(core.TextBodyLarge).SetText(line)
	}
	d.AddOKOnly()
	d.RunDialog(a.body)
}

//////// Theme

func (a *App) setTheme(t string) {
	switch t {
	case gc.ThemeLight:
		core.AppearanceSettings.Theme = core.ThemeLight
	case gc.ThemeDark:
		core.AppearanceSettings.Theme = core.ThemeDark
	default:
		t = gc.ThemeSystem
		core.AppearanceSettings.Theme = core.ThemeAuto
	}
	a.prefs.Theme = t
	core.AppearanceSettings.Apply()
	core.UpdateAll()
	if a.tools != nil {
		a.tools.sync()
	}
	a.refreshMenus()
	a.savePrefsSoon()
}

// themeDark reports whether the dark scheme is active.
func themeDark() bool { return matcolor.SchemeIsDark }

//////// Preferences

func (a *App) snapshotPrefs() gc.Prefs {
	p := a.prefs
	p.SelectedID = a.selID
	p.Categories = append([]gc.Category(nil), a.prefs.Categories...)
	return p
}

// savePrefsSoon saves the preferences 500 ms after the last change.
func (a *App) savePrefsSoon() {
	if a.bench {
		return
	}
	snap := a.snapshotPrefs()
	a.prefsMu.Lock()
	defer a.prefsMu.Unlock()
	a.prefsSnap = snap
	if a.prefsTimer == nil {
		a.prefsTimer = time.AfterFunc(500*time.Millisecond, a.flushPrefs)
	} else {
		a.prefsTimer.Reset(500 * time.Millisecond)
	}
}

// flushPrefs writes the latest preferences now.
func (a *App) flushPrefs() {
	a.prefsMu.Lock()
	defer a.prefsMu.Unlock()
	if a.prefsTimer != nil {
		a.prefsTimer.Stop()
		a.prefsTimer = nil
	}
	if a.bench || a.prefsSnap.Theme == "" {
		return
	}
	_ = a.pst.Save(a.prefsSnap)
}

// saveOnQuit saves immediately with the current state (UI thread).
func (a *App) saveOnQuit() {
	a.prefsMu.Lock()
	a.prefsSnap = a.snapshotPrefs()
	a.prefsMu.Unlock()
	a.flushPrefs()
}

// restoreSelection finds the saved selection after launch.
func (a *App) restoreSelection() {
	id := a.prefs.SelectedID
	if id == 0 || a.bench {
		return
	}
	q := a.query
	go func() {
		row, ok, err := gc.FindRow(context.Background(), a.svc, q, id, 2000)
		if err != nil || !ok {
			return
		}
		a.async(func() {
			if a.query.Search != q.Search || a.selID != 0 || a.selRow >= 0 {
				return // the user already changed things
			}
			a.activeList().scrollTop(row)
			a.selectItem(row, id)
		})
	}()
}

//////// Status messages

// flash shows a transient message in the status bar for 2 seconds.
func (a *App) flash(msg string) {
	a.flashMsg = msg
	if a.flashTimer != nil {
		a.flashTimer.Stop()
	}
	a.flashTimer = time.AfterFunc(2*time.Second, func() {
		a.async(func() {
			a.flashMsg = ""
			a.updateStatus()
		})
	})
	a.updateStatus()
}

// statusText builds the left status text.
func (a *App) statusText() string {
	if a.flashMsg != "" {
		return a.flashMsg
	}
	total := a.pager.Total()
	if total < 0 {
		return "Loading…"
	}
	first, last := a.activeList().visibleRange()
	rows := ""
	if total > 0 {
		rows = " · rows " + strconv.Itoa(first+1) + "-" + strconv.Itoa(last+1)
	}
	st := a.pager.Stats()
	return gc.FormatCount(total) + " items" + rows +
		" · last fetch " + strconv.Itoa(int(st.LastFetch.Milliseconds())) + " ms · " +
		strconv.Itoa(st.CachedPages) + " pages cached"
}

func (a *App) updateStatus() {
	if a.status == nil {
		return
	}
	a.status.update()
}

var _ = states.Invisible
var _ = styles.Row
