package main

import (
	"runtime"
	"strconv"
	"sync"

	"github.com/crgimenes/glaze"
	"github.com/crgimenes/glaze/menu"
)

// menuState is what the page pushes to the native menu (radio checks and the
// enabled flags). glaze/menu items have no checked state and no mutators, so a
// change rebuilds the whole menu and a check is drawn as a leading mark.
type menuState struct {
	view, theme          string
	exporting, hasExport bool
}

type nativeMenu struct {
	w    glaze.WebView
	mu   sync.Mutex
	st   menuState
	menu *menu.Menu
}

// act runs a page action, the same GGUIX.actions[...] the in-page menu and
// the keyboard use (static/app.js).
func (m *nativeMenu) act(name string) func() {
	js := "(function(){var f=window.GGUIX&&GGUIX.actions&&GGUIX.actions[" + strconv.Quote(name) + "];if(f)f();})()"
	return func() { m.w.Eval(js) }
}

func mark(on bool, s string) string {
	if on {
		return "✓ " + s
	}
	return "   " + s
}

func (m *nativeMenu) items() []menu.Item {
	s := m.st
	it := func(title, short, action string) menu.Item {
		return menu.Item{Title: title, Shortcut: short, OnClick: m.act(action)}
	}
	return []menu.Item{
		{Title: "GGUIX", Submenu: []menu.Item{
			it("About GGUIX", "", "about"),
			{Separator: true},
			it("Quit GGUIX", "cmd+q", "quit"),
		}},
		{Title: "File", Submenu: []menu.Item{
			{Title: "Export CSV…", Shortcut: "cmd+e", OnClick: m.act("export"), Disabled: s.exporting},
			it("Import Notes…", "cmd+o", "import-notes"),
			{Title: "Reveal Last Export", OnClick: m.act("reveal-export"), Disabled: !s.hasExport},
		}},
		{Title: "Edit", Submenu: []menu.Item{
			{Title: "Cut", Shortcut: "cmd+x", Selector: "cut:"},
			{Title: "Copy", Shortcut: "cmd+c", Selector: "copy:"},
			{Title: "Paste", Shortcut: "cmd+v", Selector: "paste:"},
			{Title: "Select All", Shortcut: "cmd+a", Selector: "selectAll:"},
			{Separator: true},
			it("Copy Row", "cmd+shift+c", "copy-row"),
		}},
		{Title: "View", Submenu: []menu.Item{
			it(mark(s.view == "table", "Table"), "cmd+1", "view-table"),
			it(mark(s.view == "cards", "Cards"), "cmd+2", "view-cards"),
			{Separator: true},
			{Title: "Theme", Submenu: []menu.Item{
				it(mark(s.theme == "system", "System"), "", "theme-system"),
				it(mark(s.theme == "light", "Light"), "", "theme-light"),
				it(mark(s.theme == "dark", "Dark"), "", "theme-dark"),
			}},
			{Separator: true},
			it("Reload", "cmd+r", "reload"),
			it("Focus Search", "cmd+f", "focus-search"),
		}},
		{Title: "Help", Submenu: []menu.Item{
			it("Host Info…", "cmd+i", "host"),
		}},
	}
}

// installNativeMenu installs the menu before the run loop starts (Dispatch must
// stay nil then, see menu.Options) and binds gguixMenuState for the page.
// It reports whether a menu was installed.
func installNativeMenu(w glaze.WebView) bool {
	if runtime.GOOS == "linux" {
		// glaze/menu draws no bar on Linux and would only add duplicate
		// shortcut handlers, so keep the in-page menubar and app.js shortcuts.
		return false
	}
	m := &nativeMenu{w: w, st: menuState{view: "table", theme: "system"}}
	h, err := menu.Set(m.items(), menu.Options{Window: w.Window()})
	if err != nil {
		return false // e.g. ErrUnsupported: the page draws its own menubar
	}
	m.menu = h
	_ = w.Bind("gguixMenuState", func(view, theme string, exporting, hasExport bool) {
		next := menuState{view, theme, exporting, hasExport}
		m.mu.Lock()
		defer m.mu.Unlock()
		if next == m.st {
			return
		}
		m.st = next
		// Runs on a Bind goroutine with the loop already running, so the
		// dispatcher is allowed here.
		if nh, err := menu.Set(m.items(), menu.Options{Window: w.Window(), Dispatch: w.Dispatch}); err == nil {
			m.menu = nh
		}
	})
	return true
}
