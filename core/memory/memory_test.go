package memory

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"gguix/core"
)

func newSvc(rows int) *Service { return New(Options{Rows: rows, Seed: 1, ExportRowsPerSec: -1}) }

func all(t *testing.T, s *Service) []core.Item {
	t.Helper()
	out := make([]core.Item, 0, s.n)
	for id := int64(1); id <= int64(s.n); id++ {
		out = append(out, s.itemLocked(id))
	}
	return out
}

func page(t *testing.T, s *Service, q core.Query, off, lim int) core.Page {
	t.Helper()
	p, err := s.Page(context.Background(), core.PageRequest{Query: q, Offset: off, Limit: lim})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestGeneratedItemsValidAndDeterministic(t *testing.T) {
	a, b := newSvc(5000), newSvc(5000)
	steel := 0
	for _, it := range all(t, a) {
		if v := core.ValidateItem(it); v != nil {
			t.Fatalf("item %d invalid: %v", it.ID, v)
		}
		if strings.Contains(it.Name, "Steel") {
			steel++
		}
	}
	if steel == 0 {
		t.Fatal("no steel items")
	}
	if a.itemLocked(77) != b.itemLocked(77) {
		t.Fatal("not deterministic")
	}
	if it := a.itemLocked(1); it.SKU != "SKU-0000001" {
		t.Fatal(it.SKU)
	}
}

func TestTupleOrderEqualsNameOrder(t *testing.T) {
	s := newSvc(3000)
	for i := 1; i < s.n; i++ {
		tc := cmpSign(int(s.name[i-1]) - int(s.name[i]))
		sc := strings.Compare(s.baseName(i-1), s.baseName(i))
		if tc != sc {
			t.Fatalf("rows %d,%d: tuple %d string %d (%q vs %q)", i, i+1, tc, sc, s.baseName(i-1), s.baseName(i))
		}
	}
}

func cmpSign(x int) int {
	switch {
	case x < 0:
		return -1
	case x > 0:
		return 1
	}
	return 0
}

// brute computes the reference result for q.
func brute(items []core.Item, q core.Query) []int64 {
	q = q.Normalize()
	var out []core.Item
	for _, it := range items {
		if q.Matches(it) {
			out = append(out, it)
		}
	}
	slices.SortFunc(out, func(a, b core.Item) int { return core.CompareItems(a, b, q.SortBy, q.Desc) })
	ids := make([]int64, len(out))
	for i, it := range out {
		ids[i] = it.ID
	}
	return ids
}

func ids(p core.Page) []int64 {
	out := make([]int64, len(p.Items))
	for i, it := range p.Items {
		out[i] = it.ID
	}
	return out
}

func checkQueries(t *testing.T, s *Service) {
	t.Helper()
	items := all(t, s)
	st := core.Backordered
	queries := []core.Query{{}, {Desc: true}, {Search: "steel"}, {Search: "SKU-00001"}, {Categories: []core.Category{core.Toys, core.Office}},
		{Status: &st, SortBy: core.SortPrice}, {Search: "ed st", SortBy: core.SortName, Desc: true}}
	for _, f := range core.AllSortFields {
		queries = append(queries, core.Query{SortBy: f}, core.Query{SortBy: f, Desc: true}, core.Query{SortBy: f, Search: "oak"})
	}
	for _, q := range queries {
		want := brute(items, q)
		var got []int64
		for off := 0; ; off += 500 {
			p := page(t, s, q, off, 500)
			if p.Total != len(want) {
				t.Fatalf("%s: total %d want %d", q.Key(), p.Total, len(want))
			}
			got = append(got, ids(p)...)
			if len(p.Items) < 500 {
				break
			}
		}
		if !slices.Equal(got, want) {
			t.Fatalf("%s: order mismatch", q.Key())
		}
	}
}

func TestQueriesMatchReference(t *testing.T) {
	s := newSvc(2000)
	checkQueries(t, s)
	// Edits must move items in cached indexes and filters.
	ctx := context.Background()
	for _, id := range []int64{5, 77, 1999} {
		it, _ := s.Get(ctx, id)
		it.Name = "Aardvark Steel " + it.Name
		it.PriceCents = 1
		it.Rating = 0.1
		it.Featured = !it.Featured
		if _, err := s.Update(ctx, it); err != nil {
			t.Fatal(err)
		}
	}
	checkQueries(t, s)
	if p := page(t, s, core.Query{SortBy: core.SortName}, 0, 1); !strings.HasPrefix(p.Items[0].Name, "Aardvark") {
		t.Fatalf("edited name not first: %q", p.Items[0].Name)
	}
}

func TestUpdateValidation(t *testing.T) {
	s := newSvc(10)
	ctx := context.Background()
	it, _ := s.Get(ctx, 3)
	it.Name = "ab"
	_, err := s.Update(ctx, it)
	if v := core.AsValidationError(err); v == nil || v.For("name") != core.MsgName {
		t.Fatalf("err %v", err)
	}
	if _, err := s.Get(ctx, 11); !errors.Is(err, core.ErrNotFound) {
		t.Fatal(err)
	}
	it, _ = s.Get(ctx, 3)
	it.SKU = "hacked"
	out, err := s.Update(ctx, it)
	if err != nil || out.SKU != "SKU-0000003" || out.UpdatedAt.IsZero() {
		t.Fatalf("%v %+v", err, out)
	}
}

func TestPageBounds(t *testing.T) {
	s := newSvc(10)
	if p := page(t, s, core.Query{}, 8, 5); len(p.Items) != 2 || p.Total != 10 {
		t.Fatal(p)
	}
	if p := page(t, s, core.Query{}, 50, 5); len(p.Items) != 0 || p.Items == nil {
		t.Fatal("expected empty non-nil items")
	}
	if _, err := s.Page(context.Background(), core.PageRequest{Limit: 501}); err == nil {
		t.Fatal("limit 501 accepted")
	}
}

func TestExport(t *testing.T) {
	s := newSvc(300)
	var buf bytes.Buffer
	var last [2]int
	n, err := s.Export(context.Background(), core.Query{Search: "oak"}, &buf, func(d, tot int) { last = [2]int{d, tot} })
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if lines[0] != core.CSVHeaderLine() || !strings.HasPrefix(lines[len(lines)-1], "# checksum ") || len(lines) != n+2 {
		t.Fatalf("bad csv: n=%d lines=%d", n, len(lines))
	}
	if last != [2]int{n, n} {
		t.Fatal(last)
	}
}

func TestExportCancel(t *testing.T) {
	s := New(Options{Rows: 1000, ExportRowsPerSec: 200})
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	start := time.Now()
	n, err := s.Export(ctx, core.Query{}, &bytes.Buffer{}, nil)
	if !errors.Is(err, context.Canceled) || n >= 1000 || time.Since(start) > 400*time.Millisecond {
		t.Fatalf("n=%d err=%v after %v", n, err, time.Since(start))
	}
}

func TestLatency(t *testing.T) {
	s := New(Options{Rows: 10, Latency: 30 * time.Millisecond})
	start := time.Now()
	page(t, s, core.Query{}, 0, 5)
	if d := time.Since(start); d < 30*time.Millisecond {
		t.Fatalf("latency %v", d)
	}
}

func TestFindRow(t *testing.T) {
	s := newSvc(1000)
	q := core.Query{SortBy: core.SortName}
	p := page(t, s, q, 450, 1)
	row, ok, err := core.FindRow(context.Background(), s, q, p.Items[0].ID, 2000)
	if err != nil || !ok || row != 450 {
		t.Fatalf("row=%d ok=%v err=%v", row, ok, err)
	}
	if _, ok, _ := core.FindRow(context.Background(), s, q, p.Items[0].ID, 400); ok {
		t.Fatal("found beyond maxRows")
	}
}

func BenchmarkFirstPageSearchMillion(b *testing.B) {
	s := New(Options{Rows: 1_000_000})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.mu.Lock()
		clear(s.results)
		s.mu.Unlock()
		if _, err := s.Page(context.Background(), core.PageRequest{Query: core.Query{Search: "steel"}, Limit: 200}); err != nil {
			b.Fatal(err)
		}
	}
}

func TestMillionRowTimings(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	start := time.Now()
	s := New(Options{Rows: 1_000_000})
	gen := time.Since(start)
	ctx := context.Background()
	timed := func(q core.Query, off int) time.Duration {
		t0 := time.Now()
		if _, err := s.Page(ctx, core.PageRequest{Query: q, Offset: off, Limit: 200}); err != nil {
			t.Fatal(err)
		}
		return time.Since(t0)
	}
	first := timed(core.Query{SortBy: core.SortName, Desc: true}, 0)
	again := timed(core.Query{SortBy: core.SortName, Desc: true}, 500_000)
	search := timed(core.Query{SortBy: core.SortName, Desc: true, Search: "steel"}, 0)
	plain := timed(core.Query{}, 500_000)
	t.Logf("generate=%v firstSort=%v cachedSort=%v search=%v plain=%v", gen, first, again, search, plain)
	if again > 30*time.Millisecond || plain > 30*time.Millisecond {
		t.Errorf("cached page too slow: %v %v", again, plain)
	}
}
