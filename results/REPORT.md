# GGUIX results report

GGUIX compares five ways to build a GUI app in Go. Each one implements the same app from one spec. The app has a 1M-row table, a 1M-item card list, a form, themes, a background job and native desktop features. All five sit on the same shared Go core.

This report is organized around three questions a reader is likely to start from:

1. **Do I have a lot of data?** How each stack feels with a million rows.
2. **Do I need cgo?** Whether a cgo core can run, and what native libraries the shipped binary needs at run time.
3. **Do I need the web, native, or both?** What each stack gives on each side, and what reaching both costs.

It catalogues trade-offs. It does not pick a winner.

All numbers are from the clean re-run of 2026-11-09 (`results/DEVIATIONS.md`, "Clean re-run"), unless a line says otherwise.

## Summary

**Lots of data**
- On the macOS desktop, Fyne and Cogent stay instant or responsive with no visible stutter.
- Wails measures frame gaps of 91 ms, and 140 to 201 ms in its data-layer variants, during a very fast scroll. These are real, measurable gaps that some people could notice. In a hands-on check by the author they did not stand out: an occasional flicker on an aggressive flick, and the controls never lagged.
- GoTTH has no stutter, but switching to the card view takes about 1 s and blank rows show while scrolling.
- The MIQT base app stutters visibly. That is our data layer, not Qt: the same toolkit with a different data layer scrolls smoothly (14 to 16 ms).
- Memory is not felt on a normal machine. Fyne peaks at about 1.4 GB, the others at 0.5 to 0.8 GB.

**cgo**
- A cgo core runs on the desktop in every stack. On the web it runs only where Go runs on a server: Wails and GoTTH, or Fyne and Cogent split into a WASM UI plus a Go server.
- Every desktop binary links some native libraries. `gotth-glaze` (built with `CGO_ENABLED=0`) links the fewest: on Linux only libc, libdl and libpthread, loading WebKitGTK at run time. It still depends on glibc, like every Linux build here. MIQT links Qt and libstdc++ (libc++ on macOS), which must be present or shipped.

**Web, native or both**
- Reaching both costs about the same code everywhere: 10 to 20% of lines are target-specific.
- The web builds differ in what the user waits for. Wails and GoTTH load 0.3 MB or less and are ready in about 0.2 s. Cogent loads 6.9 MB and takes 1.5 s to be ready; Fyne loads 12.4 MB and takes 0.8 s, then stutters while scrolling.
- Native feel costs something. MIQT gives real Qt widgets but ships 111 MB, takes about 2 minutes to build cold, and has no browser target. Wails gives native menus and dialogs with Node in the toolchain. Fyne and Cogent draw their own controls and dialogs.

**Test on the OS you ship to.** The Linux runs found three bugs in our apps that macOS hid. All three are fixed.

## What we built and how we measured

| id | Stack | Web route |
|---|---|---|
| fyne | Fyne v2.8.2 | Go WASM, or WASM UI plus a Go server ("web-remote") |
| cogent | Cogent Core v0.5.27 | Go WASM, or web-remote |
| wails | Wails v3.1.0, Svelte 5, Tailwind v4, shadcn-svelte | Wails server mode |
| gotth | templ, htmx, Tailwind, shadcn-templ; webview_go desktop shell | Go web server (primary target) |
| miqt | MIQT v0.15.0 on Qt 6.12.0 | none (no browser target) |

Variants answer follow-up questions:
- `<id>-fit` and `<id>-fitcap`: each app with a data layer suited to its toolkit. `-fit` allows 4 to 6 requests in flight; `-fitcap` keeps the shared core's limit of 2.
- `gotth-glaze`: the GoTTH app in a cgo-free desktop shell.
- `miqt-gotth`: the GoTTH web UI inside Qt WebEngine, via MIQT.

Method:
- **Spec:** `SPEC.md` v1.2. Per-app briefs are in `briefs/`.
- **Shared core:** `core/`, a Go module with no dependencies. It serves 1M rows with 50 ms of simulated latency, in pages of 200 rows, with 20 pages cached and 2 requests in flight.
- **Builders:** each app was written by an isolated AI agent (Claude Sonnet) that saw only its own brief. Notes for each app are in `results/<id>/`.
- **Harness:** `bench/measure.sh`, on one Apple M1 Max, macOS arm64 (`results/ENVIRONMENT.md`). Output goes to `results/<id>/metrics.json`. The harness was audited and fixed before the clean re-run (`results/HARNESS_CHANGES.md`): release builds, the binary launched directly, gzip on every web server, one definition of web "ready", absolute scroll pacing, and WebView and GPU memory counted.
- **Desktop numbers:** median of 5 runs. Ranges (min to max) are given where a result sits near a band edge.
- **Web numbers:** one run per app in Chrome 156 via Playwright, at device scale factor 2. Treat only large gaps as real.
- **Judging:** scores are from `results/judge2/scores.md`, re-checked against the clean re-run. Style was judged blind (`results/judge2/shots_blind/`).
- **Linux:** a function check in an Ubuntu 24.04 arm64 container (`results/LINUX.md`). No Linux timings.

MIQT has no browser target. It is the control group: it shows what dropping the web requirement buys.

### Perception bands

Numbers are read in bands of what a user notices. Apps in the same band count as tied. In the tables, **bold** marks the best value in each row (lowest time, gap, size or memory; highest score). Bold alone is not a ranking: a value in the same band as the bold one feels the same to a user.

| Band | Meaning |
|---|---|
| about 100 ms or less | instant |
| up to about 1 s | responsive |
| over 1 s | a noticeable wait |
| frame gaps over about 100 ms | visible stutter |

### Terms used in the tables

- **Frame gap:** how late a frame was against a steady 16 ms (60 fps) schedule. The longest gap in a scroll is the worst hitch; p95 is the typical bad frame.
- **Blank rows (blank-row steps):** the app fetches rows in pages from the core, which answers after 50 ms of simulated latency. While a page is in flight the table shows empty placeholder rows instead of data. The scroll test takes 200 steps; a blank-row step is one where any visible row was still a placeholder. 0 means data was always on screen; 200 would mean it never was.
- **The scroll test:** 200 steps of 25 rows each, one every 16 ms: about 5,000 rows in 3.2 s, or about 1,500 rows per second. That is a fast fling or scrollbar drag, not gentle wheel scrolling (see "How realistic is the scroll test?").
- **Custom cells:** three table cells that are drawn rather than plain text: a stock-level bar, a status badge and a five-star rating. The spec requires them in every app.

## 1. Do I have a lot of data?

### Desktop, base apps (macOS, median of 5)

| | fyne | cogent | wails | gotth | miqt |
|---|---|---|---|---|---|
| Ready (ms) | 572 | 555 | 623 | 632 | **443** |
| Jump to row 500k (ms) | 146 | 113 | **90** | 232 | 666 |
| Table scroll, max frame gap (ms) | 40 | **19** | 91 (68 to 108) | 92 (86 to 102) | 251 |
| Table scroll, p95 frame gap (ms) | 30 | **3.3** | 54 | 77 | 229 |
| Blank-row steps while scrolling | **0** | 24 | 64 | 155 | 115 |
| Cards scroll, max frame gap (ms) | 25 | **2.5** | 8 | 73 | 428 |
| Switch to cards (ms) | **104** | 112 | 194 | 969 | 928 |
| Sort (ms) | **254** | 262 | 280 | 365 | 622 |
| Search (ms) | **66** | **66** | 100 | 183 | 70 |
| Idle memory (MB) | 213 | 294 | 307 | 266 | **196** |
| Peak memory (MB) | 1412 | **513** | 824 | 730 | 702 |

Memory is the whole app: the process tree plus, for Wails and GoTTH, the WebKit processes.

What a user would notice:
- **Start-up** is responsive in all five.
- **Jumping** far into the table is instant in Wails, close to instant in Cogent, responsive in Fyne and GoTTH, and slowest in MIQT (666 ms).
- **Scrolling:** Fyne and Cogent are smooth. Wails and GoTTH sit just under the stutter line; in one Wails run and one GoTTH run of five the longest gap passed 100 ms. MIQT stutters visibly in the table and the cards.
- **Blank rows** flash while scrolling fast in GoTTH and MIQT, and less in Wails and Cogent. Fyne shows none.
- **Switching to cards** is instant or close in Fyne and Cogent. GoTTH and MIQT take close to 1 s, at the edge of a noticeable wait.
- **Sort and search** are responsive or instant everywhere.
- **Memory** is not felt on a normal machine. Fyne's 1.4 GB peak would matter on a machine with little RAM.

The Wails scroll result changed with the harness fix. Before it, the scroll ran at relative pacing, so slow frames stretched the run instead of showing up as gaps, and Wails measured 22 ms (`results/HARNESS_CHANGES.md`, B4). The 91 ms figure holds across all five runs. An isolation test confirmed the cause: with only the old pacing put back, Wails measured 6 to 28 ms again, because the run stretched from 3.2 s to over 4 s (`results/DEVIATIONS.md`, "Wails scroll pacing isolation test").

### How realistic is the scroll test?

The test is a stress test: about 1,500 rows per second, like dragging the scrollbar or a hard trackpad fling through a million rows. Gentle wheel scrolling, a few rows per notch, was not measured.
- A worst gap under 100 ms (Fyne, Cogent, base Wails, GoTTH) is unlikely to be seen in normal use.
- A worst gap of 100 to 200 ms (wails-fitcap, wails-fit) is one or two dropped-frame hitches during a fast drag. Some people could notice them. In a hands-on check of wails-fitcap and base Wails (below) they showed as an occasional flicker on an aggressive flick and did not stand out to the author.
- Gaps of 250 ms or more that repeat (base MIQT: p95 229 ms; gotth-fit and gotth-fitcap: p95 over 300 ms) mean most frames are late. That is visible as a jerky scroll even at moderate speeds.
- Blank rows are a separate effect: they show only when scrolling outruns the 50 ms fetch, so slow scrolling hides them.

The 100 ms "visible stutter" line is a starting point, not a verdict. The checks below are what calibrate it.

**Hands-on check (2026-11-10).** The author used base Wails and wails-fitcap one at a time on the same machine, with a free-spinning mouse wheel and the scrollbar (`results/wails-fitcap/manual-check.md`, `results/wails/manual-check.md`):
- Gentle wheel scrolling: smooth in both.
- Wheel flicked as fast as it goes: still acceptable; an occasional flicker, not perceived as a problem.
- Fast scrollbar drag: blank rows while far pages load, judged normal because the data has to arrive.
- The controls never lagged; only the data caught up. Card view was very acceptable.
- Both felt the same, despite 91 ms against 140 ms worst gaps and 64 against about 1 blank-row step.

So the benchmark measures gaps that could be noticeable to someone, and that is worth knowing. To the author they did not stand out in use. Perception varies between people and input devices, so the report gives both: the measured gaps, and how they felt in one person's hands. Two more hands-on checks (2026-11-10) show the gap numbers alone do not predict feel. Base MIQT (repeated gaps of about 250 ms) lagged badly on a fast wheel spin and a scrollbar drag, and the scrollbar itself lagged, not only the data (`results/miqt/manual-check.md`). gotth-fitcap (repeated gaps of about 320 ms) showed only the most minor lag, and dragging was much more responsive than MIQT (`results/gotth-fitcap/manual-check.md`). The difference the harness misses is whether the controls keep up: in the WebView apps they did, in base MIQT they did not. gotth-fit was not checked by hand.

### Desktop, data-layer variants (median of 5)

Each variant replaces the shared pager with a data layer suited to its toolkit (`results/<id>-fit/adapter.md`).

| | fyne -fit / -fitcap | cogent -fit / -fitcap | wails -fit / -fitcap | gotth -fit / -fitcap | miqt -fit / -fitcap |
|---|---|---|---|---|---|
| Jump (ms) | 144 / 121 | 109 / 80 | 94 / 90 | 226 / 180 | 219 / 183 |
| Table scroll, max frame gap (ms) | 26 / 26 | 15 / 19 | 201 / 140 | 335 / 321 | 14 / 16 |
| Blank-row steps | 0 / 0 | 0 / 0 | 1 / 1 | 0 / 0 | 5 / 2 |
| Cards scroll, max frame gap (ms) | 18 / 14 | 2.6 / 2.8 | 8 / 8 | 93 / 107 | 1 / 5 |
| Switch to cards (ms) | 65 / 65 | 56 / 54 | 103 / 105 | 460 / 454 | 985 / 981 |
| Sort (ms) | 272 / 283 | 254 / 255 | 287 / 284 | 394 / 396 | 504 / 505 |

The variants pull in different directions:
- **MIQT:** scrolling goes from visible stutter to smooth. The card switch stays close to 1 s. So the base app's stutter came from our data layer, not from Qt.
- **Fyne and Cogent:** the same bands as the base apps. Cogent loses its blank rows, and the card switch gets faster.
- **Wails and GoTTH in the desktop WebView:** the blank rows go away, but the worst frame gaps grow (Wails 140 to 201 ms, GoTTH 321 to 335 ms). For Wails, a hands-on check found no perceived difference from the base app (see above). For gotth-fitcap, a hands-on check found only minor lag despite repeated gaps of about 320 ms (see above). It also found a bug: the table's scrollbar did not render, though dragging in its area worked.
- **The same variants in Chrome are smooth:** wails-fitcap 23 ms, gotth-fitcap 68 ms (single runs, Web section). The difference is the engine, not the HTML. A follow-up test ran the same Wails web builds in Chrome and in Playwright's WebKit, 3 runs each with a fresh server: Chrome's worst scroll gaps were 20 to 42 ms, WebKit's 141 to 304 ms (`results/DEVIATIONS.md`, "Chrome vs WebKit engine test"). The same test did not reproduce harness conditions for GoTTH, so GoTTH's engine split rests on the single runs above. The cause inside WebKit has not been found.

### Web (Chrome, single run)

| | fyne wasm | fyne remote | cogent wasm | cogent remote | wails | gotth |
|---|---|---|---|---|---|---|
| Initial transfer, gzip (KB) | 12,390 | 12,367 | 6,941 | 8,544 | 265 | **156** |
| Ready (ms) | 830 | 822 | 1,453 | 1,437 | **179** | 232 |
| Jump (ms) | 187 | 182 | 138 | 113 | **79** | 259 |
| Scroll, max frame gap (ms) | 126 | 117 | **15** | 22 | 21 | 152 |
| Cards, max frame gap (ms) | 52 | 27 | 7.9 | **4.1** | 6.2 | 110 |
| Switch to cards (ms) | **170** | 195 | 191 | 326 | 186 | 1,024 |
| Sort (ms) | 825 | 360 | 994 | 390 | **265** | 352 |
| Export, max frame gap (ms) | 166 | 12 | 4.7 | **0.2** | 1.6 | 25 |
| Peak DOM nodes | 102 | 102 | 1,437 | 1,609 | 118,277 | 143,344 |

"Ready" counts from navigation start, including the WASM download and compile for Fyne and Cogent (`HARNESS_CHANGES.md`, B3).

What a user would notice:
- **First load:** Wails and GoTTH are ready in about 0.2 s. Fyne takes 0.8 s and Cogent 1.4 s, a noticeable wait, even on a local link. Over a slow link the 7 to 12 MB downloads would add more.
- **Wails** is instant or responsive for every action, with no stutter.
- **Cogent** is smooth and responsive once loaded. A sort in pure WASM takes about 1 s; with the Go server split it is 390 ms.
- **GoTTH** stutters a little while scrolling, and the card switch is a wait of about 1 s, as on desktop.
- **Fyne** stutters while scrolling, in both routes. The Go server split speeds up sort and removes the export hitch, but not the scroll stutter.

Fyne's web build shows a missing-glyph box where the sort arrow should be in the ID header; the desktop build draws the arrow. The cause has not been checked (`results/fyne/web-notes.md`).

## 2. Do I need cgo?

Two separate questions sit under "cgo": can the app run a core that uses cgo, and what native libraries must be present when the shipped binary runs.

### Can a cgo core run?

The cgo probe builds each app against a core that calls C (`cgoprobe` in `metrics.json`).

| | fyne | cogent | wails | gotth | miqt | gotth-glaze |
|---|---|---|---|---|---|---|
| Desktop | PASS | PASS | PASS | PASS | PASS | FAIL as built, PASS with cgo on |
| Web | FAIL | FAIL | PASS | PASS | n/a | PASS |
| Web-remote | PASS | PASS | n/a | n/a | n/a | n/a |

- WASM cannot use cgo. Fyne and Cogent can still run a cgo core by moving it to a Go server (web-remote).
- In Wails and GoTTH the core runs in a normal Go process on both targets, so cgo works on both.
- gotth-glaze's whole point is `CGO_ENABLED=0`, so a cgo core cannot link into that build. Turning cgo on fixes the probe and gives up the property.

### What does the shipped binary link? (macOS, `otool -L`, 2026-11-10)

| App | Linked at run time, beyond libSystem |
|---|---|
| gotth-glaze | libresolv, CoreFoundation, Security |
| gotth | adds WebKit, Foundation, libobjc, libc++ |
| wails | WebKit, AppKit, Cocoa, Carbon, UserNotifications and other system frameworks |
| fyne | AppKit, Cocoa, OpenGL, CoreVideo, IOKit, UserNotifications |
| cogent | AppKit, Cocoa, OpenGL, Metal, QuartzCore, IOKit |
| miqt | QtWidgets, QtGui, QtCore, libc++ |
| miqt-gotth | adds Qt WebEngine, Qt Quick, Qt QML, Qt Network and five more Qt modules |

On macOS every system framework above ships with the OS, so only the Qt apps carry libraries of their own (bundled in the .app: 78 MB for MIQT, 293 MB for miqt-gotth).

On Linux the picture is what matters for "will this binary run on that machine". From the x86-64 cross builds (`objdump -p`, 2026-11-10):

| App | NEEDED libraries |
|---|---|
| gotth-glaze | libc, libdl, libpthread (WebKitGTK is loaded at run time with dlopen) |
| cogent | libX11, libm, libgcc_s, libc |
| fyne | libGL, libwayland-client, libm, libc |
| gotth (webview_go) | webkit2gtk-4.0, GTK 3, GLib, JavaScriptCore, libstdc++, libc |
| wails (default GTK4 build) | webkitgtk-6.0, GTK 4, libsoup 3, GLib, JavaScriptCore, libX11, libc |
| miqt | Qt6Widgets, Qt6Gui, Qt6Core, libstdc++, libc |

- **Every Linux desktop binary links glibc dynamically,** gotth-glaze included. A binary built against a newer glibc can fail to start on an older one. gotth-glaze links the fewest libraries, but it is not free of that tie.
- **Fyne and Cogent** need only base X11, Wayland or GL libraries, which a desktop install has.
- **Wails and GoTTH** need WebKitGTK. webview_go, GoTTH's shell, pins `webkit2gtk-4.0` (`webview_go webview.go:9`), which Ubuntu 24.04 does not ship, so a stock build fails there. Wails v3.1.0 defaults to GTK4 and needs `-tags gtk3` for the GTK3 backend.
- **gotth-glaze** still needs WebKitGTK present to open a window, but because it loads it at run time, it cross-compiles without Docker.
- **MIQT** needs Qt 6 at run time, from the system or shipped with the app.

## 3. Do I need the web, native, or both?

### Where Go runs, and as whom

| Route | Where the Go code runs | Acts on the host as |
|---|---|---|
| Native toolkit (Fyne, Cogent, MIQT desktop) | the user's machine | the user |
| WebView shell (Wails, GoTTH, gotth-glaze, miqt-gotth desktop) | the user's machine | the user |
| Local server plus browser (GoTTH or Wails server run by the user) | the user's machine | the user |
| Shared server (Wails, GoTTH, Fyne and Cogent web-remote) | the server | the server account |
| WASM (Fyne, Cogent web) | the browser sandbox | nothing on the host |

Every app shows the running user on the desktop. The four apps with a web target label the server process user on the web (`results/<id>/capabilities.md`).

### What reaching both costs in code

| | fyne | cogent | wails | gotth |
|---|---|---|---|---|
| Shared lines | 2401 | 2561 | 3867 | 3392 |
| Desktop-only lines | 250 | 258 | 248 | 590 |
| Web-only lines | 130 | 100 | 197 | 231 |
| Dual-target overhead | 13.7% | 12.3% | **10.3%** | 19.5% |

Non-blank lines, with generated and vendored code excluded (`loc` in `metrics.json`).
- Wails' overhead is understated, because some `isWeb` branches sit in shared files (`results/judge2/scores.md`).
- GoTTH's desktop share is larger because its desktop shell carries native menus and dialogs.
- MIQT is n/a: it has no web target.

The overheads are close. Reach differs mainly in what the user gets on each side.

### Native desktop widgets

- **MIQT:** real Qt widgets, QMenuBar and QFileDialog.
- **Wails:** HTML controls, with a native menu bar, native file dialogs and notifications. Reveal in Finder selects the exported file (user-checked, `results/wails/fixpass.md`).
- **Fyne:** framework-drawn controls and dialogs.
  - The window keeps a minimum size from its content (`fyne v2.8.2 internal/driver/glfw/window_desktop.go:300`).
  - Linux notifications go over D-Bus (`app/app_xdg.go:80-81`).
- **GoTTH:** HTML controls. The default build uses an in-page menu. A `-tags native` macOS build adds a real menu bar (user-checked, `results/gotth/fixpass.md`).
- **Cogent:** no native menu bar, a Cogent-drawn file picker, and no notification API. The minimum window size is set through the exported glfw handle (user-checked, `results/cogent/fixpass.md`).

### Custom rendering

| | fyne | cogent | wails | gotth | miqt |
|---|---|---|---|---|---|
| Custom rendering lines | 581 | 550 | 100 | **90** | 300 |

Lines of code for the three custom cells (stock-level bar, status badge, star rating). The HTML stacks draw them in about 100 lines of markup and CSS. The canvas toolkits (Fyne, Cogent) build them from shapes in over 500 lines of Go, and MIQT's Qt paint delegates take 300.

### Style (blind)

Source: `results/judge2/style_blind.md`, from the dark table view only.
- **Wails and GoTTH** share the shadcn look and tie at 4.
- **Fyne and MIQT** score 3: clean but plain.
- **Cogent** scores 2, for clipped columns, unboxed form fields and a stray character in a field.

### Native desktop shells for a web-first Go app

GoTTH is a web app. Two other shells give it a desktop window:
- **gotth-glaze:** a cgo-free webview, from the glaze library. glaze is new; it is included to show that the combination is feasible, not as a recommendation of glaze itself.
- **miqt-gotth:** MIQT's Qt WebEngine view.

| macOS | gotth (webview_go) | gotth-glaze | miqt-gotth |
|---|---|---|---|
| Cold build (s) | 9.4 | 8.1 | 133 |
| Incremental build (s) | 1.6 | 0.8 | 3.9 |
| Bundle (MB) | 11.1 | 9.5 | 335 |
| Scroll, max frame gap (ms) | 92 | 83 | 280 |
| Cards, max frame gap (ms) | 73 | 79 | 213 |
| Switch to cards (ms) | 969 | 970 | 1,859 |
| Peak memory (MB) | 730 | 766 | 1,193 |
| Cross builds | Linux, Windows | Linux, Windows, no Docker | none |

**gotth-glaze**
- It builds with `CGO_ENABLED=0` and cross-compiles for Linux and Windows without Docker (`results/gotth-glaze/shell.md`).
- It feels the same as GoTTH: no stutter, and a card switch close to 1 s.
- It has a native macOS menu, and native file dialogs on Linux.
- Library gaps:
  - no Linux menu bar (`glaze@v0.0.83 menu/menu_linux.go:1-3`)
  - no menu shortcuts on Windows (`menu/menu_windows.go:9-11`)
  - no notification outside macOS (`native@v0.1.18 notify/notify_other.go:1,5`), and no native clipboard on Linux (`clipboard/clipboard_linux.go:26,28`). Windows has a Win32 clipboard backend (`clipboard/clipboard_windows.go`).
- The Windows build was compiled but not run.

**miqt-gotth**
- It has native Qt menus, file dialogs and clipboard (`results/miqt-gotth/capabilities.md`).
- The bundle is 335 MB, mostly Qt WebEngine.
- Desktop scrolling stutters visibly, and the card switch is a noticeable wait of almost 2 s.
- There is no ready-made cross-build route, because MIQT's Docker images do not include WebEngine (`results/miqt-gotth/shell.md`). It does build natively on Linux (see Linux).

## Build and ship

### Build loop (macOS)

| | fyne | cogent | wails | gotth | miqt |
|---|---|---|---|---|---|
| Plain `go build` from clean | yes | yes | after the frontend build | after `templ generate` | yes |
| Cold `make build` (s) | 19.1 | 23.5 | 39.2 | **9.4** | 113 |
| Incremental Go change (s) | **1.5** | 2.2 | 2.5 | 1.6 | 2.9 |
| Incremental UI change (s) | | | 4.1 | 1.3 | |
| No-op build (s) | **0.2** | 1.1 | 3.7 | 0.5 | **0.2** |

- An incremental rebuild is a short wait of 1.5 to 3 s everywhere.
- Cold builds differ more. MIQT's first build takes about 2 minutes, because the Qt bindings compile.

### Toolchain

- **Fyne and Cogent:** Go, the Xcode command-line tools (cgo), and CLIs installed with Go. Their cross builds use Docker.
- **Cogent packaging** also needs Python's `pip` on PATH: `core pack` runs `pip install dmgbuild` to make a DMG. On this machine only `pip3` was on PATH, so `make package` failed after building the .app (`results/DEVIATIONS.md`). Build, run and tests need nothing extra.
- **GoTTH:** adds templ, shadcn-templ and a Tailwind binary. Node is used only for one browser test.
- **Wails:** needs Node and npm.
- **MIQT:** needs Qt, pkgconf and a C++ compiler.

Sources: `results/<id>/toolchain.md`. These describe macOS.

**On Linux** (Ubuntu 24.04, `linux/Dockerfile`), every build dependency came from distro packages plus the Go and Node tarballs; nothing was built from source by hand:
- **Fyne and Cogent:** a C compiler, pkg-config, and the GL, X11, Wayland and xkbcommon dev packages.
- **Wails:** the GTK 4 and `libwebkitgtk-6.0-dev` packages (GTK 3 and `libwebkit2gtk-4.1-dev` for `-tags gtk3`), plus Node and npm.
- **GoTTH:** GTK 3 and WebKitGTK dev packages, and a pkg-config alias, because webview_go asks for `webkit2gtk-4.0`, which Ubuntu 24.04 no longer ships.
- **MIQT:** the distro's Qt 6 dev packages (6.4.2), so there is no Qt build to wait for; miqt-gotth adds the Qt WebEngine packages.

The Dockerfile installs one superset for all apps, so it does not show the minimum set per app, and no Linux build times were taken.

### Runtime and bundle

| | fyne | cogent | wails | gotth | miqt |
|---|---|---|---|---|---|
| macOS app bundle (MB) | 23.0 | 26.5 | 11.4 | **11.1** | 111 (78 of it Qt) |
| What the user needs | nothing extra | nothing extra | system WebView | system WebView | bundled Qt |

All are release builds. Cogent's bundle size was measured by hand from the .app `core pack` built before its DMG step failed (`results/DEVIATIONS.md`). On Linux, GoTTH also needs `zenity` and a clipboard tool, and the WebView apps need WebKitGTK (see Linux).

### Cross builds

All base apps and variants cross-compile for Linux and Windows except miqt-gotth (no route).
- Cogent's own `core build -t` route fails for both Linux and Windows. The Makefile falls back to Docker (GL headers for Linux, mingw for Windows), which produced x86-64 binaries (checked with `file`, not run).
- gotth-glaze is the only one that needs no Docker.
- Cross builds were compiled, not run. Linux function comes from native builds in the container.

### Tests

All tests passed, with no failures:

| | fyne | cogent | wails | gotth | miqt |
|---|---|---|---|---|---|
| Tests passed | 12 | 12 | 34 (includes Playwright) | 12 | 18 |

Source: `tests` in `metrics.json`. GoTTH's browser test needs Node and Chrome.

## Linux

Function was checked in an Ubuntu 24.04 arm64 container, on X11 (Xvfb with openbox) and on Wayland (sway headless). Details and file citations are in `results/LINUX.md` and `results/<id>/linux.md`.

| App | Starts, X11 and Wayland | Menu bar | File dialogs | Notification | Clipboard | Reveal |
|---|---|---|---|---|---|---|
| fyne | Y | in-window, framework-drawn | Fyne-drawn | Y (D-Bus) | Y | opens the parent folder |
| cogent | Y | in-window buttons | Cogent-drawn | none (no API) | Y | opens the parent folder |
| wails (GTK3 build) | Y | native GtkMenuBar | native GTK3 chooser | Y (D-Bus) | Y | opens the folder |
| gotth (webview_go) | Y | in-page | zenity dialogs | Y | Y (xclip, wl-clipboard) | opens the folder |
| miqt | Y | native QMenuBar | Qt QFileDialog | only with a tray watcher | Y | opens the folder |
| gotth-glaze (GTK3 path) | Y (GTK3 path only) | in-page (glaze has no Linux menu) | native GtkFileChooserNative | none (not supported by the library) | through the page only | opens the parent folder |
| miqt-gotth | Y | native QMenuBar | Qt QFileDialog | only with a tray watcher | Y | opens the folder |

- `make test` passes for all apps on Linux, with no failures. GoTTH, gotth-glaze and miqt-gotth pass 11 with 1 not feasible (a browser test that needs Chrome). Wails passes 34 of 34 with Chromium.
- No app selects the file when revealing it. All of them open a folder.
- On Ubuntu's Qt 6.4, miqt-gotth loses some styling. The likely cause is that its older Chromium lacks a CSS function the Tailwind v4 stylesheet uses; this was not verified in source (`results/miqt-gotth/linux.md:36-38`).
- The Linux runs found three bugs in our apps, all now fixed.

**Runtime dependencies on Linux:**
- GoTTH needs `zenity` (or qarma or matedialog), plus xclip, xsel or wl-clipboard.
- webview_go links `webkit2gtk-4.0`, absent on Ubuntu 24.04; the container used a pkg-config alias.
- Wails v3.1.0 defaults to GTK4. The GTK3 backend needs `-tags gtk3`.
- MIQT notifications need a StatusNotifierWatcher, which stock GNOME does not have.

**Not verified on Linux:**
- **Timings.** The container renders in software at an unrepresentative resolution.
- **Sharpness.**
- **GTK4 rendering** for the default Wails build and for glaze's GTK4 path. Both start with the WebKit sandbox on, but their window content could not be captured (`results/wails/linux.md`, `results/gotth-glaze/linux.md`).
- **Real desktop environments.** GNOME and KDE were not tested, so neither were portals, global menus or real notification servers.

**Container concessions (method notes):**
- The sandbox-on runs of the Wails GTK4 build and glaze's GTK4 path ran as an unprivileged user, with `seccomp=unconfined` and `systempaths=unconfined`.
- glaze's GTK3 path was run by moving `libwebkitgtk-6.0` aside.
- miqt-gotth was checked with the Chromium sandbox on, as a normal user, with `seccomp=unconfined`. Its feature checks ran with the sandbox disabled.
- Qt was the distro's 6.4.2, not 6.12.
- Weston's headless compositor has no seat, so Fyne and Cogent cannot start there. Wayland results come from sway instead.

## Scores

Scores are 1 to 5, from `results/judge2/scores.md`, where the justifications and file citations are. The clean re-run and the hands-on check changed these scores (`results/judge2/changes.md`): Fyne performance 4 to 5, Cogent reach 5 to 4, and fyne-fitcap 4 to 5. Wails performance went 5 to 4 on the numbers and back to 5 after the hands-on check; wails-fitcap went 5 to 3 and back to 5. gotth-fitcap went 2 to 4 after its own hands-on check.

| Factor | fyne | cogent | wails | gotth | miqt |
|---|---|---|---|---|---|
| 1 Bootstrap and build loop | **4** | **4** | 3 | 3 | 3 |
| 2 Runtime requirements | **5** | **5** | 4 | 3 | 3 |
| 3 Toolchain footprint | **4** | **4** | 2 | 3 | 2 |
| 4 Native desktop widgets | 3 | 2 | 4 | 3 | **5** |
| 5 Desktop plus web reach | 3 | **4** | **4** | **4** | N/A |
| 6 cgo-core tolerance | 4 | 4 | **5** | **5** | 3 |
| 7 Code clarity | **4** | 3 | 2 | 3 | 3 |
| 8 Testability | **4** | **4** | **4** | 3 | **4** |
| 9 Performance with large data | **5** | **5** | **5** | 3 | 2 |
| 10 Style appeal (blind) | 3 | 2 | **4** | **4** | 3 |
| 11 Native host access | 4 | 4 | 4 | 4 | 4 |
| 12 Custom rendering | 3 | 4 | **5** | 4 | 3 |

No weighted total is given. Readers will weight these differently.

> **What -fitcap means:** the same app with its data layer rewritten to suit its toolkit, still limited to the shared core's 2 requests in flight. It answers "is the slowness the toolkit, or our data layer?" See section 1.

| Variant performance | fyne-fitcap | cogent-fitcap | wails-fitcap | gotth-fitcap | miqt-fitcap |
|---|---|---|---|---|---|
| Score | **5** | **5** | **5** | 2 | 4 |

| GoTTH shells | gotth (webview_go) | gotth-glaze | miqt-gotth |
|---|---|---|---|
| Build loop | 3 | 4 | 2 |
| Runtime requirements | 3 | 4 | 1 |
| Toolchain | 3 | 4 | 2 |
| Native integration | 3 | 3 | 4 |
| Cross builds | 4 | 5 | 1 |
| Performance | 3 | 3 | 2 |
| Linux function | 3 | 3 | 2 |

## Trade-offs by situation

These follow from the results above. They are conditional, not rankings.

- **If native feel matters most on the desktop:** MIQT gives real Qt widgets and dialogs. It costs a large bundle, slow cold builds, Qt tooling, Qt libraries at run time, and no browser target. Smooth large lists need a Qt-suited data layer (miqt-fitcap).
- **If you want desktop and web from one pure-Go codebase:** Fyne and Cogent are both smooth on desktop. On the web, Cogent stays smooth after a 1.4 s first load; Fyne loads 12 MB and stutters while scrolling. Neither has native menus or dialogs.
- **If a polished UI matters and Node in the toolchain is fine:** Wails has the smallest, fastest web build and native menus and dialogs. Its desktop frame gaps during a very fast scroll (91 to 201 ms) are measurable, but did not stand out in a hands-on check.
- **If the app is web-first:** GoTTH has the fastest cold build and runs a cgo core on web and desktop. Expect to own the list-windowing code and a 1 s card switch. For a desktop window, a cgo-free webview like glaze is the lighter shell if the core needs no cgo; miqt-gotth gives native Qt menus at a large cost in size and speed.
- **If the shipped binary should link as few native libraries as possible:** gotth-glaze links only base glibc libraries and loads WebKitGTK at run time. It still needs glibc and WebKitGTK on the machine. Every other desktop build links a native toolkit or WebView directly.
- **If Linux comes first:** in the container, Wails' GTK3 build was the only app with a native menu, a native file chooser and D-Bus notifications working out of the box. Verify timings, GTK4 and your desktop environment on real hardware first.

## Method and limits

- **Scope:** one machine, one workload, and one implementation per stack, written by AI agents. A gap in an implementation is not proof of a framework limit.
- **Timings:** desktop timings are macOS arm64 only. Web timings are single runs. There are no Linux or Windows timings.
- **WebView engine:** desktop Wails and GoTTH run in macOS WKWebView; their web numbers are Chrome. The two engines behave differently on the same HTML (data-layer variants).
- **Judging:** scores other than style are not blind, because the code reveals the framework. Style was judged on one state only: dark theme, table view, nothing selected.
- **GoTTH patches:** GoTTH patches vendored shadcn-templ JS, and disables an htmx scan, for WebKit performance.
- **Agent notes:** benchmark numbers in the agents' own notes are not used. Only harness runs count.
- **History:** method changes are recorded in `results/DEVIATIONS.md` and `results/HARNESS_CHANGES.md`. Corrections to the agents' notes are in `results/ERRATA.md`. Superseded data is in `results/archive/`.
