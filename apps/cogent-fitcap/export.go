package main

// Export job logic shared by desktop and web (brief section 5.5).

import (
	"context"
	"io"
	"sync/atomic"

	"cogentcore.org/core/core"

	gc "gguix/core"
)

// exportDest is where an export writes.
type exportDest interface {
	io.Writer
	// Finish completes the export (close the file, start the download).
	Finish() error
	// Abort discards the partial output.
	Abort()
	// Path is the file written, "" if none.
	Path() string
}

type exportState struct {
	running   bool
	done      atomic.Int64
	total     atomic.Int64
	cancel    context.CancelFunc
	lastPath  string
	progQueue atomic.Bool
}

func (e *exportState) percent() int {
	t := e.total.Load()
	if t <= 0 {
		return 0
	}
	return int(e.done.Load() * 100 / t)
}

func (a *App) exportStateChanged() {
	a.tools.exportB.SetEnabled(!a.exp.running)
	a.tools.exportB.NeedsRender()
	a.status.update()
	a.status.exportBox.Update()
	a.status.root.NeedsLayout()
	a.refreshMenus()
}

// runExport exports the current query to dest in the background.
func (a *App) runExport(dest exportDest, finished func(rows int, err error)) {
	if a.exp.running {
		dest.Abort()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.exp.running = true
	a.exp.cancel = cancel
	a.exp.done.Store(0)
	a.exp.total.Store(0)
	a.exportStateChanged()
	q := a.query
	go func() {
		n, err := a.svc.Export(ctx, q, dest, func(d, t int) {
			a.exp.done.Store(int64(d))
			a.exp.total.Store(int64(t))
			if !a.exp.progQueue.Swap(true) {
				go func() {
					a.exp.progQueue.Store(false)
					a.async(a.status.update)
				}()
			}
		})
		cancelled := ctx.Err() != nil
		if cancelled || err != nil {
			dest.Abort()
		} else if ferr := dest.Finish(); ferr != nil {
			err = ferr
		}
		a.async(func() {
			a.exp.running = false
			a.exp.cancel = nil
			a.exportStateChanged()
			switch {
			case cancelled:
				a.flash("Export cancelled")
				err = context.Canceled
			case err != nil:
				if finished == nil {
					core.ErrorDialog(a.body, err, "Export failed")
				}
			default:
				if p := dest.Path(); p != "" {
					a.exp.lastPath = p
				}
				// No OS notification API in Cogent Core: in-app message.
				core.MessageSnackbar(a.body, "Export complete: "+gc.FormatCount(n)+" rows written")
			}
			if finished != nil {
				finished(n, err)
			}
		})
	}()
}

func (a *App) cancelExport() {
	if a.exp.running && a.exp.cancel != nil {
		a.exp.cancel()
	}
}
