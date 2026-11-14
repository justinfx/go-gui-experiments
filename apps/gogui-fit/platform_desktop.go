//go:build !js

package main

import (
	"errors"
	"os"

	"github.com/go-gui-org/go-gui/gui"

	"gguix/core/host"
)

// desktopHooks wires go-gui's native file dialogs and notifications. go-gui
// has no reveal-in-file-manager call, so Reveal uses the core helper.
func desktopHooks(w *gui.Window, guiApp *gui.App) platformHooks {
	return platformHooks{
		HostActions: true,
		SaveFile: func(name string, done func(string, bool, error)) {
			w.NativeSaveDialog(gui.NativeSaveDialogCfg{
				Title: "Export CSV", DefaultName: name, DefaultExtension: "csv", ConfirmOverwrite: true,
				Filters: []gui.NativeFileFilter{{Name: "CSV", Extensions: []string{"csv"}}},
				OnDone: func(r gui.NativeDialogResult, _ *gui.Window) {
					switch {
					case r.Status == gui.DialogError:
						done("", false, errors.New(r.ErrorMessage))
					case r.Status != gui.DialogOK || len(r.Paths) == 0:
						done("", false, nil)
					default:
						done(r.Paths[0].Path, true, nil)
					}
				},
			})
		},
		OpenTextFile: func(done func([]byte, bool, error)) {
			w.NativeOpenDialog(gui.NativeOpenDialogCfg{
				Title:   "Import Notes",
				Filters: []gui.NativeFileFilter{{Name: "Text", Extensions: []string{"txt"}}},
				OnDone: func(r gui.NativeDialogResult, _ *gui.Window) {
					if r.Status == gui.DialogError {
						done(nil, false, errors.New(r.ErrorMessage))
						return
					}
					if r.Status != gui.DialogOK || len(r.Paths) == 0 {
						done(nil, false, nil)
						return
					}
					p := r.Paths[0].Path
					// Stat first so a huge file is not read to learn it is too big.
					if st, err := os.Stat(p); err == nil && st.Size() > 64*1024 {
						done(make([]byte, 64*1024+1), true, nil)
						return
					}
					b, err := os.ReadFile(p)
					done(b, err == nil, err)
				},
			})
		},
		Notify: func(title, body string) bool {
			w.NativeNotification(gui.NativeNotificationCfg{Title: title, Body: body,
				OnDone: func(r gui.NativeNotificationResult, w *gui.Window) {
					if r.Status != gui.NotificationOK {
						a := gui.State[app](w)
						a.setMessage(title+": "+body, 4e9)
					}
				}})
			return true
		},
		Reveal: func(path string) error {
			if path == "" {
				return errors.New("nothing to reveal")
			}
			return host.RevealInFileManager(path)
		},
		Quit: func() {
			if guiApp != nil {
				gui.DispatchQuitRequest(guiApp)
			} else {
				w.Close()
			}
		},
	}
}
