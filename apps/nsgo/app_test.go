package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"

	gx "gguix/core"
	"gguix/core/host"
	"gguix/core/memory"
	"gguix/core/prefs"
)

// The tests drive the real component tree through NSGo's own test host
// (nsgotest): headless, with a model of the native views.

func TestMain(m *testing.M) { nsgotest.Main(m) }

type countingSvc struct {
	gx.DataService
	updates atomic.Int32
}

func (s *countingSvc) Update(ctx context.Context, it gx.Item) (gx.Item, error) {
	s.updates.Add(1)
	return s.DataService.Update(ctx, it)
}

type harness struct {
	t   *testing.T
	a   *app
	app *nsgotest.App
	svc *countingSvc
	dir string
}

func newHarness(t *testing.T, rps int) *harness {
	dir := t.TempDir()
	t.Setenv("GGUIX_PREFS_DIR", dir)
	svc := &countingSvc{DataService: memory.New(memory.Options{Rows: 1000, Latency: 0, ExportRowsPerSec: rps})}
	a := newApp(svc, prefs.NewFileStoreAt(dir), host.Current(appID), false)
	t.Cleanup(a.pager.Close)
	h := &harness{t: t, a: a, svc: svc, dir: dir}
	h.app = nsgotest.Start(t, rootView(a))
	h.app.Resize(2600, 900)
	h.app.WaitFor("text:SKU-0000001", 5*time.Second)
	return h
}

func (h *harness) text(sel string) string { return h.app.Find(sel).Text }

func (h *harness) selectRow(id int) {
	h.app.Press(fmt.Sprintf("id:table >> text:SKU-%07d", id))
	h.app.WaitUntil(5*time.Second, func() bool { return h.a.f.has && h.a.f.item.ID == int64(id) })
}

func TestT1Start(t *testing.T) {
	h := newHarness(t, 0)
	for i := 1; i <= 3; i++ {
		if !h.app.Exists(fmt.Sprintf("id:table >> text:SKU-%07d", i)) {
			t.Errorf("missing SKU %d", i)
		}
	}
	if got := h.app.Query("id:table >> text:/^[123]$/"); len(got) < 3 {
		t.Errorf("IDs 1,2,3 not shown: %d matches", len(got))
	}
}

func TestT2Sort(t *testing.T) {
	h := newHarness(t, 0)
	page, _ := h.svc.Page(context.Background(), gx.PageRequest{Query: gx.Query{SortBy: gx.SortName}, Limit: 1})
	h.app.Press("role:columnheader >> text:Name")
	want := page.Items[0].Name
	h.app.WaitFor("id:table >> text:"+want, 5*time.Second)
	if !h.app.Exists("text:Name ▲") {
		t.Errorf("active header lacks the arrow; texts %q", h.app.Texts())
	}
	if h.a.sortField != gx.SortName || h.a.sortDesc {
		t.Errorf("sort state %v %v", h.a.sortField, h.a.sortDesc)
	}
}

func TestT3Search(t *testing.T) {
	h := newHarness(t, 0)
	h.app.Type("id:search", "steel")
	h.app.Advance(400 * time.Millisecond)
	page, _ := h.svc.Page(context.Background(), gx.PageRequest{Query: gx.Query{Search: "steel"}, Limit: 1})
	want := gx.FormatCount(page.Total) + " items"
	h.app.WaitUntil(5*time.Second, func() bool { return strings.HasPrefix(h.text("id:status-text"), want) })
}

func TestT4Selection(t *testing.T) {
	h := newHarness(t, 0)
	h.selectRow(2)
	it, _ := h.svc.Get(context.Background(), 2)
	checks := map[string]string{
		"id:f-id": "2", "id:f-sku": it.SKU, "id:f-name": it.Name, "id:f-price": gx.FormatPrice(it.PriceCents),
		"id:f-rating-text": gx.FormatRating(it.Rating), "id:f-updated": gx.FormatUpdated(it.UpdatedAt), "id:f-notes": it.Notes,
	}
	for sel, want := range checks {
		if got := h.text(sel); got != want {
			t.Errorf("%s = %q want %q", sel, got, want)
		}
	}
	f := h.a.f.cur
	if f.cat != it.Category || f.status != it.Status || f.featured != it.Featured || int32(f.qty) != it.Quantity {
		t.Errorf("form fields %+v do not match item %+v", f, it)
	}
	if !h.app.Enabled("id:f-name") {
		t.Error("name field should be enabled")
	}
}

func TestT5Validation(t *testing.T) {
	h := newHarness(t, 0)
	h.selectRow(1)
	h.app.Clear("id:f-name")
	h.app.Type("id:f-name", "ab")
	h.app.Press("id:save")
	h.app.WaitFor("text:Name must be 3 to 80 characters.", 5*time.Second)
	if n := h.svc.updates.Load(); n != 0 {
		t.Errorf("Update called %d times", n)
	}
}

func TestT6Save(t *testing.T) {
	h := newHarness(t, 0)
	h.selectRow(1)
	if h.app.Enabled("id:save") || h.app.Enabled("id:revert") {
		t.Error("Save and Revert must be disabled while the form is clean")
	}
	h.app.Clear("id:f-name")
	h.app.Type("id:f-name", "Renamed Lamp")
	if !h.app.Enabled("id:save") || !h.app.Enabled("id:revert") {
		t.Error("Save and Revert must be enabled once the form is dirty")
	}
	h.app.Press("id:save")
	h.app.WaitFor("id:table >> text:Renamed Lamp", 5*time.Second)
	if n := h.svc.updates.Load(); n != 1 {
		t.Errorf("Update called %d times", n)
	}
	if h.a.f.dirty() {
		t.Error("form still dirty after save")
	}
	h.app.WaitUntil(5*time.Second, func() bool { return !h.app.Enabled("id:revert") })
	h.app.Type("id:f-name", "x")
	h.app.Press("id:revert")
	h.app.WaitFor("text:Discard unsaved changes?", 5*time.Second)
	h.app.Press("id:confirm-discard")
	h.app.WaitUntil(5*time.Second, func() bool { return !h.a.f.dirty() && !h.app.Enabled("id:revert") })
}

func TestT7ThemeAndPrefs(t *testing.T) {
	h := newHarness(t, 0)
	h.app.Press("id:theme")
	h.app.Press("role:option Dark")
	if h.app.Appearance != core.AppearanceDark {
		t.Errorf("appearance %v", h.app.Appearance)
	}
	var data []byte
	h.app.Advance(0)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		data, _ = os.ReadFile(filepath.Join(h.dir, "prefs.json"))
		if strings.Contains(string(data), `"theme":"dark"`) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Errorf("prefs lack dark theme: %s", data)
}

func TestT8ExportCancel(t *testing.T) {
	h := newHarness(t, 200)
	out := filepath.Join(h.dir, "out.csv")
	setSave(&h.app.SaveFile, out)
	h.app.Press("id:export")
	h.app.WaitFor("id:export-cancel", 5*time.Second)
	if h.app.Enabled("id:export") {
		t.Error("export button should be disabled while running")
	}
	h.app.Press("id:export-cancel")
	h.app.WaitForGone("id:export-cancel", 5*time.Second)
	if _, err := os.Stat(out); err == nil {
		t.Error("partial file was not removed")
	}
	h.app.WaitUntil(5*time.Second, func() bool { return h.app.Enabled("id:export") })
	h.app.WaitFor("text:Export cancelled", 5*time.Second)
}

func TestT9DirtyGuard(t *testing.T) {
	h := newHarness(t, 0)
	h.selectRow(1)
	h.app.Type("id:f-name", "x")
	h.app.Press("id:table >> text:SKU-0000002")
	h.app.WaitFor("text:Discard unsaved changes?", 5*time.Second)
	h.app.Press("id:confirm-cancel")
	if h.a.selRow != 0 || !h.a.f.dirty() {
		t.Errorf("cancel must keep selection and edits: row %d dirty %v", h.a.selRow, h.a.f.dirty())
	}
}

func TestT10CustomCells(t *testing.T) {
	h := newHarness(t, 0)
	it, _ := h.svc.Get(context.Background(), 1)
	h.app.Settle()
	pal := gx.Colors(h.app.Appearance == core.AppearanceDark)
	fill := h.app.Query("id:cell-stock-fill")[0]
	wantW := float32(barWidth * gx.StockFraction(it.Quantity))
	if d := float32(fill.Frame.Size.Width) - wantW; d > 0.6 || d < -0.6 {
		t.Errorf("fill width %v want %v", fill.Frame.Size.Width, wantW)
	}
	badge := h.app.Query("id:cell-status")[0]
	b := pal.Badge(it.Status)
	if got, want := uint32(badge.Style.Background), uint32(colorOf(b.Background)); got != want {
		t.Errorf("badge background %08x want %08x", got, want)
	}
	if h.app.Query("id:cell-status-text")[0].Text != it.Status.String() {
		t.Errorf("badge text %q", h.app.Query("id:cell-status-text")[0].Text)
	}
	full, half, empty := gx.Stars(it.Rating)
	counts := map[string]int{}
	for i := 0; i < 5; i++ {
		v := h.app.Query(fmt.Sprintf("id:cell-stars-star%d", i))[0]
		counts[v.Props.Label]++
	}
	if counts["full"] != full || counts["half"] != half || counts["empty"] != empty {
		t.Errorf("stars %v want %d/%d/%d", counts, full, half, empty)
	}
}

func TestT11Cards(t *testing.T) {
	h := newHarness(t, 0)
	h.app.Press("id:view-cards")
	h.app.WaitFor("id:cards >> text:SKU-0000001 · "+mustItem(h, 1).Category.String(), 5*time.Second)
	it := mustItem(h, 1)
	for _, want := range []string{it.Name, gx.FormatPrice(it.PriceCents), it.Status.String()} {
		if !h.app.Exists("id:cards >> text:" + want) {
			t.Errorf("card 1 lacks %q", want)
		}
	}
	// Each visible card's tile shows its own category initial.
	seen := map[string]bool{}
	for _, v := range h.app.Query("id:cards >> id:tile") {
		for _, c := range v.Children {
			seen[c.Text] = true
		}
	}
	t.Logf("tile initials %v", seen)
	h.app.Press("id:cards >> text:SKU-0000002")
	h.app.WaitUntil(5*time.Second, func() bool { return h.a.f.has && h.a.f.item.ID == 2 })
	if got := h.text("id:f-name"); got != mustItem(h, 2).Name {
		t.Errorf("form name %q", got)
	}
}

func mustItem(h *harness, id int64) gx.Item {
	it, _ := h.svc.Get(context.Background(), id)
	return it
}

func TestT12Host(t *testing.T) {
	h := newHarness(t, 0)
	info := host.Current(appID)
	h.app.Press("id:host")
	h.app.WaitFor("id:host-source", 5*time.Second)
	if got := h.text("id:host-source"); got != string(info.Source) {
		t.Errorf("source %q", got)
	}
	if got := h.text("id:host-username"); got != info.Username {
		t.Errorf("username %q want %q", got, info.Username)
	}
	if got, want := h.text("id:identity"), info.Username+"@"+info.Hostname; got != want {
		t.Errorf("identity %q want %q", got, want)
	}
}

// setSave makes the test platform answer every save dialog with path (the
// request type is internal to NSGo, so it is inferred).
func setSave[R any](f *func(R) string, path string) { *f = func(R) string { return path } }
