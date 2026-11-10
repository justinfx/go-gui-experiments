package main

import (
	"context"
	"errors"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"unicode/utf8"

	"gguix/core"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Event payloads. Wails matches registered event data types exactly, so the
// emit calls below use these types verbatim.
type ExportProgress struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}
type ExportDone struct {
	Rows     int  `json:"rows"`
	Notified bool `json:"notified"` // an OS notification was sent
}
type ExportError struct {
	Message string `json:"message"`
}
type MenuAction struct {
	Action string `json:"action"`
}

func init() {
	application.RegisterEvent[ExportProgress]("export:progress")
	application.RegisterEvent[ExportDone]("export:done")
	application.RegisterEvent[ExportError]("export:error")
	application.RegisterEvent[application.Void]("export:cancelled")
	application.RegisterEvent[MenuAction]("menu:action")
	application.RegisterEvent[application.Void]("app:flush")
}

const appID = "wails"

// maxNotesFile is the largest notes file Import Notes accepts.
const maxNotesFile = 64 * 1024

type emitFunc func(name string, data any)

// ItemService is the one Wails service. Its exported methods are the
// frontend's whole backend API.
type ItemService struct {
	svc   core.DataService
	store core.PrefsStore
	plat  platform
	emit  emitFunc
	bench bool

	mu         sync.Mutex
	job        *exportJob
	lastExport string
	winW, winH int
	prefsCache core.Prefs
	havePrefs  bool
	jobs       map[string]*exportJob
	seq        int

	flushed     chan struct{} // closed-and-replaced signal: the frontend flushed prefs
	onMenuState func(MenuState)
}

func newItemService(svc core.DataService, store core.PrefsStore, emit emitFunc) *ItemService {
	return &ItemService{svc: svc, store: store, emit: emit, jobs: map[string]*exportJob{}, flushed: make(chan struct{}, 1)}
}

// ServiceName names the service in the bindings.
func (s *ItemService) ServiceName() string { return "ItemService" }

// ---- app info ----

// AppInfo describes the build and target.
type AppInfo struct {
	Target      string `json:"target"` // "desktop" or "web"
	Framework   string `json:"framework"`
	GoVersion   string `json:"goVersion"`
	BuildMarker string `json:"buildMarker"`
	Bench       bool   `json:"bench"`
	// ProcessStartMs is core.ProcessStart in Unix milliseconds (bench "ready").
	ProcessStartMs int64 `json:"processStartMs"`
}

// AppInfo returns build information used by the About dialog and to decide
// which target-specific UI to show.
func (s *ItemService) AppInfo() AppInfo {
	if os.Getenv("GGUIX_DEBUG") != "" {
		log.Println("AppInfo called")
	}
	return AppInfo{
		Target: s.plat.Target(), Framework: frameworkVersion(), GoVersion: runtime.Version(),
		BuildMarker: BuildMarker, Bench: s.bench, ProcessStartMs: core.ProcessStart.UnixMilli(),
	}
}

func frameworkVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, d := range bi.Deps {
			if d.Path == "github.com/wailsapp/wails/v3" {
				return "Wails " + d.Version
			}
		}
	}
	return "Wails v3"
}

// ---- data ----

// Page returns one page of the query result.
func (s *ItemService) Page(req PageArg) (PageView, error) {
	q, err := req.Query.ToQuery()
	if err != nil {
		return PageView{}, err
	}
	p, err := s.svc.Page(context.Background(), core.PageRequest{Query: q, Offset: req.Offset, Limit: req.Limit})
	if err != nil {
		return PageView{}, err
	}
	out := PageView{Offset: p.Offset, Total: p.Total, TotalText: core.FormatCount(p.Total), QueryKey: p.QueryKey, Items: make([]ItemView, len(p.Items))}
	for i, it := range p.Items {
		out.Items[i] = ToView(it)
	}
	return out, nil
}

// Get returns one item.
func (s *ItemService) Get(id int64) (ItemView, error) {
	it, err := s.svc.Get(context.Background(), id)
	if err != nil {
		return ItemView{}, err
	}
	return ToView(it), nil
}

// Update parses, validates and stores an edit. Invalid input is reported in
// UpdateResult.Errors (and DataService.Update is not called).
func (s *ItemService) Update(in ItemEdit) (UpdateResult, error) {
	it, errs := validateEdit(in)
	if len(errs) > 0 {
		return UpdateResult{Errors: errs}, nil
	}
	stored, err := s.svc.Update(context.Background(), it)
	if err != nil {
		if ve := core.AsValidationError(err); ve != nil {
			for _, fe := range ve.Errors {
				errs = append(errs, FieldErr{fe.Field, fe.Message})
			}
			return UpdateResult{Errors: errs}, nil
		}
		return UpdateResult{}, err
	}
	v := ToView(stored)
	return UpdateResult{Item: &v, Errors: []FieldErr{}}, nil
}

// FindRow returns the 0-based row of item id within the first maxRows rows of
// the query result, or -1 if it is not there.
func (s *ItemService) FindRow(q QueryArg, id int64, maxRows int) (int, error) {
	cq, err := q.ToQuery()
	if err != nil {
		return -1, err
	}
	row, ok, err := core.FindRow(context.Background(), s.svc, cq, id, maxRows)
	if err != nil || !ok {
		return -1, err
	}
	return row, nil
}

// ---- preferences ----

// LoadPrefs returns the saved preferences (defaults if none).
func (s *ItemService) LoadPrefs() (PrefsView, error) {
	p, err := s.store.Load()
	if err != nil {
		return prefsToView(core.DefaultPrefs()), err
	}
	return prefsToView(p.Sanitize()), nil
}

// SavePrefs stores the preferences. The window size comes from the window
// itself, not from the frontend.
func (s *ItemService) SavePrefs(v PrefsView) error {
	p, err := prefsFromView(v)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.winW > 0 && s.winH > 0 {
		p.WindowWidth, p.WindowHeight = s.winW, s.winH
	} else if cur, err := s.store.Load(); err == nil {
		p.WindowWidth, p.WindowHeight = cur.WindowWidth, cur.WindowHeight
	}
	s.prefsCache, s.havePrefs = p, true
	return s.store.Save(p)
}

// noteWindowSize records the window size and writes it to the preferences.
func (s *ItemService) noteWindowSize(w, h int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.winW, s.winH = w, h
	p := s.prefsCache
	if !s.havePrefs {
		cur, err := s.store.Load()
		if err != nil {
			return
		}
		p = cur
	}
	p.WindowWidth, p.WindowHeight = w, h
	s.prefsCache, s.havePrefs = p, true
	_ = s.store.Save(p)
}

// ---- clipboard and notes ----

// CopyResult says what CopyRow did.
type CopyResult struct {
	Text   string `json:"text"`
	Copied bool   `json:"copied"` // false: the frontend must use the browser clipboard
}

// CopyRow puts core.ItemCSVRow(item) on the system clipboard.
func (s *ItemService) CopyRow(id int64) (CopyResult, error) {
	it, err := s.svc.Get(context.Background(), id)
	if err != nil {
		return CopyResult{}, err
	}
	text := core.ItemCSVRow(it)
	return CopyResult{Text: text, Copied: s.plat.CopyText(text)}, nil
}

// ImportResult is the outcome of Import Notes.
type ImportResult struct {
	Cancelled bool   `json:"cancelled"`
	Text      string `json:"text"`
	Error     string `json:"error"` // shown by the frontend if ErrorShown is false
	ErrorSeen bool   `json:"errorShown"`
}

// ImportNotes opens a native file dialog for a .txt file and returns up to
// the first 500 characters.
func (s *ItemService) ImportNotes() (ImportResult, error) {
	path, ok, err := s.plat.ChooseNotesFile()
	if err != nil {
		return ImportResult{}, err
	}
	if !ok {
		return ImportResult{Cancelled: true}, nil
	}
	text, err := readNotes(path)
	if err != nil {
		shown := s.plat.ShowError("Import Notes", err.Error())
		return ImportResult{Error: err.Error(), ErrorSeen: shown}, nil
	}
	return ImportResult{Text: text}, nil
}

var errTooLarge = errors.New("File is too large.")

// readNotes reads a notes file: at most 64 KB, UTF-8, first 500 characters.
func readNotes(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return "", err
	}
	if st.Size() > maxNotesFile {
		return "", errTooLarge
	}
	b := make([]byte, st.Size())
	n, err := f.Read(b)
	if err != nil && n == 0 && st.Size() > 0 {
		return "", err
	}
	return firstRunes(strings.ToValidUTF8(string(b[:n]), "�"), 500), nil
}

func firstRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// ---- host ----

// HostInfo returns the host panel content for this target.
func (s *ItemService) HostInfo() HostView {
	i := s.plat.HostInfo()
	v := HostView{
		Source: string(i.Source), Username: i.Username, FullName: i.FullName, UID: i.UID, HomeDir: i.HomeDir,
		Hostname: i.Hostname, OS: i.OS, Arch: i.Arch, PID: i.PID, ConfigDir: i.ConfigDir, Locale: i.Locale,
		Notes: append([]string{}, i.Notes...), Identity: i.Identity(), Actions: i.Available() && s.plat.Target() == "desktop",
	}
	for _, r := range i.Rows() {
		v.Rows = append(v.Rows, HostRow{r.Label, r.Value})
	}
	return v
}

// RevealConfigFolder reveals the preferences directory in the file manager.
func (s *ItemService) RevealConfigFolder() error {
	i := s.plat.HostInfo()
	if !i.Available() {
		return errUnsupported
	}
	if err := os.MkdirAll(i.ConfigDir, 0o755); err != nil {
		return err
	}
	return s.plat.Reveal(i.ConfigDir)
}

// RevealLastExport reveals the most recently exported file.
func (s *ItemService) RevealLastExport() error {
	if !s.plat.HostInfo().Available() {
		return errUnsupported
	}
	s.mu.Lock()
	p := s.lastExport
	s.mu.Unlock()
	if p == "" {
		return errors.New("No export has completed in this session.")
	}
	return s.plat.Reveal(p)
}

// HasLastExport reports whether an export file is known (enables Reveal Last Export).
func (s *ItemService) HasLastExport() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastExport != ""
}

// Quit asks the platform to quit the app.
func (s *ItemService) Quit() { s.plat.Quit() }

// PrefsFlushed is called by the frontend after it has saved pending
// preferences in response to the app:flush event (desktop quit).
func (s *ItemService) PrefsFlushed() {
	select {
	case s.flushed <- struct{}{}:
	default:
	}
}

// MenuState is pushed by the frontend so that checks and enabled states
// match the UI.
type MenuState struct {
	View          string `json:"view"`  // table | cards
	Theme         string `json:"theme"` // system | light | dark
	ExportRunning bool   `json:"exportRunning"`
	HasExport     bool   `json:"hasExport"`
	HasSelection  bool   `json:"hasSelection"`
}

// SetMenuState tells the native menu about the UI state (desktop only; the
// web target has no menu and ignores it).
func (s *ItemService) SetMenuState(st MenuState) {
	if s.onMenuState != nil {
		s.onMenuState(st)
	}
}

// Palette returns the exact custom-rendering colors from core.Colors.
func (s *ItemService) Palette() PaletteSet {
	return PaletteSet{Light: paletteView(core.Colors(false)), Dark: paletteView(core.Colors(true))}
}
