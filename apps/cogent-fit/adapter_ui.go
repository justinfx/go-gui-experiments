package main

// The UI side of the data layer (second pass, see results/cogent/adapter.md):
// change coalescing onto the UI thread, and the per-pass row snapshot that
// the table and the cards list read from.

import (
	"time"

	gc "gguix/core"
)

// refreshFrame is the shortest time between two refresh passes.
const refreshFrame = 16 * time.Millisecond

// modelChanged is the row model's change callback. It runs on fetch
// goroutines and is only called when the visible rows or the total changed.
// All changes inside one frame are folded into a single refresh pass: the
// first change arms one timer, later ones only bump the generation, and the
// pass takes the UI lock (AsyncLock) once.
func (a *App) modelChanged() {
	a.gen.Add(1)
	if a.queued.Swap(true) {
		return
	}
	wait := refreshFrame - time.Since(time.Unix(0, a.lastRefresh.Load()))
	if wait < 0 {
		wait = 0
	}
	time.AfterFunc(wait, func() { a.async(a.refresh) })
}

// rowSnap is the rows of one view window, copied from the model under one
// lock at the start of an update pass. Cell updaters read it instead of
// asking the model row by row.
type rowSnap struct {
	first int
	items []gc.Item
	ok    []bool
}

// take fills the snapshot for rows [first, first+n).
func (s *rowSnap) take(m *rowModel, first, n int) {
	if cap(s.items) < n {
		s.items = make([]gc.Item, n)
		s.ok = make([]bool, n)
	}
	s.items, s.ok = s.items[:n], s.ok[:n]
	s.first = first
	m.Fill(first, s.items, s.ok)
}

// get returns row si from the snapshot, falling back to the model for rows
// outside it.
func (s *rowSnap) get(m *rowModel, si int) (gc.Item, bool) {
	if i := si - s.first; i >= 0 && i < len(s.items) {
		return s.items[i], s.ok[i]
	}
	return m.Row(si)
}
