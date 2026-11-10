package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/matcolor"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/tree"

	gc "gguix/core"
	"gguix/core/host"
	"gguix/core/memory"
	"gguix/core/prefs"
)

var passed, failed, notFeasible atomic.Int64

func TestMain(m *testing.M) {
	code := m.Run()
	fmt.Printf("GGUIX_TESTS passed=%d failed=%d not_feasible=%d\n", passed.Load(), failed.Load(), notFeasible.Load())
	os.Exit(code)
}

// record counts the outcome of a scenario test.
func record(t *testing.T) {
	t.Cleanup(func() {
		switch {
		case t.Failed():
			failed.Add(1)
		case t.Skipped():
			notFeasible.Add(1)
		default:
			passed.Add(1)
		}
	})
}

// countingSvc counts Update calls.
type countingSvc struct {
	gc.DataService
	updates atomic.Int64
}

func (c *countingSvc) Update(ctx context.Context, it gc.Item) (gc.Item, error) {
	c.updates.Add(1)
	return c.DataService.Update(ctx, it)
}

type harness struct {
	t   *testing.T
	a   *App
	b   *core.Body
	svc *countingSvc
	dir string
}

func newHarness(t *testing.T, opts memory.Options) *harness {
	t.Helper()
	record(t)
	dir := t.TempDir()
	t.Setenv("GGUIX_PREFS_DIR", dir)
	svc := &countingSvc{DataService: memory.New(opts)}
	a := newApp(svc, prefs.NewFileStoreAt(dir), false)
	b := core.NewBody(windowTitle)
	a.buildUI(b)
	t.Cleanup(func() { core.AppearanceSettings.Theme = core.ThemeLight; core.AppearanceSettings.Apply() })
	return &harness{t: t, a: a, b: b, svc: svc, dir: dir}
}

// run shows the window and runs fn once the app has started.
func (h *harness) run(fn func()) {
	h.b.RenderImage(func() {
		h.ui(h.a.start)
		h.waitFor("first page", func() bool { return h.a.visibleLoaded() })
		fn()
	})
}

func (h *harness) ui(f func()) { h.a.async(f) }

func (h *harness) waitFor(what string, cond func() bool) {
	h.t.Helper()
	dl := time.Now().Add(8 * time.Second)
	for time.Now().Before(dl) {
		ok := false
		h.ui(func() { ok = cond() })
		if ok {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	h.t.Fatalf("timeout waiting for %s", what)
}

func (h *harness) cell(row, col int) core.Widget {
	return h.a.table.ListGrid.Child(row*nCols + col).(core.Widget)
}

func (h *harness) cellText(row, col int) string {
	return h.cell(row, col).(*core.Text).Text
}

func (h *harness) click(w core.Widget) { h.ui(func() { w.AsWidget().Send(events.Click) }) }

func (h *harness) get(f func() string) string {
	var s string
	h.ui(func() { s = f() })
	return s
}

func texts(n tree.Node) []string {
	var out []string
	n.AsTree().WalkDown(func(c tree.Node) bool {
		if tx, ok := c.(*core.Text); ok {
			out = append(out, tx.Text)
		}
		return tree.Continue
	})
	return out
}

func smallOpts() memory.Options { return memory.Options{Rows: 1000, Latency: 0} }

func TestT1Loads(t *testing.T) {
	h := newHarness(t, smallOpts())
	h.run(func() {
		for r := 0; r < 3; r++ {
			if got := h.get(func() string { return h.cellText(r, colID) }); got != fmt.Sprint(r+1) {
				t.Errorf("row %d id=%q", r, got)
			}
			if got := h.get(func() string { return h.cellText(r, colSKU) }); got != gc.SKUFor(int64(r+1)) {
				t.Errorf("row %d sku=%q", r, got)
			}
		}
	})
}

func TestT2Sort(t *testing.T) {
	h := newHarness(t, smallOpts())
	pg, _ := h.svc.Page(context.Background(), gc.PageRequest{Query: gc.Query{SortBy: gc.SortName}, Limit: 1})
	h.run(func() {
		btn := h.a.table.header.Child(2 * colName).(core.Widget)
		h.click(btn)
		h.waitFor("sorted", func() bool { return h.a.visibleLoaded() && h.cellText(0, colName) == pg.Items[0].Name })
		if txt := h.get(func() string { return btn.(*core.Button).Text }); !strings.HasSuffix(txt, "▲") {
			t.Errorf("header %q", txt)
		}
	})
}

func TestT3Search(t *testing.T) {
	h := newHarness(t, smallOpts())
	pg, _ := h.svc.Page(context.Background(), gc.PageRequest{Query: gc.Query{Search: "steel"}, Limit: 1})
	h.run(func() {
		h.ui(func() {
			h.a.tools.search.SetText("steel")
			h.a.tools.search.Send(events.Input)
		})
		want := gc.FormatCount(pg.Total) + " items"
		h.waitFor("count", func() bool { return strings.HasPrefix(h.a.status.StatusText(), want) })
	})
}

func (h *harness) selectRow(r int) {
	h.click(h.cell(r, colName))
}

func TestT4Selection(t *testing.T) {
	h := newHarness(t, smallOpts())
	h.run(func() {
		h.selectRow(1)
		h.waitFor("form", func() bool { return h.a.form.has })
		it, _ := h.svc.Get(context.Background(), 2)
		f := h.a.form
		h.ui(func() {
			chk := func(name, got, want string) {
				if got != want {
					t.Errorf("%s: %q want %q", name, got, want)
				}
			}
			chk("id", f.idT.Text, "2")
			chk("sku", f.skuT.Text, it.SKU)
			chk("name", f.name.Text(), it.Name)
			chk("category", f.cat.CurrentItem.Value.(string), it.Category.String())
			chk("price", f.price.Text(), gc.FormatPrice(it.PriceCents))
			chk("qty", fmt.Sprint(int32(f.qty.Value)), fmt.Sprint(it.Quantity))
			chk("status", f.statusValue().String(), it.Status.String())
			chk("rating", gc.FormatRating(f.ratingValue()), gc.FormatRating(it.Rating))
			chk("featured", fmt.Sprint(f.featured.IsChecked()), fmt.Sprint(it.Featured))
			chk("restock", f.restock.Text(), restockString(it))
			chk("updated", f.updT.Text, gc.FormatUpdated(it.UpdatedAt))
			chk("notes", strings.TrimRight(f.notesText(), "\n"), it.Notes)
		})
	})
}

func TestT5Validation(t *testing.T) {
	h := newHarness(t, smallOpts())
	h.run(func() {
		h.selectRow(0)
		h.waitFor("form", func() bool { return h.a.form.has })
		h.ui(func() {
			h.a.form.name.SetText("ab")
			h.a.form.name.Send(events.Input)
			h.a.form.save()
		})
		if got := h.get(func() string { return h.a.form.Errors()["name"] }); got != gc.MsgName {
			t.Errorf("error %q", got)
		}
		if n := h.svc.updates.Load(); n != 0 {
			t.Errorf("Update called %d times", n)
		}
	})
}

func TestT6Save(t *testing.T) {
	h := newHarness(t, smallOpts())
	h.run(func() {
		h.selectRow(0)
		h.waitFor("form", func() bool { return h.a.form.has })
		h.ui(func() {
			h.a.form.name.SetText("Valid New Name")
			h.a.form.name.Send(events.Input)
			h.a.form.save()
		})
		h.waitFor("saved row", func() bool { return h.cellText(0, colName) == "Valid New Name" })
	})
}

func TestT7ThemePrefs(t *testing.T) {
	h := newHarness(t, smallOpts())
	h.run(func() {
		h.ui(func() {
			h.a.tools.themeC.SetCurrentIndex(2)
			h.a.tools.themeC.Send(events.Change)
		})
		if !matcolor.SchemeIsDark {
			t.Errorf("not dark")
		}
		time.Sleep(900 * time.Millisecond)
		raw, err := os.ReadFile(filepath.Join(h.dir, "prefs.json"))
		if err != nil {
			t.Fatal(err)
		}
		var p gc.Prefs
		if json.Unmarshal(raw, &p) != nil || p.Theme != "dark" {
			t.Errorf("prefs %s", raw)
		}
	})
}

func TestT8ExportCancel(t *testing.T) {
	o := smallOpts()
	o.ExportRowsPerSec = 200
	h := newHarness(t, o)
	path := filepath.Join(h.dir, "out.csv")
	h.run(func() {
		var done atomic.Bool
		h.ui(func() { h.a.startExportTo(path, func(int, error) { done.Store(true) }) })
		h.waitFor("running", func() bool { return h.a.exp.running && h.a.tools.exportB.IsDisabled() })
		time.Sleep(200 * time.Millisecond)
		h.click(h.a.status.cancel)
		h.waitFor("stopped", func() bool { return done.Load() && !h.a.exp.running })
		if _, err := os.Stat(path); err == nil {
			t.Errorf("partial file not removed")
		}
		h.waitFor("button", func() bool { return !h.a.tools.exportB.IsDisabled() })
	})
}

func TestT9DirtyGuard(t *testing.T) {
	h := newHarness(t, smallOpts())
	h.run(func() {
		h.selectRow(0)
		h.waitFor("form", func() bool { return h.a.form.has })
		h.ui(func() {
			h.a.form.name.SetText("Edited Name")
			h.a.form.name.Send(events.Input)
		})
		h.selectRow(1)
		h.waitFor("confirm", func() bool { return h.a.confirming })
		if got := h.get(func() string { return h.a.form.idT.Text }); got != "1" {
			t.Errorf("selection changed to %s", got)
		}
	})
}

func TestT10CustomCells(t *testing.T) {
	h := newHarness(t, smallOpts())
	it, _ := h.svc.Get(context.Background(), 1)
	h.run(func() {
		h.ui(func() {
			pal := gc.Colors(matcolor.SchemeIsDark)
			sr := h.cell(0, colStock).(*stockRow)
			if sr.Bar.Fraction() != gc.StockFraction(it.Quantity) {
				t.Errorf("fraction %v", sr.Bar.Fraction())
			}
			if sr.Bar.FillColor() != gc.HexColor(pal.StockFillFor(it.Quantity)) {
				t.Errorf("fill color")
			}
			sc := h.cell(0, colStatus).(*statusCell)
			bg, fg := sc.Pill.BadgeColors()
			bc := pal.Badge(it.Status)
			if sc.Pill.Text.Text != it.Status.String() || bg != gc.HexColor(bc.Background) || fg != gc.HexColor(bc.Text) {
				t.Errorf("badge %q %v %v want %v", sc.Pill.Text.Text, bg, fg, bc)
			}
			rr := h.cell(0, colRating).(*ratingRow)
			f, hf, e := gc.Stars(it.Rating)
			if rr.Stars.Full != f || rr.Stars.Half != hf || rr.Stars.Empty != e {
				t.Errorf("stars %d %d %d", rr.Stars.Full, rr.Stars.Half, rr.Stars.Empty)
			}
		})
	})
}

func TestT11Cards(t *testing.T) {
	h := newHarness(t, smallOpts())
	it, _ := h.svc.Get(context.Background(), 1)
	h.run(func() {
		h.ui(func() {
			h.a.tools.views.SelectValue("Cards")
			h.a.setView(viewCards)
		})
		h.waitFor("cards loaded", func() bool { return h.a.visibleLoaded() && h.a.cards.ListGrid.NumChildren() > 1 })
		h.ui(func() {
			c := h.a.cards.ListGrid.Child(0).(*cardWidget)
			if c.Name.Text != it.Name || c.Price.Text != gc.FormatPrice(it.PriceCents) || c.Status.Text.Text != it.Status.String() {
				t.Errorf("card %q %q %q", c.Name.Text, c.Price.Text, c.Status.Text.Text)
			}
		})
		h.click(h.a.cards.ListGrid.Child(1).(core.Widget))
		h.waitFor("form", func() bool { return h.a.form.has && h.a.form.idT.Text == "2" })
	})
}

func TestT12Host(t *testing.T) {
	h := newHarness(t, smallOpts())
	h.run(func() {
		h.waitFor("host", func() bool { return h.a.hostInfo.Source != "" })
		want := host.Current(appID)
		h.ui(func() {
			tx := texts(h.a.hostBody())
			find := func(label string) string {
				for i, s := range tx {
					if s == label && i+1 < len(tx) {
						return tx[i+1]
					}
				}
				return "<missing>"
			}
			if find("Source") != string(want.Source) || find("Username") != want.Username {
				t.Errorf("panel %v", tx)
			}
			if got := h.a.status.IdentText(); got != want.Identity() {
				t.Errorf("status %q want %q", got, want.Identity())
			}
		})
	})
}

var _ = io.Discard
var _ = colors.Scheme
