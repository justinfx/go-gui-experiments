package gotth

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"gguix/core"
)

var (
	benchDone  atomic.Bool
	benchWatch atomic.Pointer[time.Timer]
)

// kickBenchWatchdog (re)starts the stall timer: if no metric arrives for 45 s
// the shell reports the stall, prints done and quits instead of hanging.
func (s *Server) kickBenchWatchdog() {
	if s.opts.Quit == nil || benchDone.Load() {
		return
	}
	t := time.AfterFunc(45*time.Second, func() {
		if benchDone.Load() {
			return
		}
		core.EmitMetric("error", map[string]any{"message": "bench stalled: no metric for 45 s"})
		s.finishBench(true)
	})
	if old := benchWatch.Swap(t); old != nil {
		old.Stop()
	}
}

// finishBench prints the done metric (if the page did not) and quits the shell.
func (s *Server) finishBench(emit bool) {
	if benchDone.Swap(true) {
		return
	}
	if emit {
		core.EmitMetric("done", nil)
	}
	_ = s.Flush()
	if s.opts.Quit != nil {
		go s.opts.Quit()
	}
}

// handleBenchMetric receives one GGUIX_METRIC line from the bench script and
// writes it to stdout through core.EmitMetric.
func (s *Server) handleBenchMetric(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	line := strings.TrimPrefix(strings.TrimSpace(string(b)), core.MetricPrefix)
	var fields map[string]any
	if err := json.Unmarshal([]byte(line), &fields); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ev, _ := fields["event"].(string)
	delete(fields, "event")
	s.kickBenchWatchdog()
	if ev == "done" {
		w.WriteHeader(http.StatusNoContent)
		if !benchDone.Load() {
			core.EmitMetric(ev, fields)
		}
		s.finishBench(false)
		return
	}
	core.EmitMetric(ev, fields)
	w.WriteHeader(http.StatusNoContent)
	// Safety net: the last step is the export (or an error). If the page does
	// not send "done" shortly after, print it and quit anyway.
	if (ev == "export" || ev == "error") && s.opts.Quit != nil {
		time.AfterFunc(5*time.Second, func() { s.finishBench(true) })
	}
}

func (s *Server) handleBenchQuit(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
	s.finishBench(false)
}
