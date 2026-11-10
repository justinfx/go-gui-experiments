//go:build js && wasm

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"syscall/js"

	"fyne.io/fyne/v2"

	"gguix/core"
	"gguix/core/host"
)

// Everything in this file serves the web (WASM) targets only.

func (u *ui) platformInit(p core.Prefs) {}

func (u *ui) windowSize() (int, int, bool) { return 0, 0, false }

func (u *ui) quitApp() {}

// loadHost reads the host info in the background: with the remote build tag
// it is a network call.
func (u *ui) loadHost() {
	u.setHostInfo(host.Info{Source: host.SourceUnavailable})
	go func() {
		info := host.Current(appID)
		u.do(func() { u.setHostInfo(info) })
	}()
}

// Host actions would act on the server (or nothing), never on the browser
// user's machine.
func hostActionsEnabled(info host.Info) bool { return false }

func (u *ui) revealConfigFolder(host.Info) {}
func (u *ui) revealLastExport()           {}

func benchExportWriter(string) (io.Writer, func()) { return io.Discard, func() {} }

// startExport builds the CSV in memory and hands it to the browser as a
// download named gguix-export.csv.
func (u *ui) startExport() {
	if u.exporting {
		return
	}
	var buf bytes.Buffer
	u.runExport(&buf, func(rows int, err error) {
		switch {
		case errors.Is(err, context.Canceled):
			u.flash("Export cancelled")
		case err != nil:
			u.showError(err)
		default:
			downloadBytes("gguix-export.csv", buf.Bytes())
			u.notify("Export complete", fmt.Sprintf("%s rows written", core.FormatCount(rows)))
		}
	})
}

// notify has no OS notification on the web target here, so it shows the
// message in the app.
func (u *ui) notify(title, body string) {
	u.flash(title + ": " + body)
	fyne.CurrentApp().SendNotification(fyne.NewNotification(title, body))
}

func downloadBytes(name string, data []byte) {
	arr := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(arr, data)
	blob := js.Global().Get("Blob").New([]any{arr}, map[string]any{"type": "text/csv"})
	url := js.Global().Get("URL").Call("createObjectURL", blob)
	doc := js.Global().Get("document")
	a := doc.Call("createElement", "a")
	a.Set("href", url)
	a.Set("download", name)
	doc.Get("body").Call("appendChild", a)
	a.Call("click")
	a.Call("remove")
	js.Global().Call("setTimeout", js.FuncOf(func(js.Value, []js.Value) any {
		js.Global().Get("URL").Call("revokeObjectURL", url)
		return nil
	}), 10000)
}

// importNotes uses a browser file input, because Fyne's file dialog does
// nothing under WASM in this version.
func (u *ui) importNotes() {
	if !u.form.has {
		return
	}
	doc := js.Global().Get("document")
	in := doc.Call("createElement", "input")
	in.Set("type", "file")
	in.Set("accept", ".txt,text/plain")
	var onChange, onBuf js.Func
	onBuf = js.FuncOf(func(_ js.Value, args []js.Value) any {
		u8 := js.Global().Get("Uint8Array").New(args[0])
		data := make([]byte, u8.Get("length").Int())
		js.CopyBytesToGo(data, u8)
		u.do(func() { u.importNotesData(data) })
		return nil
	})
	onChange = js.FuncOf(func(_ js.Value, _ []js.Value) any {
		files := in.Get("files")
		if files.Length() == 0 {
			return nil
		}
		f := files.Index(0)
		if f.Get("size").Int() > core.MaxImportFileSize {
			u.do(func() { u.showError(errTooLarge) })
			return nil
		}
		f.Call("arrayBuffer").Call("then", onBuf)
		return nil
	})
	in.Call("addEventListener", "change", onChange)
	in.Call("click")
}
