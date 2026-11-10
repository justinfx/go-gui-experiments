//go:build !js

package main

import (
	"context"
	"slices"
	"time"

	"gguix/core"

	"github.com/mappu/miqt/qt6/mainthread"
)

// Data layer for MIQT, part 1: rowStore.
//
// rowStore replaces core.Pager. It is owned by the Qt main thread: the models'
// data() callbacks, the scroll handlers and the fetch completions all run
// there, so reads need no mutex and a read can never race a page arrival.
// Service calls run on goroutines and hand their result back through
// mainthread.Start. Compared with core.Pager it:
//
//   - cancels in-flight requests that the view has scrolled away from, so a
//     fast scroll never queues stale pages behind the 50 ms service latency;
//   - prefetches in the scroll direction, sized by the measured scroll speed
//     (rows per second times a latency budget), instead of a fixed margin;
//   - runs more requests in parallel (the service is a remote, high-latency
//     source) while staying under the 200 rows per call and 10,000 rows held
//     limits;
//   - reports arrivals as a batch of page ranges, so the models can emit one
//     dataChanged for the rows that are on screen and nothing for the rest.

const (
	storePageSize  = 200
	storeMaxPages  = 40 // 8,000 rows, under the 10,000 row cap
	storeMaxFlight = 2
	storeMargin    = 50                     // rows kept around the visible range
	storeLookahead = 220 * time.Millisecond // how far ahead of the scroll speed to prefetch
	storeMaxAhead  = 3000                   // cap on the lookahead, in rows
)

type storePage struct {
	items   []core.Item
	touched uint64
	stale   bool
}

type rowStore struct {
	svc core.DataService
	ctx context.Context

	// onArrive runs on the UI thread after a batch of pages landed. It gets
	// the page row ranges that arrived since the last call.
	onArrive func(ranges [][2]int)

	q      core.Query
	gen    uint64
	total  int // -1 until the first page of the query arrives
	pages  map[int]*storePage
	flight map[int]context.CancelFunc
	failed map[int]time.Time

	first, last int
	lastTime    time.Time
	speed       float64 // signed rows per second, smoothed
	clock       uint64
	lastFetch   time.Duration
	started     bool
	calls       int // service calls started (diagnostics)

	arrived      [][2]int
	flushPending bool
}

func newRowStore(ctx context.Context, svc core.DataService) *rowStore {
	return &rowStore{
		svc: svc, ctx: ctx, total: -1, last: 30,
		pages:  map[int]*storePage{},
		flight: map[int]context.CancelFunc{},
		failed: map[int]time.Time{},
	}
}

// setQuery switches query: drop everything, cancel everything, fetch page 0.
func (s *rowStore) setQuery(q core.Query) {
	s.cancelAll()
	s.gen++
	s.q = q.Normalize()
	s.started = true
	s.total = -1
	clear(s.pages)
	clear(s.failed)
	s.arrived = nil
	height := max(0, s.last-s.first)
	s.first, s.last = 0, height
	s.speed = 0
	s.pump()
}

func (s *rowStore) cancelAll() {
	for k, c := range s.flight {
		c()
		delete(s.flight, k)
	}
}

// invalidate refetches the wanted pages (after an edit) and keeps showing
// their old data until the fresh page arrives.
func (s *rowStore) invalidate() {
	if !s.started {
		return
	}
	s.cancelAll()
	s.gen++
	clear(s.failed)
	for k, pg := range s.pages {
		if s.visiblePage(k) {
			pg.stale = true
		} else {
			delete(s.pages, k)
		}
	}
	s.pump()
}

func (s *rowStore) visiblePage(k int) bool {
	return k*storePageSize <= s.last && (k+1)*storePageSize-1 >= s.first
}

// setVisible reports the rows on screen. It measures the scroll speed and
// reschedules the fetches.
func (s *rowStore) setVisible(first, last int) {
	first = max(0, first)
	last = max(first, last)
	now := time.Now()
	if !s.lastTime.IsZero() && first != s.first {
		dt := now.Sub(s.lastTime).Seconds()
		if dt > 0.001 {
			v := float64(first-s.first) / dt
			s.speed = 0.5*s.speed + 0.5*v
		}
	} else if first == s.first && now.Sub(s.lastTime) > 150*time.Millisecond {
		s.speed = 0
	}
	if first != s.first {
		s.lastTime = now
	}
	s.first, s.last = first, last
	s.pump()
}

// wanted lists the pages worth having, most urgent first: the visible pages,
// then the pages in the scroll direction (or both sides when still), nearest
// first.
func (s *rowStore) wanted() []int {
	ps := storePageSize
	anchor := s.first / ps
	if s.total < 0 {
		return []int{anchor}
	}
	if s.total == 0 {
		return nil
	}
	ahead := min(storeMaxAhead, max(storeMargin, int(abs(s.speed)*storeLookahead.Seconds())))
	behind := storeMargin
	lo, hi := s.first-behind, s.last+behind
	switch {
	case s.speed > 50:
		hi = s.last + ahead
	case s.speed < -50:
		lo = s.first - ahead
	}
	lo = max(0, lo)
	hi = min(s.total-1, hi)
	var vis, pre []int
	for k := lo / ps; k <= hi/ps && hi >= lo; k++ {
		if s.visiblePage(k) {
			vis = append(vis, k)
		} else {
			pre = append(pre, k)
		}
	}
	dist := func(a, b int) int {
		da, db := abs2(a-anchor), abs2(b-anchor)
		if da != db {
			return da - db
		}
		return a - b
	}
	slices.SortFunc(vis, dist)
	slices.SortFunc(pre, dist)
	return append(vis, pre...)
}

func (s *rowStore) pump() {
	if !s.started {
		return
	}
	want := s.wanted()
	wantSet := make(map[int]bool, len(want))
	s.clock++
	for _, k := range want {
		wantSet[k] = true
		if pg := s.pages[k]; pg != nil {
			pg.touched = s.clock
		}
	}
	// Cancel requests nobody needs any more.
	for k, c := range s.flight {
		if !wantSet[k] {
			c()
			delete(s.flight, k)
		}
	}
	now := time.Now()
	for _, k := range want {
		if len(s.flight) >= storeMaxFlight {
			break
		}
		if pg := s.pages[k]; pg != nil && !pg.stale {
			continue
		}
		if _, ok := s.flight[k]; ok {
			continue
		}
		if t, ok := s.failed[k]; ok && now.Sub(t) < time.Second {
			continue
		}
		s.fetch(k)
	}
	s.evict(wantSet)
}

func (s *rowStore) fetch(k int) {
	ctx, cancel := context.WithCancel(s.ctx)
	s.flight[k] = cancel
	s.calls++
	gen, q := s.gen, s.q
	go func() {
		start := time.Now()
		page, err := s.svc.Page(ctx, core.PageRequest{Query: q, Offset: k * storePageSize, Limit: storePageSize})
		elapsed := time.Since(start)
		if ctx.Err() != nil {
			return
		}
		mainthread.Start(func() { s.deliver(gen, k, page, err, elapsed) })
	}()
}

func (s *rowStore) deliver(gen uint64, k int, page core.Page, err error, elapsed time.Duration) {
	if gen != s.gen || s.ctx.Err() != nil {
		return
	}
	cancel, ok := s.flight[k]
	if !ok {
		return
	}
	cancel()
	delete(s.flight, k)
	if err != nil {
		s.failed[k] = time.Now()
		time.AfterFunc(time.Second, func() {
			mainthread.Start(func() {
				if gen == s.gen {
					s.pump()
				}
			})
		})
		s.pump()
		return
	}
	delete(s.failed, k)
	s.clock++
	s.pages[k] = &storePage{items: page.Items, touched: s.clock}
	s.total = page.Total
	s.lastFetch = elapsed
	maxPage := (s.total - 1) / storePageSize
	for pk := range s.pages {
		if pk > maxPage {
			delete(s.pages, pk)
		}
	}
	s.arrived = append(s.arrived, [2]int{k * storePageSize, k*storePageSize + len(page.Items) - 1})
	s.queueFlush()
	s.pump()
}

// queueFlush coalesces page arrivals: everything that lands within a few
// milliseconds produces one notification and one repaint.
func (s *rowStore) queueFlush() {
	if s.flushPending {
		return
	}
	s.flushPending = true
	time.AfterFunc(4*time.Millisecond, func() { mainthread.Start(s.flush) })
}

func (s *rowStore) flush() {
	s.flushPending = false
	if s.ctx.Err() != nil { // app is shutting down; the widgets may be gone
		return
	}
	r := s.arrived
	s.arrived = nil
	if s.onArrive != nil {
		s.onArrive(r)
	}
}

func (s *rowStore) evict(keep map[int]bool) {
	for len(s.pages) > storeMaxPages {
		victim, found := 0, false
		var oldest uint64
		for k, pg := range s.pages {
			if keep[k] || s.visiblePage(k) {
				continue
			}
			if !found || pg.touched < oldest {
				victim, oldest, found = k, pg.touched, true
			}
		}
		if !found {
			return
		}
		delete(s.pages, victim)
	}
}

// ---- reads (UI thread)

func (s *rowStore) Total() int { return s.total }

func (s *rowStore) Row(i int) (core.Item, bool) {
	if i < 0 || (s.total >= 0 && i >= s.total) {
		return core.Item{}, false
	}
	pg := s.pages[i/storePageSize]
	if pg == nil {
		return core.Item{}, false
	}
	j := i % storePageSize
	if j >= len(pg.items) {
		return core.Item{}, false
	}
	return pg.items[j], true
}

// RangeLoaded reports whether Total is known and rows first..last all have data.
func (s *rowStore) RangeLoaded(first, last int) bool {
	if s.total < 0 {
		return false
	}
	last = min(last, s.total-1)
	for k := max(0, first) / storePageSize; k <= last/storePageSize && last >= 0; k++ {
		pg := s.pages[k]
		if pg == nil {
			return false
		}
		end := min(last, (k+1)*storePageSize-1) - k*storePageSize
		if end >= len(pg.items) {
			return false
		}
	}
	return true
}

// settled is true when no arrived page is waiting for its UI notification.
func (s *rowStore) settled() bool { return !s.flushPending }

func (s *rowStore) stats() (lastFetch time.Duration, cached int) {
	return s.lastFetch, len(s.pages)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func abs2(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
