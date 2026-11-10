package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"gguix/core"
)

// exportJob is one background export.
type exportJob struct {
	id     string
	q      core.Query
	ctx    context.Context
	cancel context.CancelFunc
	path   string // desktop: output file
	done   chan struct{}
}

// ExportStart is returned by StartExport.
type ExportStart struct {
	Started bool   `json:"started"`
	ID      string `json:"id"`
	Path    string `json:"path"` // desktop: the chosen file
	URL     string `json:"url"`  // web: the URL to download from
}

// StartExport starts a background export of the query.
//
// Desktop: asks for a file with the native save dialog (unless in bench
// mode), then writes it from a goroutine. Web: registers a job and returns
// the URL that streams the CSV; the export runs while the browser reads it.
// Progress and results are reported with the export:* events.
func (s *ItemService) StartExport(qa QueryArg) (ExportStart, error) {
	q, err := qa.ToQuery()
	if err != nil {
		return ExportStart{}, err
	}
	s.mu.Lock()
	if s.job != nil {
		s.mu.Unlock()
		return ExportStart{}, errors.New("An export is already running.")
	}
	s.mu.Unlock()

	path := ""
	if s.plat.Target() == "desktop" {
		if p := s.benchExportPath(); p != "" {
			path = p
		} else {
			p, ok, err := s.plat.ChooseExportPath("gguix-export.csv")
			if err != nil {
				return ExportStart{}, err
			}
			if !ok {
				return ExportStart{}, nil
			}
			path = p
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	if s.job != nil {
		s.mu.Unlock()
		cancel()
		return ExportStart{}, errors.New("An export is already running.")
	}
	s.seq++
	j := &exportJob{id: "e" + strconv.Itoa(s.seq), q: q, ctx: ctx, cancel: cancel, path: path, done: make(chan struct{})}
	s.job = j
	s.jobs[j.id] = j
	s.mu.Unlock()

	if path != "" {
		go s.exportToFile(j)
		return ExportStart{Started: true, ID: j.id, Path: path}, nil
	}
	return ExportStart{Started: true, ID: j.id, URL: "/export/" + j.id}, nil
}

// CancelExport cancels the running export, if any.
func (s *ItemService) CancelExport() {
	s.mu.Lock()
	j := s.job
	s.mu.Unlock()
	if j != nil {
		j.cancel()
	}
}

// finish clears the running job.
func (s *ItemService) finish(j *exportJob) {
	s.mu.Lock()
	if s.job == j {
		s.job = nil
	}
	delete(s.jobs, j.id)
	s.mu.Unlock()
	j.cancel()
	close(j.done)
}

// runExport streams the job's CSV to w and reports the outcome through events.
// It returns the rows written and the error (context.Canceled when cancelled).
func (s *ItemService) runExport(j *exportJob, w io.Writer) (int, error) {
	rows, err := s.svc.Export(j.ctx, j.q, w, func(done, total int) {
		s.emit("export:progress", ExportProgress{Done: done, Total: total})
	})
	return rows, err
}

func (s *ItemService) exportToFile(j *exportJob) {
	f, err := os.Create(j.path)
	if err != nil {
		s.finish(j)
		s.exportFailed(err)
		return
	}
	rows, err := s.runExport(j, f)
	cerr := f.Close()
	if err == nil {
		err = cerr
	}
	cancelled := j.ctx.Err() != nil && err != nil
	if err != nil {
		_ = os.Remove(j.path)
	} else {
		s.mu.Lock()
		s.lastExport = j.path
		s.mu.Unlock()
	}
	s.finish(j)
	switch {
	case cancelled || errors.Is(err, context.Canceled):
		s.emit("export:cancelled", nil)
	case err != nil:
		s.exportFailed(err)
	default:
		s.exportDone(rows)
	}
}

func (s *ItemService) exportFailed(err error) {
	s.emit("export:error", ExportError{Message: err.Error()})
	s.plat.ShowError("Export failed", err.Error())
}

func (s *ItemService) exportDone(rows int) {
	notified := false
	if !s.bench {
		body := fmt.Sprintf("%s rows written", core.FormatCount(rows))
		notified = s.plat.Notify("Export complete", body) == nil
	}
	s.emit("export:done", ExportDone{Rows: rows, Notified: notified || s.bench})
}

// serveExport runs a registered web job, writing the CSV to w. Used by the
// web target's HTTP handler.
func (s *ItemService) serveExport(id string, w io.Writer) (started bool) {
	s.mu.Lock()
	j := s.jobs[id]
	s.mu.Unlock()
	if j == nil {
		return false
	}
	rows, err := s.runExport(j, w)
	cancelled := j.ctx.Err() != nil && err != nil
	s.finish(j)
	switch {
	case cancelled || errors.Is(err, context.Canceled):
		s.emit("export:cancelled", nil)
	case err != nil:
		s.emit("export:error", ExportError{Message: err.Error()})
	default:
		s.exportDone(rows)
	}
	return true
}
