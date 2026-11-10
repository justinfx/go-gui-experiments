# Data layer, second pass

## Design

`core.Pager` is replaced by `rowModel`, a lazily fetching row source shaped around how a Cogent Core `core.ListBase` works. `ListBase` (the base of `List` and `Table`) renders only the rows in its window, and on every update pass each cell's `Updater` asks for its row. So the model has to answer "row i, if you have it" without blocking, and tell the UI when something shown changed. The model plus a small UI adapter do exactly that, using these toolkit APIs:

- `ListBase` with a zero-size backing slice and the `Lister` hooks (`UpdateSliceSize`, `MakeRow`, `RowWidgetNs`), as in the first pass. The slice length is the model's `Total`.
- `Updater` functions. The list's own `Updater` copies the window's rows from the model once (`rowSnap.take`), and the cell updaters read that snapshot. Parents update before children in a pass, so one lock and one copy serve all cells.
- The toolkit's scroll path. The scrollbar handler (`ListGrid.ScrollChanged`) sets `StartIndex` and calls `UpdateTree` and `NeedsRender`, without rebuilding the widget plan. `scrollTop` now does the same when the row count has not changed, and only calls `Update` (plan rebuild) when `Total` changed.
- `AsyncLock` / `AsyncUnlock` to run the refresh on the UI thread, once per coalesced batch.

Model behaviour (`adapter_model.go`):

- Pages of 200 rows per service call. At most 36 cached pages (7,200 rows), at most 6 calls in flight, so memory stays under 10,000 rows. The dataset is never enumerated.
- Motion aware read-ahead. `SetVisible` samples the window top over time to get a smoothed scroll velocity. While moving, it fetches ahead in the direction of travel by `velocity x (measured latency + 80 ms)`, between 100 and 1,400 rows. When idle it keeps a 50 row margin like the pager. A jump of more than 3,000 rows resets the velocity, so a jump does not trigger a read-ahead storm.
- Cancellation. Pages in flight that fall out of the wanted set are cancelled through the context (the service honours it), which frees slots after a jump or sort.
- Relevant-only change notification. The change callback fires only when a page arriving intersects the visible window or the pinned row (a selection waiting for its data), or when `Total` changes. Read-ahead pages arriving off screen never wake the UI.
- Coalescing (`adapter_ui.go`). All changes within one 16 ms frame fold into one refresh: the first arms a single timer, later ones only bump a generation counter. The bench "loaded" check is unchanged in meaning: the range is loaded in the model and the refresh for the latest generation has run.

## Before and after

| Change | First app (`core.Pager`) | Now | Reason |
|---|---|---|---|
| Read-ahead | fixed 50 rows each side | velocity based, in the direction of travel, up to 1,400 rows | at 50 ms latency a fast scroll outruns a 50 row margin; this removed the placeholder steps |
| In-flight calls | 2, never cancelled | 6, obsolete ones cancelled | latency is per call, so concurrency is cheap; cancelling frees slots after jumps |
| Cache size | 20 pages | 36 pages | room for read-ahead, still under 10,000 rows |
| Change notifications | every page arrival, including off screen prefetch | only visible or pinned pages, or `Total` | fewer UI refreshes for data nobody sees |
| Refresh scheduling | a new goroutine and 6 ms sleep per burst | one timer per 16 ms frame, based on the last refresh time | one lock acquisition per frame |
| Cell data access | each cell calls the pager (one lock and one item copy per cell) | one snapshot per pass | fewer locks per pass |
| Scrolling by code (go to row, bench scroll) | `t.Update()` (plan rebuild) plus an extra `SetVisible` | `UpdateTree` and render, the scrollbar's own path | no widget plan rebuild per step |
| Query change | scrolled both the table and the hidden cards list with `Update` | scrolls the active list, only resets the hidden one | the hidden list is refreshed when shown |

## Files

- `adapter_model.go`: `rowModel` (cache, read-ahead, cancellation, notification rules).
- `adapter_ui.go`: change coalescing (`modelChanged`) and the `rowSnap` window snapshot.
- `adapter_model_test.go`: test that the fetch limits hold during a long scroll (counted in the `GGUIX_TESTS` line).
- Edited to use it: `app.go` (model field, refresh bookkeeping, list view interface), `table.go` and `cards.go` (snapshot, `scrollTop`, `resetTop`), `bench_bench.go` (prints a `GGUIX_ADAPTER` counter line to stderr at the end; it is not a `GGUIX_METRIC` line and the metric lines are unchanged).
- All of these are shared code and have no target-specific blocks, so they serve desktop, `web` and `web-remote`.

## Limits

- The service latency is fixed at 50 ms per call and 200 rows is the largest call, so the first view of a new position always costs at least one round trip. Only read-ahead can hide it.
- A sort on a new field builds an index inside the core and cannot be hidden. The `sort` step is dominated by that.
- `core.Table` still cannot sit on a model (it binds to reflected slices), so the table stays a `ListBase` subclass with its own header (first-pass limitation, see `friction.md`).
- I did not use a second fetch tier of smaller calls for the first visible rows: latency is per call, not per row, so it would not help.
- Could not run `make web` serving in this session because another process held port 8090. Both wasm targets (default and `remote`) compile; serving is unchanged from the first app.

## Benchmarks (indicative only)

Desktop `make bench` style run on a shared machine (binary run directly, window session not visible), two runs of the new layer against the first app's single run:

| Metric | First app | New run 1 | New run 2 |
|---|---|---|---|
| ready ms | 915 | 649 | 588 |
| jump ms | 60 | 84 | 79 |
| scroll placeholder_steps | 24 | 0 | 0 |
| scroll max_gap_ms / p95 | 18.6 / 4.1 | 18 / 3.9 | 22.6 / 4.5 |
| cards_scroll placeholder_steps | 1 | 0 | 0 |
| cards switch_ms | 109 | 56 | 79 |
| sort ms | 344 | 264 | 294 |
| search ms | 64 | 59 | 66 |
| export ms | 507 | 507 | 500 |

The timing-independent signals: placeholder steps went from 24 and 1 to 0 and 0. In the new runs the whole script made 44 service calls (1 to 2 cancelled), 8 model change callbacks and 10 refresh passes (`GGUIX_ADAPTER` line). The first app's call and refresh counts were not instrumented, so there is no number to compare. Jump got slightly slower in these runs (about 20 ms), within what this shared machine's noise allows, but I did not separate noise from a real cost. `make test`: `GGUIX_TESTS passed=13 failed=0 not_feasible=0`.
