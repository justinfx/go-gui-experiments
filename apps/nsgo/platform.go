package main

import (
	"io"

	"github.com/nsgo-dev/nsgo/core"
)

// sharedHooks are the hooks NSGo's own services serve identically on the
// desktop and in the browser: the file picker (on the web File.Open reads the
// chosen file) and notifications (the system's, or the browser's
// Notification API).
func sharedHooks(cx *core.Scope) platformHooks {
	return platformHooks{
		OpenTextFile: func(done func([]byte, bool, error)) {
			cx.PickFiles(core.PickFilesOptions{Types: []string{".txt"}}, func(files []core.File, err error) {
				if err != nil {
					done(nil, false, err)
					return
				}
				if len(files) == 0 {
					done(nil, false, nil)
					return
				}
				f := files[0]
				if f.Size > 64*1024 {
					done(make([]byte, 64*1024+1), true, nil)
					return
				}
				r, err := f.Open()
				if err != nil {
					done(nil, false, err)
					return
				}
				defer r.Close()
				b, err := io.ReadAll(io.LimitReader(r, 64*1024+1))
				done(b, err == nil, err)
			})
		},
		Notify: func(title, body string, shown func(bool)) {
			if err := cx.ShowNotification(core.Notification{ID: "export", Title: title, Body: body}, shown); err != nil {
				shown(false)
			}
		},
	}
}
