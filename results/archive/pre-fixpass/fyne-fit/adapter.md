# Fyne data layer, second pass

## Design

The prescribed `core.Pager` is replaced by `rowStore` (`apps/fyne/adapter_store.go`), a windowed, lazily fetching row model shaped for how Fyne pulls data. Fyne's `widget.Table` and `widget.List` are pull based: they call `Length`, then `UpdateCell` / `UpdateItem` once per visible cell on the UI thread, and only redraw when `Refresh()` is called. The store is built around that.

- **Lock free reads.** `Total`, `Row` and `RangeLoaded` read an immutable snapshot (`atomic.Pointer`, copy on write, at most about 45 pages). The per-cell callbacks, which run roughly 250 times per scroll step in the table, never take a mutex or wait on a fetch.
- **Direction aware window.** `SetVisible` keeps a smoothed scroll velocity. The wanted page window is the viewport plus 50 rows, widened up to 1,000 rows ahead of the scroll direction in proportion to speed. Pages are requested visible first, then ahead of the scroll before behind it.
- **Cancellable fetches.** Each service call has its own context. When the window moves, calls for pages that left it are cancelled, so a jump or fast scroll does not queue behind obsolete requests. Four calls may run at once (the pager allowed two).
- **Coalesced, classified notifications.** Change callbacks are limited to one per 16 ms (leading edge immediately, the rest merged into one trailing call) and carry `Visible`, which is true only when the arriving page intersects the visible range or the total changed. Prefetch arrivals only touch the status bar.
- **Refresh only when stale.** The UI keeps a `viewStale` flag, set when visible data changed or a placeholder was drawn. `Table.Refresh()` / `List.Refresh()` run only then. On the scroll path (`refreshIfStale`) a range that was fully loaded when Fyne drew it is not redrawn again.
- **Cheaper per-cell hook.** Cells used to re-read the scroll offset (through the private `Table.offset` field) on every cell update. Now they do so only when a cell outside the known visible range is drawn, which is exactly when the view has scrolled.

Toolkit APIs relied on: `fyne.Do` (UI thread hop), `widget.Table.Length/UpdateCell/Refresh/ScrollToOffset`, `widget.List.Length/UpdateItem/Refresh/ScrollToOffset/GetScrollOffset`.

Limits kept: 200 rows per `Page` call, at most 40 cached pages (8,000 rows) plus 4 in flight (under 10,000), the whole dataset is never enumerated, all rows come from `core.DataService` (also through the remote client in `web-remote`), `core/` untouched.

## Before and after

| Change | First app | Now | Reason |
|---|---|---|---|
| Row reads | `Pager.Row` takes a mutex per call | atomic snapshot, no lock | cell callbacks are the hottest path and shared a lock with fetch completion |
| Prefetch | symmetric 50 row margin | velocity scaled, ahead of scroll | fast scrolling outran the fixed margin |
| Obsolete requests | run to completion, count against 2 slots | cancelled when the page leaves the window | jump and sort wait less |
| Concurrency | 2 in flight | 4 in flight | 50 ms latency per call; more overlap |
| Notifications | one UI hop per page arrival, any page | at most one per frame, visible-only triggers a view refresh | prefetch pages caused full redraws of unchanged cells |
| Bench scroll step | `ScrollToOffset` then full `Table.Refresh()` every step | refresh only if stale | the second redraw doubled the per-step UI cost |
| Per-cell range sync | offset read for every cell | only for cells outside known range | fewer private-field reads and size computations |
| Status bar | updated every arrival | same, but not tied to a view refresh | cheap |

## Files

- `apps/fyne/adapter_store.go` (new, shared): the store.
- `apps/fyne/ui.go`: constructs the store, `pagerChanged`, `afterPagerChange`, `refreshViews`, `refreshIfStale`, `syncIfOutside`, `viewStale`.
- `apps/fyne/bench_bench.go`: `ScrollToRow` uses `refreshIfStale`.
- Unchanged: `core/`, tests, Makefile.

## Limits

- Each table cell is still a Fyne widget refreshed through `Refresh()` (10 columns by about 25 rows per step). That draw cost, not data, now dominates the table scroll step; avoiding it would need a custom-drawn table, which the styling rule forbids.
- The first sort by a field builds an index inside the core (about 250 ms of the `sort` metric); the latency is simulated in the core and cannot be changed.
- A cheaper unchanged-cell skip in `tableCell.Set` was tried and removed, because skipping `Refresh` risks stale colors after a theme switch.
- Row totals are only known after the first page of a query; an extra one-row probe would be an additional service call, so it was not added.
- `placeholder_steps` was already 0 in the first app; there is nothing for the store to improve there, only to keep at 0 while the UI steps got faster.

## Benchmarks (indicative only, shared machine)

| metric | first app | now |
|---|---|---|
| ready ms | 773 | 580 to 612 |
| jump ms | 135 | 115 to 134 |
| scroll total_ms | 7076 | 4254 to 4669 |
| scroll p95 / max gap ms | 31 / 37 | 21 / 31 to 36 |
| scroll placeholder_steps | 0 | 0 |
| cards_scroll switch_ms | 104 | 64 to 73 |
| cards_scroll total_ms | 3210 | 3205 to 3215 (at the 200 x 16 ms floor) |
| cards placeholder_steps | 0 | 0 |
| sort ms | 288 | 279 |
| search ms | 58 | 61 |
| export ms | 524 | 503 |

Two bench rounds were run after the initial implementation (limit was three). `make test`: `GGUIX_TESTS passed=12 failed=0 not_feasible=0`. `make web` was started and served HTTP 200 (on port 8097, since 8090 may be taken); the `remote` wasm build compiles.
