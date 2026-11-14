# Capabilities: go-gui v0.96.0

| Requirement | Status | How | Notes |
|---|---|---|---|
| Window title, size, min size, size restore | native | `WindowCfg{Width,Height,MinWidth,MinHeight}`, size saved from `EventResized` | |
| Search field with 250 ms debounce | framework-drawn | `gui.Input` + `time.AfterFunc` posted via `QueueCommand` | |
| Category popup | framework-drawn | Button + floating `Column` with `Checkbox` and Clear | closes by clicking the button again, not on outside click |
| Status filter | framework-drawn | `gui.Select` | |
| Go-to-row | framework-drawn | `gui.Input` (digits only via `PreTextChange`) + Go button, Enter | |
| View switcher | framework-drawn | `gui.SegmentedControl` | |
| Theme dropdown | framework-drawn | `gui.Select`, synced with View > Theme | |
| Split view | framework-drawn | `gui.Splitter`, draggable | second pass: detail pane min 420 px, form labels stacked above controls, so the form fits the pane |
| Table | framework-drawn | second pass: `datagrid.New` with a `DataGridDataSource` (`adapter_grid.go`) over the app's row store (`adapter.go`) | **gap: no continuous scroll over 1,000,000 rows.** DataGrid source mode is offset paging: it holds one page (5,000 rows here), virtualizes inside that page and shows its own pager row (prev/next, Rows a-b/total, Jump). The scrollbar spans the page, not Total |
| Column resize | framework-drawn | DataGrid `GridColumnCfg.Resizable` | last column pinned to the remaining pane width (no stretch column in DataGrid; widths are cached once, so it is forced through MinWidth = MaxWidth); it fills only when the pane is wider than the other columns |
| Sort headers with arrows | framework-drawn | DataGrid `Sortable` + `OnQueryChange`, mapped to the core sort | arrow is DataGrid's own glyph after the title, not part of the label text; DataGrid cycles asc/desc/none, the app maps none back to asc to keep the toggle |
| Virtualization | framework-drawn | table: DataGrid row virtualization within the page (needs a fixed `Height`, measured from the pane in `AmendLayout`); cards: `VirtualList` | placeholder `…` rows: gap in the table, while the next page loads DataGrid keeps showing the previous page (counted as not loaded) |
| Keyboard Up/Down | framework-drawn | DataGrid's own keyboard selection | stops at the page edge (gap) |
| Stock bar cell | text fallback (table) | table: `████░░░░ 182` text, `CellFormat` text color = fill / low-stock color; cards: drawn bar | DataGrid has no per-cell view callback |
| Status badge | text fallback (table) | table: status text, `CellFormat` badge background on the whole cell + badge text color (no pill); cards: drawn pill | |
| Stars | text fallback (table) | table: `★★★⯪☆ 3.5` glyphs, `CellFormat` star color (empty stars same color); cards: `DrawCanvas` stars | |
| Card item | framework-drawn | `VirtualList` item = row container (tile, columns, chips) | height 96 + 8 gap |
| Detail form: text fields | framework-drawn | `gui.Input` (read-only for ID/SKU/Updated) | |
| Category dropdown | framework-drawn | `gui.Select` | |
| Quantity spin box | framework-drawn | `gui.NumericInput` (Up/Down steps, min/max) | |
| Status radios | framework-drawn | `gui.RadioButtonGroupRow` | |
| Rating slider | framework-drawn | `gui.Slider` step 0.1 + label | |
| Featured | framework-drawn | `gui.Checkbox` | |
| Restock date picker | framework-drawn | `gui.InputDate` + Clear | |
| Notes multi-line | framework-drawn | `gui.Input{Mode: InputMultiline}` | |
| Confirm dialog | framework-drawn | `w.Dialog(DialogCustom)` with Discard/Cancel | `NativeConfirmDialog` has fixed button labels |
| Clipboard | native | `w.SetClipboard` | |
| File dialogs | native | `w.NativeSaveDialog`, `w.NativeOpenDialog` | |
| Notifications | native | `w.NativeNotification`, status-bar fallback on failure | |
| Menus | native (macOS) | `App.SetNativeMenubar`, `IncludeEditMenu` | Copy Row in File (Edit menu fixed); checked items, not radio |
| Shortcuts | native | `w.RegisterCommands` (Global), referenced by menu `CommandID` | |
| Theme modes, live system switch | native | `SetTheme(ThemeLight/ThemeDark)`, `FollowSystemAppearance` + `OnSystemAppearance` | |
| UI-thread mechanism | native | `Window.QueueCommand` (+ `InvalidateLayout` from pager goroutine) | |
| Native prefs facility (unused) | n/a | `gui.LoadSettings`/`SaveSettings` (JSON under UserConfigDir, localStorage on web) | core `PrefsStore` used instead |
| Reveal in file manager | fallback | `host.RevealInFileManager` | go-gui has no reveal API |
| Host panel desktop | native | `host.Current`, all fields, actions enabled | |
| Host panel web | fallback | `Source: unavailable`, actions disabled with note | build verified, not run in a browser here |
| Host panel web-remote | fallback | `server-process` via `/api/host`, actions disabled | build verified only |
| Export desktop | native | save dialog, background goroutine, progress/cancel | |
| Export web | framework-drawn + syscall/js | in-memory CSV, Blob download | not exercised in browser this session |
| Web target | works (build) | `GOOS=js` build, `gogui.js` + `wasm_exec.js`; backend is Canvas2D, not WebGL | `make web` built and served |
| Web-remote target | works (build) | `-tags remote` | |
| Package | native | go-gui `cmd/buildapp` | produced `bin/GGUIX-gogui.app` |
| cgoprobe desktop | PASS | | |
| cgoprobe web | FAIL | no cgo on wasm (expected) | |
| cgoprobe web-remote | PASS | client without tag, server with cgoprobe | server run not verified this session |
| cross-linux | PASS | `CGO_ENABLED=0`, no extra tools | ~35 MB |
| cross-windows | PASS | `CGO_ENABLED=0`, no extra tools | ~26 MB |
| T1-T12 | pass | `gui.NewTestWindow`, `TestRender`, `TestClick`, layout walk, plus model calls | T10 checks fill width/color via layout shapes; badge colors via palette |

## Target notes

Desktop-only code is in `main_desktop.go`, `platform_desktop.go` (native dialogs, notifications, reveal) and `menu_desktop.go` (native menubar). Web-only code is `main_web.go`, `platform_web.go` (Blob download, file input via syscall/js, no menubar) and `web/index.html`. Everything else (model, view, cells, commands, bench driver) is shared, because go-gui renders the same widget tree on both. The Makefile marks desktop and web blocks.
