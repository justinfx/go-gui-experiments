# GoTTH friction log

Start of work: 14:24 (2026-11-08).

### 14:24 Install templ and shadcn-templ
- Step: Installing the Go-installed CLI tools from section 6.5
- Command/API: `go install github.com/a-h/templ/cmd/templ@v0.3.1100`, `go install github.com/axadrn/shadcn-templ/v2/cmd/shadcn-templ@v2.0.0`
- Outcome: Both installed in about 25 s. `shadcn-templ --help` lists init/add/bundle/apply/preset.
- Time lost: 0
- Category: toolchain
- Resolution: none needed

### 14:25 Bootstrap with the shadcn-templ CLI
- Step: Project bootstrap using the framework's own template
- Command/API: `cd apps && shadcn-templ init gotth --template templ --base-color neutral --silent`
- Outcome: Scaffolded into the existing empty `apps/gotth`. The template names the module `gotth`, uses Go 1.24, defaults to style `base-nova`, loads htmx 4.0.0-beta6 from a CDN, and expects the `tailwindcss` binary and `task` on PATH. The registry (https://shadcn-templ.com) is fetched over the network at init and add time.
- Time lost: 3
- Category: docs-gap
- Resolution: Renamed the module to `gguix/apps/gotth` with sed over imports and components.json aliases; replaced CDN htmx with vendored 2.0.11; Tailwind standalone CLI is downloaded by `make deps`; the Taskfile is replaced by the Makefile.

### 14:27 Component names differ from the brief
- Step: `shadcn-templ add radiogroup alertdialog nativeselect togglegroup ...`
- Command/API: `shadcn-templ add <names>`
- Outcome: Registry item names are kebab-case (`radio-group`, `alert-dialog`, `native-select`, `toggle-group`); the camel names fail with "was not found". There is no `sonner` item (toast is `toast`) and no `datepicker` item (calendar is `calendar`). One failing name aborts the whole multi-name add.
- Time lost: 2
- Category: docs-gap
- Resolution: Added components one at a time using kebab-case names.

### 14:40 Port 8090 already in use on this machine
- Step: First run of the web target
- Command/API: `bin/gguix-gotth-web` listening on localhost:8090
- Outcome: Another program (process named `proxy`, pid 4054, not ours) holds 127.0.0.1:8090. Bind failed.
- Time lost: 3
- Category: toolchain
- Resolution: cmd/web tries 127.0.0.1, then [::1], then the next ports; it now listens on [::1]:8090, which http://localhost:8090 resolves to. The other process was not touched.

### 14:30 htmx revealed trigger does not see nested scroll containers
- Step: Designing windowed scroll sentinels (brief says hx-trigger="revealed")
- Command/API: read htmx 2.0.11 source: `revealed` listens to window scroll only and compares against window.innerHeight
- Outcome: inside a fixed-height scroll container it will not fire on container scroll. Not yet verified in a browser.
- Time lost: 5
- Category: framework-limitation
- Resolution: using `hx-trigger="intersect once root:#scroller"` instead (implemented, untested)

### 14:35 shadcn-templ Table wrapper breaks sticky header
- Step: Table markup
- Outcome: table.Table wraps in an overflow-x-auto div, so a sticky thead cannot stick to the app scroller. Own <table> element used with the library Header/Row/Head/Cell parts.
- Time lost: 2
- Category: api-confusion
- Resolution: workaround in views/list.templ

### 14:45 templ treats `{ expr }` inside a script element as literal text
- Step: Putting the theme script and the bench boot values in the page head
- Command/API: `<script>{ templ.Raw(themeScript) }</script>`
- Outcome: Output contained the literal text `{ templ.Raw(themeScript) }`. The theme script and `window.GGUIX_BENCH` were silently never defined; bench mode only worked because the bench file itself was included. Found much later while debugging the bench.
- Time lost: 40
- Category: docs-gap
- Resolution: `@templ.Raw("<script>" + js + "</script>")`

### 14:50 htmx out-of-band attributes and table rows
- Step: Updating the saved row and the toolbar out of band
- Command/API: `hx-swap-oob?={ oob }` (boolean attribute) and an OOB `<tr>`
- Outcome: A bare `hx-swap-oob` attribute is not treated as `true`, so the span and div elements stayed in the main swap and showed up as stray text in the status bar. An OOB `<tr>` outside a `<template>` is dropped by the HTML parser and leaves only its text.
- Time lost: 15
- Category: api-confusion
- Resolution: explicit `hx-swap-oob="true"`; table rows wrapped in `<template>` (views/list.templ RefreshedItem)

### 15:00 htmx `revealed` and `intersect root:` do not work for the windowed scroll
- Step: Bottom and top sentinels in the scroll container (brief: `hx-trigger="revealed"`)
- Command/API: attempt 1 `revealed` (source reading: it listens only to window scroll and compares with window.innerHeight, so it cannot see a nested scroll container); attempt 2 `intersect once root:#scroller` (worked for the first render, but did not fire after a region swap that replaced the scroller); attempt 3 an app-owned IntersectionObserver sending a custom event (worked, but observers only deliver while the page is being rendered, which stalls when the window is occluded or the screen is locked)
- Outcome: Final solution is a visibility check with getBoundingClientRect after every scroll, every swap and on a 100 ms timer, which sends a custom `gguix-visible` event that the sentinels use as `hx-trigger`.
- Time lost: 90
- Category: framework-limitation
- Resolution: static/app.js `loadVisiblePlaceholders`

### 17:00 Desktop webview: bench stalls and huge per-swap cost in WebKit
- Step: First `make bench` on the desktop shell
- Command/API: `GGUIX_BENCH=1 bin/gguix-gotth`
- Outcome: Bench stopped after the jump step, later after other steps, and was flaky. Causes found one by one with `sample`, request logging (`GOTTH_LOG=1`) and temporary probes: (1) the shadcn-templ `lifecycle.js` re-runs `document.querySelectorAll` for every component selector on every DOM mutation, so cost grows with the whole document; with 1,000 rows each swap took seconds and finally the swap never completed; (2) htmx 2.0.11 scans each swapped fragment with an XPath query for `hx-on:*` attributes, very slow in WebKit; (3) a sticky `<thead>` makes WebKit lay out the 1,000 row table on every scroll step (about 180 ms each); (4) `border-collapse: collapse` on the table is slow in WebKit; (5) `htmx.ajax` promises did not always settle in WKWebView.
- Time lost: 240
- Category: runtime-bug
- Resolution: (1) patched `components/baseui/lifecycle.js` to inspect only added nodes and re-bundled with `shadcn-templ bundle`; (2) page head script that replaces the XPath expression so it matches nothing (this app uses no hx-on); (3) table header moved into its own table above the scroller with horizontal scroll sync; (4) `border-separate` with cell borders; (5) helpers wait for a `gguix:region` event instead of the promise.

### 17:10 Screen is locked, the pane is shared
- Step: Looking at the desktop window
- Command/API: `screencapture -x`
- Outcome: Screenshots are black because the login session is locked (`CGSSessionScreenIsLocked`). The webview is then occluded and WebKit throttles timers, so the bench tick-probe gaps (max_gap_ms, p95_gap_ms) are inflated and few ticks are recorded. The built-in browser pane is shared with another agent session (a Fyne app on :8091 appeared in my tab list); I used my own tab id.
- Time lost: 20
- Category: toolchain
- Resolution: recorded the limitation in capabilities.md; added `NSProcessInfo beginActivity` (cmd/desktop/appnap_desktop.m) so macOS App Nap cannot suspend the app.

### 17:50 Cross-compiling the webview shell
- Step: `make cross-linux`
- Command/API: Docker `golang:1.27.3` image, `apt-get install libwebkit2gtk-4.0-dev`
- Outcome: Package not found; the default image is based on a newer Debian that only has webkit2gtk-4.1 while webview_go v0.0.0-20240831 links `webkit2gtk-4.0`.
- Time lost: 5
- Category: toolchain
- Resolution: switched to the `golang:1.27.3-bookworm` image

### 18:05 Windows cross build of the webview shell
- Step: `make cross-windows`
- Command/API: Docker `golang:1.27.3-bookworm`, mingw-w64, `CC=x86_64-w64-mingw32-gcc`
- Outcome: attempt 1 failed in webview_go's bundled `WebView2.h`: `EventToken.h` not found (mingw ships `eventtoken.h`, Linux paths are case sensitive). Attempt 2 added a symlink for the capitalised name and produced a 20 MB `.exe`.
- Time lost: 12
- Category: toolchain
- Resolution: `ln -sf .../eventtoken.h .../EventToken.h` in `desktop/cross.sh`

### 18:20 Desktop bench still intermittently stalls
- Step: Repeating `make bench`
- Command/API: `GGUIX_BENCH=1 bin/gguix-gotth`
- Outcome: about one run in seven stops responding during the cards to table switch (process idle, no JS timers running). Not understood after sampling the WebContent process. The screen is locked on this machine, which may be a factor.
- Time lost: 30
- Category: runtime-bug
- Resolution: unresolved. Bench steps are wrapped so a failing step reports `error` and continues; a Go watchdog prints `error` and `done` and quits after 45 s without a metric.

## Summary

- Wall time to first window: web page rendering in the browser pane at about 14:40 (16 minutes); desktop webview window at about 17:05 (the day included a pause for usage limits).
- Wall time to feature complete: about 18:30 (about 4 hours of work), with gaps listed in capabilities.md (split divider, column resize, native menu bar).
- Entries per category: toolchain 5, docs-gap 3, api-confusion 2, framework-limitation 2, runtime-bug 2, build-error 0, core-issue 0. (Counted from the entries above; the 14:30 and 15:00 htmx entries are both framework-limitation.)
- Three biggest pain points:
  1. WebKit performance of server-rendered 1,000-row windows: the shadcn-templ lifecycle observer, htmx's XPath scan, sticky table headers and collapsed borders each made the desktop shell unusable until fixed (about 4 hours including finding them).
  2. htmx `revealed` and `intersect` triggers do not work reliably for a nested scroll container, so windowed scrolling needed an app-owned visibility check.
  3. templ silently turning `{ expr }` inside `<script>` into literal text, which hid a dead bench flag and theme script for hours.

### 2026-11-09 Fix pass: split view, column resize, native shim
- Step: closing gaps that rested on wrong assumptions (shadcn `resizable` exists; `webview.Window()` gives the NSWindow).
- Command/API: `shadcn-templ add resizable`; cgo/Objective-C shim (`NSMenu`, `NSOpenPanel`, `NSSavePanel`, `UNUserNotificationCenter`, `NSPasteboard`) behind `-tags native`.
- Outcome: `resizable` worked first time and its drag was verified; its panel wrapper needed `overflow:hidden` on the list panel so the list scroller kept its own scrollbars. Column resize needed only `<col>` width changes because the tables were already `table-layout: fixed`. The shim compiles and both binaries launch. `UNUserNotificationCenter` needs a bundle identifier, so a bare binary uses deprecated `NSUserNotificationCenter`. Runtime behaviour of menu, dialogs and notification could not be verified: osascript has no assistive access and the available screen capture shows the whole display, not the app.
- Time lost: 5 (a Bash safety check refused two harmless commands containing `$$`/heredocs; rewritten as script files)
- Category: toolchain
- Resolution: shim kept behind the `native` tag as instructed; default build unchanged apart from the split and column resize.
