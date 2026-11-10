package main

// adapter_store.go is the data layer of the app: a windowed, lazily fetching
// row store that replaces core.Pager. It differs from the prescribed pager in
// how it serves the Fyne views:
//
//   - reads (Row, Total, RangeLoaded) are lock free: they read an immutable
//     snapshot published through an atomic pointer, so the per-cell update
//     callbacks of widget.Table and widget.List never contend with fetches;
//   - the window of wanted pages follows the scroll direction and speed, so
//     fast scrolling prefetches ahead of the viewport instead of symmetric;
//   - requests for pages that left the window are cancelled, so a jump does
//     not wait behind obsolete fetches;
//   - change notifications are coalesced (at most one per frame) and say
//     whether a page inside the visible range changed, so the UI refreshes
//     the view only when something on screen can be different.
//
// All rows come from core.DataService.Page with at most 200 rows per call,
// and at most maxPages*200 rows (plus in-flight pages) are held in memory.

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"gguix/core"
)

const (
	storePageSize   = 200 // rows per service call (limit: 200)
	storeMaxPages   = 40  // 8,000 rows cached (limit: 10,000 held at once)
	storeMaxFlight  = 2   // concurrent service calls
	storeBaseMargin = 50  // rows kept around the viewport
	storeMaxAhead   = 1000
	storeRetry      = time.Second
	storeNotifyGap  = 16 * time.Millisecond
)

// storeSnap is an immutable view of the cache.
type storeSnap struct {
	total int // -1 until the first page of the query arrives
	pages map[int][]core.Item
}

// storeChange tells the UI what a notification is about.
type storeChange struct {
	Visible bool // rows or total in the visible range changed
}

type rowStore struct {
	svc      core.DataService
	onChange func(storeChange)

	snap atomic.Pointer[storeSnap]

	mu        sync.Mutex
	gen       uint64
	q         core.Query
	started   bool
	closed    bool
	first     int
	last      int
	vel       float64 // smoothed scroll velocity in rows per update
	touched   map[int]uint64
	clock     uint64
	stale     map[int]bool
	inFlight  map[int]context.CancelFunc
	failed    map[int]time.Time
	lastFetch time.Duration

	// coalesced notification state
	notifyMu      sync.Mutex
	notifyTimer   *time.Timer
	lastNotify    time.Time
	pendingChange storeChange
	pendingAny    bool
}

func newRowStore(svc core.DataService, onChange func(storeChange)) *rowStore {
	s := &rowStore{
		svc: svc, onChange: onChange,
		last:     storePageSize - 1,
		touched:  map[int]uint64{},
		stale:    map[int]bool{},
		inFlight: map[int]context.CancelFunc{},
		failed:   map[int]time.Time{},
	}
	s.snap.Store(&storeSnap{total: -1, pages: map[int][]core.Item{}})
	return s
}

func (s *rowStore) PageSize() int { return storePageSize }

// SetQuery switches to a new query: the cache is cleared, requests of the
// old query are cancelled, and the page at the top is fetched.
func (s *rowStore) SetQuery(q core.Query) {
	s.mu.Lock()
	s.resetLocked()
	s.q = q.Normalize()
	s.started = true
	clear(s.touched)
	clear(s.stale)
	height := max(0, s.last-s.first)
	s.first, s.last, s.vel = 0, height, 0
	s.snap.Store(&storeSnap{total: -1, pages: map[int][]core.Item{}})
	s.scheduleLocked()
	s.mu.Unlock()
	s.notify(storeChange{Visible: true})
}

func (s *rowStore) Query() core.Query {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.q
}

// SetVisible reports the inclusive visible row range and (re)plans fetches.
func (s *rowStore) SetVisible(first, last int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	first = max(0, first)
	if last < first {
		last = first
	}
	delta := float64(first - s.first)
	if delta > 3000 || delta < -3000 {
		s.vel = 0 // a jump is not a scroll
	} else {
		s.vel = 0.5*s.vel + 0.5*delta
	}
	s.first, s.last = first, last
	s.clock++
	for k := first / storePageSize; k <= last/storePageSize; k++ {
		s.touched[k] = s.clock
	}
	s.scheduleLocked()
	s.evictLocked()
}

func (s *rowStore) Total() int { return s.snap.Load().total }

// Row returns row i if loaded. It never blocks and takes no lock.
func (s *rowStore) Row(i int) (core.Item, bool) {
	return rowFrom(s.snap.Load(), i)
}

func rowFrom(sn *storeSnap, i int) (core.Item, bool) {
	if i < 0 || (sn.total >= 0 && i >= sn.total) {
		return core.Item{}, false
	}
	pg, ok := sn.pages[i/storePageSize]
	if !ok {
		return core.Item{}, false
	}
	j := i % storePageSize
	if j >= len(pg) {
		return core.Item{}, false
	}
	return pg[j], true
}

// RangeLoaded reports that Total is known and every row of [first,last]
// (clamped to the result) has data.
func (s *rowStore) RangeLoaded(first, last int) bool {
	sn := s.snap.Load()
	if sn.total < 0 {
		return false
	}
	last = min(last, sn.total-1)
	for k := max(0, first) / storePageSize; k <= last/storePageSize && last >= 0; k++ {
		pg, ok := sn.pages[k]
		if !ok {
			return false
		}
		// only the last page of a result can be short
		lo := max(first, k*storePageSize)
		hi := min(last, k*storePageSize+storePageSize-1)
		if hi-k*storePageSize >= len(pg) || lo > hi {
			return false
		}
	}
	return true
}

// Invalidate refetches the pages of the visible window after an Update. The
// current rows stay readable until the fresh ones arrive; others are dropped.
func (s *rowStore) Invalidate() {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return
	}
	s.resetLocked()
	old := s.snap.Load()
	keep := map[int][]core.Item{}
	clear(s.stale)
	for k, pg := range old.pages {
		if s.visiblePage(k) {
			keep[k] = pg
			s.stale[k] = true
		}
	}
	s.snap.Store(&storeSnap{total: old.total, pages: keep})
	s.scheduleLocked()
	s.mu.Unlock()
}

type storeStats struct {
	LastFetch   time.Duration
	CachedPages int
	InFlight    int
}

func (s *rowStore) Stats() storeStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return storeStats{LastFetch: s.lastFetch, CachedPages: len(s.snap.Load().pages), InFlight: len(s.inFlight)}
}

// Close cancels in-flight requests.
func (s *rowStore) Close() {
	s.mu.Lock()
	s.closed = true
	s.resetLocked()
	s.started = false
	s.mu.Unlock()
}

func (s *rowStore) resetLocked() {
	s.gen++
	for _, c := range s.inFlight {
		c()
	}
	clear(s.inFlight)
	clear(s.failed)
}

func (s *rowStore) visiblePage(k int) bool {
	return k*storePageSize <= s.last && (k+1)*storePageSize-1 >= s.first
}

// window returns the wanted page range [lo,hi] in rows: the viewport plus a
// margin, widened ahead of the scroll direction in proportion to its speed.
func (s *rowStore) windowLocked(total int) (int, int) {
	back, fwd := storeBaseMargin, storeBaseMargin
	ahead := min(storeMaxAhead, int(10*abs(s.vel)))
	if s.vel > 0 {
		fwd = max(fwd, ahead)
	} else if s.vel < 0 {
		back = max(back, ahead)
	}
	lo := max(0, s.first-back)
	hi := s.last + fwd
	if total >= 0 {
		hi = min(hi, total-1)
	}
	return lo, hi
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func (s *rowStore) scheduleLocked() {
	if !s.started {
		return
	}
	sn := s.snap.Load()
	if sn.total == 0 {
		return
	}
	anchor := s.first / storePageSize
	var order []int
	want := map[int]bool{}
	if sn.total < 0 {
		order = []int{anchor}
		want[anchor] = true
	} else {
		lo, hi := s.windowLocked(sn.total)
		if hi < lo {
			return
		}
		var vis, pre []int
		for k := lo / storePageSize; k <= hi/storePageSize; k++ {
			want[k] = true
			if s.visiblePage(k) {
				vis = append(vis, k)
			} else {
				pre = append(pre, k)
			}
		}
		// prefetch pages are ordered by distance, ahead of the scroll first
		dir := 1
		if s.vel < 0 {
			dir = -1
		}
		slices.SortFunc(vis, func(a, b int) int { return absInt(a-anchor) - absInt(b-anchor) })
		slices.SortFunc(pre, func(a, b int) int {
			da, db := absInt(a-anchor), absInt(b-anchor)
			if (a-anchor)*dir < 0 {
				da += 2
			}
			if (b-anchor)*dir < 0 {
				db += 2
			}
			return da - db
		})
		order = append(vis, pre...)
	}
	// cancel requests that left the window
	for k, c := range s.inFlight {
		if !want[k] {
			c()
			delete(s.inFlight, k)
		}
	}
	now := time.Now()
	for _, k := range order {
		if len(s.inFlight) >= storeMaxFlight {
			return
		}
		if _, ok := sn.pages[k]; ok && !s.stale[k] {
			continue
		}
		if _, ok := s.inFlight[k]; ok {
			continue
		}
		if t, ok := s.failed[k]; ok && now.Sub(t) < storeRetry {
			continue
		}
		ctx, cancel := context.WithCancel(context.Background())
		s.inFlight[k] = cancel
		go s.fetch(ctx, s.gen, s.q, k)
	}
}

func (s *rowStore) fetch(ctx context.Context, gen uint64, q core.Query, k int) {
	start := time.Now()
	page, err := s.svc.Page(ctx, core.PageRequest{Query: q, Offset: k * storePageSize, Limit: storePageSize})
	elapsed := time.Since(start)

	s.mu.Lock()
	if gen != s.gen || s.closed {
		s.mu.Unlock()
		return
	}
	if ctx.Err() != nil { // cancelled because the page left the window
		s.mu.Unlock()
		return
	}
	if c, ok := s.inFlight[k]; ok {
		c()
		delete(s.inFlight, k)
	}
	if err != nil {
		s.failed[k] = time.Now()
		time.AfterFunc(storeRetry, func() {
			s.mu.Lock()
			if gen == s.gen && !s.closed {
				s.scheduleLocked()
			}
			s.mu.Unlock()
		})
		s.scheduleLocked()
		s.mu.Unlock()
		return
	}
	delete(s.failed, k)
	delete(s.stale, k)
	s.clock++
	if _, ok := s.touched[k]; !ok {
		s.touched[k] = s.clock - 1
	}
	old := s.snap.Load()
	pages := make(map[int][]core.Item, len(old.pages)+1)
	maxPage := (page.Total - 1) / storePageSize
	for pk, pg := range old.pages {
		if pk <= maxPage {
			pages[pk] = pg
		}
	}
	pages[k] = page.Items
	changedTotal := old.total != page.Total
	s.snap.Store(&storeSnap{total: page.Total, pages: pages})
	s.lastFetch = elapsed
	s.evictLocked()
	visible := changedTotal || s.visiblePage(k)
	s.scheduleLocked()
	s.mu.Unlock()
	s.notify(storeChange{Visible: visible})
}

// evictLocked drops pages outside the window, least recently touched first,
// until at most storeMaxPages remain. Visible pages are never dropped.
func (s *rowStore) evictLocked() {
	sn := s.snap.Load()
	if len(sn.pages) <= storeMaxPages {
		return
	}
	pages := make(map[int][]core.Item, len(sn.pages))
	for k, pg := range sn.pages {
		pages[k] = pg
	}
	for len(pages) > storeMaxPages {
		victim, found := 0, false
		var oldest uint64
		for k := range pages {
			if s.visiblePage(k) {
				continue
			}
			if t := s.touched[k]; !found || t < oldest {
				victim, oldest, found = k, t, true
			}
		}
		if !found {
			break
		}
		delete(pages, victim)
		delete(s.touched, victim)
		delete(s.stale, victim)
	}
	s.snap.Store(&storeSnap{total: sn.total, pages: pages})
}

// notify coalesces change notifications to at most one per storeNotifyGap.
// The first one after a quiet period fires at once; the rest are merged into
// one trailing call.
func (s *rowStore) notify(c storeChange) {
	if s.onChange == nil {
		return
	}
	s.notifyMu.Lock()
	s.pendingChange.Visible = s.pendingChange.Visible || c.Visible
	s.pendingAny = true
	if s.notifyTimer != nil { // trailing call already scheduled
		s.notifyMu.Unlock()
		return
	}
	wait := storeNotifyGap - time.Since(s.lastNotify)
	if wait <= 0 {
		s.fireLocked()
		return
	}
	s.notifyTimer = time.AfterFunc(wait, func() {
		s.notifyMu.Lock()
		s.notifyTimer = nil
		s.fireLocked()
	})
	s.notifyMu.Unlock()
}

// fireLocked releases notifyMu and calls onChange with the merged change.
func (s *rowStore) fireLocked() {
	c := s.pendingChange
	s.pendingChange = storeChange{}
	s.pendingAny = false
	s.lastNotify = time.Now()
	s.notifyMu.Unlock()
	s.onChange(c)
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
