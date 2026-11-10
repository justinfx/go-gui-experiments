//go:build !js

package main

import (
	"image"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"

	gc "gguix/core"
)

// The framework draws its menus inside the window (no native menu bar).

func (a *App) refreshMenus() {}

func (a *App) addMenuBar(b *core.Body) {
	bar := core.NewFrame(b)
	bar.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Grow.Set(1, 0)
		s.Gap.Zero()
		s.Padding.Zero()
		s.Align.Items = styles.Center
	})
	menu := func(name string, fill func(m *core.Scene)) {
		core.NewButton(bar).SetText(name).SetType(core.ButtonAction).SetMenu(func(m *core.Scene, pos image.Point) { fill(m) })
	}
	item := func(m *core.Scene, text, shortcut string, fn func()) *core.Button {
		bt := core.NewButton(m).SetText(text)
		if shortcut != "" {
			bt.SetShortcut(key.Chord(shortcut))
		}
		bt.OnClick(func(e events.Event) { fn() })
		return bt
	}
	mac := system.TheApp.SystemPlatform() == system.MacOS
	about := func(m *core.Scene) {
		item(m, "About GGUIX", "", a.showAbout)
		item(m, "Quit", "Command+Q", func() { go system.TheApp.QuitReq() })
	}
	if mac {
		menu("GGUIX", about)
	}
	menu("File", func(m *core.Scene) {
		item(m, "Export CSV…", "Command+E", a.startExport).SetEnabled(!a.exp.running)
		item(m, "Import Notes…", "Command+O", func() { a.form.importNotes() }).SetEnabled(a.form.has)
		item(m, "Reveal Last Export", "", a.revealLastExport).SetEnabled(a.exp.lastPath != "")
	})
	menu("Edit", func(m *core.Scene) {
		item(m, "Cut", "", func() { a.editKey(key.CodeX) })
		item(m, "Copy", "", func() { a.editKey(key.CodeC) })
		item(m, "Paste", "", func() { a.editKey(key.CodeV) })
		item(m, "Select All", "", func() { a.editKey(key.CodeA) })
		core.NewSeparator(m)
		item(m, "Copy Row", "Command+Shift+C", a.form.copyRow)
	})
	menu("View", func(m *core.Scene) {
		radio := func(on bool) icons.Icon {
			if on {
				return icons.RadioButtonChecked
			}
			return icons.RadioButtonUnchecked
		}
		item(m, "Table", "Command+1", func() { a.tools.views.SelectValue("Table"); a.setView(viewTable) }).SetIcon(radio(a.view == viewTable))
		item(m, "Cards", "Command+2", func() { a.tools.views.SelectValue("Cards"); a.setView(viewCards) }).SetIcon(radio(a.view == viewCards))
		core.NewButton(m).SetText("Theme").SetMenu(func(tm *core.Scene, pos image.Point) {
			for i, n := range themeNames {
				v := []string{gc.ThemeSystem, gc.ThemeLight, gc.ThemeDark}[i]
				item(tm, n, "", func() { a.setTheme(v) }).SetIcon(radio(a.prefs.Theme == v))
			}
		})
		item(m, "Reload", "Command+R", a.reload)
		item(m, "Focus Search", "Command+F", func() { a.tools.search.SetFocus() })
	})
	if mac {
		menu("Help", func(m *core.Scene) { item(m, "Host Info…", "Command+I", a.showHost) })
	} else {
		menu("Help", func(m *core.Scene) {
			about(m)
			item(m, "Host Info…", "Command+I", a.showHost)
		})
	}
}

// editKey sends a Cut/Copy/Paste/Select All key chord to the last text widget.
func (a *App) editKey(code key.Codes) {
	w := a.lastText
	if w == nil {
		return
	}
	mod := key.Control
	if system.TheApp.SystemPlatform() == system.MacOS {
		mod = key.Meta
	}
	var m key.Modifiers
	m.SetFlag(true, mod)
	r := map[key.Codes]rune{key.CodeX: 'x', key.CodeC: 'c', key.CodeV: 'v', key.CodeA: 'a'}[code]
	w.AsWidget().HandleEvent(events.NewKey(events.KeyChord, r, code, m))
}
