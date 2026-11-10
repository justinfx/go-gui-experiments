# GGUIX results report

Five Go GUI approaches each implemented one spec'd app (`SPEC.md` v1.2: 1M-row paged table with custom cells, 1M-item card list, detail form, prefs, theming, background job, native menus/dialogs/notifications, host identity) on top of an unchanged shared Go `core`. Each was built by an isolated agent from its own brief, then measured by one harness (`bench/measure.sh`) on an Apple M1 Max, macOS 26.6.2, Go 1.27.3 (see `ENVIRONMENT.md`). Deviations and evaluator errors are in `DEVIATIONS.md`; judge material is in `judge/`.

This report gives guidance by situation, not a single winner. MIQT has no browser target (it supports desktop and Android ARM64, and binds Qt WebEngine for embedded HTML) and serves as the control group: it shows what dropping the web requirement buys.

## Approaches

| id | Stack | Web route |
|---|---|---|
| fyne | Fyne v2.8.2 | Go WASM (canvas), or WASM UI plus Go server (web-remote) |
| cogent | Cogent Core v0.5.27 | Go WASM (canvas, via `core build web`; v0.5 also offers a GoScript `js` platform, not used), or web-remote |
| wails | Wails v3.1.0 + Svelte 5 + Tailwind v4 + shadcn-svelte | Wails server mode (Go server, HTML UI) |
| gotth | templ + htmx + Tailwind + shadcn-templ, webview_go shell | Go web server is the primary target |
| miqt | MIQT v0.15.0 on Qt 6.12.0 | none |

## Headline findings

1. **Cogent had the smoothest frame gaps (scroll, cards) on desktop and web in the first pass.** Desktop scroll p95 frame gap 3.9 ms, cards 1.8 ms, jump 62 ms; its WASM build kept the same 3.9 ms scroll p95 in Chrome, though with 58 placeholder steps, and Wails was faster on web ready, jump and sort. The cost is a 37 MB wasm (8 MB gzip) and missing native desktop features (menu bar, file dialogs, notifications), which are framework limits in v0.5.27.
2. **Wails gave the best all-round result for desktop plus web**, with smooth desktop and web performance, a 792 KB web payload, native menus/dialogs/notifications, both cross builds passing, and the highest blind style score (tied). The costs: Node/npm (24 direct deps, about 194 MB node_modules), code spread across Go, TS, Svelte and CSS, the WebKit process memory (peak 901 MB total), and a custom scaled scroller to get past the browser's maximum element height for 1M rows.
3. **Fyne shared its UI across both targets in a single language, but its web build was the weakest.** The same Go UI ran as WASM with 13.7% target-specific code, but the WASM payload was 39 MB and web scrolling stalled (p95 190 ms). Desktop performance was fine but peak memory was the highest measured (1.4 GB). Several widget gaps (no sortable header, row selection, spin box, segmented control) are framework limits.
4. **GoTTH had the simplest deployment story and the fastest builds**, with a single Go binary, no Node needed to build, the fastest cold build (10.4 s), the fastest UI rebuild (1.7 s), and a Go-change rebuild (1.9 s) tied with MIQT. It ran the cgo core on web and desktop, and both cross builds passed. Its large-list performance was the weakest of the HTML approaches (scroll p95 75 ms desktop, 113 ms web, 152 to 166 placeholder steps), The variant pass showed the web side was an implementation issue (web p95 113 to 54 ms), but its desktop webview regressed (75 to 317 ms) for a reason not found. The desktop shell (webview_go) has no menu or dialog API; adding them needs cgo against the native window handle.
5. **MIQT, the control, delivered the most native desktop experience** (real Qt widgets, menus, dialogs, paint delegates) at the cost of a 113 MB bundle, a 111 s cold build, about 1 GB of Qt tooling, and no web. Its first-pass large-list performance was poor (scroll p95 232 ms). The variant pass showed this slowness was removable by redesigning the data layer (Qt model, coalesced repaints, speed-based prefetch, more parallel fetches), not an inherent Qt limit: it then had the lowest scroll and card frame gaps measured (p95 0.9 and 0.1 ms), at the cost of slower start-up, jump and card switch, and higher idle memory.

## Measured results (harness only)

### Build and package

| | fyne | cogent | wails | gotth | miqt |
|---|---|---|---|---|---|
| Plain `go build` from clean | yes, 2.6 s | yes, 4.0 s | no (needs frontend/dist) | no (needs templ generate) | yes, 12.2 s |
| Cold `make build` (s) | 20.3 | 22.4 | 40.6 | **10.4** | 110.8 |
| Incremental Go change (s) | 2.3 | 2.2 | 2.5 | **1.9** | **1.9** |
| Incremental UI change (s) | n/a | n/a | 4.0 | **1.7** | n/a |
| No-op build (s) | **0.18** | 2.78 | 3.56 | 1.17 | 0.19 |
| App bundle (MB) | 33 | 27 | **12** | 14 | 113 (80 MB Qt frameworks) |
| External dylibs | none | none | none (system WebKit) | none (system WebKit) | bundled Qt |
| Tests passed (failed) | 12 (0) | 12 (0) | 33 (0) | 12 (0) | 18 (0) |
| cgoprobe desktop / web / web-remote | PASS / FAIL / PASS | PASS / FAIL / PASS | PASS / PASS / n/a | PASS / PASS / n/a | PASS / n/a / n/a |
| Cross Linux / Windows | PASS / FAIL | FAIL / FAIL | PASS / PASS | PASS / PASS | PASS (1459 s, emulated) / FAIL |

Cross-build notes: these harness results differ from two agents' own reports (Fyne Windows and Cogent Linux had been reported as passing with workarounds). Fyne's Windows failure is in fyne-cross's packaging step. Cogent Linux failed because the container's Go toolchain was broken ("package image is not in std"), and Cogent Windows failed because cross compiling from macOS has no Windows C toolchain, so the cgo files in the webgpu GLFW driver drop out (the error reads "build constraints exclude all Go files"). MIQT's Windows image ships Go 1.26, and downloading the Go 1.27 toolchain inside it failed with a TLS certificate error. These are mostly tooling fragility, not code faults, and are not re-investigated here.

### Desktop bench (median of 5 runs)

| | fyne | cogent | wails | gotth | miqt |
|---|---|---|---|---|---|
| Ready (ms) | 577 | 638 | 622 | 608 | **430** |
| Jump to row 500k (ms) | 138 | **62** | 87 | 230 | 652 |
| Table scroll p95 gap (ms) | 28.9 | **3.9** | 12 | 75 | 232 |
| Table scroll placeholder steps | **0** | 25 | 24 | 152 | 117 |
| Cards scroll p95 gap (ms) | 8.5 | **1.8** | 6 | 52 | 308 |
| Table to cards switch (ms) | **104** | 111 | 193 | 973 | 908 |
| Sort (ms) | 281 | **261** | 285 | 360 | 614 |
| Search (ms) | 70 | **65** | 100 | 182 | 72 |
| Idle memory (MB) | 216 | 325 | 306 (175 WebKit) | 264 (138 WebKit) | **196** |
| Peak memory (MB) | 1428 | 661 | 901 (686 WebKit) | 717 (554 WebKit) | 706 |

Run-to-run spread was tight for most metrics, but MIQT and Cogent had wide ranges on ready, jump, search and sort (min and max are in metrics.json), so treat small differences there as noise.

### Web bench (Chrome via Playwright, single run per app)

| | fyne wasm | fyne remote | cogent wasm | cogent remote | wails | gotth |
|---|---|---|---|---|---|---|
| Initial transfer (KB) | 39,647 | 12,689 | 6,942 | 8,552 | **792** | 1,346 |
| Ready (ms) | 1325 | 647 | 703 | 650 | **169** | 191 |
| Jump (ms) | 648 | 1014 | 254 | 115 | **77** | 244 |
| Scroll p95 gap (ms) | 190 | 198 | **3.9** | **3.7** | 7.4 | 113 |
| Cards p95 gap (ms) | 120 | 138 | 1.1 | **0.1** | 2.2 | 38 |
| Export max gap (ms) | 768 | 142 | 4.3 | **0.1** | 2.5 | 20 |
| Peak JS heap (MB) | 21.6 | 13.5 | 9.3 | **7.9** | 43.8 | 12.4 |
| Peak DOM nodes | 107 | 109 | 1,439 | 1,634 | 78,418 | 110,878 |

The canvas-based WASM UIs keep the DOM tiny. Wails and GoTTH render HTML and grow the DOM much larger, but Wails stays smooth.

### Code (non-blank lines; generated and vendored excluded)

| | fyne | cogent | wails | gotth | miqt |
|---|---|---|---|---|---|
| Shared | 2401 Go | 2555 Go | 2976 (880 Go, 1107 TS, 743 Svelte, 124 CSS) | 3331 (978 Go, 982 templ, 622 JS, 692 CSS) | 22 |
| Desktop-only | 250 | 238 | 239 | 461 | 2304 |
| Web-only | 130 | 100 | 128 | 149 | n/a |
| Dual-target overhead | 13.7% | **11.7%** | 11.0% (understated, so not bolded) | 15.5% | N/A |
| Custom rendering lines | 581 | 550 | 100 | **90** | 300 |
| Longest Go function | 95 | 181 | **51** | 93 | 138 |
| Excluded vendored | 0 | 0 | 2,593 (shadcn-svelte) | about 27,000 (shadcn-templ, htmx) | 0 |

How to read these:
- The totals are not comparable on their own. Fyne and Cogent are single-language Go; Wails and GoTTH split into four languages.
- Wails' overhead is understated: the judge found unmarked `isWeb`/`isDesktop` branches in shared files (`frontend/src/lib/state.svelte.ts:174,199,430,467`, `App.svelte:35`). GoTTH has two minor unmarked branches.
- Workaround code sits inside the "shared" counts:
  - Fyne: a tappable sort header, an app-drawn row highlight, and a reflect read of the private table scroll offset.
  - Cogent: a virtual table on `core.ListBase`, because `core.Table` needs a full slice.
  - Wails: a scaled scroller, a custom split divider and column resize.
  - GoTTH: about 600 non-blank lines of hand-written JS (all of `static/app.js`, of which windowing is a part).
- HTML stacks draw custom cells in far fewer lines (90 to 100) than canvas toolkits (550 to 581) or Qt delegates (300).

## Judge scores (1 to 5)

Style uses the properly blind re-judge (`judge/style_blind.md`, title bars removed, labels P to T). The first style pass was not blind (window titles named frameworks) and is shown for reference only. Other criteria come from `judge/scores.md`, which necessarily saw code.

| Criterion | fyne | cogent | wails | gotth | miqt |
|---|---|---|---|---|---|
| Bootstrap and build loop | 4 | 4 | 3 | 3 | 3 |
| Runtime requirements | 5 | 5 | 4 | 4 | 3 |
| Toolchain footprint | 4 | 4 | 2 | 3 | 2 |
| Native widgets and OS integration | 2 | 2 | 4 | 2 | **5** |
| Desktop and web reach | 3 | **5** | 4 | 4 | N/A |
| cgo-core tolerance | 4 | 4 | **5** | **5** | 3 |
| Code clarity | **4** | 3 | 2 | 3 | 3 |
| Testability | 4 | 4 | 4 | 3 | 4 |
| Performance | 3 | **5** | 4 | 2 | 2 |
| Style, blind | 2 | 3 | **4** | **4** | 3 |
| (Style, not blind) | 3 | 3 | 5 | 4 | 3 |
| Native host access | 4 | 4 | 4 | 4 | 4 |
| Custom rendering | 3 | 4 | **5** | 4 | 3 |

No weighted total is given; different readers will weight these very differently.

## Framework limits vs implementation gaps

| App | Gap | Cause |
|---|---|---|
| fyne | No segmented control, sortable header, row selection or spin box; Fyne-drawn file dialogs on macOS and Windows (native only for Linux flatpak and mobile); scroll offset private. No min-size call, but window min size follows the content's MinSize, which met the requirement | framework |
| fyne | 39 MB wasm, slow web scroll and export | framework (probable; cause not isolated) |
| fyne | 1.4 GB peak memory | unknown; fyne-fit with an 8,000-row cache cap still peaked at 1.25 GB, so cache size alone does not explain it |
| cogent | No native menu bar, file dialogs or notifications; no portable min-size API (reachable through the exported glfw handle `desktop.Window.Glw`, untried); `core.Table` needs a full in-memory slice; the `core` CLI cannot cross compile desktop targets (Docker works) | framework (v0.5.27) |
| cogent | Web status bar stuck on "Loading..." | implementation (probable missing redraw) |
| wails | Reveal-in-Finder not implemented (v3.1.0 has `app.Env.OpenFileManager(path, true)`, which the agent missed) | implementation |
| wails | go build needs the frontend built first | framework |
| wails | Custom split and column resize (shadcn-svelte does offer Resizable, but it was not in the brief's component list); unmarked target branches | implementation / brief scope |
| gotth | No menu, dialog or notification API in webview_go; it exposes the native window handle (`Window()`), so these need hand-written cgo, which the app did not attempt (it used osascript and pbcopy) | framework gap, workable via native handle |
| gotth | Fixed splitter, no column resize, scroll jank | implementation (shadcn-templ has `resizable`) |
| gotth | Patched vendored shadcn-templ JS and an htmx scan for WebKit | implementation workaround for library behaviour |
| miqt | Slow scroll, jump and card list; selected row not highlighted | implementation (probable) |
| miqt | No browser target (MIQT v0.15.0 does bind Qt WebEngine and WebChannel, so a desktop app can embed HTML views; not used here) | by design |
| all | Wrong Cogent CLI install path in the brief | evaluator error, not counted |

## Variant pass: a data layer suited to each toolkit

The results above all use the prescribed paging layer. Fixing that layer suits some toolkits better than others, so each app got a second pass (`apps/<id>-fit`, brief `briefs/FIT_ADDENDUM.md`): the same agent model replaced the data path, and the view-update logic that depends on it, with a design idiomatic to its toolkit. The following stayed fixed:
- the core service and its 50 ms latency
- at most 200 rows per call, at most 10,000 rows in memory, and no preloading
- the bench script and all behaviour

Each agent's design and reasoning is in `results/<id>-fit/adapter.md`.

### Desktop bench, first pass vs variant (median of 5 runs)

| | fyne | fyne-fit | cogent | cogent-fit | wails | wails-fit | gotth | gotth-fit | miqt | miqt-fit |
|---|---|---|---|---|---|---|---|---|---|---|
| Ready (ms) | 577 | 565 | 638 | 586 | 622 | 622 | 608 | 662 | 430 | 686 |
| Jump (ms) | 138 | 115 | 62 | 82 | 87 | 80 | 230 | 222 | 652 | 228 |
| Scroll p95 gap (ms) | 28.9 | 21.1 | 3.9 | 4.3 | 12 | 9 | 75 | 317 | 232 | **0.9** |
| Scroll placeholder steps | 0 | 0 | 25 | 0 | 24 | 0 | 152 | 0 | 117 | 10 |
| Cards p95 gap (ms) | 8.5 | 11.8 | 1.8 | 2.2 | 6 | 5 | 52 | 61 | 308 | **0.1** |
| Cards placeholder steps | 0 | 0 | 0 | 0 | 0 | 0 | 30 | 0 | 67 | 0 |
| Switch to cards (ms) | 104 | 63 | 111 | 79 | 193 | 103 | 973 | 458 | 908 | 991 |
| Sort (ms) | 281 | 261 | 261 | 251 | 285 | 286 | 360 | 392 | 614 | 510 |
| Peak memory (MB) | 1428 | 1246 | 661 | 661 | 901 | 782 | 717 | 699 | 706 | 698 |
| Idle memory (MB) | 216 | 214 | 325 | 277 | 306 | 301 | 264 | 283 | 196 | 395 |
| Tests passed | 12 | 12 | 12 | 13 | 33 | 36 | 12 | 12 | 18 | 18 |

### Web bench, variants (Chrome, single run per app)

| | fyne-fit | cogent-fit | wails-fit | gotth-fit |
|---|---|---|---|---|
| Transfer (KB) | 39,663 | 6,942 | 794 | 2,260 |
| Jump (ms) | 510 (was 648) | 250 (was 254) | 79 (was 77) | 173 (was 244) |
| Scroll p95 gap (ms) | 173 (was 190) | 3.4 (was 3.9) | **2.8** (was 7.4) | 54 (was 113) |
| Scroll placeholder steps | 0 | 54 | 0 | 0 (was 166) |
| Cards p95 gap (ms) | 135 (was 120) | 0.1 | 2.5 | 12.9 (was 38) |

The web-remote route was not re-measured for the variants.

### Data-layer cost (non-blank lines, tests excluded)

| | New data layer | Other app lines changed | Replaced |
|---|---|---|---|
| fyne-fit | 438 Go (`adapter_store.go`) | +44 / -13 | `core.Pager` (shared, 0 app lines) |
| cogent-fit | 494 Go (`adapter_model.go`, `adapter_ui.go`) | +77 / -24 | `core.Pager` |
| wails-fit | 86 Go + 359 TS (`adapter_pages.go`, `lib/data/rowstore.svelte.ts`) | +54 / -48 (plus 23 generated binding lines) | 252-line TS pager |
| gotth-fit | 176 Go (`adapter_data.go`) + a rewritten windowing section of `static/app.js` | +401 / -316 | server-rendered windowing |
| miqt-fit | 457 Go (`adapter_store_desktop.go`, `adapter_models_desktop.go`) | +125 / -257 | `core.Pager` plus the old models |

### What the variants show

- **MIQT's first-pass slowness came from the data layer, not from Qt.** With a main-thread page cache, coalesced repaints, one model shared by both views, and no `dataChanged` storms reaching a hidden 1M-row `QListView`, table scroll p95 fell from 232 ms to 0.9 ms and cards from 308 ms to 0.1 ms. It now has the lowest scroll and card frame gaps measured, though 10 placeholder steps remain and its jump (228 ms) is slower than Cogent, Wails or Fyne. The agent also raised parallel fetches from 2 to 6 and added cancellation and speed-based prefetch. Two Qt-level costs remain:
  - The first switch to Cards still lays out every item in the `QListView`, about 1 s. Qt's batched layout avoids that but clamped the scroll range, so the agent dropped it.
  - A sort still inserts 1M rows into the table.

  Idle memory rose from 196 to 395 MB, a cost not yet investigated.
- **GoTTH improved in the browser and regressed in the desktop shell.** A real virtual scroller with htmx-loaded page blocks removed every placeholder step and halved web scroll stalls (p95 113 to 54 ms) and the card switch (web 1008 to 356 ms; desktop 973 to 458 ms). In WKWebView, though, desktop scroll p95 rose from 75 to 317 ms. The agent saw a 300 to 390 ms stall at the start of desktop scrolling and could not find its cause, so for GoTTH the desktop webview and Chrome behave quite differently.
- **Wails, Cogent and Fyne were already near their ceilings**, so the variants mostly removed placeholder steps and sped up the view switch:
  - Wails: per-page Svelte reactivity and batched fetches gave the best web scroll p95 measured (2.8 ms); Cogent-fit stays ahead on web cards (0.1 vs 2.5 ms).
  - Cogent: still the smoothest Go-drawn toolkit; median jump 62 vs 82 ms, within run noise. Its web export max gap regressed from 4.3 to 104.5 ms.
  - Fyne: improved a little, and is now limited by the cost of redrawing cell widgets. Its web build stays slow.
- **Getting this right took toolkit knowledge.** The MIQT fix depended on knowing exactly how a `QListView` reacts to model signals. The GoTTH fix replaced a hand-made windowing scheme with one built on htmx's own attribute-driven loading. That is the difference between a first pass and a second, informed pass by the same model.

The variant changes are data-path only, so screenshots and style scores were not redone. Deviations specific to the variants (copy fix, parallel fetch limits, extra bench rounds) are in `DEVIATIONS.md`.

## Guidance by situation

- **Go-only desktop app, native feel matters most:** MIQT gave real native widgets and OS integration, if you accept Qt tooling, a 113 MB bundle and slow cold builds. With a Qt-idiomatic model it also had the best scroll frame gaps, but only after a design that needs real Qt model/view knowledge.
- **Same Go UI on desktop and web, Go-only toolchain:** Cogent performed best and had the cleanest split. Fyne also works but its web build was large and slow in this workload. Both need the server split (web-remote) for a cgo core, and both lack native menus/dialogs to varying degrees.
- **Desktop plus web with a polished modern UI:** Wails, if Node/npm in the toolchain is acceptable. It had the best web payload, smooth performance, native desktop features, and passing cross builds.
- **Web-first app that also ships as a desktop app, minimal toolchain:** GoTTH. It had the fastest builds, a single Go binary and full cgo tolerance, but expect to own the large-list windowing code and to work around webview_go's missing menus and dialogs.
- **Large data with custom cells:** all five managed 1M rows. With the shared pager, Cogent and Wails were smoothest. With toolkit-suited data layers, MIQT had the best desktop scroll frame gaps and Wails the best web scroll p95, while GoTTH improved in the browser but not in its desktop webview.

## Caveats

- One machine, one workload, one implementation per stack, produced by AI agents (Claude Sonnet) from isolated briefs. An implementation gap is not proof of a framework limit.
- Agent-reported benchmark numbers were discarded (screen locked during their sessions); only harness runs are used.
- Harness fixes applied mid-run (port 8097, metric line splitting, `GOTTH_PORT`) and the style re-judge are recorded in `DEVIATIONS.md`.
- Judge scores beyond style cannot be blind, since code reveals the framework.
