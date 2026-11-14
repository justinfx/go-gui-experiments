# go-gui second pass: data layer for DataGrid

## Design

The table is go-gui's `datagrid` (v0.96.0) with a `DataGridDataSource`. In source mode DataGrid is an **offset pager**: it asks the source for one page (`GridOffsetPageReq{StartIndex, EndIndex}`), keeps that page, virtualizes rows inside it and draws its own pager row (prev/next, "Rows a-b/total", Jump). **It cannot scroll continuously over 1,000,000 rows**: there is no virtual row count, and the scrollbar spans the current page only. This is a framework gap, recorded in capabilities.md.

The design leans into that model instead of fighting it:

- **One row store (`rowStore`, adapter.go)** replaces `core.Pager`. Rows live in 200-row blocks (one service call each). It has the pager's method set, so the form, selection and status bar code did not change.
- **Big grid pages, filled in parallel.** A grid page is 5,000 rows. `gridSource.FetchData` runs on DataGrid's own fetch goroutine (DataGrid already does the async work and the stale-request drop) and calls `rowStore.Fetch`, which reuses cached blocks and fetches the missing ones in parallel, 25 calls of 200 rows, about one latency for the whole page. The grid applies the page in one step: one layout per page instead of one per 200-row page as before.
- **Cards** stay on `VirtualList`; the store fetches visible blocks first then a 50-row margin, up to 6 in flight, LRU of 25 blocks (5,000 rows). Fetching for cards is switched off while the table is shown.
- **Memory cap:** grid page 5,000 + LRU 5,000 = 10,000 rows at most; blocks covered by the grid page are dropped from the LRU so a row is held once.
- **Coalesced wakes:** block arrivals within 4 ms produce one `InvalidateLayout`.
- **Query through the grid's own query state:** the core sort maps to `GridQueryState.Sorts` (so DataGrid draws the header arrow and its header click comes back through `OnQueryChange`); search, filters and a refresh revision ride in one `GridFilter` so the grid's query signature changes exactly when the core query does, which is how DataGrid resets to page 0 and refetches. The source decodes it back into a `core.Query`.
- **Abort:** DataGrid's `GridAbortSignal` is bridged to a context, so superseded page fetches cancel their service calls.

Toolkit APIs used: `datagrid.New`, `DataGridDataSource`, `GridOffsetPageReq`, `GridQueryState`, `OnQueryChange`, `OnSelectionChange`, `CellFormat`, `GetSourceStats`, `PageIndex/PageLimit`, `Window.ScrollVerticalTo/ScrollVerticalOffset`, `AmendLayout`, `QueueCommand`, `VirtualList`.

## Before and after

| Data path | First app | Second pass | Why |
|---|---|---|---|
| Table widget | header buttons + `VirtualList`, inside a wider horizontal scroll area | DataGrid | brief; fixes scrollbar placement and overdraw from the manual check |
| Fetch unit for the table | 200-row page, max 2 in flight, one UI wake per page | 5,000-row grid page, 25 parallel 200-row calls, one apply | fewer placeholder windows; one relayout per page |
| Cards fetch | core.Pager, 2 in flight | rowStore, 6 in flight, visible first | quicker fill after a jump |
| Change notification | one wake per page arrival | coalesced (4 ms) | fewer relayouts |
| Scroll to row | `ScrollToIndex` on the list | page index + `ScrollVerticalTo` within the page, at once when the page is already there | a deferred scroll cost a second frame per step and doubled scroll time in round 1 |
| Save refresh | `Pager.Invalidate` | patch item in place, invalidate, bump grid revision | grid has no public refetch |
| Loaded check (table) | `Pager.RangeLoaded` | grid has applied the page holding the visible rows (`GetSourceStats().Loading` false and source served that offset) | an old page still on screen counts as not loaded |

Layout fixes: the grid gets the pane's measured height (DataGrid virtualizes only with a fixed Height); the last column is sized to the remaining width; the detail form stacks labels above controls, the slider fills, the pane minimum is 420 px.

## Files

- `apps/gogui-fit/adapter.go`: `rowStore` (blocks, LRU, parallel page fill, coalesced wakes).
- `apps/gogui-fit/adapter_grid.go`: `gridSource` (DataGridDataSource), query mapping, columns, cell text fallbacks and `CellFormat`.
- Changed: `view.go` (table = DataGrid, grid measure, scroll, form layout), `app.go` (store instead of pager), `bench_bench.go` (table loaded check, store counters on stderr), `app_test.go` (grid IDs for header/rows; T10 checks fallbacks in table, drawn cells in cards).

## Limits

- No continuous scroll over the full result: DataGrid source mode is paged. A 5,000-row page is the largest that fits the 10,000-row cap next to the cards cache.
- No `…` placeholder rows in the table: while a page loads DataGrid shows the previous page. The bench counts those as not loaded.
- Stock bar, status badge and stars are text with `CellFormat` colors in the table (no per-cell view). The badge colors the whole cell; empty stars share the filled color.
- Up/Down stops at the page edge; DataGrid's own pager/jump bypass `OnPageChange`, so the app infers the page from what the source served.
- Sort arrow is DataGrid's glyph, not part of the header label.

## Benchmarks (indicative only)

macOS, one machine, two `make bench` rounds plus exploratory direct runs (raw: `raw/fit-bench-round1.log`, `raw/fit-bench-round2.log`). First app medians from `results/gogui/metrics.json` for comparison.

| Metric | First app (median of 5) | Second pass round 2 |
|---|---|---|
| ready ms | 277 | 340 |
| jump ms | 85 | 74 |
| scroll total ms | 3558 | 3385 |
| scroll placeholder_steps | 24 | 0 |
| cards switch ms | 117 | 62 |
| cards total ms / placeholder_steps | 3424 / 0 | 3498 / 0 |
| sort ms | 262 | 272 |
| search ms | 56 | 74 |
| export ms | 505 | 503 |

Placeholder steps fall to 0 because the whole scroll range (rows 500,000 to 505,000) sits in one 5,000-row page; only the last step enters the next page, and its wait is not a counted step. With 1,000-row pages, an exploratory run gave 16 placeholder steps. That is the honest trade: big pages hide fetches until the user crosses a page edge, where the whole page is replaced at once. `ready` is slower (first page is 5,000 rows plus DataGrid's first layout). Store counters for a full bench run (5,000-row page): 154 service calls, 14 coalesced UI wakes, 5,000 rows held at quit.
