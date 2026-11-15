# Capabilities: IUP-Go

Status values: native / framework-drawn / fallback / gap / not feasible. "Verified" means seen working (macOS, Chrome web, or Linux container) in this pass; unverified items are implemented from the pinned docs and source.

| Requirement | Status | How | Notes |
|---|---|---|---|
| Table, 1M rows, virtual | framework-drawn (default) / native | default: `IupMatrixEx` callback mode, `VALUE_CB`/`DRAW_CB` from `core.Pager` (`table_matrix.go`, tags `ctrl,matrix`); `TABLE=native`: `IupTable` `VIRTUALMODE=YES`, `VALUE_CB`/`IMAGE_CB` (`table_native.go`); web uses the native Table | verified macOS (both), Linux GTK (both start; Matrix shows custom cells), web (Table) |
| Custom cells (Matrix, default) | framework-drawn | `DRAW_CB` with IupDraw: stock bar, status pill with the text inside, stars | no images, exact section 4.5.1 colors; works on macOS and GTK |
| Custom cells (native Table) | fallback | `IMAGE_CB` images rendered in Go (`cells.go`), registered per theme | Table cannot draw into cells. Badge is a colored pill swatch with the status text beside it, not text inside the pill. 1x pixels, soft on Retina. Matrix not tried. T10 checks pixels and specs. Shown on macOS and web. **On Linux GTK3 the images did not appear** (cells show plain text, "182 182" in Qty and Stock); GTK's table code has image support, cause not found |
| Visible range / top row (Matrix) | native | `ORIGIN` sets the exact top row; `SCROLLTOP_CB` and the control height give the range | smoke bench scroll max gap 2 to 10 ms |
| Visible range / top row (native Table) | fallback | inferred from `VALUE_CB` requests (row fetched once per row, cached); top row by one SHOW (bottom-row trick down, direct up) | no scroll attribute or callback in Table. Status bar showed "rows 3-31" on GTK for a table at the top, so the inference is off by two rows there. Bench not run |
| Column resize | native | `USERRESIZE=YES` | not hand-verified |
| Header sort with arrows | native | `SORTABLE=YES`, `SORT_CB` returns IGNORE, app sets titles | verified macOS, web, GTK |
| Row selection, Up/Down | native | `ENTERITEM_CB`, `FOCUSCELL` | selection and form fill verified on macOS, web, GTK (click) |
| Cards view | framework-drawn | `IupCanvas` with vertical scrollbar in card units, cards drawn with IupDraw (`cards.go`) | IUP has no virtual list with custom items. Verified macOS dark and web. Wheel/keys not hand-verified |
| Form controls | native | Text, List dropdown, Text with `SPIN`, Radio of Toggles, Val, Toggle, DatePick + Clear, MultiLine (`form.go`) | DatePick has no empty state, so a "none" label tracks whether a date is set. The form is wider than its pane at 1280 px on GTK (Discontinued label clipped) |
| View switcher | native | two Toggles in `IupRadio` | showed no selection on macOS until re-applied after the window is shown (fixed, see friction.md) |
| Split view | native | `IupSplit` | draggable, not hand-verified |
| Menus | native | `IupMenu` (`menu_desktop.go`); web has none | Verified on GTK (File menu opens, Export/Import/Reveal items with shortcuts shown). macOS menu not hand-checked; Cocoa adds its own app menu |
| Shortcuts | native | `\tCtrl+X` in item titles | Verified on GTK under sway: Ctrl+E (export dialog), Ctrl+I (host dialog), Ctrl+Shift+C (copy row), Ctrl+Q (process exits) |
| File dialogs | native | `IupFileDlg` | Verified on GTK: native GTK save dialog, name pre-filled `gguix-export.csv`, CSV filter; export wrote 347,958 bytes. Typing over the selected stem keeps `.csv`, so a typed path gets a doubled extension unless the whole name is selected. Web: browser download / file input via main thread |
| Confirm dialog | native / fallback | desktop `IupAlarm`; web a modeless IUP dialog | T9 uses a hook; not seen on screen |
| Clipboard | native | `IupClipboard` TEXT; web `navigator.clipboard` via main thread | Verified on GTK: `wl-paste` returned the CSV row |
| Notifications | native | `IupNotify`, ERROR_CB falls back to a status message | Verified on GTK: dunst showed "Export complete / 3,000 rows written", app name empty. macOS (unbundled) not checked. Web: status message |
| Theme modes | native | `APPEARANCE` global, `APPEARANCECHANGED_CB`; the app's own palette follows the choice immediately | macOS dark verified. On GTK3 IUP's `DARKMODE` stayed NO after `APPEARANCE=DARK` even with Adwaita-dark installed (only a dark `GTK_THEME` at start changes it); T7 now checks the app palette and only logs `DARKMODE` |
| Live system theme | gap (not demonstrated) | `APPEARANCECHANGED_CB` | not shown on macOS; in the container `gsettings set color-scheme prefer-dark` changed nothing (no portal or dark theme) |
| UI-thread mechanism | native | `IupPostMessage` + `POSTMESSAGE_CB`; `runtime.LockOSThread()` in init (desktop) | other goroutines panic if they touch attributes. In the browser the app is one Worker thread |
| Native prefs facility (unused) | n/a | `IupConfig` (ini file in the user config dir) | core `PrefsStore` used instead. Web: preferences do not persist (Worker has no localStorage) |
| Host panel, desktop | native | dialog with `IupGridBox` (`dialogs.go`) | verified on GTK: Source native, root, hostname; reveal buttons call `xdg-open /tmp/pf-sway` and `xdg-open /tmp` (parent folder of the last export, file not selected). IUP has no reveal call, so `core/host` is used |
| Host panel, web | native | same dialog, modeless, `HostActions=false` | in-process: Source unavailable, buttons disabled with tip. web-remote: Source `server-process` from `/api/host` (checked in Chrome) |
| Window class / app_id | native | taken from the binary name | X11 `WM_CLASS gguix-iupgo`, sway `app_id gguix-iupgo` |
| Packaging | native | `iupkg package --os darwin` | .app 5.5 MB, ad-hoc signed |
| `CGO_ENABLED=0` build | native | prebuilt library loaded at run time via purego | 7.1 MB, started on macOS |
| `web` | native | IUP Emscripten module + Go js/wasm app (`web/`, Makefile) | works in Chrome; app 7 MB wasm + 0.65 MB `iup.wasm`; needs the service worker for COOP/COEP |
| `web-remote` | native | `-tags remote`, served by the core server | works in Chrome; host info from the server |
| Web download | native | main-thread helper (`gguix-main.js`); progress and Cancel in IUP | verified: `gguix-export.csv` downloaded |
| `bench-web` | untested | same build, `?bench=1` passed to the Worker | not run (no measurement allowed in this pass) |
| cgoprobe desktop | PASS | `-tags cgoprobe` | |
| cgoprobe web | FAIL | the wasm build cannot take the cgo checksum file | `undefined: cgoprobe_requires_cgo_but_cgo_is_unavailable_for_this_target` |
| cgoprobe web-remote | PASS | wasm client without the tag, server with it | server built; app not re-run against it |
| cross-windows / cross-linux | PASS | `CGO_ENABLED=0 go build`, about 1 s, 7.6 MB / 7.3 MB | not run on those systems |
| Linux runtime | PASS (partial) | see `linux.md` | x11 and sway |
| Qt build | not feasible | not attempted | |
| Tests T1-T12 | native | real toolkit, mapped window, callbacks driven (`app_test.go`; the Table pixel checks are in `app_native_test.go`) | 12 of 12 on macOS and on Linux x11 for both variants. Web: not run |

## Review pass additions

| Item | Status | Notes |
|---|---|---|
| Data paging | native | every view reads `core.Pager` (200-row pages, prefetch, 2 in flight); the table asks once per row, not per cell; a query change keeps the table and shows placeholders until the new page arrives |
| Large table shrink | fallback | `NUMLIN` decrease costs about 3.4 us per row, so a decrease of more than 5,000 rows swaps in a new table widget |
| Smoke bench | measured | see the table in friction.md: Matrix (default) scroll 3.2 s, max gap 2 ms, sort 505 ms, search 86 ms; native Table scroll 11.7 s, max gap 143 ms, sort 359 ms, search 65 ms |
| MatrixEx instead of Table | done, default | see friction.md |
| Linux x11 interactive | PASS | Ctrl+Shift+C, Ctrl+2 cards, Ctrl+E dialog, export file, notification, host reveal, Ctrl+Q |

## Target notes

Desktop-only files: `main_desktop.go`, `platform_desktop.go`, `menu_desktop.go`, `dialogs_desktop.go` and the `gguix:target desktop` block of the Makefile. Web-only files: `main_web.go`, `platform_web.go`, `menu_web.go`, everything under `web/`, and the `gguix:target web` block. The model and views (`app.go`, `ui.go`, `table.go`, `cards.go`, `cells.go`, `form.go`, `dialogs.go`, `export.go`) are shared and unchanged between targets. What differed: no menu bar and no blocking dialogs on the web, downloads/file pick/clipboard go through the main thread because the IUP loader runs the app in a Worker, and the web build needs the service worker for cross-origin isolation.
