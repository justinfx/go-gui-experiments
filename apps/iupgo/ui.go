package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gen2brain/iup-go/iup"

	"gguix/core"
)

// ui holds the IUP elements and the view state that is not part of the
// model. Everything here is used on the UI thread only.
type ui struct {
	dlg iup.Ihandle

	// Toolbar.
	search, catBtn, statusList, goTo, goBtn  iup.Ihandle
	tglTable, tglCards, exportBtn, themeList iup.Ihandle
	catMenu                                  iup.Ihandle
	catItems                                 [8]iup.Ihandle

	// Main area.
	split, zbox, empty iup.Ihandle
	tbl                tableView
	crd                cardsView
	frm                formView

	// Status bar.
	statusLbl, progZ, prog, pctLbl, cancelBtn, idLbl, hostBtn iup.Ihandle

	// Menu items that change state.
	mExport, mReveal, mTable, mCards iup.Ihandle
	mTheme                           [3]iup.Ihandle
	hostReveal, hostRevealExp        iup.Ihandle

	searchTimer, msgTimer iup.Ihandle

	// total is the row count the views show, -1 while unknown.
	total int
	// wantTop is a row to make the top visible row once the total is
	// applied, -1 when none.
	wantTop int
	msgGen  int
	inSel   bool
}

// post runs f on the UI thread. It is safe to call from any goroutine.
// Mechanism: IupPostMessage to the main dialog, delivered to POSTMESSAGE_CB.
func (a *app) post(f func()) {
	iup.PostMessage(a.ui.dlg, "", 0, f)
}

func set(ih iup.Ihandle, kv ...string) {
	for i := 0; i+1 < len(kv); i += 2 {
		iup.SetAttribute(ih, kv[i], kv[i+1])
	}
}

// buildUI creates every element and maps nothing yet.
func (a *app) buildUI() iup.Ihandle {
	u := &a.ui
	u.wantTop, u.total = -1, -1

	// ---- toolbar ----
	u.search = iup.Text()
	set(u.search, "CUEBANNER", "Search name or SKU", "RASTERSIZE", "280x26", "NAME", "search")
	iup.On(u.search, iup.ValueChangedCB, func(ih iup.Ihandle) int {
		a.setSearch(iup.GetAttribute(ih, "VALUE"))
		return iup.DEFAULT
	})

	u.catBtn = iup.Button("Categories")
	iup.On(u.catBtn, iup.ActionCB, func(iup.Ihandle) int { a.popupCategories(); return iup.DEFAULT })
	a.buildCategoryMenu()

	u.statusList = iup.List()
	set(u.statusList, "DROPDOWN", "YES", "VISIBLEITEMS", "4")
	for i, s := range statusChoices {
		iup.SetAttributeId(u.statusList, "", i+1, s)
	}
	iup.On(u.statusList, iup.ListActionCB, func(ih iup.Ihandle, text string, item, state int) int {
		if state == 1 && !u.inSel {
			a.setStatusFilter(statusChoices[item-1])
		}
		return iup.DEFAULT
	})

	u.goTo = iup.Text()
	set(u.goTo, "CUEBANNER", "Go to row", "RASTERSIZE", "90x26", "FILTER", "NUMBER")
	u.goBtn = iup.Button("Go")
	iup.On(u.goBtn, iup.ActionCB, func(iup.Ihandle) int { a.goClicked(); return iup.DEFAULT })
	iup.On(u.goTo, iup.KAnyCB, func(ih iup.Ihandle, c int) int {
		if c == iup.K_CR {
			a.goClicked()
			return iup.IGNORE
		}
		return iup.CONTINUE
	})

	u.tglTable = iup.Toggle("Table")
	u.tglCards = iup.Toggle("Cards")
	iup.On(u.tglTable, iup.ToggleActionCB, func(ih iup.Ihandle, st int) int {
		if st == 1 {
			a.setView(0)
		}
		return iup.DEFAULT
	})
	iup.On(u.tglCards, iup.ToggleActionCB, func(ih iup.Ihandle, st int) int {
		if st == 1 {
			a.setView(1)
		}
		return iup.DEFAULT
	})
	views := radioGroup(u.tglTable, u.tglCards)
	iup.SetHandle("views", views)

	u.exportBtn = iup.Button("Export CSV")
	iup.On(u.exportBtn, iup.ActionCB, func(iup.Ihandle) int { a.beginExport(); return iup.DEFAULT })

	u.themeList = iup.List()
	set(u.themeList, "DROPDOWN", "YES", "VISIBLEITEMS", "3")
	for i, s := range themeChoices {
		iup.SetAttributeId(u.themeList, "", i+1, s)
	}
	iup.On(u.themeList, iup.ListActionCB, func(ih iup.Ihandle, text string, item, state int) int {
		if state == 1 && !u.inSel {
			a.setTheme(themeChoices[item-1])
			a.uiSyncThemeControls()
		}
		return iup.DEFAULT
	})

	toolbar := iup.Hbox(u.search, u.catBtn, u.statusList, u.goTo, u.goBtn, views, iup.Fill(), u.exportBtn, u.themeList)
	set(toolbar, "GAP", "8", "ALIGNMENT", "ACENTER")

	// ---- main area ----
	a.buildTable()
	a.buildCards()
	u.empty = iup.Label("No items match.")
	set(u.empty, "ALIGNMENT", "ACENTER:ACENTER", "EXPAND", "YES")
	u.zbox = iup.Zbox(u.tbl.ih, u.crd.ih, u.empty)
	set(u.zbox, "ALIGNMENT", "NORTHWEST")
	left := iup.Vbox(u.zbox)
	set(left, "MARGIN", "0x0", "EXPAND", "YES")

	right := a.buildForm()
	u.split = iup.Split(left, right)
	set(u.split, "ORIENTATION", "VERTICAL", "VALUE", "650", "LAYOUTDRAG", "NO", "MINMAX", "300:800")

	// ---- status bar ----
	u.statusLbl = iup.Label("")
	set(u.statusLbl, "EXPAND", "HORIZONTAL", "ELLIPSIS", "YES")
	u.prog = iup.ProgressBar()
	set(u.prog, "MIN", "0", "MAX", "100", "RASTERSIZE", "200x")
	u.pctLbl = iup.Label("0%")
	set(u.pctLbl, "RASTERSIZE", "40x")
	u.cancelBtn = iup.Button("Cancel")
	iup.On(u.cancelBtn, iup.ActionCB, func(iup.Ihandle) int { a.cancelExport(); return iup.DEFAULT })
	progBox := iup.Hbox(u.prog, u.pctLbl, u.cancelBtn)
	set(progBox, "GAP", "6", "ALIGNMENT", "ACENTER")
	u.progZ = iup.Zbox(iup.Fill(), progBox)
	set(u.progZ, "ALIGNMENT", "ACENTER", "NAME", "progz")
	u.idLbl = iup.Label("")
	u.hostBtn = iup.Button("Host")
	iup.On(u.hostBtn, iup.ActionCB, func(iup.Ihandle) int { a.openHost(); return iup.DEFAULT })
	statusbar := iup.Hbox(u.statusLbl, u.progZ, u.idLbl, u.hostBtn)
	set(statusbar, "GAP", "10", "ALIGNMENT", "ACENTER")

	root := iup.Vbox(toolbar, u.split, statusbar)
	set(root, "MARGIN", "8x8", "GAP", "6")

	u.dlg = iup.Dialog(root)
	set(u.dlg, "TITLE", "GGUIX - IUP-Go", "RASTERSIZE", "1280x800", "MINSIZE", "1024x640", "SHRINK", "YES")
	iup.On(u.dlg, iup.PostMessageCB, func(ih iup.Ihandle, s string, i int, p any) int {
		if f, ok := p.(func()); ok {
			f()
		}
		return iup.DEFAULT
	})
	iup.On(u.dlg, iup.AppearanceChangedCB, func(ih iup.Ihandle, dark int) int {
		a.uiThemeChanged()
		return iup.DEFAULT
	})
	iup.On(u.dlg, iup.ResizeCB, func(ih iup.Ihandle, w, h int) int {
		a.onResize()
		return iup.DEFAULT
	})

	u.searchTimer = iup.Timer()
	set(u.searchTimer, "TIME", "250")
	iup.On(u.searchTimer, iup.TimerActionCB, func(ih iup.Ihandle) int {
		set(ih, "RUN", "NO")
		a.applySearchNow()
		return iup.DEFAULT
	})
	u.msgTimer = iup.Timer()
	iup.On(u.msgTimer, iup.TimerActionCB, func(ih iup.Ihandle) int {
		set(ih, "RUN", "NO")
		a.msg = ""
		a.uiStatus()
		return iup.DEFAULT
	})

	a.buildMenu()
	return u.dlg
}

func (a *app) goClicked() {
	if n, ok := parseRowNumber(iup.GetAttribute(a.ui.goTo, "VALUE")); ok {
		a.goToRow(n)
	}
}

func (a *app) onResize() {
	_, w, h := iup.GetInt2(a.ui.dlg, "RASTERSIZE")
	a.setWindowSize(w, h)
}

// ---- category popup ---------------------------------------------------

func (a *app) buildCategoryMenu() {
	u := &a.ui
	var items []iup.Ihandle
	for _, c := range core.AllCategories {
		c := c
		it := iup.MenuItem(c.String())
		set(it, "AUTOTOGGLE", "YES", "VALUE", "OFF")
		iup.On(it, iup.ActionCB, func(iup.Ihandle) int { a.toggleCategory(c); a.uiCategoryLabel(); return iup.DEFAULT })
		u.catItems[c] = it
		items = append(items, it)
	}
	clear := iup.MenuItem("Clear")
	iup.On(clear, iup.ActionCB, func(iup.Ihandle) int { a.clearCategories(); a.uiCategoryLabel(); return iup.DEFAULT })
	items = append(items, iup.MenuSeparator(), clear)
	u.catMenu = iup.Menu(items...)
}

func (a *app) popupCategories() {
	u := &a.ui
	for c, it := range u.catItems {
		v := "OFF"
		if a.cats[c] {
			v = "ON"
		}
		set(it, "VALUE", v)
	}
	_, x, y := iup.GetInt2(u.catBtn, "SCREENPOSITION")
	_, _, h := iup.GetInt2(u.catBtn, "RASTERSIZE")
	iup.Popup(u.catMenu, x, y+h)
}

func (a *app) uiCategoryLabel() {
	t := "Categories"
	if n := a.catCount(); n > 0 {
		t = fmt.Sprintf("Categories (%d)", n)
	}
	set(a.ui.catBtn, "TITLE", t)
	iup.Refresh(a.ui.catBtn)
}

// ---- model to view notifications -------------------------------------

// uiInit pushes the restored state into the controls (before first show).
func (a *app) uiInit() {
	u := &a.ui
	u.inSel = true
	set(u.search, "VALUE", a.search)
	for i, s := range statusChoices {
		if s == a.statusSel {
			iup.SetAttribute(u.statusList, "VALUE", i+1)
		}
	}
	a.uiCategoryLabel()
	a.uiSyncThemeControls()
	u.inSel = false
	a.uiApplyTheme()
	a.uiViewControls()
	a.uiSortHeaders()
	a.uiForm()
	a.uiExport()
	a.uiStatus()
	a.ready.Store(true)
	a.onPagerChange()
}

func (a *app) uiSyncThemeControls() {
	u := &a.ui
	u.inSel = true
	for i, s := range themeChoices {
		on := "OFF"
		if s == a.themeSel {
			iup.SetAttribute(u.themeList, "VALUE", i+1)
			on = "ON"
		}
		if u.mTheme[i] != 0 {
			set(u.mTheme[i], "VALUE", on)
		}
	}
	u.inSel = false
}

// uiApplyTheme applies the theme choice through the APPEARANCE global and
// refreshes everything that is drawn by the app.
func (a *app) uiApplyTheme() {
	v := "SYSTEM"
	switch a.themeSel {
	case "Light":
		v = "LIGHT"
	case "Dark":
		v = "DARK"
	}
	iup.SetGlobal("APPEARANCE", v)
	a.uiThemeChanged()
	// On GTK the global colors (and so DARKMODE) follow after the main loop
	// handles the settings change; an explicit choice must not wait for it.
	switch a.themeSel {
	case "Dark":
		a.setDark(true)
	case "Light":
		a.setDark(false)
	}
}

func (a *app) setDark(d bool) {
	if a.dark == d {
		return
	}
	a.dark = d
	a.pal = core.Colors(d)
	a.prewarmCellImages()
	a.tableTheme()
	a.cardsTheme()
	a.uiStyleForm()
	a.uiRefreshData(true)
}

// uiStyleForm gives the date picker the entry background of the active
// theme. IupDatePick takes its BACKCOLOR from TXTBGCOLOR once, at creation,
// so it stays on the startup theme (black in the light theme) otherwise.
func (a *app) uiStyleForm() {
	if a.ui.frm.date == 0 {
		return
	}
	bg, fg := rgbStr(globalRGB("TXTBGCOLOR")), rgbStr(globalRGB("TXTFGCOLOR"))
	if a.themeSel != "System" {
		if a.dark {
			bg, fg = "30 30 30", "235 235 235"
		} else {
			bg, fg = "255 255 255", "20 20 20"
		}
	}
	set(a.ui.frm.date, "BACKCOLOR", bg, "BGCOLOR", bg, "FGCOLOR", fg)
	iup.Update(a.ui.frm.date)
}

func (a *app) uiThemeChanged() {
	a.dark = iup.GetGlobalBool("DARKMODE")
	a.pal = core.Colors(a.dark)
	a.prewarmCellImages()
	a.tableTheme()
	a.cardsTheme()
	a.uiStyleForm()
	a.uiRefreshData(true)
}

func (a *app) uiViewControls() {
	u := &a.ui
	if a.viewMode == 1 {
		set(u.tglCards, "VALUE", "ON")
		showPage(u.zbox, "gxcards", u.crd.ih)
		if u.mCards != 0 {
			set(u.mCards, "VALUE", "ON")
		}
	} else {
		set(u.tglTable, "VALUE", "ON")
		showPage(u.zbox, "gxtable", u.tbl.ih)
		if u.mTable != 0 {
			set(u.mTable, "VALUE", "ON")
		}
	}
	a.uiEmptyPage()
}

func (a *app) uiViewChanged() {
	a.uiViewControls()
}

// uiEmptyPage shows "No items match." when the result is empty.
func (a *app) uiEmptyPage() {
	u := &a.ui
	if u.total == 0 {
		showPage(u.zbox, "gxempty", u.empty)
		return
	}
	if a.viewMode == 1 {
		showPage(u.zbox, "gxcards", u.crd.ih)
	} else {
		showPage(u.zbox, "gxtable", u.tbl.ih)
	}
}

// uiQueryChanged runs when search, filters or sort changed: the views are
// emptied until the first page of the new query arrives.
func (a *app) uiQueryChanged() {
	a.tableInvalidate()
	a.uiSortHeaders()
	a.uiStatus()
}

// onPagerChange runs on the UI thread after the pager's data or total
// changed.
func (a *app) onPagerChange() {
	a.pending.Store(false)
	total := a.pager.Total()
	a.tableInvalidate()
	a.syncForm()
	a.uiApplyTotal(total)
	a.uiRefreshData(false)
	a.uiStatus()
}

// uiApplyTotal sets the row count of both views.
func (a *app) uiApplyTotal(total int) {
	u := &a.ui
	if total == u.total {
		return
	}
	u.total = total
	a.tableSetTotal(total)
	a.cardsSetTotal(total)
	a.uiEmptyPage()
	if u.wantTop >= 0 && total >= 0 {
		row := u.wantTop
		u.wantTop = -1
		a.uiScrollTo(min(row, max(total-1, 0)))
	}
}

// uiRefreshData re-queries the visible cells (table) or repaints (cards).
func (a *app) uiRefreshData(force bool) {
	if a.viewMode == 1 {
		a.cardsRedraw()
	} else {
		a.tableRedraw(force)
	}
}

// uiScrollTo makes row the top visible row of the active view.
func (a *app) uiScrollTo(row int) {
	u := &a.ui
	if u.total < 0 {
		u.wantTop = row
		return
	}
	if a.viewMode == 1 {
		a.cardsScrollTo(row)
	} else {
		a.tableScrollTo(row)
	}
}

// uiSelectionChanged shows the selection in the active views and the form.
func (a *app) uiSelectionChanged() {
	a.tableSelect(a.selRow)
	a.cardsRedraw()
	a.uiForm()
}

// uiRestoreSelection puts the views back on the model's selection after a
// cancelled change.
func (a *app) uiRestoreSelection() {
	a.tableSelect(a.selRow)
	a.cardsRedraw()
}

func (a *app) uiStatus() {
	u := &a.ui
	if u.statusLbl == 0 {
		return
	}
	s := a.statusText()
	if a.msg != "" {
		s += "   " + a.msg
	}
	set(u.statusLbl, "TITLE", s)
	id := a.identity()
	set(u.idLbl, "TITLE", id)
}

func (a *app) uiMessage(d time.Duration) {
	u := &a.ui
	a.uiStatus()
	set(u.msgTimer, "RUN", "NO", "TIME", strconv.Itoa(int(d.Milliseconds())), "RUN", "YES")
}

func (a *app) uiDebounceSearch() {
	set(a.ui.searchTimer, "RUN", "NO", "RUN", "YES")
}

// uiExport reflects the export state in the status bar and the controls.
// uiRestoreCursor puts the arrow cursor back on the window and flushes, so
// a busy cursor left by a modal dialog or a blocking native call is cleared.
func (a *app) uiRestoreCursor() {
	for _, w := range []iup.Ihandle{a.ui.dlg, a.ui.crd.ih} {
		if w != 0 {
			set(w, "CURSOR", "ARROW")
		}
	}
	iup.Flush()
}

func (a *app) uiExport() {
	u := &a.ui
	if u.prog == 0 {
		return
	}
	if a.exp.running {
		showPage(u.progZ, "gxprog", iup.GetChild(u.progZ, 1))
		set(u.prog, "VALUE", strconv.Itoa(a.exportPercent()))
		set(u.pctLbl, "TITLE", fmt.Sprintf("%d%%", a.exportPercent()))
	} else {
		showPage(u.progZ, "gxnoprog", iup.GetChild(u.progZ, 0))
	}
	en := "YES"
	if a.exp.running {
		en = "NO"
	}
	set(u.exportBtn, "ACTIVE", en)
	if u.mExport != 0 {
		set(u.mExport, "ACTIVE", en)
	}
	rev := "NO"
	if a.exp.lastPath != "" && a.plat.HostActions {
		rev = "YES"
	}
	if u.mReveal != 0 {
		set(u.mReveal, "ACTIVE", rev)
	}
	if u.hostRevealExp != 0 {
		set(u.hostRevealExp, "ACTIVE", rev)
	}
}

// ---- sort headers -----------------------------------------------------

func headerTitle(i int, a *app) string {
	c := tableCols[i]
	t := c.title
	if c.sort != "" && c.sort == a.sortField {
		if a.sortDesc {
			t += " ▼"
		} else {
			t += " ▲"
		}
	}
	return t
}

func (a *app) uiSortHeaders() { a.tableHeaders() }

// clean trims for display.
func clean(s string) string { return strings.TrimSpace(s) }

func sizeStr(w, h int) string { return fmt.Sprintf("%dx%d", w, h) }

// showPage makes child the visible page of the zbox.
func showPage(z iup.Ihandle, name string, child iup.Ihandle) {
	iup.SetHandle(name, child)
	set(z, "VALUE", name)
}

// setViewFromUI keeps the toolbar toggles in step with the menu.
func (a *app) setViewFromUI(v int) {
	a.setView(v)
	a.uiViewControls()
}

// radioGroup returns an exclusive group of toggles. On Cocoa, radio buttons
// that share a superview and an action are one AppKit group, and every IUP
// toggle uses the same action, so two IupRadio groups placed in the same
// native parent deselect each other (setting the form's status toggle cleared
// the Table/Cards selector). The BackgroundBox is a native container: each
// group gets its own superview. IupRadio is a void container and cannot do
// that itself.
func radioGroup(toggles ...iup.Ihandle) iup.Ihandle {
	h := iup.Hbox(toggles...)
	return iup.BackgroundBox(iup.Radio(h))
}
