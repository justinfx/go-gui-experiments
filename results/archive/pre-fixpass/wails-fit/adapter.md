# Data layer (second pass)

## Design

The first app followed the prescribed pager: a TypeScript port of the section 4.6.1 algorithm, a global `version` counter in the app state and one IPC call per 200-row page. This pass replaces it with a lazily fetching row model, `RowStore`, written in the style of a toolkit data model (the role Qt's `QAbstractItemModel` with `canFetchMore` plays) on top of Svelte 5 reactivity and the Wails binding layer.

Key ideas:
- **Fine-grained reactivity.** Pages live in a `SvelteMap` (`svelte/reactivity`). A view that calls `row(i)` subscribes only to the page holding row `i`, so a page arriving re-renders only its own rows. The old `version++` made every visible row and the status bar re-evaluate on every response, including prefetch pages nobody was looking at. Status-bar numbers (cached pages, last fetch) are written through a throttle (100 ms); row and total state is written at once.
- **Direction-aware read-ahead.** `setVisible` measures scroll velocity (jumps over 5,000 rows count as a jump, not speed). While scrolling, the wanted window extends up to 800 rows (about 400 ms at the current speed) in the direction of travel and keeps only 50 rows behind. At rest it keeps 50 rows on both sides.
- **Wanted set recomputed every time.** Pages scrolled past are never queued. Responses that land for pages no longer wanted are kept only if there is room, and the cache evicts the page furthest from the viewport (not LRU by touch).
- **Batched, parallel fetches (client and server).** Pages needed together go out as one `ItemService.PageBatch` call (up to 4 ranges). Go runs one `DataService.Page` per range in its own goroutine, so a batch costs one simulated latency whatever its size. Visible pages may exceed the read-ahead concurrency limit by two calls so they never queue behind prefetch.
- **No extra service calls.** Selecting a row that is not loaded yet (go-to-row, restore) waits for its page via `whenLoaded(i)`, which pins the page in the wanted set. The first app made a separate one-row `Page` call. After Save, `FindRow` (up to 10 service calls) is skipped when the sort is by id or SKU, which an edit cannot reorder.
- **Start-up.** `AppInfo`, `LoadPrefs` and `Palette` are sent together; `ready` no longer waits for `HasLastExport`.

Toolkit APIs relied on: Svelte 5 runes and `SvelteMap`, Wails v3 generated TypeScript bindings (`ItemService.PageBatch`), Go goroutines inside a bound service method, `@tanstack/svelte-virtual` for the visible range (unchanged).

Fixed rules kept: `core/` untouched; every row comes from `DataService.Page`; at most 200 rows per call (checked in `PageBatch`, `Page` and the store constructor); at most 40 pages (8,000 rows) cached, enforced after every insert; the whole dataset is never read. Web routes use the same binding over server mode.

## Before and after

| Area | First app | Now | Why |
|---|---|---|---|
| Change notification | one global `version++` per response | per-page reactive reads, throttled stats | prefetch arrivals no longer re-render visible rows |
| Prefetch | fixed 50-row margin both ways, page order by distance | velocity-scaled window ahead (up to 800 rows), 50 behind | rows are there before they scroll into view |
| Requests | one IPC call per page, 2 in flight | up to 4 pages per IPC call, parallel in Go, 3 calls (5 for visible pages) | fewer round trips, visible pages never wait |
| Queue | stale pages could stay queued, fetch in flight kept all | wanted set recomputed on every move | no work for pages already scrolled past |
| Cache | 20 pages, LRU by touch | 40 pages (8,000 rows), evict furthest | room for read-ahead under the 10,000 row cap |
| Select unloaded row | extra 1-row `Page` call | wait for the pinned page | one less service call |
| Save | `FindRow` always | skipped for id/SKU sort | up to 10 service calls saved |
| Start-up | 3 sequential IPC steps plus `HasLastExport` before first render | `Promise.all`, export flag not awaited | shorter path to ready |

## Files

- `apps/wails/frontend/src/lib/data/rowstore.svelte.ts` (row model) and `rowstore.test.ts` (9 unit tests)
- `apps/wails/adapter_pages.go` (`PageBatch`, call counters) and `adapter_pages_test.go`
- Wiring: `frontend/src/lib/state.svelte.ts`, `frontend/src/lib/api.ts`, `frontend/src/bench/bench.ts` (renamed `pager` to `rows`), `service.go` (`Page` shares `pageToView`, limit check, call count logged in bench mode on Quit)
- Removed: `frontend/src/lib/pager.ts`, `pager.test.ts`

## Limits

- An in-flight `DataService.Page` call cannot be cancelled (its latency is fixed in core), so a few pages requested just before a direction change still land. They are kept only if useful or if there is room.
- The scroll bench waits 16 ms per step, so `total_ms` for `scroll` and `cards_scroll` is dominated by the script's own sleeps (about 4 s); it cannot show gains.
- `sort` time is dominated by the core's index build on the first sort by a field; the adapter cannot change that.
- A server-push or streaming model (the server sending pages ahead without being asked) would need a change to `core` or its latency model, so it was not tried.
- Reactive per-page rendering could not be unit-tested with Vitest here (rune files named `.test.ts` are not compiled); it is covered by the Playwright and bench runs.

## Benchmarks (indicative only)

Desktop, `GGUIX_BENCH=1`, 1,000,000 rows, 50 ms latency, other jobs on the machine, so treat as direction.

| Metric | First app | Now (3 runs) |
|---|---|---|
| ready ms | 616 | 615, 611, 604 |
| jump ms | 88 | 80, 94, 86 |
| scroll placeholder_steps | 24 | 0, 0, 0 |
| scroll total_ms / max_gap / p95 | 4259 / 28 / 13 | 4121 / 21 / 8 ... 4146 / 17 / 14 |
| cards_scroll switch_ms | 189 | 101, 101, 104 |
| cards_scroll placeholder_steps | 0 | 0 |
| sort ms | 290 | 282, 295, 338 |
| search ms | 105 | 97, 98, 106 |
| export ms | 515 | 512, 513, 515 |

Service calls over a whole bench run: 44 `DataService.Page` calls in 44 IPC batches (logged as `GGUIX_INFO` on quit). The most reliable signal is the drop of scroll placeholder steps from 24 to 0 and the faster switch to Cards (one round trip fewer). Other metrics are within noise.

Tests: `make test` ends with `GGUIX_TESTS passed=36 failed=0 not_feasible=0` (was 33). The web targets build and serve in the Playwright run (server mode on port 8191, because 8090 is held by other processes on this host).
