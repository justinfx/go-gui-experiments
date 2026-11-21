//go:build js

package ui

import (
	"bytes"

	"github.com/doug/gophics/app"
	"github.com/doug/gophics/shell"
)

func platformConfig(*app.Config) {}

// memSink builds the CSV in memory, then hands it to FilePicker.Save, which
// on the web shell triggers a browser download.
type memSink struct {
	buf bytes.Buffer
	s   *App
}

func (m *memSink) Write(p []byte) (int, error) { return m.buf.Write(p) }
func (m *memSink) Abort()                      { m.buf.Reset() }
func (m *memSink) Commit() (string, error) {
	fp := m.s.ctx.FilePicker()
	if fp == nil {
		return "", nil
	}
	fp.Save(shell.SaveOptions{Name: "gguix-export.csv", Accept: []string{".csv", "text/csv"}}, m.buf.Bytes(), func(error) {})
	return "", nil
}

// chooseExportSink: no dialog on the web; the export becomes a download.
func (s *App) chooseExportSink(done func(exportSink, error)) { done(&memSink{s: s}, nil) }

// Quit is a no-op in the browser.
func (s *App) Quit() {}

func (s *App) RevealConfig()     {}
func (s *App) RevealLastExport() {}

// publishMenus: the web shell has no menu bar (ctx.Menus() is nil).
func (s *App) publishMenus() {}
