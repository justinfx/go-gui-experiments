//go:build !js

package main

import (
	"runtime"
	"runtime/debug"
	"strings"

	"gguix/core"

	qt "github.com/mappu/miqt/qt6"
)

type menuActions struct {
	exportAct    *qt.QAction
	importAct    *qt.QAction
	revealExport *qt.QAction
	copyRowAct   *qt.QAction
	tableAct     *qt.QAction
	cardsAct     *qt.QAction
	themeActs    []*qt.QAction
	all          map[string]*qt.QAction
}

// editable gives the Edit menu a uniform way to reach the focused text field.
type editable interface {
	HasFocus() bool
	Cut()
	Copy()
	Paste()
	SelectAll()
}

func (a *App) buildMenus() {
	mb := a.win.MenuBar()
	m := &menuActions{all: map[string]*qt.QAction{}}
	a.menus = m

	newAct := func(menu *qt.QMenu, text, shortcut string, fn func()) *qt.QAction {
		act := qt.NewQAction5(text, menu.QObject)
		if shortcut != "" {
			act.SetShortcut(qt.NewQKeySequence2(shortcut))
		}
		act.OnTriggered(fn)
		menu.QWidget.AddAction(act)
		m.all[text] = act
		return act
	}

	// App menu on macOS (Qt moves the role-tagged actions there); Help elsewhere.
	appMenu := mb.AddMenuWithTitle("GGUIX")
	about := newAct(appMenu, "About GGUIX", "", a.showAbout)
	about.SetMenuRole(qt.QAction__AboutRole)
	quit := newAct(appMenu, "Quit", "Ctrl+Q", a.quit)
	quit.SetMenuRole(qt.QAction__QuitRole)

	file := mb.AddMenuWithTitle("File")
	m.exportAct = newAct(file, "Export CSV…", "Ctrl+E", a.startExportInteractive)
	m.importAct = newAct(file, "Import Notes…", "Ctrl+O", func() { a.form.doImport() })
	m.revealExport = newAct(file, "Reveal Last Export", "", a.revealLastExport)
	m.revealExport.SetEnabled(false)

	edit := mb.AddMenuWithTitle("Edit")
	focused := func() editable {
		for _, e := range []editable{a.search, a.goEdit, a.form.name, a.form.price, a.form.notes} {
			if e.HasFocus() {
				return e
			}
		}
		return nil
	}
	newAct(edit, "Cut", "Ctrl+X", func() {
		if e := focused(); e != nil {
			e.Cut()
		}
	})
	newAct(edit, "Copy", "Ctrl+C", func() {
		if e := focused(); e != nil {
			e.Copy()
		}
	})
	newAct(edit, "Paste", "Ctrl+V", func() {
		if e := focused(); e != nil {
			e.Paste()
		}
	})
	newAct(edit, "Select All", "Ctrl+A", func() {
		if e := focused(); e != nil {
			e.SelectAll()
		}
	})
	edit.AddSeparator()
	m.copyRowAct = newAct(edit, "Copy Row", "Ctrl+Shift+C", func() { a.form.doCopyRow() })

	view := mb.AddMenuWithTitle("View")
	viewGroup := qt.NewQActionGroup(view.QObject)
	m.tableAct = newAct(view, "Table", "Ctrl+1", func() { a.setView(false, true) })
	m.cardsAct = newAct(view, "Cards", "Ctrl+2", func() { a.setView(true, true) })
	for _, act := range []*qt.QAction{m.tableAct, m.cardsAct} {
		act.SetCheckable(true)
		viewGroup.AddAction(act)
	}
	view.AddSeparator()
	themeMenu := view.AddMenuWithTitle("Theme")
	themeGroup := qt.NewQActionGroup(themeMenu.QObject)
	for i, name := range []string{"System", "Light", "Dark"} {
		mode := []string{core.ThemeSystem, core.ThemeLight, core.ThemeDark}[i]
		act := newAct(themeMenu, name, "", func() {
			if a.suppress == 0 {
				a.applyTheme(mode, true)
			}
		})
		act.SetCheckable(true)
		themeGroup.AddAction(act)
		m.themeActs = append(m.themeActs, act)
	}
	view.AddSeparator()
	newAct(view, "Reload", "Ctrl+R", a.reload)
	newAct(view, "Focus Search", "Ctrl+F", a.focusSearch)

	help := mb.AddMenuWithTitle("Help")
	newAct(help, "Host Info…", "Ctrl+I", a.showHostPanel)
	if runtime.GOOS != "darwin" {
		// Outside macOS there is no app menu: keep About in Help as well.
		newAct(help, "About GGUIX ", "", a.showAbout)
	}
}

// frameworkVersion reports the MIQT module version and the Qt runtime version.
func frameworkVersion() string {
	miqt := "unknown"
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, d := range bi.Deps {
			if d.Path == "github.com/mappu/miqt" {
				miqt = d.Version
			}
		}
	}
	return "MIQT " + miqt + " / Qt " + qt.QLibraryInfo_Version().ToString()
}

func aboutLines() []string {
	return []string{
		"GGUIX - MIQT",
		frameworkVersion(),
		runtime.Version(),
		"BuildMarker " + BuildMarker,
	}
}

func (a *App) showAbout() {
	if a.aboutBox == nil {
		b := qt.NewQMessageBox(a.win.QWidget)
		b.SetWindowTitle("About GGUIX")
		b.SetTextFormat(qt.PlainText)
		b.SetStandardButtons(qt.QMessageBox__Ok)
		a.aboutBox = b
	}
	a.aboutBox.SetText(strings.Join(aboutLines(), "\n"))
	a.aboutBox.Open()
}
