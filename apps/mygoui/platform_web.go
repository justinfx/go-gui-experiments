//go:build js

package main

import (
	"syscall/js"

	"github.com/egoist/mygo/ui"
)

// webHooks: no file dialogs, menus or notifications exist in the browser
// for the framework, so the export is a download made with syscall/js, the
// notes import uses a file input element, and the finished export is
// announced in the status bar.
func webHooks() platformHooks {
	return platformHooks{
		HostActions:  false,
		Deliver:      download,
		OpenTextFile: pickTextFile,
		Quit:         func() {},
	}
}

func download(name string, data []byte) error {
	g := js.Global()
	arr := g.Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(arr, data)
	blob := g.Get("Blob").New([]any{arr}, map[string]any{"type": "text/csv"})
	url := g.Get("URL").Call("createObjectURL", blob)
	doc := g.Get("document")
	a := doc.Call("createElement", "a")
	a.Set("href", url)
	a.Set("download", name)
	doc.Get("body").Call("appendChild", a)
	a.Call("click")
	doc.Get("body").Call("removeChild", a)
	g.Get("URL").Call("revokeObjectURL", url)
	return nil
}

func pickTextFile() ([]byte, bool, error) {
	g := js.Global()
	in := g.Get("document").Call("createElement", "input")
	in.Set("type", "file")
	in.Set("accept", ".txt,text/plain")
	type result struct {
		data []byte
		ok   bool
		err  error
	}
	ch := make(chan result, 1)
	var fns []js.Func
	release := func() {
		for _, f := range fns {
			f.Release()
		}
	}
	onChange := js.FuncOf(func(this js.Value, args []js.Value) any {
		files := in.Get("files")
		if files.Length() == 0 {
			ch <- result{}
			return nil
		}
		file := files.Index(0)
		if file.Get("size").Int() > 64*1024 {
			ch <- result{data: make([]byte, 64*1024+1), ok: true}
			return nil
		}
		then := js.FuncOf(func(this js.Value, args []js.Value) any {
			s := args[0].String()
			ch <- result{data: []byte(s), ok: true}
			return nil
		})
		fns = append(fns, then)
		file.Call("text").Call("then", then)
		return nil
	})
	onCancel := js.FuncOf(func(this js.Value, args []js.Value) any { ch <- result{}; return nil })
	fns = append(fns, onChange, onCancel)
	in.Call("addEventListener", "change", onChange)
	in.Call("addEventListener", "cancel", onCancel)
	in.Call("click")
	r := <-ch
	release()
	if r.err == nil && !r.ok {
		return nil, false, nil
	}
	return r.data, r.ok, r.err
}

// platformShortcuts: the browser has no menu bar, so the shortcuts the
// desktop menu owns are handled by the view.
func (a *app) platformShortcuts(c *ui.Context) {
	switch {
	case c.Shortcut(ui.Cmd, ui.KeyE):
		a.beginExport()
	case c.Shortcut(ui.Cmd, ui.KeyO):
		a.beginImport()
	case c.Shortcut(ui.Cmd, ui.Key1):
		a.setView(0)
	case c.Shortcut(ui.Cmd, ui.Key2):
		a.setView(1)
	case c.Shortcut(ui.Cmd, ui.KeyR):
		a.rebuildQuery(true)
	case c.Shortcut(ui.Cmd, ui.KeyF):
		a.focusSrch = true
	case c.Shortcut(ui.Cmd, ui.KeyI):
		a.hostOpen = true
	case c.Shortcut(ui.Cmd|ui.Shift, ui.KeyC):
		a.copyRow()
	}
}
