//go:build !js

package main

import (
	"fmt"

	"github.com/go-gui-org/go-gui/gui"
)

// The native macOS menubar (go-gui NativeMenubarCfg). Items with a
// CommandID run the registered command, so the shortcut and the menu
// share one action. go-gui menu items have Checked but no radio type, so
// the View and Theme choices are checked items rebuilt when they change.

var guiApp *gui.App

func (a *app) menuKey() string {
	return fmt.Sprint(a.viewMode, a.themeSel, a.exp.running, a.exp.lastPath != "")
}

func (a *app) buildMenu() gui.NativeMenubarCfg {
	m := gui.ShortcutModifier()
	sc := func(k gui.KeyCode, mods gui.Modifier) gui.Shortcut { return gui.Shortcut{Key: k, Modifiers: mods} }
	theme := func(name string) gui.NativeMenuItemCfg {
		return gui.NativeMenuItemCfg{ID: "theme." + name, Text: name, Checked: a.themeSel == name}
	}
	return gui.NativeMenubarCfg{
		AppName:         "GGUIX",
		AboutActionID:   "about",
		IncludeEditMenu: true,
		Menus: []gui.NativeMenuCfg{
			{Title: "File", Items: []gui.NativeMenuItemCfg{
				{ID: "export", Text: "Export CSV…", CommandID: "export", Shortcut: sc(gui.KeyE, m), Disabled: a.exp.running},
				{ID: "import", Text: "Import Notes…", CommandID: "import", Shortcut: sc(gui.KeyO, m)},
				{ID: "reveal", Text: "Reveal Last Export", Disabled: a.exp.lastPath == ""},
				{Separator: true},
				// The auto-wired Edit menu cannot take extra items, so
				// Copy Row lives here.
				{ID: "copyrow", Text: "Copy Row", CommandID: "copyrow", Shortcut: sc(gui.KeyC, m|gui.ModShift)},
			}},
			{Title: "View", Items: []gui.NativeMenuItemCfg{
				{ID: "view.table", Text: "Table", CommandID: "view.table", Shortcut: sc(gui.Key1, m), Checked: a.viewMode == 0},
				{ID: "view.cards", Text: "Cards", CommandID: "view.cards", Shortcut: sc(gui.Key2, m), Checked: a.viewMode == 1},
				{Separator: true},
				{ID: "theme", Text: "Theme", Submenu: []gui.NativeMenuItemCfg{theme("System"), theme("Light"), theme("Dark")}},
				{Separator: true},
				{ID: "reload", Text: "Reload", CommandID: "reload", Shortcut: sc(gui.KeyR, m)},
				{ID: "focus.search", Text: "Focus Search", CommandID: "focus.search", Shortcut: sc(gui.KeyF, m)},
			}},
			{Title: "Help", Items: []gui.NativeMenuItemCfg{
				{ID: "host", Text: "Host Info…", CommandID: "host", Shortcut: sc(gui.KeyI, m)},
			}},
		},
		OnAction: func(id string) {
			a.post(func() {
				switch id {
				case "about":
					a.openAbout()
				case "reveal":
					a.reveal(a.exp.lastPath)
				case "theme.System", "theme.Light", "theme.Dark":
					a.setTheme(id[len("theme."):])
				}
			})
		},
	}
}

// platformFrame reinstalls the menubar when a checked or disabled state
// in it changed. It runs in the view function, so the install is queued.
func (a *app) platformFrame(w *gui.Window) {
	if guiApp == nil {
		return
	}
	k := a.menuKey()
	if k == a.menuState {
		return
	}
	a.menuState = k
	w.QueueCommand(func(*gui.Window) { guiApp.SetNativeMenubar(a.buildMenu()) })
}
