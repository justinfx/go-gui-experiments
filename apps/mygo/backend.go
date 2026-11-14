package main

import (
	"gguix/core"
	"gguix/core/backend"
	"gguix/core/memory"
	"gguix/core/prefs"
)

// openBackend returns the core services. Bench mode ignores the GGUIX_ROWS
// style overrides and always uses the default 1,000,000 row dataset.
func openBackend(bench bool) (core.DataService, core.PrefsStore, error) {
	if bench {
		return memory.New(memory.DefaultOptions()), prefs.Default(appID), nil
	}
	return backend.Open(appID)
}
