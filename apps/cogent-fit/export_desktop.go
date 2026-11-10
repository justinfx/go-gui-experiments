//go:build !js

package main

import (
	"os"
	"path/filepath"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
)

type fileDest struct {
	f    *os.File
	path string
}

func newFileDest(path string) (*fileDest, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &fileDest{f: f, path: path}, nil
}

func (d *fileDest) Write(p []byte) (int, error) { return d.f.Write(p) }
func (d *fileDest) Finish() error               { return d.f.Close() }
func (d *fileDest) Abort()                      { d.f.Close(); os.Remove(d.path) }
func (d *fileDest) Path() string                { return d.path }

// pickFile shows the framework file picker dialog (no native dialog exists).
func (a *App) pickFile(title, start, ext string, ok func(path string)) {
	d := core.NewBody(title)
	fp := core.NewFilePicker(d).SetFilename(start)
	if ext != "" {
		fp.SetExtensions(ext)
	}
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) { ok(fp.SelectedFile()) })
	})
	d.RunDialog(a.body)
}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return h
}

// startExport asks for a file and exports to it.
func (a *App) startExport() {
	if a.exp.running {
		return
	}
	a.pickFile("Export CSV", filepath.Join(homeDir(), "gguix-export.csv"), ".csv", func(path string) {
		if path == "" {
			return
		}
		d, err := newFileDest(path)
		if err != nil {
			core.ErrorDialog(a.body, err, "Export failed")
			return
		}
		a.runExport(d, nil)
	})
}

// startExportTo exports to a path without a dialog (bench, tests).
func (a *App) startExportTo(path string, finished func(int, error)) {
	d, err := newFileDest(path)
	if err != nil {
		finished(0, err)
		return
	}
	a.runExport(d, finished)
}

// importNotes opens a file picker filtered to .txt.
func (f *detailForm) importNotes() {
	f.app.pickFile("Import Notes", homeDir()+"/", ".txt", func(path string) {
		st, err := os.Stat(path)
		if err != nil {
			core.ErrorDialog(f.app.body, err, "Import Notes")
			return
		}
		if st.Size() > maxImportBytes {
			f.applyImported(make([]byte, maxImportBytes+1))
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			core.ErrorDialog(f.app.body, err, "Import Notes")
			return
		}
		f.applyImported(data)
	})
}
