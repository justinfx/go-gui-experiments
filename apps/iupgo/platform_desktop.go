//go:build !js

package main

import (
	"errors"
	"os"

	"github.com/gen2brain/iup-go/iup"

	"gguix/core/host"
)

// desktopHooks wires IUP's own file dialogs, notification, message dialogs
// and clipboard. IUP has no reveal-in-file-manager call, so Reveal uses
// the core helper.
func desktopHooks(a *app) platformHooks {
	return platformHooks{
		HostActions: true,
		SaveFile: func(name string, done func(string, bool, error)) {
			d := iup.FileDlg()
			defer iup.Destroy(d)
			set(d, "DIALOGTYPE", "SAVE", "TITLE", "Export CSV", "FILE", name, "EXTFILTER", "CSV files|*.csv|All files|*.*|", "EXTDEFAULT", "csv")
			iup.SetAttributeHandle(d, "PARENTDIALOG", a.ui.dlg)
			iup.Popup(d, iup.CENTERPARENT, iup.CENTERPARENT)
			if iup.GetAttribute(d, "STATUS") == "-1" || iup.GetAttribute(d, "VALUE") == "" {
				done("", false, nil)
				return
			}
			done(iup.GetAttribute(d, "VALUE"), true, nil)
		},
		OpenTextFile: func(done func([]byte, bool, error)) {
			d := iup.FileDlg()
			defer iup.Destroy(d)
			set(d, "DIALOGTYPE", "OPEN", "TITLE", "Import Notes", "EXTFILTER", "Text files|*.txt|", "EXTDEFAULT", "txt")
			iup.SetAttributeHandle(d, "PARENTDIALOG", a.ui.dlg)
			iup.Popup(d, iup.CENTERPARENT, iup.CENTERPARENT)
			p := iup.GetAttribute(d, "VALUE")
			if iup.GetAttribute(d, "STATUS") == "-1" || p == "" {
				done(nil, false, nil)
				return
			}
			// Stat first so a huge file is not read to learn it is too big.
			if st, err := os.Stat(p); err == nil && st.Size() > 64*1024 {
				done(make([]byte, 64*1024+1), true, nil)
				return
			}
			b, err := os.ReadFile(p)
			done(b, err == nil, err)
		},
		Notify: func(title, body string, failed func()) bool {
			n := iup.Notify()
			if iup.GetAttribute(n, "AVAILABLE") == "NO" {
				iup.Destroy(n)
				return false
			}
			set(n, "TITLE", title, "BODY", body)
			iup.On(n, iup.NotifyDismissCB, func(ih iup.Ihandle, reason int) int { iup.Destroy(ih); return iup.DEFAULT })
			iup.On(n, iup.ErrorCB, func(ih iup.Ihandle, msg string) int { failed(); return iup.DEFAULT })
			set(n, "SHOW", "YES")
			return true
		},
		Reveal: func(path string) error {
			if path == "" {
				return errors.New("nothing to reveal")
			}
			return host.RevealInFileManager(path)
		},
		Confirm: func(title, msg, okLabel, cancelLabel string, ok, cancel func()) {
			if iup.Alarm(title, msg, okLabel, cancelLabel, "") == 1 {
				ok()
			} else {
				cancel()
			}
		},
		Alert: func(msg string) { iup.MessageError(a.ui.dlg, msg) },
		Clipboard: func(text string) {
			c := iup.Clipboard()
			set(c, "TEXT", text)
			iup.Destroy(c)
		},
		Quit: func() { a.quit() },
	}
}

// quit saves the preferences and ends the main loop.
func (a *app) quit() {
	a.flushPrefs()
	iup.ExitLoop()
}
