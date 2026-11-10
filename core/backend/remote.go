//go:build remote

package backend

import (
	"gguix/core"
	"gguix/core/remote"
)

// Mode is "memory" in default builds and "remote" with the remote tag.
const Mode = "remote"

// Open returns a remote client (used as both DataService and PrefsStore).
func Open(appID string) (core.DataService, core.PrefsStore, error) {
	base, err := baseURL()
	if err != nil {
		return nil, nil, err
	}
	c := remote.New(base, appID)
	return c, c, nil
}
