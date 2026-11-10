package core

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"time"
)

// PagerOptions configure a Pager. Zero fields take the defaults shown.
type PagerOptions struct {
	PageSize       int // 200
	MaxPages       int // 20 (LRU of cached pages per query)
	MaxInFlight    int // 2
	PrefetchMargin int // 50 rows
}

// DefaultPagerOptions returns {200, 20, 2, 50}.
func DefaultPagerOptions() PagerOptions {
	return PagerOptions{PageSize: 200, MaxPages: 20, MaxInFlight: 2, PrefetchMargin: 50}
}

// PagerStats describe the pager's current state, for the status bar.
type PagerStats struct {
	LastFetch   time.Duration // duration of the most recent completed page request
	CachedPages int
	InFlight    int
}

// Pager caches pages of a query result for a virtualized view. It never
// blocks: Row returns false for rows that are not loaded yet and the pager
// fetches the pages the view needs in the background.
//
// Algorithm:
//  1. Page k covers rows [k*PageSize, (k+1)*PageSize).
//  2. SetVisible(first, last) needs the pages covering
//     [first-PrefetchMargin, last+PrefetchMargin] clamped to [0, Total-1].
//     Pages that intersect [first, last] are "visible", the others "prefetch".
//     While Total is unknown, only the page containing first is requested.
//  3. Visible pages are fetched before prefetch pages, each group in
//     ascending distance from the page containing first. At most MaxInFlight
//     requests run at once. Cached or in-flight pages are not requested again.
//  4. Every response carries the generation it was requested under;
//     responses from an older generation (before SetQuery or Invalidate) are
//     discarded.
//  5. The cache is LRU by last SetVisible touch. When over MaxPages, the
//     least recently touched pages that are not visible are evicted.
//  6. A failed request is retried after one second.
//
// All methods are safe for concurrent use. onChange is called from
// background goroutines (possibly concurrently) whenever rows or Total
// change; the UI must marshal its refresh onto the UI thread.
type Pager struct {
	svc      DataService
	opts     PagerOptions
	onChange func()

	mu        sync.Mutex
	gen       uint64
	ctx       context.Context
	cancel    context.CancelFunc
	q         Query
	started   bool
	total     int
	pages     map[int]*pagerPage
	inFlight  map[int]bool
	failed    map[int]time.Time
	first     int
	last      int
	clock     uint64
	lastFetch time.Duration
}

type pagerPage struct {
	items   []Item
	touched uint64
	stale   bool
}

// NewPager creates a pager. Call SetQuery to start loading.
func NewPager(svc DataService, opts PagerOptions, onChange func()) *Pager {
	d := DefaultPagerOptions()
	if opts.PageSize <= 0 {
		opts.PageSize = d.PageSize
	}
	if opts.MaxPages <= 0 {
		opts.MaxPages = d.MaxPages
	}
	if opts.MaxInFlight <= 0 {
		opts.MaxInFlight = d.MaxInFlight
	}
	if opts.PrefetchMargin < 0 {
		opts.PrefetchMargin = d.PrefetchMargin
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Pager{
		svc: svc, opts: opts, onChange: onChange,
		ctx: ctx, cancel: cancel,
		total: -1, last: opts.PageSize - 1,
		pages: map[int]*pagerPage{}, inFlight: map[int]bool{}, failed: map[int]time.Time{},
	}
}

// Options returns the effective options.
func (p *Pager) Options() PagerOptions { return p.opts }

// SetQuery switches to a new query: clears the cache, bumps the generation
// (in-flight responses are discarded), moves the visible range to the top
// (keeping its height) and fetches page 0.
func (p *Pager) SetQuery(q Query) {
	p.mu.Lock()
	p.resetLocked()
	p.q = q.Normalize()
	p.started = true
	p.total = -1
	clear(p.pages)
	height := max(0, p.last-p.first)
	p.first, p.last = 0, height
	p.scheduleLocked()
	p.mu.Unlock()
	p.notify()
}

// Query returns the current (normalized) query.
func (p *Pager) Query() Query {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.q
}

// SetVisible reports the inclusive range of row indexes the view shows and
// schedules the fetches it needs. It does not call onChange.
func (p *Pager) SetVisible(first, last int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	first = max(0, first)
	if last < first {
		last = first
	}
	p.first, p.last = first, last
	p.clock++
	for k := first / p.opts.PageSize; k <= last/p.opts.PageSize; k++ {
		if pg := p.pages[k]; pg != nil {
			pg.touched = p.clock
		}
	}
	p.scheduleLocked()
	p.evictLocked()
}

// VisibleRange returns the range last passed to SetVisible (or reset by SetQuery).
func (p *Pager) VisibleRange() (first, last int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.first, p.last
}

// Total returns the number of rows in the current result, or -1 until the
// first page of the current query arrives.
func (p *Pager) Total() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.total
}

// Row returns row i if its page is loaded. It never blocks.
func (p *Pager) Row(i int) (Item, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.rowLocked(i)
}

func (p *Pager) rowLocked(i int) (Item, bool) {
	if i < 0 || (p.total >= 0 && i >= p.total) {
		return Item{}, false
	}
	pg := p.pages[i/p.opts.PageSize]
	if pg == nil {
		return Item{}, false
	}
	j := i % p.opts.PageSize
	if j >= len(pg.items) {
		return Item{}, false
	}
	return pg.items[j], true
}

// RangeLoaded reports whether Total is known and every row in
// [first, min(last, Total-1)] is loaded. An empty result counts as loaded.
func (p *Pager) RangeLoaded(first, last int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.total < 0 {
		return false
	}
	last = min(last, p.total-1)
	for i := max(0, first); i <= last; i++ {
		if _, ok := p.rowLocked(i); !ok {
			return false
		}
	}
	return true
}

// Invalidate refetches the visible pages of the current query (for example
// after an Update), keeping their current data available until the fresh
// data arrives, and drops all other cached pages.
func (p *Pager) Invalidate() {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return
	}
	p.resetLocked()
	for k, pg := range p.pages {
		if p.visibleLocked(k) {
			pg.stale = true
		} else {
			delete(p.pages, k)
		}
	}
	p.scheduleLocked()
	p.mu.Unlock()
}

// Stats returns the last fetch duration and cache counts.
func (p *Pager) Stats() PagerStats {
	p.mu.Lock()
	defer p.mu.Unlock()
	return PagerStats{LastFetch: p.lastFetch, CachedPages: len(p.pages), InFlight: len(p.inFlight)}
}

// Close cancels in-flight requests. The pager must not be used afterwards.
func (p *Pager) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cancel()
	p.started = false
}

func (p *Pager) resetLocked() {
	p.cancel()
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.gen++
	clear(p.inFlight)
	clear(p.failed)
}

func (p *Pager) visibleLocked(k int) bool {
	start, end := k*p.opts.PageSize, (k+1)*p.opts.PageSize-1
	return start <= p.last && end >= p.first
}

func (p *Pager) scheduleLocked() {
	if !p.started || p.total == 0 {
		return
	}
	ps := p.opts.PageSize
	anchor := p.first / ps
	var order []int
	if p.total < 0 {
		order = []int{anchor}
	} else {
		lo := max(0, p.first-p.opts.PrefetchMargin)
		hi := min(p.total-1, p.last+p.opts.PrefetchMargin)
		if hi < lo {
			return
		}
		var vis, pre []int
		for k := lo / ps; k <= hi/ps; k++ {
			if p.visibleLocked(k) {
				vis = append(vis, k)
			} else {
				pre = append(pre, k)
			}
		}
		byDist := func(a, b int) int {
			return cmp.Or(cmp.Compare(absInt(a-anchor), absInt(b-anchor)), cmp.Compare(a, b))
		}
		slices.SortFunc(vis, byDist)
		slices.SortFunc(pre, byDist)
		order = append(vis, pre...)
	}
	now := time.Now()
	for _, k := range order {
		if len(p.inFlight) >= p.opts.MaxInFlight {
			return
		}
		if pg := p.pages[k]; pg != nil && !pg.stale {
			continue
		}
		if p.inFlight[k] {
			continue
		}
		if t, ok := p.failed[k]; ok && now.Sub(t) < time.Second {
			continue
		}
		p.inFlight[k] = true
		go p.fetch(p.ctx, p.gen, p.q, k)
	}
}

func (p *Pager) fetch(ctx context.Context, gen uint64, q Query, k int) {
	start := time.Now()
	page, err := p.svc.Page(ctx, PageRequest{Query: q, Offset: k * p.opts.PageSize, Limit: p.opts.PageSize})
	elapsed := time.Since(start)

	p.mu.Lock()
	if gen != p.gen {
		p.mu.Unlock()
		return
	}
	delete(p.inFlight, k)
	if err != nil {
		p.failed[k] = time.Now()
		time.AfterFunc(time.Second, func() {
			p.mu.Lock()
			if gen == p.gen {
				p.scheduleLocked()
			}
			p.mu.Unlock()
		})
		p.scheduleLocked()
		p.mu.Unlock()
		return
	}
	delete(p.failed, k)
	p.clock++
	p.pages[k] = &pagerPage{items: page.Items, touched: p.clock}
	p.total = page.Total
	p.lastFetch = elapsed
	maxPage := (p.total - 1) / p.opts.PageSize
	for pk := range p.pages {
		if pk > maxPage {
			delete(p.pages, pk)
		}
	}
	p.evictLocked()
	p.scheduleLocked()
	p.mu.Unlock()
	p.notify()
}

func (p *Pager) evictLocked() {
	for len(p.pages) > p.opts.MaxPages {
		victim, found := 0, false
		var oldest uint64
		for k, pg := range p.pages {
			if p.visibleLocked(k) {
				continue
			}
			if !found || pg.touched < oldest {
				victim, oldest, found = k, pg.touched, true
			}
		}
		if !found {
			return
		}
		delete(p.pages, victim)
	}
}

func (p *Pager) notify() {
	if p.onChange != nil {
		p.onChange()
	}
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
