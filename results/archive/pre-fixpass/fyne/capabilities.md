# Capabilities: Fyne v2.8.2

Status values: native, framework-drawn, fallback, gap, not feasible.

| Requirement | Status | How it was done | Notes |
|---|---|---|---|
| Window title, default 1280x800, restore saved size (desktop) | native | `Window.Resize` from prefs; title `GGUIX - Fyne` | saved on close from `Canvas().Size()` |
| Minimum window size 1024x640 | fallback | content wrapper layout with a minimum size (Fyne has no window min-size call) | toolbar is about 1000 px wide at default font |
| Toolbar: search entry 280 px, 250 ms debounce | native | `widget.Entry` in a fixed width layout, `time.AfterFunc` | |
| Category filter button + popup with 8 checks and Clear | native | `widget.Button` opening `widget.PopUp` with `widget.Check` list | label `Categories (n)` |
| Status filter dropdown | native | `widget.Select` | |
| Go-to-row field + Go (Enter too) | native | `widget.Entry` + `Button`; clamps | scrolls and selects |
| View switcher Table / Cards | fallback | two `widget.Button`s, active one high importance | Fyne has no segmented control |
| Export CSV button, theme dropdown | native | `Button`, `Select` | |
| Split view (65/35), draggable divider | native | `container.NewHSplit`, `SetOffset(0.65)` | |
| Status bar text, progress bar, Cancel, identity, Host button | native | HBox of Label / ProgressBar / Button | progress text formatter blanked, separate `NN%` label |
| Table (virtualized, fixed header) | native | `widget.Table` with `ShowHeaderRow`, 11 columns, 1M rows | row heights left at default so layout is O(visible rows) |
| Column resize by dragging | native | built into `widget.Table` header gaps; initial widths via `SetColumnWidth` | |
| Header click sorts, arrow in header text | fallback | custom tappable header (`widget.Label` subclass) via `CreateHeader`/`UpdateHeader` | table has no sort support |
| Stock bar cell | native (custom) | `canvas.Rectangle` track and fill with `CornerRadius`; widget `stockBar` in widgets.go, used inside `tableCell` (table.go) | exact 4.5.1 colors via `core.Colors` |
| Status badge cell | native (custom) | `pill` widget: rounded `canvas.Rectangle` + `canvas.Text` | fully rounded ends |
| Stars cell | native (custom) | five `canvas.ArbitraryPolygon` stars, half star is a left-half polygon over an empty one, plus text | concave polygon renders correctly |
| Placeholder rows show `…` | native | `tableCell.Set(..., nil)` | |
| Row selection and highlight | fallback | `Table` only selects cells; the row highlight is drawn by the app's cell with the theme selection color and the table's own marker is removed | |
| Up/Down keyboard selection | fallback | `gridTable.TypedKey` override | `OnHighlighted` also fires on hover so it is not used |
| Zero results text | native | centered label over the view | |
| Scroll offset / visible rows | fallback | private `offset` field read with reflect+unsafe; falls back to the last offset set | framework-limitation |
| Detail form: ID/SKU | native | `widget.Label` | |
| Name, Price | native | `widget.Entry` | |
| Category | native | `widget.Select` (8 options) | |
| Quantity | fallback | `widget.Entry` with integer validator plus +/- buttons | gap: no spin box |
| Status radio, horizontal | native | `widget.RadioGroup`, `Horizontal = true` | |
| Rating slider with label | native | `widget.Slider` 0..5 step 0.1 + `Label` | |
| Featured | native | `widget.Check` | |
| Restock date + Clear | native | `widget.DateEntry` (2.6+) with calendar popup, plus `Button` | shows the locale format (dd/mm/yyyy by default), not ISO |
| Updated | native | `Label` | |
| Notes multi-line, 4 rows | native | `NewMultiLineEntry`, `SetMinRowsVisible(4)`, word wrap | |
| Per-field errors in theme error color | native | `Label` with `DangerImportance` under each field | |
| Dirty state, Save/Revert enable, `Unsaved changes` | native | form compares against the loaded item | label is below the buttons (pane is narrow) |
| Save | native | validate with core then `DataService.Update` in a goroutine, `fyne.Do` back | pager invalidated |
| Confirm dialog (Revert and dirty selection change) | framework-drawn | `dialog.NewConfirm` with Discard / Cancel | |
| Copy Row (clipboard) | native | `Window.Clipboard().SetContent` | message in status bar for 2 s |
| Import Notes... file dialog | framework-drawn (desktop), fallback (web) | `dialog.NewFileOpen` with `.txt` filter; web: `<input type=file>` through `syscall/js` | Fyne file dialogs are Fyne-drawn on macOS, not native; they are empty stubs on WASM. 64 KB limit and 500 char cut done |
| Export: save dialog | framework-drawn | `dialog.NewFileSave`, default name `gguix-export.csv` | |
| Export: background job, progress, cancel, delete partial | native | goroutine + `fyne.Do`, `context` cancel | UI stays usable |
| Export: OS notification | native (desktop) | `App.SendNotification` (macOS uses osascript) | web: in-app status message plus `SendNotification` (browser notifications are not guaranteed) |
| Export on web | fallback | export built in memory then Blob download via `syscall/js`; with `web-remote` the core client streams from `/api/export` into the buffer first | not a true browser stream |
| Menu bar | native | `fyne.MainMenu`, native macOS menu bar | |
| Menu shortcuts | native | `MenuItem.Shortcut` with `desktop.CustomShortcut` (Cmd on macOS); non-macOS also registered with `Canvas.AddShortcut` | |
| App menu / About / Quit | native with deviation | the macOS driver moves an item labelled exactly `About` to the app menu, so it is labelled `About` on macOS and `About GGUIX` elsewhere | Quit is the standard macOS one; `IsQuit` item in Help elsewhere |
| Edit menu Cut/Copy/Paste/Select All | fallback | Fyne has no default edit menu; items forward `fyne.Shortcut`s to the focused widget | plus `Copy Row` |
| View menu radio items, Theme submenu | fallback | `MenuItem.Checked` (no radio type) kept exclusive by the app | |
| Reload, Focus Search, Host Info | native | menu items | |
| About dialog | framework-drawn | `dialog.NewCustom`: app name, Fyne version (build info), `runtime.Version()`, `BuildMarker 0` | |
| Theme: System/Light/Dark | native | default theme; Light/Dark use a small wrapper theme that forces the variant | colors are still the default theme's |
| Live system theme switching | native | System mode uses `theme.DefaultTheme()` which follows the OS variant | not exercised by an automated test |
| Theme dropdown and menu in sync | native | single `setThemeMode` | |
| UI-thread mechanism | native | `fyne.Do` (and `fyne.DoAndWait` not needed) | all callbacks from pager, timers, export go through `fyne.Do` |
| Native prefs facility (unused) | n/a | Fyne has `App.Preferences()` (key/value, stored in the app's config dir, localStorage-like on web); not used per brief, core `PrefsStore` used | |
| Prefs: debounce 500 ms, on quit, restore incl. `FindRow` | native | `time.AfterFunc` + `Window.SetOnClosed` | |
| Cards view (custom item) | native (custom) | `widget.List` with custom `cardItem` (BaseWidget + renderer, files cards.go), 96 px cards, 8 px gap (100 px item + list padding), 1 px / 2 px primary border, tile, 3 text lines, chips, price, stars, stock | list selection highlight is not used (the app draws the border) |
| Cards keyboard, go-to-row, shared selection | native | list subclass `TypedKey`; `ScrollToOffset` | |
| Host panel desktop | native | `host.Current`; dialog with all rows; Reveal buttons | reveal uses core `host.RevealInFileManager` (`open -R`); Fyne has no reveal facility (`App.OpenURL` opens only) |
| Host panel `web` | native | Source `unavailable`, buttons disabled, text `Not available in the browser` | |
| Host panel `web-remote` | native | Source `server-process`; verified `GET /api/host` returns the server user; buttons disabled | |
| Status bar identity | native | `Info.Identity()` | |
| web (WASM, in-process core) | native | `fyne package --target wasm`, moved to bin/web, `python3 -m http.server` | works in Chrome; 40 MB wasm; see bench numbers |
| web-remote | native | `fyne package --target wasm --tags remote`, served by `core/cmd/server -static` (also built with `-tags cgoprobe`) | see bench numbers |
| cgoprobe desktop | PASS | `go build -tags cgoprobe` (the desktop already needs cgo) | |
| cgoprobe web | FAIL | `undefined: cgoprobe_requires_cgo_but_cgo_is_unavailable_for_this_target` | expected: wasm has no cgo |
| cgoprobe web-remote | PASS | client wasm built without the tag, server built with `-tags cgoprobe`, app ran against it | |
| cross-linux | PASS | `fyne-cross linux -arch=amd64` after `go mod vendor`; `-env GOTOOLCHAIN=auto -env GOFLAGS=-mod=vendor`; extra tools: Docker, fyne-cross v1.6.3, 4.17 GB image | binary 25.5 MB, tar.xz 9.7 MB; two failed attempts first (see friction.md) |
| cross-windows | PASS | `fyne-cross windows -arch=amd64`, same flags; 2.97 GB image | exe 25.8 MB, zip 12.3 MB |
| macOS `.app` package | native | `fyne package --target darwin` | 33 MB |
| T1 start and load | native | Fyne test driver (`test.NewTempApp`, `test.NewWindow`) | pass |
| T2 sort | native | taps the Name header | pass |
| T3 search | native | pass | status bar count equals core Total |
| T4 selection | native | taps a table cell | all 12 fields compared |
| T5 validation | native | pass | Update not called |
| T6 save | native | pass | |
| T7 theme and prefs | native | pass | prefs.json contains `"theme": "dark"` |
| T8 export cancel | native | export slowed to 200 rows/s, cancel button | partial file removed |
| T9 dirty guard | native | pass | dialog title text checked in the overlay |
| T10 custom cells | native | checks bar fraction and width, badge text and colors, star counts and shapes, dark colors | |
| T11 cards view | native | pass | |
| T12 host info | native | pass | desktop expectation is `host.Current` |

Test result: `GGUIX_TESTS passed=12 failed=0 not_feasible=0`. The tests run on the desktop target only (Fyne test driver); the web targets were checked by running the bench and the UI in Chrome, not by an automated test suite (`not_feasible` for the web targets is not counted in the line).

## Bench results

Desktop (`make bench`, 1M rows, 50 ms latency): ready 639 ms, jump 69 ms, scroll 7250 ms total (max gap 38 ms, p95 31 ms, 0 placeholder steps), cards_scroll switch 104 ms / total 3204 ms (max gap 80 ms, p95 10 ms, 0 placeholder steps), sort 281 ms, search 77 ms (49,898 rows), export 503 ms (max gap 3 ms).

Web in-process (headless Chrome, SwiftShader software GL, so slow): ready 5531 ms, jump 894 ms, scroll 53,889 ms (p95 gap 202 ms), cards_scroll 14,398 ms, sort 1783 ms, search 676 ms, export 1567 ms (gap 990 ms, the export runs on the UI thread side in wasm). Treat the web numbers as an upper bound on a machine without GPU acceleration.

Web remote (core server built with `-tags cgoprobe`, 50 ms latency, same headless Chrome): ready 2756 ms, jump 715 ms, scroll 53,615 ms (max gap 1526 ms, p95 204 ms), cards_scroll switch 697 ms / total 11,111 ms (31 placeholder steps), sort 838 ms, search 640 ms (49,898 rows), export 990 ms (max gap 115 ms). `GET /api/host` returned the server process user with `source: server-process`; the Host panel reads that same value (checked by code path and API, the dialog itself was not screenshotted on the web). Port 8090 was occupied by another process on this machine, so web runs used 8091.

## Target notes

The desktop and web targets share almost everything (ui.go, table.go, cards.go, form.go, widgets.go, import.go, main.go). What differs:

- `platform_desktop.go` / `menus_desktop.go` (desktop): window size restore and quit, menu bar and shortcuts, save/open dialogs, OS notification, file manager reveal, bench export to a file, synchronous `host.Current`.
- `platform_web.go` / `menus_web.go` (web): no menu bar or window size, export built in memory and downloaded through `syscall/js`, notes import through a browser file input (Fyne's WASM file dialogs are stubs), host info fetched in a goroutine (network call with the `remote` tag), host actions disabled.
- `bench_bench.go` is the shared bench driver.
- The `web`, `web-remote` and `bench-web` recipes in the Makefile are in a `# gguix:target web` block.
- Tests (`*_test.go`) use the Fyne test driver and run for the desktop target.
