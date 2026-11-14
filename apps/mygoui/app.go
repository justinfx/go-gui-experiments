package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"

	"gguix/core"
	"gguix/core/host"
)

const appID = "mygoui"

// Names shown in the toolbar dropdowns.
var (
	statusChoices = []string{"All statuses", "Active", "Backordered", "Discontinued"}
	themeChoices  = []string{"System", "Light", "Dark"}
)

// platformHooks is what differs between the desktop and the browser. The
// desktop and web files fill it in; tests replace it.
type platformHooks struct {
	// SaveFile asks where to write the export. ok is false when cancelled.
	SaveFile func(defaultName string) (path string, ok bool, err error)
	// OpenTextFile asks for a .txt file to import. ok is false when cancelled.
	OpenTextFile func() (data []byte, ok bool, err error)
	// Notify tells the user an export finished; it returns false when the
	// system cannot show notifications, and the app shows a toast instead.
	Notify func(title, body string) bool
	// Reveal shows a file or folder in the file manager.
	Reveal func(path string) error
	// Deliver hands the finished export to the user when there is no file
	// dialog (browser download). Nil on the desktop.
	Deliver func(name string, data []byte) error
	// Quit closes the app.
	Quit func()
	// HostActions says whether Reveal works for this target.
	HostActions bool
}

// formFields is everything the detail form edits, comparable so that the
// dirty state is a plain comparison with the snapshot taken on load.
type formFields struct {
	name, price, notes, cat string
	qty                     float64
	status                  core.Status
	rating                  float64
	featured                bool
	hasDate                 bool
	date                    time.Time
}

type form struct {
	item  core.Item // the item as loaded or last saved
	has   bool
	cur   formFields
	snap  formFields
	errs  map[string]string
	saved bool
}

func fieldsOf(it core.Item) formFields {
	f := formFields{
		name:     it.Name,
		price:    core.FormatPrice(it.PriceCents),
		notes:    it.Notes,
		cat:      it.Category.String(),
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

func (f *form) load(it core.Item) {
	f.item, f.has = it, true
	f.cur = fieldsOf(it)
	f.snap = f.cur
	f.errs = nil
}

func (f *form) clear() {
	*f = form{}
}

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
	it.Quantity = int32(f.cur.qty)
	it.Status = f.cur.status
	it.Rating = float32(round1(f.cur.rating))
	it.Featured = f.cur.featured
	if f.cur.hasDate {
		y, m, d := f.cur.date.Date()
		it.RestockDate = &core.Date{Year: y, Month: m, Day: d}
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

// app is the whole application state. Everything but the fields marked
// with mu is touched only on the UI thread.
type app struct {
	svc   core.DataService
	store core.PrefsStore
	pager *core.Pager
	plat  platformHooks
	info  host.Info

	// post runs f on the UI thread and draws a frame. wake draws a frame.
	post func(f func())
	wake func()

	bench bool

	// Query state.
	prefs      core.Prefs
	query      core.Query
	search     string
	searchAt   time.Time // when the pending search applies; zero when none
	cats       [8]bool
	catsOpen   bool
	statusSel  string
	themeSel   string
	viewMode   int // 0 table, 1 cards
	sortField  core.SortField
	sortDesc   bool
	sortDummy  ui.SortOrder
	goTo       string
	focusSrch  bool
	splitPct   float32
	dark       bool
	pal        core.Palette
	themeCache *ui.Theme

	// Lists.
	table ui.ListState
	cards ui.ListState

	// Selection: selRow is the row index in the current result (-1 none).
	selRow       int
	selID        int64
	keepSelID    int64 // selected item to keep once the first page loads
	pendingSel   int   // row waiting for the dirty guard, -1 none
	scrollTo     int   // row to make the top visible row, -1 none
	wantTop      int
	wantFrames   int
	visFirst     int
	visLast      int
	visCount     int
	scrollTopNow bool
	f            form
	confirmOpen  bool
	confirmFn    func()
	confirmNo    func()

	// Dialogs.
	aboutOpen bool
	hostOpen  bool
	errText   string
	errOpen   bool

	// Messages.
	msg      string
	msgUntil time.Time

	// Export.
	exp struct {
		running      bool
		done, total  int
		cancel       context.CancelFunc
		lastPath     string
		doneCallback func(rows int, err error)
	}

	// Prefs saving.
	prefMu    sync.Mutex
	prefLast  core.Prefs
	prefTimer *time.Timer
	prefDirty bool

	menuSync func()
	clipOut  string
	frames   int
	lastView time.Time
}

func newApp(svc core.DataService, store core.PrefsStore, info host.Info, bench bool) *app {
	a := &app{
		svc: svc, store: store, info: info, bench: bench,
		selRow: -1, pendingSel: -1, scrollTo: -1, wantTop: -1,
		visCount: 30, visLast: 29,
		splitPct:  65,
		statusSel: statusChoices[0], themeSel: themeChoices[0],
		post: func(f func()) { f() },
		wake: func() {},
	}
	a.exp.total = 1
	a.pager = core.NewPager(svc, core.PagerOptions{PageSize: 200, MaxPages: 20, MaxInFlight: 2, PrefetchMargin: 50}, func() { a.wake() })
	a.loadPrefs()
	a.rebuildQuery(false)
	return a
}

// loadPrefs restores the saved preferences into the UI state (before the
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
	changed := !prefsEqual(cur, a.prefLast)
	if changed {
		a.prefLast = cur
		a.prefDirty = true
		if a.prefTimer != nil {
			a.prefTimer.Stop()
		}
		a.prefTimer = time.AfterFunc(500*time.Millisecond, a.flushPrefs)
	}
	a.prefMu.Unlock()
	a.prefs = cur
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
	a.prefMu.Lock()
	if a.prefLast.WindowWidth != w || a.prefLast.WindowHeight != h {
		a.prefLast.WindowWidth, a.prefLast.WindowHeight = w, h
		a.prefDirty = true
		if a.prefTimer != nil {
			a.prefTimer.Stop()
		}
		a.prefTimer = time.AfterFunc(500*time.Millisecond, a.flushPrefs)
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
	}
}

// requestScroll makes row the top visible row of the active view.
func (a *app) requestScroll(row int) {
	if row < 0 {
		row = 0
	}
	a.scrollTo = row
	a.wantTop, a.wantFrames = row, 4
	a.visFirst, a.visLast = row, row+a.visCount-1
	a.pager.SetVisible(a.visFirst, a.visLast)
	a.wake()
}

func (a *app) applySearchNow() {
	a.searchAt = time.Time{}
	a.rebuildQuery(true)
}

// activeList is the list state of the shown view.
func (a *app) activeList() *ui.ListState {
	if a.viewMode == 1 {
		return &a.cards
	}
	return &a.table
}

func (a *app) setMessage(s string, d time.Duration) {
	a.msg = s
	a.msgUntil = time.Now().Add(d)
	a.wake()
}

func (a *app) showError(s string) {
	a.errText, a.errOpen = s, true
	a.wake()
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
	a.wake()
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
	a.wake()
}

// syncForm loads the form for a selected row whose data just arrived, and
// resolves a selection kept across a query change.
func (a *app) syncForm() {
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
			}
		}
	}
	if a.selRow >= 0 && (!a.f.has || a.selID == 0) {
		if it, ok := a.pager.Row(a.selRow); ok {
			a.selID = it.ID
			a.f.load(it)
		}
	}
	if a.selRow < 0 && a.keepSelID == 0 && a.f.has && !a.f.dirty() {
		a.f.clear()
		a.selID = 0
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
			return
		}
		a.showError(err.Error())
		return
	}
	a.f.load(got)
	a.pager.Invalidate()
	a.wake()
}

func (a *app) revert() {
	a.askDiscard(func() { a.f.cur = a.f.snap; a.f.errs = nil }, nil)
}

// copyRow queues the saved item's CSV row for the clipboard; the next frame
// writes it (Context.WriteClipboard works in the window and in a browser).
func (a *app) copyRow() {
	if !a.f.has {
		return
	}
	a.clipOut = core.ItemCSVRow(a.f.item)
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
	a.setNotes(string(data))
}

func (a *app) setNotes(s string) {
	r := []rune(s)
	if len(r) > 500 {
		r = r[:500]
	}
	a.f.cur.notes = string(r)
	a.wake()
}

// ---- export -----------------------------------------------------------

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

// setView switches between the table (0) and the cards (1).
func (a *app) setView(v int) {
	if a.viewMode == v {
		return
	}
	a.viewMode = v
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
