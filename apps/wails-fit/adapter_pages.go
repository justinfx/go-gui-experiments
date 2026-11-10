package main

// Data layer, Go side (category: shared). The frontend row store
// (frontend/src/lib/data/) asks for several page ranges at once. PageBatch
// fans them out to the core data service in parallel goroutines, so one IPC
// round trip costs one simulated service latency instead of one per range.
// Each range is still its own DataService.Page call of at most maxRangeRows rows.

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"

	"gguix/core"
)

const (
	maxRangeRows   = 200 // fetch limit per service call
	maxBatchRanges = 8   // ranges per PageBatch call
)

// serviceCalls counts DataService.Page calls made for the view (bench aid).
var (
	serviceCalls atomic.Int64
	batchCalls   atomic.Int64
)

// RangeArg is one window of rows in a query result.
type RangeArg struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// BatchArg is a query plus the row windows to fetch for it.
type BatchArg struct {
	Query  QueryArg   `json:"query"`
	Ranges []RangeArg `json:"ranges"`
}

// PageBatch returns one PageView per requested range, in order. The ranges are
// fetched concurrently. If any range fails the whole call fails.
func (s *ItemService) PageBatch(req BatchArg) ([]PageView, error) {
	if len(req.Ranges) == 0 || len(req.Ranges) > maxBatchRanges {
		return nil, fmt.Errorf("PageBatch needs 1 to %d ranges", maxBatchRanges)
	}
	q, err := req.Query.ToQuery()
	if err != nil {
		return nil, err
	}
	batchCalls.Add(1)
	out := make([]PageView, len(req.Ranges))
	errs := make([]error, len(req.Ranges))
	var wg sync.WaitGroup
	for i, r := range req.Ranges {
		if r.Limit < 1 || r.Limit > maxRangeRows || r.Offset < 0 {
			return nil, fmt.Errorf("bad range %d+%d (limit is 1 to %d)", r.Offset, r.Limit, maxRangeRows)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			serviceCalls.Add(1)
			p, err := s.svc.Page(context.Background(), core.PageRequest{Query: q, Offset: r.Offset, Limit: r.Limit})
			if err != nil {
				errs[i] = err
				return
			}
			out[i] = pageToView(p)
		}()
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// pageToView converts a core page, formatting every cell value once on the Go side.
func pageToView(p core.Page) PageView {
	v := PageView{Offset: p.Offset, Total: p.Total, TotalText: core.FormatCount(p.Total), QueryKey: p.QueryKey, Items: make([]ItemView, len(p.Items))}
	for i, it := range p.Items {
		v.Items[i] = ToView(it)
	}
	return v
}

// logServiceCalls prints the data-path call counts in bench mode.
func logServiceCalls() {
	if os.Getenv("GGUIX_BENCH") == "1" {
		log.Printf("GGUIX_INFO service_page_calls=%d batch_ipc_calls=%d", serviceCalls.Load(), batchCalls.Load())
	}
}
