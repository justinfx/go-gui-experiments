package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"gguix/core"
	"gguix/core/host"
	"gguix/core/memory"
	"gguix/core/prefs"
)

type fakePlat struct {
	path   string
	notes  string
	errs   []string
	notify []string
	clip   string
}

func (p *fakePlat) Target() string      { return "desktop" }
func (p *fakePlat) HostInfo() host.Info { return host.Current(appID) }
func (p *fakePlat) ChooseExportPath(string) (string, bool, error) {
	return p.path, p.path != "", nil
}
func (p *fakePlat) ChooseNotesFile() (string, bool, error) { return p.notes, p.notes != "", nil }
func (p *fakePlat) ShowError(t, m string) bool             { p.errs = append(p.errs, m); return true }
func (p *fakePlat) CopyText(s string) bool                 { p.clip = s; return true }
func (p *fakePlat) Notify(t, b string) error               { p.notify = append(p.notify, t+"|"+b); return nil }
func (p *fakePlat) Reveal(string) error                    { return nil }
func (p *fakePlat) Quit()                                  {}

type countingSvc struct {
	core.DataService
	mu      sync.Mutex
	updates int
}

func (c *countingSvc) Update(ctx context.Context, it core.Item) (core.Item, error) {
	c.mu.Lock()
	c.updates++
	c.mu.Unlock()
	return c.DataService.Update(ctx, it)
}

type evlog struct {
	mu   sync.Mutex
	list []string
}

func (e *evlog) emit(name string, _ any) { e.mu.Lock(); e.list = append(e.list, name); e.mu.Unlock() }
func (e *evlog) has(name string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, n := range e.list {
		if n == name {
			return true
		}
	}
	return false
}

func newTestService(t *testing.T, rps int) (*ItemService, *countingSvc, *fakePlat, *evlog) {
	t.Helper()
	t.Setenv("GGUIX_PREFS_DIR", t.TempDir())
	ds := &countingSvc{DataService: memory.New(memory.Options{Rows: 1000, Latency: 0, ExportRowsPerSec: rps})}
	ev := &evlog{}
	s := newItemService(ds, prefs.NewFileStore(appID), ev.emit)
	fp := &fakePlat{}
	s.plat = fp
	return s, ds, fp, ev
}

func TestPageMatchesCore(t *testing.T) {
	s, ds, _, _ := newTestService(t, -1)
	p, err := s.Page(PageArg{Query: QueryArg{SortBy: "name"}, Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	want, _ := ds.Page(context.Background(), core.PageRequest{Query: core.Query{SortBy: core.SortName}, Limit: 5})
	if p.Items[0].Name != want.Items[0].Name || p.Total != want.Total || p.TotalText != core.FormatCount(want.Total) {
		t.Fatalf("mismatch %v vs %v", p.Items[0].Name, want.Items[0].Name)
	}
	if p.Items[0].PriceText != core.FormatPrice(want.Items[0].PriceCents) {
		t.Fatal("price text")
	}
}

func TestUpdateValidationDoesNotCallService(t *testing.T) {
	s, ds, _, _ := newTestService(t, -1)
	it, _ := s.Get(1)
	e := ItemEdit{ID: 1, Name: "ab", Category: it.Category, Price: it.PriceText, Quantity: int64(it.Quantity), Status: it.Status, Rating: float64(it.Rating), Notes: it.Notes}
	r, err := s.Update(e)
	if err != nil || r.Item != nil || len(r.Errors) != 1 || r.Errors[0].Message != "Name must be 3 to 80 characters." {
		t.Fatalf("%+v %v", r, err)
	}
	if ds.updates != 0 {
		t.Fatal("Update was called")
	}
	e.Name = "Renamed Lantern"
	e.Price = "abc"
	r, _ = s.Update(e)
	if len(r.Errors) != 1 || r.Errors[0].Field != "priceCents" || r.Errors[0].Message != "Enter a price like 12.34." {
		t.Fatalf("%+v", r.Errors)
	}
	e.Price = "$1,234.50"
	r, _ = s.Update(e)
	if r.Item == nil || r.Item.Name != "Renamed Lantern" || r.Item.PriceText != "$1,234.50" || ds.updates != 1 {
		t.Fatalf("%+v", r)
	}
}

func TestExportCancelRemovesFile(t *testing.T) {
	s, _, fp, ev := newTestService(t, 200)
	fp.path = filepath.Join(t.TempDir(), "out.csv")
	st, err := s.StartExport(QueryArg{})
	if err != nil || !st.Started {
		t.Fatal(err, st)
	}
	time.Sleep(300 * time.Millisecond)
	s.CancelExport()
	deadline := time.Now().Add(3 * time.Second)
	for !ev.has("export:cancelled") && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !ev.has("export:cancelled") {
		t.Fatal("no cancelled event")
	}
	if _, err := os.Stat(fp.path); !os.IsNotExist(err) {
		t.Fatal("partial file remains")
	}
	// A new export can start again.
	fp.path = filepath.Join(t.TempDir(), "again.csv")
	s.svc.(*countingSvc).DataService = memory.New(memory.Options{Rows: 50, Latency: 0, ExportRowsPerSec: -1})
	if _, err := s.StartExport(QueryArg{}); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(3 * time.Second)
	for !ev.has("export:done") && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	b, _ := os.ReadFile(fp.path)
	if !strings.Contains(string(b), "# checksum") || len(fp.notify) != 1 || fp.notify[0] != "Export complete|50 rows written" {
		t.Fatalf("%q %v", b, fp.notify)
	}
	if !s.HasLastExport() {
		t.Fatal("last export")
	}
}

func TestHostInfoDesktop(t *testing.T) {
	s, _, _, _ := newTestService(t, -1)
	h := s.HostInfo()
	if h.Source != "native" || h.Username == "" || !h.Actions || h.Rows[0].Label != "Source" {
		t.Fatalf("%+v", h)
	}
}

func TestNotesTooLarge(t *testing.T) {
	s, _, fp, _ := newTestService(t, -1)
	f := filepath.Join(t.TempDir(), "n.txt")
	os.WriteFile(f, []byte(strings.Repeat("x", 65*1024)), 0o644)
	fp.notes = f
	r, _ := s.ImportNotes()
	if r.Error != "File is too large." || len(fp.errs) != 1 {
		t.Fatalf("%+v", r)
	}
	os.WriteFile(f, []byte(strings.Repeat("é", 600)), 0o644)
	r, _ = s.ImportNotes()
	if len([]rune(r.Text)) != 500 {
		t.Fatal(len([]rune(r.Text)))
	}
}

func TestPrefsRoundTripAndWindowSize(t *testing.T) {
	s, _, _, _ := newTestService(t, -1)
	s.noteWindowSize(1111, 777)
	v := prefsToView(core.DefaultPrefs())
	v.Theme = "dark"
	v.Categories = []string{"Toys", "Garden"}
	if err := s.SavePrefs(v); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(os.Getenv("GGUIX_PREFS_DIR"), "prefs.json"))
	if !strings.Contains(string(b), `"theme":"dark"`) {
		t.Fatal(string(b))
	}
	got, _ := s.LoadPrefs()
	if got.WindowWidth != 1111 || got.Theme != "dark" || len(got.Categories) != 2 {
		t.Fatalf("%+v", got)
	}
}

func TestFindRow(t *testing.T) {
	s, _, _, _ := newTestService(t, -1)
	r, err := s.FindRow(QueryArg{}, 42, 2000)
	if err != nil || r != 41 {
		t.Fatal(r, err)
	}
	r, _ = s.FindRow(QueryArg{}, 5000, 2000)
	if r != -1 {
		t.Fatal(r)
	}
}

func TestExportStreamWriter(t *testing.T) {
	s, _, _, _ := newTestService(t, -1)
	s.svc.(*countingSvc).DataService = memory.New(memory.Options{Rows: 20, Latency: 0, ExportRowsPerSec: -1})
	s.plat = &webLike{}
	st, err := s.StartExport(QueryArg{})
	if err != nil || st.URL == "" {
		t.Fatal(err, st)
	}
	if !s.serveExport(st.ID, io.Discard) {
		t.Fatal("job not found")
	}
}

type webLike struct{ fakePlat }

func (webLike) Target() string { return "web" }

// TestWriteOracle writes the expected values the Playwright tests compare the
// UI against. They come from the core, not from the app.
func TestWriteOracle(t *testing.T) {
	ds := memory.New(memory.Options{Rows: 1000, Latency: 0})
	ctx := context.Background()
	byName, _ := ds.Page(ctx, core.PageRequest{Query: core.Query{SortBy: core.SortName}, Limit: 1})
	steel, _ := ds.Page(ctx, core.PageRequest{Query: core.Query{Search: "steel"}, Limit: 1})
	first, _ := ds.Page(ctx, core.PageRequest{Limit: 3})
	var items []ItemView
	for _, it := range first.Items {
		items = append(items, ToView(it))
	}
	h := host.Server(appID)
	o := map[string]any{
		"firstItems": items, "nameFirst": byName.Items[0].Name, "steelTotal": core.FormatCount(steel.Total),
		"palette": Palette(), "serverUser": h.Username, "serverHost": h.Hostname,
	}
	b, _ := json.MarshalIndent(o, "", " ")
	_ = os.MkdirAll("frontend/e2e", 0o755)
	if err := os.WriteFile("frontend/e2e/oracle.json", b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func Palette() PaletteSet {
	return PaletteSet{Light: paletteView(core.Colors(false)), Dark: paletteView(core.Colors(true))}
}
