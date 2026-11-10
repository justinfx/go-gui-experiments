package gotth

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gguix/core"
	"gguix/core/memory"
)

// spySvc records every Page call so the tests can check the fetch limits.
type spySvc struct {
	core.DataService
	mu     sync.Mutex
	limits []int
	gets   atomic.Int32
}

func (s *spySvc) Page(ctx context.Context, req core.PageRequest) (core.Page, error) {
	s.mu.Lock()
	s.limits = append(s.limits, req.Limit)
	s.mu.Unlock()
	return s.DataService.Page(ctx, req)
}

func (s *spySvc) Get(ctx context.Context, id int64) (core.Item, error) {
	s.gets.Add(1)
	return s.DataService.Get(ctx, id)
}

func newSpy(latency time.Duration) (*spySvc, *cachedSvc) {
	spy := &spySvc{DataService: memory.New(memory.Options{Rows: 5000, Latency: latency, Seed: 1})}
	return spy, newCachedSvc(spy)
}

// Concurrent requests for rows of one page (the browser's page fetch and the
// go-to-row lookup) share a single service call.
func TestAdapterSingleFlight(t *testing.T) {
	_, c := newSpy(30 * time.Millisecond)
	ctx := context.Background()
	var wg sync.WaitGroup
	for _, req := range []core.PageRequest{
		{Offset: 400, Limit: 200},
		{Offset: 450, Limit: 1},
		{Offset: 599, Limit: 1},
		{Offset: 400, Limit: 200},
	} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pg, err := c.Page(ctx, req)
			if err != nil || len(pg.Items) != req.Limit || pg.Items[0].ID != int64(req.Offset+1) {
				t.Errorf("request %+v: got %d items, err %v", req, len(pg.Items), err)
			}
		}()
	}
	wg.Wait()
	if calls, _ := c.Stats(); calls != 1 {
		t.Errorf("service called %d times, want 1", calls)
	}
}

// Every service call asks for at most 200 rows, and no more than
// maxCachedPages pages stay in memory, however many pages are visited.
func TestAdapterLimits(t *testing.T) {
	spy, c := newSpy(0)
	ctx := context.Background()
	for p := 0; p < 25; p++ {
		if _, err := c.Page(ctx, core.PageRequest{Offset: p * 200, Limit: 200}); err != nil {
			t.Fatal(err)
		}
	}
	// A request that does not fit one aligned page goes to the service as is.
	if _, err := c.Page(ctx, core.PageRequest{Offset: 100, Limit: 200}); err != nil {
		t.Fatal(err)
	}
	spy.mu.Lock()
	for _, l := range spy.limits {
		if l > 200 {
			t.Errorf("service call with limit %d", l)
		}
	}
	spy.mu.Unlock()
	c.mu.Lock()
	n := len(c.pages)
	c.mu.Unlock()
	if n > maxCachedPages || n*pageRows > 10000 {
		t.Errorf("%d pages cached", n)
	}
	// A cached page is served again without a call.
	before, _ := c.Stats()
	if _, err := c.Page(ctx, core.PageRequest{Offset: 24 * 200, Limit: 200}); err != nil {
		t.Fatal(err)
	}
	if after, _ := c.Stats(); after != before {
		t.Error("cached page fetched again")
	}
}

// The form is filled from a cached page; an Update drops the cache.
func TestAdapterGetAndUpdate(t *testing.T) {
	spy, c := newSpy(0)
	ctx := context.Background()
	pg, err := c.Page(ctx, core.PageRequest{Offset: 0, Limit: 200})
	if err != nil {
		t.Fatal(err)
	}
	it, err := c.Get(ctx, 7)
	if err != nil || it.ID != 7 || spy.gets.Load() != 0 {
		t.Fatalf("Get 7: %v, %d service Gets", err, spy.gets.Load())
	}
	if _, err := c.Get(ctx, 4000); err != nil || spy.gets.Load() != 1 {
		t.Fatalf("Get outside the cache should reach the service once, got %d", spy.gets.Load())
	}
	edit := pg.Items[6]
	edit.Name = "Renamed By Test"
	if _, err := c.Update(ctx, edit); err != nil {
		t.Fatal(err)
	}
	pg2, err := c.Page(ctx, core.PageRequest{Offset: 0, Limit: 200})
	if err != nil || pg2.Items[6].Name != "Renamed By Test" {
		t.Errorf("page after Update shows %q (err %v)", pg2.Items[6].Name, err)
	}
	if calls, _ := c.Stats(); calls != 2 {
		t.Errorf("want one refetch after Update, calls=%d", calls)
	}
}

// A caller that gives up does not lose the page: the call finishes and is
// cached for the next request.
func TestAdapterAbandonedCall(t *testing.T) {
	_, c := newSpy(40 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	if _, err := c.Page(ctx, core.PageRequest{Offset: 1000, Limit: 200}); err == nil {
		t.Fatal("expected the caller to time out")
	}
	time.Sleep(80 * time.Millisecond)
	start := time.Now()
	if _, err := c.Page(context.Background(), core.PageRequest{Offset: 1000, Limit: 200}); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 20*time.Millisecond {
		t.Error("abandoned call was not cached")
	}
	if calls, _ := c.Stats(); calls != 1 {
		t.Errorf("calls=%d want 1", calls)
	}
}
