//go:build js

package prefs

import (
	"encoding/json"
	"errors"
	"syscall/js"

	"gguix/core"
)

// Dir returns "browser localStorage" in js/wasm builds.
func Dir(appID string) string { return "browser localStorage" }

// Default returns a LocalStorageStore for appID.
func Default(appID string) core.PrefsStore { return NewLocalStorageStore(appID) }

// LocalStorageStore keeps prefs in the browser's localStorage under the key
// "gguix/<appID>/prefs".
type LocalStorageStore struct{ key string }

// NewLocalStorageStore returns a store for appID.
func NewLocalStorageStore(appID string) *LocalStorageStore {
	return &LocalStorageStore{key: "gguix/" + appID + "/prefs"}
}

func storage() (js.Value, error) {
	ls := js.Global().Get("localStorage")
	if ls.IsUndefined() || ls.IsNull() {
		return js.Value{}, errors.New("localStorage unavailable")
	}
	return ls, nil
}

func (s *LocalStorageStore) Load() (core.Prefs, error) {
	p := core.DefaultPrefs()
	ls, err := storage()
	if err != nil {
		return p, err
	}
	v := ls.Call("getItem", s.key)
	if v.IsNull() || v.IsUndefined() {
		return p, nil
	}
	if err := json.Unmarshal([]byte(v.String()), &p); err != nil {
		return core.DefaultPrefs(), err
	}
	return p.Sanitize(), nil
}

func (s *LocalStorageStore) Save(p core.Prefs) error {
	ls, err := storage()
	if err != nil {
		return err
	}
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	ls.Call("setItem", s.key, string(b))
	return nil
}
