//go:build e2e

package main

// Test-only harness (not a MyGo target). MyGo has no browser mode, so the
// Playwright suite cannot drive the real webview. This binary serves the
// built frontend over HTTP and forwards the page's mygo.call() and mygo.on()
// (the shim in frontend/e2e/shim.js) to the real ItemService with
// encoding/json, using a fake desktop platform. It tests the app (frontend
// plus Go service plus core), not MyGo's IPC.

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"gguix/core/host"
)

type sub chan string

var (
	subMu sync.Mutex
	subs  = map[sub]bool{}
)

func emitEvent(name string, data any) {
	b, _ := json.Marshal(data)
	if data == nil {
		b = []byte("{}")
	}
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", name, b)
	subMu.Lock()
	for c := range subs {
		select {
		case c <- msg:
		default:
		}
	}
	subMu.Unlock()
}

type e2ePlat struct{ dir string }

func (p e2ePlat) Target() string { return "desktop" }
func (p e2ePlat) HostInfo() host.Info {
	return host.Current(appID)
}
func (p e2ePlat) ChooseExportPath(string) (string, bool, error) {
	return filepath.Join(p.dir, "export.csv"), true, nil
}
func (p e2ePlat) ChooseNotesFile() (string, bool, error) { return "", false, nil }
func (p e2ePlat) ShowError(string, string) bool          { return false }
func (p e2ePlat) CopyText(string) bool                   { return false }
func (p e2ePlat) Notify(string, string) error            { return nil }
func (p e2ePlat) Reveal(string) error                    { return nil }
func (p e2ePlat) Quit()                                  {}

func main() {
	ds, ps, err := openBackend(false)
	if err != nil {
		log.Fatal(err)
	}
	svc := newItemService(ds, ps, emitEvent)
	dir, _ := os.MkdirTemp("", "gguix-mygo-e2e")
	svc.plat = e2ePlat{dir}
	port := os.Getenv("GGUIX_E2E_PORT")
	if port == "" {
		port = "8192"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("/rpc/", func(w http.ResponseWriter, r *http.Request) {
		method := strings.TrimPrefix(r.URL.Path, "/rpc/ItemService.")
		m := reflect.ValueOf(svc).MethodByName(method)
		if !m.IsValid() {
			http.Error(w, "no method "+method, 404)
			return
		}
		var raw []json.RawMessage
		_ = json.NewDecoder(r.Body).Decode(&raw)
		mt := m.Type()
		in := make([]reflect.Value, mt.NumIn())
		for i := range in {
			v := reflect.New(mt.In(i))
			if i < len(raw) {
				if err := json.Unmarshal(raw[i], v.Interface()); err != nil {
					http.Error(w, err.Error(), 400)
					return
				}
			}
			in[i] = v.Elem()
		}
		out := m.Call(in)
		var result any
		for _, o := range out {
			if e, ok := o.Interface().(error); ok && e != nil {
				http.Error(w, e.Error(), 500)
				return
			}
			if o.Type().String() != "error" {
				result = o.Interface()
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	})
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		fl := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		c := make(sub, 64)
		subMu.Lock()
		subs[c] = true
		subMu.Unlock()
		defer func() { subMu.Lock(); delete(subs, c); subMu.Unlock() }()
		fmt.Fprint(w, ": ok\n\n")
		fl.Flush()
		for {
			select {
			case m := <-c:
				fmt.Fprint(w, m)
				fl.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})
	mux.Handle("/", http.FileServer(http.Dir("frontend/dist")))
	log.Fatal(http.ListenAndServe("127.0.0.1:"+port, mux))
}
