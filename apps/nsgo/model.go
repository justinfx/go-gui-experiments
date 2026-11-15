package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nsgo-dev/nsgo/core"

	gx "gguix/core"
	"gguix/core/host"
)

const appID = "nsgo"

// Names shown in the toolbar dropdowns.
var (
	statusChoices = []string{"All statuses", "Active", "Backordered", "Discontinued"}
	themeChoices  = []string{"System", "Light", "Dark"}
)

// rev is a revision counter signal. A view function that reads it re-runs
// when the model calls Bump, which is how this app tells NSGo "this part of
// the model changed". All Bump calls happen on the engine goroutine.
type rev struct{ s *core.Signal[int] }

func newRev(cx *core.Scope) rev { return rev{core.NewSignal(cx, 0)} }

// Watch subscribes the calling binding to the counter.
func (r rev) Watch() { r.s.Get() }

// Bump tells every subscriber the model part changed.
func (r rev) Bump() { r.s.Update(func(n int) int { return n + 1 }) }

// platformHooks is what differs between the desktop and the browser. The
// desktop and web files fill it in; tests replace it.
type platformHooks struct {
	// SaveFile asks where to write the export and calls done on the UI
	// thread; ok is false when cancelled.
	SaveFile func(defaultName string, done func(path string, ok bool, err error))
	// OpenTextFile asks for a .txt file and calls done with its bytes on the
	// UI thread (at most 64 KB + 1 byte are read).
	OpenTextFile func(done func(data []byte, ok bool, err error))
	// Notify tells the user an export finished; it calls shown with false
	// when the system cannot show notifications, and the app shows a
	// message instead.
	Notify func(title, body string, shown func(bool))
	// RevealFile shows a file in the file manager; OpenFolder opens a folder.
	RevealFile func(path string) error
	OpenFolder func(path string) error
	// Stream, when set (web-remote), asks the browser for a writable file
	// and calls done on any goroutine; ok is false when the user cancelled,
	// err is errNoStream when the browser has no save picker (the export then
	// falls back to Deliver).
	Stream func(name string, done func(s *streamSink, ok bool, err error))
	// Deliver hands the finished export to the user when there is no file
	// dialog (browser download). Nil on the desktop.
	Deliver func(name string, data []byte) error
	// Quit closes the app.
	Quit func()
	// HostActions says whether the reveal actions work for this target.
	HostActions bool
}

// streamSink is a writable file the export streams into. Finish closes it
// when the export worked and aborts it (discarding the partial file) when not.
type streamSink struct {
	W      io.Writer
	Finish func(err error)
}

var errNoStream = errors.New("no streaming save in this browser")

// formFields is everything the detail form edits, comparable so that the
// dirty state is a plain comparison with the snapshot taken on load.
type formFields struct {
	name, price, notes string
	cat                gx.Category
	qty                float64
	status             gx.Status
	rating             float64
	featured           bool
	hasDate            bool
	date               time.Time
}

type form struct {
	item gx.Item // the item as loaded or last saved
	has  bool
	cur  formFields
	snap formFields
	errs map[string]string
}

func fieldsOf(it gx.Item) formFields {
	f := formFields{
		name:     it.Name,
		price:    gx.FormatPrice(it.PriceCents),
		notes:    it.Notes,
		cat:      it.Category,
		qty:      float64(it.Quantity),
		status:   it.Status,
		rating:   round1(float64(it.Rating)),
		featured: it.Featured,
	}
	if it.RestockDate != nil {
		d := it.RestockDate
		f.hasDate = true
		f.date = time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
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

func (f *form) load(it gx.Item) {
	f.item, f.has = it, true
	f.cur = fieldsOf(it)
	f.snap = f.cur
	f.errs = nil
}

func (f *form) clear() { *f = form{} }

func (f *form) dirty() bool { return f.has && f.cur != f.snap }

// toItem builds the item the form describes. Parse problems are returned as
// field errors, validation problems by core.ValidateItem.
func (f *form) toItem() (gx.Item, map[string]string) {
	errs := map[string]string{}
	it := f.item
	it.Name = f.cur.name
	it.Notes = f.cur.notes
	it.Category = f.cur.cat
	if cents, err := gx.ParsePrice(f.cur.price); err != nil {
		errs["priceCents"] = err.Error()
	} else {
		it.PriceCents = cents
	}
	q := int64(f.cur.qty + 0.5)
	if q < 0 || q > 1_000_000 {
		errs["quantity"] = "Quantity must be between 0 and 1,000,000."
	} else {
		it.Quantity = int32(q)
	}
	it.Status = f.cur.status
	it.Rating = float32(round1(f.cur.rating))
	it.Featured = f.cur.featured
	if f.cur.hasDate {
		y, m, d := f.cur.date.Date()
		it.RestockDate = &gx.Date{Year: y, Month: m, Day: d}
	} else {
		it.RestockDate = nil
	}
	if ve := gx.ValidateItem(it); ve != nil {
		addErrors(errs, ve)
	}
	return it, errs
}

func addErrors(errs map[string]string, ve *gx.ValidationError) {
	for _, fe := range ve.Errors {
		if prev, ok := errs[fe.Field]; ok {
			errs[fe.Field] = prev + " " + fe.Message
		} else {
			errs[fe.Field] = fe.Message
		}
	}
}

// app is the whole application model. It is touched only on the engine
// goroutine (NSGo's UI thread), except where a field says otherwise.
type app struct {
	cx    *core.Scope
	svc   gx.DataService
	store gx.PrefsStore
	pager *gx.Pager
	plat  platformHooks
	info  host.Info

	bench bool

	// Revision counters. rRows: pager data or paging stats changed.
	// rQuery: toolbar state (search, filters, sort, view, theme).
	// rSel: selection. rForm: form fields. rDlg: dialogs and messages.
	// rExp: export progress. rVis: visible range.
	rRows, rQuery, rSel, rForm, rDlg, rExp, rVis rev

	// rowsDirty is set by the pager's goroutine when it has news the UI
	// thread has not yet turned into a refresh.
	rowsDirty atomic.Bool

	// Query state.
	prefs     gx.Prefs
	query     gx.Query
	search    string
	cats      [8]bool
	statusSel string
	themeSel  string
	viewMode  int // 0 table, 1 cards
	sortField gx.SortField
	sortDesc  bool
	goTo      string
	searchGen int
	stopSrch  func() bool

	// scrollFn makes row the top visible row of the given view (0 table, 1
	// cards). Set by the mounted view; nil when nothing is mounted.
	scrollFn func(view, row int)
	// focusSearchFn focuses the search field.
	focusSearchFn func()
	// wantTop is the row a scroll was asked for and not yet seen reported
	// by the view, -1 for none.
	wantTop   int
	wantTries int
	wantGen   int

	// Selection: selRow is the row index in the current result (-1 none).
	selRow     int
	selID      int64
	keepSelID  int64 // selected item to keep once the first page loads
	pendingSel int   // row waiting for the dirty guard, -1 none
	visFirst   int
	visLast    int
	visCount   int
	f          form

	// Dialogs.
	confirmOpen bool
	confirmFn   func()
	confirmNo   func()
	aboutOpen   bool
	hostOpen    bool
	errText     string
	errOpen     bool
	catsOpen    bool

	// Messages.
	msg      string
	msgUntil time.Time
	msgGen   int

	// Export.
	exp struct {
		running     bool
		done, total int
		cancel      context.CancelFunc
		lastPath    string
	}

	// Prefs saving.
	prefMu    sync.Mutex
	prefLast  gx.Prefs
	prefTimer *time.Timer
	prefDirty bool
}

// newApp builds the model. The revision counters are created by bind.
func newApp(svc gx.DataService, store gx.PrefsStore, info host.Info, bench bool) *app {
	a := &app{
		svc: svc, store: store, info: info, bench: bench,
		selRow: -1, pendingSel: -1, wantTop: -1,
		visCount: 30, visLast: 29,
		statusSel: statusChoices[0], themeSel: themeChoices[0],
	}
	a.exp.total = 1
	a.pager = gx.NewPager(svc, gx.PagerOptions{PageSize: 200, MaxPages: 40, MaxInFlight: 4, PrefetchMargin: 400}, a.pagerChanged)
	a.loadPrefs()
	a.rebuildQuery(false)
	return a
}

// bind attaches the model to the app's root scope and makes its signals.
func (a *app) bind(cx *core.Scope) {
	a.cx = cx
	a.rRows, a.rQuery, a.rSel, a.rForm = newRev(cx), newRev(cx), newRev(cx), newRev(cx)
	a.rDlg, a.rExp, a.rVis = newRev(cx), newRev(cx), newRev(cx)
}

// post runs f on the engine goroutine; safe from any goroutine.
func (a *app) post(f func()) { a.cx.Post(f) }

// pagerChanged is the pager's onChange, called from a background goroutine.
// It coalesces notifications: at most one post is queued at a time.
func (a *app) pagerChanged() {
	if a.cx == nil {
		return
	}
	if a.rowsDirty.Swap(true) {
		return
	}
	a.post(func() {
		a.rowsDirty.Store(false)
		a.rRows.Bump()
		a.syncForm()
	})
}

// loadPrefs restores the saved preferences into the UI state (before the
// first query).
func (a *app) loadPrefs() {
	p, err := a.store.Load()
	if err != nil {
		p = gx.DefaultPrefs()
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
		a.sortField = gx.SortID
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
func (a *app) currentPrefs() gx.Prefs {
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

func prefsEqual(a, b gx.Prefs) bool {
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
	changed := !prefsEqual(cur, a.prefLast)
	if changed {
		a.prefLast = cur
		a.armPrefTimer()
	}
	a.prefMu.Unlock()
	a.prefs = cur
}

// armPrefTimer (prefMu held) marks the prefs dirty and restarts the debounce.
func (a *app) armPrefTimer() {
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
	if a.bench || w < 200 || h < 200 {
		return
	}
	a.prefMu.Lock()
	if a.prefLast.WindowWidth != w || a.prefLast.WindowHeight != h {
		a.prefLast.WindowWidth, a.prefLast.WindowHeight = w, h
		a.armPrefTimer()
	}
	a.prefMu.Unlock()
	a.prefs.WindowWidth, a.prefs.WindowHeight = w, h
}

// rebuildQuery makes the core query from the toolbar state and starts it.
// reset scrolls to the top and drops the selection unless it is found in
// the first page.
func (a *app) rebuildQuery(reset bool) {
	q := gx.Query{Search: strings.TrimSpace(a.search), SortBy: a.sortField, Desc: a.sortDesc}
	for _, c := range gx.AllCategories {
		if a.cats[c] {
			q.Categories = append(q.Categories, c)
		}
	}
	for _, s := range gx.AllStatuses {
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
		a.rSel.Bump()
	}
	if a.cx != nil {
		a.rQuery.Bump()
		a.rRows.Bump()
		a.syncPrefs()
	}
}

// requestScroll makes row the top visible row of the active view.
func (a *app) requestScroll(row int) {
	if row < 0 {
		row = 0
	}
	a.wantTop, a.wantTries = row, 0
	a.wantGen++
	gen := a.wantGen
	if a.cx != nil {
		// A scroll the view never reports (it was already there) must not
		// pull a later user scroll back, so the request expires.
		a.cx.AfterFunc(500*time.Millisecond, func() {
			if a.wantGen == gen {
				a.wantTop = -1
			}
		})
	}
	a.visFirst, a.visLast = row, row+a.visCount-1
	a.pager.SetVisible(a.visFirst, a.visLast)
	if a.scrollFn != nil {
		a.scrollFn(a.viewMode, row)
	}
	if a.cx != nil {
		a.rVis.Bump()
	}
}

func (a *app) applySearchNow() {
	if a.stopSrch != nil {
		a.stopSrch()
		a.stopSrch = nil
	}
	a.rebuildQuery(true)
}

func (a *app) setMessage(s string, d time.Duration) {
	a.msg = s
	a.msgUntil = time.Now().Add(d)
	a.msgGen++
	gen := a.msgGen
	a.rDlg.Bump()
	a.cx.AfterFunc(d, func() {
		if a.msgGen == gen {
			a.msg = ""
			a.rDlg.Bump()
		}
	})
}

func (a *app) showError(s string) {
	a.errText, a.errOpen = s, true
	a.rDlg.Bump()
}

// ---- selection --------------------------------------------------------

// requestSelect selects row, asking first when the form has edits.
func (a *app) requestSelect(row int) {
	if a.f.dirty() {
		if it, ok := a.pager.Row(row); ok && it.ID == a.f.item.ID {
			return
		}
		a.pendingSel = row
		a.askDiscard(func() {
			r := a.pendingSel
			a.pendingSel = -1
			a.f.cur = a.f.snap
			a.selectRow(r)
		}, func() { a.pendingSel = -1 })
		return
	}
	a.selectRow(row)
}

func (a *app) askDiscard(ok func(), cancel func()) {
	a.confirmOpen = true
	a.confirmFn = ok
	a.confirmNo = cancel
	a.rDlg.Bump()
}

// confirmAnswer closes the confirm dialog, running the matching callback.
func (a *app) confirmAnswer(discard bool) {
	if !a.confirmOpen {
		return
	}
	ok, no := a.confirmFn, a.confirmNo
	a.confirmOpen, a.confirmFn, a.confirmNo = false, nil, nil
	a.rDlg.Bump()
	if discard && ok != nil {
		ok()
	} else if !discard && no != nil {
		no()
	}
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
	a.rSel.Bump()
	a.rForm.Bump()
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
		a.rSel.Bump()
		a.rForm.Bump()
		a.syncPrefs()
	}
}

// goToRow handles the go-to-row field and the bench driver.
func (a *app) goToRow(n int) {
	total := a.pager.Total()
	if total <= 0 {
		return
	}
	idx := max(0, min(n-1, total-1))
	a.goToIndex(idx)
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
		a.rForm.Bump()
		return
	}
	got, err := a.svc.Update(context.Background(), it)
	if err != nil {
		if ve, ok := err.(*gx.ValidationError); ok {
			errs := map[string]string{}
			addErrors(errs, ve)
			a.f.errs = errs
			a.rForm.Bump()
			return
		}
		a.showError(err.Error())
		return
	}
	a.f.load(got)
	a.pager.Invalidate()
	a.rForm.Bump()
	a.rRows.Bump()
}

func (a *app) revert() {
	a.askDiscard(func() {
		a.f.cur = a.f.snap
		a.f.errs = nil
		a.rForm.Bump()
	}, nil)
}

// copyRow puts the saved item's CSV row on the clipboard.
func (a *app) copyRow() {
	if !a.f.has {
		return
	}
	a.cx.SetClipboard(gx.ItemCSVRow(a.f.item))
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
	a.rForm.Bump()
}

// beginImport is the Import Notes action.
func (a *app) beginImport() {
	if !a.f.has || a.plat.OpenTextFile == nil {
		return
	}
	a.plat.OpenTextFile(func(data []byte, ok bool, err error) {
		switch {
		case err != nil:
			a.showError(err.Error())
		case ok:
			a.importNotes(data)
		}
	})
}

// ---- status ----------------------------------------------------------

func (a *app) statusText() string {
	total := a.pager.Total()
	s := fmt.Sprintf("%s items", gx.FormatCount(max(total, 0)))
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

// setView switches between the table (0) and the cards (1).
func (a *app) setView(v int) {
	if a.viewMode == v {
		return
	}
	a.viewMode = v
	a.rQuery.Bump()
	a.syncPrefs()
	a.viewChanged()
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
		row, ok, err := gx.FindRow(context.Background(), a.svc, q, id, 2000)
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

// viewChanged scrolls the new view to the selection (top) or to the top.
func (a *app) viewChanged() {
	if a.selRow >= 0 {
		a.requestScroll(a.selRow)
	} else {
		a.requestScroll(0)
	}
}

// setSearch records typed search text and applies it 250 ms after the
// last keystroke (no debounce in bench mode).
func (a *app) setSearch(s string) {
	a.search = s
	a.rQuery.Bump()
	if a.bench {
		a.applySearchNow()
		return
	}
	if a.stopSrch != nil {
		a.stopSrch()
	}
	a.stopSrch = a.cx.AfterFunc(250*time.Millisecond, func() {
		a.stopSrch = nil
		a.rebuildQuery(true)
	})
}

// setSort is a header click: a new field sorts ascending, the active one
// toggles direction.
func (a *app) setSort(f gx.SortField) {
	if a.sortField == f {
		a.sortDesc = !a.sortDesc
	} else {
		a.sortField, a.sortDesc = f, false
	}
	a.rebuildQuery(true)
}

func (a *app) toggleCategory(c gx.Category) {
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

func (a *app) setTheme(s string) {
	a.themeSel = s
	a.cx.SetAppearance(appearanceOf(s))
	a.rQuery.Bump()
	a.syncPrefs()
}

func appearanceOf(s string) core.Appearance {
	switch s {
	case "Dark":
		return core.AppearanceDark
	case "Light":
		return core.AppearanceLight
	}
	return core.AppearanceSystem
}

// setVisible records the visible range the view measured (inclusive).
func (a *app) setVisible(first, last int) {
	if last < first {
		last = first
	}
	// While a scroll the app asked for is not yet seen in the view, keep the
	// asked-for range; the view reports its old position first.
	if a.wantTop >= 0 {
		if first == a.wantTop {
			a.wantTop = -1
		} else if a.wantTries < 3 {
			a.wantTries++
			if a.scrollFn != nil {
				a.scrollFn(a.viewMode, a.wantTop)
			}
			return
		} else {
			a.wantTop = -1
		}
	}
	if first == a.visFirst && last == a.visLast {
		return
	}
	a.visFirst, a.visLast = first, last
	a.visCount = max(last-first+1, 1)
	a.pager.SetVisible(first, last)
	a.rVis.Bump()
}

// ---- export ----------------------------------------------------------

const exportName = "gguix-export.csv"

func (a *app) exportPercent() int {
	if a.exp.total <= 0 {
		return 0
	}
	return min(100, a.exp.done*100/a.exp.total)
}
