//go:build !js

package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gen2brain/iup-go/iup"

	"gguix/core"
	"gguix/core/host"
	"gguix/core/memory"
	"gguix/core/prefs"
)

// IUP has no headless mode. The tests open the real toolkit on the main
// thread, build the real window (mapped, never shown), and drive the app's
// presentation layer: the same callbacks the controls call, with the
// widgets' own values read back through IUP attributes. Test goroutines
// reach the UI thread with IupPostMessage.

var (
	ta       *app
	tdir     string
	tUpdates atomic.Int32
	tConfirm atomic.Int32
	tSvc     core.DataService
	tMu      sync.Mutex
)

type countSvc struct{ core.DataService }

func (s countSvc) Update(ctx context.Context, it core.Item) (core.Item, error) {
	tUpdates.Add(1)
	return s.DataService.Update(ctx, it)
}

func init() { runtime.LockOSThread() }

func TestMain(m *testing.M) {
	tdir, _ = os.MkdirTemp("", "iupgo-test")
	os.Setenv("GGUIX_PREFS_DIR", tdir)
	iup.Open()
	opts := memory.Options{Rows: 1000, Latency: 0, ExportRowsPerSec: 200}
	if os.Getenv("GGUIX_PROF") != "" {
		opts = memory.DefaultOptions()
	}
	tSvc = countSvc{memory.New(opts)}
	ta = newApp(tSvc, prefs.NewFileStoreAt(tdir), host.Current(appID), false)
	ta.plat = platformHooks{
		HostActions: true,
		Notify:      func(string, string, func()) bool { return true },
		Confirm: func(_, _, _, _ string, ok, cancel func()) {
			tConfirm.Add(1)
			cancel()
		},
		Alert:     func(string) {},
		Clipboard: func(string) {},
		Quit:      func() {},
	}
	dlg := ta.buildUI()
	iup.Map(dlg)
	ta.uiInit()
	code := make(chan int, 1)
	go func() {
		c := m.Run()
		ta.post(func() { iup.ExitLoop() })
		code <- c
	}()
	iup.MainLoop()
	c := <-code
	ta.pager.Close()
	os.RemoveAll(tdir)
	os.Exit(c)
}

// onUI runs f on the UI thread and waits for it.
func onUI(f func()) {
	done := make(chan struct{})
	ta.post(func() { f(); close(done) })
	<-done
}

func until(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for i := 0; i < 600; i++ {
		ok := false
		onUI(func() { ok = cond() })
		if ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// reset returns the app to a clean state for the next test.
func reset(t *testing.T) {
	t.Helper()
	onUI(func() {
		a := ta
		a.f.cur = a.f.snap
		a.search, a.statusSel, a.sortField, a.sortDesc = "", statusChoices[0], core.SortID, false
		a.cats = [8]bool{}
		a.selRow, a.selID = -1, 0
		a.f.clear()
		a.setViewFromUI(0)
		a.rebuildQuery(true)
		a.setVisible(0, 24)
	})
	until(t, "first page", func() bool { return ta.pager.Total() > 0 && ta.pager.RangeLoaded(0, 24) && !ta.pending.Load() })
}

func cell(row, col int) string { return ta.tableCellText(row, col) }

func TestT1Start(t *testing.T) {
	reset(t)
	onUI(func() {
		for r := 0; r < 3; r++ {
			if got := cell(r, colID); got != string(rune('1'+r)) {
				t.Errorf("row %d id = %q", r, got)
			}
			want := "SKU-000000" + string(rune('1'+r))
			if got := cell(r, colSKU); got != want {
				t.Errorf("row %d sku = %q want %q", r, got, want)
			}
		}
	})
}

func TestT2Sort(t *testing.T) {
	reset(t)
	var want string
	pg, _ := tSvc.Page(context.Background(), core.PageRequest{Query: core.Query{SortBy: core.SortName}, Limit: 1})
	want = pg.Items[0].Name
	onUI(func() { ta.tableSortClick(colName) })
	until(t, "sorted", func() bool { return ta.pager.RangeLoaded(0, 24) && !ta.pending.Load() })
	onUI(func() {
		if got := cell(0, colName); got != want {
			t.Errorf("first name %q want %q", got, want)
		}
		if h := ta.tableHeaderTitle(colName); !strings.HasSuffix(h, "▲") {
			t.Errorf("header %q", h)
		}
	})
}

func TestT3Search(t *testing.T) {
	reset(t)
	pg, _ := tSvc.Page(context.Background(), core.PageRequest{Query: core.Query{Search: "steel"}, Limit: 1})
	onUI(func() { ta.search = "steel"; ta.applySearchNow() })
	until(t, "search", func() bool { return ta.pager.Total() == pg.Total && !ta.pending.Load() })
	onUI(func() {
		if s := iup.GetAttribute(ta.ui.statusLbl, "TITLE"); !strings.HasPrefix(s, core.FormatCount(pg.Total)+" items") {
			t.Errorf("status %q", s)
		}
	})
}

func selectAndWait(t *testing.T, row int) {
	onUI(func() { ta.requestSelect(row) })
	until(t, "form", func() bool { return ta.f.has && ta.f.item.ID == int64(row+1) })
}

func TestT4Selection(t *testing.T) {
	reset(t)
	selectAndWait(t, 1)
	onUI(func() {
		f := &ta.ui.frm
		it := ta.f.item
		eq := func(name, got, want string) {
			if got != want {
				t.Errorf("%s = %q want %q", name, got, want)
			}
		}
		eq("id", iup.GetAttribute(f.id, "VALUE"), "2")
		eq("sku", iup.GetAttribute(f.sku, "VALUE"), it.SKU)
		eq("name", iup.GetAttribute(f.name, "VALUE"), it.Name)
		eq("price", iup.GetAttribute(f.price, "VALUE"), core.FormatPrice(it.PriceCents))
		eq("qty", iup.GetAttribute(f.qty, "VALUE"), core.StockLabel(it.Quantity)[6:])
		eq("category", iup.GetAttribute(f.cat, "VALUE"), string(rune('1'+int(it.Category))))
		eq("featured", iup.GetAttribute(f.featured, "VALUE"), map[bool]string{true: "ON", false: "OFF"}[it.Featured])
		eq("status", iup.GetAttribute(f.status[it.Status], "VALUE"), "ON")
		eq("updated", iup.GetAttribute(f.upd, "VALUE"), core.FormatUpdated(it.UpdatedAt))
		eq("notes", iup.GetAttribute(f.notes, "VALUE"), it.Notes)
		eq("rating", iup.GetAttribute(f.ratingLbl, "TITLE"), core.FormatRating(it.Rating))
		// Selecting an item must not clear the Table/Cards selector (the
		// form's status radio shares its native parent on Cocoa).
		eq("table toggle", iup.GetAttribute(ta.ui.tglTable, "VALUE"), "ON")
		eq("cards toggle", iup.GetAttribute(ta.ui.tglCards, "VALUE"), "OFF")
		// The empty-state hint is hidden but stays in the layout, so it
		// can never come back clipped.
		eq("hint visible", iup.GetAttribute(f.hint, "VISIBLE"), "NO")
		eq("hint floating", iup.GetAttribute(f.hint, "FLOATING"), "NO")
	})
}

func typeName(s string) {
	set(ta.ui.frm.name, "VALUE", s)
	ta.formEdited()
}

func TestT5Validation(t *testing.T) {
	reset(t)
	selectAndWait(t, 0)
	before := tUpdates.Load()
	onUI(func() {
		typeName("ab")
		if iup.GetAttribute(ta.ui.frm.revert, "ACTIVE") != "YES" {
			t.Errorf("revert should be active while dirty")
		}
		ta.save()
		if got := iup.GetAttribute(ta.ui.frm.errLbl["name"], "TITLE"); got != "Name must be 3 to 80 characters." {
			t.Errorf("error text %q", got)
		}
		if !ta.f.dirty() {
			t.Errorf("form should stay dirty")
		}
	})
	if tUpdates.Load() != before {
		t.Errorf("Update was called")
	}
	onUI(func() { ta.f.cur = ta.f.snap; ta.f.errs = nil; ta.uiForm() })
}

func TestT6Save(t *testing.T) {
	reset(t)
	selectAndWait(t, 0)
	onUI(func() { typeName("Renamed Lamp"); ta.save() })
	until(t, "saved row", func() bool { return cell(0, colName) == "Renamed Lamp" && !ta.pending.Load() })
	onUI(func() {
		if ta.f.dirty() || iup.GetAttribute(ta.ui.frm.name, "VALUE") != "Renamed Lamp" {
			t.Errorf("form not reloaded")
		}
	})
}

func TestT7ThemePrefs(t *testing.T) {
	reset(t)
	onUI(func() { ta.setTheme("Dark"); ta.uiSyncThemeControls(); ta.flushPrefs() })
	onUI(func() {
		if !ta.dark || ta.pal.StockTrack != core.Colors(true).StockTrack {
			t.Errorf("dark palette not applied")
		}
	})
	// The toolkit's own DARKMODE is only logged: on GTK3 it follows the theme
	// colors, which stay light in the container unless the GTK theme starts dark.
	time.Sleep(300 * time.Millisecond)
	onUI(func() { t.Logf("IUP DARKMODE after APPEARANCE=DARK: %s", iup.GetGlobal("DARKMODE")) })
	b, err := os.ReadFile(filepath.Join(tdir, "prefs.json"))
	if err != nil || !strings.Contains(strings.ReplaceAll(string(b), " ", ""), `"theme":"dark"`) {
		t.Errorf("prefs %q %v", b, err)
	}
	onUI(func() { ta.setTheme("Light") })
}

func TestT8ExportCancel(t *testing.T) {
	reset(t)
	path := filepath.Join(tdir, "x.csv")
	onUI(func() { ta.runExport(path, nil) })
	until(t, "running", func() bool { return ta.exp.running && iup.GetAttribute(ta.ui.exportBtn, "ACTIVE") == "NO" })
	time.Sleep(200 * time.Millisecond)
	onUI(func() { ta.cancelExport() })
	until(t, "stopped", func() bool { return !ta.exp.running })
	onUI(func() {
		if _, err := os.Stat(path); err == nil {
			t.Errorf("partial file still exists")
		}
		if iup.GetAttribute(ta.ui.exportBtn, "ACTIVE") != "YES" {
			t.Errorf("export button not re-enabled")
		}
	})
}

func TestT9DirtyGuard(t *testing.T) {
	reset(t)
	selectAndWait(t, 0)
	n := tConfirm.Load()
	onUI(func() { typeName("Changed Name"); ta.requestSelect(3) })
	if tConfirm.Load() != n+1 {
		t.Errorf("confirm dialog not shown")
	}
	onUI(func() {
		if ta.selRow != 0 || ta.f.item.ID != 1 {
			t.Errorf("selection changed despite cancel")
		}
		ta.f.cur = ta.f.snap
		ta.uiForm()
	})
}

func TestT10CustomCells(t *testing.T) {
	reset(t)
	onUI(func() {
		it, _ := ta.pager.Row(0)
		p := core.Colors(ta.dark)
		bar := ta.cellSpecFor(it, colStock)
		if bar.Fraction != core.StockFraction(it.Quantity) || bar.Fill != p.StockFillFor(it.Quantity) || bar.Track != p.StockTrack {
			t.Errorf("bar spec %+v", bar)
		}
		st := ta.cellSpecFor(it, colStatus)
		b := p.Badge(it.Status)
		if st.Text != it.Status.String() || st.BadgeBG != b.Background || st.BadgeFG != b.Text {
			t.Errorf("badge spec %+v", st)
		}
		r := ta.cellSpecFor(it, colRating)
		f, h, e := core.Stars(it.Rating)
		if r.Full != f || r.Half != h || r.Empty != e || r.StarFilled != p.StarFilled {
			t.Errorf("stars %+v", r)
		}
		nativeCellChecks(t, it)
	})
}

func TestT11Cards(t *testing.T) {
	reset(t)
	onUI(func() { ta.setViewFromUI(1); ta.cardsRange() })
	until(t, "cards", func() bool { return ta.viewMode == 1 && ta.pager.RangeLoaded(0, 2) })
	onUI(func() {
		it, _ := ta.pager.Row(0)
		sp := ta.cardSpecFor(0)
		b := ta.pal.Badge(it.Status)
		if sp.Placeholder || sp.Name != it.Name || sp.Price != core.FormatPrice(it.PriceCents) ||
			sp.Status != it.Status.String() || sp.BadgeBG != b.Background {
			t.Errorf("card spec %+v", sp)
		}
		row, ok := ta.cardAt(cardStep + 10)
		if !ok || row != 1 {
			t.Fatalf("cardAt = %d %v", row, ok)
		}
		ta.requestSelect(row)
	})
	until(t, "form", func() bool { return ta.f.has && ta.f.item.ID == 2 })
	onUI(func() {
		if iup.GetAttribute(ta.ui.frm.id, "VALUE") != "2" {
			t.Errorf("form not filled")
		}
		ta.setViewFromUI(0)
	})
}

func TestT12Host(t *testing.T) {
	reset(t)
	onUI(func() {
		rows := ta.hostRows()
		want := host.Current(appID)
		if rows[0] != [2]string{"Source", string(want.Source)} || want.Source != host.SourceNative {
			t.Errorf("source row %v", rows[0])
		}
		if rows[1] != [2]string{"Username", want.Username} {
			t.Errorf("username row %v", rows[1])
		}
		if got := iup.GetAttribute(ta.ui.idLbl, "TITLE"); got != want.Username+"@"+want.Hostname {
			t.Errorf("status identity %q", got)
		}
	})
}
