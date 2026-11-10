# Deviations and evaluator errors

## Evaluator (spec) errors
- The Cogent brief gave a CLI install path (`.../cmd/core`) that does not exist in v0.5.27; the correct command is `go install cogentcore.org/core@v0.5.27`. Not counted as framework friction.
- `tools/loc` computes dual-target overhead from desktop and web marked code; MIQT is desktop-only, so its value (all files `_desktop.go`) is reported as N/A.

## Harness changes after the first full run
- Web port moved from 8090 to 8097 (`GGUIX_WEB_PORT`) because unrelated host processes held 8090. Make variables `PORT`, `SERVE_PORT`, `WEB_PORT` and env `PORT`, `GOTTH_PORT` are passed.
- `bench/web.mjs` now splits console messages on newlines: Go WASM (Cogent) batches several `GGUIX_METRIC` lines into one console message, which the first run rejected (both Cogent web steps timed out).
- GoTTH reads `GOTTH_PORT`, not `PORT`; its first web run failed with "port 8097 never opened". Cogent and GoTTH web steps were rerun with `--only web` and merged into metrics.json.

## Excluded data
- Benchmark numbers in the implementers' own notes are invalid (the screen was locked during their sessions). Only harness `metrics.json` counts.

## Implementation changes disclosed
- GoTTH patched vendored shadcn-templ `components/baseui/lifecycle.js` and disabled an htmx hx-on XPath scan for WebKit performance.

## Screenshots
- Captured by preseeding prefs (theme, view, selectedId=1) per state; server-backed web apps (Wails, GoTTH) restarted per state. Observed app behaviour, not capture faults: MIQT table does not highlight the selected row; Cogent web status bar stays "Loading..." after data loads; Fyne web shows a replacement glyph in the ID header.

## Judging
- First judge pass (results/judge/style.md, scores.md): blinding failed for style because desktop window titles named the framework (e.g. "GGUIX - MIQT"); the judge identified A=wails, B=fyne, C=cogent, D=gotth, E=miqt. Its non-style scores stand (code and metrics are inherently not blind).
- Style was re-judged blind (results/judge/style_blind.md) by a fresh agent seeing only shots_blind/: desktop shots with the top 64 px (title bar) removed, relabelled P to T (mapping in label_map2.json). In-app menu bars drawn by some apps remain visible.

## Native-fit variant pass
- After the fixed-core results, a second pass gave each app a data layer suited to its toolkit (`briefs/FIT_ADDENDUM.md`), keeping the core service, latency, fetch limits (200 rows per call, 10,000 rows in memory, no preload), bench script and behaviour fixed. Each agent (Sonnet, same as the first pass) started from its own first-pass app in ~/src/gguix-work/<id>-fit. Variants are stored as apps/<id>-fit and results/<id>-fit. Motivation: first-pass list performance for MIQT and GoTTH (and Fyne memory) looked implementation-limited, so a toolkit-idiomatic adapter tests each toolkit's ceiling more fairly.
- Variant measurement fixes: (1) the copy into apps/<id>-fit excluded `bin/`, which removed GoTTH's Tailwind binary; `make deps` then failed (GitHub 504), so the new CSS was not built and every desktop jump timed out. The same Tailwind binary was copied from the first-pass app and GoTTH-fit was re-measured. (2) `measure.py` looked for `bin/gguix-<id>` using the `-fit` id; it now strips the suffix, and sizes and idle_rss were re-measured for all variants.
- Variant rule notes: several variants raised concurrent service calls from 2 to 4 to 6 (not limited by the addendum); GoTTH-fit and MIQT-fit ran more than the suggested three bench rounds (reported as measurement and debugging, not tuning).

## Public fact-check (subagent, public sources only)
- Wrong: "Wails has no reveal-in-Finder API". Wails v3.1.0 has `EnvironmentManager.OpenFileManager(path, selectFile)`. Reclassified as an implementation gap.
- Corrected cause: Cogent Windows cross failure is the missing Windows C toolchain when cross compiling from macOS (cgo files drop out), not a lack of Windows support in wgpuglfw.
- Clarified: Cogent web used Go WASM; v0.5 also has a GoScript `js` platform that was not tested. Browser element-height cap (about 33.5M px) holds for Chrome and WebKit; Firefox is lower (not tested). Fyne notifications use osascript only for unbundled binaries.
- MIQT README lists older Windows images; the image actually selected by miqt-docker v0.15.0 here was win64-cross-go1.26-qt6.5-static (from the build log).
- Added after a user query: MIQT v0.15.0 includes `qt6/webengine` and `qt6/webchannel` bindings (verified in the module source and README). "No web target" means no browser target only.

## Report audit against raw data (subagent, findings re-checked by hand)
Corrected in REPORT.md: GoTTH-fit card-switch figures were desktop values in a web sentence; GoTTH JS size (about 600 lines, not 1k); Wails bundle 12 MB (not 11); Wails-fit other-file changes +54/-48 (not +94/-88); web tables are single runs; Cogent-fit web export regression (4.3 to 104.5 ms) added; run-to-run spread was not tight for MIQT and Cogent. Superlatives narrowed to the metric they rest on ("smoothest" became lowest frame gap; Cogent "fastest renderer"; Fyne "most UI shared"; MIQT "best large-list performance"). Causal claims softened where the evidence was thin (MIQT "adapter, not Qt", which also involved more parallel fetches; GoTTH desktop; Fyne wasm cause; Fyne memory). EVALUATION_BRIEF.md: blinding description, the "verified" core claim, and a variant-pass caveat.
- Capability fact-check, pass 2 (module source): corrections to REPORT.md, EVALUATION_BRIEF.md, and an errata list for agent notes in results/ERRATA.md.

## Cross build reruns (2026-11-09)

Four cross builds failed in the 09:45 to 10:41 batch: fyne linux, fyne-fit linux and windows, wails-fit linux. Docker Desktop restarted during that window (confirmed by the user). All four passed on an unchanged re-run once Docker was stable. The failures are recorded as environment errors, not framework results.

## Web benches re-run at device scale factor 2 (2026-11-09)

bench/web.mjs created its Playwright context without deviceScaleFactor, so every earlier web bench ran at 1x and the bench window looked blurry on a Retina display (most visibly Fyne's canvas text). A 1x run also understates the work for canvas-rendering apps (Fyne, Cogent), which must fill four times as many pixels at 2x. web.mjs now defaults to deviceScaleFactor 2 (GGUIX_DPR overrides). All web benches were re-run at 2x; the 1x values are kept in results/archive/web-dpr1/. Only the 2x values are reported.

## Fyne web transfer size (2026-11-09)

The Fyne web targets are served with `python3 -m http.server`, which never compresses, while Cogent's core server does. So Fyne's measured initial transfer (39.6 MB) is uncompressed and not like-for-like. The report quotes the gzip size of the wasm (about 12 MB) for comparison. Not re-measured.

## Variant Makefiles served the base app (found 2026-11-09)

The -fit and -fitcap Makefiles for Cogent and Fyne were copied from the base apps with hard-coded paths left unchanged (`../apps/cogent/...`, `../apps/fyne/...`). Effect: every Cogent variant web target (web, web-remote, bench-web) and both Fyne variants' web-remote target served the BASE app's build, so those web benches measured the base app. Fixed to each app's own path; the web benches for fyne-fit, fyne-fitcap, cogent-fit and cogent-fitcap were re-run. Fyne's pure-WASM web target was also moved from `python3 -m http.server` (no compression) to the core server (gzip), and fyne's web benches re-run, so transfer sizes are like-for-like with Cogent. Desktop benches were unaffected (they build in each app's own directory).

## Harness audit fixes (2026-11-09)
- A harness audit found measurement-validity, fairness and provenance problems (benches sampled through make and wrapper processes, unstripped Fyne builds, no gzip on the Wails and GoTTH web servers, wasm "ready" timed from Go init instead of navigation start, stale-listener and stale-binary risks, among others). The fixes are listed item by item in `results/HARNESS_CHANGES.md`.
- All results produced before the clean re-run that follows these fixes are superseded and must not be cited.

## Clean re-run (2026-11-09, 19:30 to 23:08 local)
- All 17 apps were re-measured sequentially after the harness audit fixes, 5 desktop runs each, with the screen awake and unlocked. Every step completed with no step errors. The run was paused once between fyne-fitcap and gotth (no step in progress) for a window-only video capture test, then resumed.
- `measure.py` merges into an existing metrics.json, so a key from a step that no longer runs survives. miqt-fit and miqt-fitcap kept a stale `web` entry ("port 8097 never opened") from before audit fix B8 skipped web for desktop-only MIQT apps. That stale entry was deleted; no other stale keys were found (every stored step's start time is after the run's provenance timestamp).

### Cogent package step (clean re-run)

- `make package` fails for `cogent`, `cogent-fit` and `cogent-fitcap` (rc=2 in `metrics.json` `sizes.package`). `core pack` builds `GGUIX-cogent.app`, then runs `pip install dmgbuild` to make a DMG; this machine has `pip3` (Homebrew) but no `pip` on PATH, so the step stops. Before C4 the Makefile ignored this failure, and the old 27.1 MB bundle size came from a stale `.app`.
- The `.app` that `core pack` leaves before the DMG step was measured by hand: 27,112 KB (26.5 MB) with `du -sk`, on 2026-11-10. REPORT.md uses that figure. It is the only bundle size not taken from `metrics.json`.
- For the report: Cogent's packaging needs Python's `pip` on PATH, which is an extra provisioning step the other stacks do not have.

### Wails scroll pacing isolation test (2026-11-10)

Question: is the Wails scroll gap rise (22 ms before the audit, 91 ms in the clean re-run) caused by the B4 pacing fix alone?

Method: in `apps/wails/frontend/src/bench/bench.ts` `scrollSteps`, the absolute deadline was replaced by the pre-B4 `await sleep(16)` after each step. Everything else as in the clean re-run: release `make build`, `bin/gguix-wails` launched directly with `GGUIX_BENCH=1`, 5 runs. The file was then restored and the app rebuilt; no results file was changed by this test.

| Table scroll | Max gap (ms), runs 1 to 5 | Median | Run length (ms) | Blank-row steps |
|---|---|---|---|---|
| Pre-B4 pacing (this test) | 28, 9, 6, 10, 6 | 9 | 4,002 to 4,244 | 0 to 22 |
| B4 pacing (clean re-run) | 91, 74, 108, 92, 68 | 91 | about 3,215 | 58 to 65 |

Result: with the old pacing and everything else current, the gap drops back to the old range. The pacing change alone explains the difference. The run lengths show why: the 200 steps are scheduled for 3.2 s, but with relative pacing they take 4.0 to 4.2 s, so the old test scrolled about 25% slower and absorbed slow frames instead of reporting them. The Wails app did not get worse; the old harness asked less of it and could not see the gaps.

## Chrome vs WebKit engine test (2026-11-10)

Question: is the WebView stutter on the macOS desktop (wails-fitcap, gotth-fitcap) caused by the engine rather than the HTML? Same web build, served by the app's own `make bench-web`, driven by `bench/web.mjs` in Chrome 156 and in Playwright's WebKit (`GGUIX_ENGINE=webkit`, headed). 3 runs per engine, fresh server per session, engine order alternated per round. Script and raw JSON: session scratchpad `engine-test.sh`, `eng-*.json` (not in the repo).

Evaluator errors on the way, both fixed before the valid run:
1. The first attempt reused one server for all six sessions of an app. Later sessions ran on warm core caches (Wails sort 269 ms on the first run, about 74 ms after) and inherited GoTTH's server-side UI state (search failed). Discarded.
2. `bench/web.mjs` never cleared its timeout timer, so Node stayed alive for the full timeout (300 s) after every session. This cost wall time only, never measurements, including in earlier harness web runs. Fixed: the script exits once its JSON is written.

Valid results, table scroll worst frame gap per run (ms):

| App | Chrome | WebKit |
|---|---|---|
| wails | 24, 26, 42 (p95 median 14; blank steps median 39) | 141, 164, 153 (p95 median 146; blank steps median 95) |
| wails-fitcap | 20, 30, 26 (p95 median 14; blank 0) | 304, 224, 231 (p95 median 162; blank 68) |

Other Wails events stay in the same band across engines (jump 80 to 97 ms, sort 267 to 285 ms, cards switch 92 to 201 ms).

Conclusion for Wails: the same HTML scrolls smoothly in Chrome and stutters in WebKit, reproducing the desktop WKWebView result. The engine, not the markup, explains the difference. Playwright's WebKit is a WebKit build, not the system WKWebView, so this supports the desktop finding rather than measuring it.

GoTTH: inconclusive, not used. Every GoTTH session in both engines logged "search did not finish", and the scroll results (worst gap 2 to 13 ms, 0 blank steps) do not match the harness's own Chrome run of the same app (152 ms, 162 blank steps), so this setup does not reproduce the harness conditions for GoTTH. Cause not investigated.

## Screenshot identity override (2026-11-10)

`core/host/native.go` reads `GGUIX_SHOT_IDENTITY=user@host` and, when set, shows that instead of the real username and hostname. Used only for the article screenshots (`demo@workstation`), so they don't show the author's machine. Unset, behavior is unchanged; no timing or function result depends on it. The five desktop binaries were rebuilt with `make build` after the change.
