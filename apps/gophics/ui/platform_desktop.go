//go:build !js

package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/doug/gophics/app"
	"github.com/doug/gophics/shell"
	"github.com/doug/gophics/theme"
	"github.com/doug/gophics/widget"

	"gguix/core"
	"gguix/core/host"
)

func platformConfig(cfg *app.Config) {
	// The Go fonts have no ✓; let the OS fonts cover it (desktop only).
	cfg.SystemFonts = true
}

// chooseExportSink asks for the export file name, then its folder.
// Gophics' FilePicker.Save (shell/filepicker.go) takes the bytes up front and
// never reports the chosen path (shell/desktop/filepicker_darwin.go save), so
// a streamed, cancellable export cannot use the native save panel. The
// closest fallback: an in-app prompt for the file name (default
// gguix-export.csv), then the native folder chooser (NSOpenPanel for
// directories; on desktop a Folder's Token is its path).
func (s *App) chooseExportSink(done func(exportSink, error)) {
	fp := s.ctx.FolderPicker()
	if fp == nil {
		done(nil, errors.New("no folder chooser on this platform"))
		return
	}
	if s.ExportName == "" {
		s.ExportName = "gguix-export.csv"
	}
	cancel := func() { s.CloseDialog(); done(nil, nil) }
	choose := func() {
		name := exportFileName(s.ExportName)
		s.CloseDialog()
		if name == "" {
			done(nil, errors.New("invalid file name"))
			return
		}
		fp.Open(func(f shell.Folder, err error) {
			if err != nil || f == nil {
				done(nil, err)
				return
			}
			sink, err := newFileSink(filepath.Join(f.Token(), name))
			if err != nil {
				done(nil, err)
				return
			}
			done(sink, nil)
		})
	}
	s.showDialog("Export CSV", widget.Sized{W: 360, Child: widget.Column(
		theme.Heading("Export CSV"),
		gapH(8),
		theme.Body("File name"),
		gapH(4),
		Field{Value: s.ExportName, Autofocus: true, OnKeyPreview: s.fieldKeyPreview,
			OnChange: func(v string) { s.ExportName = v },
			OnSubmit: func(string) { s.SetState(choose) }},
		gapH(16),
		buttonsRow(
			theme.Button{Label: "Cancel", OnTap: s.act(cancel)}, gapW(8),
			theme.Button{Label: "Choose Folder…", Primary: true, OnTap: s.act(choose)},
		),
	)})
}

// exportFileName cleans the typed export name: base name only, .csv added
// when missing; "" when nothing usable is left.
func exportFileName(v string) string {
	name := filepath.Base(strings.TrimSpace(v))
	if name == "." || name == "/" || name == "" {
		return ""
	}
	if !strings.EqualFold(filepath.Ext(name), ".csv") {
		name += ".csv"
	}
	return name
}

// Quit saves prefs and exits. Gophics exposes no programmatic window close
// to widget code (shell.Window.Close is not reachable from a Ctx).
func (s *App) Quit() {
	s.SavePrefsSync()
	os.Exit(0)
}

// RevealConfig reveals the prefs directory in Finder (core helper; Gophics has
// no reveal API).
func (s *App) RevealConfig() {
	if err := host.RevealInFileManager(s.Host.ConfigDir); err != nil {
		s.ShowMessage("Reveal failed", err.Error())
	}
}

// RevealLastExport reveals the last exported file.
func (s *App) RevealLastExport() {
	if s.LastExport == "" {
		return
	}
	if err := host.RevealInFileManager(s.LastExport); err != nil {
		s.ShowMessage("Reveal failed", err.Error())
	}
}

const (
	miAbout = iota + 1
	miExport
	miImport
	miReveal
	miCopyRow
	miTable
	miCards
	miThemeSystem
	miThemeLight
	miThemeDark
	miReload
	miFocusSearch
	miHost
)

func check(on bool, title string) string {
	if on {
		return "✓ " + title
	}
	return "    " + title
}

// publishMenus publishes the native menu bar (NSMenu on macOS) when its
// content changes. shell.MenuItem has no shortcut or checked-state fields, so
// shortcuts are handled by the root key handler and radio state is shown
// with a check mark in the title.
func (s *App) publishMenus() {
	m := s.ctx.Menus()
	if m == nil {
		return
	}
	key := fmt.Sprint(s.View, s.Theme, s.Exporting, s.LastExport != "", s.HasItem)
	if key == s.menusPublished {
		return
	}
	s.menusPublished = key
	bar := []shell.Menu{
		{Title: "GGUIX", Items: []shell.MenuItem{
			{ID: miAbout, Title: "About GGUIX"},
			{Separator: true},
			{Role: shell.RoleQuit, Title: "Quit"},
		}},
		{Title: "File", Items: []shell.MenuItem{
			{ID: miExport, Title: "Export CSV…  (⌘E)", Disabled: s.Exporting},
			{ID: miImport, Title: "Import Notes…  (⌘O)", Disabled: !s.HasItem},
			{ID: miReveal, Title: "Reveal Last Export", Disabled: s.LastExport == ""},
		}},
		{Title: "Edit", Items: []shell.MenuItem{
			{ID: miCopyRow, Title: "Copy Row  (⇧⌘C)", Disabled: !s.HasItem},
		}},
		{Title: "View", Items: []shell.MenuItem{
			{ID: miTable, Title: check(s.View == core.ViewTable, "Table  (⌘1)")},
			{ID: miCards, Title: check(s.View == core.ViewCards, "Cards  (⌘2)")},
			{Separator: true},
			{Title: "Theme", Submenu: []shell.MenuItem{
				{ID: miThemeSystem, Title: check(s.Theme == core.ThemeSystem, "System")},
				{ID: miThemeLight, Title: check(s.Theme == core.ThemeLight, "Light")},
				{ID: miThemeDark, Title: check(s.Theme == core.ThemeDark, "Dark")},
			}},
			{Separator: true},
			{ID: miReload, Title: "Reload  (⌘R)"},
			{ID: miFocusSearch, Title: "Focus Search  (⌘F)"},
		}},
		{Title: "Help", Items: []shell.MenuItem{
			{ID: miHost, Title: "Host Info…  (⌘I)"},
		}},
	}
	m.SetBar(bar, func(id int) {
		// Menu actions arrive on the AppKit main thread; hop to the UI thread.
		s.Post(func() { s.menuAction(id) })
	})
}

func (s *App) menuAction(id int) {
	switch id {
	case miAbout:
		s.ShowAbout()
	case miExport:
		s.StartExport()
	case miImport:
		s.ImportNotes()
	case miReveal:
		s.RevealLastExport()
	case miCopyRow:
		s.CopyRow()
	case miTable:
		s.SetView(core.ViewTable)
	case miCards:
		s.SetView(core.ViewCards)
	case miThemeSystem:
		s.SetTheme(core.ThemeSystem)
	case miThemeLight:
		s.SetTheme(core.ThemeLight)
	case miThemeDark:
		s.SetTheme(core.ThemeDark)
	case miReload:
		s.Reload()
	case miFocusSearch:
		s.FocusSearch()
	case miHost:
		s.ShowHost()
	}
}
