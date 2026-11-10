// Package server implements the GGUIX HTTP API over a core.DataService,
// with per-app prefs, host info and optional static file serving.
//
//	GET  /api/items?offset&limit&search&cat&status&sort&desc  -> core.Page
//	GET  /api/items/{id}                                     -> core.Item | 404
//	PUT  /api/items/{id}                                     -> core.Item | 422 core.ValidationError
//	GET  /api/export?search&cat&status&sort&desc             -> text/csv stream, X-Total-Count
//	GET  /api/prefs/{app}                                    -> core.Prefs
//	PUT  /api/prefs/{app}                                    -> 204
//	GET  /api/host?app=                                      -> host.Info (source "server-process")
//	GET  /*                                                  -> static files (index.html fallback)
package server

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gguix/core"
	"gguix/core/host"
	"gguix/core/prefs"
)

func init() { _ = mime.AddExtensionType(".wasm", "application/wasm") }

// Handler returns the API handler. staticDir may be "" for API only.
func Handler(svc core.DataService, staticDir string) http.Handler {
	s := &srv{svc: svc, stores: map[string]core.PrefsStore{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/items", s.items)
	mux.HandleFunc("GET /api/items/{id}", s.get)
	mux.HandleFunc("PUT /api/items/{id}", s.update)
	mux.HandleFunc("GET /api/export", s.export)
	mux.HandleFunc("GET /api/prefs/{app}", s.loadPrefs)
	mux.HandleFunc("PUT /api/prefs/{app}", s.savePrefs)
	mux.HandleFunc("GET /api/host", s.host)
	if staticDir != "" {
		mux.Handle("/", spa(staticDir))
	}
	return gzipMiddleware(mux)
}

type srv struct {
	svc    core.DataService
	mu     sync.Mutex
	stores map[string]core.PrefsStore
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	if errors.Is(err, core.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	if ve := core.AsValidationError(err); ve != nil {
		writeJSON(w, http.StatusUnprocessableEntity, ve)
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
}

func (s *srv) items(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	q, err := core.QueryFromValues(v)
	if err != nil {
		writeErr(w, err)
		return
	}
	off, _ := strconv.Atoi(v.Get("offset"))
	lim, err := strconv.Atoi(v.Get("limit"))
	if err != nil {
		lim = 200
	}
	p, err := s.svc.Page(r.Context(), core.PageRequest{Query: q, Offset: off, Limit: lim})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func idParam(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return 0, core.ErrNotFound
	}
	return id, nil
}

func (s *srv) get(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err == nil {
		var it core.Item
		if it, err = s.svc.Get(r.Context(), id); err == nil {
			writeJSON(w, http.StatusOK, it)
			return
		}
	}
	writeErr(w, err)
}

func (s *srv) update(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	var it core.Item
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&it); err != nil {
		writeErr(w, err)
		return
	}
	it.ID = id
	out, err := s.svc.Update(r.Context(), it)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *srv) export(w http.ResponseWriter, r *http.Request) {
	q, err := core.QueryFromValues(r.URL.Query())
	if err != nil {
		writeErr(w, err)
		return
	}
	first, err := s.svc.Page(r.Context(), core.PageRequest{Query: q, Limit: 1})
	if err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="gguix-export.csv"`)
	w.Header().Set("X-Total-Count", strconv.Itoa(first.Total))
	w.Header().Set("Access-Control-Expose-Headers", "X-Total-Count")
	w.WriteHeader(http.StatusOK)
	bw := bufio.NewWriterSize(w, 64<<10)
	flusher, _ := w.(http.Flusher)
	_, _ = s.svc.Export(r.Context(), q, bw, func(done, total int) {
		_ = bw.Flush()
		if flusher != nil {
			flusher.Flush()
		}
	})
	_ = bw.Flush()
}

func (s *srv) store(app string) core.PrefsStore {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.stores[app]
	if !ok {
		st = prefs.NewFileStore(app)
		s.stores[app] = st
	}
	return st
}

func (s *srv) loadPrefs(w http.ResponseWriter, r *http.Request) {
	p, err := s.store(r.PathValue("app")).Load()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *srv) savePrefs(w http.ResponseWriter, r *http.Request) {
	p := core.DefaultPrefs()
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&p); err != nil {
		writeErr(w, err)
		return
	}
	if err := s.store(r.PathValue("app")).Save(p.Sanitize()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *srv) host(w http.ResponseWriter, r *http.Request) {
	app := r.URL.Query().Get("app")
	if app == "" {
		app = "server"
	}
	writeJSON(w, http.StatusOK, host.Server(app))
}

// spa serves files from dir, falling back to index.html for unknown paths.
func spa(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Join(dir, filepath.FromSlash(path.Clean("/"+r.URL.Path)))
		if st, err := os.Stat(p); err != nil || (st.IsDir() && !exists(filepath.Join(p, "index.html"))) {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

type gzipWriter struct {
	http.ResponseWriter
	gz      *gzip.Writer
	started bool
	bypass  bool
}

func (g *gzipWriter) WriteHeader(code int) {
	if g.started {
		return
	}
	g.started = true
	h := g.Header()
	if code == http.StatusNoContent || code == http.StatusNotModified || h.Get("Content-Encoding") != "" {
		g.bypass = true
	} else {
		h.Del("Content-Length")
		h.Set("Content-Encoding", "gzip")
		h.Add("Vary", "Accept-Encoding")
	}
	g.ResponseWriter.WriteHeader(code)
}

func (g *gzipWriter) Write(b []byte) (int, error) {
	if !g.started {
		g.WriteHeader(http.StatusOK)
	}
	if g.bypass {
		return g.ResponseWriter.Write(b)
	}
	if g.gz == nil {
		g.gz = gzip.NewWriter(g.ResponseWriter)
	}
	return g.gz.Write(b)
}

func (g *gzipWriter) Flush() {
	if g.gz != nil {
		_ = g.gz.Flush()
	}
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") || r.URL.Path == "/api/export" {
			next.ServeHTTP(w, r)
			return
		}
		r.Header.Del("Range")
		gw := &gzipWriter{ResponseWriter: w}
		next.ServeHTTP(gw, r)
		if gw.gz != nil {
			_ = gw.gz.Close()
		}
	})
}
