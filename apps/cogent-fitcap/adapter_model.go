package main

// The data layer's model (second pass, see results/cogent/adapter.md).
//
// rowModel is a lazily fetching row source for the virtualized table and
// cards list. It replaces core.Pager with a design that fits how Cogent Core
// drives a core.ListBase: the list asks for the rows of its visible window on
// every update pass, never blocks, and expects to be told when something it
// shows has changed.
//
// What it does differently from core.Pager:
//   - Motion aware read-ahead. It estimates the scroll velocity from the
//     SetVisible calls and fetches ahead of the window in the direction of
//     travel, sized to cover the fetch latency, instead of a fixed 50 rows.
//   - Wider, cancellable fetching. Up to modelMaxInFlight page requests run at
//     once, and requests for pages that left the wanted window are cancelled
//     (the service honours the context) so a jump does not queue behind
//     obsolete fetches.
//   - Relevant-only change notification. The change callback fires only when
//     a page that intersects the visible window (or the pinned row) arrives,
//     or when Total changes. Read-ahead pages arriving off screen are silent,
//     so the UI is not refreshed for data nobody sees.
//   - Snapshots. Fill copies a whole window under one lock, so a view reads
//     its rows once per update pass.
//
// Limits respected: pages of 200 rows per service call, at most
// modelMaxPages*200 rows cached (plus in-flight pages) which stays under
// 10,000, and the whole dataset is never enumerated.

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	gc "gguix/core"
)

const (
	modelPageSize    = 200
	modelMaxPages    = 36 // 7,200 rows cached; with in-flight pages still under 10,000
	modelMaxInFlight = 2
	modelIdleMargin  = 50   // rows kept around the window when not moving
	modelMinAhead    = 100  // minimum read-ahead rows while moving
	modelMaxAhead    = 1400 // at most 7 pages of read-ahead
	modelFastRows    = 60   // rows per second above which the window counts as moving
	modelJumpRows    = 3000 // a window move bigger than this is a jump, not a scroll
)

type modelPage struct {
	items   []gc.Item
	touched uint64
	stale   bool
}

type modelStats struct {
	LastFetch   time.Duration
	CachedPages int
	InFlight    int
}

// rowModel is safe for concurrent use. onChange is called from background
// goroutines and must be cheap and non blocking.
type rowModel struct {
	svc      gc.DataService
	onChange func()

	mu       sync.Mutex
	gen      uint64
	ctx      context.Context
	cancel   context.CancelFunc
	q        gc.Query
	started  bool
	total    int
	pages    map[int]*modelPage
	flight   map[int]context.CancelFunc
	failed   map[int]time.Time
	first    int
	last     int
	pin      int // a row to notify about even when off screen, -1 for none
	clock    uint64
	lastTime time.Time // time of the last velocity sample
	lastFrom int       // window top at that sample
	speed    float64   // smoothed rows per second, signed
	latency  time.Duration
	lastFt   time.Duration

	// counters for the bench report (read without the lock)
	Calls     atomic.Int64 // service Page calls started
	Cancelled atomic.Int64 // calls cancelled because the window moved on
	Notifies  atomic.Int64 // change callbacks made
}

func newRowModel(svc gc.DataService, onChange func()) *rowModel {
	m := &rowModel{
		svc: svc, onChange: onChange,
		total: -1, pin: -1, last: modelPageSize - 1,
		latency: 60 * time.Millisecond,
		pages:   map[int]*modelPage{}, flight: map[int]context.CancelFunc{}, failed: map[int]time.Time{},
	}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	return m
}

// SetQuery switches query: clears the cache, cancels fetches and loads the
// first page.
func (m *rowModel) SetQuery(q gc.Query) {
	m.mu.Lock()
	m.resetLocked()
	m.q = q.Normalize()
	m.started = true
	m.total = -1
	clear(m.pages)
	height := max(0, m.last-m.first)
	m.first, m.last = 0, height
	m.speed, m.lastFrom, m.lastTime = 0, 0, time.Time{}
	m.pin = -1
	m.scheduleLocked()
	m.mu.Unlock()
	m.fire()
}

// Query returns the current normalized query.
func (m *rowModel) Query() gc.Query {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.q
}

// SetVisible reports the inclusive window the view shows. It never calls
// onChange; it schedules the fetches the window needs.
func (m *rowModel) SetVisible(first, last int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	first = max(0, first)
	if last < first {
		last = first
	}
	m.sampleLocked(first)
	m.first, m.last = first, last
	m.clock++
	for k := first / modelPageSize; k <= last/modelPageSize; k++ {
		if pg := m.pages[k]; pg != nil {
			pg.touched = m.clock
		}
	}
	m.scheduleLocked()
	m.evictLocked()
}

// Pin asks for a change notification when the page holding row arrives, even
// if the row is off screen (a selected row waiting for its data). -1 clears.
func (m *rowModel) Pin(row int) {
	m.mu.Lock()
	m.pin = row
	m.mu.Unlock()
}

// sampleLocked updates the smoothed scroll velocity from the window top.
func (m *rowModel) sampleLocked(first int) {
	now := time.Now()
	if m.lastTime.IsZero() {
		m.lastTime, m.lastFrom = now, first
		return
	}
	dt := now.Sub(m.lastTime)
	if dt < 4*time.Millisecond {
		return // repeated calls within one pass; keep the older sample
	}
	d := first - m.lastFrom
	m.lastTime, m.lastFrom = now, first
	if d > modelJumpRows || d < -modelJumpRows {
		m.speed = 0
		return
	}
	if dt > time.Second {
		m.speed = 0
		return
	}
	v := float64(d) / dt.Seconds()
	m.speed = 0.6*m.speed + 0.4*v
}

// Total is the number of rows in the result, or -1 until known.
func (m *rowModel) Total() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.total
}

// Row returns row i if loaded. It never blocks on the service.
func (m *rowModel) Row(i int) (gc.Item, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.rowLocked(i)
}

func (m *rowModel) rowLocked(i int) (gc.Item, bool) {
	if i < 0 || (m.total >= 0 && i >= m.total) {
		return gc.Item{}, false
	}
	pg := m.pages[i/modelPageSize]
	if pg == nil {
		return gc.Item{}, false
	}
	j := i % modelPageSize
	if j >= len(pg.items) {
		return gc.Item{}, false
	}
	return pg.items[j], true
}

// Fill copies rows [first, first+len(items)) into items and ok under one
// lock. It is the snapshot a view reads during one update pass.
func (m *rowModel) Fill(first int, items []gc.Item, ok []bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range items {
		items[i], ok[i] = m.rowLocked(first + i)
	}
}

// RangeLoaded reports whether Total is known and rows [first, last] are all
// loaded.
func (m *rowModel) RangeLoaded(first, last int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.total < 0 {
		return false
	}
	last = min(last, m.total-1)
	for i := max(0, first); i <= last; i++ {
		if _, ok := m.rowLocked(i); !ok {
			return false
		}
	}
	return true
}

// Invalidate refetches the visible pages after an edit. Their current data
// keeps serving reads until the new data arrives; other pages are dropped.
func (m *rowModel) Invalidate() {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return
	}
	m.resetLocked()
	for k, pg := range m.pages {
		if m.visibleLocked(k) {
			pg.stale = true
		} else {
			delete(m.pages, k)
		}
	}
	m.scheduleLocked()
	m.mu.Unlock()
}

// Stats returns the cache state for the status bar.
func (m *rowModel) Stats() modelStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	return modelStats{LastFetch: m.lastFt, CachedPages: len(m.pages), InFlight: len(m.flight)}
}

// Close cancels in-flight fetches.
func (m *rowModel) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancel()
	m.started = false
}

func (m *rowModel) resetLocked() {
	m.cancel()
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.gen++
	clear(m.flight)
	clear(m.failed)
}

func (m *rowModel) visibleLocked(k int) bool {
	start, end := k*modelPageSize, (k+1)*modelPageSize-1
	return start <= m.last && end >= m.first
}

// wantedLocked returns the pages worth having, most urgent first: the
// visible pages nearest the window top, then read-ahead in the direction of
// travel, then a small margin behind.
func (m *rowModel) wantedLocked() []int {
	ps := modelPageSize
	if m.total < 0 {
		return []int{m.first / ps}
	}
	if m.total == 0 {
		return nil
	}
	ahead, behind := modelIdleMargin, modelIdleMargin
	dir := 0
	if v := m.speed; v > modelFastRows || v < -modelFastRows {
		a := v
		if a < 0 {
			a = -a
		}
		rows := int(a * (m.latency + 80*time.Millisecond).Seconds())
		ahead = max(modelMinAhead, min(rows+modelMinAhead/2, modelMaxAhead))
		if v > 0 {
			dir = 1
		} else {
			dir = -1
		}
	}
	lo, hi := m.first-modelIdleMargin, m.last+modelIdleMargin
	switch dir {
	case 1:
		lo, hi = m.first-behind, m.last+ahead
	case -1:
		lo, hi = m.first-ahead, m.last+behind
	}
	lo = max(0, lo)
	hi = min(m.total-1, hi)
	if hi < lo {
		return nil
	}
	anchor := m.first / ps
	var vis, pre []int
	for k := lo / ps; k <= hi/ps; k++ {
		if m.visibleLocked(k) {
			vis = append(vis, k)
		} else {
			pre = append(pre, k)
		}
	}
	slices.SortFunc(vis, func(a, b int) int { return cmp.Compare(absInt(a-anchor), absInt(b-anchor)) })
	// Read-ahead pages: nearest to the window first, ahead of the direction
	// of travel before behind it.
	slices.SortFunc(pre, func(a, b int) int {
		da, db := absInt(a-anchor), absInt(b-anchor)
		if dir != 0 && (a-anchor)*dir > 0 != ((b-anchor)*dir > 0) {
			if (a-anchor)*dir > 0 {
				return -1
			}
			return 1
		}
		return cmp.Or(cmp.Compare(da, db), cmp.Compare(a, b))
	})
	out := append(vis, pre...)
	if len(out) > modelMaxPages {
		out = out[:modelMaxPages]
	}
	return out
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// scheduleLocked cancels fetches no longer wanted and starts the missing
// ones, up to modelMaxInFlight at once.
func (m *rowModel) scheduleLocked() {
	if !m.started || m.total == 0 {
		return
	}
	order := m.wantedLocked()
	want := make(map[int]bool, len(order))
	for _, k := range order {
		want[k] = true
	}
	for k, cancel := range m.flight {
		if !want[k] {
			cancel()
			delete(m.flight, k)
			m.Cancelled.Add(1)
		}
	}
	now := time.Now()
	for _, k := range order {
		if len(m.flight) >= modelMaxInFlight {
			return
		}
		if pg := m.pages[k]; pg != nil && !pg.stale {
			continue
		}
		if _, ok := m.flight[k]; ok {
			continue
		}
		if t, ok := m.failed[k]; ok && now.Sub(t) < time.Second {
			continue
		}
		ctx, cancel := context.WithCancel(m.ctx)
		m.flight[k] = cancel
		m.Calls.Add(1)
		go m.fetch(ctx, m.gen, m.q, k)
	}
}

func (m *rowModel) fetch(ctx context.Context, gen uint64, q gc.Query, k int) {
	start := time.Now()
	page, err := m.svc.Page(ctx, gc.PageRequest{Query: q, Offset: k * modelPageSize, Limit: modelPageSize})
	elapsed := time.Since(start)

	m.mu.Lock()
	if gen != m.gen || ctx.Err() != nil {
		m.mu.Unlock()
		return
	}
	delete(m.flight, k)
	if err != nil {
		m.failed[k] = time.Now()
		time.AfterFunc(time.Second, func() {
			m.mu.Lock()
			if gen == m.gen {
				m.scheduleLocked()
			}
			m.mu.Unlock()
		})
		m.scheduleLocked()
		m.mu.Unlock()
		return
	}
	delete(m.failed, k)
	m.clock++
	m.pages[k] = &modelPage{items: page.Items, touched: m.clock}
	totalChanged := m.total != page.Total
	m.total = page.Total
	m.lastFt = elapsed
	m.latency = (m.latency*3 + elapsed) / 4
	maxPage := (m.total - 1) / modelPageSize
	for pk := range m.pages {
		if pk > maxPage {
			delete(m.pages, pk)
		}
	}
	relevant := totalChanged || m.visibleLocked(k) ||
		(m.pin >= 0 && m.pin/modelPageSize == k)
	m.evictLocked()
	m.scheduleLocked()
	m.mu.Unlock()
	if relevant {
		m.fire()
	}
}

func (m *rowModel) evictLocked() {
	for len(m.pages) > modelMaxPages {
		victim, found := 0, false
		var oldest uint64
		for k, pg := range m.pages {
			if m.visibleLocked(k) {
				continue
			}
			if !found || pg.touched < oldest {
				victim, oldest, found = k, pg.touched, true
			}
		}
		if !found {
			return
		}
		delete(m.pages, victim)
	}
}

func (m *rowModel) fire() {
	m.Notifies.Add(1)
	if m.onChange != nil {
		m.onChange()
	}
}
