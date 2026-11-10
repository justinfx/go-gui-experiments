# GGUIX judge scores

Scores are 1 to 5 per Appendix B. Only harness metrics (`results/<id>/metrics.json`) are used for numbers. Implementer timings were collected while the screen was locked and are ignored. miqt is desktop-only by design, so it is scored N/A for web reach and normally everywhere else.

| Criterion | fyne | miqt | cogent | wails | gotth |
|---|---|---|---|---|---|
| Bootstrap and build loop | 4 | 3 | 4 | 3 | 3 |
| Runtime requirements | 5 | 3 | 5 | 4 | 4 |
| Toolchain footprint | 4 | 2 | 4 | 2 | 3 |
| Native widgets and OS integration | 2 | 5 | 2 | 4 | 2 |
| Desktop and web reach | 3 | N/A | 5 | 4 | 4 |
| Core-constraint tolerance | 4 | 3 | 4 | 5 | 5 |
| Code clarity and complexity | 4 | 3 | 3 | 2 | 3 |
| Testability | 4 | 4 | 4 | 4 | 3 |
| Performance | 3 | 2 | 5 | 4 | 2 |
| Out-of-box style appeal | 3 | 3 | 3 | 5 | 4 |
| Native host access | 4 | 4 | 4 | 4 | 4 |
| Custom rendering | 3 | 3 | 4 | 5 | 4 |

The style scores come from the blind pass in `style.md`. The mapping is A=wails, B=fyne, C=cogent, D=gotth, E=miqt. This was inferred from the window titles visible in the screenshots, not from the secret label map. Because the titles were visible, the blinding was compromised.

## LOC by bucket (from `loc` in metrics.json; generated and vendored code excluded)

| App | shared | desktop | web | overhead % | custom rendering lines | longest func | notes |
|---|---|---|---|---|---|---|---|
| fyne | 2401 (Go) | 250 | 130 | 13.7 | 581 | 95 (`form.go:newDetailForm`) | Workarounds for framework gaps live inside shared code: a tappable sort header, an app-drawn row highlight, and a reflect+unsafe read of the table scroll offset (`capabilities.md`). |
| miqt | 22 | 2304 | 2 | 99.1 (by design; meaningless) | 300 | 138 (`form_desktop.go`) | All UI is `_desktop.go`. 3 `marker_errors` (Makefile:75/80, `main_desktop.go:32` uses an invalid `gguix:target bench` marker). |
| cogent | 2555 (Go) | 238 | 100 | 11.7 | 550 | 181 (`form.go:newDetailForm`) | Workaround glue: a virtual `rowsTable` on `core.ListBase` because `core.Table` needs reflected slices; in-window menus; synthetic key chords for Edit. 2 Makefile marker errors. |
| wails | 2976 (880 Go, 1107 TS, 743 Svelte, 124 CSS) | 239 | 128 | 11.0 (understated, see below) | 100 | 51 | Workarounds: `scaled-scroll.ts` for the browser 33.5M px height cap, a custom split divider, and column resize. Excluded: 662 generated TS lines and 2593 vendored shadcn lines. |
| gotth | 3331 (978 Go, 982 templ, 622 JS, 692 CSS) | 461 | 149 | 15.5 | 90 | 93 (`server.go:handleSave`) | Large hand-written JS windowing (`static/app.js`, evict/spacer logic) as a workaround for having no virtual list. Excluded: about 26k generated lines and 27k vendored lines. |

Raw totals are not comparable. Fyne and Cogent are single-language Go. Wails and GoTTH split their shared code across 4 languages.

## Section 3.3 sample check (unmarked target branches)

- **wails: found.** `frontend/src/lib/state.svelte.ts` branches on `this.isWeb` / `this.isDesktop` outside markers at lines 174 (`hasExport = this.isDesktop`), 199 (browser Notification), 430 (`importNotes` web path), and 467 (web download stream). `frontend/src/App.svelte:35` (`if (!app.isDesktop) return;` for the menu state) is also unmarked. The marked blocks at App.svelte:39 and :75 show the rule was known. The measured 11.0% overhead is therefore understated, and this counts against clarity.
- **gotth: minor.** `server.go:131` and `views/page.templ:52-61` are marked. However, `views/page.templ:73` (`if vm.Desktop { @desktop.Menubar }`) and `server.go:312` are unmarked desktop branches. The view itself sits under `views/desktop/`, so the leak is small.
- **fyne, cogent: clean** in the sample. No `runtime.GOOS`, `syscall/js`, or platform checks appear in shared files. Target code sits in `platform_*`, `menus_*`, and `export_*` with build tags.
- **miqt:** N/A (everything is desktop). The marker misuse is noted above.

## Per-app justification

### fyne
- **Bootstrap 4:** `plain_go_build_from_clean.rc=0` (2.6 s), `cold_build.secs=20.25`, `incremental_build.BuildMarker.secs=2.31`, `warm_noop_build=0.18`. One `make run`, but there is no hot reload.
- **Runtime 5:** `sizes.app_bundle_kb=33388`, `external_dylibs=[]`.
- **Toolchain 4:** go + the `fyne` CLI (Go-installed). Docker and fyne-cross (7 GB of images) are needed only for cross builds (`toolchain.md`).
- **Native 2:** Many fallbacks in `capabilities.md`: no window min-size, no segmented control, no sortable header, no row selection, no spin box, Fyne-drawn file dialogs, and no default Edit menu.
- **Reach 3:** The same Go code runs as WASM with 13.7% overhead. However, the wasm is `web.wasm_files.raw_kb=39309` and web perf is poor (`web.bench-web.events.scroll.total_ms=54341`, `ready.ms=1325`, `export.max_gap_ms=768`).
- **Core 4:** `cgoprobe` web FAIL, web-remote client/server PASS.
- **Clarity 4:** Compact, single language, and honest markers (1 inline block). Deducted for the reflect+unsafe offset hack.
- **Tests 4:** `tests.line.passed=12`, using the Fyne test driver. The web target has no automated tests.
- **Perf 3:** Desktop jump 138 ms, scroll p95 gap 28.9 ms, cards p95 8.5 ms, 0 placeholders. However, `bench.memory.peak_process_tree_mb.median=1427.6` is the highest of all apps.
- **Host 4:** Native on desktop, `unavailable`/`server-process` on web as specified. Reveal uses the core helper.
- **Custom 3:** Cards are smooth (`cards_scroll.max_gap_ms=14`), but the code is the largest (581 lines).

### miqt (control group)
- **Bootstrap 3:** `cold_build.secs=110.75` (bindings compile), incremental 1.89 s, plain `go build` works. Needs `brew install qt pkgconf`.
- **Runtime 3:** `app_bundle_kb=113184` with 80 MB of bundled Qt frameworks after macdeployqt.
- **Toolchain 2:** Qt 6 (about 1 GB), pkgconf, C++ toolchain. Cross builds need multi-GB Docker images (`cross.cross-linux.secs=1459`, windows FAIL).
- **Native 5:** Real Qt widgets, menus, dialogs, and delegates. The only fallback is the QDateEdit "None" empty state.
- **Reach N/A.**
- **Core 3:** Desktop cgoprobe PASS. There is no web route to test, so the remote-split option is unproven.
- **Clarity 3:** Readable but long (`longest_go_func_lines=138`) with 3 `marker_errors`.
- **Tests 4:** `tests.line.passed=18`, in-process harness (`app_test.go`).
- **Perf 2:** `scroll.p95_gap_ms=231.8`, `scroll.placeholder_steps=117`, `jump.ms=652`, `cards_scroll.switch_ms=908`, `cards_scroll.p95_gap_ms=307`. These are poor for native Qt and are most likely an implementation gap in the model/pager fetch path, not a Qt limit.
- **Host 4:** Native identity. Reveal Config uses QDesktopServices; Reveal Export uses the core helper.
- **Custom 3:** The QStyledItemDelegate code is compact (300 lines), but card scrolling is janky (above).
- **Observed bug:** The selected table row is not highlighted although the form shows row 1. `syncSelection` (`app_desktop.go:470-490`) does set the selection, so this is likely the inactive-window macOS palette or the order of selection vs. model load. Implementation.

### cogent
- **Bootstrap 4:** Plain `go build` works (3.96 s), cold build 22.4 s, incremental 2.15 s. However, `warm_noop_build=2.78` is the slowest no-op build.
- **Runtime 5:** `app_bundle_kb=27112`, no external dylibs.
- **Toolchain 4:** go + the Go-installed `core` CLI. The webgpu module family is about 550 MB in the module cache. The brief's wrong CLI install path was an evaluator error and is not counted.
- **Native 2:** No native menu bar, framework-drawn file dialogs, no OS notification, no min-size API (`capabilities.md`). All are framework limitations in v0.5.27.
- **Reach 5:** The lowest honest overhead (11.7%), with clean build-tag separation. The web bench runs near desktop speed (`web.bench-web.events.scroll.p95_gap_ms=3.9`). The wasm is large (37 MB raw, 8 MB gzip).
- **Core 4:** Web FAIL, web-remote PASS.
- **Clarity 3:** Single language, but `newDetailForm` is 181 lines and there is ListBase/keychord glue.
- **Tests 4:** 12 passed.
- **Perf 5:** The best of all apps: `scroll.p95_gap_ms=3.9`, `cards_scroll.p95_gap_ms=1.8`, `jump.ms=62.3`, peak 661 MB. 25 placeholder steps.
- **Host 4:** As specified. Reveal uses the core helper.
- **Custom 4:** Smooth, but verbose (550 lines).
- **Observed bug:** The web status bar stays "Loading..." after data loads. `statusText` returns Loading while `pager.Total()<0`, and the status refresh path (`ui.go:316`) only calls `NeedsLayout`. The same code works on desktop. Implementation (probable missing refresh/render after the first page on the wasm event loop), not confirmed as a framework bug.

### wails
- **Bootstrap 3:** `plain_go_build_from_clean.rc=1` (needs `frontend/dist`), cold build 40.6 s, `warm_noop_build=3.56`. Incremental is good (2.5 s Go, 4.0 s UI), plus Vite HMR in dev.
- **Runtime 4:** `app_bundle_kb=11612`, no external dylibs. Relies on the system WebView (WebKitGTK/WebView2 required on other OSes).
- **Toolchain 2:** Node/npm, 194 MB `node_modules`, 24 direct npm deps, and the wails3 CLI.
- **Native 4:** Native menus, dialogs, notifications, and clipboard. The controls are HTML (shadcn), and the split and column resize are custom.
- **Reach 4:** 11.0% measured (lowest), web transfer only 792 KB, and web perf is excellent. It loses a point because unmarked `isWeb` branches understate the overhead.
- **Core 5:** Server-mode web builds with cgoprobe and PASSes.
- **Clarity 2:** Spread across Go, TS, Svelte, and CSS, with unmarked target branches (section 3.3 above).
- **Tests 4:** `tests.line.passed=33` (Go + Vitest + Playwright). Broad coverage, but a heavy harness.
- **Perf 4:** `scroll.p95_gap_ms=12`, cards 6, jump 87 ms. However, `peak_total_mb.median=901` (WebKit 686 MB).
- **Host 4:** Native on desktop, `server-process` on web. Reveal uses the core helper (Wails has no reveal API).
- **Custom 5:** 100 lines with smooth cards (`cards_scroll.max_gap_ms=6`).

### gotth
- **Bootstrap 3:** The fastest cold build (10.43 s) and the fastest incremental builds (1.91/1.73 s). However, `plain_go_build_from_clean.rc=1` (templ generate needed), and the tools are templ, shadcn-templ, and a Tailwind binary.
- **Runtime 4:** `app_bundle_kb=13556`. System webview.
- **Toolchain 3:** Go-installed CLIs plus a 96 MB Tailwind standalone binary. Node is needed only for T9.
- **Native 2:** webview_go has no menus or dialogs, so it uses in-page menus and osascript/pbcopy fallbacks. The splitter and column resize are implementation gaps.
- **Reach 4:** The web is the native target (`initial_transfer_kb=1346`), but overhead is the highest (15.5%), with 8 marker blocks.
- **Core 5:** cgoprobe PASS on web, desktop, and export tests. Cross builds PASS for both targets.
- **Clarity 3:** About 1k lines of hand-written JS windowing. It also patched vendored shadcn-templ JS and an htmx scan, an undisclosed maintenance burden on vendored code.
- **Tests 3:** 12 passed. T9 needs Node + headless Chrome over the DevTools protocol.
- **Perf 2:** `scroll.p95_gap_ms=75`, `placeholder_steps=152`, `cards_scroll.switch_ms=973`, `search.ms=182`. Memory is moderate (717 MB total).
- **Host 4:** Correct labels. Native actions via osascript glue.
- **Custom 4:** 90 lines (server-rendered), but card scrolling shows placeholders (30) and p95 gap of 52 ms.

## Gap list: framework vs implementation

| App | Gap | Label |
|---|---|---|
| fyne | No window min-size, segmented control, table sort/row selection, spin box, or native file dialogs; scroll offset private | framework |
| fyne | Web WASM 39 MB and very slow web scroll/export | framework (single-threaded wasm + GL canvas) |
| fyne | 1.4 GB peak memory | implementation (likely cache size); unconfirmed |
| miqt | No web target | framework (by design) |
| miqt | Slow scroll/jump, 117 placeholder steps | implementation |
| miqt | Selected row not highlighted in screenshots | implementation |
| miqt | Cross-windows FAIL (Docker image Go 1.26) | framework tooling |
| cogent | No native menu bar, file dialogs, notifications, or min-size | framework (v0.5.27) |
| cogent | `core.Table` cannot virtualize without a slice, so a ListBase rewrite was needed | framework |
| cogent | Web status bar stuck on "Loading..." | implementation |
| cogent | Cross-compile unsupported | framework (documented) |
| wails | No reveal API; custom split and column resize | framework (shadcn-svelte list) / implementation |
| wails | Unmarked target branches | implementation |
| wails | `go build` fails without the frontend build | framework |
| gotth | No menus or dialogs in webview_go | framework |
| gotth | No draggable splitter and no column resize | implementation (shadcn-templ has `resizable`; author skipped it) |
| gotth | Patched vendored shadcn-templ JS and htmx scan | implementation workaround for library issues |
| gotth | Scroll jank and placeholders | implementation (DOM windowing approach) |
| all | Cogent CLI install path wrong in brief | evaluator error, not counted |
