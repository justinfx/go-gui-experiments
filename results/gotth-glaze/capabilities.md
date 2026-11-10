# GoTTH + glaze capabilities

Status values: native, framework-drawn, fallback, gap, not feasible.
"Native" here means the browser or the webview provides it as a real control, not that the OS draws it.
In this stack every visible control is HTML. Files named are relative to `apps/gotth/`.

## Section 5 requirements

| Requirement | Status | How it was done | Notes |
|---|---|---|---|
| Window title `GGUIX - GoTTH` | native | `w.SetTitle` in `cmd/desktop/main.go`; `<title>` in `views/page.templ` | |
| Window 1280 x 800, min 1024 x 640, restore saved size | native (desktop) | `SetSize(w, h, glaze.HintNone)` from prefs, then `glaze.HintMin` 1024 x 640; `static/desktop/desktop.js` posts the window size to `/prefs/window` 500 ms after a resize | glaze has no resize callback in the interface used here, so the page reports its own size. Launched window measured 1280 x 832 including the 32 pt title bar. |
| Toolbar: search with 250 ms debounce | native | shadcn `input`; `static/app.js` debounces (0 ms in bench mode) and fires `filters-changed` on the form | placeholder `Search name or SKU`, 280 px |
| Categories popup with 8 checkboxes, `Clear`, `Categories (n)` | framework-drawn | shadcn `popover` with `checkbox` items (`form="filters"` keeps them in the filters form after the popover moves to the body); label swapped out of band by the server | |
| Status dropdown | native | shadcn `native-select` | the shadcn `select` popup was not used; the native select is far simpler to drive and test |
| Go-to-row field and `Go` button | native | form posting to `/goto`; server clamps, loads the target page, selects the row | Enter submits; asks the dirty guard first |
| View switcher Table / Cards | framework-drawn | two shadcn `button`s as a segmented control (`aria-pressed`) | |
| Export CSV button, theme dropdown | native | shadcn `button`, `native-select` | |
| Split view with draggable divider | framework-drawn | shadcn-templ `resizable` (`PanelGroup`, `Panel`, `Handle`) in `views/page.templ`, 65/35 default, min 25/20 | Default build, web and desktop. Drag verified in the browser pane. Fix pass 2026-11-09. |
| Status bar (count, rows a-b, last fetch ms, export progress, identity, Host button) | framework-drawn | `views/page.templ` `StatusBar`; count and fetch time swapped out of band; rows range computed by `static/app.js` from the pages in the DOM; export area pushed over SSE | |
| Table: columns, widths, alignment | framework-drawn | shadcn `table` Header/Row/Head/Cell parts inside my own `<table>` with `table-layout: fixed` | The `table.Table` wrapper has an `overflow-x-auto` container that breaks scrolling, so I do not use it |
| Fixed header | fallback | the header is a separate table above the scroller, horizontal scroll synced by JS | A sticky `<thead>` made WebKit re-lay out 1,000 rows on every scroll step |
| Column resize | framework-drawn | drag grip (`colGrip`) on each header cell, `<col>` widths of both tables set by `static/app.js` `applyColWidths`, kept across swaps | Not persisted. Verified in the browser pane. Fix pass 2026-11-09. |
| Sort by header click, active header shows ▲ / ▼ | native | header cells post `/query` with the sort field; server toggles direction | |
| Reset scroll, keep selection only if in first page | native | `handleQuery` | |
| Windowed infinite scroll (200 per page, 5 pages in DOM, spacers) | fallback | server-rendered pages; sentinel and spacer elements load pages; the farthest page becomes a spacer of the same measured height (`static/app.js` `evict`, `toSpacer`). See friction.md for why `revealed` was replaced | The scrollbar represents only the loaded window (plus spacers). Recorded as required. |
| `…` placeholder rows | native | sentinel row shows `…` in every cell; spacers show `…` | |
| Single selection by click, Up/Down keys | native | delegated click and keydown on the scroller, `data-state="selected"` (the shadcn row style) | |
| Zero results `No items match.` | native | | |
| Custom cell: stock bar | framework-drawn (inline SVG) | `views/cells.templ` `StockBar`: 80 x 8 SVG, track and fill rects, fill width from `core.StockFraction`, low color class | colors come from `/palette.css`, generated from `core.Colors` |
| Custom cell: status badge | framework-drawn | shadcn `badge` with a `status-<Name>` class that sets the 4.5.1 colors | |
| Custom cell: stars | framework-drawn (inline SVG) | five 12 px SVG stars, half star is the filled path clipped with `clip-path: inset(0 50% 0 0)` | |
| Cards view item | framework-drawn | `views/list.templ` `ItemCard`: shadcn `card` with overrides (96 px, 1 px border, 2 px primary border when `data-state=selected`), category tile div, three text lines, status badge and Featured chip, price, stars, stock bar | same windowing as the table, mechanism is the same server-rendered pages. No visual compromise known. |
| Detail form: ID, SKU, Updated read-only text | native | `<span>` | |
| Name, Price, Notes | native | shadcn `input`, `textarea` (4 rows) | |
| Category dropdown | native | shadcn `native-select`, 8 options | |
| Quantity spin box | native | `<input type="number" min=0 max=1000000 step=1>` | |
| Status radio group, horizontal | framework-drawn | shadcn `radio-group` | hidden native radios carry the value |
| Rating slider 0.0 to 5.0 step 0.1 with value label | framework-drawn | shadcn `slider` with one thumb (`DefaultValue: []float64{v}`); label updated on `slider-change` | |
| Featured checkbox | framework-drawn | shadcn `checkbox` | |
| Restock date picker and Clear | framework-drawn | `popover` with shadcn `calendar` (single mode), hidden input `restock`, label button, `Clear` button | |
| Dirty state, `Unsaved changes`, Save and Revert enabled | fallback | `static/app.js` compares `FormData` snapshots (taken 30 ms after each form swap); after a failed save the server marks the form `data-force-dirty` | runs in the browser, as the brief allows |
| Save, validation messages under fields in the error color | native | `/save` parses, runs `core.ValidateItem`, calls Update only if valid; messages rendered with `text-destructive` | |
| Table row and card refresh after save | native | the response carries the row or card out of band (`RefreshedItem`) | |
| Revert and select-while-dirty confirm dialog | framework-drawn | shadcn `alert-dialog` `Discard unsaved changes?` with Discard and Cancel, opened by clicking a hidden trigger | |
| Copy Row | native (desktop) | `clipboard.WriteText` from `github.com/crgimenes/native/clipboard` through `/native/copy` (`cmd/desktop/native.go`), plus the Clipboard API in the page | transient `Row copied` text in the status bar for 2 s. Write then read round trip verified on macOS with a probe program. |
| Import Notes | native (desktop and web) | web: `<input type="file">`; desktop: `WebView.OpenFile` with a `txt` filter (glaze `dialog_darwin.go`, `dialog_windows.go`, `dialog_linux.go`) | The dialog is application modal, not a sheet. Dialog opening verified on macOS through System Events (a second window appeared); choosing a file was not exercised. |
| Export in the background, progress, cancel, notification | native | `GET /export` stream, SSE progress, cancel; desktop: `WebView.SaveFile` with a `csv` filter | Dialog opening verified on macOS; Escape cancelled it. Full export path covered by T8 with a fake native layer. |
| OS notification `Export complete` | native (macOS), gap (Windows, Linux) | `notify.Send` from `github.com/crgimenes/native/notify`; macOS uses `UNUserNotificationCenter` in a bundle and osascript `display notification` for a bare binary (notify_darwin.go). `notify_other.go` returns `ErrUnsupported`; `Notify` in `cmd/desktop/native.go` swallows that and the page shows its toast | `notify.Send` returned nil from a bare binary on macOS; the banner itself was not looked at. No Windows or Linux backend in native v0.1.18. |
| Menu bar, shortcuts | native (macOS), partial (Windows), fallback (Linux) | `github.com/crgimenes/glaze/menu` `menu.Set` in `cmd/desktop/menu.go`; items run `GGUIX.actions[...]` through `WebView.Eval`; Edit items use `Selector` (`cut:`, `copy:`, `paste:`, `selectAll:`); state pushed through bound `gguixMenuState` and the whole menu is rebuilt (items have no checked state, so the radio is a leading check mark and Export / Reveal use `Disabled`) | macOS verified through System Events: top level GGUIX, File, Edit, View, Help; clicking View > Cards ran the page action and moved the check mark. Windows: Win32 menu bar, **shortcuts not wired** (menu_windows.go header), so the page keydown handler in `static/app.js` provides them. Linux: **no menu bar** (menu_linux.go header); the shell does not call `menu.Set` there, the in-page menubar is shown and `static/app.js` shortcuts apply. The macOS app menu title shows the process name `gguix-gotth`, not `GGUIX`. Windows and Linux compile only. |
| About dialog | framework-drawn | shadcn `dialog`: title, framework versions, `runtime.Version()`, `BuildMarker 0`, `UIMarker 0` (`views/marker.templ`) | |
| Theme System / Light / Dark, no restart | native | `dark` class on `<html>`, cookie `gguix-theme`, inline head script for `prefers-color-scheme` | dropdown and menu stay in sync (`G.setTheme`) |
| Live switching of System with the OS | native | `matchMedia` change listener in the head script | |
| Prefs saved 500 ms after last change and on quit | native | `Server.update` debounce; `Flush` on SIGINT (web) and when the window closes (desktop) | |
| Prefs restored before the first query, `FindRow(..., 2000)` for the selection | native | `handleIndex` on the server | skipped in bench mode |
| Threading mechanism | native | no UI thread to manage: every UI change is an htmx swap in the page. The desktop shell calls `w.Dispatch` only to terminate the window. | |
| Native prefs facility (unused) | n/a | the browser equivalent is `localStorage`/cookies; the desktop shell has none. The core `PrefsStore` is used, stored server side. | |
| Host panel | native (desktop), server-process (web) | `GET /host` renders `host.Current` (desktop) or `host.Server` (web) as the two-column list, Source row always shown, Notes below; status bar shows `<Username>@<Hostname>` | |
| Reveal Config Folder, Reveal Last Export | native (desktop), disabled (web) | desktop: `openurl.Reveal` from `github.com/crgimenes/native/openurl` (Finder selects the file; Explorer selects on Windows; Linux opens the folder), reached through an optional `Reveal` method on the shell's `Native` (`Server.reveal` in `handlers_desktop.go`, falls back to the core helper); web: disabled with `Not available in the browser` | Reveal Last Export is disabled until an export finishes. `openurl.Reveal("go.mod")` returned nil on macOS (probe); Finder was not looked at. |

## Section 6 (GoTTH specific)

| Requirement | Status | How it was done | Notes |
|---|---|---|---|
| Go net/http server in `cmd/web`, core in process | native | `cmd/web/main.go` using `backend.Open` | |
| shadcn-templ components via its CLI | native | components added with `shadcn-templ add`; two local edits recorded in friction.md (`components/baseui/lifecycle.js`) | npm was not needed |
| Windowed table and cards | fallback | see above | |
| Selection and form load/save through htmx fragments | native | | |
| Export over SSE | native | | |
| Browser features (Clipboard, Notification, file input, shortcuts) | native | `static/web/web.js`, `static/app.js` | no menu bar in the browser (recorded) |
| Theme cookie, no flash | native | | |
| Desktop: same server in the same process on a random 127.0.0.1 port, glaze window | native | `cmd/desktop/main.go` listens on `127.0.0.1:0` only; `glaze.New(false)` | Missing in glaze: window resize events (fallback: page reports its size), downloads (fallback: server writes the file). |
| `web` on port 8090 | native with a caveat | `make web` | Port 8090 on this Mac is held by a foreign process on 127.0.0.1; the server then binds `[::1]:8090` and `http://localhost:8090` works |
| `package` builds the .app | fallback | `desktop/Info.plist` plus the binary, assembled by the Makefile | glaze has no packaging tool |
| Tests with httptest, optional browser | native | `server_test.go`, `tests/browser/t9.test.mjs` (Chrome through the DevTools protocol, Node built-ins only) | |

## cgoprobe

| Target | Result | Notes |
|---|---|---|
| web server | PASS | `go build -tags cgoprobe ./cmd/web` links the C checksum (cgo on by default); the export test also passes with the tag |
| desktop shell (shipped configuration, `CGO_ENABLED=0`) | **FAIL (expected)** | `make build` uses `CGO_ENABLED=0`, so the probe build stops at `core/checksum_nocgo.go:8:9: undefined: cgoprobe_requires_cgo_but_cgo_is_unavailable_for_this_target`. The desktop shell has no cgo of its own, so the failure is the core probe file declining to build without cgo, not a dependency problem. |
| desktop shell with cgo switched on (`desktop-cgo` line) | PASS | `CGO_ENABLED=1 go build -tags cgoprobe ./cmd/desktop` links the C checksum; glaze and native do not conflict with cgo |

The desktop shell itself builds and runs with `CGO_ENABLED=0` (see the next section). Evidence: `make cgoprobe` output in `friction.md`.

## Cross builds

All four are `CGO_ENABLED=0 go build` from the macOS arm64 host, compile only (`-o /dev/null`) in this pass; `make cross-windows` and `make cross-linux` run the same command and write binaries into `bin/`. No Docker, mingw or GTK headers.

| Target | Route | Extra tools | Result | Notes |
|---|---|---|---|---|
| linux/amd64 web server | `CGO_ENABLED=0 GOOS=linux go build ./cmd/web` | none | PASS (compile) | |
| linux/amd64 desktop | `CGO_ENABLED=0 GOOS=linux go build ./cmd/desktop` | none | PASS (compile) | glaze dlopens WebKitGTK at run time (4.0, 4.1 or 6.0 with GTK3 or GTK4); the target needs those libraries. Not run on Linux. |
| windows/amd64 web server | `CGO_ENABLED=0 GOOS=windows go build ./cmd/web` | none | PASS (compile) | |
| windows/amd64 desktop | `CGO_ENABLED=0 GOOS=windows go build -ldflags=-H=windowsgui ./cmd/desktop` | none | PASS (compile) | needs the WebView2 runtime at run time. Not run on Windows. |

linux/arm64 and windows/arm64 desktop also compile.

## Platform limits (from glaze and native documentation and source)

- **Windows menu shortcuts are not wired.** `glaze/menu` on Windows builds a Win32 menu bar whose items are clickable, but `Shortcut` is ignored (`menu_windows.go` header, glaze README "shortcuts not wired"). This shell relies on the page keydown handler (`static/app.js`) for the same shortcuts there.
- **Linux has no menu bar.** `glaze/menu` on Linux installs only the shortcuts (and skips `Selector` items). To avoid doubled handlers this shell does not call it on Linux; the in-page menubar is shown instead (`installNativeMenu` returns false).
- **Notifications exist on macOS only.** `native/notify` returns `ErrUnsupported` elsewhere; the shell ignores that and the page toast covers it.
- **Reveal on Linux** opens the containing folder and cannot select the file (`openurl.Reveal` doc).
- **Menu items cannot show a checked state or change at run time** (`menu.Item` has no such fields), so the shell rebuilds the whole menu when view, theme or export state changes and prints a check mark in the title.
- The file dialogs must not be called on the UI thread (they block); the HTTP handlers that call them run on their own goroutines.

## Tests

| ID | Status | How |
|---|---|---|
| T1 starts and loads | native | httptest on `GET /`, first rows 1, 2, 3 with SKUs |
| T2 sort | native | `POST /query sort=name`, first name equals the core's first row |
| T3 search | native | count text in the response equals the core's Total |
| T4 selection | native | `POST /select`, every field checked |
| T5 validation | native | message shown, Update call counter stays 0 |
| T6 save | native | the refreshed row and the next page show the new name |
| T7 theme and prefs | native | cookie renders `class="dark"`, `prefs.json` contains `"theme":"dark"` |
| T8 export cancel | native | file export (desktop mode with a fake native layer) and web stream; partial file removed, state `cancelled`, SSE snapshot no longer running so the button is enabled; a second export can start |
| T9 dirty guard | native | real Chrome via the DevTools protocol (`tests/browser/t9.test.mjs`); skipped and counted as not feasible only when Node or Chrome is missing |
| T10 custom cells | native | HTML attributes and the generated `palette.css` compared with `core.StockFraction`, `core.Stars` and `core.Colors` for both themes | the pixel colors are not rendered; CSS correctness is by inspection |
| T11 cards | native | | |
| T12 host info | native | web (server-process) and desktop mode (native) | |

## Bench

Web target, headless Chrome (`tests/browser/bench-web.bench.mjs`, same `bench.js`): ready 175 ms, jump 209 ms, scroll total 3293 ms (151 placeholder steps, max gap 77 ms, p95 63 ms), cards switch 970 ms, cards scroll total 3203 ms (35 placeholder steps, max gap 56 ms, p95 32 ms), sort 335 ms, search 172 ms (49,898 rows), export 513 ms (49,898 rows, max gap 3 ms).

Desktop shell (glaze on WKWebView): not measured in this pass (benchmarks run elsewhere). The figures below are the webview_go GoTTH shell, kept for reference only. Typical run: ready 553 ms, jump 222 ms, scroll total 6060 ms (185 placeholder steps), cards switch 2154 ms, cards scroll total 3713 ms (163 placeholder steps), sort 1083 ms, search 721 ms (49,898 rows), export 537 ms (49,898 rows, max gap 190 ms).
- The max and p95 gaps of the scroll steps (2 to 4 s) and cards steps (1 to 2 s) are not reliable: the login session was locked during all runs, so the webview was occluded and WebKit throttled timers, leaving only a handful of probe ticks. Rerun on an unlocked screen.
- Reliability: of 7 recent desktop runs, 6 produced all eight events. One run stalled after `cards_scroll` (the page stopped responding during the cards to table swap); the Go side then prints an `error` metric ("bench stalled") and `done` and quits, so `make bench` never hangs. Earlier stalls had real causes that were fixed (see friction.md); this one is not understood. It may be related to the locked screen.
- The windowed design means a 25-row scroll step (about 1,000 px) usually lands on a page that is not loaded yet, which is why most scroll steps show placeholders.

## Target notes

The server, views, forms, export and tests are identical to `apps/gotth`. What changed is only the desktop shell. `cmd/desktop` now holds `main.go` (glaze window), `menu.go` (`glaze/menu` menu bar and state sync), `native.go` (glaze file dialogs plus `native` notify, clipboard and openurl behind the existing `gotth.Native` interface) and `appnap_darwin.go` (App Nap guard through purego `objc`, no C). The `-tags native` cgo/Objective-C shim, the osascript and pbcopy helpers and `appnap_desktop.m` are removed. One small change outside `cmd/desktop`: `handlers_desktop.go` gained `Server.reveal`, which uses an optional `Reveal` method on the shell's `Native` and otherwise the core helper. The web target is unchanged. No hand-written C or Objective-C remains in the app.

- 2026-11-09: user observed Finder windows opening during the agent's reveal probe, confirming openurl.Reveal works on macOS (visual confirmation by the user, not by the agent).
