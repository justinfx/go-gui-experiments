//go:build server

package main

// Web entry points (category: web): Wails server mode has no window, menu or
// dialogs. The only server-side extras are the streaming export download and
// generous HTTP timeouts so a full export is not cut off.

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func targetServices() []application.Service { return nil }

func configureTarget(o *application.Options, svc *ItemService) {
	o.Server = application.ServerOptions{
		Port:         8090,
		WriteTimeout: 30 * time.Minute,
		ReadTimeout:  time.Minute,
	}
}

func setupTarget(app *application.App, svc *ItemService) {
	svc.plat = &webPlatform{}
}

// assetMiddleware serves GET /export/<id>: the CSV stream for a job created
// by StartExport. Every other asset response is gzip-compressed (B2).
func assetMiddleware(svc *ItemService) application.Middleware {
	noGzip := func(r *http.Request) bool { return strings.HasPrefix(r.URL.Path, "/export/") }
	return func(next http.Handler) http.Handler {
		return gzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := strings.CutPrefix(r.URL.Path, "/export/")
			if !ok || r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}
			svc.mu.Lock()
			j := svc.jobs[id]
			svc.mu.Unlock()
			if j == nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="gguix-export.csv"`)
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusOK)
			stop := context.AfterFunc(r.Context(), j.cancel)
			defer stop()
			svc.serveExport(id, flushWriter{w, http.NewResponseController(w)})
		}), noGzip)
	}
}

type flushWriter struct {
	w  http.ResponseWriter
	rc *http.ResponseController
}

func (f flushWriter) Write(p []byte) (int, error) {
	n, err := f.w.Write(p)
	_ = f.rc.Flush()
	return n, err
}
