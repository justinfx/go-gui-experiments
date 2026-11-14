//go:build !e2e

package main

// Desktop platform (category: desktop): native dialogs, clipboard,
// notifications and Finder reveal through MyGo.

import (
	"errors"
	"runtime"

	"gguix/core/host"

	"github.com/egoist/mygo"
)

type desktopPlatform struct {
	window *mygo.Window
}

func (p *desktopPlatform) Target() string { return "desktop" }

func (p *desktopPlatform) HostInfo() host.Info { return host.Current(appID) }

func (p *desktopPlatform) ChooseExportPath(defaultName string) (string, bool, error) {
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent:            p.window,
		DefaultPath:       defaultName,
		Filters:           []mygo.FileFilter{{Name: "CSV files", Extensions: []string{"csv"}}},
		CreateDirectories: true,
	})
	if err != nil {
		return "", false, err
	}
	return path, path != "", nil
}

func (p *desktopPlatform) ChooseNotesFile() (string, bool, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:  p.window,
		Title:   "Import Notes",
		Filters: []mygo.FileFilter{{Name: "Text files", Extensions: []string{"txt"}}},
	})
	if err != nil {
		return "", false, err
	}
	if len(paths) == 0 {
		return "", false, nil
	}
	return paths[0], true, nil
}

func (p *desktopPlatform) ShowError(title, message string) bool {
	mygo.Dialog.Error(title, message)
	return true
}

func (p *desktopPlatform) CopyText(text string) bool {
	mygo.Clipboard.WriteText(text)
	return true
}

// Notify needs an app bundle on macOS: the notification center refuses
// to talk to a bare executable, which is how the bench harness launches
// the app, so notifications are skipped there with an error.
func (p *desktopPlatform) Notify(title, body string) error {
	if runtime.GOOS == "darwin" && !mygo.App.IsPackaged() {
		return errors.New("notifications need an app bundle on macOS")
	}
	return mygo.NewNotification(mygo.NotificationOptions{ID: "gguix-export-" + title, Title: title, Body: body}).Show()
}

// Reveal shows path in the file manager (the file selected, a folder opened).
func (p *desktopPlatform) Reveal(path string) error {
	mygo.Shell.ShowItemInFolder(path)
	return nil
}

func (p *desktopPlatform) Quit() { mygo.App.Quit() }
