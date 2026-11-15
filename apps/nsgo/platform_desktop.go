//go:build !js

package main

import (
	"errors"

	"github.com/nsgo-dev/nsgo/core"

	"gguix/core/host"
)

// desktopHooks adds what only the desktop has to the shared hooks: the save
// dialog, revealing files and quitting. NSGo has no reveal-with-selection
// call, so revealing a file uses the core helper (open -R on macOS); a folder
// opens through NSGo's own OpenFile.
func desktopHooks(cx *core.Scope) platformHooks {
	h := sharedHooks(cx)
	h.HostActions = true
	// SaveFile shows the platform save dialog with no data: NSGo creates an
	// empty file at the chosen path (SaveRequest.SaveTo) and returns the path,
	// which the export then fills.
	h.SaveFile = func(name string, done func(string, bool, error)) {
		cx.SaveFile(core.SaveOptions{Name: name, MIME: "text/csv"}, func(saved bool, path string, err error) {
			done(path, saved, err)
		})
	}
	h.RevealFile = func(path string) error {
		if path == "" {
			return errors.New("nothing to reveal")
		}
		return host.RevealInFileManager(path)
	}
	// NSGo's OpenFile opens a folder in the file manager; where it cannot (on
	// Linux without the OpenURI portal it reports "not opened" or an error,
	// and shows nothing), the core helper's xdg-open / open takes over.
	h.OpenFolder = func(path string) error {
		cx.OpenFile(core.SharedFile{Path: path}, func(opened bool, err error) {
			if err != nil || !opened {
				_ = host.OpenWithDefaultApp(path)
			}
		})
		return nil
	}
	h.Quit = func() { cx.Quit() }
	return h
}
