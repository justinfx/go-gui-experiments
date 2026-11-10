package gotth

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"gguix/core"
	"gguix/core/host"
)

// handleExportSave is the desktop export: a native save dialog, then the
// export runs in the background and writes to the chosen file. In bench mode
// the file is GGUIX_BENCH_EXPORT and there is no dialog.
func (s *Server) handleExportSave(w http.ResponseWriter, r *http.Request) {
	var path string
	if s.opts.Bench && s.opts.BenchExport != "" {
		path = s.opts.BenchExport
	} else {
		p, ok, err := s.opts.Native.SaveDialog("gguix-export.csv")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !ok {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		path = p
	}
	// The request context ends with the response, so the job gets its own.
	id, ctx, ok := s.beginExport(context.Background())
	if !ok {
		http.Error(w, "an export is already running", http.StatusConflict)
		return
	}
	go s.runExportFile(id, ctx, path, s.Prefs().Query())
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleNativeOpenNotes(w http.ResponseWriter, r *http.Request) {
	type resp struct {
		OK    bool   `json:"ok"`
		Text  string `json:"text"`
		Error string `json:"error"`
	}
	out := func(v resp) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
	path, ok, err := s.opts.Native.OpenDialog("txt")
	if err != nil {
		out(resp{Error: err.Error()})
		return
	}
	if !ok {
		out(resp{})
		return
	}
	st, err := os.Stat(path)
	if err != nil {
		out(resp{Error: err.Error()})
		return
	}
	if st.Size() > core.MaxImportFileSize {
		out(resp{Error: "File is too large."})
		return
	}
	b, err := os.ReadFile(path)
	if err != nil {
		out(resp{Error: err.Error()})
		return
	}
	out(resp{OK: true, Text: firstRunes(string(b), core.MaxNotesRunes)})
}

func firstRunes(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		r = r[:n]
	}
	return string(r)
}

func (s *Server) handleNativeCopy(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	if err := s.opts.Native.CopyText(r.FormValue("text")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleNativeQuit(w http.ResponseWriter, r *http.Request) {
	_ = s.Flush()
	w.WriteHeader(http.StatusNoContent)
	if s.opts.Quit != nil {
		go s.opts.Quit()
	}
}

// reveal shows path in the file manager. A Native that has its own Reveal
// (the glaze shell uses crgimenes/native openurl) is preferred over the core
// helper.
func (s *Server) reveal(path string) error {
	if r, ok := s.opts.Native.(interface{ Reveal(string) error }); ok {
		return r.Reveal(path)
	}
	return host.RevealInFileManager(path)
}

func (s *Server) handleRevealConfig(w http.ResponseWriter, r *http.Request) {
	if err := s.reveal(s.host.ConfigDir); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRevealExport(w http.ResponseWriter, r *http.Request) {
	s.exMu.Lock()
	p := s.lastExport
	s.exMu.Unlock()
	if p == "" {
		http.Error(w, "no export yet", http.StatusConflict)
		return
	}
	if err := s.reveal(p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
