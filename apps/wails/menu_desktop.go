//go:build !server

package main

// The native application menu (category: desktop). Every item that does
// something in the UI sends a menu:action event; the frontend performs it.

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var menuItems struct {
	export, revealExport, copyRow    *application.MenuItem
	viewTable, viewCards             *application.MenuItem
	themeSystem, themeLight, themeDk *application.MenuItem
}

func buildMenu(app *application.App, svc *ItemService) *application.Menu {
	act := func(name string) func(*application.Context) {
		return func(*application.Context) { emitEvent("menu:action", MenuAction{Action: name}) }
	}
	menu := app.Menu.New()
	mac := runtime.GOOS == "darwin"

	// App menu (macOS) with About and Quit.
	appMenu := menu.AddSubmenu("GGUIX")
	appMenu.Add("About GGUIX").OnClick(act("about"))
	appMenu.AddSeparator()
	if mac {
		appMenu.AddRole(application.Hide)
		appMenu.AddRole(application.HideOthers)
		appMenu.AddRole(application.ShowAll)
		appMenu.AddSeparator()
	}
	appMenu.Add("Quit").SetAccelerator("CmdOrCtrl+Q").OnClick(func(*application.Context) { app.Quit() })

	file := menu.AddSubmenu("File")
	menuItems.export = file.Add("Export CSV…").SetAccelerator("CmdOrCtrl+E").OnClick(act("export"))
	file.Add("Import Notes…").SetAccelerator("CmdOrCtrl+O").OnClick(act("import-notes"))
	menuItems.revealExport = file.Add("Reveal Last Export").SetEnabled(false).OnClick(act("reveal-export"))

	edit := menu.AddSubmenu("Edit")
	edit.AddRole(application.Undo)
	edit.AddRole(application.Redo)
	edit.AddSeparator()
	edit.AddRole(application.Cut)
	edit.AddRole(application.Copy)
	edit.AddRole(application.Paste)
	edit.AddRole(application.SelectAll)
	edit.AddSeparator()
	menuItems.copyRow = edit.Add("Copy Row").SetAccelerator("CmdOrCtrl+Shift+C").SetEnabled(false).OnClick(act("copy-row"))

	view := menu.AddSubmenu("View")
	menuItems.viewTable = view.AddRadio("Table", true).SetAccelerator("CmdOrCtrl+1").OnClick(act("view-table"))
	menuItems.viewCards = view.AddRadio("Cards", false).SetAccelerator("CmdOrCtrl+2").OnClick(act("view-cards"))
	view.AddSeparator()
	theme := view.AddSubmenu("Theme")
	menuItems.themeSystem = theme.AddRadio("System", true).OnClick(act("theme-system"))
	menuItems.themeLight = theme.AddRadio("Light", false).OnClick(act("theme-light"))
	menuItems.themeDk = theme.AddRadio("Dark", false).OnClick(act("theme-dark"))
	view.AddSeparator()
	view.Add("Reload").SetAccelerator("CmdOrCtrl+R").OnClick(act("reload"))
	view.Add("Focus Search").SetAccelerator("CmdOrCtrl+F").OnClick(act("focus-search"))

	help := menu.AddSubmenu("Help")
	help.Add("Host Info…").SetAccelerator("CmdOrCtrl+I").OnClick(act("host-info"))
	return menu
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
