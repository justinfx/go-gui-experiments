package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"gguix/core"
	"gguix/core/host"
	"gguix/core/memory"
	"gguix/core/prefs"
)

var testW = 2400

type harness struct {
	t     *testing.T
	a     *app
	tt    *ui.Tester
	svc   *countingSvc
	dir   string
	mu    sync.Mutex
	queue []func()
	notes []string
}

type countingSvc struct {
	core.DataService
	mu      sync.Mutex
	updates int
}

func (s *countingSvc) Update(ctx context.Context, it core.Item) (core.Item, error) {
	s.mu.Lock()
	s.updates++
	s.mu.Unlock()
	return s.DataService.Update(ctx, it)
}

func newHarness(t *testing.T, exportRPS int) *harness {
	dir := t.TempDir()
	t.Setenv("GGUIX_PREFS_DIR", dir)
	inner := memory.New(memory.Options{Rows: 1000, Latency: 0, ExportRowsPerSec: exportRPS})
	svc := &countingSvc{DataService: inner}
	h := &harness{t: t, svc: svc, dir: dir}
	h.a = newApp(svc, prefs.NewFileStoreAt(dir), host.Current(appID), false)
	h.a.post = func(f func()) { h.mu.Lock(); h.queue = append(h.queue, f); h.mu.Unlock() }
	h.a.plat = platformHooks{
		HostActions: true,
		Notify:      func(title, body string) bool { h.notes = append(h.notes, title+": "+body); return true },
		SaveFile:    func(string) (string, bool, error) { return "", false, nil },
	}
	h.tt = ui.NewTester(h.a.view, testW, 800)
	t.Cleanup(h.a.pager.Close)
	h.waitLoaded()
	return h
}

func (h *harness) pump() {
	h.mu.Lock()
	q := h.queue
	h.queue = nil
	h.mu.Unlock()
	for _, f := range q {
		f()
	}
	h.tt.Frame()
}

func (h *harness) until(what string, cond func() bool) {
	h.t.Helper()
	for i := 0; i < 400; i++ {
		h.pump()
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	h.t.Fatalf("timed out waiting for %s; texts %q", what, h.tt.Texts())
}

func (h *harness) waitLoaded() {
	h.until("first page", func() bool { return h.tt.HasText("SKU-0000001") })
}

func (h *harness) click(label string) {
	h.t.Helper()
	if err := h.tt.Click(label); err != nil {
		h.t.Fatalf("click %q: %v", label, err)
	}
	h.pump()
}

func TestT1Start(t *testing.T) {
	h := newHarness(t, 0)
	for _, s := range []string{"SKU-0000001", "SKU-0000002", "SKU-0000003", "1,000 items"} {
		if !h.tt.HasText(s) {
			t.Errorf("missing %q in %q", s, h.tt.Texts())
		}
	}
}

func TestT2Sort(t *testing.T) {
	h := newHarness(t, 0)
	h.click("Name")
	want, _ := h.svc.Page(context.Background(), core.PageRequest{Query: core.Query{SortBy: core.SortName}, Limit: 1})
	h.until("sorted", func() bool { return h.tt.HasText(want.Items[0].Name) && h.a.sortField == core.SortName })
	if !h.tt.HasText("Name ▲") {
		t.Errorf("header arrow missing: %q", h.tt.Texts())
	}
	if it, _ := h.a.pager.Row(0); it.Name != want.Items[0].Name {
		t.Errorf("first row %q want %q", it.Name, want.Items[0].Name)
	}
}

func TestT3Search(t *testing.T) {
	h := newHarness(t, 0)
	h.click("Search")
	h.tt.Type("steel")
	h.tt.Advance(300 * time.Millisecond)
	want, _ := h.svc.Page(context.Background(), core.PageRequest{Query: core.Query{Search: "steel"}, Limit: 1})
	h.until("count", func() bool { return h.tt.HasText(core.FormatCount(want.Total) + " items") })
}

func (h *harness) selectRow1() {
	h.click("SKU-0000001")
	h.until("form loaded", func() bool { return h.a.f.has })
}

func TestT4Selection(t *testing.T) {
	h := newHarness(t, 0)
	h.selectRow1()
	it, _ := h.svc.Get(context.Background(), 1)
	f := h.a.f.cur
	if f.name != it.Name || f.price != core.FormatPrice(it.PriceCents) || f.cat != it.Category.String() ||
		f.qty != float64(it.Quantity) || f.status != it.Status || f.featured != it.Featured || f.notes != it.Notes ||
		f.rating != round1(float64(it.Rating)) {
		t.Errorf("form %+v does not match %+v", f, it)
	}
	if !h.tt.HasText(it.SKU) || !h.tt.HasText(core.FormatUpdated(it.UpdatedAt)) {
		t.Errorf("read-only fields missing: %q", h.tt.Texts())
	}
}

func (h *harness) setName(name string) {
	h.t.Helper()
	h.click("Name") // the field is labelled by its Field
	h.tt.Key(0, ui.KeyA|0)
	h.a.f.cur.name = name
	h.pump()
}

func TestT5Validation(t *testing.T) {
	h := newHarness(t, 0)
	h.selectRow1()
	h.a.f.cur.name = "ab"
	h.pump()
	h.click("Save")
	if !h.tt.HasText("Name must be 3 to 80 characters.") {
		t.Errorf("no error shown: %q", h.tt.Texts())
	}
	if h.svc.updates != 0 {
		t.Errorf("Update was called")
	}
}

func TestT6Save(t *testing.T) {
	h := newHarness(t, 0)
	h.selectRow1()
	h.a.f.cur.name = "Renamed Thing"
	h.pump()
	if !h.tt.HasText("Unsaved changes") {
		t.Errorf("no dirty marker")
	}
	h.click("Save")
	h.until("row renamed", func() bool { return h.tt.HasText("Renamed Thing") && !h.tt.HasText("Unsaved changes") })
	if h.svc.updates != 1 {
		t.Errorf("updates %d", h.svc.updates)
	}
}

func TestT7ThemePrefs(t *testing.T) {
	h := newHarness(t, 0)
	h.click("Theme")
	h.click("Dark")
	if !h.a.dark || !h.tt.HasText("Dark") {
		t.Fatalf("not dark")
	}
	h.until("prefs saved", func() bool {
		b, _ := os.ReadFile(filepath.Join(h.dir, "prefs.json"))
		return strings.Contains(string(b), `"theme":"dark"`)
	})
}

func TestT8ExportCancel(t *testing.T) {
	h := newHarness(t, 200)
	path := filepath.Join(h.dir, "out.csv")
	h.a.runExport(path, nil)
	h.until("running", func() bool { return h.a.exp.running && h.tt.HasText("Cancel") })
	h.click("Cancel")
	h.until("stopped", func() bool { return !h.a.exp.running })
	if _, err := os.Stat(path); err == nil {
		t.Errorf("partial file still exists")
	}
	if !h.tt.HasText("Export CSV") {
		t.Fatal("no export button")
	}
	if err := h.tt.Click("Export CSV"); err != nil {
		t.Errorf("export button not usable again: %v", err)
	}
	h.a.cancelExport()
}

func TestT9DirtyGuard(t *testing.T) {
	h := newHarness(t, 0)
	h.selectRow1()
	h.a.f.cur.name = "Changed Name"
	h.pump()
	h.click("SKU-0000002")
	if !h.tt.HasText("Discard unsaved changes?") {
		t.Fatalf("no confirm: %q", h.tt.Texts())
	}
	h.click("Cancel")
	if h.a.selRow != 0 || h.a.f.cur.name != "Changed Name" {
		t.Errorf("selection or edits lost: row %d", h.a.selRow)
	}
}

func pixel(tt *ui.Tester, r ui.Rect) [3]uint8 {
	img := tt.Image()
	sc := float32(img.Bounds().Dx()) / float32(testW)
	x, y := int((r.X+r.W/2)*sc), int((r.Y+r.H/2)*sc)
	c := img.RGBAAt(x, y)
	return [3]uint8{c.R, c.G, c.B}
}

func TestT10CustomCells(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprint("dark=", dark), func(t *testing.T) {
			h := newHarness(t, 0)
			if dark {
				h.a.themeSel = "Dark"
				h.pump()
			}
			it, _ := h.svc.Get(context.Background(), 1)
			pal := core.Colors(dark)
			track, ok := h.tt.Find("stock track")
			fill, ok2 := h.tt.Find("stock fill")
			if !ok || !ok2 {
				t.Fatalf("no stock bar")
			}
			if got, want := fill.W/track.W, float32(core.StockFraction(it.Quantity)); got < want-0.02 || got > want+0.02 {
				t.Errorf("fill fraction %v want %v", got, want)
			}
			if px, want := pixel(h.tt, fill), core.HexColor(pal.StockFillFor(it.Quantity)); px != [3]uint8{want.R, want.G, want.B} {
				t.Errorf("fill color %v want %v", px, want)
			}
			badge, ok := h.tt.Find("status badge")
			if !ok || !h.tt.HasText(it.Status.String()) {
				t.Fatalf("no badge")
			}
			bg := core.HexColor(pal.Badge(it.Status).Background)
			bx := ui.Rect{X: badge.X + 5, Y: badge.Y + badge.H/2}
			if px := pixel(h.tt, bx); px != [3]uint8{bg.R, bg.G, bg.B} {
				t.Errorf("badge bg %v want %v at %v", px, bg, badge)
			}
			full, half, empty := core.Stars(it.Rating)
			got := map[string]int{}
			for _, ai := range h.tt.AccessAll() {
				if strings.HasPrefix(ai.Name, "star ") {
					got[ai.Name]++
				}
			}
			_ = got
			// Only the first row's stars: count the first five star elements.
			var first []string
			for _, ai := range h.tt.AccessAll() {
				if strings.HasPrefix(ai.Name, "star ") && len(first) < 5 {
					first = append(first, ai.Name)
				}
			}
			c := map[string]int{}
			for _, n := range first {
				c[n]++
			}
			if c["star full"] != full || c["star half"] != half || c["star empty"] != empty {
				t.Errorf("stars %v want %d/%d/%d", c, full, half, empty)
			}
		})
	}
}

func TestT11Cards(t *testing.T) {
	h := newHarness(t, 0)
	h.click("Cards")
	it1, _ := h.svc.Get(context.Background(), 1)
	it2, _ := h.svc.Get(context.Background(), 2)
	h.until("cards", func() bool { return h.tt.HasText(it1.Name) && h.tt.HasText(core.FormatPrice(it1.PriceCents)) })
	if !h.tt.HasText(it1.Status.String()) {
		t.Errorf("no badge text")
	}
	h.click(it2.Name)
	h.until("form", func() bool { return h.a.f.has && h.a.f.item.ID == 2 })
	if h.a.f.cur.name != it2.Name {
		t.Errorf("form not filled")
	}
}

func TestT12Host(t *testing.T) {
	h := newHarness(t, 0)
	info := host.Current(appID)
	if !h.tt.HasText(info.Username + "@" + info.Hostname) {
		t.Errorf("status bar identity missing")
	}
	h.click("Host")
	for _, s := range []string{"Source", string(info.Source), "Username", info.Username} {
		if !h.tt.HasText(s) {
			t.Errorf("host panel missing %q: %q", s, h.tt.Texts())
		}
	}
}
