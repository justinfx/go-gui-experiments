package main

// adapter.go is the data layer of the second pass. It replaces
// core.Pager with rowStore, one row cache shared by the DataGrid data
// source (adapter_grid.go) and the cards VirtualList.
//
//   - Rows are held in 200-row blocks (one block = one service call, the
//     fetch limit). The LRU holds at most storeMaxBlocks blocks, plus the
//     one page the DataGrid shows (gridPage), together at most 10,000 rows.
//   - The cards view asks with SetVisible (non-blocking, like the pager);
//     missing blocks are fetched in parallel, visible first.
//   - The DataGrid asks with Fetch, called from the grid's own fetch
//     goroutine: it fills the requested page from cached blocks and fetches
//     the missing ones in parallel, then the grid applies the page in one
//     step (one refresh per page instead of one per 200 rows).
//   - Change notifications are coalesced: block arrivals within a few
//     milliseconds wake the UI once.

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"gguix/core"
)

const (
	blockRows      = 200 // rows per service call (fetch limit)
	storeMaxBlocks = 25  // 5,000 rows in the LRU
	gridPageRows   = 5000
	storeInFlight  = 6
	prefetchRows   = 50
	notifyDelay    = 4 * time.Millisecond
)

type storeBlock struct {
	items   []core.Item
	touched uint64
	stale   bool
}

type storeStats struct {
	calls  atomic.Int64 // service Page calls
	wakes  atomic.Int64 // coalesced UI wakes
	blocks atomic.Int64 // block arrivals (what the old pager woke for)
}

// rowStore is the app's row cache. All methods are safe from any goroutine.
type rowStore struct {
	svc      core.DataService
	onChange func()

	mu        sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	gen       uint64
	q         core.Query
	total     int
	started   bool
	active    bool // the cards view is shown: SetVisible drives fetches
	first     int
	last      int
	blocks    map[int]*storeBlock
	inFlight  map[int]bool
	failed    map[int]time.Time
	clock     uint64
	lastFetch time.Duration

	// gridPage is the page the DataGrid currently holds (set by Fetch).
	gridStart int
	gridItems []core.Item
	gridGen   uint64
	gridReq   uint64

	notifyPending atomic.Bool
	stats         storeStats
}

func newRowStore(svc core.DataService, onChange func()) *rowStore {
	s := &rowStore{svc: svc, onChange: onChange, total: -1, gridStart: -1}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.blocks = map[int]*storeBlock{}
	s.inFlight = map[int]bool{}
	s.failed = map[int]time.Time{}
	return s
}

// SetQuery drops everything and starts the new query.
func (s *rowStore) SetQuery(q core.Query) {
	s.mu.Lock()
	s.cancel()
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.gen++
	s.q = q
	s.total = -1
	s.started = true
	s.blocks = map[int]*storeBlock{}
	s.inFlight = map[int]bool{}
	s.failed = map[int]time.Time{}
	s.gridItems, s.gridStart = nil, -1
	s.scheduleLocked()
	s.mu.Unlock()
	s.notify()
}

func (s *rowStore) Query() core.Query {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.q
}

// SetVisible is the cards view (and go-to before the grid has a page)
// telling the store what is on screen. It never blocks.
func (s *rowStore) SetVisible(first, last int) {
	s.mu.Lock()
	s.first, s.last = first, last
	for k := first / blockRows; k <= last/blockRows; k++ {
		if b := s.blocks[k]; b != nil {
			s.clock++
			b.touched = s.clock
		}
	}
	s.scheduleLocked()
	s.mu.Unlock()
}

func (s *rowStore) Total() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.total
}

func (s *rowStore) Row(i int) (core.Item, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rowLocked(i)
}

func (s *rowStore) rowLocked(i int) (core.Item, bool) {
	if i < 0 || (s.total >= 0 && i >= s.total) {
		return core.Item{}, false
	}
	if s.gridStart >= 0 && i >= s.gridStart && i < s.gridStart+len(s.gridItems) {
		return s.gridItems[i-s.gridStart], true
	}
	b := s.blocks[i/blockRows]
	if b == nil || i%blockRows >= len(b.items) {
		return core.Item{}, false
	}
	return b.items[i%blockRows], true
}

func (s *rowStore) RangeLoaded(first, last int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.total < 0 {
		return false
	}
	last = min(last, s.total-1)
	for i := first; i <= last; i++ {
		if _, ok := s.rowLocked(i); !ok {
			return false
		}
	}
	return true
}

// GridPage reports the page the DataGrid holds: start row and length
// (-1, 0 when none for the current query).
func (s *rowStore) GridPage() (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gridStart, len(s.gridItems)
}

// GridIndexOf finds item id in the grid page; returns the global row.
func (s *rowStore) GridIndexOf(id int64) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, it := range s.gridItems {
		if it.ID == id {
			return s.gridStart + i, true
		}
	}
	return 0, false
}

// Invalidate marks every block stale (they keep serving) and refetches
// the visible ones; the grid page is refetched by the grid itself.
func (s *rowStore) Invalidate() {
	s.mu.Lock()
	s.gen++
	s.inFlight = map[int]bool{}
	for k, b := range s.blocks {
		if k >= s.first/blockRows && k <= s.last/blockRows {
			b.stale = true
		} else {
			delete(s.blocks, k)
		}
	}
	s.scheduleLocked()
	s.mu.Unlock()
}

// PatchItem replaces one item wherever it is held (after Update), so the
// row shows the saved values before the refetch arrives.
func (s *rowStore) PatchItem(it core.Item) {
	s.mu.Lock()
	for i := range s.gridItems {
		if s.gridItems[i].ID == it.ID {
			s.gridItems[i] = it
		}
	}
	for _, b := range s.blocks {
		for i := range b.items {
			if b.items[i].ID == it.ID {
				b.items[i] = it
			}
		}
	}
	s.mu.Unlock()
}

func (s *rowStore) Stats() core.PagerStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.blocks)
	if s.gridStart >= 0 {
		n += (len(s.gridItems) + blockRows - 1) / blockRows
	}
	return core.PagerStats{LastFetch: s.lastFetch, CachedPages: n, InFlight: len(s.inFlight)}
}

func (s *rowStore) Close() {
	s.mu.Lock()
	s.cancel()
	s.started = false
	s.mu.Unlock()
}

// HeldRows is the number of rows in memory (LRU plus grid page).
func (s *rowStore) HeldRows() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.gridItems)
	for _, b := range s.blocks {
		n += len(b.items)
	}
	return n
}

// SetActive turns SetVisible-driven fetching on (cards view) or off
// (table view, where the DataGrid asks through Fetch).
func (s *rowStore) SetActive(on bool) {
	s.mu.Lock()
	s.active = on
	s.scheduleLocked()
	s.mu.Unlock()
}

func (s *rowStore) scheduleLocked() {
	if !s.started || !s.active || s.total == 0 {
		return
	}
	anchor := s.first / blockRows
	var order []int
	if s.total < 0 {
		order = []int{anchor}
	} else {
		lo := max(0, s.first-prefetchRows)
		hi := min(s.total-1, s.last+prefetchRows)
		if hi < lo {
			return
		}
		var vis, pre []int
		for k := lo / blockRows; k <= hi/blockRows; k++ {
			if k*blockRows <= s.last && (k+1)*blockRows-1 >= s.first {
				vis = append(vis, k)
			} else {
				pre = append(pre, k)
			}
		}
		byDist := func(a, b int) int {
			return cmp.Or(cmp.Compare(abs(a-anchor), abs(b-anchor)), cmp.Compare(a, b))
		}
		slices.SortFunc(vis, byDist)
		slices.SortFunc(pre, byDist)
		order = append(vis, pre...)
	}
	now := time.Now()
	for _, k := range order {
		if len(s.inFlight) >= storeInFlight {
			return
		}
		if b := s.blocks[k]; b != nil && !b.stale {
			continue
		}
		if s.inGridLocked(k) || s.inFlight[k] {
			continue
		}
		if t, ok := s.failed[k]; ok && now.Sub(t) < time.Second {
			continue
		}
		s.inFlight[k] = true
		go s.fetchBlock(s.ctx, s.gen, s.q, k)
	}
}

// inGridLocked: the block is fully covered by the grid page.
func (s *rowStore) inGridLocked(k int) bool {
	return s.gridStart >= 0 && k*blockRows >= s.gridStart && (k+1)*blockRows <= s.gridStart+len(s.gridItems)
}

func (s *rowStore) page(ctx context.Context, q core.Query, off int) (core.Page, time.Duration, error) {
	s.stats.calls.Add(1)
	t0 := time.Now()
	p, err := s.svc.Page(ctx, core.PageRequest{Query: q, Offset: off, Limit: blockRows})
	return p, time.Since(t0), err
}

func (s *rowStore) fetchBlock(ctx context.Context, gen uint64, q core.Query, k int) {
	p, el, err := s.page(ctx, q, k*blockRows)
	s.mu.Lock()
	if gen != s.gen {
		s.mu.Unlock()
		return
	}
	delete(s.inFlight, k)
	if err != nil {
		s.failed[k] = time.Now()
		time.AfterFunc(time.Second, func() {
			s.mu.Lock()
			if gen == s.gen {
				s.scheduleLocked()
			}
			s.mu.Unlock()
		})
		s.mu.Unlock()
		return
	}
	delete(s.failed, k)
	s.putLocked(k, p.Items)
	s.total = p.Total
	s.lastFetch = el
	s.scheduleLocked()
	s.mu.Unlock()
	s.stats.blocks.Add(1)
	s.notify()
}

func (s *rowStore) putLocked(k int, items []core.Item) {
	s.clock++
	s.blocks[k] = &storeBlock{items: items, touched: s.clock}
	if s.total >= 0 {
		maxK := (s.total - 1) / blockRows
		for bk := range s.blocks {
			if bk > maxK {
				delete(s.blocks, bk)
			}
		}
	}
	for len(s.blocks) > storeMaxBlocks {
		victim, oldest := -1, ^uint64(0)
		for bk, b := range s.blocks {
			vis := bk*blockRows <= s.last && (bk+1)*blockRows-1 >= s.first
			if !vis && b.touched < oldest {
				victim, oldest = bk, b.touched
			}
		}
		if victim < 0 {
			return
		}
		delete(s.blocks, victim)
	}
}

// Fetch returns rows [start, end) of query q for the DataGrid, blocking
// until every block is there. Cached, fresh blocks of the same query are
// reused; the rest are fetched in parallel, one service call per block.
// reqID orders concurrent grid requests: only the newest becomes the
// grid page.
func (s *rowStore) Fetch(ctx context.Context, q core.Query, start, end int, reqID uint64) ([]core.Item, int, error) {
	s.mu.Lock()
	same := s.q.Key() == q.Key()
	gen := s.gen
	total := s.total
	s.mu.Unlock()
	if total >= 0 {
		end = min(end, total)
	}
	if end <= start && total >= 0 {
		return nil, total, nil
	}
	k0, k1 := start/blockRows, max(start, end-1)/blockRows
	parts := make([][]core.Item, k1-k0+1)
	var need []int
	s.mu.Lock()
	for k := k0; k <= k1; k++ {
		if b := s.blocks[k]; same && b != nil && !b.stale {
			parts[k-k0] = b.items
		} else {
			need = append(need, k)
		}
	}
	s.mu.Unlock()
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var firstErr error
	var lastEl time.Duration
	for _, k := range need {
		wg.Add(1)
		go func(k int) {
			defer wg.Done()
			p, el, err := s.page(ctx, q, k*blockRows)
			errMu.Lock()
			defer errMu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			parts[k-k0] = p.Items
			total = p.Total
			lastEl = el
		}(k)
	}
	wg.Wait()
	if firstErr != nil {
		return nil, 0, firstErr
	}
	out := make([]core.Item, 0, end-start)
	for i, p := range parts {
		base := (k0 + i) * blockRows
		for j, it := range p {
			if r := base + j; r >= start && r < end {
				out = append(out, it)
			}
		}
	}
	s.mu.Lock()
	if same && gen == s.gen && reqID >= s.gridReq && ctx.Err() == nil {
		s.gridReq = reqID
		s.gridStart, s.gridItems, s.gridGen = start, out, gen
		s.total = total
		if lastEl > 0 {
			s.lastFetch = lastEl
		}
		// Blocks now covered by the grid page are dropped from the LRU so
		// a row is held once.
		for k := range s.blocks {
			if s.inGridLocked(k) {
				delete(s.blocks, k)
			}
		}
	}
	s.mu.Unlock()
	return out, total, nil
}

// notify wakes the UI once for a burst of arrivals.
func (s *rowStore) notify() {
	if s.onChange == nil || !s.notifyPending.CompareAndSwap(false, true) {
		return
	}
	time.AfterFunc(notifyDelay, func() {
		s.notifyPending.Store(false)
		s.stats.wakes.Add(1)
		s.onChange()
	})
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
