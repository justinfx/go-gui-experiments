//go:build !js

package main

import (
	"errors"
	"os"

	"github.com/egoist/mygo"
)

// desktopHooks wires the native dialogs, notifications and file manager.
func desktopHooks(win func() *mygo.Window) platformHooks {
	return platformHooks{
		HostActions: true,
		SaveFile: func(name string) (string, bool, error) {
			p, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
				Parent:      win(),
				Title:       "Export CSV",
				DefaultPath: name,
				Filters:     []mygo.FileFilter{{Name: "CSV", Extensions: []string{"csv"}}},
			})
			return p, p != "", err
		},
		OpenTextFile: func() ([]byte, bool, error) {
			ps, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
				Parent:  win(),
				Title:   "Import Notes",
				Filters: []mygo.FileFilter{{Name: "Text", Extensions: []string{"txt"}}},
			})
			if err != nil || len(ps) == 0 {
				return nil, false, err
			}
			// Stat first so a huge file is not read to find out it is too big.
			if st, err := os.Stat(ps[0]); err == nil && st.Size() > 64*1024 {
				return make([]byte, 64*1024+1), true, nil
			}
			b, err := os.ReadFile(ps[0])
			return b, err == nil, err
		},
		Notify: func(title, body string) bool {
			if !mygo.NotificationsSupported() {
				return false
			}
			return mygo.NewNotification(mygo.NotificationOptions{Title: title, Body: body}).Show() == nil
		},
		Reveal: func(path string) error {
			if path == "" {
				return errors.New("nothing to reveal")
			}
			mygo.Shell.ShowItemInFolder(path)
			return nil
		},
		Quit: func() { mygo.App.Quit() },
	}
}
