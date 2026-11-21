package ui

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/doug/gophics/shell"
	"github.com/doug/gophics/theme"

	"gguix/core"
)

// exportSink is where an export's CSV goes: a file (desktop) or a memory
// buffer that becomes a browser download (web).
type exportSink interface {
	io.Writer
	// Commit finishes a successful export and returns the path written ("" on web).
	Commit() (string, error)
	// Abort discards a cancelled or failed export (deletes the partial file).
	Abort()
}

// fileSink writes straight to a path.
type fileSink struct {
	f    *os.File
	path string
}

func newFileSink(path string) (*fileSink, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &fileSink{f: f, path: path}, nil
}

func (f *fileSink) Write(p []byte) (int, error) { return f.f.Write(p) }
func (f *fileSink) Commit() (string, error)     { return f.path, f.f.Close() }
func (f *fileSink) Abort()                      { f.f.Close(); os.Remove(f.path) }

type discardSink struct{}

func (discardSink) Write(p []byte) (int, error) { return len(p), nil }
func (discardSink) Commit() (string, error)     { return "", nil }
func (discardSink) Abort()                      {}

// StartExport is the Export CSV button / menu item / Cmd+E.
func (s *App) StartExport() {
	if s.Exporting {
		return
	}
	s.chooseExportSink(func(sink exportSink, err error) {
		s.Post(func() {
			if err != nil {
				s.ShowMessage("Export failed", err.Error())
				return
			}
			if sink == nil {
				return // cancelled dialog
			}
			s.runExport(sink, nil)
		})
	})
}

// StartExportTo exports to path without a dialog ("" discards). done is
// called on the UI thread.
func (s *App) StartExportTo(path string, done func(rows int, err error)) {
	if s.Exporting {
		if done != nil {
			done(0, errors.New("export already running"))
		}
		return
	}
	var sink exportSink = discardSink{}
	if path != "" {
		fs, err := newFileSink(path)
		if err != nil {
			if done != nil {
				done(0, err)
			}
			return
		}
		sink = fs
	}
	s.runExport(sink, done)
}

// CancelExport cancels the running export.
func (s *App) CancelExport() {
	if s.exportCancel != nil {
		s.exportCancel()
	}
}

func (s *App) runExport(sink exportSink, done func(int, error)) {
	ctx, cancel := context.WithCancel(context.Background())
	s.Exporting, s.ExportDone, s.ExportTotal, s.exportCancel = true, 0, max(s.Pager.Total(), 0), cancel
	q := s.Query
	go func() {
		n, err := s.svc.Export(ctx, q, sink, func(d, t int) {
			s.Post(func() { s.ExportDone, s.ExportTotal = d, t })
		})
		s.Post(func() {
			s.Exporting, s.exportCancel = false, nil
			cancel()
			switch {
			case errors.Is(err, context.Canceled):
				sink.Abort()
				s.Flash("Export cancelled")
			case err != nil:
				sink.Abort()
				s.ShowMessage("Export failed", err.Error())
			default:
				path, cerr := sink.Commit()
				if cerr != nil {
					err = cerr
					s.ShowMessage("Export failed", cerr.Error())
					break
				}
				if path != "" {
					s.LastExport = path
				}
				s.notify("Export complete", core.FormatCount(n)+" rows written")
			}
			if done != nil {
				done(n, err)
			}
		})
	}()
}

// notify posts an OS notification, or an in-app snackbar where the
// platform has none.
func (s *App) notify(title, body string) {
	s.LastNotice = title + ": " + body
	if s.opts.Bench {
		return
	}
	if n := s.ctx.Notifier(); n != nil {
		n.Authorize(func(p shell.Permission) {
			if p == shell.PermissionGranted {
				n.Notify(shell.Notification{Title: title, Body: body})
			} else {
				s.post(func() { theme.ShowSnackbar(s.ctx, title+" - "+body) })
			}
		})
		return
	}
	theme.ShowSnackbar(s.ctx, title+" - "+body)
}
