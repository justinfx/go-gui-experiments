//go:build js

package main

import (
	"syscall/js"

	"github.com/gen2brain/iup-go/iup"

	"gguix/core/host"
)

// webHooks: the browser has no save dialog or OS reveal. The export is a
// browser download made with syscall/js, the notes import uses a file input
// element, notifications fall back to a status message, and the confirm
// dialog is an IUP dialog shown with Show (a browser cannot block).
func webHooks(a *app) platformHooks {
	return platformHooks{
		Deliver:      download,
		OpenTextFile: pickTextFile,
		Confirm:      a.webConfirm,
		Alert:        func(msg string) { iup.Message("GGUIX", msg) },
		Clipboard: func(text string) {
			js.Global().Call("postMessage", map[string]any{"__gguixCopy": true, "text": text})
		},
		Quit: func() {},
	}
}

func (a *app) quit() {}

// webConfirm shows a modeless IUP dialog with two buttons.
func (a *app) webConfirm(title, msg, okLabel, cancelLabel string, ok, cancel func()) {
	okBtn := iup.Button(okLabel)
	noBtn := iup.Button(cancelLabel)
	box := iup.Vbox(iup.Label(msg), iup.Hbox(iup.Fill(), okBtn, noBtn))
	set(box, "MARGIN", "14x12", "GAP", "12")
	dlg := iup.Dialog(box)
	set(dlg, "TITLE", title, "DIALOGFRAME", "YES")
	iup.SetAttributeHandle(dlg, "PARENTDIALOG", a.ui.dlg)
	finish := func(f func()) int {
		iup.Hide(dlg)
		iup.Destroy(dlg)
		f()
		return iup.DEFAULT
	}
	iup.On(okBtn, iup.ActionCB, func(iup.Ihandle) int { return finish(ok) })
	iup.On(noBtn, iup.ActionCB, func(iup.Ihandle) int { return finish(cancel) })
	iup.ShowXY(dlg, iup.CENTERPARENT, iup.CENTERPARENT)
}

func (a *app) openAbout() {}

func (a *app) openHost() {
	// The host panel is a modeless dialog on the web.
	iup.ShowXY(a.hostDialog(), iup.CENTERPARENT, iup.CENTERPARENT)
}

var _ = host.ErrUnavailable

// The app runs in a Worker (the IUP wasm loader does that), so browser
// actions are requests posted to the main thread (web/gguix-main.js).
func download(name string, data []byte) error {
	g := js.Global()
	arr := g.Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(arr, data)
	buf := arr.Get("buffer")
	g.Call("postMessage", map[string]any{"__gguixDownload": true, "name": name, "data": buf}, []any{buf})
	return nil
}

var pickDone func([]byte, bool, error)

func pickTextFile(done func([]byte, bool, error)) {
	g := js.Global()
	if pickDone == nil {
		g.Call("addEventListener", "message", js.FuncOf(func(_ js.Value, args []js.Value) any {
			d := args[0].Get("data")
			if d.Type() != js.TypeObject || !d.Get("__gguixPicked").Truthy() || pickDone == nil {
				return nil
			}
			f := pickDone
			switch {
			case d.Get("cancel").Truthy():
				f(nil, false, nil)
			case d.Get("tooBig").Truthy():
				f(make([]byte, 64*1024+1), true, nil)
			default:
				f([]byte(d.Get("text").String()), true, nil)
			}
			return nil
		}))
	}
	pickDone = done
	g.Call("postMessage", map[string]any{"__gguixPick": true})
}
