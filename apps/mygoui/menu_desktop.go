//go:build !js

package main

import (
	"runtime"

	"github.com/egoist/mygo"
)

// buildMenu makes the native menu bar and the hook that keeps its radio
// items and enabled states in step with the app.
func (a *app) buildMenu() *mygo.Menu {
	click := func(f func()) func(*mygo.MenuItem, *mygo.Window) {
		return func(*mygo.MenuItem, *mygo.Window) { f(); a.wake() }
	}
	themeItem := func(label string) *mygo.MenuItem {
		return &mygo.MenuItem{ID: "theme-" + label, Label: label, Type: mygo.MenuItemRadio, Checked: a.themeSel == label,
			Click: click(func() { a.themeSel = label })}
	}
	about := &mygo.MenuItem{Label: "About GGUIX", Click: click(func() { a.aboutOpen = true })}
	var top []*mygo.MenuItem
	if runtime.GOOS == "darwin" {
		top = append(top, &mygo.MenuItem{Label: "GGUIX", Submenu: []*mygo.MenuItem{
			about, mygo.Separator(), {Role: mygo.RoleQuit},
		}})
	}
	file := []*mygo.MenuItem{
		{ID: "export", Label: "Export CSV…", Accelerator: "CmdOrCtrl+E", Click: click(a.beginExport)},
		{Label: "Import Notes…", Accelerator: "CmdOrCtrl+O", Click: click(a.beginImport)},
		{ID: "reveal", Label: "Reveal Last Export", Disabled: true, Click: click(func() { a.reveal(a.exp.lastPath) })},
	}
	if runtime.GOOS != "darwin" {
		file = append(file, mygo.Separator(), &mygo.MenuItem{Role: mygo.RoleQuit})
	}
	help := []*mygo.MenuItem{{Label: "Host Info…", Accelerator: "CmdOrCtrl+I", Click: click(func() { a.hostOpen = true })}}
	if runtime.GOOS != "darwin" {
		help = append([]*mygo.MenuItem{about, mygo.Separator()}, help...)
	}
	top = append(top,
		&mygo.MenuItem{Label: "File", Submenu: file},
		&mygo.MenuItem{Label: "Edit", Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleUndo}, {Role: mygo.RoleRedo}, mygo.Separator(),
			{Role: mygo.RoleCut}, {Role: mygo.RoleCopy}, {Role: mygo.RolePaste}, {Role: mygo.RoleSelectAll},
			mygo.Separator(),
			{Label: "Copy Row", Accelerator: "CmdOrCtrl+Shift+C", Click: click(a.copyRow)},
		}},
		&mygo.MenuItem{Label: "View", Submenu: []*mygo.MenuItem{
			{ID: "view-table", Label: "Table", Type: mygo.MenuItemRadio, Accelerator: "CmdOrCtrl+1", Checked: a.viewMode == 0,
				Click: click(func() { a.setView(0) })},
			{ID: "view-cards", Label: "Cards", Type: mygo.MenuItemRadio, Accelerator: "CmdOrCtrl+2", Checked: a.viewMode == 1,
				Click: click(func() { a.setView(1) })},
			mygo.Separator(),
			{Label: "Theme", Submenu: []*mygo.MenuItem{themeItem("System"), themeItem("Light"), themeItem("Dark")}},
			mygo.Separator(),
			{Label: "Reload", Accelerator: "CmdOrCtrl+R", Click: click(func() { a.rebuildQuery(true) })},
			{Label: "Focus Search", Accelerator: "CmdOrCtrl+F", Click: click(func() { a.focusSrch = true })},
		}},
		&mygo.MenuItem{Label: "Help", Submenu: help},
	)
	menu := mygo.NewMenu(top)

	last := struct {
		view      int
		theme     string
		exporting bool
		hasLast   bool
	}{view: -1}
	a.menuSync = func() {
		if last.view != a.viewMode {
			last.view = a.viewMode
			menu.ItemByID("view-table").SetChecked(a.viewMode == 0)
			menu.ItemByID("view-cards").SetChecked(a.viewMode == 1)
		}
		if last.theme != a.themeSel {
			last.theme = a.themeSel
			for _, t := range themeChoices {
				menu.ItemByID("theme-" + t).SetChecked(a.themeSel == t)
			}
		}
		if last.exporting != a.exp.running {
			last.exporting = a.exp.running
			menu.ItemByID("export").SetEnabled(!a.exp.running)
		}
		if has := a.exp.lastPath != ""; last.hasLast != has {
			last.hasLast = has
			menu.ItemByID("reveal").SetEnabled(has)
		}
	}
	return menu
}
