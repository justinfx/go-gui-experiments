package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"gguix/core"
	"gguix/core/host"
	"gguix/core/memory"
)

// scenarios are the brief's section 8 tests, T1 to T12. Each runs against a
// fresh app with 1000 rows and no latency.
var scenarios = []struct {
	id  string
	run func(t *testing.T)
}{
	{"T1_start_and_load", testStart},
	{"T2_sort", testSort},
	{"T3_search", testSearch},
	{"T4_selection", testSelection},
	{"T5_validation", testValidation},
	{"T6_save", testSave},
	{"T7_theme_and_prefs", testThemePrefs},
	{"T8_export_cancel", testExportCancel},
	{"T9_dirty_guard", testDirtyGuard},
	{"T10_custom_cells", testCustomCells},
	{"T11_cards_view", testCards},
	{"T12_host_info", testHostInfo},
}

// TestGGUIX runs the scenarios and prints the summary line that make test
// repeats as its last line.
func TestGGUIX(t *testing.T) {
	passed, failed, notFeasible := 0, 0, 0
	for _, s := range scenarios {
		ok := t.Run(s.id, s.run)
		switch {
		case t.Failed() && !ok:
			failed++
		case ok:
			passed++
		}
	}
	for _, id := range notFeasibleScenarios() {
		notFeasible++
		fmt.Println("NOT FEASIBLE:", id)
	}
	fmt.Printf("GGUIX_TESTS passed=%d failed=%d not_feasible=%d\n", passed, failed, notFeasible)
}

func newDefaultHarness(t *testing.T) *harness {
	return newHarness(t, memory.Options{Rows: 1000, Latency: 0})
}

func testStart(t *testing.T) {
	h := newDefaultHarness(t)
	h.loaded()
	cells := h.tableCells()
	for row, id := range []int64{1, 2, 3} {
		if got := cells[row][colID].Text(); got != strconv.FormatInt(id, 10) {
			t.Errorf("row %d ID = %q, want %d", row, got, id)
		}
		if got, want := cells[row][colSKU].Text(), core.SKUFor(id); got != want {
			t.Errorf("row %d SKU = %q, want %q", row, got, want)
		}
	}
}

func testSort(t *testing.T) {
	h := newDefaultHarness(t)
	h.loaded()
	test.Tap(h.header("Name"))
	h.wait("sorted rows", func() bool {
		return h.u.sortBy == core.SortName && h.u.VisibleLoaded()
	})
	want, err := h.mem.Page(context.Background(), core.PageRequest{Query: core.Query{SortBy: core.SortName}, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	got := h.tableCells()[0][colName].Text()
	if got != want.Items[0].Name {
		t.Errorf("first row name = %q, want %q", got, want.Items[0].Name)
	}
	if h.u.sortDesc {
		t.Error("first activation must sort ascending")
	}
	if !strings.HasPrefix(h.header("Name").Text, "Name ▲") {
		t.Errorf("header = %q, want ascending marker", h.header("Name").Text)
	}
	test.Tap(h.header("Name"))
	h.wait("descending", func() bool { return h.u.sortDesc && h.u.VisibleLoaded() })
	if !strings.HasSuffix(h.header("Name").Text, "▼") {
		t.Errorf("header = %q, want descending marker", h.header("Name").Text)
	}
}

func testSearch(t *testing.T) {
	h := newDefaultHarness(t)
	h.loaded()
	h.u.searchEntry.SetText("steel")
	want, err := h.mem.Page(context.Background(), core.PageRequest{Query: core.Query{Search: "steel"}, Limit: 1})
	if err != nil || want.Total == 0 {
		t.Fatalf("expected results for steel: %v %d", err, want.Total)
	}
	prefix := core.FormatCount(want.Total) + " items"
	h.wait("search result count", func() bool {
		return strings.HasPrefix(h.u.StatusText(), prefix) && h.u.VisibleLoaded()
	})
}

func selectRow(h *harness, row int) {
	h.t.Helper()
	c := h.tableCells()[row][colName]
	if c == nil {
		h.t.Fatalf("row %d not on screen", row)
	}
	h.tapAt(c, 5, 5)
	h.pump()
}

func testSelection(t *testing.T) {
	h := newDefaultHarness(t)
	h.loaded()
	selectRow(h, 4)
	h.wait("form loaded", func() bool { return h.u.form.has })
	it := h.item(5)
	f := h.u.form
	check := func(name, got, want string) {
		t.Helper()
		if got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	check("ID", f.id.Text, "5")
	check("SKU", f.sku.Text, it.SKU)
	check("Name", f.name.Text, it.Name)
	check("Category", f.category.Selected, it.Category.String())
	check("Price", f.price.Text, core.FormatPrice(it.PriceCents))
	check("Quantity", f.qty.Text, strconv.Itoa(int(it.Quantity)))
	check("Status", f.status.Selected, it.Status.String())
	check("Rating", f.ratingLb.Text, core.FormatRating(it.Rating))
	if f.rating.Value < float64(it.Rating)-0.001 || f.rating.Value > float64(it.Rating)+0.001 {
		t.Errorf("rating slider = %v, want %v", f.rating.Value, it.Rating)
	}
	if f.featured.Checked != it.Featured {
		t.Errorf("featured = %v, want %v", f.featured.Checked, it.Featured)
	}
	if it.RestockDate == nil {
		if f.restock.Date != nil {
			t.Errorf("restock date = %v, want none", f.restock.Date)
		}
	} else if f.restock.Date == nil || core.DateOf(*f.restock.Date) != *it.RestockDate {
		t.Errorf("restock date = %v, want %v", f.restock.Date, it.RestockDate)
	}
	check("Updated", f.updated.Text, core.FormatUpdated(it.UpdatedAt))
	check("Notes", f.notes.Text, it.Notes)
	if f.dirty() || !f.save.Disabled() {
		t.Error("a freshly loaded form must be clean with Save disabled")
	}
}

func testValidation(t *testing.T) {
	h := newDefaultHarness(t)
	h.loaded()
	selectRow(h, 0)
	h.u.form.name.SetText("ab")
	if h.u.form.save.Disabled() {
		t.Fatal("Save must be enabled while dirty")
	}
	test.Tap(h.u.form.save)
	h.pump()
	if got := h.u.form.Errors()["name"]; got != core.MsgName {
		t.Errorf("name error = %q, want %q", got, core.MsgName)
	}
	if n := h.svc.updates.Load(); n != 0 {
		t.Errorf("Update called %d times, want 0", n)
	}
	if !h.u.form.dirty() {
		t.Error("form must stay dirty")
	}
}

func testSave(t *testing.T) {
	h := newDefaultHarness(t)
	h.loaded()
	selectRow(h, 0)
	h.u.form.name.SetText("Renamed Test Lamp")
	test.Tap(h.u.form.save)
	h.wait("update and refresh", func() bool {
		c := h.tableCells()[0][colName]
		return h.svc.updates.Load() == 1 && c != nil && c.Text() == "Renamed Test Lamp"
	})
	if h.u.form.dirty() {
		t.Error("form must be clean after save")
	}
	if len(h.u.form.Errors()) != 0 {
		t.Errorf("errors after save: %v", h.u.form.Errors())
	}
}

func testThemePrefs(t *testing.T) {
	h := newDefaultHarness(t)
	h.loaded()
	h.u.themeSel.SetSelected("Dark")
	h.pump()
	if !isDark() {
		t.Error("dark theme not in effect")
	}
	if got := fyne.CurrentApp().Settings().Theme().Color("background", 0); got == nil {
		t.Error("no background color")
	}
	path := filepath.Join(h.dir, "prefs.json")
	re := regexp.MustCompile(`"theme":\s*"dark"`)
	h.wait("saved prefs", func() bool {
		b, err := os.ReadFile(path)
		return err == nil && re.Match(b)
	})
	h.u.themeSel.SetSelected("Light")
	h.pump()
	if isDark() {
		t.Error("light theme not in effect")
	}
}

func testDirtyGuard(t *testing.T) {
	h := newDefaultHarness(t)
	h.loaded()
	selectRow(h, 0)
	h.u.form.name.SetText("Changed Name")
	selectRow(h, 1)
	found := false
	for _, l := range h.labels(h.overlay()) {
		if l == "Discard unsaved changes?" {
			found = true
		}
	}
	if !found {
		t.Fatalf("no confirm dialog, overlay labels: %v", h.labels(h.overlay()))
	}
	if h.u.selID != 1 || h.u.form.name.Text != "Changed Name" {
		t.Error("selection or edits changed before the user confirmed")
	}
	var cancel, discard *widget.Button
	for _, b := range collect[*widget.Button](h.overlay()) {
		switch b.Text {
		case "Cancel":
			cancel = b
		case "Discard":
			discard = b
		}
	}
	if cancel == nil || discard == nil {
		t.Fatal("dialog must offer Discard and Cancel")
	}
	test.Tap(cancel)
	h.pump()
	if h.u.selID != 1 || h.u.form.name.Text != "Changed Name" {
		t.Error("Cancel must keep the selection and the edits")
	}
	selectRow(h, 1)
	for _, b := range collect[*widget.Button](h.overlay()) {
		if b.Text == "Discard" {
			test.Tap(b)
		}
	}
	h.pump()
	if h.u.selID != 2 || h.u.form.name.Text != h.item(2).Name {
		t.Errorf("Discard must select the new row, selID=%d", h.u.selID)
	}
}

func testCustomCells(t *testing.T) {
	h := newDefaultHarness(t)
	h.loaded()
	it := h.item(1)
	mk := func(col int) *tableCell {
		c := newTableCell()
		c.Set(0, col, &it, false)
		h.canvas().Content() // keep canvas alive
		c.Resize(fyne.NewSize(columns[col].width, labelMin().Height))
		return c
	}
	cells := map[int]map[int]*tableCell{0: {colStock: mk(colStock), colStatus: mk(colStatus), colRating: mk(colRating)}}
	dark := isDark()
	pal := core.Colors(dark)

	stock := test.WidgetRenderer(cells[0][colStock]).(*tableCellRenderer)
	if got, want := stock.bar.Fraction(), core.StockFraction(it.Quantity); got != want {
		t.Errorf("stock fraction = %v, want %v", got, want)
	}
	br := test.WidgetRenderer(stock.bar).(*stockBarRenderer)
	if got, want := float64(br.fill.Size().Width), core.StockFraction(it.Quantity)*stockBarW; got < want-0.5 || got > want+0.5 {
		t.Errorf("stock fill width = %v, want %v", got, want)
	}
	if got, want := core.HexColor(pal.StockFillFor(it.Quantity)), br.fill.FillColor; got != want {
		t.Errorf("stock fill color = %v, want %v", want, got)
	}
	if got, want := br.track.FillColor, core.HexColor(pal.StockTrack); got != want {
		t.Errorf("stock track color = %v, want %v", got, want)
	}

	status := test.WidgetRenderer(cells[0][colStatus]).(*tableCellRenderer)
	if status.badge.Text() != it.Status.String() {
		t.Errorf("badge text = %q, want %q", status.badge.Text(), it.Status)
	}
	bc := pal.Badge(it.Status)
	pr := test.WidgetRenderer(status.badge).(*pillRenderer)
	if pr.bg.FillColor != core.HexColor(bc.Background) || pr.txt.Color != core.HexColor(bc.Text) {
		t.Errorf("badge colors = %v/%v, want %v/%v", pr.bg.FillColor, pr.txt.Color, bc.Background, bc.Text)
	}

	rating := test.WidgetRenderer(cells[0][colRating]).(*tableCellRenderer)
	f, hf, e := rating.stars.Counts()
	wf, wh, we := core.Stars(it.Rating)
	if f != wf || hf != wh || e != we {
		t.Errorf("stars = %d/%d/%d, want %d/%d/%d", f, hf, e, wf, wh, we)
	}
	sr := test.WidgetRenderer(rating.stars).(*starRowRenderer)
	filled := 0
	for _, s := range sr.stars {
		if s.FillColor == core.HexColor(pal.StarFilled) {
			filled++
		}
	}
	if filled != wf {
		t.Errorf("filled star shapes = %d, want %d", filled, wf)
	}
	if sr.halfStar.Hidden != (wh == 0) {
		t.Error("half star visibility does not match Stars()")
	}

	// the colors follow the theme
	h.u.setThemeMode(core.ThemeLight)
	h.pump()
	h.u.setThemeMode(core.ThemeDark)
	h.pump()
	pal = core.Colors(true)
	bc = pal.Badge(it.Status)
	status = test.WidgetRenderer(mk(colStatus)).(*tableCellRenderer)
	pr = test.WidgetRenderer(status.badge).(*pillRenderer)
	if pr.bg.FillColor != core.HexColor(bc.Background) {
		t.Errorf("dark badge background = %v, want %v", pr.bg.FillColor, bc.Background)
	}
}

func testCards(t *testing.T) {
	h := newDefaultHarness(t)
	h.loaded()
	test.Tap(h.u.cardsBtn)
	h.wait("cards loaded", func() bool { return h.u.cards && h.u.VisibleLoaded() })
	cards := collect[*cardItem](h.objects())
	var first, second *cardItem
	for _, c := range cards {
		if c.loaded && c.item.ID == 1 {
			first = c
		}
		if c.loaded && c.item.ID == 2 {
			second = c
		}
	}
	if first == nil || second == nil {
		t.Fatalf("cards for items 1 and 2 not found (%d cards)", len(cards))
	}
	it := h.item(1)
	r := test.WidgetRenderer(first).(*cardRenderer)
	if r.name.Text != it.Name {
		t.Errorf("card name = %q, want %q", r.name.Text, it.Name)
	}
	if r.price.Text != core.FormatPrice(it.PriceCents) {
		t.Errorf("card price = %q, want %q", r.price.Text, core.FormatPrice(it.PriceCents))
	}
	if r.badge.Text() != it.Status.String() {
		t.Errorf("card badge = %q, want %q", r.badge.Text(), it.Status)
	}
	h.tapAt(second, 40, 40)
	h.wait("card selection", func() bool { return h.u.selID == 2 && h.u.form.has })
	if got := h.u.form.name.Text; got != h.item(2).Name {
		t.Errorf("form name = %q, want %q", got, h.item(2).Name)
	}
	if !test.WidgetRenderer(second).(*cardRenderer).c.selected {
		t.Error("selected card must show the selection border")
	}
}

func testHostInfo(t *testing.T) {
	h := newDefaultHarness(t)
	h.loaded()
	want := expectedHost()
	h.wait("identity", func() bool { return h.u.identityLbl.Text == want.Identity() })
	test.Tap(h.u.hostBtn)
	h.pump()
	labels := h.labels(h.overlay())
	value := func(label string) string {
		for i, l := range labels {
			if l == label && i+1 < len(labels) {
				return labels[i+1]
			}
		}
		return "<missing>"
	}
	if got := value("Source"); got != string(want.Source) {
		t.Errorf("Source = %q, want %q", got, want.Source)
	}
	if got := value("Username"); got != orUnavailable(want.Username) {
		t.Errorf("Username = %q, want %q", got, want.Username)
	}
	if h.u.identityLbl.Text != want.Identity() {
		t.Errorf("status bar identity = %q, want %q", h.u.identityLbl.Text, want.Identity())
	}
	revealCfg := false
	for _, b := range collect[*widget.Button](h.overlay()) {
		if b.Text == "Reveal Config Folder" {
			revealCfg = !b.Disabled()
		}
	}
	if revealCfg != (want.Source == host.SourceNative) {
		t.Errorf("Reveal Config Folder enabled = %v for source %s", revealCfg, want.Source)
	}
}

func orUnavailable(s string) string {
	if s == "" {
		return "(unavailable)"
	}
	return s
}
