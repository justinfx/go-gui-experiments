//go:build !js

package main

import (
	"runtime"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"

	"gguix/core/host"
)

// installPlatform wires everything only the desktop has: the hooks, the
// application menu (with its shortcuts) and the saved window size.
func (a *app) installPlatform(cx *core.Scope) {
	a.plat = desktopHooks(cx)
	desktop.SetTitle(cx, "GGUIX - NSGo")

	// The menu has checked items, which are plain bools in MenuItem, so it is
	// built again whenever the view or the theme changes.
	cx.Effect(func() {
		a.rQuery.Watch()
		a.rExp.Watch()
		cx.Untrack(func() { desktop.SetApplicationMenu(cx, a.menu()) })
	})

	// Window size: remember the content size as the window is resized.
	cx.Effect(func() {
		f := desktop.Frame(cx)
		if f.Width > 0 && f.Height > 0 {
			cx.Untrack(func() { a.setWindowSize(int(f.Width), int(f.Height)) })
		}
	})
}

func (a *app) menu() []core.MenuItem {
	about := core.MenuItem{Label: "About GGUIX", OnSelect: func() { a.aboutOpen = true; a.rDlg.Bump() }}
	quit := core.MenuItem{Label: "Quit", Role: core.MenuQuit, Shortcut: "mod+q"}
	hostInfo := core.MenuItem{Label: "Host Info…", Shortcut: "mod+i", OnSelect: func() { a.hostOpen = true; a.rDlg.Bump() }}

	themes := []core.MenuItem{}
	for _, t := range themeChoices {
		t := t
		themes = append(themes, core.MenuItem{Label: t, Checkable: true, Checked: a.themeSel == t, OnSelect: func() { a.setTheme(t) }})
	}
	file := core.MenuItem{Label: "File", Items: []core.MenuItem{
		{Label: "Export CSV…", Shortcut: "mod+e", Disabled: a.exp.running, OnSelect: a.startExport},
		{Label: "Import Notes…", Shortcut: "mod+o", OnSelect: a.beginImport},
		{Label: "Reveal Last Export", Disabled: a.exp.lastPath == "", OnSelect: a.revealExport},
	}}
	edit := core.MenuItem{Label: "Edit", Items: []core.MenuItem{
		{Label: "Cut", Role: core.MenuCut},
		{Label: "Copy", Role: core.MenuCopy},
		{Label: "Paste", Role: core.MenuPaste},
		{Label: "Select All", Role: core.MenuSelectAll},
		{Separator: true},
		{Label: "Copy Row", Shortcut: "mod+shift+c", OnSelect: a.copyRow},
	}}
	view := core.MenuItem{Label: "View", Items: []core.MenuItem{
		{Label: "Table", Shortcut: "mod+1", Checkable: true, Checked: a.viewMode == 0, OnSelect: func() { a.setView(0) }},
		{Label: "Cards", Shortcut: "mod+2", Checkable: true, Checked: a.viewMode == 1, OnSelect: func() { a.setView(1) }},
		{Separator: true},
		{Label: "Theme", Items: themes},
		{Separator: true},
		{Label: "Reload", Shortcut: "mod+r", OnSelect: func() { a.rebuildQuery(true) }},
		{Label: "Focus Search", Shortcut: "mod+f", OnSelect: func() {
			if a.focusSearchFn != nil {
				a.focusSearchFn()
			}
		}},
	}}
	if runtime.GOOS == "darwin" {
		return []core.MenuItem{
			{Label: "GGUIX", Items: []core.MenuItem{about, {Separator: true}, quit}},
			file, edit, view,
			{Label: "Help", Items: []core.MenuItem{hostInfo}},
		}
	}
	return []core.MenuItem{file, edit, view,
		{Label: "Help", Items: []core.MenuItem{about, hostInfo, {Separator: true}, quit}},
	}
}

var _ = host.Current
