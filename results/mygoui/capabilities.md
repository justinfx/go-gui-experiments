# Capabilities: MyGo native UI (mygoui)

Status values: native, framework-drawn, fallback, gap, not feasible. "Verified" notes say what was actually exercised. MyGo version v0.9.0; evidence is in `$(go env GOMODCACHE)/github.com/egoist/mygo@v0.9.0` (`docs/ui/*.md`, `ui/*.go`, `docs/menus.md`, `docs/ui/browsers.md`).

| Requirement | Status | How it was done | Notes |
|---|---|---|---|
| Window title, 1280x800 default, 1024x640 minimum | native | `WindowOptions{Title, Width, Height, MinWidth, MinHeight}` in `main_desktop.go` | |
| Restore saved size | native | prefs `WindowWidth/Height` fed into `WindowOptions`; `Window.OnResize` saves via core prefs | `StateKey` is not used, so the core prefs own the size. Not verified by relaunch after a manual resize. |
| Search field, 280 px, 250 ms debounce | native | `ui.TextInput` with `Placeholder`, `c.After` | tested (T3) |
| Category filter popup, 8 checkboxes, Clear, `Categories (n)` | native | `ui.Button` + `ui.Popover` with `ui.Checkbox` | |
| Status dropdown | native | `ui.Select` | |
| Go to row field and Go button | native | `ui.TextInput` (Enter via `Submitted`) + `ui.Button`; `ListState.ScrollTo(row, ui.Start)` | exercised by the bench driver (jump) |
| View switcher | native | `ui.Segmented` | |
| Export CSV button, theme dropdown | native | `ui.Button`, `ui.Select` | |
| Split view, draggable | native | `ui.SplitPercent` 65/35 | |
| Table (11 columns, fixed header, virtualized, selection, Up/Down) | native | `ui.Table` with `ListState{Selected}` | headers and body scroll sideways because the columns are 1300 px wide in a 65% pane |
| Column resize | native | built into `ui.Table` (drag a header edge, double click fits) | not exercised manually |
| Header sort with ▲/▼ | fallback | see friction 12:07: the app toggles itself and writes the glyph in the title | the table's own arrow is never shown |
| Custom cell: stock bar | native | `ui.Box` track with a child `ui.Box` fill (`Radius`, `Background`), returned from the `ui.Table` cell callback (`cells.go`) | exact section 4.5.1 colors; T10 measures the fill width and samples its pixel color |
| Custom cell: status badge | native | pill `ui.Box` (`Radius(999)`) with `ui.Text` | T10 samples the badge background |
| Custom cell: stars | framework-drawn | `ui.Box.Draw` with a painted star path, half star clipped with `Painter.Clip` | T10 counts full, half, empty |
| Placeholder rows | native | `…` text in each cell | |
| Zero results text | native | absolutely positioned `ui.Text` over the list | not tested automatically |
| Cards view | native | `ui.List` with a custom row of boxes and text, 96 px, 8 px gap (`view.go card`) | selected card: 2 px accent border; see friction 12:09 |
| Detail form: ID, SKU, Updated | native | `ui.Text` inside `ui.Field` in a `ui.Form` | |
| Name, Price | native | `ui.TextInput` | |
| Category | native | `ui.Select` | |
| Quantity | native | `ui.NumberInput` 0..1,000,000 step 1 | native spin box, no gap |
| Status radio group, horizontal | native | `ui.RadioGroup(...).Row()` | |
| Rating slider with value label | native | `ui.Slider` 0..5, rounded to 0.1 in app code | `ui.StepSlider` would draw 50 tick marks, so a plain slider is snapped by the app |
| Featured checkbox | native | `ui.Checkbox` | |
| Restock date picker and Clear | fallback | `ui.DateInput` (native calendar popup) shown when a date is set, a `Set date` button when none is | the widget has no empty state, so an unset date is a button; compromise noted |
| Notes text area, 4 lines | native | `ui.TextArea(...).Lines(4, 4)` | |
| Field errors under fields in error color | native | `ui.Field(...).Error(msg)` | tested (T5) |
| Dirty state, Save, Revert | native | app compares form fields with a snapshot | tested (T6) |
| Confirm dialog | native | `ui.AlertDialog` `Discard unsaved changes?` Cancel/Discard | drawn in the window; tested (T9) |
| Copy Row and "Row copied" | native | `Context.WriteClipboard` in the next frame, status bar message 2 s | brief names `mygo.Clipboard.WriteText`; the Context call is used because it also works in a browser |
| Import Notes… file dialog | native (desktop) / fallback (web) | `mygo.Dialog.Open` with a `.txt` filter; web uses a file input made with `syscall/js` | not exercised manually; 64 KB check and 500 character cut are in app code |
| Export CSV in background with progress and cancel | native | goroutine runs `DataService.Export`; progress through `Window.Update` | tested (T8) with the dialog bypassed |
| Save dialog | native | `mygo.Dialog.Save` with default name `gguix-export.csv` | not exercised manually (needs a person at the dialog) |
| Export complete notification | native with fallback | `mygo.NewNotification(...).Show()`; falls back to a status bar message when `NotificationsSupported()` is false or Show errors; web always uses the message | what the bare binary does on macOS was not observed |
| Web download of the export | fallback | export is built in memory and downloaded with a Blob and anchor click (`platform_web.go`); same for `web-remote` (the remote client streams into the buffer) | no gap, but not a streamed download. Not exercised by clicking in a browser. |
| Menu bar | native | `mygo.NewMenu`, `App.SetMenu` (`menu_desktop.go`): app menu (macOS) or Help, File, Edit, View with radio items and a Theme submenu, Help | built and started; items not clicked by hand |
| Shortcuts | native | `Accelerator: "CmdOrCtrl+..."` on desktop; `Context.Shortcut` in the view on web | not checked by key press on desktop |
| About dialog | native | `ui.Modal` with 4 lines, framework version constant `v0.9.0`, `runtime.Version()`, `BuildMarker 0` | |
| Theme modes | native | System keeps MyGo's default theme, Light and Dark call `SetTheme(LightTheme/DarkTheme().WithPreferences(...))` each frame | tested (T7) |
| Live system switching | native | the default theme follows the OS appearance, per the docs (`styling.md` Themes) | not toggled by hand |
| Theme dropdown and menu in sync | native | menu items updated each frame from app state | |
| UI-thread mechanism | native | `Window.Update(fn)` for goroutine work, `Window.Invalidate` from the pager callback, `mygo.RunOnMain` for the bench probe | |
| Native prefs facility (unused) | not used | MyGo has `StateKey` window state and `mygo` stores; the app uses core `PrefsStore` as required | |
| Prefs saved 500 ms after change and on quit | native | `time.AfterFunc` timer, `App.OnBeforeQuit` flush | tested (T7) |
| Restore selection | native | `core.FindRow(..., 2000)` then scroll and select | checked by screenshot with a saved `selectedId` |
| Host panel, desktop | native | `host.Current`, `ui.Modal` with the label/value list; reveal through `mygo.Shell.ShowItemInFolder` (core helper not used) | tested (T12); reveal buttons not clicked |
| Host panel, web | native (shape) | Source `unavailable`, actions disabled with tooltip `Not available in the browser` | the web page showed `unknown user` in the status bar; panel not opened by hand |
| Host panel, web-remote | not exercised | `-tags remote` builds and vets; `host.Current` fetches `/api/host` | the page was not run against the server |
| `web` target | native | `go tool mygo build -platform web` (13.5 MB wasm) served by `core/cmd/server -static` | the app renders in Chrome and the full bench script ran |
| `web-remote` target | fallback | same build with `GOFLAGS=-tags=remote` | built and vetted only |
| `cgoprobe` desktop | PASS | `CGO_ENABLED=1 GOFLAGS=-tags=cgoprobe go tool mygo build` | builds; the export test run with the C checksum was not done |
| `cgoprobe` web | FAIL | wasm build error: `core/checksum_nocgo.go: undefined: cgoprobe_requires_cgo_but_cgo_is_unavailable_for_this_target` | expected for js/wasm |
| `cgoprobe` web-remote | PASS | wasm client without the tag, `core/cmd/server` with `-tags cgoprobe` | both built; run not repeated with that server |
| `cross-windows` | not run | `go tool mygo build -skip-dmg -platform windows/amd64` is in the Makefile | not run in this session on instruction |
| `cross-linux` | not run | `go tool mygo build -skip-dmg -platform linux/amd64` | not run in this session on instruction |
| Packaging | native | `go tool mygo build -skip-dmg` makes `build/darwin-arm64/GGUIX-mygoui.app` (10.4 MB) | `make package` copies it to `bin/` |
| T1 to T12 | native | `app_test.go` drives the view with `ui.NewTester`, memory service with 1000 rows and no latency | all 12 pass, none not feasible. T8 starts the export through `runExport`, because the native save dialog cannot be driven; T10 checks fractions through measured element width, colors by sampling the rendered image, star counts through accessibility names. |
| Bench | native | `driver_bench.go` implements `bench.Driver`; same file for desktop and web | desktop printed all eight events; the web run printed all eight in the browser console |

## Target notes

The desktop and web targets share almost everything (`app.go`, `view.go`, `cells.go`, `export.go`, `driver_bench.go`). Per-target code is in whole files: `main_desktop.go`, `menu_desktop.go`, `platform_desktop.go`, `platform_shortcuts_desktop.go` (native menu bar, dialogs, notifications, window size, Finder reveal) and `main_web.go`, `platform_web.go` (no menu bar, so the shortcuts are handled in the view; the export is a Blob download; notes import uses a file input). Both fill the same `platformHooks` struct, so the shared code never checks the platform; the only platform test is in the bench driver, where `runtime.GOOS == "js"` makes the bench export go nowhere. Makefile blocks are marked `gguix:target`. A native-UI MyGo app can be compiled to the browser (WebGL 2 canvas) with the same CLI, so a web target was possible; its limits (no file dialogs, menus or notifications) are in `docs/ui/browsers.md`.
