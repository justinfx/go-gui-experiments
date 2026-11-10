package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	gc "gguix/core"
	"gguix/core/memory"
)

// limitSvc records the largest Limit asked of the service.
type limitSvc struct {
	gc.DataService
	maxLimit atomic.Int64
}

func (l *limitSvc) Page(ctx context.Context, r gc.PageRequest) (gc.Page, error) {
	for {
		cur := l.maxLimit.Load()
		if int64(r.Limit) <= cur || l.maxLimit.CompareAndSwap(cur, int64(r.Limit)) {
			break
		}
	}
	return l.DataService.Page(ctx, r)
}

// The model keeps to the fetch limits while scrolling a long way.
func TestModelLimits(t *testing.T) {
	record(t)
	svc := &limitSvc{DataService: memory.New(memory.Options{Rows: 100000, Latency: time.Millisecond})}
	m := newRowModel(svc, func() {})
	m.SetQuery(gc.Query{})
	for i := 0; i < 400 && m.Total() < 0; i++ {
		time.Sleep(5 * time.Millisecond)
	}
	for k := 0; k < 300; k++ {
		m.SetVisible(k*40, k*40+30)
		time.Sleep(2 * time.Millisecond)
		if st := m.Stats(); (st.CachedPages+st.InFlight)*modelPageSize > 10000 {
			t.Fatalf("holds %d pages cached and %d in flight", st.CachedPages, st.InFlight)
		}
	}
	if svc.maxLimit.Load() > 200 {
		t.Errorf("limit %d > 200", svc.maxLimit.Load())
	}
	m.Close()
}
