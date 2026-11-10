package main

import "gguix/core/host"

// platform is everything that differs between the desktop target and the
// server-mode web target. platform_desktop.go and platform_web.go provide the
// implementations.
type platform interface {
	// Target is "desktop" or "web".
	Target() string
	// HostInfo is the host identity for this target (native or server-process).
	HostInfo() host.Info
	// ChooseExportPath asks where to write the CSV. ok is false if cancelled.
	ChooseExportPath(defaultName string) (path string, ok bool, err error)
	// ChooseNotesFile asks for a .txt file. ok is false if cancelled.
	ChooseNotesFile() (path string, ok bool, err error)
	// ShowError shows a modal error dialog (desktop). Returns false if the
	// platform has none, so that the UI shows its own.
	ShowError(title, message string) bool
	// CopyText puts text on the system clipboard. Returns false if the
	// platform cannot, so that the UI uses the browser clipboard.
	CopyText(text string) bool
	// Notify sends an OS notification.
	Notify(title, body string) error
	// Reveal shows path in the file manager.
	Reveal(path string) error
	// Quit ends the app.
	Quit()
}
