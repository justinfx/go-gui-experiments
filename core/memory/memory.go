// Package memory provides an in-process core.DataService over a
// deterministic generated dataset (default 1,000,000 rows).
//
// Rows are stored column-wise (about 25 bytes per row). Sort indexes are
// built lazily on first use of a sort field and direction and then kept up
// to date incrementally on Update. Filtered results are cached per query
// (small LRU) and dropped on Update.
package memory

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"time"

	"gguix/core"
)

// Options configure a Service.
type Options struct {
	// Rows is the number of generated rows (0 means 1,000,000; max 9,999,999).
	Rows int
	// Latency is added once per Page, Get and Update call, after the work
	// is done. It is used as given: 0 means no latency.
	Latency time.Duration
	// ExportRowsPerSec throttles Export (0 means 100,000; negative means
	// unthrottled). Use a low value such as 200 in tests that cancel exports.
	ExportRowsPerSec int
	// Seed selects the dataset (0 means 1).
	Seed uint64
}

// DefaultOptions returns Rows 1,000,000, Latency 50 ms,
// ExportRowsPerSec 100,000, Seed 1.
func DefaultOptions() Options {
	return Options{Rows: 1_000_000, Latency: 50 * time.Millisecond, ExportRowsPerSec: 100_000, Seed: 1}
}

// MaxRows is the largest supported row count (SKUs have 7 digits).
const MaxRows = 9_999_999

var (
	baseUpdated = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	baseRestock = time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
)

// Service is an in-memory core.DataService. It is safe for concurrent use.
type Service struct {
	opts Options
	n    int

	// Columns, index = id-1.
	name     []uint32 // adjective<<16 | material<<8 | noun
	cat      []uint8
	price    []int32
	qty      []int32
	status   []uint8
	rating   []uint8 // tenths
	featured []bool
	updated  []uint32 // minutes since 2025-01-01 UTC
	restock  []uint16 // days since 2026-12-01 plus 1; 0 = none
	notesA   []uint8  // sentence index plus 1; 0 = none
	notesB   []uint8

	mu      sync.Mutex
	edits   map[int64]core.Item
	indexes map[indexKey][]int32
	results map[string][]int32
	resLRU  []string
}

type indexKey struct {
	field core.SortField
	desc  bool
}

const maxCachedResults = 8

var _ core.DataService = (*Service)(nil)

// New generates the dataset and returns the service.
func New(opts Options) *Service {
	if opts.Rows <= 0 {
		opts.Rows = 1_000_000
	}
	opts.Rows = min(opts.Rows, MaxRows)
	if opts.ExportRowsPerSec == 0 {
		opts.ExportRowsPerSec = 100_000
	}
	if opts.Seed == 0 {
		opts.Seed = 1
	}
	n := opts.Rows
	s := &Service{
		opts: opts, n: n,
		name: make([]uint32, n), cat: make([]uint8, n), price: make([]int32, n), qty: make([]int32, n),
		status: make([]uint8, n), rating: make([]uint8, n), featured: make([]bool, n), updated: make([]uint32, n),
		restock: make([]uint16, n), notesA: make([]uint8, n), notesB: make([]uint8, n),
		edits: map[int64]core.Item{}, indexes: map[indexKey][]int32{}, results: map[string][]int32{},
	}
	for i := range n {
		s.generate(i)
	}
	return s
}

// Options returns the effective options.
func (s *Service) Options() Options { return s.opts }

// Rows returns the number of rows.
func (s *Service) Rows() int { return s.n }

type rng struct{ x uint64 }

func (r *rng) next() uint64 { // splitmix64
	r.x += 0x9E3779B97F4A7C15
	z := r.x
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

func (r *rng) intn(n int) int { return int(r.next() % uint64(n)) }

func (s *Service) generate(i int) {
	id := uint64(i + 1)
	g := rng{x: s.opts.Seed*0xD1B54A32D192ED03 ^ id*0x9E3779B97F4A7C15}
	g.next()
	a, m, no := g.intn(len(adjectives)), g.intn(len(materials)), g.intn(len(nouns))
	s.name[i] = uint32(a)<<16 | uint32(m)<<8 | uint32(no)
	s.cat[i] = uint8(g.intn(8))
	s.price[i] = int32((1+g.intn(1999))*100 + g.intn(100))
	switch p := g.intn(100); {
	case p < 80:
		s.status[i] = uint8(core.Active)
	case p < 92:
		s.status[i] = uint8(core.Backordered)
	default:
		s.status[i] = uint8(core.Discontinued)
	}
	if core.Status(s.status[i]) != core.Discontinued {
		s.qty[i] = int32(g.intn(1001))
	}
	s.rating[i] = uint8(10 + g.intn(41))
	s.featured[i] = g.intn(10) == 0
	s.updated[i] = uint32(g.intn(365 * 24 * 60))
	if core.Status(s.status[i]) == core.Backordered {
		s.restock[i] = uint16(1 + g.intn(120))
	}
	if g.intn(2) == 1 {
		s.notesA[i] = uint8(1 + g.intn(len(sentences)))
		if g.intn(3) == 0 {
			s.notesB[i] = uint8(1 + g.intn(len(sentences)))
		}
	}
}

func (s *Service) baseName(i int) string {
	t := s.name[i]
	return adjectives[t>>16] + " " + materials[(t>>8)&0xff] + " " + nouns[t&0xff]
}

// itemLocked returns the current item (edited or generated). id must be valid.
func (s *Service) itemLocked(id int64) core.Item {
	if e, ok := s.edits[id]; ok {
		return e
	}
	i := int(id - 1)
	it := core.Item{
		ID:         id,
		SKU:        core.SKUFor(id),
		Name:       s.baseName(i),
		Category:   core.Category(s.cat[i]),
		PriceCents: int64(s.price[i]),
		Quantity:   s.qty[i],
		Status:     core.Status(s.status[i]),
		Rating:     core.RatingFromTenths(int(s.rating[i])),
		Featured:   s.featured[i],
		UpdatedAt:  time.Unix(baseUpdated+int64(s.updated[i])*60, 0).UTC(),
	}
	if r := s.restock[i]; r > 0 {
		d := core.DateOf(baseRestock.AddDate(0, 0, int(r)-1))
		it.RestockDate = &d
	}
	if a := s.notesA[i]; a > 0 {
		it.Notes = sentences[a-1]
		if b := s.notesB[i]; b > 0 {
			it.Notes += " " + sentences[b-1]
		}
	}
	return it
}

func (s *Service) valid(id int64) bool { return id >= 1 && id <= int64(s.n) }

func (s *Service) sleep(ctx context.Context) error {
	if s.opts.Latency <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(s.opts.Latency)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Page returns one page of the query result.
func (s *Service) Page(ctx context.Context, req core.PageRequest) (core.Page, error) {
	if req.Limit < 1 || req.Limit > core.MaxPageLimit {
		return core.Page{}, fmt.Errorf("limit must be 1..%d, got %d", core.MaxPageLimit, req.Limit)
	}
	if req.Offset < 0 {
		return core.Page{}, fmt.Errorf("offset must be >= 0, got %d", req.Offset)
	}
	if req.Query.SortBy != "" && !req.Query.SortBy.Valid() {
		return core.Page{}, fmt.Errorf("unknown sort field %q", req.Query.SortBy)
	}
	q := req.Query.Normalize()
	s.mu.Lock()
	r := s.resultLocked(q)
	total := r.len()
	end := min(total, req.Offset+req.Limit)
	items := make([]core.Item, 0, max(0, end-req.Offset))
	for i := req.Offset; i < end; i++ {
		items = append(items, s.itemLocked(int64(r.at(i))))
	}
	s.mu.Unlock()
	if err := s.sleep(ctx); err != nil {
		return core.Page{}, err
	}
	return core.Page{Offset: req.Offset, Items: items, Total: total, QueryKey: q.Key()}, nil
}

// Get returns one item or core.ErrNotFound.
func (s *Service) Get(ctx context.Context, id int64) (core.Item, error) {
	if !s.valid(id) {
		return core.Item{}, core.ErrNotFound
	}
	s.mu.Lock()
	it := s.itemLocked(id)
	s.mu.Unlock()
	if err := s.sleep(ctx); err != nil {
		return core.Item{}, err
	}
	return it, nil
}

// Update validates and stores an edit. ID and SKU cannot change; UpdatedAt
// is set to now (UTC, whole seconds) and the name is trimmed.
func (s *Service) Update(ctx context.Context, it core.Item) (core.Item, error) {
	if !s.valid(it.ID) {
		return core.Item{}, core.ErrNotFound
	}
	it.SKU = core.SKUFor(it.ID)
	it.Name = strings.TrimSpace(it.Name)
	if verr := core.ValidateItem(it); verr != nil {
		if err := s.sleep(ctx); err != nil {
			return core.Item{}, err
		}
		return core.Item{}, verr
	}
	it.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	if it.RestockDate != nil {
		d := *it.RestockDate
		it.RestockDate = &d
	}
	s.mu.Lock()
	s.edits[it.ID] = it
	for k, idx := range s.indexes {
		s.indexes[k] = s.reposition(idx, int32(it.ID), s.comparator(k.field, k.desc))
	}
	clear(s.results)
	s.resLRU = s.resLRU[:0]
	s.mu.Unlock()
	if err := s.sleep(ctx); err != nil {
		return core.Item{}, err
	}
	return it, nil
}

func (s *Service) reposition(idx []int32, id int32, cmpf func(a, b int32) int) []int32 {
	pos := slices.Index(idx, id)
	if pos < 0 {
		return idx
	}
	idx = slices.Delete(idx, pos, pos+1)
	at, _ := slices.BinarySearchFunc(idx, id, cmpf)
	return slices.Insert(idx, at, id)
}

// result is an ordered list of item IDs.
type result struct {
	ids      []int32
	identity bool // ids 1..n (or n..1 when desc), ids is nil
	desc     bool
	n        int
}

func (r result) len() int {
	if r.identity {
		return r.n
	}
	return len(r.ids)
}

func (r result) at(i int) int32 {
	if r.identity {
		if r.desc {
			return int32(r.n - i)
		}
		return int32(i + 1)
	}
	return r.ids[i]
}

func (s *Service) resultLocked(q core.Query) result {
	order := s.orderLocked(q.SortBy, q.Desc)
	if !q.Filtered() {
		return order
	}
	key := q.Key()
	if ids, ok := s.results[key]; ok {
		s.touchResult(key)
		return result{ids: ids}
	}
	m := newMatcher(q)
	out := make([]int32, 0, 1024)
	var buf []byte
	for i := range order.len() {
		id := order.at(i)
		var ok bool
		if e, edited := s.edits[int64(id)]; edited {
			ok = q.Matches(e)
		} else {
			ok, buf = s.matchBase(m, int(id-1), buf)
		}
		if ok {
			out = append(out, id)
		}
	}
	s.results[key] = out
	s.touchResult(key)
	for len(s.resLRU) > maxCachedResults {
		delete(s.results, s.resLRU[0])
		s.resLRU = s.resLRU[1:]
	}
	return result{ids: out}
}

func (s *Service) touchResult(key string) {
	if i := slices.Index(s.resLRU, key); i >= 0 {
		s.resLRU = slices.Delete(s.resLRU, i, i+1)
	}
	s.resLRU = append(s.resLRU, key)
}

func (s *Service) orderLocked(f core.SortField, desc bool) result {
	if f == core.SortID || f == core.SortSKU || f == "" {
		return result{identity: true, desc: desc, n: s.n}
	}
	k := indexKey{f, desc}
	idx, ok := s.indexes[k]
	if !ok {
		idx = make([]int32, s.n)
		for i := range idx {
			idx[i] = int32(i + 1)
		}
		slices.SortFunc(idx, s.comparator(f, desc))
		s.indexes[k] = idx
	}
	return result{ids: idx}
}

// comparator orders IDs by field (descending if desc), ties by ID ascending.
// It must be called with s.mu held.
func (s *Service) comparator(f core.SortField, desc bool) func(a, b int32) int {
	return func(a, b int32) int {
		var c int
		if f == core.SortName {
			c = s.nameCmp(a, b)
		} else {
			c = cmp.Compare(s.numKey(f, a), s.numKey(f, b))
		}
		if desc {
			c = -c
		}
		if c != 0 {
			return c
		}
		return cmp.Compare(a, b)
	}
}

func (s *Service) nameCmp(a, b int32) int {
	if len(s.edits) > 0 {
		ea, okA := s.edits[int64(a)]
		eb, okB := s.edits[int64(b)]
		if okA || okB {
			na, nb := ea.Name, eb.Name
			if !okA {
				na = s.baseName(int(a - 1))
			}
			if !okB {
				nb = s.baseName(int(b - 1))
			}
			return strings.Compare(na, nb)
		}
	}
	return cmp.Compare(s.name[a-1], s.name[b-1])
}

func (s *Service) numKey(f core.SortField, id int32) int64 {
	if len(s.edits) > 0 {
		if e, ok := s.edits[int64(id)]; ok {
			switch f {
			case core.SortCategory:
				return int64(e.Category)
			case core.SortPrice:
				return e.PriceCents
			case core.SortQuantity:
				return int64(e.Quantity)
			case core.SortStatus:
				return int64(e.Status)
			case core.SortRating:
				return int64(core.RatingTenths(e.Rating))
			case core.SortUpdated:
				return e.UpdatedAt.Unix()
			case core.SortFeatured:
				return boolKey(e.Featured)
			}
			return int64(id)
		}
	}
	i := id - 1
	switch f {
	case core.SortCategory:
		return int64(s.cat[i])
	case core.SortPrice:
		return int64(s.price[i])
	case core.SortQuantity:
		return int64(s.qty[i])
	case core.SortStatus:
		return int64(s.status[i])
	case core.SortRating:
		return int64(s.rating[i])
	case core.SortUpdated:
		return baseUpdated + int64(s.updated[i])*60
	case core.SortFeatured:
		return boolKey(s.featured[i])
	}
	return int64(id)
}

func boolKey(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

type matcher struct {
	anyCat   bool
	cats     [8]bool
	status   int // -1 = any
	needle   []byte
	checkSKU bool
}

func newMatcher(q core.Query) matcher {
	m := matcher{anyCat: len(q.Categories) == 0, status: -1}
	for _, c := range q.Categories {
		if c.Valid() {
			m.cats[c] = true
		}
	}
	if q.Status != nil {
		m.status = int(*q.Status)
	}
	m.needle = []byte(strings.ToLower(strings.TrimSpace(q.Search)))
	m.checkSKU = len(m.needle) > 0 && strings.Trim(string(m.needle), "sku-0123456789") == ""
	return m
}

func (s *Service) matchBase(m matcher, i int, buf []byte) (bool, []byte) {
	if !m.anyCat && !m.cats[s.cat[i]] {
		return false, buf
	}
	if m.status >= 0 && int(s.status[i]) != m.status {
		return false, buf
	}
	if len(m.needle) == 0 {
		return true, buf
	}
	t := s.name[i]
	buf = append(buf[:0], adjectivesLower[t>>16]...)
	buf = append(buf, ' ')
	buf = append(buf, materialsLower[(t>>8)&0xff]...)
	buf = append(buf, ' ')
	buf = append(buf, nounsLower[t&0xff]...)
	if bytes.Contains(buf, m.needle) {
		return true, buf
	}
	if m.checkSKU {
		buf = fmt.Appendf(buf[:0], "sku-%07d", i+1)
		if bytes.Contains(buf, m.needle) {
			return true, buf
		}
	}
	return false, buf
}

// Export streams the query result as CSV. See core.DataService.
func (s *Service) Export(ctx context.Context, q core.Query, w io.Writer, progress func(done, total int)) (int, error) {
	if progress == nil {
		progress = func(int, int) {}
	}
	q = q.Normalize()
	s.mu.Lock()
	r := s.resultLocked(q)
	if !r.identity {
		r.ids = slices.Clone(r.ids) // indexes change in place on Update
	}
	s.mu.Unlock()
	total := r.len()

	bw := bufio.NewWriterSize(w, 64<<10)
	header := core.CSVHeaderLine()
	crc := core.ChecksumUpdate(0, []byte(header))
	if _, err := bw.WriteString(header + "\n"); err != nil {
		return 0, err
	}
	rps := s.opts.ExportRowsPerSec
	chunk := 1000
	if rps > 0 {
		chunk = max(1, rps/100)
	}
	start := time.Now()
	lastProgress := start
	progress(0, total)
	rows := make([]string, 0, chunk)
	done := 0
	for done < total {
		if err := ctx.Err(); err != nil {
			_ = bw.Flush()
			return done, err
		}
		end := min(done+chunk, total)
		rows = rows[:0]
		s.mu.Lock()
		for i := done; i < end; i++ {
			rows = append(rows, core.ItemCSVRow(s.itemLocked(int64(r.at(i)))))
		}
		s.mu.Unlock()
		for _, row := range rows {
			crc = core.ChecksumUpdate(crc, []byte(row))
			bw.WriteString(row)
			if err := bw.WriteByte('\n'); err != nil {
				return done, err
			}
		}
		done = end
		if rps > 0 {
			target := start.Add(time.Duration(float64(done) / float64(rps) * float64(time.Second)))
			if d := time.Until(target); d > 0 {
				t := time.NewTimer(d)
				select {
				case <-t.C:
				case <-ctx.Done():
					t.Stop()
					_ = bw.Flush()
					return done, ctx.Err()
				}
			}
		}
		if time.Since(lastProgress) >= 100*time.Millisecond {
			lastProgress = time.Now()
			progress(done, total)
		}
	}
	if _, err := fmt.Fprintf(bw, "# checksum %08x\n", crc); err != nil {
		return done, err
	}
	if err := bw.Flush(); err != nil {
		return done, err
	}
	progress(total, total)
	return total, nil
}
