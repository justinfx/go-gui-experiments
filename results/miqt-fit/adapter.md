# MIQT data layer (second pass)

## Design
`core.Pager` is replaced by `rowStore` plus one shared item model. The pager was a general, mutex-guarded, background-notifying cache; the new layer is owned by the Qt main thread, which is where Qt's model/view machinery already lives.

- **rowStore (UI-thread cache).** `data()` calls, scroll handlers and fetch completions all run on the Qt main thread, so reads take no lock and cannot race an arrival. Service calls run on goroutines and hand results back with `qt6/mainthread.Start`.
- **Scroll-aware prefetch.** The store measures scroll speed (smoothed rows per second) from the view's scrollbar and prefetches speed x 220 ms of rows in the scroll direction (50 rows both sides when still, capped at 3,000 rows). The pager used a fixed 50 row margin, less than one 50 ms service round trip at the bench's scroll speed.
- **Cancellation.** Each page fetch has its own context. When the view moves on, requests for pages that are no longer wanted are cancelled (the service honours ctx), so stale pages never occupy the in-flight slots. Up to 6 requests run in parallel (pager: 2).
- **Batched notification.** Arrivals within 4 ms are merged into one UI callback. It grows or shrinks the row count if Total changed, repaints the active viewport only if an arrived page overlaps the visible rows, and settles selection and status. Prefetch pages that are off screen cause no UI work at all.
- **One model, two views.** The card list is a second `QListView` on the same `QAbstractTableModel` (`setModelColumn(0)`), so a result is counted, synced and invalidated once. The two views share one `QItemSelectionModel`, attached only to the visible view (`setSelectionModel` on view switch).
- **Sort/filter keeps the row count.** `setQuery` no longer resets the model to 0 rows and re-inserts up to 1M rows; the old count stays (all rows show placeholders) until the first page reports the new Total, then `beginInsertRows/beginRemoveRows` apply only the difference.
- **Limits kept.** Page size 200, at most 40 pages (8,000 rows) cached, never enumerates the dataset, Page/Get/Update/Export only through the core service, core untouched. A row without data shows `…` / `Loading…` and counts as not loaded; `VisibleLoaded` also requires that no arrival notification is still pending.

Toolkit APIs relied on: `QAbstractTableModel` (OnRowCount/OnData/OnHeaderData, Begin/EndInsertRows, Begin/EndRemoveRows, Begin/EndResetModel), `QListView.SetModelColumn`, `QAbstractItemView.SetSelectionModel`, `QItemSelectionModel`, `QListView.ExecuteDelayedItemsLayout`, `QAbstractItemView.Viewport().Update()`, `qt6/mainthread`, `QStyledItemDelegate` (unchanged painting).

## Before and after
| Change | First app | Now | Reason |
|---|---|---|---|
| Cache owner | `core.Pager`, mutex, callbacks from goroutines | `rowStore`, UI thread only | no lock per `data()` call, no changeSeq/refreshSeq handshake |
| Refresh on arrival | `sync` both models, `dataChanged` for visible +-60 rows on both models, update both viewports, on every pager change | one coalesced callback; repaint of the active viewport only if visible rows got data | `dataChanged` reaches the hidden QListView and forces a 1M item relayout (100 to 500 ms stalls) |
| Prefetch | 50 row margin, 2 in flight, no cancel | speed-based lookahead, 6 in flight, stale requests cancelled | fewer placeholder rows while scrolling at 50 ms latency |
| Models | table model + list model, 1M rows each | one model for both views | half the row sync cost, one selection |
| Selection | applied to both views on every change | one shared selection model attached to the visible view | selecting on the hidden list forced its layout |
| Query change | reset to 0 rows, then insert up to 1M | keep count until new Total | avoids a 1M row remove and insert for a sort |
| Go to row | select then scroll | scroll then select | selection's own scrollTo finds the row already on screen |
| Cards switch | layout forced as a side effect of selecting | `ExecuteDelayedItemsLayout()` once, explicitly | scroll range must exist before the scroll value is set |

## Files
- `apps/miqt/adapter_store_desktop.go`: `rowStore` (cache, fetch scheduling, cancellation, prefetch, batching).
- `apps/miqt/adapter_models_desktop.go`: the shared item model.
- Edited for the new layer: `app_desktop.go` (arrival handling, visible range, view switch, selection), `ui_desktop.go` (view setup), `delegates_desktop.go` and `form_desktop.go` (read from / invalidate the store), `model_desktop.go` (column table only), `driver_bench.go` (prints a non-metric `GGUIX_DIAG` line to stderr), `app_test.go` (one field rename).

## Limits
- A `QListView` still lays out all 1M items on the first switch to Cards (about 1 s, same as the first app). `Batched` layout would remove it but clamps the scroll range, and a 1M item list is required by the brief to be a list widget. A windowed proxy model would fix it but is out of proportion here.
- Sort and the first show still pay about 130 to 150 ms for `beginInsertRows(1M)` in the table view; Qt does it synchronously.
- Not done: moving row formatting to a worker thread (formatting is cheap next to the 50 ms fetch); the service page size cap (200) stops larger fetches.

## Benchmarks (indicative only: shared machine, one run each, screen may be locked)
| metric | first app | now |
|---|---|---|
| ready ms | 870 | 580 |
| jump ms | 654 | 310 |
| scroll total / max_gap / p95 / placeholder_steps | 8196 / 259 / 233 / 105 | 3210 / 15 / 1.4 / 1 |
| cards_scroll switch / total / max_gap / p95 / placeholder | 916 / 5792 / 444 / 379 / 44 | 984 / 3207 / 0.7 / 0 / 0 |
| sort ms | 631 | 503 |
| search ms | 71 | 63 |
| export ms | 514 | 520 |

Timing-independent signals (run in bench mode): 54 service Page calls and 38 UI refreshes for the whole script. `make test`: `GGUIX_TESTS passed=18 failed=0 not_feasible=0`. `make web`, `web-remote`, `bench-web` still print UNSUPPORTED.
