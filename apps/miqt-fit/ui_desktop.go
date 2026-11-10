//go:build !js

package main

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gguix/core"
	"gguix/core/host"

	qt "github.com/mappu/miqt/qt6"
)

func itoa64(n int64) string { return strconv.FormatInt(n, 10) }

func timeMonth(m int) time.Month { return time.Month(m) }

// buildUI creates the window and all widgets.
func (a *App) buildUI() {
	w := qt.NewQMainWindow2()
	a.win = w
	w.SetWindowTitle("GGUIX - MIQT")
	w.SetMinimumSize2(1024, 640)
	w.Resize(a.prefs.WindowWidth, a.prefs.WindowHeight)

	central := qt.NewQWidget2()
	root := qt.NewQVBoxLayout(central)
	root.AddLayout(a.buildToolbar().QLayout)

	a.buildViews()
	a.form = newDetailForm(a)
	a.splitter = qt.NewQSplitter3(qt.Horizontal)
	a.splitter.AddWidget(a.stack.QWidget)
	a.splitter.AddWidget(a.form.root)
	a.splitter.SetChildrenCollapsible(false)
	a.splitter.SetStretchFactor(0, 65)
	a.splitter.SetStretchFactor(1, 35)
	a.splitter.SetSizes([]int{a.prefs.WindowWidth * 65 / 100, a.prefs.WindowWidth * 35 / 100})
	root.AddWidget2(a.splitter.QWidget, 1)
	w.SetCentralWidget(central)

	a.buildStatusBar()
	a.buildMenus()
	a.wire()

	a.prefsTimer = qt.NewQTimer2(w.QObject)
	a.prefsTimer.SetSingleShot(true)
	a.prefsTimer.OnTimeout(a.savePrefsNow)

	w.OnResizeEvent(func(super func(*qt.QResizeEvent), e *qt.QResizeEvent) {
		super(e)
		a.savePrefsSoon()
	})
	w.OnCloseEvent(func(super func(*qt.QCloseEvent), e *qt.QCloseEvent) {
		a.savePrefsNow()
		a.cancelAll()
		super(e)
	})
	w.OnChangeEvent(func(super func(*qt.QEvent), e *qt.QEvent) {
		super(e)
		switch e.Type() {
		case qt.QEvent__PaletteChange, qt.QEvent__ApplicationPaletteChange:
			a.form.restyleErrors()
			a.repaintAll()
		}
	})
}

func (a *App) buildToolbar() *qt.QHBoxLayout {
	h := qt.NewQHBoxLayout2()

	a.search = qt.NewQLineEdit2()
	a.search.SetPlaceholderText("Search name or SKU")
	a.search.SetFixedWidth(280)
	h.AddWidget(a.search.QWidget)
	a.searchTimer = qt.NewQTimer2(a.search.QObject)
	a.searchTimer.SetSingleShot(true)
	a.searchTimer.SetInterval(250)

	// Category filter: a button with a menu of checkboxes and a Clear action.
	a.catBtn = qt.NewQPushButton3("Categories")
	menu := qt.NewQMenu2()
	for _, c := range core.AllCategories {
		cb := qt.NewQCheckBox3(c.String())
		cb.SetContentsMargins(8, 2, 8, 2)
		wa := qt.NewQWidgetAction(menu.QObject)
		wa.SetDefaultWidget(cb.QWidget)
		menu.QWidget.AddAction(wa.QAction)
		a.catChecks = append(a.catChecks, cb)
	}
	menu.AddSeparator()
	clear := qt.NewQAction5("Clear", menu.QObject)
	clear.OnTriggered(func() {
		a.suppress++
		for _, c := range a.catChecks {
			c.SetChecked(false)
		}
		a.suppress--
		a.applyFilters()
	})
	menu.QWidget.AddAction(clear)
	a.catBtn.SetMenu(menu)
	h.AddWidget(a.catBtn.QWidget)

	a.statusCombo = qt.NewQComboBox2()
	a.statusCombo.AddItems([]string{"All statuses", "Active", "Backordered", "Discontinued"})
	h.AddWidget(a.statusCombo.QWidget)

	a.goEdit = qt.NewQLineEdit2()
	a.goEdit.SetPlaceholderText("Go to row")
	a.goEdit.SetFixedWidth(100)
	a.goEdit.SetValidator(qt.NewQIntValidator4(0, 2_000_000_000, a.goEdit.QObject).QValidator)
	a.goBtn = qt.NewQPushButton3("Go")
	h.AddWidget(a.goEdit.QWidget)
	h.AddWidget(a.goBtn.QWidget)

	a.tableBtn = qt.NewQPushButton3("Table")
	a.cardsBtn = qt.NewQPushButton3("Cards")
	grp := qt.NewQButtonGroup2(a.win.QObject)
	for _, b := range []*qt.QPushButton{a.tableBtn, a.cardsBtn} {
		b.SetCheckable(true)
		grp.AddButton(b.QAbstractButton)
		h.AddWidget(b.QWidget)
	}
	grp.SetExclusive(true)
	h.AddStretch()

	a.exportBtn = qt.NewQPushButton3("Export CSV")
	h.AddWidget(a.exportBtn.QWidget)
	a.themeCombo = qt.NewQComboBox2()
	a.themeCombo.AddItems([]string{"System", "Light", "Dark"})
	h.AddWidget(a.themeCombo.QWidget)
	return h
}

func (a *App) buildViews() {
	a.stack = qt.NewQStackedWidget2()

	// Table
	a.table = qt.NewQTableView2()
	a.tmodel = newTableModel(a)
	a.table.SetModel(a.tmodel.QAbstractItemModel)
	a.table.SetSelectionBehavior(qt.QAbstractItemView__SelectRows)
	a.table.SetSelectionMode(qt.QAbstractItemView__SingleSelection)
	a.table.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	a.table.SetVerticalScrollMode(qt.QAbstractItemView__ScrollPerItem)
	a.table.SetShowGrid(false)
	a.table.SetWordWrap(false)
	a.table.VerticalHeader().SetVisible(false)
	hh := a.table.HorizontalHeader()
	hh.SetSectionsClickable(true)
	hh.SetSortIndicatorShown(false)
	hh.SetHighlightSections(false)
	hh.SetStretchLastSection(false)
	for i, c := range columns {
		a.table.SetColumnWidth(i, c.width)
	}
	// Uniform row height so the 1,000,000 row scrollbar is cheap.
	a.table.VerticalHeader().SetSectionResizeMode(qt.QHeaderView__Fixed)
	a.tdel = newCellDelegate(a)
	for i, c := range columns {
		if c.custom {
			a.table.SetItemDelegateForColumn(i, a.tdel.QAbstractItemDelegate)
		}
	}
	a.table.OnPaintEvent(func(super func(*qt.QPaintEvent), e *qt.QPaintEvent) {
		super(e)
		a.paintEmpty(a.table.Viewport())
	})
	a.table.OnResizeEvent(func(super func(*qt.QResizeEvent), e *qt.QResizeEvent) {
		super(e)
		if !a.viewCards {
			a.updateVisible()
		}
	})

	// Cards
	a.cards = qt.NewQListView2()
	a.cmodel = a.tmodel
	a.cards.SetModel(a.tmodel.QAbstractItemModel)
	a.selShared = a.table.SelectionModel()
	a.selParkedTable = qt.NewQItemSelectionModel3(a.tmodel.QAbstractItemModel)
	a.selParkedCards = a.cards.SelectionModel()
	a.cards.SetModelColumn(colID)
	a.cards.SetSelectionMode(qt.QAbstractItemView__SingleSelection)
	a.cards.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	a.cards.SetVerticalScrollMode(qt.QAbstractItemView__ScrollPerItem)
	a.cards.SetUniformItemSizes(true)
	a.cards.SetAutoScroll(false) // scrolling is explicit (scrollToTopRow); skip scrollTo on show
	a.cards.SetResizeMode(qt.QListView__Adjust)
	a.cards.SetHorizontalScrollBarPolicy(qt.ScrollBarAlwaysOff)
	a.cdel = newCardDelegate(a)
	a.cards.SetItemDelegate(a.cdel.QAbstractItemDelegate)
	a.cards.OnPaintEvent(func(super func(*qt.QPaintEvent), e *qt.QPaintEvent) {
		super(e)
		a.paintEmpty(a.cards.Viewport())
	})
	a.cards.OnResizeEvent(func(super func(*qt.QResizeEvent), e *qt.QResizeEvent) {
		super(e)
		if a.viewCards {
			a.updateVisible()
		}
	})

	a.stack.AddWidget(a.table.QWidget)
	a.stack.AddWidget(a.cards.QWidget)
}

// paintEmpty draws "No items match." over an empty result.
func (a *App) paintEmpty(vp *qt.QWidget) {
	if a.rows.Total() != 0 {
		return
	}
	p := qt.NewQPainter2(vp.QPaintDevice)
	p.SetPen(qt.QGuiApplication_Palette().Text().Color())
	p.DrawText7(0, 0, vp.Width(), vp.Height(), int(qt.AlignCenter), "No items match.")
	p.End()
}

func (a *App) buildStatusBar() {
	sb := a.win.StatusBar()
	sb.SetSizeGripEnabled(false)
	a.statusLabel = qt.NewQLabel3("Loading…")
	sb.AddWidget2(a.statusLabel.QWidget, 1)

	a.progress = qt.NewQProgressBar2()
	a.progress.SetRange(0, 100)
	a.progress.SetFixedWidth(200)
	a.progress.SetTextVisible(false)
	a.pctLabel = qt.NewQLabel3("0%")
	a.pctLabel.SetMinimumWidth(a.pctLabel.FontMetrics().HorizontalAdvance("100%") + 4)
	a.cancelBtn = qt.NewQPushButton3("Cancel")
	for _, w := range []*qt.QWidget{a.progress.QWidget, a.pctLabel.QWidget, a.cancelBtn.QWidget} {
		w.SetVisible(false)
		sb.AddPermanentWidget(w)
	}
	name := a.hostInfo.Identity()
	a.identity = qt.NewQLabel3(name)
	a.hostBtn = qt.NewQPushButton3("Host")
	sb.AddPermanentWidget(a.identity.QWidget)
	sb.AddPermanentWidget(a.hostBtn.QWidget)
}

// wire connects signals. It runs once after all widgets exist.
func (a *App) wire() {
	// search with 250 ms debounce
	a.search.OnTextChanged(func(string) {
		if a.suppress == 0 {
			a.searchTimer.Start2()
		}
	})
	a.searchTimer.OnTimeout(a.applyFilters)
	for _, c := range a.catChecks {
		c.OnToggled(func(bool) {
			if a.suppress == 0 {
				a.applyFilters()
			}
		})
	}
	a.statusCombo.OnCurrentIndexChanged(func(int) {
		if a.suppress == 0 {
			a.applyFilters()
		}
	})
	a.goBtn.OnClicked(a.onGo)
	a.goEdit.OnReturnPressed(a.onGo)
	a.tableBtn.OnClicked(func() { a.setView(false, true) })
	a.cardsBtn.OnClicked(func() { a.setView(true, true) })
	a.exportBtn.OnClicked(a.startExportInteractive)
	a.cancelBtn.OnClicked(a.cancelExport)
	a.themeCombo.OnCurrentIndexChanged(func(i int) {
		if a.suppress == 0 {
			a.applyTheme([]string{core.ThemeSystem, core.ThemeLight, core.ThemeDark}[i], true)
		}
	})
	a.hostBtn.OnClicked(a.showHostPanel)

	// header clicks sort
	a.table.HorizontalHeader().OnSectionClicked(a.sortByColumn)

	// scrolling feeds the row store
	a.table.VerticalScrollBar().OnValueChanged(func(int) {
		if !a.viewCards {
			a.updateVisible()
		}
	})
	a.cards.VerticalScrollBar().OnValueChanged(func(int) {
		if a.viewCards {
			a.updateVisible()
		}
	})

	// selection from either view
	onRow := func(cur, prev *qt.QModelIndex) {
		if a.suppress > 0 || !cur.IsValid() {
			return
		}
		a.selectRow(cur.Row())
	}
	a.selShared.OnCurrentRowChanged(onRow)
}

// ---------------------------------------------------------------- dialogs

func (a *App) installDialogs() {
	a.dlg = dialogs{
		confirmDiscard: func() bool {
			box := qt.NewQMessageBox2()
			box.SetIcon(qt.QMessageBox__Warning)
			box.SetWindowTitle("GGUIX")
			box.SetText("Discard unsaved changes?")
			discard := box.AddButton2("Discard", qt.QMessageBox__DestructiveRole)
			cancel := box.AddButton2("Cancel", qt.QMessageBox__RejectRole)
			box.SetDefaultButton(cancel)
			box.SetEscapeButton(cancel.QAbstractButton)
			box.Exec()
			ok := box.ClickedButton() != nil && box.ClickedButton().UnsafePointer() == discard.UnsafePointer()
			box.DeleteLater()
			return ok
		},
		errorBox: func(title, text string) {
			qt.QMessageBox_Critical(a.win.QWidget, title, text)
		},
		saveFile: func(def string) string {
			dir := def
			if home, err := os.UserHomeDir(); err == nil {
				dir = filepath.Join(home, def)
			}
			return qt.QFileDialog_GetSaveFileName4(a.win.QWidget, "Export CSV", dir, "CSV files (*.csv)")
		},
		openFile: func() string {
			dir := ""
			if home, err := os.UserHomeDir(); err == nil {
				dir = home
			}
			return qt.QFileDialog_GetOpenFileName4(a.win.QWidget, "Import Notes", dir, "Text files (*.txt)")
		},
		notify: a.osNotify,
	}
}

// osNotify sends a system notification through a QSystemTrayIcon message.
func (a *App) osNotify(title, body string) {
	if !qt.QSystemTrayIcon_IsSystemTrayAvailable() {
		return
	}
	if a.tray == nil {
		icon := a.win.Style().StandardIcon(qt.QStyle__SP_ComputerIcon, nil, nil)
		a.tray = qt.NewQSystemTrayIcon4(icon, a.win.QObject)
	}
	a.tray.Show()
	a.tray.ShowMessage3(title, body, qt.NewQIcon(), 5000)
	t := qt.NewQTimer2(a.win.QObject)
	t.SetSingleShot(true)
	t.OnTimeout(func() { a.tray.Hide() })
	t.Start(8000)
}

// ----------------------------------------------------------- host panel

type hostPanel struct {
	a        *App
	dlg      *qt.QDialog
	values   map[string]*qt.QLabel
	revConf  *qt.QPushButton
	revExp   *qt.QPushButton
	rowsForm *qt.QFormLayout
}

func (a *App) showHostPanel() {
	if a.hostDialog == nil {
		a.hostDialog = newHostPanel(a)
	}
	a.hostDialog.refresh()
	a.hostDialog.updateButtons()
	a.hostDialog.dlg.Show()
	a.hostDialog.dlg.Raise()
	a.hostDialog.dlg.ActivateWindow()
}

func newHostPanel(a *App) *hostPanel {
	p := &hostPanel{a: a, values: map[string]*qt.QLabel{}}
	p.dlg = qt.NewQDialog(a.win.QWidget)
	p.dlg.SetWindowTitle("Host Info")
	v := qt.NewQVBoxLayout(p.dlg.QWidget)
	box := qt.NewQWidget2()
	p.rowsForm = qt.NewQFormLayout(box)
	p.rowsForm.SetLabelAlignment(qt.AlignRight)
	v.AddWidget(box)
	for _, r := range a.hostInfo.Rows() {
		l := qt.NewQLabel2()
		l.SetTextInteractionFlags(qt.TextSelectableByMouse)
		p.values[r.Label] = l
		p.rowsForm.AddRow3(r.Label, l.QWidget)
	}
	for i := range a.hostInfo.Notes {
		l := qt.NewQLabel2()
		l.SetWordWrap(true)
		p.values["Note "+strconv.Itoa(i)] = l
		p.rowsForm.AddRowWithWidget(l.QWidget)
	}
	row := qt.NewQHBoxLayout2()
	p.revConf = qt.NewQPushButton3("Reveal Config Folder")
	p.revExp = qt.NewQPushButton3("Reveal Last Export")
	close := qt.NewQPushButton3("Close")
	row.AddWidget(p.revConf.QWidget)
	row.AddWidget(p.revExp.QWidget)
	row.AddStretch()
	row.AddWidget(close.QWidget)
	v.AddLayout(row.QLayout)
	p.revConf.OnClicked(a.revealConfigFolder)
	p.revExp.OnClicked(a.revealLastExport)
	close.OnClicked(func() { p.dlg.Close() })
	p.dlg.SetMinimumWidth(480)
	return p
}

func (p *hostPanel) refresh() {
	p.a.hostInfo = host.Current(appID)
	info := p.a.hostInfo
	for _, r := range info.Rows() {
		if l, ok := p.values[r.Label]; ok {
			l.SetText(r.Value)
		}
	}
	for i, n := range info.Notes {
		if l, ok := p.values["Note "+strconv.Itoa(i)]; ok {
			l.SetText(n)
		}
	}
}

func (p *hostPanel) updateButtons() {
	avail := p.a.hostInfo.Available()
	p.revConf.SetEnabled(avail)
	p.revExp.SetEnabled(avail && p.a.lastExport != "")
}

// revealConfigFolder opens the prefs directory with Qt's QDesktopServices.
func (a *App) revealConfigFolder() {
	dir := a.hostInfo.ConfigDir
	if dir == "" || !a.hostInfo.Available() {
		return
	}
	_ = os.MkdirAll(dir, 0o755)
	if !qt.QDesktopServices_OpenUrl(qt.QUrl_FromLocalFile(dir)) {
		_ = host.RevealInFileManager(dir)
	}
}

// revealLastExport selects the last exported file in Finder. Qt has no
// "reveal and select" call, so this uses the core helper (open -R).
func (a *App) revealLastExport() {
	if a.lastExport == "" || !a.hostInfo.Available() {
		return
	}
	if err := host.RevealInFileManager(a.lastExport); err != nil {
		a.dlg.errorBox("Reveal Last Export", err.Error())
	}
}
