# GGUIX: Go GUI framework evaluation (brief)

## Purpose
Compare five Go GUI approaches by having each one implement the same app from an explicit spec, then scoring the results with measured metrics, implementer reports, and a judge (style scored blind; other criteria necessarily see code).

## Frameworks under test
| id | Stack | Web route |
|---|---|---|
| fyne | Fyne v2.8 | WASM, plus WASM talking to a Go server ("web-remote") |
| cogent | Cogent Core v0.5.27 | WASM, plus web-remote |
| wails | Wails v3 + Svelte 5 + Tailwind v4 + shadcn-svelte | Wails server mode |
| miqt | MIQT (Qt 6 bindings) | none (no browser target; desktop and Android, Qt WebEngine available for embedded HTML) |
| gotth | templ + htmx + Tailwind + shadcn-templ, webview_go desktop shell | native web server |

## Method
- **Spec:** `SPEC.md` (v1.2) defines one app in detail: a 1M-row paged table with sort, search, export and custom cells (stock bar, status badge, stars), a 1M-item custom card list, a detail form, prefs, theming, a background job with progress, native menus, dialogs and notifications, and host user identity. Per-framework briefs in `briefs/` are generated from it by `tools/mkbrief`, which also checks that nothing leaks between them.
- **Shared core:** `core/` is a Go module (no dependencies) with a 1M-row in-memory service with 50 ms simulated latency, a pager (200-row pages, 20 cached, 2 in flight), prefs, host info, a remote HTTP client and server, a bench driver, and a cgo probe. Apps must use it unchanged (checked by `diff -r` at collection, per SPEC; the check output was not saved).
- **Implementation:** each app was built by an isolated Sonnet agent that saw only its own brief. The agents were not told the judging criteria. Outputs are in `apps/<id>`, with their notes (`toolchain.md`, `friction.md`, `capabilities.md`) in `results/<id>`.
- **Common interface:** every app exposes the same make targets: deps, dev, build, package, run, web, web-remote, test, bench, bench-web, cgoprobe, cross-windows, cross-linux, clean. Unsupported targets exit 3. Tests print `GGUIX_TESTS passed= failed= not_feasible=`, and the bench prints `GGUIX_METRIC {json}` events.
- **Measurement:** `bench/measure.sh <id>` is run sequentially on an unlocked macOS arm64 machine. It records cold, incremental and no-op build times, sizes, idle RSS, 5 desktop bench runs (jump, scroll, card scroll, sort, search, export, frame gaps, memory), tests, the cgo probe, cross builds, web benches (Playwright with Chrome: transfer size, heap, DOM nodes, frame gaps), and line counts (`tools/loc`). Results go to `results/<id>/metrics.json`.
- **Judging:** an Opus judge got screenshots and code labelled A to E in random order and scored them against a rubric. Style blinding failed (window titles named frameworks), so style was re-judged by a fresh agent on title-stripped screenshots labelled P to T. Other criteria see code and are not blind. The final write-up is `results/REPORT.md`.

## Evaluation factors
1. Bootstrap and iterative build experience (setup steps, build times, hot reload)
2. Runtime requirements (what the end user must have installed, bundle size)
3. Toolchain footprint beyond `go` (npm and node count against a framework)
4. Native desktop widgets (native, framework-drawn, or a fallback)
5. Desktop plus web reach, measured as the extra code needed for both targets (less is better; N/A for desktop-only)
6. cgo-core tolerance (WASM can't use cgo; a server-side core is an accepted workaround)
7. Code clarity and complexity (line counts, longest function, judge's reading)
8. Testability (what can be tested headlessly, and with what tools)
9. Performance with large data (paging, virtualization, frame stalls)
10. Out-of-the-box style appeal (light and dark)
11. Native host access (identify the running user, including web-in-desktop mode)
12. Custom rendering (paint delegates, custom cells and item views; the lines involved)

## Known caveats

- A second variant pass (`apps/<id>-fit`) let each app replace the shared pager with a toolkit-suited data layer; rule changes and fixes for it (parallel fetches raised to 4 to 6, extra bench rounds, copy fixes) are in `results/DEVIATIONS.md`, along with fact-check corrections.
- Benchmark numbers reported by the implementer agents are invalid because the screen was locked while they ran. Only the harness runs count.
- GoTTH patched vendored shadcn-templ JS and an htmx scan for WebKit performance. This must be disclosed.
- Port 8090 was occupied on the host, so the harness serves web targets on 8097.
- Spec error: the Cogent CLI install path in the brief was wrong. Count it as an evaluator error, not framework friction.
