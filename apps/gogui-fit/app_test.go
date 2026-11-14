package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-gui-org/go-gui/gui"

	"gguix/core"
	"gguix/core/host"
	"gguix/core/memory"
	"gguix/core/prefs"
)

// The tests drive the real view through go-gui's headless test window
// (gui.NewTestWindow, TestRender, TestClick) plus the app model.

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

type harness struct {
	t   *testing.T
	a   *app
	w   *gui.Window
	svc *countingSvc
	dir string
}

func newHarness(t *testing.T, rps int) *harness {
	dir := t.TempDir()
	t.Setenv("GGUIX_PREFS_DIR", dir)
	svc := &countingSvc{DataService: memory.New(memory.Options{Rows: 1000, Latency: 0, ExportRowsPerSec: rps})}
	a := newApp(svc, prefs.NewFileStoreAt(dir), host.Current(appID), false)
	w := gui.NewTestWindow(t, gui.WindowCfg{State: a, Width: 2400, Height: 900})
	a.bindWindow(w)
	a.plat = platformHooks{HostActions: true, Notify: func(string, string) bool { return true }}
	h := &harness{t: t, a: a, w: w, svc: svc, dir: dir}
	t.Cleanup(a.pager.Close)
	w.TestRender(mainView)
	h.until("first page", func() bool { return h.has("SKU-0000001") })
	return h
}

func texts(l *gui.Layout, out *[]string) {
	if l.Shape != nil && l.Shape.TC != nil && l.Shape.TC.Text != "" {
		*out = append(*out, l.Shape.TC.Text)
	}
	for i := range l.Children {
		texts(&l.Children[i], out)
	}
}

func (h *harness) texts() []string {
	var out []string
	texts(h.w.TestRender(nil), &out)
	return out
}

func (h *harness) has(s string) bool {
	for _, x := range h.texts() {
		if x == s {
			return true
		}
	}
	return false
}

func (h *harness) until(what string, cond func() bool) {
	h.t.Helper()
	for i := 0; i < 400; i++ {
		h.w.FrameFn()
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	h.t.Fatalf("timed out waiting for %s; texts %q", what, h.texts())
}

func (h *harness) click(id string) {
	h.t.Helper()
	h.w.TestRender(nil)
	if ids := h.w.ResolveID(id); len(ids) > 0 {
		id = ids[0]
	}
	if err := h.w.TestClick(id); err != nil {
		h.t.Fatalf("click %s: %v", id, err)
	}
	h.w.TestRender(nil)
}

func (h *harness) selectRow(i int) {
	h.a.requestSelect(i)
	h.until("form", func() bool { return h.a.f.has })
}

func TestT1Start(t *testing.T) {
	h := newHarness(t, 0)
	for _, s := range []string{"1", "2", "3", "SKU-0000001", "SKU-0000002", "SKU-0000003"} {
		if !h.has(s) {
			t.Errorf("missing %q", s)
		}
	}
}

func TestT2Sort(t *testing.T) {
	h := newHarness(t, 0)
	h.click(gui.ScopeID(tableID, "header", "name"))
	want, _ := h.svc.Page(context.Background(), core.PageRequest{Query: core.Query{SortBy: core.SortName}, Limit: 1})
	h.until("sorted", func() bool { it, ok := h.a.pager.Row(0); return ok && it.Name == want.Items[0].Name })
	// DataGrid draws the sort arrow as its own glyph after the title.
	if !h.has("Name") || !h.has("▲") {
		t.Errorf("no arrow")
	}
}

func TestT3Search(t *testing.T) {
	h := newHarness(t, 0)
	h.a.setSearch("steel")
	want, _ := h.svc.Page(context.Background(), core.PageRequest{Query: core.Query{Search: "steel"}, Limit: 1})
	h.until("count", func() bool {
		for _, s := range h.texts() {
			if strings.HasPrefix(s, core.FormatCount(want.Total)+" items") {
				return true
			}
		}
		return false
	})
}

func TestT4Selection(t *testing.T) {
	h := newHarness(t, 0)
	h.click(gui.ScopeID(tableID, "row", "1"))
	h.until("form", func() bool { return h.a.f.has })
	it, _ := h.svc.Get(context.Background(), 1)
	f := h.a.f.cur
	if f.name != it.Name || f.price != core.FormatPrice(it.PriceCents) || f.cat != it.Category.String() || f.status != it.Status || f.featured != it.Featured || f.notes != it.Notes {
		t.Errorf("form %+v vs %+v", f, it)
	}
	if !h.has(it.SKU) || !h.has(it.Name) || !h.has(core.FormatUpdated(it.UpdatedAt)) {
		t.Errorf("fields not shown")
	}
}

func TestT5Validation(t *testing.T) {
	h := newHarness(t, 0)
	h.selectRow(0)
	h.a.f.cur.name = "ab"
	h.click("gg:save")
	if !h.has("Name must be 3 to 80 characters.") || h.svc.updates != 0 {
		t.Errorf("validation failed: updates=%d", h.svc.updates)
	}
}

func TestT6Save(t *testing.T) {
	h := newHarness(t, 0)
	h.selectRow(0)
	h.a.f.cur.name = "Renamed Thing"
	if !h.has("Unsaved changes") {
		t.Errorf("not dirty")
	}
	h.click("gg:save")
	h.until("row renamed", func() bool {
		n := 0
		for _, s := range h.texts() {
			if s == "Renamed Thing" {
				n++
			}
		}
		return n >= 2 && !h.has("Unsaved changes") // form field and table cell
	})
}

func TestT7Theme(t *testing.T) {
	h := newHarness(t, 0)
	h.a.setTheme("Dark")
	h.w.TestRender(nil)
	if h.w.Theme().Name != gui.ThemeDark.Name || !h.a.isDark() {
		t.Fatal("not dark")
	}
	h.until("prefs", func() bool {
		b, _ := os.ReadFile(filepath.Join(h.dir, "prefs.json"))
		return strings.Contains(string(b), `"theme":"dark"`)
	})
}

func TestT8ExportCancel(t *testing.T) {
	h := newHarness(t, 200)
	p := filepath.Join(h.dir, "out.csv")
	h.a.runExport(p, nil)
	h.until("running", func() bool { return h.has("Cancel") })
	h.click("gg:cancel")
	h.until("stopped", func() bool { return !h.a.exp.running })
	if _, err := os.Stat(p); err == nil {
		t.Error("partial file kept")
	}
	h.until("enabled", func() bool { return h.has("Export cancelled") })
}

func TestT9DirtyGuard(t *testing.T) {
	h := newHarness(t, 0)
	h.selectRow(0)
	h.a.f.cur.name = "Changed Name"
	h.click(gui.ScopeID(tableID, "row", "2"))
	h.until("confirm", func() bool { return h.w.DialogIsVisible() && h.has("Discard unsaved changes?") })
	h.click("gg:dlg-cancel")
	if h.a.selRow != 0 || h.a.f.cur.name != "Changed Name" {
		t.Errorf("selection lost")
	}
}

func find(l *gui.Layout, id string) *gui.Shape {
	if l.Shape != nil && l.Shape.ID == id {
		return l.Shape
	}
	for i := range l.Children {
		if s := find(&l.Children[i], id); s != nil {
			return s
		}
	}
	return nil
}

func TestT10CustomCells(t *testing.T) {
	for _, dark := range []bool{false, true} {
		h := newHarness(t, 0)
		if dark {
			h.a.setTheme("Dark")
		} else {
			h.a.setTheme("Light")
		}
		it, _ := h.svc.Get(context.Background(), 1)
		pal := core.Colors(dark)
		// Table: DataGrid has no custom cells, so stock, status and
		// rating are text fallbacks colored through CellFormat.
		h.until("fallback cells", func() bool { return h.has(stockText(it.Quantity)) && h.has(starsText(it.Rating)) })
		row := gridRow(it)
		if f := h.a.gridCellFormat(row, 0, gridColumns(0)[6], row.Cells["stock"], h.w); f.TextColor != hexColor(pal.StockFillFor(it.Quantity)) {
			t.Errorf("stock text color")
		}
		bc := pal.Badge(it.Status)
		if f := h.a.gridCellFormat(row, 0, gridColumns(0)[7], row.Cells["status"], h.w); f.BGColor != hexColor(bc.Background) || f.TextColor != hexColor(bc.Text) {
			t.Errorf("status cell colors")
		}
		// Cards keep the drawn stock bar and stars.
		h.a.setView(1)
		h.until("cards", func() bool { return find(h.w.TestRender(nil), "stockfill0") != nil })
		fill := find(h.w.TestRender(nil), "stockfill0")
		if fill == nil {
			t.Fatal("no stock fill")
		}
		if got := fill.Width / barW; got-float32(core.StockFraction(it.Quantity)) > 0.01 || float32(core.StockFraction(it.Quantity))-got > 0.01 {
			t.Errorf("fraction %v", got)
		}
		if fill.Color != hexColor(pal.StockFillFor(it.Quantity)) {
			t.Errorf("fill color")
		}
		if !h.has(it.Status.String()) {
			t.Errorf("no badge text")
		}
		b := pal.Badge(it.Status)
		if h.a.pal.Badge(it.Status) != b {
			t.Errorf("badge colors")
		}
		f, hf, e := core.Stars(it.Rating)
		if f+hf+e != 5 || find(h.w.TestRender(nil), "starsc0") == nil {
			t.Errorf("stars")
		}
	}
}

func TestT11Cards(t *testing.T) {
	h := newHarness(t, 0)
	h.click("gg:viewsw")
	h.a.setView(1)
	it1, _ := h.svc.Get(context.Background(), 1)
	it2, _ := h.svc.Get(context.Background(), 2)
	h.until("card", func() bool { return h.has(it1.Name) && h.has(core.FormatPrice(it1.PriceCents)) && h.has(it1.Status.String()) })
	h.click("c1")
	h.until("form", func() bool { return h.a.f.has && h.a.f.item.ID == 2 })
	if h.a.f.cur.name != it2.Name {
		t.Error("form")
	}
}

func TestT12Host(t *testing.T) {
	h := newHarness(t, 0)
	info := host.Current(appID)
	if !h.has(info.Username + "@" + info.Hostname) {
		t.Error("identity")
	}
	h.click("gg:host")
	h.until("panel", func() bool { return h.has("Source") && h.has(string(info.Source)) && h.has(info.Username) })
}
