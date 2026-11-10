package core_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gguix/core"
	"gguix/core/memory"
)

// countingSvc records page requests.
type countingSvc struct {
	core.DataService
	mu      sync.Mutex
	offsets []int
	maxConc int32
	cur     int32
}

func (c *countingSvc) Page(ctx context.Context, r core.PageRequest) (core.Page, error) {
	n := atomic.AddInt32(&c.cur, 1)
	defer atomic.AddInt32(&c.cur, -1)
	c.mu.Lock()
	c.offsets = append(c.offsets, r.Offset)
	if n > c.maxConc {
		c.maxConc = n
	}
	c.mu.Unlock()
	return c.DataService.Page(ctx, r)
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timeout")
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestPagerLoadsVisibleAndPrefetches(t *testing.T) {
	svc := &countingSvc{DataService: memory.New(memory.Options{Rows: 10_000, Latency: 5 * time.Millisecond})}
	var changes atomic.Int32
	p := core.NewPager(svc, core.PagerOptions{}, func() { changes.Add(1) })
	if p.Total() != -1 {
		t.Fatal("total before query")
	}
	p.SetQuery(core.Query{})
	p.SetVisible(0, 30)
	waitFor(t, func() bool { return p.RangeLoaded(0, 30) })
	if p.Total() != 10_000 {
		t.Fatal(p.Total())
	}
	it, ok := p.Row(5)
	if !ok || it.ID != 6 {
		t.Fatalf("row 5 = %v %v", it.ID, ok)
	}
	// Jump far: visible page fetched, plus prefetch of the neighbor.
	p.SetVisible(5190, 5220)
	waitFor(t, func() bool { return p.RangeLoaded(5190, 5220) })
	waitFor(t, func() bool { _, ok := p.Row(5250); return ok }) // prefetch margin reaches page 26
	if svc.maxConc > 2 {
		t.Fatalf("max concurrency %d", svc.maxConc)
	}
	if changes.Load() == 0 {
		t.Fatal("no onChange")
	}
}

func TestPagerDiscardsStaleGenerations(t *testing.T) {
	svc := memory.New(memory.Options{Rows: 5000, Latency: 30 * time.Millisecond})
	p := core.NewPager(svc, core.PagerOptions{}, nil)
	p.SetQuery(core.Query{})
	p.SetQuery(core.Query{Search: "steel"})
	p.SetVisible(0, 10)
	waitFor(t, func() bool { return p.RangeLoaded(0, 10) })
	it, _ := p.Row(0)
	if !(core.Query{Search: "steel"}).Matches(it) {
		t.Fatalf("row from old query: %s", it.Name)
	}
}

func TestPagerEvictsAndInvalidates(t *testing.T) {
	svc := memory.New(memory.Options{Rows: 100_000})
	p := core.NewPager(svc, core.PagerOptions{MaxPages: 3, PrefetchMargin: 0}, nil)
	p.SetQuery(core.Query{})
	for _, start := range []int{0, 2000, 4000, 6000, 8000} {
		p.SetVisible(start, start+10)
		waitFor(t, func() bool { return p.RangeLoaded(start, start+10) })
	}
	if st := p.Stats(); st.CachedPages > 3 {
		t.Fatalf("cached %d", st.CachedPages)
	}
	if _, ok := p.Row(8005); !ok {
		t.Fatal("visible page evicted")
	}
	// Invalidate keeps visible data available while refetching.
	ctx := context.Background()
	item, _ := svc.Get(ctx, 8006)
	item.Name = "Renamed Item"
	if _, err := svc.Update(ctx, item); err != nil {
		t.Fatal(err)
	}
	p.Invalidate()
	if _, ok := p.Row(8005); !ok {
		t.Fatal("visible row dropped during invalidate")
	}
	waitFor(t, func() bool { it, _ := p.Row(8005); return it.Name == "Renamed Item" })
}

func TestPagerEmptyResult(t *testing.T) {
	p := core.NewPager(memory.New(memory.Options{Rows: 100}), core.PagerOptions{}, nil)
	p.SetQuery(core.Query{Search: "no such thing"})
	waitFor(t, func() bool { return p.Total() == 0 })
	if !p.RangeLoaded(0, 20) {
		t.Fatal("empty result should count as loaded")
	}
}
