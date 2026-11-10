# GoTTH capabilities

Status values: native, framework-drawn, fallback, gap, not feasible.
"Native" here means the browser or the webview provides it as a real control, not that the OS draws it.
In this stack every visible control is HTML. Files named are relative to `apps/gotth/`.

## Section 5 requirements

| Requirement | Status | How it was done | Notes |
|---|---|---|---|
| Window title `GGUIX - GoTTH` | native | `w.SetTitle` in `cmd/desktop/main.go`; `<title>` in `views/page.templ` | |
| Window 1280 x 800, min 1024 x 640, restore saved size | native (desktop) | `SetSize(w, h, HintNone)` from prefs, then `HintMin` 1024 x 640; `static/desktop/desktop.js` posts the window size to `/prefs/window` 500 ms after a resize | webview_go has no resize callback, so the page reports its own size |
| Toolbar: search with 250 ms debounce | native | shadcn `input`; `static/app.js` debounces (0 ms in bench mode) and fires `filters-changed` on the form | placeholder `Search name or SKU`, 280 px |
| Categories popup with 8 checkboxes, `Clear`, `Categories (n)` | framework-drawn | shadcn `popover` with `checkbox` items (`form="filters"` keeps them in the filters form after the popover moves to the body); label swapped out of band by the server | |
| Status dropdown | native | shadcn `native-select` | the shadcn `select` popup was not used; the native select is far simpler to drive and test |
| Go-to-row field and `Go` button | native | `G.gotoRow` in `static/app.js` clamps with the known total, scrolls locally and posts `/goto` (one-row lookup, answered from the page being fetched) to fill the form | Enter submits; asks the dirty guard first |
| View switcher Table / Cards | framework-drawn | two shadcn `button`s as a segmented control (`aria-pressed`) | |
| Export CSV button, theme dropdown | native | shadcn `button`, `native-select` | |
| Split view with draggable divider | gap | fixed 65/35 flex split, 1 px divider | shadcn-templ has a `resizable` item but I did not add it; no time. Recorded as a gap. |
| Status bar (count, rows a-b, last fetch ms, export progress, identity, Host button) | framework-drawn | `views/page.templ` `StatusBar`; count and fetch time swapped out of band; rows range computed by `static/app.js` from the pages in the DOM; export area pushed over SSE | |
| Table: columns, widths, alignment | framework-drawn | shadcn `table` Header/Row/Head/Cell parts inside my own `<table>` with `table-layout: fixed` | The `table.Table` wrapper has an `overflow-x-auto` container that breaks scrolling, so I do not use it |
| Fixed header | fallback | the header is a separate table above the scroller, horizontal scroll synced by JS | A sticky `<thead>` made WebKit re-lay out 1,000 rows on every scroll step |
| Column resize | gap | not implemented | no resize facility in the table parts |
| Sort by header click, active header shows ▲ / ▼ | native | header cells post `/query` with the sort field; server toggles direction | |
| Reset scroll, keep selection only if in first page | native | `handleQuery` | |
| Windowed infinite scroll (200 per page, at most 6 pages in DOM) | fallback | second pass (see adapter.md): a virtual scroller. `#vs` is as tall as the whole result; server-rendered 200-row page blocks are absolutely placed by `static/app.js` `sync()`; page fetches are htmx requests from placeholder blocks; `adapter_data.go` caches pages on the server. See friction.md for why `revealed` was replaced | The scrollbar now represents the whole result (scaled when 1,000,000 rows exceed the browser height limit), no longer only the loaded window. |
| `…` placeholder rows | native | a placeholder block (`.ph`) clones the `#ph-tpl` row into every visible position; it is also the element that fetches the page | |
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
| Copy Row | native / fallback | Clipboard API in the browser (`static/web/web.js`); in the desktop shell `pbcopy` through `/native/copy` (`handlers_desktop.go`) plus the Clipboard API | transient `Row copied` text in the status bar for 2 s |
| Import Notes | native (web), fallback (desktop) | web: `<input type="file" accept=".txt">` and `File.text()`; desktop: `osascript choose file of type {"txt"}` then the server reads the file | 500 character limit, 64 KB limit with the error dialog `File is too large.` |
| Export in the background, progress, cancel, notification | native (web) / fallback (desktop) | `GET /export` streams the CSV as a download, `GET /export/progress` is SSE (htmx sse extension) swapped into the status bar, `POST /export/cancel`; desktop: `osascript` save dialog, server writes the file, deletes the partial file on cancel | web cancel aborts the HTTP connection so the browser drops the download. Whole UI stays usable. |
| OS notification `Export complete` | native (web), fallback (desktop) | web: Notification API (permission asked on the first export) with the shadcn toast as in-app fallback; desktop: `osascript display notification` sent by Go when the job finishes | WKWebView has no Notification API, so the desktop fallback is osascript |
| Menu bar, shortcuts | fallback | web: no menu bar (browser has none); keyboard shortcuts in `static/app.js` (Cmd or Ctrl: E, O, Shift+C, 1, 2, R, F, I). Desktop: webview_go has no menu API, so `views/desktop/menubar.templ` draws the menus in the page with shadcn `menubar` (App, File, Edit, View with Theme submenu, Help), actions wired by `data-menu-action` | The Edit menu uses `document.execCommand`. Not a native macOS menu bar. |
| About dialog | framework-drawn | shadcn `dialog`: title, framework versions, `runtime.Version()`, `BuildMarker 0`, `UIMarker 0` (`views/marker.templ`) | |
| Theme System / Light / Dark, no restart | native | `dark` class on `<html>`, cookie `gguix-theme`, inline head script for `prefers-color-scheme` | dropdown and menu stay in sync (`G.setTheme`) |
| Live switching of System with the OS | native | `matchMedia` change listener in the head script | |
| Prefs saved 500 ms after last change and on quit | native | `Server.update` debounce; `Flush` on SIGINT (web) and when the window closes (desktop) | |
| Prefs restored before the first query, `FindRow(..., 2000)` for the selection | native | `handleIndex` on the server | skipped in bench mode |
| Threading mechanism | native | no UI thread to manage: every UI change is an htmx swap in the page. The desktop shell calls `w.Dispatch` only to terminate the window. | |
| Native prefs facility (unused) | n/a | the browser equivalent is `localStorage`/cookies; the desktop shell has none. The core `PrefsStore` is used, stored server side. | |
| Host panel | native (desktop), server-process (web) | `GET /host` renders `host.Current` (desktop) or `host.Server` (web) as the two-column list, Source row always shown, Notes below; status bar shows `<Username>@<Hostname>` | |
| Reveal Config Folder, Reveal Last Export | native (desktop), disabled (web) | desktop: `host.RevealInFileManager` (the webview has no facility of its own); web: disabled with `Not available in the browser` | Reveal Last Export is disabled until an export finishes |

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
| Desktop: same server in the same process on a random 127.0.0.1 port, webview_go window | native | `cmd/desktop/main.go` listens on `127.0.0.1:0` only | Missing in this shell: native menus (fallback: in-page menubar), native file dialogs (fallback: osascript), notifications (fallback: osascript), clipboard (fallback: pbcopy), window resize events (fallback: page reports its size), downloads (fallback: server writes the file) |
| `web` on port 8090 | native with a caveat | `make web` | Port 8090 on this Mac is held by a foreign process on 127.0.0.1; the server then binds `[::1]:8090` and `http://localhost:8090` works |
| `package` builds the .app | fallback | `desktop/Info.plist` plus the binary, assembled by the Makefile | webview_go has no packaging tool |
| Tests with httptest, optional browser | native | `server_test.go`, `tests/browser/t9.test.mjs` (Chrome through the DevTools protocol, Node built-ins only) | |

## cgoprobe

| Target | Result | Notes |
|---|---|---|
| web server | PASS | `go build -tags cgoprobe ./cmd/web` links the C checksum; the export test also passes with the tag |
| desktop shell | PASS | already uses cgo (webview and the App Nap helper) |

## Cross builds

| Target | Route | Extra tools | Result | Size | Notes |
|---|---|---|---|---|---|
| linux/amd64 web server | `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/web` | none | PASS | 14 MB | |
| linux/amd64 desktop | Docker `golang:1.27.3-bookworm` (amd64 under emulation), `libgtk-3-dev`, `libwebkit2gtk-4.0-dev`, `go build ./cmd/desktop` | Docker | PASS | 14 MB | the default `golang:1.27.3` image failed: Debian trixie has no webkit2gtk-4.0 |
| windows/amd64 web server | plain `go build` | none | PASS | 14 MB | |
| windows/amd64 desktop | Docker, mingw-w64 (`CC=x86_64-w64-mingw32-gcc`), `-H=windowsgui` | Docker, mingw-w64 | PASS | 20 MB | attempt 1 failed in webview_go's bundled WebView2 header (`EventToken.h` is spelled `eventtoken.h` in mingw); attempt 2 added a case-insensitive symlink for the header and passed |

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

Second pass (data adapter, see adapter.md). Indicative only: other agents shared the machine.

Web target, headless Chrome (`tests/browser/bench-web.bench.mjs`, same `bench.js`), typical run: ready 144 ms, jump 145 ms, scroll total 3200 ms (0 placeholder steps, max gap 78 ms, p95 25 ms), cards switch 346 ms, cards scroll total 3201 ms (0 placeholder steps, max gap 18 ms, p95 9 ms), sort 375 ms, search 171 ms (49,898 rows), export 510 ms (49,898 rows, max gap 1 ms).

Desktop shell (`make bench`, webview_go on WKWebView), two runs: ready 678 ms, jump 220 ms, scroll total 3340 ms (1 and 3 placeholder steps, one stall of 300 to 390 ms), cards switch 460 ms, cards scroll total 3202 ms (0 placeholder steps, p95 gap 65 ms), sort 400 ms, search 190 ms, export 506 ms.

First pass for comparison (headless Chrome): scroll 146 placeholder steps, cards switch 947 ms, cards scroll 27 placeholder steps; (desktop) scroll 185 placeholder steps, cards switch 2154 ms, cards scroll 163 placeholder steps, sort 1083 ms.

## Target notes

The server, views, forms, export, the data adapter (`adapter_data.go`) and tests are shared by both targets. The web target adds `cmd/web` and `static/web/web.js` (download by anchor, Clipboard API, Notification API, file input) and the web branch of the page head in `views/page.templ` (`gguix:target web`). The desktop target adds `cmd/desktop` (webview window, osascript dialogs and notifications, pbcopy, App Nap guard), `handlers_desktop.go` with the routes registered only when a `Native` is supplied (`gguix:target desktop` block in `server.go`), `views/desktop/menubar.templ`, `static/desktop/desktop.js` and `desktop/` (Info.plist, cross-build script). Everything differs for the same reason: the webview has no menu, dialog, notification, clipboard or download support, so Go does those jobs and the page asks the local server. Bench code is `static/bench/bench.js` and `metric_bench.go`; tests are `server_test.go`, `exec_test.go` and `tests/browser`.
