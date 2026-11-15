//go:build js && remote

package main

import (
	"errors"
	"syscall/js"
)

// streamHook is the web-remote export: the core client already streams the
// response of the server's /api/export (fetch body reader), and each chunk is
// written to a file the user picked with the browser's File System Access API
// (showSaveFilePicker), so the whole CSV is never held in memory. Cancel
// aborts the writable, which discards the partial file. The picker is asked
// for inside the click's user activation. Browsers without it (Firefox,
// Safari) return errNoStream and the app falls back to an in-memory download.
func streamHook() func(string, func(*streamSink, bool, error)) {
	return func(name string, done func(*streamSink, bool, error)) {
		g := js.Global()
		if !g.Get("showSaveFilePicker").Truthy() {
			done(nil, false, errNoStream)
			return
		}
		opts := map[string]any{
			"suggestedName": name,
			"types":         []any{map[string]any{"description": "CSV", "accept": map[string]any{"text/csv": []any{".csv"}}}},
		}
		pick := g.Call("showSaveFilePicker", opts)
		go func() {
			handle, err := await(pick)
			if err != nil {
				if js.Value(err.val).Get("name").String() == "AbortError" {
					done(nil, false, nil) // the user cancelled the picker
					return
				}
				done(nil, false, err)
				return
			}
			w, err := await(handle.Call("createWritable"))
			if err != nil {
				done(nil, false, err)
				return
			}
			done(&streamSink{W: jsWriter{w}, Finish: func(err error) {
				if err != nil {
					_, _ = await(w.Call("abort"))
					return
				}
				_, _ = await(w.Call("close"))
			}}, true, nil)
		}()
	}
}

type jsErr struct{ val js.Value }

func (e *jsErr) Error() string {
	if m := e.val.Get("message"); m.Truthy() {
		return m.String()
	}
	return e.val.String()
}

// await blocks the calling goroutine (never a js callback) until the promise
// settles.
func await(p js.Value) (js.Value, *jsErr) {
	type result struct {
		v js.Value
		e *jsErr
	}
	ch := make(chan result, 1)
	ok := js.FuncOf(func(_ js.Value, a []js.Value) any { ch <- result{v: a[0]}; return nil })
	bad := js.FuncOf(func(_ js.Value, a []js.Value) any { ch <- result{e: &jsErr{a[0]}}; return nil })
	p.Call("then", ok, bad)
	r := <-ch
	ok.Release()
	bad.Release()
	return r.v, r.e
}

type jsWriter struct{ w js.Value }

func (j jsWriter) Write(b []byte) (int, error) {
	arr := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(arr, b)
	if _, e := await(j.w.Call("write", arr)); e != nil {
		return 0, errors.New(e.Error())
	}
	return len(b), nil
}
