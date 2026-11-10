package core_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gguix/core"
)

func TestFormatting(t *testing.T) {
	cases := []struct{ got, want string }{
		{core.FormatPrice(123456), "$1,234.56"},
		{core.FormatPrice(5), "$0.05"},
		{core.FormatPrice(10_000_000), "$100,000.00"},
		{core.FormatRating(4.3), "4.3"},
		{core.FormatRating(5), "5.0"},
		{core.FormatCount(1_000_000), "1,000,000"},
		{core.FormatCount(999), "999"},
		{core.FormatUpdated(time.Date(2026, 1, 2, 15, 4, 59, 0, time.UTC)), "2026-01-02 15:04"},
		{core.StockLabel(432), "Stock 432"},
		{core.CategoryInitial(core.Electronics), "E"},
		{core.NotesPreview(""), "No notes"},
		{core.NotesPreview("first\nsecond"), "first"},
		{core.NotesPreview(strings.Repeat("a", 95)), strings.Repeat("a", 90) + "…"},
		{core.SKUFor(1), "SKU-0000001"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("got %q want %q", c.got, c.want)
		}
	}
}

func TestParsePrice(t *testing.T) {
	ok := map[string]int64{"1234.56": 123456, "$1,234.56": 123456, "12": 1200, "12.5": 1250, " 0.07 ": 7}
	for in, want := range ok {
		got, err := core.ParsePrice(in)
		if err != nil || got != want {
			t.Errorf("ParsePrice(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"", "abc", "1.234", "-5", "1e3"} {
		if _, err := core.ParsePrice(in); err == nil || err.Error() != "Enter a price like 12.34." {
			t.Errorf("ParsePrice(%q) err = %v", in, err)
		}
	}
}

func TestStarsAndStock(t *testing.T) {
	for r, want := range map[float32][3]int{4.3: {4, 1, 0}, 4.2: {4, 0, 1}, 0: {0, 0, 5}, 5: {5, 0, 0}, 2.7: {2, 1, 2}} {
		f, h, e := core.Stars(r)
		if [3]int{f, h, e} != want {
			t.Errorf("Stars(%v) = %d %d %d want %v", r, f, h, e, want)
		}
	}
	if core.StockFraction(432) != 0.432 || core.StockFraction(5000) != 1 || core.StockFraction(-1) != 0 {
		t.Error("StockFraction")
	}
	if !core.StockLow(199) || core.StockLow(200) {
		t.Error("StockLow")
	}
	p := core.Colors(true)
	if p.Badge(core.Backordered).Background != "#78350F" || p.StockFillFor(10) != "#F87171" || p.Tile(core.Toys) != "#DB2777" {
		t.Error("dark palette")
	}
	if c := core.HexColor("#2563EB"); c.R != 0x25 || c.G != 0x63 || c.B != 0xEB || c.A != 255 {
		t.Errorf("HexColor %v", c)
	}
}

func valid() core.Item {
	return core.Item{ID: 1, SKU: "SKU-0000001", Name: "Brushed Steel Lantern", Category: core.Garden, PriceCents: 1999, Quantity: 3, Status: core.Active, Rating: 4.3}
}

func TestValidate(t *testing.T) {
	if v := core.ValidateItem(valid()); v != nil {
		t.Fatalf("valid item: %v", v)
	}
	it := valid()
	it.Name = " ab "
	it.PriceCents = 10_000_001
	it.Quantity = 5
	it.Status = core.Discontinued
	it.Rating = 4.35
	it.Notes = strings.Repeat("x", 501)
	v := core.ValidateItem(it)
	want := map[string]string{
		"name":       core.MsgName,
		"priceCents": core.MsgPrice,
		"quantity":   core.MsgDiscontinued,
		"rating":     core.MsgRating,
		"notes":      core.MsgNotes,
	}
	for f, m := range want {
		if v.For(f) != m {
			t.Errorf("field %s: %q want %q", f, v.For(f), m)
		}
	}
	it = valid()
	it.Status = core.Backordered
	if core.ValidateItem(it).For("restockDate") != core.MsgRestock {
		t.Error("restock rule")
	}
}

func TestJSON(t *testing.T) {
	it := valid()
	d := core.Date{Year: 2026, Month: 12, Day: 3}
	it.RestockDate = &d
	b, err := json.Marshal(it)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{`"category":"Garden"`, `"status":"Active"`, `"restockDate":"2026-12-03"`, `"priceCents":1999`, `"rating":4.3`} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %s in %s", want, s)
		}
	}
	var back core.Item
	if err := json.Unmarshal(b, &back); err != nil || back.Category != core.Garden || *back.RestockDate != d {
		t.Fatalf("round trip: %v %+v", err, back)
	}
	pb, _ := json.Marshal(core.Prefs{Theme: "dark"})
	if !strings.Contains(string(pb), `"theme":"dark"`) {
		t.Errorf("prefs json %s", pb)
	}
}

func TestQueryValuesRoundTrip(t *testing.T) {
	q := core.Query{Search: " Steel ", Categories: []core.Category{core.Toys, core.Garden, core.Toys}, Status: core.StatusPtr(core.Backordered), SortBy: core.SortPrice, Desc: true}
	back, err := core.QueryFromValues(q.Values())
	if err != nil {
		t.Fatal(err)
	}
	if back.Key() != q.Key() {
		t.Fatalf("%s != %s", back.Key(), q.Key())
	}
}

func TestMetricLine(t *testing.T) {
	got := core.FormatMetric("jump", map[string]any{"ms": 12.5})
	if got != `GGUIX_METRIC {"event":"jump","ms":12.5}` {
		t.Fatal(got)
	}
}

func TestChecksumMatchesCRC32(t *testing.T) {
	// The pure Go and cgo variants must agree; this value is CRC-32/IEEE("hello").
	if c := core.ChecksumUpdate(0, []byte("hello")); c != 0x3610a686 {
		t.Fatalf("checksum %08x (cgoprobe=%v)", c, core.CgoProbe)
	}
}
