package main

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// gzipMiddleware compresses responses for clients that accept gzip. It is a
// copy of core/server/server.go gzipMiddleware (stdlib only), added by the
// harness audit (B2) so every web server in the benchmark compresses. Paths
// for which skip returns true (streamed downloads) are passed through.
func gzipMiddleware(next http.Handler, skip func(*http.Request) bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") || r.Header.Get("Upgrade") != "" || (skip != nil && skip(r)) {
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
	if code == http.StatusNoContent || code == http.StatusNotModified || h.Get("Content-Encoding") != "" || strings.HasPrefix(h.Get("Content-Type"), "text/event-stream") {
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

// Unwrap lets http.ResponseController reach the underlying writer.
func (g *gzipWriter) Unwrap() http.ResponseWriter { return g.ResponseWriter }
