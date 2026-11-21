package ui

import (
	"context"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/doug/gophics/app"
	"github.com/doug/gophics/geom"
	"github.com/doug/gophics/paint"
	"github.com/doug/gophics/theme"
	"github.com/doug/gophics/widget"

	"gguix/core"
	"gguix/core/host"
	"gguix/core/memory"
	"gguix/core/prefs"
)

type harness struct {
	t   *testing.T
	h   *app.Headless
	a   *App
	svc core.DataService
	dir string
}

func newHarness(t *testing.T, rps int) *harness {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GGUIX_PREFS_DIR", dir)
	svc := memory.New(memory.Options{Rows: 1000, Latency: 0, ExportRowsPerSec: rps})
	opts := &Options{Svc: svc, Store: prefs.NewFileStoreAt(dir), Host: host.Current(AppID), Prefs: core.DefaultPrefs()}
	cfg := Config(1280, 800)
	h, err := app.NewHeadless(Root{Opts: opts}, cfg, 1)
	if err != nil {
		t.Fatal(err)
	}
	hs := &harness{t: t, h: h, svc: svc, dir: dir}
	h.Render()
	hs.a = opts.App
	hs.waitFor("first page", hs.a.VisibleLoaded)
	return hs
}

// waitFor renders frames (which drain posted UI work) until cond holds.
func (hs *harness) waitFor(what string, cond func() bool) {
	hs.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		hs.h.Render()
		if cond() {
			hs.h.Render()
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	hs.t.Fatalf("timed out waiting for %s", what)
}

func (hs *harness) ui(fn func()) {
	hs.a.SetState(fn)
	hs.h.Render()
	hs.h.Render()
}

func (hs *harness) texts() []string { return hs.h.Scene().Texts() }

func (hs *harness) hasText(s string) bool {
	return slices.ContainsFunc(hs.texts(), func(t string) bool { return strings.Contains(t, s) })
}

func (hs *harness) page(q core.Query, n int) core.Page {
	p, err := hs.svc.Page(context.Background(), core.PageRequest{Query: q, Offset: 0, Limit: n})
	if err != nil {
		hs.t.Fatal(err)
	}
	return p
}

func (hs *harness) row(i int) core.Item {
	it, ok := hs.a.Pager.Row(i)
	if !ok {
		hs.t.Fatalf("row %d not loaded", i)
	}
	return it
}

func TestT1Startup(t *testing.T) {
	hs := newHarness(t, 0)
	for i, want := range []int64{1, 2, 3} {
		it := hs.row(i)
		if it.ID != want || it.SKU != core.SKUFor(want) {
			t.Fatalf("row %d = %d %s", i, it.ID, it.SKU)
		}
	}
	hs.h.Render()
	for _, s := range []string{"SKU-0000001", "SKU-0000002", "SKU-0000003"} {
		if !hs.hasText(s) {
			t.Fatalf("table does not show %s", s)
		}
	}
}

func TestT2SortByNameHeader(t *testing.T) {
	hs := newHarness(t, 0)
	// Tap the Name header cell: x inside the third column, y just above the body.
	x := float32(80 + 110 + 40)
	y := hs.a.listTop - 1 - headerH/2
	hs.h.Tap(geom.Pt{X: x, Y: y})
	hs.waitFor("sorted page", func() bool { return hs.a.Query.SortBy == core.SortName && hs.a.VisibleLoaded() })
	want := hs.page(core.Query{SortBy: core.SortName}, 1).Items[0]
	if got := hs.row(0); got.ID != want.ID || got.Name != want.Name {
		t.Fatalf("first row %q, want %q", got.Name, want.Name)
	}
	if hs.a.HeaderLabel(2) != "Name ▲" || !hs.hasText("Name ▲") {
		t.Fatalf("header label %q", hs.a.HeaderLabel(2))
	}
}

func TestT3Search(t *testing.T) {
	hs := newHarness(t, 0)
	hs.ui(func() { hs.a.SetSearchText("steel") })
	want := hs.page(core.Query{Search: "steel"}, 1).Total
	label := core.FormatCount(want) + " items"
	hs.waitFor("count", func() bool { return hs.hasText(label) && hs.a.Query.Search == "steel" })
}

func TestT4Selection(t *testing.T) {
	hs := newHarness(t, 0)
	// Tap the second body row.
	hs.h.Tap(geom.Pt{X: 300, Y: hs.a.listTop + TableRowH*1 + TableRowH/2})
	hs.h.Render()
	it := hs.row(1)
	if hs.a.SelectedID != it.ID {
		t.Fatalf("selected %d, want %d", hs.a.SelectedID, it.ID)
	}
	f := hs.a.Form
	want := formFrom(it)
	if f != want {
		t.Fatalf("form %+v, want %+v", f, want)
	}
	for _, s := range []string{it.SKU, it.Name, core.FormatPrice(it.PriceCents), it.Category.String(), core.FormatUpdated(it.UpdatedAt)} {
		if !hs.hasText(s) {
			t.Fatalf("form does not show %q", s)
		}
	}
}

func TestT5Validation(t *testing.T) {
	hs := newHarness(t, 0)
	hs.ui(func() { hs.a.SelectRow(0) })
	hs.ui(func() { hs.a.EditForm(func(f *Form) { f.Name = "ab" }) })
	hs.ui(hs.a.Save)
	if hs.a.Errors["name"] != core.MsgName || !hs.hasText(core.MsgName) {
		t.Fatalf("errors %v", hs.a.Errors)
	}
	if hs.a.UpdateCalls != 0 || !hs.a.Dirty() {
		t.Fatal("Update called or form not dirty")
	}
}

func TestT6Save(t *testing.T) {
	hs := newHarness(t, 0)
	hs.ui(func() { hs.a.SelectRow(0) })
	hs.ui(func() { hs.a.EditForm(func(f *Form) { f.Name = "Renamed Steel Lantern" }) })
	hs.ui(hs.a.Save)
	hs.waitFor("row refresh", func() bool {
		it, ok := hs.a.Pager.Row(0)
		return ok && it.Name == "Renamed Steel Lantern" && hs.hasText("Renamed Steel Lantern") && !hs.a.Dirty()
	})
}

func TestT7ThemePrefs(t *testing.T) {
	hs := newHarness(t, 0)
	hs.ui(func() { hs.a.SetTheme(core.ThemeDark) })
	if !hs.a.IsDark() {
		t.Fatal("not dark")
	}
	// The root fill uses the dark theme background.
	img := hs.h.Render()
	if !nearColor(img.At(5, 790), theme.Dark().Bg) {
		t.Fatalf("background %v", img.At(5, 790))
	}
	path := filepath.Join(hs.dir, "prefs.json")
	hs.waitFor("prefs saved", func() bool {
		b, err := os.ReadFile(path)
		return err == nil && strings.Contains(strings.ReplaceAll(string(b), " ", ""), `"theme":"dark"`)
	})
}

func TestT8ExportCancel(t *testing.T) {
	hs := newHarness(t, 200)
	path := filepath.Join(t.TempDir(), "export.csv")
	var doneErr error
	finished := false
	hs.ui(func() {
		hs.a.StartExportTo(path, func(_ int, err error) { finished, doneErr = true, err })
	})
	if !hs.a.Exporting || !hs.hasText("Cancel") {
		t.Fatal("export not running")
	}
	time.Sleep(300 * time.Millisecond)
	hs.ui(hs.a.CancelExport)
	hs.waitFor("export stop", func() bool { return finished })
	if doneErr == nil || hs.a.Exporting {
		t.Fatalf("not cancelled: %v", doneErr)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("partial file remains: %v", err)
	}
	if !hs.hasText("Export cancelled") || hs.hasText("%") {
		t.Fatal("status bar not reset")
	}
}

func TestT9DirtyGuard(t *testing.T) {
	hs := newHarness(t, 0)
	hs.ui(func() { hs.a.SelectRow(0) })
	hs.ui(func() { hs.a.EditForm(func(f *Form) { f.Name = "Changed Name" }) })
	hs.h.Tap(geom.Pt{X: 300, Y: hs.a.listTop + TableRowH*2 + TableRowH/2})
	hs.h.Render()
	if hs.a.DialogTitle != "Discard unsaved changes?" || !hs.hasText("Discard unsaved changes?") {
		t.Fatalf("no confirm dialog (%q)", hs.a.DialogTitle)
	}
	if hs.a.SelectedID != hs.row(0).ID {
		t.Fatal("selection moved")
	}
	hs.ui(hs.a.CancelDiscard)
	if hs.a.Form.Name != "Changed Name" {
		t.Fatal("edits lost on cancel")
	}
}

func TestT10CustomCells(t *testing.T) {
	hs := newHarness(t, 0)
	it := hs.row(0)
	for _, dark := range []bool{false, true} {
		th := theme.Light()
		if dark {
			th = theme.Dark()
		}
		pal := core.Colors(dark)
		stock := hs.a.cellContent(th, 6, it).(widget.Flex).Children[0].(StockBar)
		if stock.Fraction() != core.StockFraction(it.Quantity) || stock.FillHex() != pal.StockFillFor(it.Quantity) {
			t.Fatal("stock bar")
		}
		badge := hs.a.cellContent(th, 7, it).(StatusBadge)
		if badge.Text() != it.Status.String() || badge.Colors() != pal.Badge(it.Status) {
			t.Fatal("badge")
		}
		stars := hs.a.cellContent(th, 8, it).(widget.Flex).Children[0].(Stars)
		f, h, e := stars.Counts()
		wf, wh, we := core.Stars(it.Rating)
		if f != wf || h != wh || e != we {
			t.Fatal("stars")
		}
	}
	// And the pixels: widen the window so the Status and Rating columns are on
	// screen, then look for the light-theme badge and star colors.
	hs.h.Resize(geom.Size{W: 2400, H: 800})
	hs.h.Render()
	img := hs.h.Render()
	if !hasPixel(img, colorOf(core.Colors(false).Badge(it.Status).Background)) {
		t.Fatal("badge color not rendered")
	}
	if !hasPixel(img, colorOf(core.Colors(false).StarFilled)) {
		t.Fatal("star color not rendered")
	}
}

func TestT11Cards(t *testing.T) {
	hs := newHarness(t, 0)
	hs.ui(func() { hs.a.SetView(core.ViewCards) })
	hs.waitFor("cards", hs.a.VisibleLoaded)
	it := hs.row(0)
	for _, s := range []string{it.Name, core.FormatPrice(it.PriceCents), it.Status.String()} {
		if !hs.hasText(s) {
			t.Fatalf("card 1 missing %q", s)
		}
	}
	hs.h.Tap(geom.Pt{X: 200, Y: hs.a.listTop + CardExtent + CardH/2})
	hs.h.Render()
	it2 := hs.row(1)
	if hs.a.SelectedID != it2.ID || hs.a.Form.Name != it2.Name {
		t.Fatalf("selected %d", hs.a.SelectedID)
	}
}

func TestT12Host(t *testing.T) {
	hs := newHarness(t, 0)
	want := host.Current(AppID)
	hs.ui(hs.a.ShowHost)
	rows := hs.a.HostRows()
	if rows[0][0] != "Source" || rows[0][1] != string(want.Source) || rows[1][0] != "Username" || rows[1][1] != want.Username {
		t.Fatalf("host rows %v", rows[:2])
	}
	if !hs.hasText(string(want.Source)) || !hs.hasText(want.Identity()) {
		t.Fatal("host panel or status bar identity missing")
	}
}

func nearColor(c color.Color, want paint.Color) bool {
	r, g, b, _ := c.RGBA()
	d := func(x uint32, y float32) int { v := int(x>>8) - int(y*255+0.5); return max(v, -v) }
	return d(r, want.R) <= 3 && d(g, want.G) <= 3 && d(b, want.B) <= 3
}

func hasPixel(img image.Image, want paint.Color) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if nearColor(img.At(x, y), want) {
				return true
			}
		}
	}
	return false
}
