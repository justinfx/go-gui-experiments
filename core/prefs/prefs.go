// Package prefs provides core.PrefsStore implementations: FileStore for
// native builds, LocalStorageStore for js/wasm, and MemoryStore for tests.
//
// Default(appID) returns the right one for the build.
package prefs

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"gguix/core"
)

// EnvDir overrides the preferences directory (used by tests and the bench).
const EnvDir = "GGUIX_PREFS_DIR"

// FileStore keeps prefs as compact JSON in <dir>/prefs.json.
type FileStore struct {
	path string
	mu   sync.Mutex
}

// NewFileStore uses Dir(appID).
func NewFileStore(appID string) *FileStore { return NewFileStoreAt(Dir(appID)) }

// NewFileStoreAt uses an explicit directory.
func NewFileStoreAt(dir string) *FileStore { return &FileStore{path: filepath.Join(dir, "prefs.json")} }

// Path returns the prefs file path.
func (f *FileStore) Path() string { return f.path }

// Load reads the prefs file. A missing file returns core.DefaultPrefs().
// Fields missing from the file keep their defaults.
func (f *FileStore) Load() (core.Prefs, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := core.DefaultPrefs()
	b, err := os.ReadFile(f.path)
	if errors.Is(err, fs.ErrNotExist) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return core.DefaultPrefs(), err
	}
	return p.Sanitize(), nil
}

// Save writes the prefs atomically as compact JSON.
func (f *FileStore) Save(p core.Prefs) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return err
	}
	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, f.path)
}

// MemoryStore keeps prefs in memory, for tests.
type MemoryStore struct {
	mu    sync.Mutex
	p     core.Prefs
	saved bool
	saves int
}

// NewMemoryStore starts with core.DefaultPrefs().
func NewMemoryStore() *MemoryStore { return &MemoryStore{p: core.DefaultPrefs()} }

func (m *MemoryStore) Load() (core.Prefs, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.p, nil
}

func (m *MemoryStore) Save(p core.Prefs) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.p, m.saved = p, true
	m.saves++
	return nil
}

// Saves returns how many times Save was called.
func (m *MemoryStore) Saves() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saves
}

// JSON returns the stored prefs as compact JSON (what FileStore would write).
func (m *MemoryStore) JSON() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, _ := json.Marshal(m.p)
	return string(b)
}

var (
	_ core.PrefsStore = (*FileStore)(nil)
	_ core.PrefsStore = (*MemoryStore)(nil)
)
