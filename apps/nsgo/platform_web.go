//go:build js

package main

import (
	"github.com/nsgo-dev/nsgo/core"
)

// webHooks: the browser has no save dialog on disk, so the finished export is
// handed to NSGo's SaveFile with its bytes, which uses the browser's save
// picker or a download named gguix-export.csv. Host actions are disabled in
// the browser.
func webHooks(cx *core.Scope) platformHooks {
	h := sharedHooks(cx)
	h.Deliver = func(name string, data []byte) error {
		cx.Post(func() {
			cx.SaveFile(core.SaveOptions{Name: name, MIME: "text/csv", Data: data}, func(bool, string, error) {})
		})
		return nil
	}
	h.Stream = streamHook()
	h.Quit = func() {}
	return h
}
