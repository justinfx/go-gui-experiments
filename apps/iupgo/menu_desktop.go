//go:build !js

package main

import (
	"github.com/gen2brain/iup-go/iup"
)

// buildMenu creates the menu bar. IUP maps it to the native menu bar on
// macOS (Cocoa adds its own application menu in front of the first menu)
// and to a window menu bar on GTK and Win32. A key description after a tab
// in the title is both shown and bound; Ctrl is Command on macOS.
func (a *app) buildMenu() {
	u := &a.ui
	item := func(title string, f func()) iup.Ihandle {
		it := iup.MenuItem(title)
		iup.On(it, iup.ActionCB, func(iup.Ihandle) int { f(); return iup.DEFAULT })
		return it
	}
	u.mExport = item("Export CSV…\tCtrl+E", a.beginExport)
	importItem := item("Import Notes…\tCtrl+O", a.beginImport)
	u.mReveal = item("Reveal Last Export", a.revealLastExport)
	set(u.mReveal, "ACTIVE", "NO")
	file := iup.Submenu("File", iup.Menu(u.mExport, importItem, u.mReveal))

	focusText := func(attr, val string) func() {
		return func() {
			if f := iup.GetFocus(); f != 0 && iup.ClassMatch(f, "text") {
				iup.SetAttribute(f, attr, val)
			}
		}
	}
	edit := iup.Submenu("Edit", iup.Menu(
		item("Cut\tCtrl+X", focusText("CLIPBOARD", "CUT")),
		item("Copy\tCtrl+C", focusText("CLIPBOARD", "COPY")),
		item("Paste\tCtrl+V", focusText("CLIPBOARD", "PASTE")),
		item("Select All\tCtrl+A", focusText("SELECTION", "ALL")),
		iup.MenuSeparator(),
		item("Copy Row\tCtrl+Shift+C", a.copyRow),
	))

	u.mTable = item("Table\tCtrl+1", func() { a.setViewFromUI(0) })
	u.mCards = item("Cards\tCtrl+2", func() { a.setViewFromUI(1) })
	set(u.mTable, "VALUE", "ON")
	set(u.mCards, "VALUE", "OFF")
	viewRadio := iup.Menu(u.mTable, u.mCards)
	set(viewRadio, "RADIO", "YES")
	var th []iup.Ihandle
	for i, name := range themeChoices {
		i, name := i, name
		u.mTheme[i] = item(name, func() { a.setTheme(name); a.uiSyncThemeControls() })
		set(u.mTheme[i], "VALUE", "OFF")
		th = append(th, u.mTheme[i])
	}
	themeMenu := iup.Menu(th...)
	set(themeMenu, "RADIO", "YES")
	view := iup.Submenu("View", iup.Menu(
		iup.Submenu("Layout", viewRadio),
		iup.Submenu("Theme", themeMenu),
		iup.MenuSeparator(),
		item("Reload\tCtrl+R", func() { a.rebuildQuery(true) }),
		item("Focus Search\tCtrl+F", func() { iup.SetFocus(u.search) }),
	))
	help := iup.Submenu("Help", iup.Menu(
		item("Host Info…\tCtrl+I", a.openHost),
		iup.MenuSeparator(),
		item("About GGUIX", a.openAbout),
		item("Quit\tCtrl+Q", func() { a.quit() }),
	))
	menu := iup.Menu(file, edit, view, help)
	iup.SetHandle("gguixmenu", menu)
	set(u.dlg, "MENU", "gguixmenu")
}
