//go:build !js

package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"

	"gguix/core"
	"gguix/core/host"
	"gguix/core/prefs"
)

// Everything in this file serves the desktop target only.

// platformInit restores the window size, builds the menu bar and registers
// the close handler.
func (u *ui) platformInit(p core.Prefs) {
	w, h := max(p.WindowWidth, minWinW), max(p.WindowHeight, minWinH)
	u.win.Resize(fyne.NewSize(float32(w), float32(h)))
	u.win.SetOnClosed(func() { u.savePrefsNow() })
	u.buildMenu()
}

func (u *ui) windowSize() (int, int, bool) {
	s := u.win.Canvas().Size()
	if s.Width <= 0 || s.Height <= 0 {
		return 0, 0, false
	}
	return int(s.Width), int(s.Height), true
}

func (u *ui) quitApp() { u.app.Quit() }

func (u *ui) loadHost() { u.setHostInfo(host.Current(appID)) }

func hostActionsEnabled(info host.Info) bool { return info.Available() }

// startExport asks for a destination with a save dialog, then exports.
func (u *ui) startExport() {
	if u.exporting {
		return
	}
	d := dialog.NewFileSave(func(wc fyne.URIWriteCloser, err error) {
		if err != nil {
			u.showError(err)
			return
		}
		if wc == nil {
			return
		}
		u.exportTo(wc, wc.URI().Path())
	}, u.win)
	d.SetFileName("gguix-export.csv")
	d.SetFilter(storage.NewExtensionFileFilter([]string{".csv"}))
	d.Show()
}

func (u *ui) exportTo(wc io.WriteCloser, path string) {
	bw := bufio.NewWriterSize(wc, 64<<10)
	u.runExport(bw, func(rows int, err error) {
		ferr := bw.Flush()
		cerr := wc.Close()
		switch {
		case errors.Is(err, context.Canceled):
			_ = os.Remove(path)
			u.flash("Export cancelled")
		case err != nil:
			_ = os.Remove(path)
			u.showError(err)
		case ferr != nil || cerr != nil:
			_ = os.Remove(path)
			u.showError(errors.Join(ferr, cerr))
		default:
			u.lastExport = path
			u.updateMenus()
			u.notify("Export complete", fmt.Sprintf("%s rows written", core.FormatCount(rows)))
		}
	})
}

// notify sends an OS notification.
func (u *ui) notify(title, body string) {
	u.app.SendNotification(fyne.NewNotification(title, body))
	u.flash(title + ": " + body)
}

func benchExportWriter(path string) (io.Writer, func()) {
	if path == "" {
		return io.Discard, func() {}
	}
	f, err := os.Create(path)
	if err != nil {
		return io.Discard, func() {}
	}
	bw := bufio.NewWriterSize(f, 64<<10)
	return bw, func() { _ = bw.Flush(); _ = f.Close() }
}

// importNotes opens a file dialog for a .txt file.
func (u *ui) importNotes() {
	if !u.form.has {
		return
	}
	d := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
		if err != nil {
			u.showError(err)
			return
		}
		if rc == nil {
			return
		}
		defer rc.Close()
		data, err := io.ReadAll(io.LimitReader(rc, core.MaxImportFileSize+1))
		if err != nil {
			u.showError(err)
			return
		}
		u.importNotesData(data)
	}, u.win)
	d.SetFilter(storage.NewExtensionFileFilter([]string{".txt"}))
	d.Show()
}

func (u *ui) revealConfigFolder(info host.Info) {
	dir := info.ConfigDir
	if dir == "" {
		dir = prefs.Dir(appID)
	}
	_ = os.MkdirAll(dir, 0o755)
	if err := host.RevealInFileManager(dir); err != nil {
		u.showError(err)
	}
}

func (u *ui) revealLastExport() {
	if u.lastExport == "" {
		return
	}
	if err := host.RevealInFileManager(u.lastExport); err != nil {
		u.showError(err)
	}
}
