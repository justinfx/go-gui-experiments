# Capabilities: Cogent Core v0.5.27

Status values: native, framework-drawn, fallback, gap, not feasible. All of Cogent Core's widgets are framework-drawn (there are no OS controls); "native" here means a framework-provided widget used as intended. Verification note: the Mac was at the login screen while I worked, so no desktop window was ever shown to a person. Everything was verified in the framework's offscreen window (tests, PNG renders) and, for the web targets, in a real browser pane.

## Section 5 behavior

| Requirement | Status | How it was done | Notes |
|---|---|---|---|
| Window title, 1280 x 800, min 1024 x 640, restore saved size | fallback | `core.NewBody("GGUIX - Cogent Core")`; on show `SystemWindow.SetWinSize` from prefs; `WindowResize` saves size (`main_desktop.go`) | No documented API for a minimum window size; the 1024 x 640 minimum is not enforced (gap). Framework also keeps its own window geometry file. Not seen on a real window. |
| Menu bar | fallback | Row of `core.Button` menus inside the window (`menus_desktop.go`) | No native menu bar support in v0.5.27 (gap for "native"). |
| Search field with 250 ms debounce | native | `core.TextField`, `OnInput` plus `time.AfterFunc` | |
| Category filter popup, `Categories (n)`, Clear | native | `Button.SetMenu` with `core.Switch` checkboxes | |
| Status filter dropdown | native | `core.Chooser` | |
| Go to row field and Go button, Enter, clamping | native | `TextField` + `Button`; sets `StartIndex`, moves scrollbar, selects | |
| Table/Cards switcher | native | `core.Switches` segmented button | |
| Export CSV button, theme dropdown | native | `Button`, `Chooser` | |
| Split view, draggable divider | native | `core.Splits` 0.65 / 0.35 | |
| Status bar text, progress bar, Cancel, identity, Host button | native | `Text`, `core.Meter`, `Button` | |
| Table, virtualized on the row model | fallback | `rowsTable` on `core.ListBase` with a zero-size backing slice (`table.go`), fed by `rowModel` (`adapter_model.go`, second pass; replaces `core.Pager`) | `core.Table` itself only binds to reflected slices (see friction). No row data is materialized. |
| Table columns, widths, alignment | native | custom cells per column | |
| Column resize | native | `core.Handle` after each header (copied from `Table`'s approach), double click resets | |
| Header sort with arrow in the label | native | header `Button` text `Name ▲` / `▼` | |
| Fixed header | native | header frame above the scrolling grid | Header scrolls horizontally with the body. |
| Custom cell: stock bar | native | `stockBar` (two rounded `core.Frame`s, widths from `StockFraction`) | Mechanism: custom child widget per cell. |
| Custom cell: status badge | native | `pill` (a styled `core.Text`, fully rounded) | Palette from `gguix/core`. |
| Custom cell: stars | native | `starsW` (`core.Canvas` drawing star polygons, half star clipped) | Canvas coordinates are normalized. |
| Placeholder rows `…`, `No items match.` | native | `rowItem` returns not loaded; stack frame shows an empty-state text | |
| Selection by click, Up/Down keys, default selection style | native | `ListBase` read-only selection | |
| Selection kept only if in first page after query change | native | `App.resolveSelection` | |
| Detail form controls | native | see rows below | |
| ID, SKU, Updated read-only | native | `core.Text` | |
| Name, Price | native | `core.TextField` | |
| Category dropdown (8) | native | `core.Chooser` | |
| Quantity spin box 0..1,000,000 | native | `core.Spinner` | |
| Status radio group, horizontal | native | `core.Switches` radio, mutex | Rendered vertically in a narrow form pane (wraps). |
| Rating slider 0..5 step 0.1 with value label | native | `core.Slider` + `Text` | |
| Featured checkbox | native | `core.Switch` checkbox | |
| Restock date with date picker and Clear | native | `TextField` with a calendar icon that opens `core.DatePicker` in a dialog, plus `Clear` | Picker is a dialog, not inline. |
| Notes text area, 4 lines, wraps | native | `textcore.Editor` (`text/textcore`) | `core.TextField` is single-line semantics. |
| Dirty state, Save/Revert enablement, `Unsaved changes` | native | `detailForm.dirty` | |
| Save, validation messages under fields in error color | native | `ValidateItem`; `colors.Scheme.Error.Base` | |
| Revert and dirty-guard confirm dialog | native | `core.NewBody("Discard unsaved changes?")` with Discard/Cancel (`confirmDiscard`) | |
| Copy Row (clipboard) | native | `WidgetBase.Clipboard().Write(mimedata.NewText(..))`, status message 2 s | |
| Import Notes file dialog | framework-drawn | `core.FilePicker` in a dialog, `.txt` filter; web uses a browser file input via `syscall/js` | No native open panel (gap for native). 64 KB limit and 500-char cut done. |
| Export in background, progress, cancel, partial file removed | native | goroutine with `DataService.Export`; `AsyncLock` updates | |
| Export save dialog | framework-drawn | `core.FilePicker` | No native save panel (gap for native). |
| Export-complete OS notification | gap | In-app snackbar `Export complete: N rows written` | No notification API in v0.5.27. |
| Web export as download | native | in-memory buffer then `Blob` + anchor click (`export_web.go`); web-remote streams the server CSV into the buffer through the core client | Download is not streamed to disk by the browser. |
| File, Edit, View, Help menus and items | fallback | `menus_desktop.go` | In-window menus. |
| Shortcuts (Cmd+E, O, Shift+C, 1, 2, R, F, I, Q) | native | `Button.SetShortcut`; the framework gathers shortcuts of menu buttons window-wide | Not exercised by key presses (display locked). |
| Edit menu Cut/Copy/Paste/Select All | fallback | Sends synthetic key chords to the last focused text widget | The framework has no default edit menu and the TextField methods are unexported. |
| About dialog | native | `core.NewBody` with 4 lines (`aboutLines`) | |
| Theme modes System/Light/Dark | native | `AppearanceSettings.Theme` + `Apply()` + `UpdateAll()` without saving | |
| Live switching with the OS appearance | native (not observed) | framework handles `ScreenUpdate` when Theme is Auto | Not observed on a real window. |
| Theme dropdown and menu in sync | native | `toolbarParts.sync` and menus rebuilt on open | |
| Prefs through the core PrefsStore, 500 ms debounce, on quit | native | `savePrefsSoon`, `AddQuitCleanFunc(saveOnQuit)` | |
| Native prefs facility (unused) | n/a | Would have been a `core.Settings` type registered with `core.AddAppSettings`, saved as TOML in `~/Library/Cogent Core/` | Not used as required. |
| Restore prefs then `FindRow` | native | `restoreSelection` | |
| UI-thread mechanism | native | `WidgetBase.AsyncLock` / `AsyncUnlock` from goroutines (`App.async`) | Event handlers and updaters already run on the UI thread. |
| Cards view | native | `cardsList` on `ListBase`, one `cardWidget` frame per row (`cards.go`) | Notes line is clipped, not ellipsized beyond the 90 character `NotesPreview`. |
| Card selection border, placeholder `Loading…`, keys | native | frame style from selected index | |
| Host panel | native | `hostBody` dialog from `host.Info.Rows()`, Notes lines | See per-target rows below. |
| Reveal actions | fallback | `host.RevealInFileManager` (no framework facility) | |

## Target specific

| Requirement | Status | How it was done | Notes |
|---|---|---|---|
| Host panel, desktop | native | Source `native`, username and hostname shown, actions enabled | Test T12 checks the dialog content against `host.Current`. |
| Host panel, `web` | native | Source `unavailable`, actions disabled with tooltip `Not available in the browser` | Not opened in the browser. |
| Host panel, `web-remote` | native | Source `server-process` from `GET /api/host`; actions disabled | Status bar showed the server user in the browser. |
| `web` target | native | `core build wasm -o bin/web`; 38 MB wasm (about 8 MB gzip) with remote, similar without | Build verified; the in-process variant was not opened in the browser. |
| `web-remote` target | fallback | `GGUIX_EXTRA_TAGS=remote` through the `go` wrapper `web/gotags-shim.sh`, served by `core/cmd/server -static` | `core build` has no tags option. Opened in the browser; table, selection and form worked. |
| Web bench | native | `?bench=1`; all 8 metrics printed in the console on web-remote | |
| Desktop bench | not feasible on screen | Real window blocked by the locked display; same script run in the offscreen window | Offscreen run: ready 352 ms, jump 94 ms, scroll 3.2 s (max gap 42 ms), cards_scroll 3.2 s, sort 352 ms, search 82 ms, export 533 ms. |
| `cgoprobe` desktop | native | PASS: builds with `-tags cgoprobe` and T1 passes | |
| `cgoprobe` web | gap | FAIL: `checksum_nocgo.go:8:9: undefined: cgoprobe_requires_cgo_but_cgo_is_unavailable_for_this_target` (cgo cannot target js/wasm) | |
| `cgoprobe` web-remote client / server | native | PASS: client builds without cgoprobe; server builds with `-tags cgoprobe` | |
| `cross-windows` | gap | `core build -t windows/amd64` fails: cgo (webgpu, glfw) needs a Windows C toolchain; FAIL, no output; a Docker mingw route was not tried | "Cross-compiling between desktop platforms is planned but not yet supported" in the framework docs. |
| `cross-linux` | PASS via Docker | `core build -t linux/amd64` fails (cgo without a Linux toolchain). Docker route: golang:1.25 linux/amd64 container, GOTOOLCHAIN=go1.27.3, apt gcc libgl1-mesa-dev libegl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev, then go build; output bin/linux/gguix-cogent, 41.6 MB ELF, 6 min 45 s including downloads | Not run. `make cross-linux` falls back to this automatically. |
| Package | fallback | `core pack -o .`, moved to `bin/GGUIX-cogent.app` | dmg step not run (needs pip dmgbuild). Bundle not launched. |

## Tests

| Scenario | Status | Notes |
|---|---|---|
| T1 loads | native (pass) | IDs 1, 2, 3 and SKU text read from the grid cells |
| T2 sort | native (pass) | header click, first name equals core's first row, arrow shown |
| T3 search | native (pass) | typing `steel` through TextField input events, count in status bar |
| T4 selection | native (pass) | every form control compared with the item |
| T5 validation | native (pass) | message shown, no `Update` call |
| T6 save | native (pass) | row shows the new name after the row model refetch |
| T7 theme and prefs | native (pass) | dark scheme active, prefs file has theme `dark` |
| T8 export cancel | native (pass) | partial file removed, button re-enabled |
| T9 dirty guard | native (pass) | the app is in the confirming state and selection stays |
| T10 custom cells | native (pass) | stock fraction, badge colors, star counts |
| T11 cards | native (pass) | card 1 content, clicking card 2 fills the form |
| T12 host | native (pass) | panel Source and Username, status bar identity |

`make test` ends with `GGUIX_TESTS passed=13 failed=0 not_feasible=0` (12 scenarios plus the data layer limits test). The tests drive widgets with `Send(events.Click)` and `Input` events in the offscreen window, not with real mouse input.

## Target notes

Second pass: the data layer is shared code (`adapter_model.go`, `adapter_ui.go`, see `adapter.md`) and has no target-specific lines.

The desktop and web targets share almost everything. Differences are in files with a `_desktop.go` or `_web.go` suffix: `export_desktop.go` (framework file picker, file writer, `Import Notes` from disk) versus `export_web.go` (in-memory buffer and Blob download, `<input type=file>`); `menus_desktop.go` (in-window menus, shortcuts, edit-menu key chords) versus `menus_web.go` (none); `main_desktop.go` (window size restore, quit hook); `quit_*.go`. `web/gotags-shim.sh` and the web targets in the Makefile are marked `# gguix:target web`. Bench code lives in `bench_bench.go` and `bench_offscreen_test.go`. The `remote` tag is not used by any app code: `gguix/core/backend` picks the remote client and the app only calls `backend.Open`. The Host panel and Reveal actions use `host.Info.Source` to decide whether actions are enabled.
