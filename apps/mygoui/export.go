package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"time"

	"gguix/core"
)

const exportName = "gguix-export.csv"

// beginExport is the Export CSV action: ask where to save (desktop) or
// build the file in memory for a download (web), then run it.
func (a *app) beginExport() {
	if a.exp.running {
		return
	}
	if a.plat.Deliver != nil {
		a.runExport("", nil)
		return
	}
	if a.plat.SaveFile == nil {
		return
	}
	go func() {
		path, ok, err := a.plat.SaveFile(exportName)
		a.post(func() {
			if err != nil {
				a.showError(err.Error())
				return
			}
			if ok {
				a.runExport(path, nil)
			}
		})
	}()
}

// runExport exports the current query in the background. With a path it
// writes that file; with an empty path it collects the data in memory and
// hands it to plat.Deliver (or discards it when there is none, as the bench
// does on the web). done, when given, is called from the export goroutine.
func (a *app) runExport(path string, done func(rows int, err error)) {
	if a.exp.running {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.exp.running, a.exp.done, a.exp.total, a.exp.cancel = true, 0, 1, cancel
	q := a.query
	a.wake()
	go func() {
		var (
			w    io.Writer = io.Discard
			buf  *bytes.Buffer
			file *os.File
			err  error
		)
		switch {
		case path != "":
			file, err = os.Create(path)
			w = file
		case a.plat.Deliver != nil && !a.bench:
			buf = &bytes.Buffer{}
			w = buf
		}
		rows := 0
		if err == nil {
			rows, err = a.svc.Export(ctx, q, w, func(d, t int) {
				a.post(func() { a.exp.done, a.exp.total = d, max(t, 1) })
			})
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
		title, body := "Export complete", core.FormatCount(rows)+" rows written"
		if a.plat.Notify == nil || !a.plat.Notify(title, body) {
			a.setMessage(title+": "+body, 4*time.Second)
		}
	}
	a.wake()
}

func (a *app) cancelExport() {
	if a.exp.cancel != nil {
		a.exp.cancel()
	}
}

func (a *app) exportPercent() int {
	if a.exp.total <= 0 {
		return 0
	}
	return min(100, a.exp.done*100/a.exp.total)
}
