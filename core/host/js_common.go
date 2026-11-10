//go:build js

package host

import (
	"runtime"

	"gguix/core/prefs"
)

func unavailable(appID string) Info {
	return Info{
		Source:    SourceUnavailable,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		ConfigDir: prefs.Dir(appID),
		Notes:     []string{"browser sandbox: no host access"},
	}
}

// RevealInFileManager is unavailable in the browser.
func RevealInFileManager(path string) error { return ErrUnavailable }

// OpenWithDefaultApp is unavailable in the browser.
func OpenWithDefaultApp(path string) error { return ErrUnavailable }
