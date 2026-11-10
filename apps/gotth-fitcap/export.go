package gotth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gguix/apps/gotth/views"
	"gguix/core"
)

// ExportState returns the current export status.
func (s *Server) ExportState() views.ExportVM {
	s.exMu.Lock()
	defer s.exMu.Unlock()
	return s.ex
}

// beginExport marks a new export as running. ok is false when one is running.
func (s *Server) beginExport(parent context.Context) (id int, ctx context.Context, ok bool) {
	s.exMu.Lock()
	defer s.exMu.Unlock()
	if s.ex.State == "running" {
		return 0, nil, false
	}
	s.exSeq++
	ctx, s.exCancel = context.WithCancel(parent)
	s.ex = views.ExportVM{State: "running", ID: s.exSeq}
	s.broadcastLocked()
	return s.exSeq, ctx, true
}

func (s *Server) exportProgress(id int) func(done, total int) {
	return func(done, total int) {
		s.exMu.Lock()
		defer s.exMu.Unlock()
		if s.ex.ID == id && s.ex.State == "running" {
			s.ex.Done, s.ex.Total = done, total
			s.broadcastLocked()
		}
	}
}

// endExport records how an export finished.
func (s *Server) endExport(id int, rows int, runErr error, ctxErr error, path string) {
	s.exMu.Lock()
	defer s.exMu.Unlock()
	if s.ex.ID != id {
		return
	}
	if s.exCancel != nil {
		s.exCancel()
		s.exCancel = nil
	}
	switch {
	case ctxErr != nil && errors.Is(runErr, context.Canceled):
		s.ex.State = "cancelled"
	case runErr != nil:
		s.ex.State, s.ex.Err = "error", runErr.Error()
	default:
		s.ex.State, s.ex.Rows = "done", rows
		s.ex.Done = s.ex.Total
		if path != "" {
			s.lastExport = path
		}
	}
	s.broadcastLocked()
}

func (s *Server) broadcastLocked() {
	for ch := range s.exSubs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// flushWriter flushes after every write so the browser sees the download
// progress.
type flushWriter struct {
	w io.Writer
	f http.Flusher
}

func (fw flushWriter) Write(b []byte) (int, error) {
	n, err := fw.w.Write(b)
	if fw.f != nil {
		fw.f.Flush()
	}
	return n, err
}

// handleExportWeb streams the CSV for the current query as a download.
func (s *Server) handleExportWeb(w http.ResponseWriter, r *http.Request) {
	id, ctx, ok := s.beginExport(r.Context())
	if !ok {
		http.Error(w, "an export is already running", http.StatusConflict)
		return
	}
	q := s.Prefs().Query()
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="gguix-export.csv"`)
	w.Header().Set("Cache-Control", "no-store")
	fl, _ := w.(http.Flusher)
	n, err := s.opts.Svc.Export(ctx, q, flushWriter{w, fl}, s.exportProgress(id))
	s.endExport(id, n, err, ctx.Err(), "")
	if err != nil {
		// Abort the connection so the browser treats the download as
		// failed instead of saving a truncated file.
		panic(http.ErrAbortHandler)
	}
}

func (s *Server) handleExportCancel(w http.ResponseWriter, r *http.Request) {
	s.exMu.Lock()
	c := s.exCancel
	s.exMu.Unlock()
	if c != nil {
		c()
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleExportProgress pushes the export status as server-sent events.
func (s *Server) handleExportProgress(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch := make(chan struct{}, 1)
	s.exMu.Lock()
	s.exSubs[ch] = struct{}{}
	s.exMu.Unlock()
	defer func() {
		s.exMu.Lock()
		delete(s.exSubs, ch)
		s.exMu.Unlock()
	}()
	send := func() {
		html := renderString(r.Context(), views.ExportStatus(s.ExportState()))
		fmt.Fprint(w, "event: export\n")
		for _, line := range strings.Split(html, "\n") {
			fmt.Fprintf(w, "data: %s\n", line)
		}
		fmt.Fprint(w, "\n")
		fl.Flush()
	}
	send()
	keep := time.NewTicker(15 * time.Second)
	defer keep.Stop()
	for {
		select {
		case <-ch:
			send()
		case <-keep.C:
			fmt.Fprint(w, ": keepalive\n\n")
			fl.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// runExportFile exports the current query to a file (desktop). The partial
// file is removed when the export is cancelled or fails.
func (s *Server) runExportFile(id int, ctx context.Context, path string, q core.Query) {
	f, err := createFile(path)
	if err != nil {
		s.endExport(id, 0, err, nil, "")
		return
	}
	n, runErr := s.opts.Svc.Export(ctx, q, f, s.exportProgress(id))
	closeErr := f.Close()
	if runErr == nil {
		runErr = closeErr
	}
	if runErr != nil {
		removeFile(path)
	}
	s.endExport(id, n, runErr, ctx.Err(), path)
	if runErr == nil && s.opts.Native != nil && !s.opts.Bench {
		_ = s.opts.Native.Notify("Export complete", core.FormatCount(n)+" rows written")
	}
}
