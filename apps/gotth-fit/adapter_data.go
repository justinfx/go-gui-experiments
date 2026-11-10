package gotth

import (
	"container/list"
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"gguix/core"
)

// The data adapter: a core.DataService decorator that sits between the HTTP
// handlers and the slow core service.
//
// The view asks for rows in whatever shape suits it (a page, one row for
// go-to-row, one item for the form). The adapter turns every row request into
// a request for the aligned 200-row page that holds it and keeps those pages
// in a small server-side LRU, so
//
//   - concurrent requests for the same page (the browser's page fetch and the
//     go-to-row lookup, a table to cards switch, a retried request) share one
//     service call (single flight);
//   - a page that is fetched is not fetched again while it stays in the LRU;
//   - the form is filled from a cached page when the row is in one, with no
//     service call at all;
//   - a call that the browser abandons (the user scrolled past it) still
//     finishes and lands in the cache, because the call is detached from the
//     request context. The bound on work is the LRU size, not the client.
//
// Limits (checked by adapter_data_test.go): every service call asks for at
// most pageRows rows, and at most maxCachedPages*pageRows rows are held.
const (
	pageRows       = 200 // rows per service call; equals views.PageSize
	maxCachedPages = 24  // 4,800 rows held, well below the 10,000 row limit
	fetchTimeout   = 30 * time.Second
)

type pageKey struct {
	q   string
	idx int
}

type pageEntry struct {
	key   pageKey
	ready chan struct{} // closed when page/err are set
	page  core.Page
	err   error
	el    *list.Element // position in the LRU; nil while in flight
}

// cachedSvc implements core.DataService.
type cachedSvc struct {
	core.DataService

	mu    sync.Mutex
	gen   uint64
	pages map[pageKey]*pageEntry
	lru   *list.List // of *pageEntry, front = most recently used

	calls atomic.Int64 // underlying Page calls
	hits  atomic.Int64 // requests answered without a service call
	trace bool
}

func newCachedSvc(svc core.DataService) *cachedSvc {
	return &cachedSvc{
		DataService: svc,
		pages:       map[pageKey]*pageEntry{},
		lru:         list.New(),
		trace:       os.Getenv("GOTTH_LOG") == "1",
	}
}

// Stats returns the number of service Page calls and of requests served from
// the cache or shared with a call already in flight.
func (c *cachedSvc) Stats() (calls, hits int64) { return c.calls.Load(), c.hits.Load() }

// Page serves a request from the aligned page that contains it. A request
// that does not fit inside one aligned page goes straight to the service.
func (c *cachedSvc) Page(ctx context.Context, req core.PageRequest) (core.Page, error) {
	if req.Limit < 1 || req.Limit > pageRows || req.Offset < 0 || req.Offset%pageRows+req.Limit > pageRows {
		return c.DataService.Page(ctx, req)
	}
	idx := req.Offset / pageRows
	pg, err := c.pageAt(ctx, req.Query, idx)
	if err != nil {
		return core.Page{}, err
	}
	lo := req.Offset - idx*pageRows
	hi := min(lo+req.Limit, len(pg.Items))
	out := pg
	out.Offset = req.Offset
	if lo >= hi {
		out.Items = nil
	} else {
		out.Items = pg.Items[lo:hi:hi]
	}
	return out, nil
}

// pageAt returns aligned page idx of q, fetching it when needed.
func (c *cachedSvc) pageAt(ctx context.Context, q core.Query, idx int) (core.Page, error) {
	key := pageKey{q.Key(), idx}
	c.mu.Lock()
	e, ok := c.pages[key]
	if ok {
		if e.el != nil {
			c.lru.MoveToFront(e.el)
		}
		c.mu.Unlock()
		c.hits.Add(1)
	} else {
		e = &pageEntry{key: key, ready: make(chan struct{})}
		c.pages[key] = e
		gen := c.gen
		c.mu.Unlock()
		go c.fetch(e, q, gen)
	}
	select {
	case <-e.ready:
		return e.page, e.err
	case <-ctx.Done():
		return core.Page{}, ctx.Err()
	}
}

func (c *cachedSvc) fetch(e *pageEntry, q core.Query, gen uint64) {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()
	c.calls.Add(1)
	t0 := time.Now()
	pg, err := c.DataService.Page(ctx, core.PageRequest{Query: q, Offset: e.key.idx * pageRows, Limit: pageRows})
	if c.trace {
		fmt.Fprintf(os.Stderr, "SVC  Page %s #%d %s err=%v\n", e.key.q, e.key.idx, time.Since(t0).Round(time.Millisecond), err)
	}
	c.mu.Lock()
	e.page, e.err = pg, err
	if err != nil || gen != c.gen {
		// Failed, or an Update happened while the call was in flight: serve the
		// waiters but do not keep the page.
		if c.pages[e.key] == e {
			delete(c.pages, e.key)
		}
	} else {
		e.el = c.lru.PushFront(e)
		for c.lru.Len() > maxCachedPages {
			old := c.lru.Remove(c.lru.Back()).(*pageEntry)
			delete(c.pages, old.key)
		}
	}
	c.mu.Unlock()
	close(e.ready)
}

// Get answers from a cached page when the item is in one.
func (c *cachedSvc) Get(ctx context.Context, id int64) (core.Item, error) {
	c.mu.Lock()
	for el := c.lru.Front(); el != nil; el = el.Next() {
		for _, it := range el.Value.(*pageEntry).page.Items {
			if it.ID == id {
				c.lru.MoveToFront(el)
				c.mu.Unlock()
				c.hits.Add(1)
				return it, nil
			}
		}
	}
	c.mu.Unlock()
	return c.DataService.Get(ctx, id)
}

// Update writes through and drops every cached page: an edit can change the
// order and the membership of any query.
func (c *cachedSvc) Update(ctx context.Context, it core.Item) (core.Item, error) {
	saved, err := c.DataService.Update(ctx, it)
	if err == nil {
		c.Invalidate()
	}
	return saved, err
}

// Invalidate drops all cached pages and discards calls still in flight.
func (c *cachedSvc) Invalidate() {
	c.mu.Lock()
	c.gen++
	c.pages = map[pageKey]*pageEntry{}
	c.lru.Init()
	c.mu.Unlock()
}
