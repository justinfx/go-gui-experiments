# Friction log: IUP-Go

Times are approximate wall clock, 2026-11-15.

### 09:12 Pin main commit
- Step: add the dependency
- Command/API: `go get github.com/gen2brain/iup-go/iup@cbcdf09`
- Outcome: resolved to v0.32.1-0.20261114161355-cbcdf09dde04, 10 s of download
- Time lost: 0
- Category: toolchain
- Resolution: none needed

### 09:15 Docs are not in the module cache
- Step: verify APIs against the pinned version
- Command/API: `ls $(go env GOMODCACHE)/github.com/gen2brain/iup-go/iup@*`
- Outcome: the `iup` module has Go bindings and C sources but not the `docs/` folder; element docs (iup_table.md etc.) had to be fetched from GitHub at the pinned commit. `go doc` works well for the Go API.
- Time lost: 5
- Category: docs-gap
- Resolution: fetched docs for the pinned commit into a scratch directory; read C sources in `iup/external/src` for behavior

### 09:20 Table has no scroll position or scroll callback
- Step: table visible range and "top visible row"
- Command/API: IupTable attributes (docs/elem/iup_table.md, `external/src/cocoa/iupcocoa_table.m`)
- Outcome: no attribute reports the top row and no callback fires on scroll. SHOW scrolls the minimum distance, so a row is not made the top row by itself.
- Time lost: 20
- Category: framework-limitation
- Resolution: the visible range is inferred from the rows VALUE_CB is asked for (flushed by a 25 ms timer, a full REDRAW re-measures the height). Scroll-to-top shows the row below the view first, then the target. This is an approximation and is the least certain part of the app; see capabilities.md.

### 09:25 Table cells can only show images
- Step: custom cells (stock bar, status badge, stars)
- Command/API: `SHOWIMAGE`, `IMAGE_CB`
- Outcome: Table can show one image per cell next to the cell text but cannot draw into a cell, and cannot color text per cell in virtual mode. Matrix/MatrixEx (drawn) was not tried.
- Time lost: 10
- Category: framework-limitation
- Resolution: small RGBA images rendered in Go with the standard library, registered with IupImage; the badge is a colored pill swatch with the status text beside it (not text inside the pill)

### 09:28 Callbacks from goroutines panic
- Step: UI-thread mechanism
- Command/API: `iup.PostMessage(ih, "", 0, func())` with `POSTMESSAGE_CB`
- Outcome: works; attribute calls from other goroutines panic by design. The binding needs `runtime.LockOSThread()` in `init` so the main goroutine stays on the main thread.
- Time lost: 0
- Category: api-confusion
- Resolution: documented in capabilities.md

### 09:30 Cocoa application menu quits without Go cleanup
- Step: Quit
- Command/API: Cocoa driver auto-creates an application menu whose Quit calls `terminate:`
- Outcome: that path exits the process without returning from MainLoop, so the "save prefs on quit" step does not run from it. Prefs are still saved by the 500 ms debounce. The app's own Help > Quit (Ctrl+Q on Linux) saves and exits normally. Not verified by hand.
- Time lost: 5
- Category: framework-limitation
- Resolution: recorded as a gap

### 09:35 Error text color
- Step: validation messages
- Command/API: n/a
- Outcome: IUP has no theme error/danger color
- Time lost: 0
- Category: framework-limitation
- Resolution: used the section 4.5.1 low-stock red for the current theme

### 09:40 Tests
- Step: automated tests
- Command/API: `go test` with `runtime.LockOSThread`, `iup.Open`, `iup.Map` (not Show), `iup.MainLoop` on the main goroutine, tests in goroutines using PostMessage
- Outcome: works on the first try, 12 of 12 pass. No display is shown but a window server connection is needed.
- Time lost: 0
- Category: n/a
- Resolution: recorded

### 09:45 CGO_ENABLED=0 and cross builds
- Command/API: `CGO_ENABLED=0 go build`, with GOOS=linux and GOOS=windows
- Outcome: all build in about 1 s; the macOS binary (7.1 MB) started and showed its window (screenshot taken). It links only system frameworks via purego, no extra library file was needed on the host.
- Time lost: 0
- Category: n/a
- Resolution: recorded

### 09:41 Web: emsdk and the wasm module
- Step: install emsdk, build the IUP module, build the Go app for js/wasm
- Command/API: see toolchain.md (`git clone emsdk`, `emsdk install latest`, `bash build-wasm.sh -O -m`, `GOOS=js GOARCH=wasm go build`)
- Outcome: emsdk installed in about a minute (1.9 GB). `build-wasm.sh` fails under `sh` on macOS ("syntax error near unexpected token") and needs bash; it also assumes the repo layout and writes into its own directory, which is read-only in the module cache. Module built in 13 s from a copy.
- Time lost: 8
- Category: toolchain
- Resolution: Makefile copies the module to `~/iupwasm` and runs the script with bash
- Evidence: `iup/external/wasm/README.md`, `build-wasm.sh`

### 09:43 Web: cross-origin isolation and the core server
- Step: serve the build with `core/cmd/server -static`
- Command/API: iup/external/wasm/README.md ("the server must send COOP/COEP")
- Outcome: the core server sends neither header, so `SharedArrayBuffer` is undefined and the page stays blank. Core is read-only.
- Time lost: 12
- Category: core-issue
- Resolution: a small service worker (`web/coi.js`, `web/coi-sw.js`) adds the headers and the page reloads once. Works in Chrome; it did not register in the desktop app's embedded browser pane (service worker fetch error), so the web target was checked with Playwright and Google Chrome. Also needed an empty `resources.json` because the server answers unknown paths with `index.html`.

### 09:44 Web: the app runs in a Worker
- Step: download, file pick, clipboard, bench flag
- Command/API: `iup/external/wasm/web/index.html` creates `new Worker('worker.js')`
- Outcome: no `document`, no `localStorage`, no clipboard API in the Go code. The first export panicked in `document.createElement`. `core.BenchEnabled` reads `location.search` of the Worker (the worker URL, not the page).
- Time lost: 15
- Category: framework-limitation
- Resolution: `web/gguix-main.js` on the main thread handles download, file picker and clipboard requests posted by the app; the loader's `new Worker('worker.js')` is patched to pass `location.search` so `?bench=1` reaches the Worker. Preferences cannot persist on the web: core's `LocalStorageStore` needs `localStorage`, which a Worker lacks (gap, core cannot be changed).

### 09:46 Web: blocking dialogs
- Step: confirm dialog, host panel
- Command/API: `IupAlarm`, `IupPopup`
- Outcome: the wasm README says they do not block. Replaced by a modeless IUP dialog with callbacks (`platform_web.go`); the confirm hook in the model is callback based on every target.
- Time lost: 5
- Category: framework-limitation
- Resolution: done

### 09:48 Radio toggles showed no selection
- Step: macOS toolbar Table/Cards radio
- Command/API: `VALUE=ON` on a Toggle inside IupRadio after `IupMap`
- Outcome: both toggles OFF after startup. Setting it again after the window is shown works (test confirmed). Fixed by re-applying the view controls after `ShowXY`.
- Time lost: 12
- Category: runtime-bug
- Resolution: `a.uiViewControls()` after `iup.ShowXY` in `main_desktop.go` and `main_web.go`

### 09:50 Zbox pages
- Step: switch Zbox pages
- Command/API: `VALUE_HANDLE` on IupZbox
- Outcome: had no effect; `SetHandle(name)` plus `VALUE=name` works
- Time lost: 5
- Category: api-confusion
- Resolution: `showPage` helper in `ui.go`. This also fixed the missing export progress bar.

### 09:55 Linux
- Step: GTK3 build, tests and interactive checks in the container
- Outcome: builds; tests 11/12 by default, 12/12 with `GTK_THEME=Adwaita:dark` (T7 needs a GTK theme with a dark variant: IUP derives DARKMODE from the theme colors and the container has none). Table cell images do not show on GTK (see capabilities.md). `wlinput.py` needs `xkbcli`, which the image lacks.
- Time lost: 25
- Category: toolchain
- Resolution: see results/iupgo/linux.md

### Review pass: table was slow on sort and search
- Step: profile the quick bench (sort 3,424 ms, search 3,422 ms, jump 253 ms)
- Command/API: Go timing around `rebuildQuery` with the default 1M-row dataset, then around `NUMLIN`
- Outcome: core answers in 50 to 260 ms. The 3.4 s was `NUMLIN` going from 1,000,000 to 0 on every query change (removing rows from a native table costs about 3.4 us per row on Cocoa; growing is cheap, creating a table is cheap).
- Time lost: 40
- Category: framework-limitation
- Resolution: a query change no longer shrinks the table (rows show placeholders until the new total arrives); a shrink of more than 5,000 rows swaps in a fresh table widget (`tableReplace`). Also: the pager is asked once per row (not once per cell), cell images are cached by value key (no string building in the hot path), and a pager change that touches no visible row skips the REDRAW. After the fix (smoke bench, one run): sort 365 ms, search 85 ms, jump 265 ms.

### Review pass: table scroll, Table vs MatrixEx
- Step: smoke-bench the one-SHOW change, then build the table on IupMatrixEx
- Outcome: one SHOW per step did not help (Table scroll 11.5 s total, max gap 143 to 158 ms, p95 about 120 ms in three runs): the cost is the native table's layout and cell callbacks per step, not the number of SHOW calls. IupMatrixEx (build tags `ctrl,matrix`, `table_matrix.go`) in callback mode with `ORIGIN` (exact top row), `SCROLLTOP_CB` and `RESIZEMATRIX_CB` (visible range, no inference) and `DRAW_CB` (stock bar, status pill with its text inside, stars drawn directly, no images) scrolls the same script in 3.2 s with max gap 2 to 10 ms.
- Two Matrix costs found and avoided: scrolling a deep matrix back to the top and shrinking a large one cost about 0.4 s each in place, so those cases swap in a fresh matrix (`matrixReplace`). Matrix also calls `ENTERITEM_CB` on focus, so selection comes from `CLICK_CB` and from `ENTERITEM_CB` only after Up/Down/PgUp/PgDn.
- Both variants stay selectable: `make ... TABLE=native|matrix` (Go tags `ctrl,matrix` for matrix). **Matrix is the default** because it scrolls far better. Web keeps the native (DOM) Table.
- Smoke numbers (macOS, rough, other work possible):

| Event | Table (`TABLE=native`) | MatrixEx (default) |
|---|---|---|
| ready | 659 ms | 1,003 ms |
| jump | 234 ms | 85 ms |
| scroll total | 11,697 ms | 3,218 ms |
| scroll max / p95 gap | 143 / 121 ms | 2 / 0.4 ms |
| scroll placeholder steps | 0 | 25 |
| sort | 359 ms | 505 ms |
| search | 65 ms | 86 ms |
| export max gap | 78 ms | 0.1 ms |

- Trade-offs of Matrix: it is a drawn control (grid lines, centered titles, own look, not the native table), the 25 placeholder steps come from scrolling faster than the pager can fetch at 50 ms latency (the Table was too slow to outrun it), and ready, sort and search are a little slower.
- Category: framework-limitation
- Resolution: Matrix default

### Review pass: Table images on GTK
- Step: find why stock bar, badge and stars do not appear on GTK3
- Outcome: not found. Evidence: `IMAGE_CB` is called for every cell; for loaded rows it returns `gxbar-l-182`, `gxbadge-l-0`, `gxstars-l-3-1`; `IupDrawGetImageInfo` resolves them to 80x20, 30x20 and 68x20 at 32 bpp; images created before the window is mapped behave the same; `FITIMAGE=YES` and `NO` both show nothing. The GTK driver code (`iupgtk_table.c` virtual model `get_value`, the pixbuf renderer and `gtkTableImageCellDataFunc`) reads correct, so the failing step is inside GTK drawing and was not isolated.
- Time lost: 35
- Category: runtime-bug
- Resolution: unresolved for the native Table, recorded as gap (macOS and web draw them). The default Matrix variant draws the cells itself (no images) and shows the bars and pills on GTK (`linux/sway/start.png`).

### Review pass: binding leak
- Every `VALUE_CB` and `IMAGE_CB` return is a `C.CString` that the Go binding never frees (`bind_callbacks.go`, `goIupTableValueCB`), so each drawn cell leaks a small string. Fewer redraws reduce it; not fixable from the app.
- Category: framework-limitation

### Hands-on pass (macOS): notification blocks the UI thread
- `iupdrvNotifyShow` (`iupcocoa_notify.m`) waits on semaphores on the calling (main) thread: up to 2 s for the settings query, up to 30 s for the first authorization request, then 2 s for delivery. The window shows the busy cursor for that time, which matches the "spinner until the notification is clicked" report. IUP has no asynchronous notify call.
- Mitigation in the app: the progress bar and cursor are restored (`CURSOR=ARROW` plus `IupFlush`) before the notification is posted, and the call runs from a posted message. The block itself is a framework limit; it remains for the first-run authorization prompt.
- Category: framework-limitation

### Hands-on pass: widget border color and date picker background
- Native controls on macOS (text, list, date picker) have no border color attribute, so low-contrast borders in the light theme cannot be changed from the app. `IupDatePick` is a generic composite: it copies `TXTBGCOLOR` into `BACKCOLOR` once in its create method, so it keeps the startup theme (black in light). The app now re-sets `BACKCOLOR`, `BGCOLOR` and `FGCOLOR` on every theme change.
- Category: framework-limitation (borders), api-confusion (date picker colors)

### Hands-on pass: text entry vertical alignment
- Cocoa `IupText` is top aligned and has no vertical alignment attribute (`ALIGNMENT` is horizontal only). The app fixes the height with `RASTERSIZE` and centers the entries in the toolbar; the glyphs still sit near the top of the field. Not fully fixable without a custom control.
- Category: framework-limitation

## Summary

- Wall time to first window: about 35 minutes (09:12 to about 09:47, times above are approximate).
- Wall time to feature complete: about 2 h 30 min including web, Linux and the follow-up fixes.
- Entries per category: toolchain 3, docs-gap 1, framework-limitation 6, api-confusion 2, core-issue 1, runtime-bug 1.
- Biggest pain points: (1) Table has no scroll position or top-row control and on GTK shows no cell images, (2) the web loader runs the app in a Worker with no DOM, localStorage or COOP/COEP from the core server, (3) docs are not shipped in the module cache.

### 15:15 Two IupRadio groups in one dialog clear each other on macOS
- Step: hands-on fix, clicking a row cleared the Table/Cards selector
- Command/API: `iup.Radio(iup.Hbox(toggles...))` twice (toolbar view selector, form status), `iupcocoa_toggle.m` (`setButtonType:NSButtonTypeRadio`, `setAction:@selector(myToggleClickAction:)` for every toggle)
- Outcome: AppKit groups radio-type NSButtons that share a superview and an action. IupRadio is a void container, so both groups' toggles land in the same native parent and every IUP toggle uses the same action; loading the form (VALUE=ON on a status toggle) deselected the Table/Cards toggle. Confirmed with a standalone AppKit program (no window): two radio buttons with different targets in one superview deselect each other; a different superview does not. The docs do not mention it.
- Time lost: 15
- Category: framework-limitation
- Resolution: each group is wrapped in `iup.BackgroundBox` (a native container, so its own superview). Test T4 asserts the selector keeps its state after a selection. The headless test cannot prove the Cocoa grouping itself, only that the state is kept.

### 15:25 Label toggled between FLOATING IGNORE and NO shows clipped
- Step: hands-on fix, "Select an item to edi" after a row click
- Command/API: `FLOATING`/`VISIBLE` on a Label inside a Vbox
- Outcome: a label that starts out FLOATING=IGNORE is left out of the box's size computation, and flipping it back later did not re-run layout with the label's natural size. Not reproduced here (no GUI run on the author's Mac); cause inferred from the layout code and the symptom.
- Time lost: 10
- Category: api-confusion
- Resolution: the hint stays in the layout (EXPAND=HORIZONTAL, FLOATING=NO) and only VISIBLE is toggled; the hidden row keeps one line of height. Test T4 asserts VISIBLE=NO and FLOATING=NO after a selection.
