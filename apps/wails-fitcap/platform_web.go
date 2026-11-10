//go:build server

package main

// Web platform (category: web): Wails server mode. Nothing here can touch the
// visitor's machine; the frontend uses browser equivalents.

import (
	"errors"

	"gguix/core/host"
)

type webPlatform struct{ quit func() }

func (p *webPlatform) Target() string { return "web" }

// The identity is the server process's, never the browser user's.
func (p *webPlatform) HostInfo() host.Info { return host.Server(appID) }

func (p *webPlatform) ChooseExportPath(string) (string, bool, error) {
	return "", false, errUnsupported
}
func (p *webPlatform) ChooseNotesFile() (string, bool, error) { return "", false, errUnsupported }
func (p *webPlatform) ShowError(string, string) bool          { return false }
func (p *webPlatform) CopyText(string) bool                   { return false }
func (p *webPlatform) Notify(string, string) error {
	return errors.New("OS notifications are not available in the browser")
}
func (p *webPlatform) Reveal(string) error { return errUnsupported }
func (p *webPlatform) Quit()               {}
