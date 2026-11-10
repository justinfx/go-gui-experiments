package main

import (
	"context"
	"fmt"
	"image/color"
	"io"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"gguix/core"
	"gguix/core/host"
)

const (
	appID          = "fyne"
	searchDebounce = 250 * time.Millisecond
	prefsDebounce  = 500 * time.Millisecond
	flashDuration  = 2 * time.Second
	minWinW        = 1024
	minWinH        = 640
)

// deps are the services the UI is built on.
type deps struct {
	svc   core.DataService
	store core.PrefsStore
	bench bool
}

// shownState records what the last view refresh displayed (for the bench
// "loaded" check and for tests).
type shownState struct {
	first, last int
	total       int
	ok          bool
}

// ui is the whole application window.
type ui struct {
	app   fyne.App
	win   fyne.Window
	svc   core.DataService
	store core.PrefsStore
	pager *rowStore
	bench bool

	// query state
	search   string
	cats     []core.Category
	status   *core.Status
	sortBy   core.SortField
	sortDesc bool
	cards    bool
	themeMod string

	// selection
	selID      int64
	selRow     int
	pendingIdx int
	awaitFirst bool

	// visible range reported to the pager
	visFirst, visLast int
	visSet            bool
	shown             shownState
	refreshPending    atomic.Bool
	viewStale         atomic.Bool // visible rows may differ from what is drawn

	// widgets
	searchEntry *widget.Entry
	catBtn      *widget.Button
	statusSel   *widget.Select
	gotoEntry   *widget.Entry
	tableBtn    *widget.Button
	cardsBtn    *widget.Button
	exportBtn   *widget.Button
	themeSel    *widget.Select
	table       *gridTable
	list        *cardList
	viewStack   *fyne.Container
	emptyLbl    fyne.CanvasObject
	form        *detailForm
	statusLbl   *widget.Label
	flashLbl    *widget.Label
	progress    *widget.ProgressBar
	pctLbl      *widget.Label
	cancelBtn   *widget.Button
	exportBox   *fyne.Container
	identityLbl *widget.Label
	statusBar   *fyne.Container
	hostBtn     *widget.Button
	root        fyne.CanvasObject

	menus *menuItems // desktop menu bar state; nil on the web

	runOnUI func(func()) // test hook; nil means fyne.Do

	suppress bool // programmatic widget changes do not count as user input

	// timers
	mu          sync.Mutex
	searchTimer *time.Timer
	prefsTimer  *time.Timer
	flashTimer  *time.Timer

	// export
	exporting    bool
	exportCancel context.CancelFunc
	lastExport   string

	// host
	hostMu sync.Mutex
	host   host.Info
}

// newUI builds the window content. It does not start any loading.
func newUI(a fyne.App, w fyne.Window, d deps) *ui {
	u := &ui{app: a, win: w, svc: d.svc, store: d.store, bench: d.bench, selRow: -1, pendingIdx: -1}
	u.suppress = true // widget callbacks fired while building are not user input
	defer func() { u.suppress = false }()
	u.pager = newRowStore(d.svc, u.pagerChanged)

	u.form = newDetailForm(u)
	u.table = newGridTable(u)
	u.list = newCardList(u)
	u.list.Hide()
	u.emptyLbl = container.NewCenter(widget.NewLabel("No items match."))
	u.emptyLbl.Hide()
	u.viewStack = container.NewStack(u.table, u.list, u.emptyLbl)

	split := container.NewHSplit(u.viewStack, u.form.root)
	split.SetOffset(0.65)

	u.root = container.NewBorder(u.buildToolbar(), u.buildStatusBar(), nil, nil, split)
	w.SetContent(container.New(&minSizeLayout{min: fyne.NewSize(minWinW, minWinH)}, u.root))
	w.SetTitle("GGUIX - Fyne")
	return u
}

// minSizeLayout stretches its children and makes sure the window cannot be
// resized below min.
type minSizeLayout struct{ min fyne.Size }

func (l *minSizeLayout) Layout(objs []fyne.CanvasObject, s fyne.Size) {
	for _, o := range objs {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(s)
	}
}

func (l *minSizeLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	m := l.min
	for _, o := range objs {
		m = m.Max(o.MinSize())
	}
	return m
}

// fixedWidth gives its single child a fixed width.
type fixedWidth struct{ w float32 }

func (l *fixedWidth) Layout(objs []fyne.CanvasObject, s fyne.Size) {
	for _, o := range objs {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(s)
	}
}

func (l *fixedWidth) MinSize(objs []fyne.CanvasObject) fyne.Size {
	h := float32(0)
	for _, o := range objs {
		h = maxf(h, o.MinSize().Height)
	}
	return fyne.NewSize(l.w, h)
}

// do runs f on the UI thread using Fyne's documented mechanism, fyne.Do.
// Tests replace it with a queue they drain themselves.
func (u *ui) do(f func()) {
	if u.runOnUI != nil {
		u.runOnUI(f)
		return
	}
	fyne.Do(f)
}

// ---------------------------------------------------------------- building

func (u *ui) buildToolbar() fyne.CanvasObject {
	u.searchEntry = widget.NewEntry()
	u.searchEntry.SetPlaceHolder("Search name or SKU")
	u.searchEntry.OnChanged = u.onSearchTyped

	u.catBtn = widget.NewButton("Categories", u.showCategoryPopup)

	u.statusSel = widget.NewSelect([]string{"All statuses", "Active", "Backordered", "Discontinued"}, u.onStatusPicked)
	u.statusSel.SetSelected("All statuses")

	u.gotoEntry = widget.NewEntry()
	u.gotoEntry.SetPlaceHolder("Go to row")
	u.gotoEntry.OnSubmitted = func(string) { u.goToEntered() }
	goBtn := widget.NewButton("Go", u.goToEntered)

	u.tableBtn = widget.NewButton("Table", func() { u.setView(false) })
	u.cardsBtn = widget.NewButton("Cards", func() { u.setView(true) })

	u.exportBtn = widget.NewButton("Export CSV", func() { u.startExport() })

	u.themeSel = widget.NewSelect([]string{"System", "Light", "Dark"}, u.onThemePicked)
	u.themeSel.SetSelected("System")

	return container.NewHBox(
		container.New(&fixedWidth{w: 280}, u.searchEntry),
		u.catBtn,
		u.statusSel,
		container.New(&fixedWidth{w: 110}, u.gotoEntry), goBtn,
		container.NewHBox(u.tableBtn, u.cardsBtn),
		layout.NewSpacer(),
		u.exportBtn,
		u.themeSel,
	)
}

func (u *ui) buildStatusBar() fyne.CanvasObject {
	u.statusLbl = widget.NewLabel("Loading…")
	u.flashLbl = widget.NewLabel("")
	u.progress = widget.NewProgressBar()
	u.progress.TextFormatter = func() string { return "" }
	u.pctLbl = widget.NewLabel("0%")
	u.cancelBtn = widget.NewButton("Cancel", func() { u.cancelExport() })
	u.exportBox = container.NewHBox(container.New(&fixedWidth{w: 200}, u.progress), u.pctLbl, u.cancelBtn)
	u.exportBox.Hide()
	u.identityLbl = widget.NewLabel("")
	u.hostBtn = widget.NewButton("Host", u.showHostDialog)
	u.statusBar = container.NewHBox(u.statusLbl, u.flashLbl, layout.NewSpacer(), u.exportBox, u.identityLbl, u.hostBtn)
	return container.NewVBox(widget.NewSeparator(), u.statusBar)
}

// ---------------------------------------------------------------- start up

// start restores the saved preferences, runs the first query and starts
// loading host information.
func (u *ui) start() {
	p, err := u.store.Load()
	if err != nil {
		p = core.DefaultPrefs()
	}
	p = p.Sanitize()
	if u.bench {
		p.SelectedID = 0
	}
	u.search, u.cats, u.status = p.Search, p.Categories, p.Status
	u.sortBy, u.sortDesc = p.SortBy, p.SortDesc
	u.themeMod = p.Theme
	u.cards = p.View == core.ViewCards

	u.suppress = true
	u.searchEntry.SetText(u.search)
	u.statusSel.SetSelected(statusLabel(u.status))
	u.themeSel.SetSelected(themeLabel(u.themeMod))
	u.suppress = false
	u.updateCatButton()
	u.applyTheme(u.themeMod)
	u.applyViewVisibility()

	u.platformInit(p)

	q := u.currentQuery()
	u.pager.SetQuery(q)
	u.setVisible(0, 20)
	u.loadHost()
	u.updateMenus()

	if p.SelectedID > 0 {
		id := p.SelectedID
		key := q.Key()
		go func() {
			row, ok, err := core.FindRow(context.Background(), u.svc, q, id, 2000)
			if err != nil || !ok {
				return
			}
			u.do(func() {
				if u.pager.Query().Key() != key {
					return
				}
				u.goToRow(row)
			})
		}()
	}
}

func statusLabel(s *core.Status) string {
	if s == nil {
		return "All statuses"
	}
	return s.String()
}

func themeLabel(mode string) string {
	switch mode {
	case core.ThemeLight:
		return "Light"
	case core.ThemeDark:
		return "Dark"
	}
	return "System"
}

// ---------------------------------------------------------------- query

func (u *ui) currentQuery() core.Query {
	return core.Query{Search: u.search, Categories: u.cats, Status: u.status, SortBy: u.sortBy, Desc: u.sortDesc}.Normalize()
}

// queryChanged applies the query state: it reloads from the first page and
// scrolls to the top.
func (u *ui) queryChanged() {
	u.pendingIdx = -1
	u.selRow = -1
	u.awaitFirst = u.selID != 0
	u.pager.SetQuery(u.currentQuery())
	u.scrollToTop()
	u.visSet = false
	u.setVisible(0, u.viewRows()-1)
	u.table.Refresh()
	u.list.Refresh()
	u.emptyLbl.Hide()
	u.updateStatus()
	u.updateMenus()
	u.savePrefsSoon()
}

func (u *ui) viewRows() int {
	if u.cards {
		f, l := u.list.visibleRange()
		return l - f + 1
	}
	f, l := u.table.visibleRange()
	return l - f + 1
}

func (u *ui) scrollToTop() {
	u.table.ScrollToTop()
	u.list.ScrollToTop()
}

func (u *ui) onSearchTyped(text string) {
	if u.suppress {
		return
	}
	u.mu.Lock()
	if u.searchTimer != nil {
		u.searchTimer.Stop()
	}
	u.searchTimer = time.AfterFunc(searchDebounce, func() {
		u.do(func() { u.applySearch(text) })
	})
	u.mu.Unlock()
}

func (u *ui) applySearch(text string) {
	if strings.TrimSpace(text) == strings.TrimSpace(u.search) && text == u.search {
		return
	}
	u.search = text
	u.queryChanged()
}

func (u *ui) onStatusPicked(s string) {
	if u.suppress {
		return
	}
	if st, err := core.ParseStatus(s); err == nil {
		u.status = &st
	} else {
		u.status = nil
	}
	u.queryChanged()
}

func (u *ui) updateCatButton() {
	if n := len(u.cats); n > 0 {
		u.catBtn.SetText(fmt.Sprintf("Categories (%d)", n))
	} else {
		u.catBtn.SetText("Categories")
	}
}

func (u *ui) hasCat(c core.Category) bool {
	for _, x := range u.cats {
		if x == c {
			return true
		}
	}
	return false
}

func (u *ui) showCategoryPopup() {
	checks := make([]*widget.Check, 0, len(core.AllCategories))
	box := container.NewVBox()
	for _, c := range core.AllCategories {
		chk := widget.NewCheck(c.String(), func(on bool) {
			if u.suppress {
				return
			}
			u.setCategory(c, on)
		})
		u.suppress = true
		chk.SetChecked(u.hasCat(c))
		u.suppress = false
		checks = append(checks, chk)
		box.Add(chk)
	}
	clear := widget.NewButton("Clear", func() {
		u.suppress = true
		for _, chk := range checks {
			chk.SetChecked(false)
		}
		u.suppress = false
		u.cats = nil
		u.updateCatButton()
		u.queryChanged()
	})
	pop := widget.NewPopUp(container.NewVBox(box, widget.NewSeparator(), clear), u.win.Canvas())
	pop.ShowAtRelativePosition(fyne.NewPos(0, u.catBtn.Size().Height), u.catBtn)
}

func (u *ui) setCategory(c core.Category, on bool) {
	var out []core.Category
	for _, x := range core.AllCategories {
		if x == c {
			if on {
				out = append(out, x)
			}
		} else if u.hasCat(x) {
			out = append(out, x)
		}
	}
	u.cats = out
	u.updateCatButton()
	u.queryChanged()
}

// sortByColumn is called when a table header is tapped.
func (u *ui) sortByColumn(col int) {
	if col < 0 || col >= len(columns) {
		return
	}
	f := columns[col].sort
	if f == u.sortBy {
		u.sortDesc = !u.sortDesc
	} else {
		u.sortBy, u.sortDesc = f, false
	}
	u.queryChanged()
	u.table.Refresh()
}

func (u *ui) updateTableHeader(id widget.TableCellID, o fyne.CanvasObject) {
	h, ok := o.(*headerCell)
	if !ok || id.Col < 0 || id.Col >= len(columns) {
		return
	}
	c := columns[id.Col]
	text := c.title
	if c.sort == u.sortBy {
		if u.sortDesc {
			text += " ▼"
		} else {
			text += " ▲"
		}
	}
	h.col = id.Col
	h.Alignment = c.align
	if h.Text != text {
		h.SetText(text)
	}
}

// ---------------------------------------------------------------- views

func (u *ui) applyViewVisibility() {
	if u.cards {
		u.table.Hide()
		u.list.Show()
		u.tableBtn.Importance = widget.MediumImportance
		u.cardsBtn.Importance = widget.HighImportance
	} else {
		u.list.Hide()
		u.table.Show()
		u.tableBtn.Importance = widget.HighImportance
		u.cardsBtn.Importance = widget.MediumImportance
	}
	u.tableBtn.Refresh()
	u.cardsBtn.Refresh()
}

// setView switches between the table and the cards view. The new view is
// scrolled to the selected item (or the top).
func (u *ui) setView(cards bool) {
	if cards == u.cards {
		return
	}
	u.cards = cards
	u.applyViewVisibility()
	u.viewStack.Refresh()
	if u.selRow >= 0 {
		u.scrollToRow(u.selRow)
	} else {
		u.scrollToTop()
	}
	u.visSet = false
	u.syncVisible()
	u.refreshViews()
	u.updateMenus()
	u.savePrefsSoon()
}

func (u *ui) scrollToRow(row int) {
	if u.cards {
		u.list.scrollToRow(row)
	} else {
		u.table.scrollToRow(row)
	}
	u.visSet = false
	u.syncVisible()
}

func (u *ui) ensureVisible(row int) {
	if u.cards {
		u.list.ensureVisible(row)
	} else {
		u.table.ensureVisible(row)
	}
	u.syncVisible()
}

// syncVisible reads the visible row range of the active view and tells the
// pager when it changed.
func (u *ui) syncVisible() {
	var f, l int
	if u.cards {
		f, l = u.list.visibleRange()
	} else {
		f, l = u.table.visibleRange()
	}
	if u.visSet && f == u.visFirst && l == u.visLast {
		return
	}
	u.setVisible(f, l)
}

// syncIfOutside is the per-cell hook: a cell drawn outside the known visible
// range means the view scrolled, so the range is re-read. Cells inside the
// range skip the offset lookup.
func (u *ui) syncIfOutside(row int) {
	if u.visSet && row >= u.visFirst && row <= u.visLast {
		return
	}
	u.syncVisible()
}

func (u *ui) setVisible(first, last int) {
	u.visFirst, u.visLast, u.visSet = first, last, true
	u.pager.SetVisible(first, last)
	u.updateStatus()
}

func (u *ui) updateTableCell(id widget.TableCellID, o fyne.CanvasObject) {
	c, ok := o.(*tableCell)
	if !ok {
		return
	}
	it, loaded := u.pager.Row(id.Row)
	if loaded {
		c.Set(id.Row, id.Col, &it, u.selID != 0 && it.ID == u.selID)
	} else {
		c.Set(id.Row, id.Col, nil, false)
		u.viewStale.Store(true)
	}
	u.syncIfOutside(id.Row)
}

func (u *ui) updateCard(id widget.ListItemID, o fyne.CanvasObject) {
	c, ok := o.(*cardItem)
	if !ok {
		return
	}
	it, loaded := u.pager.Row(id)
	if loaded {
		c.Set(&it, u.selID != 0 && it.ID == u.selID)
	} else {
		c.Set(nil, false)
		u.viewStale.Store(true)
	}
	u.syncIfOutside(id)
}

// pagerChanged is the row store's change callback; it runs on a background
// goroutine (already coalesced to one call per frame) and hands over to the
// UI thread. Changes outside the visible range only refresh the status bar.
func (u *ui) pagerChanged(c storeChange) {
	if c.Visible {
		u.viewStale.Store(true)
	}
	if !u.refreshPending.CompareAndSwap(false, true) {
		return
	}
	u.do(func() {
		u.refreshPending.Store(false)
		u.afterPagerChange()
	})
}

func (u *ui) afterPagerChange() {
	if u.viewStale.Load() {
		u.resolveFirstPage()
		u.resolvePending()
		u.refreshViews()
	}
	u.updateStatus()
}

// resolveFirstPage keeps the selection after a query change only when the
// selected item is in the first page.
func (u *ui) resolveFirstPage() {
	if !u.awaitFirst {
		return
	}
	total := u.pager.Total()
	if total < 0 {
		return
	}
	n := min(total, u.pager.PageSize())
	if !u.pager.RangeLoaded(0, n-1) {
		return
	}
	u.awaitFirst = false
	for i := 0; i < n; i++ {
		if it, ok := u.pager.Row(i); ok && it.ID == u.selID {
			u.selRow = i
			return
		}
	}
	if u.form.dirty() {
		return // keep the edits; only the highlight is gone
	}
	u.clearSelection()
}

func (u *ui) clearSelection() {
	u.selID, u.selRow, u.pendingIdx = 0, -1, -1
	u.form.Clear()
	u.savePrefsSoon()
	u.updateMenus()
}

func (u *ui) resolvePending() {
	if u.pendingIdx < 0 {
		return
	}
	if it, ok := u.pager.Row(u.pendingIdx); ok {
		i := u.pendingIdx
		u.pendingIdx = -1
		u.applySelection(i, it)
	}
}

// refreshViews redraws the active view with the pager's current rows.
func (u *ui) refreshViews() {
	u.viewStale.Store(false)
	total := u.pager.Total()
	if u.cards {
		u.list.Refresh()
	} else {
		u.table.Refresh()
	}
	if total == 0 {
		u.emptyLbl.Show()
	} else {
		u.emptyLbl.Hide()
	}
	u.shown = shownState{first: u.visFirst, last: u.visLast, total: total, ok: total >= 0 && u.pager.RangeLoaded(u.visFirst, u.visLast)}
}

// refreshIfStale is refreshViews for the scroll path: when the rows were all
// loaded while the cells were drawn and nothing visible changed since, the
// toolkit's own scroll update already drew the right data, so the full
// Refresh is skipped and only the shown state is recorded.
func (u *ui) refreshIfStale() {
	if u.viewStale.Load() {
		u.refreshViews()
		return
	}
	total := u.pager.Total()
	u.shown = shownState{first: u.visFirst, last: u.visLast, total: total, ok: total >= 0 && u.pager.RangeLoaded(u.visFirst, u.visLast)}
}

// VisibleLoaded reports that every visible row has data and has been drawn.
func (u *ui) VisibleLoaded() bool {
	s := u.shown
	return s.ok && s.first == u.visFirst && s.last == u.visLast && s.total == u.pager.Total() &&
		u.pager.RangeLoaded(u.visFirst, u.visLast)
}

func (u *ui) updateStatus() {
	total := u.pager.Total()
	var sb strings.Builder
	if total < 0 {
		sb.WriteString("Loading…")
	} else {
		sb.WriteString(core.FormatCount(total) + " items")
		if total > 0 {
			a := min(u.visFirst, total-1) + 1
			b := min(u.visLast, total-1) + 1
			sb.WriteString(fmt.Sprintf(" · rows %d-%d", a, b))
		}
	}
	st := u.pager.Stats()
	sb.WriteString(fmt.Sprintf(" · last fetch %d ms · %d pages cached", st.LastFetch.Milliseconds(), st.CachedPages))
	if s := sb.String(); s != u.statusLbl.Text {
		u.statusLbl.SetText(s)
	}
}

// StatusText is the left part of the status bar.
func (u *ui) StatusText() string { return u.statusLbl.Text }

func (u *ui) flash(msg string) {
	u.flashLbl.SetText(msg)
	u.mu.Lock()
	if u.flashTimer != nil {
		u.flashTimer.Stop()
	}
	u.flashTimer = time.AfterFunc(flashDuration, func() {
		u.do(func() {
			if u.flashLbl.Text == msg {
				u.flashLbl.SetText("")
			}
		})
	})
	u.mu.Unlock()
}

// ---------------------------------------------------------------- selection

func (u *ui) total() int { return max(u.pager.Total(), 0) }

// confirmDiscard runs onOK at once when the form is clean, otherwise after
// the user confirms discarding the edits.
func (u *ui) confirmDiscard(onOK func()) {
	d := dialog.NewConfirm("Discard unsaved changes?", "Your edits to this item will be lost.", func(ok bool) {
		if ok {
			onOK()
		}
	}, u.win)
	d.SetConfirmText("Discard")
	d.SetDismissText("Cancel")
	d.Show()
}

func (u *ui) guarded(f func()) {
	if u.form.dirty() {
		u.confirmDiscard(f)
		return
	}
	f()
}

// selectIndex selects row i of the result (user action, guarded when the form
// has unsaved edits).
func (u *ui) selectIndex(i int) {
	if i < 0 || i >= u.total() {
		return
	}
	if it, ok := u.pager.Row(i); ok && it.ID == u.selID && u.form.has {
		return
	}
	u.guarded(func() { u.doSelect(i) })
}

func (u *ui) doSelect(i int) {
	if it, ok := u.pager.Row(i); ok {
		u.pendingIdx = -1
		u.applySelection(i, it)
		return
	}
	u.pendingIdx = i
}

func (u *ui) applySelection(i int, it core.Item) {
	u.selRow, u.selID = i, it.ID
	u.form.Load(it)
	u.refreshViews()
	u.savePrefsSoon()
	u.updateMenus()
}

func (u *ui) moveSelection(delta int) {
	total := u.total()
	if total == 0 {
		return
	}
	cur := u.selRow
	if cur < 0 {
		cur = u.visFirst
		if delta > 0 {
			cur--
		} else {
			cur++
		}
	}
	next := max(0, min(cur+delta, total-1))
	u.ensureVisible(next)
	u.selectIndex(next)
}

// goToRow scrolls so that row index i is the top row and selects it.
func (u *ui) goToRow(i int) {
	total := u.total()
	if total == 0 {
		return
	}
	i = max(0, min(i, total-1))
	u.guarded(func() {
		u.scrollToRow(i)
		u.doSelect(i)
		u.refreshViews()
	})
}

func (u *ui) goToEntered() {
	n, err := strconv.Atoi(strings.TrimSpace(strings.ReplaceAll(u.gotoEntry.Text, ",", "")))
	if err != nil {
		return
	}
	u.goToRow(n - 1)
}

// saveItem stores an edited item through the data service.
func (u *ui) saveItem(it core.Item, done func(core.Item, error)) {
	go func() {
		saved, err := u.svc.Update(context.Background(), it)
		u.do(func() {
			if err == nil {
				u.pager.Invalidate()
			}
			done(saved, err)
		})
	}()
}

func (u *ui) copyRow() {
	if !u.form.has {
		return
	}
	u.win.Clipboard().SetContent(core.ItemCSVRow(u.form.item))
	u.flash("Row copied")
}

func (u *ui) showError(err error) {
	dialog.ShowError(err, u.win)
}

// ---------------------------------------------------------------- theme

// forcedTheme is the default theme pinned to one variant.
type forcedTheme struct {
	fyne.Theme
	variant fyne.ThemeVariant
}

func (t forcedTheme) Color(n fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return t.Theme.Color(n, t.variant)
}

func (u *ui) onThemePicked(s string) {
	if u.suppress {
		return
	}
	mode := core.ThemeSystem
	switch s {
	case "Light":
		mode = core.ThemeLight
	case "Dark":
		mode = core.ThemeDark
	}
	u.setThemeMode(mode)
}

// setThemeMode applies a theme mode and keeps the dropdown and menu in sync.
func (u *ui) setThemeMode(mode string) {
	u.themeMod = mode
	u.applyTheme(mode)
	u.suppress = true
	u.themeSel.SetSelected(themeLabel(mode))
	u.suppress = false
	u.updateMenus()
	u.savePrefsSoon()
}

func (u *ui) applyTheme(mode string) {
	s := u.app.Settings()
	switch mode {
	case core.ThemeLight:
		s.SetTheme(forcedTheme{theme.DefaultTheme(), theme.VariantLight})
	case core.ThemeDark:
		s.SetTheme(forcedTheme{theme.DefaultTheme(), theme.VariantDark})
	default:
		s.SetTheme(theme.DefaultTheme())
	}
}

// ---------------------------------------------------------------- prefs

func (u *ui) snapshot() core.Prefs {
	p := core.Prefs{
		Theme:        u.themeMod,
		SortBy:       u.sortBy,
		SortDesc:     u.sortDesc,
		Search:       u.search,
		Categories:   u.cats,
		Status:       u.status,
		SelectedID:   u.selID,
		View:         core.ViewTable,
		WindowWidth:  core.DefaultPrefs().WindowWidth,
		WindowHeight: core.DefaultPrefs().WindowHeight,
	}
	if u.cards {
		p.View = core.ViewCards
	}
	if w, h, ok := u.windowSize(); ok {
		p.WindowWidth, p.WindowHeight = w, h
	}
	return p
}

func (u *ui) savePrefsSoon() {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.prefsTimer != nil {
		u.prefsTimer.Stop()
	}
	u.prefsTimer = time.AfterFunc(prefsDebounce, func() {
		u.do(func() {
			p := u.snapshot()
			go func() { _ = u.store.Save(p) }()
		})
	})
}

// savePrefsNow saves synchronously (used on quit).
func (u *ui) savePrefsNow() {
	u.mu.Lock()
	if u.prefsTimer != nil {
		u.prefsTimer.Stop()
	}
	u.mu.Unlock()
	_ = u.store.Save(u.snapshot())
}

// ---------------------------------------------------------------- export

func (u *ui) setExporting(on bool) {
	u.exporting = on
	if on {
		u.exportBtn.Disable()
		u.progress.SetValue(0)
		u.pctLbl.SetText("0%")
		u.exportBox.Show()
	} else {
		u.exportBtn.Enable()
		u.exportBox.Hide()
	}
	u.updateMenus()
}

// runExport exports the current query into w in the background. finish runs
// on the UI thread when the export has ended; err is context.Canceled after
// a cancel.
func (u *ui) runExport(w io.Writer, finish func(rows int, err error)) {
	ctx, cancel := context.WithCancel(context.Background())
	u.exportCancel = cancel
	u.setExporting(true)
	q := u.pager.Query()
	go func() {
		rows, err := u.svc.Export(ctx, q, w, func(done, total int) {
			u.do(func() { u.setProgress(done, total) })
		})
		if err == nil && ctx.Err() != nil {
			err = ctx.Err()
		}
		u.do(func() {
			cancel()
			u.exportCancel = nil
			u.setExporting(false)
			finish(rows, err)
		})
	}()
}

func (u *ui) setProgress(done, total int) {
	if !u.exporting || total <= 0 {
		return
	}
	f := float64(done) / float64(total)
	u.progress.SetValue(f)
	u.pctLbl.SetText(fmt.Sprintf("%d%%", int(f*100)))
}

func (u *ui) cancelExport() {
	if u.exportCancel != nil {
		u.exportCancel()
	}
}

// ---------------------------------------------------------------- host / about

func (u *ui) hostInfo() host.Info {
	u.hostMu.Lock()
	defer u.hostMu.Unlock()
	return u.host
}

func (u *ui) setHostInfo(info host.Info) {
	u.hostMu.Lock()
	u.host = info
	u.hostMu.Unlock()
	u.identityLbl.SetText(info.Identity())
	u.statusBar.Refresh()
}

const hostUnavailableMsg = "Not available in the browser"

func (u *ui) showHostDialog() {
	info := u.hostInfo()
	form := container.New(layout.NewFormLayout())
	for _, r := range info.Rows() {
		form.Add(widget.NewLabelWithStyle(r.Label, fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}))
		form.Add(widget.NewLabel(r.Value))
	}
	items := []fyne.CanvasObject{form}
	if len(info.Notes) > 0 {
		notes := container.NewVBox()
		for _, n := range info.Notes {
			l := widget.NewLabel(n)
			l.Wrapping = fyne.TextWrapWord
			notes.Add(l)
		}
		items = append(items, notes)
	}
	cfg := widget.NewButton("Reveal Config Folder", func() { u.revealConfigFolder(info) })
	last := widget.NewButton("Reveal Last Export", func() { u.revealLastExport() })
	if !hostActionsEnabled(info) {
		cfg.Disable()
		last.Disable()
		items = append(items, widget.NewLabel(hostUnavailableMsg))
	} else if u.lastExport == "" {
		last.Disable()
	}
	items = append(items, container.NewHBox(cfg, last))
	d := dialog.NewCustom("Host", "Close", container.NewVBox(items...), u.win)
	d.Show()
}

func (u *ui) showAbout() {
	ver := "unknown"
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, m := range bi.Deps {
			if m.Path == "fyne.io/fyne/v2" {
				ver = m.Version
			}
		}
	}
	box := container.NewVBox(
		widget.NewLabelWithStyle("GGUIX - Fyne", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Fyne "+ver),
		widget.NewLabel(runtime.Version()),
		widget.NewLabel("BuildMarker "+BuildMarker),
	)
	dialog.NewCustom("About GGUIX", "Close", box, u.win).Show()
}
