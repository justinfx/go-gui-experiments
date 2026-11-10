//go:build !server

package main

// Desktop platform (category: desktop): native dialogs, clipboard,
// notifications and Finder reveal through Wails and the core host package.

import (
	"errors"

	"gguix/core/host"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

type desktopPlatform struct {
	app      *application.App
	window   application.Window
	notifier *notifications.NotificationService
}

func (p *desktopPlatform) Target() string { return "desktop" }

func (p *desktopPlatform) HostInfo() host.Info { return host.Current(appID) }

func (p *desktopPlatform) ChooseExportPath(defaultName string) (string, bool, error) {
	d := p.app.Dialog.SaveFile().SetFilename(defaultName).AddFilter("CSV files", "*.csv").
		CanCreateDirectories(true)
	if p.window != nil {
		d.AttachToWindow(p.window)
	}
	path, err := d.PromptForSingleSelection()
	if err != nil {
		return "", false, err
	}
	return path, path != "", nil
}

func (p *desktopPlatform) ChooseNotesFile() (string, bool, error) {
	d := p.app.Dialog.OpenFile().SetTitle("Import Notes").AddFilter("Text files", "*.txt").
		CanChooseFiles(true).CanChooseDirectories(false)
	if p.window != nil {
		d.AttachToWindow(p.window)
	}
	path, err := d.PromptForSingleSelection()
	if err != nil {
		return "", false, err
	}
	return path, path != "", nil
}

func (p *desktopPlatform) ShowError(title, message string) bool {
	d := p.app.Dialog.Error().SetTitle(title).SetMessage(message)
	if p.window != nil {
		d.AttachToWindow(p.window)
	}
	d.Show()
	return true
}

func (p *desktopPlatform) CopyText(text string) bool { return p.app.Clipboard.SetText(text) }

func (p *desktopPlatform) Notify(title, body string) error {
	if p.notifier == nil {
		return errors.New("notifications unavailable")
	}
	ok, err := p.notifier.CheckNotificationAuthorization()
	if err == nil && !ok {
		ok, err = p.notifier.RequestNotificationAuthorization()
	}
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("notification permission denied")
	}
	return p.notifier.SendNotification(notifications.NotificationOptions{
		ID: "gguix-export-" + title, Title: title, Body: body,
	})
}

func (p *desktopPlatform) Reveal(path string) error { return host.RevealInFileManager(path) }

func (p *desktopPlatform) Quit() { p.app.Quit() }
