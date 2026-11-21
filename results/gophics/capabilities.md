# Capabilities: Gophics (`8d36097`)

Status values: native, framework-drawn, fallback, gap, not feasible. Gophics draws every widget itself, so "framework-drawn" is the normal state for controls. Evidence paths are under `$GOMODCACHE/github.com/doug/gophics@v0.0.0-20261103180631-8d360971eb89` (written `gophics@8d36097/`). App files are under `apps/gophics/`.

## Window, layout, menus

| Requirement | Status | How it was done | Notes |
|---|---|---|---|
| Title `GGUIX - Gophics` | native | `app.Config.Title` | |
| Default size 1280x800, restore saved size | framework-drawn | `app.Config.Size` from prefs loaded in `main.go`; size tracked from root `LayoutBuilder` constraints | |
| Minimum size 1024x640 | gap | no min-size field in `app.Config` or `shell.WindowControl` | fallback: restored size is clamped to at least 1024x640; the user can still shrink the window |
| Toolbar (search, categories, status, go-to, view switch, export, theme) | framework-drawn | `theme.Field`, `theme.Button`, `ui.Select`, `theme.Segmented` | search debounce 250 ms with `time.AfterFunc` + `Ctx.Post` |
| Category popup with 8 checkboxes and Clear | framework-drawn | `widget.Overlay.ShowFrom` with `theme.Checkbox` list, anchored by `LayoutObserver` (`ui/dialogs.go`) | |
| Status / theme / category dropdowns | fallback | `ui.Select` = `theme.Button` + `theme.ShowMenu` (`ui/dropdown.go`) | `theme.Dropdown` aborts the macOS app (runtime bug, see friction 15:12) |
| Split view, draggable divider | gap | no split container in the catalog | fallback: divider built from `widget.Interactive{OnDrag}` in a Row; it is draggable (`ui/view.go` mainArea) |
| Status bar (count, rows, fetch, pages, progress, Cancel, identity, Host) | framework-drawn | `theme.Progress`, `theme.Button`, `widget.Text` | |
| Native menu bar | native | `ctx.Menus().SetBar` (NSMenu on macOS via gogpu) in `ui/platform_desktop.go` | Linux: not checked (no menu visible in the X11 screenshot) |
| Menu radio items (View, Theme) | fallback | `shell.MenuItem` has no checked state; current item shown as `✓ Title` | |
| Menu shortcut labels / key equivalents | gap | `shell.MenuItem` has no shortcut field | shortcut text appended to titles, e.g. `Export CSV…  (⌘E)` |
| Shortcuts Cmd+E, O, Shift+C, 1, 2, R, F, I, Q | fallback | root `widget.Interactive{OnKey}` plus `widget.TextField.OnKeyPreview` on every text field (`ui/actions.go` fieldKeyPreview, `ui/field.go`); Gophics has no keymap API and keys reach only the focused widget | fix pass 1: now also work while a text field has focus. `theme.Field` does not expose `OnKeyPreview`, so `ui.Field` repeats `theme.Field`'s Build with the same theme values. Plain Cmd+C/X/V/A stay with the field. Ctrl+2, Ctrl+Shift+C, Ctrl+I, Ctrl+Q verified on Linux X11 (before the fix) |
| Quit (Cmd+Q) | native / fallback | menu `RoleQuit` (AppKit terminate); key handler calls `os.Exit` after saving prefs | no programmatic window close reachable from a `Ctx` |
| Edit menu Cut/Copy/Paste/Select All | gap | no edit roles in `shell.MenuRole`; the text field handles the keys itself | Edit menu has only Copy Row |
| About dialog | framework-drawn | `theme.ShowDialog`; framework version from `debug.ReadBuildInfo` | |
| Focus Search (Cmd+F) | fallback | no imperative focus API; the field is remounted with `Autofocus` (`FocusSearch`) | |

## Table and cards

| Requirement | Status | How it was done | Notes |
|---|---|---|---|
| Virtualized table body | framework-drawn | `widget.LazyList` with `ScrollController` (`ui/table.go`) | `theme.Table` exists but exposes no scroll controller, so go-to-row and scroll restore were impossible with it |
| Fixed header, sort on click, ▲/▼ in label | framework-drawn | header row above the list; arrow is part of the label text | |
| Column resize | framework-drawn | 6 px grips with `Interactive.OnDrag` | not offered by `theme.Table`; built by hand. Fix pass 1: the grip's 1 px divider is a `Decorated` (as `theme.Divider`), not a `Fill`, which expands to the whole grip |
| Row height | fallback | 32 px instead of the theme's 33 | `LazyList` sums float32 heights over all rows; a power of two stays exact at 1M rows |
| Scrollbar represents Total | gap (partial) | `widget.Scroll`'s overlay scrollbar (thumb sized from the full extent) | fix pass 1: the vertical bar used to sit at the far right of the 1,320 px table, off screen; now the list spans the visible width so the bar shows at the pane edge. The bar still fades out when idle and its drag target exists only while visible (`widget/scroll.go` scrollbarThumb, scrollbarBox.Paint); no always-visible option |
| Wide table | framework-drawn | fix pass 1: a 10 px horizontal `widget.Scroll` strip under the table owns the offset (`OnOffset`); header and rows are shifted with `widget.Transform` (`ui/table.go` hShift) | 1,320 px of columns in a ~830 px pane. Horizontal wheel over the rows does not scroll (the vertical list takes the event); use the strip |
| Selection highlight, Up/Down keys | framework-drawn | theme `Selection` color (fix pass 1; was `theme.Table`'s `Lerp(Bg, Primary, 0.10)`, too faint); root key handler | |
| `…` placeholder rows, `No items match.` | framework-drawn | | |
| Stock bar cell | framework-drawn | `widget.Canvas` with two `FillRRect` (`ui/cells.go` StockBar) | exact 4.5.1 colors via `core.HexColor` |
| Status badge cell | framework-drawn | `widget.Decorated{Radius: 999}` + `widget.Text` | |
| Stars cell | framework-drawn | `widget.Canvas`, five retained `paint.Path` stars plus five left-half paths | fix pass 1: the half star used a nested `PushClip`; on the GPU backend that fill escaped the table's clip and drew over the form. Now no clip is used; `TestStarsClippedGPU` (tag `gophics_gpu`) checks it |
| Featured ✓ | framework-drawn | text `✓` | Go fonts lack ✓; desktop uses `Config.SystemFonts`; web has no system fonts and logs a missing-glyph warning (tofu) |
| Card item | framework-drawn | `ui.Card` stateless widget: `Decorated` frame + `CategoryTile`, `StatusBadge`, `FeaturedChip`, `Stars`, `StockBar` in a `LazyList` (`ui/cards.go`) | fixed 96 px + 8 px gap. Fix pass 1: padding stays 12 px when selected (it shrank by 1 px, so content jumped) |
| View switch keeps selection visible | framework-drawn | `SetView` scrolls the selected row to the top | |

## Detail form

| Control | Status | How | Notes |
|---|---|---|---|
| ID, SKU, Updated read-only | framework-drawn | `widget.Text` | |
| Name, Price, Notes | framework-drawn | `theme.Field` (Notes `Multiline`, fixed 4-line box) | |
| Category | fallback | `ui.Select` | Dropdown crash |
| Quantity spin box | gap | no spin box; `theme.Field` with integer validation on Save | |
| Status radio group | framework-drawn | three `theme.Radio` in a Row | |
| Rating slider 0.0..5.0 step 0.1 | framework-drawn | `theme.Slider` (0..1) mapped to tenths, value label beside it | |
| Featured | framework-drawn | `theme.Checkbox` | |
| Restock date picker + Clear | framework-drawn | `theme.ShowDatePicker` (`Pick…`) plus a `YYYY-MM-DD` field and `Clear` | |
| Disabled state with no selection | gap | theme controls have no Disabled field | fallback: `widget.Opacity` 0.45 and no-op callbacks. On Linux (llvmpipe) the dimming did not show for Save/Revert |
| Dirty state, Unsaved changes, Save/Revert enable | framework-drawn | | |
| Validation messages in theme error color | framework-drawn | `theme.Theme.Danger` | |
| Confirm dialog | framework-drawn | `theme.ShowDialog` with Discard/Cancel | not a native NSAlert |
| Copy Row to clipboard | native | `ctx.Clipboard().ClipboardWrite`; `Row copied` in the status bar for 2 s | verified on Linux X11 with `xclip` |
| Import Notes file dialog (.txt, 64 KB limit) | native | `ctx.FilePicker().Open` (NSOpenPanel via zero-cgo objc) | not exercised interactively on macOS |

## Export, notifications, host

| Requirement | Status | How | Notes |
|---|---|---|---|
| Export save dialog | fallback | `FilePicker.Save` takes the bytes first and never returns the path (`shell/filepicker.go`, `shell/desktop/filepicker_darwin.go` save), so a streamed export cannot use it. Fix pass 1: an in-app `Export CSV` prompt for the file name (default `gguix-export.csv`, `.csv` added if missing), then `ctx.FolderPicker()` (native folder chooser) | not a native save panel; two steps |
| Background export, progress, Cancel, partial file deleted | framework-drawn | goroutine + `Ctx.Post`; T8 | UI stays usable (bench export max gap 15.7 ms) |
| OS notification | native | `ctx.Notifier()` (macOS: `osascript display notification`) | falls back to `theme.ShowSnackbar` |
| Web export download | framework-drawn | in-memory CSV, then `FilePicker.Save` (browser download) | download not clicked through in the browser |
| web-remote export | fallback | `remote` DataService streams `/api/export` into memory, then download | not a direct server-to-browser download |
| Reveal Config Folder / Last Export | fallback | Gophics has no reveal API; `host.RevealInFileManager` | |
| Host panel, desktop | native | `host.Current`, Source `native`; all rows + notes; reveal enabled | T12 |
| Host panel, web | framework-drawn | Source `unavailable`; reveal buttons disabled with tooltip and caption `Not available in the browser` | |
| Host panel, web-remote | framework-drawn | Source `server-process` via `GET /api/host`; actions disabled | Source row always shown |

## Theme, prefs, threading

| Requirement | Status | How | Notes |
|---|---|---|---|
| Light / Dark / System | framework-drawn | `theme.Light()` / `theme.Dark()` / `ctx.DarkMode()` provided via `widget.Provide[theme.Theme]` | |
| Live system switching | framework-drawn | `Ctx.DarkMode` is documented to rebuild on change | not toggled live during this run |
| Dropdown and View > Theme in sync | framework-drawn | both read `App.Theme`; menu republished on change | |
| Prefs (debounced 500 ms, on quit) | framework-drawn | core `PrefsStore`; Save on a goroutine (js deadlock otherwise) | native facility unused: `ctx.Preferences()` (JSON under `os.UserConfigDir`, localStorage on web) |
| Restore + FindRow selection | framework-drawn | `core.FindRow` in a goroutine, then scroll and select | |
| UI-thread mechanism | native | `widget.Ctx.Post()` / `StateBase.PostState`; pager callbacks coalesced into one `SetState` | menu actions arrive on the AppKit thread and are re-posted |

## Targets, cgo, cross

| Requirement | Status | Notes |
|---|---|---|
| Desktop build `CGO_ENABLED=0` | native | supported and used; 22.5 MB binary, links only libSystem, CoreFoundation, Security |
| `make package` | native | `gophics package` produced `GGUIX-gophics.app` (15 MB, unsigned) |
| `web` | framework-drawn | WebGPU in Chrome; bench printed all eight metrics (pane was hidden, so timer-throttled numbers) |
| `web-remote` | framework-drawn | ran against `core/cmd/server` built with `cgoprobe` after the prefs fix |
| `dev` | native | `gophics dev -p desktop` (hot restart keeping state) |
| CGOPROBE desktop | PASS | builds with cgo, T1/T8 pass (export calls C checksum) |
| CGOPROBE web | FAIL | `cgoprobe_requires_cgo_but_cgo_is_unavailable_for_this_target` (no cgo on js/wasm) |
| CGOPROBE web-remote client / server | PASS / PASS | server answered `/api/export` with the cgo checksum line; app loaded |
| cross-linux | PASS | `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build`, 25.4 MB, no extra tools |
| cross-windows | PASS | same route, 24.2 MB `.exe`, no extra tools |
| Linux runtime (x11, sway) | PASS | see `linux.md` |

## Tests (`make test`: `GGUIX_TESTS passed=12 failed=0 not_feasible=0`)

All run through `app.NewHeadless` (framework headless harness; real layout, taps and rendered pixels) plus the App presentation methods (`ui/app_test.go`).

| ID | Status | How |
|---|---|---|
| T1 | pass | rows 1..3 IDs and SKUs, SKU text in the drawn scene |
| T2 | pass | headless `Tap` on the Name header, compared with core's first `SortName` row; label `Name ▲` |
| T3 | pass | search via the debounced setter; status bar text matches core Total |
| T4 | pass | `Tap` on row 2 fills every form field |
| T5 | pass | `Name must be 3 to 80 characters.` drawn; Update not called |
| T6 | pass | saved name appears in the refreshed row |
| T7 | pass | dark background pixel; `"theme":"dark"` in prefs.json |
| T8 | pass | cancel removes the partial file, export button re-enabled, `Export cancelled` |
| T9 | pass | `Tap` on another row shows the confirm dialog; selection kept |
| T10 | pass | bar fraction, badge text and colors (light and dark), star counts; badge and star colors found in rendered pixels |
| T11 | pass | card shows name, price, badge; `Tap` on card 2 selects item 2 |
| T12 | pass | Source and Username rows; identity in the status bar |

## Desktop bench (one run, macOS)

`ready 392 ms, jump 78 ms, scroll total 6682 ms (max gap 17.8, p95 15.0, 0 placeholder steps), cards_scroll switch 77 ms / total 6674 ms (max gap 19.1), sort 280 ms, search 70 ms (49,898), export 509 ms (max gap 15.7)`.

## Target notes

Desktop and web share everything except four places. `ui/platform_desktop.go` holds the native menu bar, the folder-chooser export destination, Quit (`os.Exit`), Reveal (core helper) and `SystemFonts`; `ui/platform_web.go` replaces them with an in-memory CSV handed to `FilePicker.Save` (a browser download), no menu, and no-op Quit/Reveal, because the web shell has no menu bar, no file paths and no process to exit. `ui/driver_bench.go` has one `gguix:target web` block that discards the bench CSV in the browser. The Makefile marks `dev`/`build`/`package`/`run`/`bench`/`cross-*` as desktop and `web`/`web-remote`/`bench-web`/`serve` as web. One shared rule came from the web: nothing may block the UI goroutine (prefs are loaded in `main` and saved on a goroutine), since a blocking fetch there kills the wasm runtime.
