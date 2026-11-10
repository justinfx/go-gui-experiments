# Capabilities: Wails v3.1.0

Status values: native / framework-drawn / fallback / gap / not feasible. "Verified" means exercised by an automated test or a run on this machine. The Mac was at the lock screen for the whole desktop session (screenshots were black), so desktop-only UI (native menu, native dialogs, OS notification, window size restore) was built against the Wails source and docs but NOT visually verified. Desktop startup, paging and the first two bench steps were verified through stdout.

| Requirement | Status | How it was done | Notes |
|---|---|---|---|
| Window title, size 1280x800, min 1024x640 | native | `WebviewWindowOptions{Title, Width, Height, MinWidth, MinHeight}` in `main_desktop.go` | Not visually verified. |
| Restore saved window size (desktop) | native | `OnWindowEvent(events.Common.WindowDidResize)` debounced, `noteWindowSize` writes core prefs; read back in `setupTarget` | Not verified: whether `Size()` and the options width are the same box (content vs frame). |
| Search field, 250 ms debounce | framework-drawn | shadcn `Input`, timer in `state.svelte.ts` | Verified (T3). |
| Categories popup with 8 checkboxes and Clear | framework-drawn | shadcn `Popover` + `Checkbox` | Label shows `Categories (n)`. |
| Status filter dropdown | framework-drawn | shadcn `Select` | |
| Go-to-row field and Go button | framework-drawn | number `Input`, `app.goToRow` clamps, scrolls, selects | Works in both views and in the bench. |
| View switcher Table / Cards | framework-drawn | two shadcn `Button`s (segmented look) | Verified (T11). |
| Export CSV button, Theme dropdown | framework-drawn | `Button`, `Select` | |
| Split view with draggable divider | fallback | A custom 4 px pointer-drag divider in `App.svelte`; shadcn-svelte has no resizable component in the requested list | Draggable, 65/35 default. |
| Status bar text, export progress, Cancel, identity, Host button | framework-drawn | `StatusBar.svelte`, shadcn `Progress` | Verified (T3, T8, T12). |
| Table columns, widths, alignment | framework-drawn | shadcn table classes on real `<table>` elements, absolute positioned window of rows | |
| Column resize by dragging | fallback | Own pointer-drag handle on each header cell (no widget support) | Not covered by a test. |
| Sort by header with ▲/▼ | framework-drawn | Header button, glyph in label | Verified (T2). Stock column sorts by quantity. |
| Virtualized table | framework-drawn | `@tanstack/svelte-virtual` inside `VirtualScroller.svelte` | See "Scaled scroll" below. |
| Scaled scroll for 1M rows | fallback | `scaled-scroll.ts`: browsers cap element height near 33.5 million px; 1M rows at 37 px or 1M cards at 104 px exceed it, so scrollTop is mapped to a virtual offset with an exact anchor for programmatic jumps | A real friction point; see friction.md. |
| Row selection, Up/Down keys, selected style, `…` placeholders, `No items match.` | framework-drawn | `data-state=selected` row style from shadcn | Selection by click verified (T4, T9). |
| Custom cell: stock bar | framework-drawn | `StockBar.svelte`, inline SVG, 80x8, rounded | Verified (T10, light and dark). |
| Custom cell: status badge | framework-drawn | `StatusBadge.svelte`, pill with inline colors from core.Colors | Verified (T10). Colors come from Go (`Palette` method), none hand-copied. |
| Custom cell: stars | framework-drawn | `Stars.svelte`, inline SVG star path, half star by clip path | Verified (T10). |
| Cards view item | framework-drawn | `Card.svelte` + `CardList.svelte` with `@tanstack/svelte-virtual`; 96 px card in a 104 px slot; tile, 4 line column, right column; selected 2 px primary border | Verified (T11). Uses plain divs with shadcn theme tokens, not the shadcn `Card` component (its padding and ring styles do not fit a fixed 96 px layout). |
| Form: ID, SKU, Updated read-only text | framework-drawn | spans | |
| Form: Name, Price text fields | framework-drawn | shadcn `Input` | |
| Form: Category dropdown | framework-drawn | shadcn `Select` | |
| Form: Quantity spin box | fallback | shadcn `Input type=number` (browser spinner, min 0 max 1,000,000 step 1); no shadcn spin box | Validation by core. |
| Form: Status radio group horizontal | framework-drawn | shadcn `RadioGroup` | |
| Form: Rating slider with label | framework-drawn | shadcn `Slider` 0 to 50 tenths, label `(t/10).toFixed(1)` | The label is JS formatting of tenths, identical to `FormatRating` for 0.1 steps. |
| Form: Featured checkbox | framework-drawn | shadcn `Checkbox` | |
| Form: Restock date picker + Clear | framework-drawn | shadcn `Calendar` in a `Popover`, `@internationalized/date` | |
| Form: Notes 4 lines | framework-drawn | shadcn `Textarea rows=4` | |
| Dirty state, Save, Revert, Unsaved changes | framework-drawn | `FormModel.dirty` | Verified (T5, T9). |
| Save with validation messages under fields | framework-drawn | Go `Update` returns field errors; text in `text-destructive` | Verified (T5, T6). Go test proves DataService.Update is not called on invalid input. |
| Confirm dialog (discard) | framework-drawn | shadcn `AlertDialog`, in-app (not a native sheet) | Verified (T9). |
| Copy Row / clipboard | native (desktop), fallback (web) | Desktop: `app.Clipboard.SetText` in `platform_desktop.go`; web: `navigator.clipboard` | Message "Row copied" in the status bar for 2 s. Not verified on desktop. |
| Import Notes file dialog | native (desktop), fallback (web) | `app.Dialog.OpenFile()` filter `*.txt`; web: hidden `<input type=file accept=.txt>` | Go test covers the 64 KB and 500 character rules. Error dialog is native on desktop (`app.Dialog.Error()`), in-app AlertDialog on web. |
| Export save dialog | native (desktop), fallback (web) | `app.Dialog.SaveFile()` default name `gguix-export.csv`; web: streamed `/export/<id>` read by `fetch`, then a Blob download | Web memory use is the whole CSV (about 100 MB for 1M rows). |
| Export in background, progress, cancel, partial file deleted | native | Go goroutine, `export:progress/done/error/cancelled` events | Go test verifies the partial file is removed and a new export can start; T8 verifies the UI. |
| Export complete notification | native (desktop), fallback (web) | `services/notifications` on desktop; in-app toast (sonner) plus the browser Notification API if already granted on web | The notification service refuses to start outside an app bundle on macOS (it stops the whole app), so it is only registered inside `.app` (see friction.md). Not verified on screen. |
| Native menu bar and roles | native | `app.Menu.New()`, `AddSubmenu`, roles for Edit and Hide items | Not visually verified. Web has a shadcn `DropdownMenu` menu bar instead. |
| Shortcuts Cmd+Q/E/O/Shift+C/1/2/R/F/I | native (desktop), fallback (web) | `SetAccelerator("CmdOrCtrl+...")`; web: a window `keydown` handler | Menu state (radio checks, enabled) is pushed by the frontend with `SetMenuState`. |
| About dialog | framework-drawn | shadcn `Dialog` opened from a menu event | Lines: title, framework version (from build info), Go version, BuildMarker, UIMarker. |
| Theme modes and live system switching | native | `mode-watcher` (follows `prefers-color-scheme` of the webview, live) | Theme dropdown and menu stay in sync. T7 verifies dark class and prefs. Live OS switching was not exercised. |
| UI thread mechanism | native | Frontend code runs in the webview; Go to UI calls are Wails events (`Event.Emit`) and Go dialogs/menu calls are marshalled by Wails to the main thread | The bench probe is a JS `setTimeout` drift probe. |
| Native prefs facility (unused) | n/a | Wails has no preferences API; the closest is the `kvstore` service (JSON file). Core `PrefsStore` is used as required. | |
| Host panel, desktop | native | `host.Current` read in Go; Source `native`; actions through `host.RevealInFileManager` | Wails offers no reveal-in-Finder API (`Browser.OpenFile` opens a file); core helper used. Not visually verified. |
| Host panel, web | fallback | `host.Server`; Source `server-process`; buttons disabled with title `Not available in the browser` | Verified (T12). |
| Web target (server mode) | native | `-tags server`, `wails3 task build:server`, `WAILS_SERVER_PORT` | Port 8090 was occupied on this host by other processes, so the server was run and tested on 8191/8193; `make web` uses 8090. |
| cgoprobe desktop | PASS | `go build -tags production,cgoprobe` plus the export tests with the C checksum | |
| cgoprobe web | PASS | `go build -tags server,production,cgoprobe`, started, `/health` OK | |
| Cross build windows/amd64 | PASS | `wails3 task build GOOS=windows ARCH=amd64`; native Go cross compile, no cgo, no Docker used; output 12.2 MB `gguix-wails.exe` | Not run on Windows. |
| Cross build linux/amd64 | see friction.md | Docker route. Attempt 1 failed: image has Go 1.26, core needs 1.27. Attempt 2: edited Dockerfile to golang:1.27; failed: `gcc: unrecognized option -m64` because the image was arm64 (Apple Silicon host) and linux-amd64 uses plain gcc. Attempt 3: amd64 image built with `--platform linux/amd64`. Final result recorded at the end of this file. | |
| Package | native | `wails3 task package` then rename to `GGUIX-wails.app` and ad-hoc sign | Result recorded at the end. |
| T1 to T12 | see below | Playwright against server mode (real Go core, Chrome) | |
| T1 start and load | pass | e2e | |
| T2 sort | pass | e2e | |
| T3 search | pass | e2e | |
| T4 selection | pass | e2e | |
| T5 validation | pass | e2e plus Go test | |
| T6 save | pass | e2e | |
| T7 theme and prefs | pass | e2e (reads prefs.json) | |
| T8 export cancel | pass | e2e (UI) and Go test (partial file removed) | The web target has no file, so the file part is only covered by the Go test. |
| T9 dirty guard | pass | e2e | |
| T10 custom cells | pass | e2e, light and dark | |
| T11 cards | pass | e2e | |
| T12 host info | pass | e2e (Source server-process, Username, status bar identity) | The desktop `native` Source is covered by the Go test `TestHostInfoDesktop`, not by a UI test. |
| Bench, web (`?bench=1`) | works | Headless Chrome, default 1M rows, 50 ms latency: all eight metrics printed (see friction.md summary) | |
| Bench, desktop | partial | `make bench` printed `ready` and `jump`, then stalled | The screen was locked, so the WKWebView was occluded and its timers did not run. Needs an unlocked desktop to complete. |

## Target notes

The desktop and web targets share all of `frontend/`, `service.go`, `export.go`, `view.go`, `main.go` and `marker.go`. What differs is isolated in `platform_desktop.go` and `platform_web.go` (dialogs, clipboard, notifications, reveal, host identity), `main_desktop.go` (window, notification service, quit and prefs flush) with `menu_desktop.go` (the native menu), and `main_web.go` (server options and the streaming `/export/<id>` HTTP handler). In the frontend, target differences are branched on `AppInfo.target` and marked with `gguix:target web` blocks in `App.svelte` (keyboard shortcuts and the in-page menu bar) and `frontend/src/lib/components/web/WebMenuBar.svelte`. The export flow differs because a browser cannot write a file: the web target streams the CSV over HTTP and saves it as a Blob download, while the desktop writes the file in Go. Server mode reads prefs from the server's file store, so all browsers share them.

## Final cross build and package results

- cross-windows: PASS, `bin/gguix-wails.exe` 12.2 MB, no Docker.
- cross-linux: PASS, `bin/gguix-wails-linux-amd64` 11.4 MB, via an amd64 `wails-cross` image (Makefile target does this).
- package: PASS, `bin/GGUIX-wails.app` (11 MB, ad-hoc signed, verified with `codesign --verify`). It starts with the notification service registered and printed `ready` and `jump` bench metrics.
