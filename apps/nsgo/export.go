package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"time"

	gx "gguix/core"
)

// beginExport is the Export CSV action: ask where to save (desktop) or
// build the file in memory for a download (web), then run it.
func (a *app) beginExport() {
	if a.exp.running {
		return
	}
	if a.plat.Stream != nil {
		a.plat.Stream(exportName, func(s *streamSink, ok bool, err error) {
			a.post(func() {
				switch {
				case errors.Is(err, errNoStream):
					a.runExportTo("", nil, nil)
				case err != nil:
					a.showError(err.Error())
				case ok:
					a.runExportTo("", s, nil)
				}
			})
		})
		return
	}
	if a.plat.Deliver != nil {
		a.runExport("", nil)
		return
	}
	if a.plat.SaveFile == nil {
		return
	}
	a.plat.SaveFile(exportName, func(path string, ok bool, err error) {
		if err != nil {
			a.showError(err.Error())
			return
		}
		if ok {
			a.runExport(path, nil)
		}
	})
}

// runExport exports the current query in the background. With a path it
// writes that file; with an empty path it collects the data in memory and
// hands it to plat.Deliver (or discards it when there is none, as the bench
// does on the web). done, when given, is called from the export goroutine.
func (a *app) runExport(path string, done func(rows int, err error)) {
	a.runExportTo(path, nil, done)
}

// runExportTo is runExport that can stream into a sink instead.
func (a *app) runExportTo(path string, sink *streamSink, done func(rows int, err error)) {
	if a.exp.running {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.exp.running, a.exp.done, a.exp.total, a.exp.cancel = true, 0, 1, cancel
	q := a.query
	a.rExp.Bump()
	go func() {
		var (
			w    io.Writer = io.Discard
			buf  *bytes.Buffer
			file *os.File
			err  error
		)
		switch {
		case sink != nil:
			w = sink.W
		case path != "":
			file, err = os.Create(path)
			w = file
		case a.plat.Deliver != nil && !a.bench:
			buf = &bytes.Buffer{}
			w = buf
		}
		rows := 0
		if err == nil {
			rows, err = a.svc.Export(ctx, q, ctxWriter{ctx, w}, func(d, t int) {
				a.post(func() {
					a.exp.done, a.exp.total = d, max(t, 1)
					a.rExp.Bump()
				})
			})
		}
		if sink != nil {
			sink.Finish(err)
		}
		if file != nil {
			if cerr := file.Close(); err == nil {
				err = cerr
			}
			if err != nil {
				os.Remove(path)
			}
		}
		if err == nil && buf != nil {
			err = a.plat.Deliver(exportName, buf.Bytes())
		}
		cancel()
		a.post(func() { a.finishExport(path, rows, err) })
		if done != nil {
			done(rows, err)
		}
	}()
}

func (a *app) finishExport(path string, rows int, err error) {
	a.exp.running = false
	a.exp.cancel = nil
	switch {
	case errors.Is(err, context.Canceled):
		a.setMessage("Export cancelled", 2*time.Second)
	case err != nil:
		a.showError(err.Error())
	default:
		if path != "" {
			a.exp.lastPath = path
		}
		title, body := "Export complete", gx.FormatCount(rows)+" rows written"
		if a.plat.Notify == nil {
			a.setMessage(title+": "+body, 4*time.Second)
		} else {
			a.plat.Notify(title, body, func(shown bool) {
				if !shown {
					a.setMessage(title+": "+body, 4*time.Second)
				}
			})
		}
	}
	a.rExp.Bump()
}

func (a *app) cancelExport() {
	if a.exp.cancel != nil {
		a.exp.cancel()
	}
}

// ctxWriter fails a write once ctx is done. The remote client in a browser
// reads the response body with fetch, and Go's js/wasm transport only honors
// a context while waiting for the response headers, not while reading the
// body, so without this Cancel would not stop a streaming export.
type ctxWriter struct {
	ctx context.Context
	w   io.Writer
}

func (c ctxWriter) Write(b []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.w.Write(b)
}
