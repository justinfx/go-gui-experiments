# Capabilities: NSGo

Pinned `github.com/nsgo-dev/nsgo` `v0.0.0-20261114190341-2345b2fa5aba` (commit `2345b2f`). Evidence paths are in the module cache (`$(go env GOMODCACHE)/github.com/nsgo-dev/nsgo@v0.0.0-20261114190341-2345b2fa5aba/`, shortened to `nsgo/`): `ARCHITECTURE.md`, `docs/public/en/*.md`, `base/datatable/datatable.go`, `core/services.go`, `desktop/`.

Status words: native (a platform control), framework-drawn (NSGo draws it from views), fallback, gap, not feasible.

## Section 5 and 6

| Requirement | Status | How it was done | Notes |
|---|---|---|---|
| Window title, size 1280 x 800, minimum 1024 x 640, restore size | native | `nsgo.WindowSize`, `nsgo.WindowConstraints`, `desktop.SetTitle`; size saved from `desktop.Frame(cx)` in an effect | Linux honors size only (`nsgo/docs`: GTK has no maximum sizes). Restore verified by the code path only, not by relaunching with a changed size |
| Menu bar | native | `desktop.SetApplicationMenu` (macOS menu bar, GTK menu bar in the window on Linux) | File / Edit / View / Help seen in the Linux screenshots; macOS menu bar not inspected by screenshot (window-only rule), covered by `nsgotest` menu data |
| Shortcuts (Cmd/Ctrl+E, O, Shift+C, 1, 2, R, F, I, Q) | native | `MenuItem.Shortcut` (`mod+...`); `MenuQuit` role for Quit; on the web `cx.OnKey` | Linux: Ctrl+2, Ctrl+1, Ctrl+I, Ctrl+E, Ctrl+Shift+C, Ctrl+Q fired under X11 and sway |
| Search field, 250 ms debounce | native | `ui.Input`, `cx.AfterFunc(250ms)` | |
| Category filter popup with 8 checkboxes and Clear | framework-drawn | `ui.Menu` with checked `ui.MenuItem`s (`NoCloseOnSelect`), trigger label `Categories (n)` rebuilt by a memo | a checkable menu, not literal checkboxes |
| Status dropdown | framework-drawn | `ui.Select[string]` | popup list drawn by NSGo |
| Go-to-row field and Go | native | `ui.Input` (number keyboard) + `ui.Button`; Enter via `OnSubmit` | |
| View switcher | framework-drawn | `ui.Tabs[int]` segmented | |
| Export CSV button, theme dropdown | native | `ui.Button`, `ui.Select` | |
| Split view, draggable divider | framework-drawn | `base.ResizablePanelGroup`; right panel 440 pt, min 320 | not drag-tested by hand |
| Status bar, progress bar, percent, Cancel, user@host, Host button | framework-drawn | views, `ui.Progress`, `ui.Button` | |
| Table (virtualized, 1M rows) | framework-drawn | `base/datatable.DataTable`, `RowCount`/`Text` from the pager; no `ui` wrapper exists | keeps only the pager window. Gallery styles reused (the base table has no look) |
| Column resize | framework-drawn | built in (drag the edge, double press fits) | not exercised by hand |
| Column move | framework-drawn | built in | extra |
| Sort by header click, arrow in the label | fallback | `RenderHeaderCell` with a pressable view; label ` ▲`/` ▼`; Stock has no sort field | re-checked: the only sort hook is `Column.Sortable` + `OnSortChange` (`base/datatable/datatable.go` `sortNext`, a chevron button, 3-state cycle unsorted/asc/desc, key lost on the clear); a header press only selects the column (`headerPress`); `ui` has no table wrapper (`ls ui/`). No better API exists |
| Selection, Up/Down | framework-drawn | `Selection` + `OnSelectionChange`; arrows built in (`NoLoopSelection`) | selection style is the table's `RowSelected` part |
| Placeholder rows `…`, `No items match.` | framework-drawn | `Text` returns `…`; `Empty` node | |
| Stock bar, status badge, stars (table) | framework-drawn | `DataTable.RenderCell` returns a normal view tree (`cells.go`): track and fill views with radius, pill view, `★` glyph text with a clipped half star | the mechanism the brief expects; glyph stars, not shapes |
| Cards (virtualized, custom item) | framework-drawn | a `base/datatable.DataTable` with one column, a 104 pt row (96 pt card + 8 pt gap), a 1 pt invisible header and `RenderCell` returning the card view tree; selection, arrows and paging as in the table | deviation: not `base.List`/`ui.List`, because only the DataTable scales its scroll range (`base/datatable/layout.go` `maxExtent = 3_000_000`). `core.List` (`core/list.go`, `liststate.go`) has no chunk or scale mode and lays out in float32. Verified headless at rows 200,000 and 500,000 (no commit-check panic, top row correct) |
| Detail form: ID, SKU read-only text | native | `core.Text` | |
| Name, Price, Notes | native | `ui.Input` (Notes `Multiline`, 92 pt high) | |
| Category | framework-drawn | `ui.Select[gx.Category]` | |
| Quantity spin box | framework-drawn | `ui.NumberInput` 0..1,000,000 step 1 | clamps to its range (that is the section 5.4 range), so the "Quantity must be..." message cannot be reached from the UI; the message is still produced by `toItem` if a value outside the range ever arrives. Not a gap: the control does what the brief asks |
| Status radio group, horizontal | framework-drawn | `ui.RadioGroup` + `ui.Radio` | wraps to two lines in a narrow panel |
| Rating slider 0.0..5.0 step 0.1, value label | framework-drawn | `ui.Slider` | |
| Featured checkbox | framework-drawn | `ui.Checkbox` | |
| Restock date picker + Clear | framework-drawn | `ui.DatePicker` (`%Y-%m-%d`) + `ui.Button` | no gap |
| Updated read-only | native | text | |
| Save, Revert, Copy Row, Import Notes, Unsaved changes | native | `ui.Button`, `core.Show` | |
| Field errors in the theme error color | framework-drawn | `ui.ColorsOf(cx).Danger` | |
| Confirm dialog | framework-drawn | `ui.AlertDialog` (`Discard unsaved changes?`, Discard/Cancel) | `cx.ShowAlert` (system alert) exists; not used because the test model and look were simpler with the in-app dialog |
| Error dialog | framework-drawn | `ui.Dialog` | |
| About dialog | framework-drawn | `ui.Dialog`; version from `debug.ReadBuildInfo` | |
| Clipboard (Copy Row) | native | `cx.SetClipboard` | `xclip` and `wl-paste` read the row on Linux |
| File dialogs (save, open filtered to .txt) | native | `cx.SaveFile` (empty file placeholder), `cx.PickFiles{Types: .txt}` | macOS dialogs not driven by hand; Linux through the portal (see `linux.md`) |
| Background export, progress, cancel, partial file deleted | native | goroutine + `cx.Post`; Cancel removes the file | T8 |
| OS notification `Export complete` | native | `cx.ShowNotification` after `RequestNotificationPermission`; falls back to a status-bar message when not shown | Linux: shown by dunst. macOS from an unsigned binary not checked |
| Web export as a download | native | `cx.SaveFile{Data}` (browser save picker or download) | checked in the Claude browser pane (Chromium): an anchor download named `gguix-export.csv` was triggered. `web-remote` also builds the file in memory (not a streamed download) |
| Theme System/Light/Dark, live | native | `cx.SetAppearance`; selector and menu kept in sync (menu rebuilt in an effect) | system follow not observed on Linux in the container (see `linux.md`); macOS not switched live |
| UI-thread mechanism | n/a | `cx.Post(fn)` (any goroutine to the engine goroutine); signals are written only there. Timers: `cx.AfterFunc` | `nsgotest` runs posted functions when it settles |
| Preferences through the core `PrefsStore` | native | `gguix/core/prefs` | the framework's own facility would be `cx.Storage()` (key-value, 5 MiB, per app) or `core.NewPersisted` (typed, versioned JSON); unused as required |
| Prefs saved 500 ms after change and on quit | native | debounce timer; `cx.OnQuit` flush | |
| Restore selection with `FindRow` | native | `restoreSelection` | verified on macOS with a prefs file (`selectedId` 5 selected and scrolled to the top in Cards) |
| Host panel, desktop | native | `host.Current`; Reveal Last Export: `host.RevealInFileManager` (no NSGo call selects a file); Reveal Config Folder: `cx.OpenFile` on the folder, falling back to the core helper (`xdg-open`) when NSGo reports not opened | Linux: both ran `xdg-open` (Config Folder through the fallback, because the container portal has no OpenURI) |
| Host panel, `web` | native | Source `unavailable`, actions disabled with `Not available in the browser` | checked in the Claude browser pane (Chromium) |
| Host panel, `web-remote` | native | Source `server-process` from `GET /api/host`; actions disabled | checked in the Claude browser pane (Chromium) |
| `web` target | native | `go tool nsgo build -target web` (DOM, `syscall/js`) copied to `bin/web`, served by `core/cmd/server`; table, form, export and host dialog checked in the Claude browser pane (Chromium) | route: the tool has no serve-only flag, so `make web` uses `build` and copies `dist/<name>-<version>-web` |
| `web-remote` target | native | same with `GOFLAGS=-tags=remote` (the tool passes no tags of its own in a release build) | host dialog checked; data came from the server |
| `bench-web` | native | same build, `?bench=1` | script not run (see Bench) |
| cgoprobe, desktop | PASS | `CGO_ENABLED=1 go build -tags cgoprobe` | |
| cgoprobe, web | FAIL | `GOOS=js` has no cgo: `core/checksum_nocgo.go: undefined: cgoprobe_requires_cgo_but_cgo_is_unavailable_for_this_target` | expected |
| cgoprobe, web-remote | PASS | WASM client built without `cgoprobe`, `core/cmd/server` with it; app checked with the normal server | |
| cross-windows | PASS | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build`, 18.2 MB, no extra tool | not run on Windows. `nsgo build -target windows_x64` also exists |
| cross-linux | PASS | `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build`, 18.4 MB | the arm64 build ran in the container |
| package (.app) | PASS | `go tool nsgo build -target macos_arm64` (11 MB `.app`, signed ad hoc) | package launched? not run |
| Window identity (Linux) | fallback | the Makefile passes NSGo's own `-X core.appID/appName/...` ldflags (what `nsgo build` does); `identity_desktop.go` calls `g_set_prgname` through purego so GTK sets `WM_CLASS` and the Wayland `app_id` | NSGo has no option for it (`internal/platform/linux` never sets a program name or application id). Result: `WM_CLASS dev.gguix.nsgo`, `app_id dev.gguix.nsgo`, notification `app_name dev.gguix.nsgo`. Adds `github.com/ebitengine/purego` (already NSGo's dependency, same version) as a direct requirement |
| Dialog Escape | native | `ui.Dialog` closes on Escape | headless and Linux (X11, sway) verified |
| Bench mode, 8 events | not run | `bench_bench.go` implements `bench.Driver` (RunOnUI = `cx.Post`); script starts after the first animation frame | the coordinator measures. Cards now use the DataTable (see above) |

## Resolved: Cards over a million rows

The first version used `base.List`, whose layout is float32 (exact to 2^24 = 16.7 million pt); at 104 pt a card, row 200,001 is past it and nsgotest's commit check panicked. Cards now use the DataTable, which scales the scroll range. Trade-off: the scroll range is scaled (as in the table), so one wheel step moves more than a card at the far end of a million rows, and the list widget the brief names (`widget.List` equivalent) is not what is used.

## Section 8 tests (`make test`: `GGUIX_TESTS passed=12 failed=0 not_feasible=0`)

All run through `nsgotest` (NSGo's own headless test host) against the real component tree, with `memory.New(Rows 1000, Latency 0)`, a temp `GGUIX_PREFS_DIR`, a 2600 x 900 surface.

| ID | Status | Notes |
|---|---|---|
| T1 start | native | rows 1 to 3 and their SKUs found in the table |
| T2 sort | native | press `role:columnheader >> text:Name`; first row equals core's first `SortName` row; header shows ` ▲` |
| T3 search | native | typing `steel`, count in the status bar equals core's total |
| T4 selection | native | every form field equals the item |
| T5 validation | native | message shown, Update not called |
| T6 save | native | Update called once; table row shows the new name |
| T7 theme and prefs | native | `ui.Select` option `Dark`; `Appearance` is dark; prefs file contains `"theme":"dark"` |
| T8 export cancel | native | simulated save dialog answers a path; Cancel removes the file; button re-enabled; `Export cancelled` shown |
| T9 dirty guard | native | confirm dialog; Cancel keeps selection and edits |
| T10 custom cells | native | fill frame width equals `80 * StockFraction`, badge background equals the palette, text equals the status, star kinds equal `Stars(rating)` |
| T11 cards | native | card 1 text, price, status; pressing card 2 selects it and fills the form |
| T12 host | native | Source, Username and `user@host` equal `host.Current` |

## Target notes

The desktop and web targets share almost everything: `model.go`, `view.go`, `cells.go`, `export.go` and `platform.go` are `shared`, because NSGo's own services (`SaveFile`, `PickFiles`, `ShowNotification`, `SetClipboard`) work on both. What differs: `main_desktop.go` (window size and constraints, `nsgo.Run`), `menu_desktop.go` (the application menu, window-size saving, title), `platform_desktop.go` (empty-file save dialog, reveal in the file manager, quit) are desktop; `main_web.go`, `install_web.go` (keyboard shortcuts instead of a menu bar) and `platform_web.go` (export delivered as bytes to `SaveFile`, no host actions) are web. `bench_bench.go` is the bench driver and `app_test.go` the tests. The Makefile has `# gguix:target desktop` and `# gguix:target web` blocks.
