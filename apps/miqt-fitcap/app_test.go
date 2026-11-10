//go:build !js

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gguix/core"
	"gguix/core/host"
	"gguix/core/memory"
	"gguix/core/prefs"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
)

var passed, failed, notFeasible atomic.Int32

// track records the outcome of a scenario for the final GGUIX_TESTS line.
func track(t *testing.T) {
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

// TestMain owns the Qt main thread: the event loop runs here and the tests
// run in a goroutine, reaching widgets through mainthread.Wait.
func TestMain(m *testing.M) {
	os.Setenv("QT_QPA_PLATFORM", "offscreen")
	dir, err := os.MkdirTemp("", "gguix-miqt-test")
	if err != nil {
		panic(err)
	}
	os.Setenv("GGUIX_PREFS_DIR", dir)
	qt.NewQApplication([]string{"gguix-test"})
	qt.QGuiApplication_SetQuitOnLastWindowClosed(false)
	code := 0
	go func() {
		code = m.Run()
		fmt.Printf("GGUIX_TESTS passed=%d failed=%d not_feasible=%d\n", passed.Load(), failed.Load(), notFeasible.Load())
		mainthread.Start(qt.QCoreApplication_Quit)
	}()
	qt.QApplication_Exec()
	os.RemoveAll(dir)
	os.Exit(code)
}

func ui(f func()) { mainthread.Wait(f) }

// countingSvc counts Update calls.
type countingSvc struct {
	core.DataService
	updates atomic.Int32
}

func (c *countingSvc) Update(ctx context.Context, it core.Item) (core.Item, error) {
	c.updates.Add(1)
	return c.DataService.Update(ctx, it)
}

type harness struct {
	t     *testing.T
	a     *App
	svc   *countingSvc
	ref   core.DataService
	dir   string
	conf  atomic.Int32
	allow atomic.Bool
	save  atomic.Value // string
	errs  []string
	mu    sync.Mutex
}

func newHarness(t *testing.T, exportRPS int) *harness {
	t.Helper()
	track(t)
	h := &harness{t: t, dir: t.TempDir()}
	t.Setenv("GGUIX_PREFS_DIR", h.dir)
	opts := memory.Options{Rows: 1000, Latency: 0, ExportRowsPerSec: exportRPS}
	h.svc = &countingSvc{DataService: memory.New(opts)}
	h.ref = memory.New(opts)
	ui(func() {
		h.a = newApp(options{svc: h.svc, store: prefs.NewFileStore(appID)})
		h.a.dlg.confirmDiscard = func() bool { h.conf.Add(1); return h.allow.Load() }
		h.a.dlg.errorBox = func(title, text string) { h.mu.Lock(); h.errs = append(h.errs, title+": "+text); h.mu.Unlock() }
		h.a.dlg.saveFile = func(string) string {
			p, _ := h.save.Load().(string)
			return p
		}
		h.a.dlg.notify = func(string, string) {}
		h.a.win.Show()
		h.a.start()
	})
	t.Cleanup(func() {
		ui(func() {
			h.a.cancelExport()
			h.a.cancelAll()
			h.a.win.Close()
			h.a.win.DeleteLater()
		})
	})
	h.waitLoaded()
	return h
}

// waitFor polls cond on the UI thread.
func (h *harness) waitFor(what string, d time.Duration, cond func() bool) {
	h.t.Helper()
	deadline := time.Now().Add(d)
	for {
		ok := false
		ui(func() { ok = cond() })
		if ok {
			return
		}
		if time.Now().After(deadline) {
			h.t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (h *harness) waitLoaded() {
	h.t.Helper()
	h.waitFor("first page", 5*time.Second, func() bool { return h.a.tmodel.shown > 0 && h.a.visibleLoaded() })
}

func (h *harness) cell(row, col int) string {
	var s string
	ui(func() {
		m := h.a.tmodel
		s = m.Data(m.Index(row, col, qt.NewQModelIndex()), int(qt.DisplayRole)).ToString()
	})
	return s
}

func (h *harness) status() string {
	var s string
	ui(func() { s = h.a.statusLabel.Text() })
	return s
}

// selectRowUser selects a row through the selection model, like a click.
func (h *harness) selectRowUser(row int) {
	ui(func() {
		m := h.a.tmodel
		h.a.table.SelectionModel().SetCurrentIndex(m.Index(row, 0, qt.NewQModelIndex()),
			qt.QItemSelectionModel__ClearAndSelect|qt.QItemSelectionModel__Rows)
	})
}

func (h *harness) item(id int64) core.Item {
	it, err := h.ref.Get(context.Background(), id)
	if err != nil {
		h.t.Fatal(err)
	}
	return it
}

func (h *harness) shot(name string) {
	dir := os.Getenv("GGUIX_SHOTS")
	if dir == "" {
		return
	}
	ui(func() {
		qt.QCoreApplication_ProcessEvents()
		h.a.win.Grab().Save(filepath.Join(dir, name+".png"))
	})
}

func TestT1StartsAndLoads(t *testing.T) {
	h := newHarness(t, 0)
	for i, id := range []int{1, 2, 3} {
		if got := h.cell(i, colID); got != fmt.Sprint(id) {
			t.Errorf("row %d ID = %q, want %d", i, got, id)
		}
		if got, want := h.cell(i, colSKU), core.SKUFor(int64(id)); got != want {
			t.Errorf("row %d SKU = %q, want %q", i, got, want)
		}
	}
	if s := h.status(); !strings.HasPrefix(s, "1,000 items") {
		t.Errorf("status = %q", s)
	}
	h.shot("t1-table")
}

func TestT2Sort(t *testing.T) {
	h := newHarness(t, 0)
	ui(func() { h.a.table.HorizontalHeader().SectionClicked(colName) })
	page, err := h.ref.Page(context.Background(), core.PageRequest{Query: core.Query{SortBy: core.SortName}, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	want := page.Items[0].Name
	h.waitFor("sorted rows", 5*time.Second, func() bool { return h.a.visibleLoaded() && h.cell0() == want })
	var hdr string
	ui(func() {
		hdr = h.a.tmodel.HeaderData(colName, qt.Horizontal, int(qt.DisplayRole)).ToString()
	})
	if hdr != "Name ▲" {
		t.Errorf("header = %q, want %q", hdr, "Name ▲")
	}
	ui(func() { h.a.table.HorizontalHeader().SectionClicked(colName) })
	h.waitFor("descending", 5*time.Second, func() bool { return h.a.visibleLoaded() && h.a.query.Desc })
}

// cell0 reads row 0, Name; call only from the UI thread.
func (h *harness) cell0() string {
	m := h.a.tmodel
	return m.Data(m.Index(0, colName, qt.NewQModelIndex()), int(qt.DisplayRole)).ToString()
}

func TestT3Search(t *testing.T) {
	h := newHarness(t, 0)
	page, err := h.ref.Page(context.Background(), core.PageRequest{Query: core.Query{Search: "steel"}, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total == 0 {
		t.Fatal("no steel rows in the test dataset")
	}
	ui(func() { h.a.search.SetText("steel") })
	want := core.FormatCount(page.Total) + " items"
	h.waitFor("count in status bar", 5*time.Second, func() bool {
		return strings.HasPrefix(h.a.statusLabel.Text(), want)
	})
}

func TestT4Selection(t *testing.T) {
	h := newHarness(t, 0)
	h.selectRowUser(3)
	want := h.item(4)
	h.waitFor("form loaded", 5*time.Second, func() bool { return h.a.form.item != nil })
	ui(func() {
		f := h.a.form
		check := func(name string, got, exp any) {
			if fmt.Sprint(got) != fmt.Sprint(exp) {
				t.Errorf("%s = %v, want %v", name, got, exp)
			}
		}
		check("ID", f.idL.Text(), want.ID)
		check("SKU", f.skuL.Text(), want.SKU)
		check("Name", f.name.Text(), want.Name)
		check("Category", f.cat.CurrentIndex(), int(want.Category))
		check("Price", f.price.Text(), core.FormatPrice(want.PriceCents))
		check("Quantity", f.qty.Value(), want.Quantity)
		check("Status", f.statusGroup.CheckedId(), int(want.Status))
		check("Rating", f.rating.Value(), core.RatingTenths(want.Rating))
		check("Rating label", f.ratingL.Text(), core.FormatRating(want.Rating))
		check("Featured", f.featured.IsChecked(), want.Featured)
		exp := ""
		if want.RestockDate != nil {
			exp = want.RestockDate.String()
		}
		check("Restock", f.restockString(), exp)
		check("Updated", f.updL.Text(), core.FormatUpdated(want.UpdatedAt))
		check("Notes", f.notes.ToPlainText(), want.Notes)
		if f.isDirty() {
			t.Error("form is dirty right after loading")
		}
		if !f.name.IsEnabled() {
			t.Error("form is disabled with a selection")
		}
	})
}

func TestT5Validation(t *testing.T) {
	h := newHarness(t, 0)
	h.selectRowUser(0)
	h.waitFor("form loaded", 5*time.Second, func() bool { return h.a.form.item != nil })
	ui(func() { h.a.form.name.SetText("ab"); h.a.form.save.Click() })
	var msg string
	var dirty bool
	ui(func() { msg = h.a.form.errorText("name"); dirty = h.a.form.isDirty() })
	if msg != "Name must be 3 to 80 characters." {
		t.Errorf("error under Name = %q", msg)
	}
	if !dirty {
		t.Error("form should stay dirty")
	}
	time.Sleep(100 * time.Millisecond)
	if n := h.svc.updates.Load(); n != 0 {
		t.Errorf("Update was called %d times", n)
	}
}

func TestT6Save(t *testing.T) {
	h := newHarness(t, 0)
	h.selectRowUser(0)
	h.waitFor("form loaded", 5*time.Second, func() bool { return h.a.form.item != nil })
	ui(func() { h.a.form.name.SetText("Renamed Test Lantern"); h.a.form.save.Click() })
	h.waitFor("saved row in table", 5*time.Second, func() bool {
		return h.a.visibleLoaded() && h.svc.updates.Load() == 1 && h.cell0() == "Renamed Test Lantern"
	})
	ui(func() {
		if h.a.form.isDirty() {
			t.Error("form still dirty after save")
		}
	})
}

func TestT7ThemeAndPrefs(t *testing.T) {
	h := newHarness(t, 0)
	ui(func() { h.a.themeCombo.SetCurrentIndex(2) })
	var dark bool
	var theme string
	ui(func() { dark, theme = h.a.isDark(), h.a.theme })
	if !dark || theme != core.ThemeDark {
		t.Errorf("theme = %q dark = %v", theme, dark)
	}
	h.waitFor("prefs file", 5*time.Second, func() bool {
		b, err := os.ReadFile(filepath.Join(h.dir, "prefs.json"))
		return err == nil && strings.Contains(string(b), `"theme":"dark"`)
	})
	h.shot("t7-dark")
}

func TestT8ExportCancel(t *testing.T) {
	h := newHarness(t, 200)
	out := filepath.Join(t.TempDir(), "out.csv")
	h.save.Store(out)
	ui(func() { h.a.exportBtn.Click() })
	h.waitFor("export running", 5*time.Second, func() bool {
		_, err := os.Stat(out)
		return h.a.exporting && err == nil && !h.a.exportBtn.IsEnabled() && !h.a.cancelBtn.IsHidden()
	})
	ui(func() { h.a.cancelBtn.Click() })
	h.waitFor("export stopped", 5*time.Second, func() bool { return !h.a.exporting && h.a.exportBtn.IsEnabled() })
	if _, err := os.Stat(out); err == nil {
		t.Error("partial file was not removed")
	}
	var msg string
	ui(func() { msg = h.a.win.StatusBar().CurrentMessage() })
	if msg != "Export cancelled" {
		t.Errorf("status message = %q", msg)
	}
}

func TestT9DirtyGuard(t *testing.T) {
	h := newHarness(t, 0)
	h.selectRowUser(0)
	h.waitFor("form loaded", 5*time.Second, func() bool { return h.a.form.item != nil })
	ui(func() { h.a.form.name.SetText("Edited Name") })
	h.allow.Store(false)
	h.selectRowUser(1)
	if n := h.conf.Load(); n != 1 {
		t.Fatalf("confirm dialog shown %d times, want 1", n)
	}
	ui(func() {
		if h.a.selectedRow != 0 || h.a.form.name.Text() != "Edited Name" {
			t.Errorf("selection or edits changed after Cancel: row %d name %q", h.a.selectedRow, h.a.form.name.Text())
		}
	})
	h.allow.Store(true)
	h.selectRowUser(1)
	h.waitFor("moved to row 2", 5*time.Second, func() bool { return h.a.selectedRow == 1 && h.a.form.item != nil && h.a.form.item.ID == 2 })
	if n := h.conf.Load(); n != 2 {
		t.Errorf("confirm dialog shown %d times, want 2", n)
	}
}

func TestT10CustomCells(t *testing.T) {
	h := newHarness(t, 0)
	it := h.item(1)
	for _, dark := range []bool{false, true} {
		theme := core.ThemeLight
		if dark {
			theme = core.ThemeDark
		}
		ui(func() {
			h.a.applyTheme(theme, false)
			h.a.tdel.log = map[[2]int]paintRecord{}
			// Status and Rating lie beyond the right edge at the default width.
			h.a.table.HorizontalScrollBar().SetValue(h.a.table.HorizontalScrollBar().Maximum())
			h.a.table.Grab() // forces a paint through the delegate
		})
		pal := core.Colors(dark)
		var stock, status, rating paintRecord
		ui(func() {
			stock, status, rating = h.a.tdel.log[[2]int{0, colStock}], h.a.tdel.log[[2]int{0, colStatus}], h.a.tdel.log[[2]int{0, colRating}]
		})
		if stock.Kind != "stock" || status.Kind != "status" || rating.Kind != "rating" {
			t.Fatalf("delegate did not paint row 1 (dark=%v): %+v %+v %+v", dark, stock, status, rating)
		}
		if stock.Fraction != core.StockFraction(it.Quantity) {
			t.Errorf("stock fraction %v, want %v", stock.Fraction, core.StockFraction(it.Quantity))
		}
		if stock.TrackHex != pal.StockTrack || stock.FillHex != pal.StockFillFor(it.Quantity) {
			t.Errorf("stock colors %s/%s, want %s/%s", stock.TrackHex, stock.FillHex, pal.StockTrack, pal.StockFillFor(it.Quantity))
		}
		bc := pal.Badge(it.Status)
		if status.BadgeText != it.Status.String() || status.BadgeBG != bc.Background || status.BadgeFG != bc.Text || status.Dark != dark {
			t.Errorf("badge %+v, want %s %v dark=%v", status, it.Status, bc, dark)
		}
		f, hf, e := core.Stars(it.Rating)
		if rating.Full != f || rating.Half != hf || rating.Empty != e || rating.RatingText != core.FormatRating(it.Rating) {
			t.Errorf("stars %d/%d/%d %q, want %d/%d/%d", rating.Full, rating.Half, rating.Empty, rating.RatingText, f, hf, e)
		}
		if rating.StarOn != pal.StarFilled || rating.StarOff != pal.StarEmpty {
			t.Errorf("star colors %s/%s", rating.StarOn, rating.StarOff)
		}
		h.shot(fmt.Sprintf("t10-table-dark-%v", dark))
	}
}

func TestT11Cards(t *testing.T) {
	h := newHarness(t, 0)
	ui(func() { h.a.cardsBtn.Click() })
	h.waitFor("cards loaded", 5*time.Second, func() bool { return h.a.viewCards && h.a.visibleLoaded() })
	it := h.item(1)
	ui(func() {
		h.a.cdel.log = map[int]paintRecord{}
		h.a.cards.Grab()
	})
	var rec paintRecord
	ui(func() { rec = h.a.cdel.log[0] })
	pal := core.Colors(h.dark())
	if rec.Name != it.Name || rec.Price != core.FormatPrice(it.PriceCents) || rec.BadgeText != it.Status.String() || rec.BadgeBG != pal.Badge(it.Status).Background {
		t.Errorf("card 1 = %+v", rec)
	}
	h.shot("t11-cards")
	ui(func() {
		m := h.a.cmodel
		h.a.cards.SelectionModel().SetCurrentIndex(m.Index(1, 0, qt.NewQModelIndex()),
			qt.QItemSelectionModel__ClearAndSelect)
	})
	h.waitFor("item 2 in form", 5*time.Second, func() bool { return h.a.form.item != nil && h.a.form.item.ID == 2 })
	want := h.item(2)
	ui(func() {
		if h.a.form.name.Text() != want.Name {
			t.Errorf("form name %q, want %q", h.a.form.name.Text(), want.Name)
		}
	})
	ui(func() { h.a.cdel.log = map[int]paintRecord{}; h.a.cards.Grab() })
	ui(func() {
		if !h.a.cdel.log[1].Selected {
			t.Error("card 2 was not painted as selected")
		}
	})
	h.shot("t11-cards-selected")
}

func (h *harness) dark() bool {
	var d bool
	ui(func() { d = h.a.isDark() })
	return d
}

func TestT12HostInfo(t *testing.T) {
	h := newHarness(t, 0)
	want := host.Current(appID)
	ui(func() { h.a.showHostPanel() })
	ui(func() {
		p := h.a.hostDialog
		if got := p.values["Source"].Text(); got != string(want.Source) {
			t.Errorf("Source = %q, want %q", got, want.Source)
		}
		if got := p.values["Username"].Text(); got != want.Username {
			t.Errorf("Username = %q, want %q", got, want.Username)
		}
		if got, want := h.a.identity.Text(), want.Username+"@"+want.Hostname; got != want {
			t.Errorf("status bar identity = %q, want %q", got, want)
		}
		if !p.revConf.IsEnabled() || p.revExp.IsEnabled() {
			t.Errorf("reveal buttons: config enabled=%v export enabled=%v", p.revConf.IsEnabled(), p.revExp.IsEnabled())
		}
	})
	h.shot("t12-host")
}

// Extra scenarios beyond T1 to T12.

func TestGoToRowAndViewSwitch(t *testing.T) {
	h := newHarness(t, 0)
	ui(func() { h.a.goEdit.SetText("500"); h.a.goBtn.Click() })
	h.waitFor("row 500 selected at top", 5*time.Second, func() bool {
		return h.a.selectedRow == 499 && h.a.topRow() == 499 && h.a.visibleLoaded() && h.a.form.item != nil && h.a.form.item.ID == 500
	})
	ui(func() { h.a.cardsBtn.Click() })
	h.waitFor("card view keeps row at top", 5*time.Second, func() bool {
		return h.a.viewCards && h.a.topRow() == 499 && h.a.selectedRow == 499
	})
	ui(func() { h.a.goEdit.SetText("999999"); h.a.goBtn.Click() })
	h.waitFor("clamped to last row", 5*time.Second, func() bool { return h.a.selectedRow == 999 })
}

func TestFilters(t *testing.T) {
	h := newHarness(t, 0)
	ui(func() {
		h.a.catChecks[0].SetChecked(true)
		h.a.catChecks[2].SetChecked(true)
	})
	want, err := h.ref.Page(context.Background(), core.PageRequest{Query: core.Query{Categories: []core.Category{core.Electronics, core.Kitchen}}, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	h.waitFor("category filter", 5*time.Second, func() bool {
		return strings.HasPrefix(h.a.statusLabel.Text(), core.FormatCount(want.Total)+" items") && h.a.catBtn.Text() == "Categories (2)"
	})
	ui(func() { h.a.statusCombo.SetCurrentIndex(2) })
	h.waitFor("status filter", 5*time.Second, func() bool { return h.a.visibleLoaded() && h.a.query.Status != nil })
	ui(func() { h.a.search.SetText("zzzz-no-such-item") })
	h.waitFor("empty result", 5*time.Second, func() bool {
		return h.a.rows.Total() == 0 && strings.HasPrefix(h.a.statusLabel.Text(), "0 items")
	})
	h.shot("filters-empty")
}

func TestExportCompletes(t *testing.T) {
	h := newHarness(t, -1)
	out := filepath.Join(t.TempDir(), "all.csv")
	h.save.Store(out)
	ui(func() { h.a.exportBtn.Click() })
	h.waitFor("export done", 10*time.Second, func() bool { return !h.a.exporting && h.a.lastExport == out })
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "\n# checksum ") {
		t.Error("export has no checksum trailer")
	}
	if !strings.Contains(string(b), "SKU-0001000") {
		t.Error("export is missing the last row")
	}
	var notice string
	ui(func() { notice = h.a.lastNotice })
	if notice != "Export complete: 1,000 rows written" {
		t.Errorf("notice = %q", notice)
	}
	ui(func() {
		if !h.a.menus.revealExport.IsEnabled() || !h.a.hostDialogRevealReady() {
			t.Error("Reveal Last Export should be enabled after an export")
		}
	})
}

func (a *App) hostDialogRevealReady() bool { return a.lastExport != "" }

func TestImportNotes(t *testing.T) {
	h := newHarness(t, 0)
	h.selectRowUser(0)
	h.waitFor("form loaded", 5*time.Second, func() bool { return h.a.form.item != nil })
	dir := t.TempDir()
	small := filepath.Join(dir, "n.txt")
	os.WriteFile(small, []byte(strings.Repeat("x", 600)), 0o644)
	big := filepath.Join(dir, "big.txt")
	os.WriteFile(big, []byte(strings.Repeat("y", 70000)), 0o644)
	ui(func() {
		h.a.dlg.openFile = func() string { return small }
		h.a.form.importBtn.Click()
	})
	ui(func() {
		if got := len(h.a.form.notes.ToPlainText()); got != 500 {
			t.Errorf("imported %d chars, want 500", got)
		}
		if !h.a.form.isDirty() {
			t.Error("form should be dirty after import")
		}
		h.a.dlg.openFile = func() string { return big }
		h.a.form.importBtn.Click()
	})
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.errs) != 1 || !strings.HasSuffix(h.errs[0], "File is too large.") {
		t.Errorf("errors = %v", h.errs)
	}
}

func TestPrefsRestore(t *testing.T) {
	h := newHarness(t, 0)
	ui(func() {
		h.a.search.SetText("steel")
		h.a.themeCombo.SetCurrentIndex(1)
		h.a.cardsBtn.Click()
	})
	h.waitFor("search applied", 5*time.Second, func() bool { return h.a.query.Search == "steel" && h.a.visibleLoaded() })
	h.selectRowUserCards(1)
	h.waitFor("selected", 5*time.Second, func() bool { return h.a.selectedID != 0 })
	var id int64
	ui(func() { id = h.a.selectedID; h.a.savePrefsNow() })
	p, err := prefs.NewFileStore(appID).Load()
	if err != nil {
		t.Fatal(err)
	}
	if p.Search != "steel" || p.Theme != "light" || p.View != "cards" || p.SelectedID != id {
		t.Errorf("saved prefs = %+v", p)
	}
	// A second app instance restores them.
	var b *App
	ui(func() {
		b = newApp(options{svc: h.svc, store: prefs.NewFileStore(appID)})
		b.dlg.confirmDiscard = func() bool { return true }
		b.win.Show()
		b.start()
	})
	defer ui(func() { b.cancelAll(); b.win.Close() })
	deadline := time.Now().Add(5 * time.Second)
	for {
		ok := false
		ui(func() { ok = b.selectedID == id && b.viewCards && b.theme == "light" && b.search.Text() == "steel" })
		if ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("second instance did not restore prefs")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (h *harness) selectRowUserCards(row int) {
	ui(func() {
		h.a.cards.SelectionModel().SetCurrentIndex(h.a.cmodel.Index(row, 0, qt.NewQModelIndex()), qt.QItemSelectionModel__ClearAndSelect)
	})
}

func TestAboutLines(t *testing.T) {
	track(t)
	var lines []string
	ui(func() { lines = aboutLines() })
	if len(lines) != 4 || lines[0] != "GGUIX - MIQT" || lines[2] != runtime.Version() || lines[3] != "BuildMarker "+BuildMarker {
		t.Errorf("about = %q", lines)
	}
	if !strings.HasPrefix(lines[1], "MIQT v0.15.0") {
		t.Errorf("framework line = %q", lines[1])
	}
}

var _ = io.Discard
