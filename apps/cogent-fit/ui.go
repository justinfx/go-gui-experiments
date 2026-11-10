package main

// Window layout shared by desktop and web: toolbar row, split main area,
// status bar (brief sections 5.1 and 5.2).

import (
	"image"
	"runtime"
	"strconv"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"

	gc "gguix/core"
	"gguix/core/host"
)

var statusFilterNames = []string{"All statuses", "Active", "Backordered", "Discontinued"}
var themeNames = []string{"System", "Light", "Dark"}

func themeIndex(t string) int {
	switch t {
	case gc.ThemeLight:
		return 1
	case gc.ThemeDark:
		return 2
	}
	return 0
}

// buildUI creates every widget of the main window.
func (a *App) buildUI(b *core.Body) {
	a.body = b
	a.addMenuBar(b)
	a.tools = newToolbar(a, b)

	sp := core.NewSplits(b).SetSplits(0.65, 0.35)
	sp.Styler(func(s *styles.Style) { s.Grow.Set(1, 1) })

	a.stack = core.NewFrame(sp)
	a.stack.Styler(func(s *styles.Style) {
		s.Display = styles.Stacked
		s.Grow.Set(1, 1)
		s.Min.X.Dp(300)
	})
	a.table = tree.New[rowsTable](a.stack)
	a.table.app = a
	a.cards = tree.New[cardsList](a.stack)
	a.cards.app = a
	ef := core.NewFrame(a.stack)
	ef.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Justify.Content = styles.Center
		s.Align.Items = styles.Center
	})
	a.empty = core.NewText(ef).SetType(core.TextBodyLarge).SetText("No items match.")
	a.stack.StackTop = int(a.view)

	a.form = newDetailForm(a, sp)
	a.status = newStatusBar(a, b)
	a.tools.sync()
	for _, w := range []core.Widget{a.tools.search, a.tools.goTo, a.form.name, a.form.price, a.form.restock, a.form.notes} {
		w := w
		w.AsWidget().OnFocus(func(e events.Event) { a.lastText = w })
	}
}

// start runs once the window is shown.
func (a *App) start() {
	a.setThemeNoSave(a.prefs.Theme)
	a.tools.sync()
	a.applyQuery()
	a.restoreSelection()
	go func() {
		info := host.Current(appID)
		a.async(func() {
			a.hostInfo = info
			a.status.update()
			a.status.root.Update()
		})
	}()
}

func (a *App) setThemeNoSave(t string) {
	switch t {
	case gc.ThemeLight:
		core.AppearanceSettings.Theme = core.ThemeLight
	case gc.ThemeDark:
		core.AppearanceSettings.Theme = core.ThemeDark
	default:
		core.AppearanceSettings.Theme = core.ThemeAuto
	}
	core.AppearanceSettings.Apply()
	core.UpdateAll()
	a.refreshMenus()
}

// aboutLines are the lines of the About dialog.
func aboutLines() []string {
	return []string{
		windowTitle,
		"Cogent Core " + system.CoreVersion,
		runtime.Version(),
		"BuildMarker " + BuildMarker,
	}
}

//////// Toolbar

type toolbarParts struct {
	app     *App
	root    *core.Frame
	search  *core.TextField
	catBtn  *core.Button
	statusC *core.Chooser
	goTo    *core.TextField
	goBtn   *core.Button
	views   *core.Switches
	exportB *core.Button
	themeC  *core.Chooser
}

func newToolbar(a *App, parent core.Widget) *toolbarParts {
	t := &toolbarParts{app: a}
	t.root = core.NewFrame(parent)
	t.root.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Grow.Set(1, 0)
		s.Gap.Set(units.Dp(8))
		s.Align.Items = styles.Center
		s.Wrap = true
	})

	t.search = core.NewTextField(t.root).SetPlaceholder("Search name or SKU")
	t.search.Styler(func(s *styles.Style) {
		s.Min.X.Dp(280)
		s.Max.X.Dp(280)
		s.Grow.Set(0, 0)
	})
	t.search.OnInput(func(e events.Event) { a.setSearch(t.search.Text(), false) })

	t.catBtn = core.NewButton(t.root).SetText("Categories").SetType(core.ButtonOutlined)
	t.catBtn.SetMenu(func(m *core.Scene, pos image.Point) {
		for _, c := range gc.AllCategories {
			sw := core.NewSwitch(m).SetType(core.SwitchCheckbox).SetText(c.String())
			sw.SetChecked(hasCategory(a.prefs.Categories, c))
			sw.OnChange(func(e events.Event) {
				cats := toggleCategory(a.prefs.Categories, c, sw.IsChecked())
				a.setCategories(cats)
			})
		}
		core.NewSeparator(m)
		core.NewButton(m).SetText("Clear").OnClick(func(e events.Event) {
			a.setCategories(nil)
		})
	})

	t.statusC = core.NewChooser(t.root).SetStrings(statusFilterNames...)
	t.statusC.OnChange(func(e events.Event) {
		i := t.statusC.CurrentIndex
		if i <= 0 {
			a.setStatus(nil)
			return
		}
		st := gc.Status(i - 1)
		a.setStatus(&st)
	})

	t.goTo = core.NewTextField(t.root).SetPlaceholder("Go to row")
	t.goTo.Styler(func(s *styles.Style) {
		s.Min.X.Dp(110)
		s.Max.X.Dp(130)
		s.Grow.Set(0, 0)
	})
	t.goTo.OnKeyChord(func(e events.Event) {
		if e.KeyChord() == "ReturnEnter" {
			e.SetHandled()
			a.goToRowText(t.goTo.Text())
		}
	})
	t.goBtn = core.NewButton(t.root).SetText("Go")
	t.goBtn.OnClick(func(e events.Event) { a.goToRowText(t.goTo.Text()) })

	t.views = core.NewSwitches(t.root).SetType(core.SwitchSegmentedButton).SetMutex(true).SetStrings("Table", "Cards")
	t.views.OnChange(func(e events.Event) {
		if it := t.views.SelectedItem(); it != nil {
			if it.Value == "Cards" {
				a.setView(viewCards)
			} else {
				a.setView(viewTable)
			}
		}
	})

	core.NewStretch(t.root)

	t.exportB = core.NewButton(t.root).SetText("Export CSV")
	t.exportB.OnClick(func(e events.Event) { a.startExport() })

	t.themeC = core.NewChooser(t.root).SetStrings(themeNames...)
	t.themeC.OnChange(func(e events.Event) {
		a.setTheme([]string{gc.ThemeSystem, gc.ThemeLight, gc.ThemeDark}[t.themeC.CurrentIndex])
	})
	return t
}

func hasCategory(cs []gc.Category, c gc.Category) bool {
	for _, x := range cs {
		if x == c {
			return true
		}
	}
	return false
}

func toggleCategory(cs []gc.Category, c gc.Category, on bool) []gc.Category {
	var out []gc.Category
	for _, x := range gc.AllCategories { // keep AllCategories order
		if x == c {
			if on {
				out = append(out, x)
			}
		} else if hasCategory(cs, x) {
			out = append(out, x)
		}
	}
	return out
}

// sync updates the toolbar widgets from the preferences.
func (t *toolbarParts) sync() {
	a := t.app
	label := "Categories"
	if n := len(a.prefs.Categories); n > 0 {
		label = "Categories (" + strconv.Itoa(n) + ")"
	}
	t.catBtn.SetText(label)
	t.catBtn.NeedsRender()
	idx := 0
	if a.prefs.Status != nil {
		idx = int(*a.prefs.Status) + 1
	}
	t.statusC.SetCurrentIndex(idx)
	t.themeC.SetCurrentIndex(themeIndex(a.prefs.Theme))
	if a.view == viewCards {
		t.views.SelectValue("Cards")
	} else {
		t.views.SelectValue("Table")
	}
	if t.search.Text() != a.prefs.Search && !t.search.StateIs(states.Focused) {
		t.search.SetText(a.prefs.Search)
	}
	t.exportB.SetEnabled(!a.exp.running)
	t.root.UpdateRender()
}

//////// Status bar

type statusBar struct {
	app       *App
	root      *core.Frame
	text      *core.Text
	exportBox *core.Frame
	meter     *core.Meter
	pct       *core.Text
	cancel    *core.Button
	ident     *core.Text
	hostB     *core.Button
}

func newStatusBar(a *App, parent core.Widget) *statusBar {
	s := &statusBar{app: a}
	s.root = core.NewFrame(parent)
	s.root.Styler(func(st *styles.Style) {
		st.Direction = styles.Row
		st.Grow.Set(1, 0)
		st.Gap.Set(units.Dp(12))
		st.Align.Items = styles.Center
	})
	s.text = core.NewText(s.root).SetType(core.TextBodyMedium)
	core.NewStretch(s.root)

	s.exportBox = core.NewFrame(s.root)
	s.exportBox.Styler(func(st *styles.Style) {
		st.Direction = styles.Row
		st.Gap.Set(units.Dp(8))
		st.Align.Items = styles.Center
		st.Grow.Set(0, 0)
		if !a.exp.running {
			st.Display = styles.DisplayNone
		}
	})
	s.meter = core.NewMeter(s.exportBox).SetType(core.MeterLinear).SetMin(0).SetMax(100)
	s.meter.Styler(func(st *styles.Style) {
		st.Min.X.Dp(200)
		st.Max.X.Dp(200)
		st.Grow.Set(0, 0)
	})
	s.pct = core.NewText(s.exportBox).SetType(core.TextBodyMedium).SetText("0%")
	s.cancel = core.NewButton(s.exportBox).SetText("Cancel").SetType(core.ButtonOutlined)
	s.cancel.OnClick(func(e events.Event) { a.cancelExport() })

	s.ident = core.NewText(s.root).SetType(core.TextBodyMedium)
	s.hostB = core.NewButton(s.root).SetText("Host").SetType(core.ButtonOutlined)
	s.hostB.OnClick(func(e events.Event) { a.showHost() })
	s.update()
	return s
}

// update refreshes the status bar texts (UI thread).
func (s *statusBar) update() {
	a := s.app
	if t := a.statusText(); t != s.text.Text {
		s.text.SetText(t)
		s.text.NeedsLayout()
	}
	if t := a.hostInfo.Identity(); t != s.ident.Text {
		s.ident.SetText(t)
		s.ident.NeedsLayout()
	}
	if a.exp.running {
		pct := a.exp.percent()
		s.meter.SetValue(float32(pct))
		s.meter.NeedsRender()
		s.pct.SetText(strconv.Itoa(pct) + "%")
		s.pct.NeedsRender()
	}
}

// StatusText returns the left status text (for tests).
func (s *statusBar) StatusText() string { return s.text.Text }

// IdentText returns the identity text (for tests).
func (s *statusBar) IdentText() string { return s.ident.Text }
