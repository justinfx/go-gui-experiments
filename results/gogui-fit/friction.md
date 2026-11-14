# Friction log: go-gui

### 14:20 Module not in cache, no project template
- Step: bootstrap
- Command/API: `go get github.com/go-gui-org/go-gui@v0.96.0`
- Outcome: go-gui has no CLI template; the README says `go get`. Worked first time.
- Time lost: 0
- Category: docs-gap
- Resolution: plain `go mod init` plus `go get`.

### 14:25 Table and DataGrid do not fit a 1,000,000 row pager
- Step: choosing the table widget
- Command/API: `gui.Table(TableCfg{Data []TableRowCfg})`, `datagrid.DataGridCfg{DataSource}`
- Outcome: `Table` wants every row materialized in `Data`; `datagrid` has its own paging/data-source model and no per-cell view callback, so neither can use `core.Pager` with custom cells.
- Time lost: 10
- Category: framework-limitation
- Resolution: table built from a header row of ghost buttons plus `gui.VirtualList` (fixed `ItemHeight`) of row containers. Column resize recorded as a gap.

### 14:30 Edit menu cannot be extended
- Step: native menubar
- Command/API: `NativeMenubarCfg.IncludeEditMenu`
- Outcome: the auto-wired Edit menu is fixed; there are no radio items (only `Checked`).
- Time lost: 5
- Category: framework-limitation
- Resolution: Copy Row placed in File; View/Theme use checked items and the menubar is reinstalled when they change.

### 14:40 InputDate DateFormat panic
- Step: first launch
- Command/API: `InputDateCfg{DateFormat: "2006-01-02"}`
- Outcome: panic: "has no YYYY, MM or DD token". The format uses its own tokens, not Go layout strings.
- Time lost: 3
- Category: api-confusion
- Resolution: `DateFormat: "YYYY-MM-DD"`.

### 14:45 Pending UI commands in tests
- Step: tests
- Command/API: `w.QueueCommand` with `gui.NewTestWindow`
- Outcome: commands queued from timers (search debounce) only run on `FrameFn`, not plain `TestRender`; scoped IDs need `w.ResolveID`.
- Time lost: 5
- Category: docs-gap
- Resolution: tests call `FrameFn` in the wait loop and resolve leaf IDs.

### 14:50 cgoprobe web target
- Step: `make cgoprobe`
- Command/API: `GOOS=js go build -tags cgoprobe`
- Outcome: fails as expected (no cgo on wasm).
- Time lost: 0
- Category: toolchain
- Resolution: reported, not fixed.

### Second pass (gogui-fit) DataGrid source mode is paged only
- Step: replacing the hand-built table with `datagrid`
- Command/API: `datagrid.DataGridCfg{DataSource, PaginationKind: GridPaginationOffset, PageLimit}`
- Outcome: the source grid holds one page and draws a pager row; there is no virtual row count for continuous scrolling. It virtualizes only with a fixed `Height`. Its own pager buttons move the offset without calling `OnPageChange`, and there is no public refetch.
- Time lost: 20
- Category: framework-limitation
- Resolution: 5,000-row pages filled by 25 parallel 200-row calls; Height measured in `AmendLayout`; app keeps `PageIndex` in step with the page the source last served; refetch after Save by bumping a revision carried in `GridQueryState.Filters`.

### Second pass (gogui-fit) No custom cells in DataGrid
- Step: stock bar, status badge, stars
- Command/API: `DataGridCfg.CellFormat`
- Outcome: only text plus background/text color per cell.
- Time lost: 5
- Category: framework-limitation
- Resolution: text fallbacks with `CellFormat` colors; cards keep the drawn versions. Test T10 now checks the fallbacks in the table and the drawn cells in the cards.

## Summary

- Wall time to first window: about 25 minutes.
- Wall time to feature complete: about 40 minutes (bench not run in this session).
- Entries per category: docs-gap 2, framework-limitation 2, api-confusion 1, toolchain 1.
- Biggest pain points: no Table/DataGrid that takes an external pager with custom cells; fixed native Edit menu and no radio menu items; small API surprises (date tokens, test command pumping).
