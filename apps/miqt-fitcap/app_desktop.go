//go:build !js

package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"gguix/core"
	"gguix/core/host"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
)

const appID = "miqt"

// App is the whole GGUIX window: toolbar, table or cards view, detail form,
// status bar and menus. Every method must run on the Qt main thread unless
// stated otherwise.
type App struct {
	svc   core.DataService
	store core.PrefsStore
	rows  *rowStore
	dlg   dialogs
	bench bool

	prefs     core.Prefs
	query     core.Query
	theme     string
	viewCards bool
	sortCol   int

	// selection
	selectedID  int64
	selectedRow int // -1 none
	pendingRow  int // row clicked before its data arrived, -1 none
	checkSel    bool
	relocating  bool
	suppress    int

	// visible range of the active view (0-based, inclusive)
	visFirst, visLast int

	win            *qt.QMainWindow
	search         *qt.QLineEdit
	searchTimer    *qt.QTimer
	catBtn         *qt.QPushButton
	catChecks      []*qt.QCheckBox
	statusCombo    *qt.QComboBox
	goEdit         *qt.QLineEdit
	goBtn          *qt.QPushButton
	tableBtn       *qt.QPushButton
	cardsBtn       *qt.QPushButton
	exportBtn      *qt.QPushButton
	themeCombo     *qt.QComboBox
	splitter       *qt.QSplitter
	stack          *qt.QStackedWidget
	table          *qt.QTableView
	tmodel         *tableModel
	tdel           *cellDelegate
	cards          *qt.QListView
	selShared      *qt.QItemSelectionModel // the one selection, attached to the active view only
	selParkedTable *qt.QItemSelectionModel
	selParkedCards *qt.QItemSelectionModel
	cmodel         *tableModel // same model as tmodel; the card list is a second view on it
	cdel           *cardDelegate
	form           *detailForm
	statusLabel    *qt.QLabel
	progress       *qt.QProgressBar
	pctLabel       *qt.QLabel
	cancelBtn      *qt.QPushButton
	identity       *qt.QLabel
	hostBtn        *qt.QPushButton
	prefsTimer     *qt.QTimer
	tray           *qt.QSystemTrayIcon
	menus          *menuActions
	hostDialog     *hostPanel
	aboutBox       *qt.QMessageBox
	lastNotice     string
	hostInfo       host.Info
	exporting      bool
	refreshes      int // UI refreshes from page arrivals (diagnostics)
	exportCancel   context.CancelFunc
	lastExport     string
	ctx            context.Context
	cancelAll      context.CancelFunc
}

// dialogs are the modal interactions. Tests replace them.
type dialogs struct {
	confirmDiscard func() bool
	errorBox       func(title, text string)
	saveFile       func(defaultName string) string
	openFile       func() string
	notify         func(title, body string) // OS notification attempt; returns nothing
}

type options struct {
	svc    core.DataService
	store  core.PrefsStore
	bench  bool
	nodlg  bool // do not install real dialogs (tests install their own)
	noSave bool
}

func newApp(o options) *App {
	a := &App{svc: o.svc, store: o.store, bench: o.bench, selectedRow: -1, pendingRow: -1}
	a.ctx, a.cancelAll = context.WithCancel(context.Background())
	a.hostInfo = host.Current(appID)
	a.installDialogs()

	p, err := a.store.Load()
	if err != nil {
		p = core.DefaultPrefs()
	}
	a.prefs = p.Sanitize()
	if a.bench {
		a.prefs.View = core.ViewTable
	}
	a.query = a.prefs.Query()
	a.theme = a.prefs.Theme
	a.viewCards = a.prefs.View == core.ViewCards
	a.sortCol = columnForSort(a.query.SortBy)

	a.rows = newRowStore(a.ctx, a.svc)
	a.rows.onArrive = a.onRowsArrived

	a.buildUI()
	a.applyTheme(a.theme, false)
	a.syncControlsFromQuery()
	a.setView(a.viewCards, false)
	return a
}

// start loads the first query and restores the selection. Call after Show.
func (a *App) start() {
	a.rows.setQuery(a.query)
	a.tmodel.reset(0)
	a.updateVisible()
	a.updateStatus()
	if a.bench || a.prefs.SelectedID == 0 {
		return
	}
	id, q := a.prefs.SelectedID, a.query
	go func() {
		row, ok, err := core.FindRow(a.ctx, a.svc, q, id, 2000)
		if err != nil || !ok {
			return
		}
		mainthread.Start(func() {
			if a.query.Normalize().Key() != q.Normalize().Key() || a.selectedRow >= 0 {
				return
			}
			a.goToRow(row)
		})
	}()
}

// ---------------------------------------------------------------- row store

// onRowsArrived runs on the UI thread once per batch of arrived pages. It
// updates the row count, tells the active view about the rows it shows that
// just got data, and settles selection and status. Pages outside the visible
// range (prefetch) cost nothing here.
func (a *App) onRowsArrived(ranges [][2]int) {
	total := max(0, a.rows.Total())
	a.refreshes++
	a.suppress++
	if total != a.tmodel.shown {
		a.tmodel.sync(total)
	}
	a.suppress--
	a.updateVisible()
	repaint := false
	for _, r := range ranges {
		lo, hi := max(r[0], a.visFirst), min(r[1], a.visLast)
		if lo > hi {
			continue
		}
		repaint = true
	}
	if repaint {
		a.activeView().Viewport().Update()
	}
	a.resolveSelection(total)
	a.updateStatus()
}

// resolveSelection handles selection state that depends on loaded data:
// a row selected before it loaded, the rule that a selection survives a
// query change only if it is on the first page, and relocating the
// selected item after an edit moved it.
func (a *App) resolveSelection(total int) {
	if a.pendingRow >= 0 {
		if it, ok := a.rows.Row(a.pendingRow); ok {
			row := a.pendingRow
			a.pendingRow = -1
			a.selectedRow, a.selectedID = row, it.ID
			a.form.load(it)
			a.syncSelection()
			a.savePrefsSoon()
		}
	}
	if a.checkSel && a.rows.Total() >= 0 {
		last := min(storePageSize, total) - 1
		if total == 0 || a.rows.RangeLoaded(0, last) {
			a.checkSel = false
			found := -1
			for i := 0; i <= last; i++ {
				if it, ok := a.rows.Row(i); ok && it.ID == a.selectedID {
					found = i
					break
				}
			}
			if a.selectedID != 0 && found < 0 {
				a.clearSelection()
			} else if found >= 0 {
				a.selectedRow = found
				a.syncSelection()
			}
		}
	}
	if a.selectedRow >= 0 && a.selectedRow < total && a.pendingRow < 0 && !a.relocating && !a.checkSel {
		if it, ok := a.rows.Row(a.selectedRow); ok && it.ID != a.selectedID && a.selectedID != 0 {
			a.relocate()
		}
	}
}

// relocate finds the selected item again after an edit changed its row.
func (a *App) relocate() {
	a.relocating = true
	id, q := a.selectedID, a.query
	go func() {
		row, ok, _ := core.FindRow(a.ctx, a.svc, q, id, 2000)
		mainthread.Start(func() {
			a.relocating = false
			if a.query.Key() != q.Key() || a.selectedID != id {
				return
			}
			if ok {
				a.selectedRow = row
			} else {
				a.selectedRow = -1
			}
			a.syncSelection()
		})
	}()
}

// ------------------------------------------------------------- visible

func (a *App) activeView() *qt.QAbstractItemView {
	if a.viewCards {
		return a.cards.QAbstractItemView
	}
	return a.table.QAbstractItemView
}

func (a *App) rowExtent() int {
	if a.viewCards {
		return cardRowHeight
	}
	return a.table.VerticalHeader().DefaultSectionSize()
}

// topRow returns the row index at the top of the active view.
func (a *App) topRow() int {
	return a.activeView().VerticalScrollBar().Value()
}

// updateVisible recomputes the visible range of the active view, tells the
// row store and refreshes the status bar.
func (a *App) updateVisible() {
	total := a.rows.Total()
	first := a.topRow()
	h := a.activeView().Viewport().Height()
	n := max(1, (h+a.rowExtent()-1)/a.rowExtent())
	last := first + n - 1
	if total >= 0 {
		last = min(last, total-1)
		first = max(0, min(first, total-1))
	}
	if last < first {
		last = first
	}
	a.visFirst, a.visLast = first, last
	a.rows.setVisible(first, last)
	a.updateStatus()
}

func (a *App) updateStatus() {
	if a.statusLabel == nil {
		return
	}
	total := a.rows.Total()
	var s string
	if total < 0 {
		s = "Loading…"
	} else {
		s = core.FormatCount(total) + " items"
		if total > 0 {
			s += fmt.Sprintf(" · rows %d-%d", a.visFirst+1, min(a.visLast, total-1)+1)
		}
		lf, cached := a.rows.stats()
		s += fmt.Sprintf(" · last fetch %d ms · %d pages cached", lf.Milliseconds(), cached)
	}
	if a.statusLabel.Text() != s {
		a.statusLabel.SetText(s)
	}
}

// visibleLoaded reports whether every visible row has data and the UI
// refresh for the latest page arrival has been made.
func (a *App) visibleLoaded() bool {
	if a.rows.Total() < 0 || !a.rows.settled() {
		return false
	}
	if a.rows.Total() == 0 {
		return true
	}
	return a.rows.RangeLoaded(a.visFirst, a.visLast)
}

// ------------------------------------------------------------- queries

// setQuery applies a new query: reload from the first page, scroll to top.
func (a *App) setQuery(q core.Query) {
	q = q.Normalize()
	a.query = q
	a.checkSel = a.selectedID != 0
	a.pendingRow = -1
	a.selectedRow = -1
	a.rows.setQuery(q)
	a.suppress++
	a.syncSelection()
	a.suppress--
	a.table.VerticalScrollBar().SetValue(0)
	a.cards.VerticalScrollBar().SetValue(0)
	a.updateVisible()
	a.tmodel.headersChanged()
	a.table.Viewport().Update()
	a.cards.Viewport().Update()
	a.savePrefsSoon()
}

func (a *App) currentQuery() core.Query {
	q := core.Query{Search: strings.TrimSpace(a.search.Text()), SortBy: a.query.SortBy, Desc: a.query.Desc}
	for i, c := range a.catChecks {
		if c.IsChecked() {
			q.Categories = append(q.Categories, core.AllCategories[i])
		}
	}
	if i := a.statusCombo.CurrentIndex(); i > 0 {
		q.Status = core.StatusPtr(core.AllStatuses[i-1])
	}
	return q.Normalize()
}

// applyFilters is called when search or filters changed.
func (a *App) applyFilters() {
	a.updateCategoryButton()
	a.setQuery(a.currentQuery())
}

func (a *App) updateCategoryButton() {
	n := 0
	for _, c := range a.catChecks {
		if c.IsChecked() {
			n++
		}
	}
	if n == 0 {
		a.catBtn.SetText("Categories")
	} else {
		a.catBtn.SetText("Categories (" + strconv.Itoa(n) + ")")
	}
}

// sortByColumn mirrors a header click.
func (a *App) sortByColumn(col int) {
	if col < 0 || col >= len(columns) || columns[col].sort == "" {
		return
	}
	f := columns[col].sort
	q := a.query
	if col == a.sortCol {
		q.Desc = !q.Desc
	} else {
		q.SortBy, q.Desc = f, false
	}
	a.sortCol = col
	a.setQuerySort(q)
}

func (a *App) setQuerySort(q core.Query) {
	cur := a.currentQuery()
	cur.SortBy, cur.Desc = q.SortBy, q.Desc
	a.setQuery(cur)
}

// syncControlsFromQuery puts the saved query into the toolbar widgets.
func (a *App) syncControlsFromQuery() {
	q := a.query
	a.suppress++
	a.search.SetText(q.Search)
	for i, c := range a.catChecks {
		c.SetChecked(false)
		for _, qc := range q.Categories {
			if qc == core.AllCategories[i] {
				c.SetChecked(true)
			}
		}
	}
	idx := 0
	if q.Status != nil {
		idx = int(*q.Status) + 1
	}
	a.statusCombo.SetCurrentIndex(idx)
	a.themeCombo.SetCurrentIndex(themeIndex(a.theme))
	a.suppress--
	a.updateCategoryButton()
}

// ------------------------------------------------------------ selection

// selectRow selects a row (user action or go-to). It asks before
// discarding unsaved edits.
func (a *App) selectRow(row int) {
	if row < 0 || row == a.selectedRow {
		return
	}
	if a.form.isDirty() && !a.dlg.confirmDiscard() {
		a.syncSelection()
		return
	}
	a.pendingRow = -1
	a.selectedRow = row
	if it, ok := a.rows.Row(row); ok {
		a.selectedID = it.ID
		a.form.load(it)
	} else {
		a.selectedID = 0
		a.pendingRow = row
		a.form.clear()
	}
	a.syncSelection()
	a.savePrefsSoon()
}

func (a *App) clearSelection() {
	a.selectedRow, a.selectedID, a.pendingRow = -1, 0, -1
	a.form.clear()
	a.syncSelection()
	a.savePrefsSoon()
}

// syncSelection shows selectedRow in both views without raising events.
func (a *App) syncSelection() {
	a.suppress++
	defer func() { a.suppress-- }()
	sel := func(v *qt.QAbstractItemView, m *qt.QAbstractItemModel, shown int, flags qt.QItemSelectionModel__SelectionFlag) {
		sm := v.SelectionModel()
		row := a.selectedRow
		if a.pendingRow >= 0 {
			row = a.pendingRow
		}
		if row < 0 || row >= shown {
			sm.ClearSelection()
			sm.ClearCurrentIndex()
			return
		}
		idx := m.Index(row, 0, qt.NewQModelIndex())
		sm.SetCurrentIndex(idx, flags)
	}
	f := qt.QItemSelectionModel__ClearAndSelect | qt.QItemSelectionModel__Current | qt.QItemSelectionModel__Rows
	sel(a.activeView(), a.tmodel.QAbstractItemModel, a.tmodel.shown, f)
}

// scrollToTopRow makes row the top row of the active view.
func (a *App) scrollToTopRow(row int) {
	a.table.VerticalScrollBar().SetValue(row)
	a.cards.VerticalScrollBar().SetValue(row)
	a.updateVisible()
}

// goToRow scrolls row to the top and selects it.
func (a *App) goToRow(row int) {
	total := a.rows.Total()
	if total <= 0 {
		return
	}
	row = max(0, min(row, total-1))
	// Scroll first: the selection's own scrollTo then finds the row on screen.
	a.scrollToTopRow(row)
	a.selectRow(row)
}

func (a *App) onGo() {
	s := strings.TrimSpace(a.goEdit.Text())
	n, err := strconv.Atoi(s)
	if err != nil {
		return
	}
	a.goToRow(n - 1)
}

// ---------------------------------------------------------------- views

func (a *App) setView(cards bool, save bool) {
	a.suppress++
	a.viewCards = cards
	if cards {
		a.stack.SetCurrentIndex(1)
	} else {
		a.stack.SetCurrentIndex(0)
	}
	// Only the visible view is attached to the shared selection model: a
	// selection change on the hidden QListView forces a layout of its 1M items.
	if cards {
		a.table.SetSelectionModel(a.selParkedTable)
		a.cards.SetSelectionModel(a.selShared)
		// the hidden list has not laid out its rows (so no scroll range yet)
		a.cards.ExecuteDelayedItemsLayout()
	} else {
		a.cards.SetSelectionModel(a.selParkedCards)
		a.table.SetSelectionModel(a.selShared)
	}
	a.tableBtn.SetChecked(!cards)
	a.cardsBtn.SetChecked(cards)
	if a.menus != nil {
		a.menus.tableAct.SetChecked(!cards)
		a.menus.cardsAct.SetChecked(cards)
	}
	a.suppress--
	row := a.selectedRow
	if a.pendingRow >= 0 {
		row = a.pendingRow
	}
	if row >= 0 && a.rows.Total() > 0 {
		a.scrollToTopRow(row)
	} else {
		a.scrollToTopRow(0)
	}
	a.syncSelection()
	a.updateVisible()
	if save {
		a.savePrefsSoon()
	}
}

// ---------------------------------------------------------------- theme

func themeIndex(t string) int {
	switch t {
	case core.ThemeLight:
		return 1
	case core.ThemeDark:
		return 2
	}
	return 0
}

// isDark says whether custom-rendered elements use the dark palette.
func (a *App) isDark() bool {
	switch a.theme {
	case core.ThemeDark:
		return true
	case core.ThemeLight:
		return false
	}
	return qt.QGuiApplication_Palette().Window().Color().Lightness() < 128
}

// applyTheme switches the Qt color scheme. QStyleHints.setColorScheme is not
// in the Go binding, so the Q_PROPERTY is written through QObject.setProperty.
func (a *App) applyTheme(t string, save bool) {
	a.theme = t
	scheme := 0 // Qt::ColorScheme::Unknown follows the system
	switch t {
	case core.ThemeLight:
		scheme = 1
	case core.ThemeDark:
		scheme = 2
	}
	hints := qt.QGuiApplication_StyleHints()
	hints.QObject.SetProperty("colorScheme", qt.NewQVariant4(scheme))
	a.suppress++
	a.themeCombo.SetCurrentIndex(themeIndex(t))
	if a.menus != nil {
		a.menus.themeActs[themeIndex(t)].SetChecked(true)
	}
	a.suppress--
	a.repaintAll()
	if save {
		a.savePrefsSoon()
	}
}

func (a *App) repaintAll() {
	if a.table == nil {
		return
	}
	a.table.Viewport().Update()
	a.cards.Viewport().Update()
}

// ---------------------------------------------------------------- prefs

func (a *App) collectPrefs() core.Prefs {
	p := core.DefaultPrefs()
	p.Theme = a.theme
	p.SetQuery(a.query)
	p.SelectedID = a.selectedID
	if a.viewCards {
		p.View = core.ViewCards
	}
	p.WindowWidth, p.WindowHeight = a.win.Width(), a.win.Height()
	return p
}

func (a *App) savePrefsSoon() {
	if a.prefsTimer != nil {
		a.prefsTimer.Start(500)
	}
}

func (a *App) savePrefsNow() {
	if a.prefsTimer != nil {
		a.prefsTimer.Stop()
	}
	_ = a.store.Save(a.collectPrefs())
}

// --------------------------------------------------------------- export

func (a *App) startExportInteractive() {
	if a.exporting {
		return
	}
	path := a.dlg.saveFile("gguix-export.csv")
	if path == "" {
		return
	}
	a.startExport(path, nil)
}

// startExport runs the export in the background. done (optional) is called
// on the UI thread when it finishes.
func (a *App) startExport(path string, done func(rows int, err error)) {
	if a.exporting {
		return
	}
	f, err := os.Create(path)
	if err != nil {
		a.dlg.errorBox("Export failed", err.Error())
		if done != nil {
			done(0, err)
		}
		return
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.exportCancel = cancel
	a.setExporting(true)
	q := a.query
	var last atomic.Int64
	go func() {
		rows, err := a.svc.Export(ctx, q, f, func(d, total int) {
			pct := 0
			if total > 0 {
				pct = d * 100 / total
			}
			if last.Swap(int64(pct)) == int64(pct) && d != total {
				return
			}
			mainthread.Start(func() { a.setProgress(pct) })
		})
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
		cancelled := ctx.Err() != nil
		if err != nil || cancelled {
			_ = os.Remove(path)
		}
		mainthread.Start(func() {
			cancel()
			a.exportCancel = nil
			a.setExporting(false)
			switch {
			case cancelled:
				a.win.StatusBar().ShowMessage2("Export cancelled", 2000)
			case err != nil:
				a.dlg.errorBox("Export failed", err.Error())
			default:
				a.lastExport = path
				a.menus.revealExport.SetEnabled(true)
				if a.hostDialog != nil {
					a.hostDialog.updateButtons()
				}
				body := core.FormatCount(rows) + " rows written"
				a.lastNotice = "Export complete: " + body
				if !a.bench {
					a.win.StatusBar().ShowMessage2(a.lastNotice, 4000)
					a.dlg.notify("Export complete", body)
				}
			}
			if done != nil {
				if cancelled && err == nil {
					err = context.Canceled
				}
				done(rows, err)
			}
		})
	}()
}

func (a *App) setExporting(on bool) {
	a.exporting = on
	a.exportBtn.SetEnabled(!on)
	a.menus.exportAct.SetEnabled(!on)
	a.progress.SetVisible(on)
	a.pctLabel.SetVisible(on)
	a.cancelBtn.SetVisible(on)
	if on {
		a.setProgress(0)
	}
}

func (a *App) setProgress(pct int) {
	a.progress.SetValue(pct)
	a.pctLabel.SetText(fmt.Sprintf("%d%%", pct))
}

func (a *App) cancelExport() {
	if a.exportCancel != nil {
		a.exportCancel()
	}
}

// -------------------------------------------------------------- misc UI

func (a *App) focusSearch() {
	a.search.SetFocus()
	a.search.SelectAll()
}

func (a *App) reload() { a.setQuery(a.currentQuery()) }

func (a *App) flash(text string) { a.win.StatusBar().ShowMessage2(text, 2000) }

func (a *App) quit() {
	a.savePrefsNow()
	a.cancelAll()
	qt.QCoreApplication_Quit()
}

var _ = time.Second
