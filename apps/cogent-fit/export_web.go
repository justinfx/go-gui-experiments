//go:build js

package main

import (
	"bytes"
	"syscall/js"
)

// bufDest builds the CSV in memory and hands it to the browser as a download.
type bufDest struct {
	bytes.Buffer
	discard bool
}

func (b *bufDest) Write(p []byte) (int, error) {
	if b.discard {
		return len(p), nil
	}
	return b.Buffer.Write(p)
}
func (b *bufDest) Abort()       {}
func (b *bufDest) Path() string { return "" }
func (b *bufDest) Finish() error {
	if b.discard {
		return nil
	}
	arr := js.Global().Get("Uint8Array").New(b.Len())
	js.CopyBytesToJS(arr, b.Bytes())
	parts := js.Global().Get("Array").New(arr)
	blob := js.Global().Get("Blob").New(parts, map[string]any{"type": "text/csv"})
	url := js.Global().Get("URL").Call("createObjectURL", blob)
	doc := js.Global().Get("document")
	a := doc.Call("createElement", "a")
	a.Set("href", url)
	a.Set("download", "gguix-export.csv")
	doc.Get("body").Call("appendChild", a)
	a.Call("click")
	doc.Get("body").Call("removeChild", a)
	js.Global().Get("URL").Call("revokeObjectURL", url)
	return nil
}

func (a *App) startExport() {
	if a.exp.running {
		return
	}
	a.runExport(&bufDest{}, nil)
}

// startExportTo discards the CSV (bench).
func (a *App) startExportTo(path string, finished func(int, error)) {
	a.runExport(&bufDest{discard: true}, finished)
}

// importNotes uses a browser file input.
func (f *detailForm) importNotes() {
	doc := js.Global().Get("document")
	in := doc.Call("createElement", "input")
	in.Set("type", "file")
	in.Set("accept", ".txt")
	var cb js.Func
	cb = js.FuncOf(func(this js.Value, args []js.Value) any {
		files := in.Get("files")
		if files.Length() == 0 {
			return nil
		}
		file := files.Index(0)
		if file.Get("size").Int() > maxImportBytes {
			go f.app.async(func() { f.applyImported(make([]byte, maxImportBytes+1)) })
			return nil
		}
		file.Call("arrayBuffer").Call("then", js.FuncOf(func(_ js.Value, r []js.Value) any {
			u8 := js.Global().Get("Uint8Array").New(r[0])
			data := make([]byte, u8.Get("length").Int())
			js.CopyBytesToGo(data, u8)
			go f.app.async(func() { f.applyImported(data) })
			return nil
		}))
		return nil
	})
	in.Call("addEventListener", "change", cb)
	in.Call("click")
}
