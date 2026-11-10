//go:build !remote

package backend

import (
	"gguix/core"
	"gguix/core/memory"
	"gguix/core/prefs"
)

// Mode is "memory" in default builds and "remote" with the remote tag.
const Mode = "memory"

// Open returns the in-process service and the default prefs store for appID.
func Open(appID string) (core.DataService, core.PrefsStore, error) {
	return memory.New(MemoryOptions()), prefs.Default(appID), nil
}
