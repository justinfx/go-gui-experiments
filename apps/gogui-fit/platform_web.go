//go:build js

package main

import (
	"syscall/js"

	"github.com/go-gui-org/go-gui/gui"
)

// webHooks: go-gui's web backend has no save dialog or OS notification, so
// the export is a browser download made with syscall/js, the notes import
// uses a file input element, and finished exports are announced in the
// status bar. Host actions are disabled in the browser.
func webHooks() platformHooks {
	return platformHooks{Deliver: download, OpenTextFile: pickTextFile, Quit: func() {}}
}

// platformFrame: no native menubar on the web; shortcuts come from the
// registered commands.
func (a *app) platformFrame(*gui.Window) {}

func download(name string, data []byte) error {
	g := js.Global()
	arr := g.Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(arr, data)
	blob := g.Get("Blob").New([]any{arr}, map[string]any{"type": "text/csv"})
	url := g.Get("URL").Call("createObjectURL", blob)
	doc := g.Get("document")
	el := doc.Call("createElement", "a")
	el.Set("href", url)
	el.Set("download", name)
	doc.Get("body").Call("appendChild", el)
	el.Call("click")
	doc.Get("body").Call("removeChild", el)
	g.Get("URL").Call("revokeObjectURL", url)
	return nil
}

func pickTextFile(done func([]byte, bool, error)) {
	in := js.Global().Get("document").Call("createElement", "input")
	in.Set("type", "file")
	in.Set("accept", ".txt,text/plain")
	var onChange, then js.Func
	onChange = js.FuncOf(func(js.Value, []js.Value) any {
		files := in.Get("files")
		if files.Length() == 0 {
			done(nil, false, nil)
			return nil
		}
		f := files.Index(0)
		if f.Get("size").Int() > 64*1024 {
			done(make([]byte, 64*1024+1), true, nil)
			return nil
		}
		then = js.FuncOf(func(_ js.Value, args []js.Value) any {
			done([]byte(args[0].String()), true, nil)
			return nil
		})
		f.Call("text").Call("then", then)
		return nil
	})
	in.Call("addEventListener", "change", onChange)
	in.Call("click")
}
