//go:build !e2e

package main

// The native application menu (category: desktop). Every item that does
// something in the UI sends a menu:action event; the frontend performs it.

import (
	"github.com/egoist/mygo"
)

var menuItems struct {
	export, revealExport, copyRow    *mygo.MenuItem
	viewTable, viewCards             *mygo.MenuItem
	themeSystem, themeLight, themeDk *mygo.MenuItem
}

func buildMenu(svc *ItemService) *mygo.Menu {
	act := func(name string) func(*mygo.MenuItem, *mygo.Window) {
		return func(*mygo.MenuItem, *mygo.Window) { emitEvent("menu:action", MenuAction{Action: name}) }
	}
	item := func(label, accel, action string) *mygo.MenuItem {
		return &mygo.MenuItem{Label: label, Accelerator: accel, Click: act(action)}
	}
	radio := func(label, accel, action string, checked bool) *mygo.MenuItem {
		return &mygo.MenuItem{Label: label, Accelerator: accel, Type: mygo.MenuItemRadio, Checked: checked, Click: act(action)}
	}
	exp := item("Export CSV…", "CmdOrCtrl+E", "export")
	rev := item("Reveal Last Export", "", "reveal-export")
	rev.Disabled = true
	cpy := item("Copy Row", "CmdOrCtrl+Shift+C", "copy-row")
	cpy.Disabled = true
	menuItems.export, menuItems.revealExport, menuItems.copyRow = exp, rev, cpy
	menuItems.viewTable = radio("Table", "CmdOrCtrl+1", "view-table", true)
	menuItems.viewCards = radio("Cards", "CmdOrCtrl+2", "view-cards", false)
	menuItems.themeSystem = radio("System", "", "theme-system", true)
	menuItems.themeLight = radio("Light", "", "theme-light", false)
	menuItems.themeDk = radio("Dark", "", "theme-dark", false)

	return mygo.NewMenu([]*mygo.MenuItem{
		{Label: "GGUIX", Submenu: []*mygo.MenuItem{
			item("About GGUIX", "", "about"),
			mygo.Separator(),
			{Role: mygo.RoleHide},
			{Role: mygo.RoleHideOthers},
			{Role: mygo.RoleUnhide},
			mygo.Separator(),
			{Role: mygo.RoleQuit, Accelerator: "CmdOrCtrl+Q"},
		}},
		{Label: "File", Submenu: []*mygo.MenuItem{
			exp,
			item("Import Notes…", "CmdOrCtrl+O", "import-notes"),
			rev,
		}},
		{Label: "Edit", Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleUndo},
			{Role: mygo.RoleRedo},
			mygo.Separator(),
			{Role: mygo.RoleCut},
			{Role: mygo.RoleCopy},
			{Role: mygo.RolePaste},
			{Role: mygo.RoleSelectAll},
			mygo.Separator(),
			cpy,
		}},
		{Label: "View", Submenu: []*mygo.MenuItem{
			menuItems.viewTable,
			menuItems.viewCards,
			mygo.Separator(),
			{Label: "Theme", Submenu: []*mygo.MenuItem{menuItems.themeSystem, menuItems.themeLight, menuItems.themeDk}},
			mygo.Separator(),
			item("Reload", "CmdOrCtrl+R", "reload"),
			item("Focus Search", "CmdOrCtrl+F", "focus-search"),
		}},
		{Label: "Help", Submenu: []*mygo.MenuItem{
			item("Host Info…", "CmdOrCtrl+I", "host-info"),
		}},
	})
}

func menuStateUpdater() func(MenuState) {
	return func(st MenuState) {
		menuItems.viewTable.SetChecked(st.View != "cards")
		menuItems.viewCards.SetChecked(st.View == "cards")
		menuItems.themeSystem.SetChecked(st.Theme != "light" && st.Theme != "dark")
		menuItems.themeLight.SetChecked(st.Theme == "light")
		menuItems.themeDk.SetChecked(st.Theme == "dark")
		menuItems.export.SetEnabled(!st.ExportRunning)
		menuItems.revealExport.SetEnabled(st.HasExport)
		menuItems.copyRow.SetEnabled(st.HasSelection)
	}
}
