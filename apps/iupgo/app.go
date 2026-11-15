package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gen2brain/iup-go/iup"

	"gguix/core"
	"gguix/core/host"
)

const appID = "iupgo"

// Names shown in the toolbar dropdowns.
var (
	statusChoices = []string{"All statuses", "Active", "Backordered", "Discontinued"}
	themeChoices  = []string{"System", "Light", "Dark"}
)

// platformHooks is what differs between the desktop and the browser. The
// desktop and web files fill it in; tests replace it.
type platformHooks struct {
	// SaveFile asks where to write the export and calls done (any goroutine
	// is allowed, the app marshals); ok is false when cancelled.
	SaveFile func(defaultName string, done func(path string, ok bool, err error))
	// OpenTextFile asks for a .txt file and calls done with its bytes.
	OpenTextFile func(done func(data []byte, ok bool, err error))
	// Notify tells the user an export finished; false means the system
	// cannot show notifications, so the app shows a message instead.
	Notify func(title, body string, failed func()) bool
	// Reveal shows a file or folder in the file manager.
	Reveal func(path string) error
	// Deliver hands the finished export to the user when there is no file
	// dialog (browser download). Nil on the desktop.
	Deliver func(name string, data []byte) error
	// Confirm asks a yes/no question with the given button labels and calls
	// exactly one of ok or cancel. On the desktop it runs synchronously.
	Confirm func(title, msg, okLabel, cancelLabel string, ok, cancel func())
	// Alert shows an error message.
	Alert func(msg string)
	// Clipboard puts text on the system clipboard.
	Clipboard func(text string)
	// Quit closes the app.
	Quit func()
	// HostActions says whether Reveal works for this target.
	HostActions bool
}

// formFields is everything the detail form edits, comparable so that the
// dirty state is a plain comparison with the snapshot taken on load.
type formFields struct {
	name, price, notes, cat string
	qty                     string
	status                  core.Status
	rating                  float64
	featured                bool
	hasDate                 bool
	date                    core.Date
}

type form struct {
	item core.Item // the item as loaded or last saved
	has  bool
	cur  formFields
	snap formFields
	errs map[string]string
}

func fieldsOf(it core.Item) formFields {
	f := formFields{
		name:     it.Name,
		price:    core.FormatPrice(it.PriceCents),
		notes:    it.Notes,
		cat:      it.Category.String(),
		qty:      fmt.Sprint(it.Quantity),
		status:   it.Status,
		rating:   round1(float64(it.Rating)),
		featured: it.Featured,
	}
	if it.RestockDate != nil {
		f.hasDate = true
		f.date = *it.RestockDate
	}
	return f
}

func round1(v float64) float64 {
	if v < 0 {
		v = 0
	}
	if v > 5 {
		v = 5
	}
	return float64(int(v*10+0.5)) / 10
}

func (f *form) load(it core.Item) {
	f.item, f.has = it, true
	f.cur = fieldsOf(it)
	f.snap = f.cur
	f.errs = nil
}

func (f *form) clear() { *f = form{} }

func (f *form) dirty() bool { return f.has && f.cur != f.snap }

// toItem builds the item the form describes. Parse problems are returned as
// field errors, validation problems by core.ValidateItem.
func (f *form) toItem() (core.Item, map[string]string) {
	errs := map[string]string{}
	it := f.item
	it.Name = f.cur.name
	it.Notes = f.cur.notes
	if c, err := core.ParseCategory(f.cur.cat); err == nil {
		it.Category = c
	}
	if cents, err := core.ParsePrice(f.cur.price); err != nil {
		errs["priceCents"] = err.Error()
	} else {
		it.PriceCents = cents
	}
	if q, err := strconv.Atoi(strings.ReplaceAll(strings.TrimSpace(f.cur.qty), ",", "")); err != nil || q < 0 || q > 1_000_000 {
		errs["quantity"] = "Quantity must be between 0 and 1,000,000."
	} else {
		it.Quantity = int32(q)
	}
	it.Status = f.cur.status
	it.Rating = float32(round1(f.cur.rating))
	it.Featured = f.cur.featured
	if f.cur.hasDate {
		d := f.cur.date
		it.RestockDate = &d
	} else {
		it.RestockDate = nil
	}
	if ve := core.ValidateItem(it); ve != nil {
		for _, fe := range ve.Errors {
			if prev, ok := errs[fe.Field]; ok {
				errs[fe.Field] = prev + " " + fe.Message
			} else {
				errs[fe.Field] = fe.Message
			}
		}
	}
	return it, errs
}

// app is the whole application. Everything except the fields guarded by a
// mutex or atomics is touched only on the IUP UI thread.
type app struct {
	svc   core.DataService
	store core.PrefsStore
	pager *core.Pager
	plat  platformHooks
	info  host.Info
	bench bool

	// Query state.
	prefs     core.Prefs
	query     core.Query
	search    string
	cats      [8]bool
	statusSel string
	themeSel  string
	viewMode  int // 0 table, 1 cards
	sortField core.SortField
	sortDesc  bool
	dark      bool
	pal       core.Palette

	// Selection: selRow is the row index in the current result (-1 none).
	selRow    int
	selID     int64
	keepSelID int64 // selected item to keep once the first page loads

	// Visible range of the active view (0-based, inclusive) and how many
	// rows fit.
	visFirst, visLast int
	visH              int

	f form

	// Pager change coalescing: pending is set from the pager goroutine.
	pending atomic.Bool

	// Export.
	exp struct {
		running     bool
		done, total int
		cancel      context.CancelFunc
		lastPath    string
	}

	// Message shown in the status bar for a short time.
	msg string

	// Prefs saving.
	prefMu    sync.Mutex
	prefLast  core.Prefs
	prefTimer *time.Timer
	prefDirty bool

	ready   atomic.Bool
	ui      ui // widgets and view state (ui.go)
	hostDlg iup.Ihandle
}

func newApp(svc core.DataService, store core.PrefsStore, info host.Info, bench bool) *app {
	a := &app{
		svc: svc, store: store, info: info, bench: bench,
		selRow: -1, visH: 25, visLast: 24, pal: core.Colors(false),
		statusSel: statusChoices[0], themeSel: themeChoices[0],
	}
	a.exp.total = 1
	a.pager = core.NewPager(svc, core.PagerOptions{PageSize: 200, MaxPages: 20, MaxInFlight: 2, PrefetchMargin: 50}, a.pagerChanged)
	a.loadPrefs()
	a.rebuildQuery(false)
	return a
}

// pagerChanged runs on a pager goroutine; it only schedules the UI work.
func (a *app) pagerChanged() {
	if !a.ready.Load() {
		return
	}
	if a.pending.CompareAndSwap(false, true) {
		a.post(a.onPagerChange)
	}
}

// loadPrefs restores the saved preferences into the state (before the
// first query).
func (a *app) loadPrefs() {
	p, err := a.store.Load()
	if err != nil {
		p = core.DefaultPrefs()
	}
	if a.bench {
		p.SelectedID = 0
	}
	a.prefs = p
	a.prefLast = p
	switch p.Theme {
	case "light":
		a.themeSel = "Light"
	case "dark":
		a.themeSel = "Dark"
	default:
		a.themeSel = "System"
	}
	if p.View == "cards" {
		a.viewMode = 1
	}
	a.sortField, a.sortDesc = p.SortBy, p.SortDesc
	if a.sortField == "" {
		a.sortField = core.SortID
	}
	a.search = p.Search
	for _, c := range p.Categories {
		if int(c) < len(a.cats) {
			a.cats[c] = true
		}
	}
	a.statusSel = statusChoices[0]
	if p.Status != nil {
		a.statusSel = p.Status.String()
	}
	a.selID = p.SelectedID
}

// currentPrefs builds the preferences from the live state.
func (a *app) currentPrefs() core.Prefs {
	p := a.prefs
	p.Theme = strings.ToLower(a.themeSel)
	p.View = "table"
	if a.viewMode == 1 {
		p.View = "cards"
	}
	p.SortBy, p.SortDesc = a.sortField, a.sortDesc
	p.Search = a.search
	p.Categories = a.query.Categories
	p.Status = a.query.Status
	p.SelectedID = a.selID
	return p
}

func prefsEqual(a, b core.Prefs) bool {
	if a.Theme != b.Theme || a.SortBy != b.SortBy || a.SortDesc != b.SortDesc || a.Search != b.Search ||
		a.SelectedID != b.SelectedID || a.View != b.View || a.WindowWidth != b.WindowWidth || a.WindowHeight != b.WindowHeight {
		return false
	}
	if len(a.Categories) != len(b.Categories) {
		return false
	}
	for i := range a.Categories {
		if a.Categories[i] != b.Categories[i] {
			return false
		}
	}
	if (a.Status == nil) != (b.Status == nil) || (a.Status != nil && *a.Status != *b.Status) {
		return false
	}
	return true
}

// syncPrefs notices a change of the saved state and starts the 500 ms
// debounce timer. The timer saves from its own goroutine.
func (a *app) syncPrefs() {
	if a.bench {
		return
	}
	cur := a.currentPrefs()
	a.prefMu.Lock()
	cur.WindowWidth, cur.WindowHeight = a.prefLast.WindowWidth, a.prefLast.WindowHeight
	if !prefsEqual(cur, a.prefLast) {
		a.prefLast = cur
		a.schedulePrefsLocked()
	}
	a.prefMu.Unlock()
	a.prefs = cur
}

func (a *app) schedulePrefsLocked() {
	a.prefDirty = true
	if a.prefTimer != nil {
		a.prefTimer.Stop()
	}
	a.prefTimer = time.AfterFunc(500*time.Millisecond, a.flushPrefs)
}

// flushPrefs writes the preferences now (debounce timer and quit).
func (a *app) flushPrefs() {
	a.prefMu.Lock()
	defer a.prefMu.Unlock()
	if !a.prefDirty {
		return
	}
	a.prefDirty = false
	_ = a.store.Save(a.prefLast)
}

func (a *app) setWindowSize(w, h int) {
	if a.bench || w <= 0 || h <= 0 {
		return
	}
	a.prefMu.Lock()
	if a.prefLast.WindowWidth != w || a.prefLast.WindowHeight != h {
		a.prefLast.WindowWidth, a.prefLast.WindowHeight = w, h
		a.schedulePrefsLocked()
	}
	a.prefMu.Unlock()
	a.prefs.WindowWidth, a.prefs.WindowHeight = w, h
}

// rebuildQuery makes the core query from the toolbar state and starts it.
// reset scrolls to the top and drops the selection unless it is found in
// the first page.
func (a *app) rebuildQuery(reset bool) {
	q := core.Query{Search: strings.TrimSpace(a.search), SortBy: a.sortField, Desc: a.sortDesc}
	for _, c := range core.AllCategories {
		if a.cats[c] {
			q.Categories = append(q.Categories, c)
		}
	}
	for _, s := range core.AllStatuses {
		if a.statusSel == s.String() {
			s := s
			q.Status = &s
		}
	}
	a.query = q
	a.pager.SetQuery(q)
	if reset {
		if a.selID != 0 {
			a.keepSelID = a.selID
		}
		a.selRow = -1
		a.requestScroll(0)
		a.uiQueryChanged()
	}
	a.syncPrefs()
}

// requestScroll makes row the top visible row of the active view.
func (a *app) requestScroll(row int) {
	if row < 0 {
		row = 0
	}
	a.visFirst, a.visLast = row, row+a.visH-1
	a.pager.SetVisible(a.visFirst, a.visLast)
	a.uiScrollTo(row)
}

func (a *app) setMessage(s string, d time.Duration) {
	a.msg = s
	a.uiMessage(d)
}

func (a *app) showError(s string) {
	if a.plat.Alert != nil {
		a.plat.Alert(s)
	}
}

// ---- selection --------------------------------------------------------

// requestSelect selects row, asking first when the form has edits.
func (a *app) requestSelect(row int) {
	if a.f.dirty() {
		if it, ok := a.pager.Row(row); ok && it.ID == a.f.item.ID {
			return
		}
		a.plat.Confirm("GGUIX", "Discard unsaved changes?", "Discard", "Cancel", func() {
			a.f.cur = a.f.snap
			a.selectRow(row)
		}, func() {
			a.uiRestoreSelection()
		})
		return
	}
	a.selectRow(row)
}

// selectRow makes row the selection; the form loads when its data is there.
func (a *app) selectRow(row int) {
	a.selRow = row
	a.selID = 0
	if it, ok := a.pager.Row(row); ok {
		a.selID = it.ID
		a.f.load(it)
	} else {
		// Loaded later by syncForm.
		a.f.clear()
		a.keepSelID = 0
	}
	a.uiSelectionChanged()
	a.syncPrefs()
}

// syncForm loads the form for a selected row whose data just arrived, and
// resolves a selection kept across a query change.
func (a *app) syncForm() {
	changed := false
	if a.keepSelID != 0 {
		total := a.pager.Total()
		if total >= 0 {
			n := min(total, 200)
			if n == 0 || a.pager.RangeLoaded(0, n-1) {
				id := a.keepSelID
				a.keepSelID = 0
				found := false
				for i := 0; i < n; i++ {
					if it, ok := a.pager.Row(i); ok && it.ID == id {
						a.selRow, a.selID = i, id
						if !a.f.has || a.f.item.ID != id {
							a.f.load(it)
						}
						found = true
						break
					}
				}
				if !found {
					a.selRow, a.selID = -1, 0
					a.f.clear()
				}
				changed = true
			}
		}
	}
	if a.selRow >= 0 && (!a.f.has || a.selID == 0) {
		if it, ok := a.pager.Row(a.selRow); ok {
			a.selID = it.ID
			a.f.load(it)
			changed = true
		}
	}
	if a.selRow < 0 && a.keepSelID == 0 && a.f.has && !a.f.dirty() {
		a.f.clear()
		a.selID = 0
		changed = true
	}
	if changed {
		a.uiSelectionChanged()
		a.syncPrefs()
	}
}

// goToRow handles the go-to-row field and the bench driver (1-based n).
func (a *app) goToRow(n int) {
	total := a.pager.Total()
	if total <= 0 {
		return
	}
	a.goToIndex(max(0, min(n-1, total-1)))
}

func (a *app) goToIndex(idx int) {
	a.requestScroll(idx)
	a.requestSelect(idx)
}

func parseRowNumber(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(strings.ReplaceAll(s, ",", ""))
	if err != nil {
		return 0, false
	}
	return n, true
}

// ---- saving -----------------------------------------------------------

func (a *app) save() {
	if !a.f.has {
		return
	}
	it, errs := a.f.toItem()
	if len(errs) > 0 {
		a.f.errs = errs
		a.uiForm()
		return
	}
	got, err := a.svc.Update(context.Background(), it)
	if err != nil {
		if ve, ok := err.(*core.ValidationError); ok {
			errs := map[string]string{}
			for _, fe := range ve.Errors {
				if p, ok := errs[fe.Field]; ok {
					errs[fe.Field] = p + " " + fe.Message
				} else {
					errs[fe.Field] = fe.Message
				}
			}
			a.f.errs = errs
			a.uiForm()
			return
		}
		a.showError(err.Error())
		return
	}
	a.f.load(got)
	a.pager.Invalidate()
	a.uiForm()
	a.uiRefreshData(true)
}

func (a *app) revert() {
	a.plat.Confirm("GGUIX", "Discard unsaved changes?", "Discard", "Cancel", func() {
		a.f.cur = a.f.snap
		a.f.errs = nil
		a.uiForm()
	}, func() {})
}

func (a *app) copyRow() {
	if !a.f.has {
		return
	}
	if a.plat.Clipboard != nil {
		a.plat.Clipboard(core.ItemCSVRow(a.f.item))
	}
	a.setMessage("Row copied", 2*time.Second)
}

// importNotes puts the first 500 characters of a chosen text file into the
// notes field. Files over 64 KB are rejected.
func (a *app) importNotes(data []byte) {
	if !a.f.has {
		return
	}
	if len(data) > 64*1024 {
		a.showError("File is too large.")
		return
	}
	r := []rune(string(data))
	if len(r) > 500 {
		r = r[:500]
	}
	a.f.cur.notes = string(r)
	a.uiForm()
}

func (a *app) beginImport() {
	if !a.f.has || a.plat.OpenTextFile == nil {
		return
	}
	a.plat.OpenTextFile(func(data []byte, ok bool, err error) {
		a.post(func() {
			switch {
			case err != nil:
				a.showError(err.Error())
			case ok:
				a.importNotes(data)
			}
		})
	})
}

// ---- status text ------------------------------------------------------

func (a *app) statusText() string {
	total := a.pager.Total()
	s := fmt.Sprintf("%s items", core.FormatCount(max(total, 0)))
	if total > 0 {
		lo := a.visFirst + 1
		hi := min(a.visLast+1, total)
		if hi < lo {
			hi = lo
		}
		s += fmt.Sprintf(" · rows %d-%d", lo, hi)
	}
	st := a.pager.Stats()
	s += fmt.Sprintf(" · last fetch %d ms · %d pages cached", st.LastFetch.Milliseconds(), st.CachedPages)
	return s
}

func (a *app) identity() string {
	if a.info.Username == "" {
		return "unknown user"
	}
	return a.info.Identity()
}

// ---- view, search, sort, filters, theme -------------------------------

// setView switches between the table (0) and the cards (1) and scrolls the
// new view to the selection (top) or to the top.
func (a *app) setView(v int) {
	if a.viewMode == v {
		return
	}
	a.viewMode = v
	a.uiViewChanged()
	if a.selRow >= 0 {
		a.requestScroll(a.selRow)
	} else {
		a.requestScroll(0)
	}
	a.syncPrefs()
}

// restoreSelection finds the saved selection in the first 2000 rows of the
// restored query, scrolls it to the top and selects it.
func (a *app) restoreSelection() {
	id := a.selID
	if id == 0 || a.bench {
		return
	}
	q := a.query
	go func() {
		row, ok, err := core.FindRow(context.Background(), a.svc, q, id, 2000)
		a.post(func() {
			if err != nil || !ok || a.selID != id {
				if a.selRow < 0 {
					a.selID = 0
				}
				return
			}
			a.requestScroll(row)
			a.selectRow(row)
		})
	}()
}

func (a *app) applySearchNow() {
	a.rebuildQuery(true)
}

// setSearch records typed search text; the UI applies it after the
// debounce (immediately in bench mode).
func (a *app) setSearch(s string) {
	a.search = s
	if a.bench {
		a.applySearchNow()
		return
	}
	a.uiDebounceSearch()
}

// setSort is a header click: a new field sorts ascending, the active one
// toggles direction.
func (a *app) setSort(f core.SortField) {
	if a.sortField == f {
		a.sortDesc = !a.sortDesc
	} else {
		a.sortField, a.sortDesc = f, false
	}
	a.rebuildQuery(true)
}

func (a *app) toggleCategory(c core.Category) {
	a.cats[c] = !a.cats[c]
	a.rebuildQuery(true)
}

func (a *app) clearCategories() {
	a.cats = [8]bool{}
	a.rebuildQuery(true)
}

func (a *app) catCount() int {
	n := 0
	for _, v := range a.cats {
		if v {
			n++
		}
	}
	return n
}

func (a *app) setStatusFilter(s string) {
	a.statusSel = s
	a.rebuildQuery(true)
}

// setTheme records the choice and applies it through IUP's APPEARANCE
// global.
func (a *app) setTheme(s string) {
	a.themeSel = s
	a.uiApplyTheme()
	a.syncPrefs()
}

// moveSelection is the Up/Down key in the cards view (the native table
// moves its own selection).
func (a *app) moveSelection(delta int) {
	total := a.pager.Total()
	if total <= 0 {
		return
	}
	r := a.selRow + delta
	if a.selRow < 0 {
		r = a.visFirst
	}
	r = max(0, min(r, total-1))
	if r == a.selRow {
		return
	}
	if r < a.visFirst || r > a.visLast {
		top := r
		if delta > 0 {
			top = r - (a.visLast - a.visFirst)
		}
		a.requestScroll(max(top, 0))
	}
	a.requestSelect(r)
}

// setVisible records the visible range the active view measured.
func (a *app) setVisible(first, last int) {
	if first == a.visFirst && last == a.visLast {
		return
	}
	a.visFirst, a.visLast = first, last
	a.visH = max(last-first+1, 1)
	a.pager.SetVisible(first, last)
	a.uiStatus()
}
