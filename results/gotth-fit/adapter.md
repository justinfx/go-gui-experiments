# GoTTH data layer, second pass

## Design

The first app copied the brief's windowing recipe: a table of sentinel rows, five 200-row pages in the DOM, spacers for evicted pages, and one page loaded at a time as a sentinel scrolled into view. The scrollbar covered only the loaded window, and a scroll step of 25 rows (about 1,000 px) nearly always landed on a page that was not loaded yet.

The new layer is what a GoTTH developer would build when the server owns the data and htmx moves HTML: a virtual scroller over server-rendered page blocks, with a page cache on the server. The two halves are small and each does one job.

Browser half (`static/app.js`, "list window"):

- `#vs` is one box as tall as the whole result. Rows have a fixed pitch (38 px table row, 104 px card, both measured from or fixed by the first app's markup), so the first and last visible row come from `scrollTop` alone and nothing is measured on scroll.
- A page is a block of 200 rows or cards, absolutely positioned. Blocks come from the server as plain templ output (`PageBlock`). Inside a block, 50-row chunks use CSS `content-visibility: auto` with an inline height, so the browser skips layout and paint for the parts that are off screen.
- A page that is wanted but not loaded is a placeholder block. It carries `…` rows (or `Loading…` cards) for the part that is on screen, and it is itself the htmx element: `hx-get="/page?...&p=N"`, `hx-trigger="gguix-load"`, `hx-swap="outerHTML"`. The scroll handler only sends it the `gguix-load` event. When the response arrives, htmx swaps the real block in. This keeps the data path in htmx's own request and swap machinery.
- `sync()` runs on every scroll event. It wants the visible pages first, then two ahead in the scrolling direction and one behind (one on each side when idle), starts at most four fetches in priority order, and aborts and removes placeholders that are no longer wanted (`htmx:abort`). Only the latest viewport is fetched. Loaded pages far from the viewport are dropped, and at most six stay in the DOM.
- Go-to-row no longer swaps a region. The total is known, so the list scrolls locally and a `/goto` request fetches only the form of that row.
- A result taller than 10,000,000 px (1,000,000 table rows is 38 million) cannot be one element, because browsers cap element height. The scroll space is then scaled (`scale = (maxPx - viewport) / (totalPx - viewport)`) and block positions are recomputed from `scrollTop` on every scroll event. Programmatic jumps remember the exact row so the target row lands at the top without rounding error.

Server half (`adapter_data.go`): `cachedSvc` wraps the core `DataService` and is installed in `gotth.New`, so every handler (list, form, go-to-row, restore, FindRow) goes through it.

- Every row request becomes a request for the aligned 200-row page that holds it. A one-row go-to-row lookup and the browser's page fetch are the same page, so they share one service call (single flight).
- Pages sit in an LRU of 24 pages (4,800 rows). A page fetched once is not fetched again while it stays there: scrolling back, switching between Table and Cards, and clicking a row are served without a 50 ms call.
- The form is filled from a cached page when the item is in one (`Get` scans the cache), so selecting a visible row costs no service call.
- The service call is detached from the HTTP request context. When the browser aborts a request because the user scrolled on, the page still finishes and lands in the cache.
- `Update` writes through and drops all cached pages, because an edit can change the order or the membership of any query. Calls in flight at that moment are not stored.

Toolkit APIs relied on: htmx `hx-get` / `hx-trigger` with a custom event / `hx-swap` / `hx-sync`, `htmx.process`, `htmx.trigger`, `htmx:abort`, `htmx:afterSwap`, `htmx:afterRequest`, `htmx.ajax`; templ components and `<template>` for the placeholder row; shadcn-templ table and card components unchanged; CSS `content-visibility`.

## Before and after

| Data-path change | First app | Now | Why |
|---|---|---|---|
| Scroll space | Only loaded pages plus spacers, so the scrollbar covered a window | One box for the whole result (scaled above 10 M px) | Any row is reachable by scrolling; no spacer bookkeeping, no scroll anchoring corrections |
| Page loading | One page at a time, down or up, by a sentinel found with `getBoundingClientRect` after every scroll and on a 100 ms timer | Up to 4 parallel page fetches, ordered by distance, with read-ahead in the scroll direction | The 50 ms latency is paid in parallel and ahead of the viewport |
| Stale requests | Not cancelled; each scroll step queued behind the last | Placeholders that are no longer wanted are aborted | Only the current viewport is fetched |
| Scroll position to rows | Measured with `getBoundingClientRect` on rows, spacers and sentinels | Arithmetic on `scrollTop` | No forced layout during scroll |
| Eviction | Replace the farthest page by a spacer of measured height | Remove the block; the space is virtual | No layout shift, no measuring |
| Offscreen cost | Whole 1,000-row window laid out | 50-row chunks with `content-visibility: auto` | Cheaper swaps, smaller gaps |
| Go-to-row | `/goto` made 2 serial service calls (total, then page) and swapped the whole region | Local scroll plus a one-row form request; both share the page's single service call | Jump needs one 50 ms call, not two in a row plus a region swap |
| Table to Cards switch | `FindRow` over up to 2,000 rows (10 serial 50 ms calls) when the selected row was no longer in the DOM | The client remembers the selected row index and sends it | No scan; cached pages are reused |
| Row to form | `Get` call (50 ms) | Served from a cached page when present | The row was just fetched |
| Repeated pages | Fetched again every time | Server LRU, single flight | Table/Cards switch, scroll back, retries cost nothing |
| Refresh trigger | 100 ms interval timer plus scroll listener plus MutationObservers on swaps | Scroll listener and `htmx:afterSwap` only | Fewer wake-ups and refreshes |
| "Loaded" bookkeeping | Counted every htmx request in flight | Checks that every page in the visible range is a real block (the section 9 definition) | Prefetch requests no longer delay "loaded" |

Service calls per bench run: 50 `Page` calls with 1,000,000 rows (counted with `GOTTH_LOG=1`; each is 200 rows).

## Files

- `adapter_data.go`: `cachedSvc` (page cache, single flight, write-through `Update`).
- `adapter_data_test.go`: unit tests (single flight, 200-row and 24-page limits, cache hit, `Update` invalidation, abandoned call).
- `server.go`: installs the adapter in `New`; `handlePage` returns one page block; `handleGoto` returns the form for one row.
- `views/list.templ`, `views/model.go`: `ListRegion` (`#scroller` / `#vs`), `PageBlock`, `Placeholders`, row heights, virtual height and block top helpers.
- `static/app.js`: the "list window" section (`sync`, placeholders, scaling, `scrollToRow`, `gotoRow`), keyboard navigation by row index, go-to-row submit handler.
- `assets/css/globals.css`: fixed row height and `content-visibility` (layout only).
- `views/page.templ`: the Go to row form is no longer an htmx form.
- `static/bench/bench.js` is unchanged.

## Limits

- `core.FindRow` (restore of the selected row on launch) is core code and fetches pages one after another, so a restore can still cost up to ten serial 50 ms calls. It now fills the cache, and bench mode skips it. A read-ahead in the adapter would speed it up but would start fetching pages nobody asked for, which the "never preload" rule discourages.
- The 50 ms latency per call is fixed by the core, so a cold jump cannot be faster than one call plus a render.
- The first sort by a field builds an index inside the core; the sort step (about 350 to 400 ms) is dominated by it.
- With 1,000,000 rows the scroll space is scaled (about 0.26 for the table), so one pixel of scrollbar movement is about four rows. That is inherent to browser element height limits.
- A single stall of 300 to 390 ms at the start of the desktop scroll step remains (not seen in headless Chrome). It looks like three page swaps landing together in WKWebView; I did not find a cheaper swap path within three rounds.
- Heavy jobs were kept small: three bench rounds on the web target while iterating, two desktop runs at the end. Other agents shared the machine and the screen may have been locked, so timings are direction only.

## Benchmarks (indicative only)

Headless Chrome, `tests/browser/bench-web.bench.mjs`, first app then now:

| Metric | First app | Now |
|---|---|---|
| ready | 167 ms | 144 ms |
| jump | 211 ms | 145 ms |
| scroll placeholder_steps | 146 | 0 |
| scroll p95 gap | 63 ms | 25 ms |
| cards switch_ms | 947 ms | 346 ms |
| cards placeholder_steps | 27 | 0 |
| cards p95 gap | 37 ms | 9 ms |
| sort | 338 ms | 375 ms (index build, noisy) |
| search | 169 ms | 171 ms |

Desktop shell (`make bench`), first app (recorded in capabilities.md) then two runs now: scroll placeholder steps 185 then 1 and 3; cards switch 2154 ms then about 460 ms; cards placeholder steps 163 then 0; sort 1083 ms then about 400 ms; jump 222 ms then 220 ms.

Signals that do not depend on timing: placeholder steps (0 to 3 instead of 146 and 185), service Page calls (50 per run), and view refreshes (page swaps only; the 100 ms timer is gone).
