//go:build !js

package prefs

import (
	"os"
	"path/filepath"

	"gguix/core"
)

// Dir returns the prefs directory: $GGUIX_PREFS_DIR if set, otherwise
// os.UserConfigDir()/gguix/<appID>. In js/wasm builds it returns
// "browser localStorage".
func Dir(appID string) string {
	if d := os.Getenv(EnvDir); d != "" {
		return d
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "gguix", appID)
}

// Default returns a FileStore for appID (native builds) or a
// LocalStorageStore (js/wasm builds).
func Default(appID string) core.PrefsStore { return NewFileStore(appID) }
