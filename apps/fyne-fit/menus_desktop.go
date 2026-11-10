//go:build !js

package main

import (
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"gguix/core"
)

// The menu bar. Desktop target only.

type menuState struct {
	exporting, hasSel, hasLast, cards, hasForm bool
	theme                                      string
}

type menuItems struct {
	main                                         *fyne.MainMenu
	export, importN, reveal, copyRow             *fyne.MenuItem
	table, cardsI                                *fyne.MenuItem
	system, light, dark                          *fyne.MenuItem
	last                                         menuState
	built                                        bool
}

func (u *ui) shortcut(key fyne.KeyName, extra fyne.KeyModifier) fyne.Shortcut {
	mod := fyne.KeyModifierSuper
	if runtime.GOOS != "darwin" {
		mod = fyne.KeyModifierControl
	}
	return &desktop.CustomShortcut{KeyName: key, Modifier: mod | extra}
}

// sendShortcut delivers a clipboard shortcut to the focused widget.
func (u *ui) sendShortcut(s fyne.Shortcut) {
	if sc, ok := u.win.Canvas().Focused().(fyne.Shortcutable); ok {
		sc.TypedShortcut(s)
	}
}

func (u *ui) buildMenu() {
	m := &menuItems{}
	u.menus = m
	item := func(label string, key fyne.KeyName, extra fyne.KeyModifier, action func()) *fyne.MenuItem {
		it := fyne.NewMenuItem(label, action)
		if key != "" {
			it.Shortcut = u.shortcut(key, extra)
			if runtime.GOOS != "darwin" {
				// Native macOS menus handle their key equivalents; elsewhere
				// the canvas has to do it.
				u.win.Canvas().AddShortcut(it.Shortcut, func(fyne.Shortcut) { action() })
			}
		}
		return it
	}
	clip := u.win.Clipboard()
	m.export = item("Export CSV…", fyne.KeyE, 0, func() { u.startExport() })
	m.importN = item("Import Notes…", fyne.KeyO, 0, func() { u.importNotes() })
	m.reveal = item("Reveal Last Export", "", 0, func() { u.revealLastExport() })
	file := fyne.NewMenu("File", m.export, m.importN, fyne.NewMenuItemSeparator(), m.reveal)

	cut := fyne.NewMenuItem("Cut", func() { u.sendShortcut(&fyne.ShortcutCut{Clipboard: clip}) })
	cut.Shortcut = &fyne.ShortcutCut{}
	cp := fyne.NewMenuItem("Copy", func() { u.sendShortcut(&fyne.ShortcutCopy{Clipboard: clip}) })
	cp.Shortcut = &fyne.ShortcutCopy{}
	paste := fyne.NewMenuItem("Paste", func() { u.sendShortcut(&fyne.ShortcutPaste{Clipboard: clip}) })
	paste.Shortcut = &fyne.ShortcutPaste{}
	all := fyne.NewMenuItem("Select All", func() { u.sendShortcut(&fyne.ShortcutSelectAll{}) })
	all.Shortcut = &fyne.ShortcutSelectAll{}
	m.copyRow = item("Copy Row", fyne.KeyC, fyne.KeyModifierShift, func() { u.copyRow() })
	edit := fyne.NewMenu("Edit", cut, cp, paste, all, fyne.NewMenuItemSeparator(), m.copyRow)

	m.table = item("Table", fyne.Key1, 0, func() { u.setView(false) })
	m.cardsI = item("Cards", fyne.Key2, 0, func() { u.setView(true) })
	m.system = fyne.NewMenuItem("System", func() { u.setThemeMode(core.ThemeSystem) })
	m.light = fyne.NewMenuItem("Light", func() { u.setThemeMode(core.ThemeLight) })
	m.dark = fyne.NewMenuItem("Dark", func() { u.setThemeMode(core.ThemeDark) })
	themeItem := fyne.NewMenuItem("Theme", nil)
	themeItem.ChildMenu = fyne.NewMenu("Theme", m.system, m.light, m.dark)
	reload := item("Reload", fyne.KeyR, 0, func() { u.queryChanged() })
	focus := item("Focus Search", fyne.KeyF, 0, func() { u.win.Canvas().Focus(u.searchEntry) })
	view := fyne.NewMenu("View", m.table, m.cardsI, fyne.NewMenuItemSeparator(), themeItem,
		fyne.NewMenuItemSeparator(), reload, focus)

	hostInfo := item("Host Info…", fyne.KeyI, 0, func() { u.showHostDialog() })
	// Fyne moves a menu item labelled exactly "About" into the macOS app menu.
	aboutLabel := "About GGUIX"
	if runtime.GOOS == "darwin" {
		aboutLabel = "About"
	}
	about := fyne.NewMenuItem(aboutLabel, func() { u.showAbout() })
	help := fyne.NewMenu("Help", hostInfo, about)
	if runtime.GOOS != "darwin" {
		quit := item("Quit", fyne.KeyQ, 0, func() { u.app.Quit() })
		quit.IsQuit = true
		help.Items = append(help.Items, fyne.NewMenuItemSeparator(), quit)
	}

	m.main = fyne.NewMainMenu(file, edit, view, help)
	u.win.SetMainMenu(m.main)
	m.built = true
}

// updateMenus syncs the enabled and checked state of menu items.
func (u *ui) updateMenus() {
	m := u.menus
	if m == nil || !m.built {
		return
	}
	st := menuState{
		exporting: u.exporting, hasSel: u.form.has, hasLast: u.lastExport != "",
		cards: u.cards, theme: u.themeMod,
	}
	if st == m.last {
		return
	}
	m.last = st
	m.export.Disabled = st.exporting
	m.importN.Disabled = !st.hasSel
	m.copyRow.Disabled = !st.hasSel
	m.reveal.Disabled = !st.hasLast
	m.table.Checked, m.cardsI.Checked = !st.cards, st.cards
	m.system.Checked = st.theme == core.ThemeSystem
	m.light.Checked = st.theme == core.ThemeLight
	m.dark.Checked = st.theme == core.ThemeDark
	m.main.Refresh()
}
