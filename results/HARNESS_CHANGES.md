# Harness changes from the 2026-11-09 audit

Every change made for the harness audit, with the files touched and the audit item it answers. All results measured before the clean re-run are superseded (see `results/DEVIATIONS.md`, "Harness audit fixes (2026-11-09)").

## A. Measurement validity

**A1. Bench launches the built binary directly** (`bench/measure.py`, `bench()`).
- `make build` runs once before the runs, outside the sampling (log `raw/bench-build.log`). If it fails or the binary is missing, the step records an error instead of benching an old binary.
- Each run launches `bin/gguix-<base>` directly with the same env idle_rss uses plus `GGUIX_BENCH=1` and `GGUIX_BENCH_EXPORT`, in its own session, and samples only that process tree (plus new WebKit processes, as before). Peak memory no longer includes make, `wails3`/`task`, shells or `grep`.
- Wails: `wails3 task run` (build/darwin/Taskfile.yml `run`) copies `bin/gguix-wails` into `bin/gguix-wails.dev.app` with `Info.dev.plist`, re-signs it and runs `Contents/MacOS/gguix-wails`. The harness now runs `bin/gguix-wails` itself, the same binary, as idle_rss already did.
- stdout and stderr are still captured to `raw/bench-run<N>.log` and parsed for `GGUIX_METRIC` lines.
- `metrics.json` `bench.launch` names the launched binary.

**A2. Wails `bench` depends on `build`** (`apps/wails/Makefile`, `apps/wails-fit/Makefile`, `apps/wails-fitcap/Makefile`): `bench: build`.

**A3. Cross builds use the app's own directory.**
- `apps/gotth/desktop/cross.sh`, `apps/gotth-fit/desktop/cross.sh`, `apps/gotth-fitcap/desktop/cross.sh`: Docker `-w "/src/apps/$(basename "$PWD")"` instead of the hard-coded `/src/apps/gotth` (the script `cd`s to the app dir first). Before this, the fit variants cross-built the base app.
- `apps/gotth-glaze/desktop/cross.sh` uses no Docker (plain `CGO_ENABLED=0` build in its own dir) and `apps/miqt-gotth` has no cross.sh (its cross targets report UNSUPPORTED); both were already correct.
- `apps/cogent-fitcap/Makefile` cross-windows and cross-linux Docker fallbacks pointed at `/w/apps/cogent-fit`; now `/w/apps/cogent-fitcap`.

**A4. Plain `go build` timing with generated sources** (`bench/measure.py`, `plain_build()`). After `make clean`, Wails apps run `wails3 task common:build:frontend` (bindings and `frontend/dist`) and GoTTH family apps (gotth, -fit, -fitcap, -glaze, miqt-gotth) run `make gen` (templ and Tailwind), untimed. Then `go build ./...` is timed. The record gets `gen` (command, rc, secs) and `plain_build_note` saying generated sources were present. If generation fails, the status is `n/a: needs generated sources` instead of a failed time.

## B. Fairness

**B1. Release builds.**
- Fyne (`apps/fyne`, `apps/fyne-fit`, `apps/fyne-fitcap` Makefiles): desktop `go build -tags release -trimpath -ldflags "-s -w"`; `fyne package --release` for the darwin `.app` and for the wasm build (`web`, `web-remote`, `bench-web`). Checked in fyne tools v1.7.4 (`cmd/fyne/internal/commands/build.go`: `--release` adds `-trimpath`, `-s -w` and the `release` tag; `package-wasm.go`: `webgl-debug.js` is written only when `!release`, so it is no longer shipped).
- GoTTH family: `-trimpath -ldflags "-s -w"` on the desktop and `cmd/web` builds (`apps/gotth` also on `build-native`); gotth-glaze keeps `CGO_ENABLED=0`.
- MIQT (`miqt`, `miqt-fit`, `miqt-fitcap`, `miqt-gotth`): already `-s -w`; `-trimpath` added. miqt-gotth's `cmd/web` build also gets `-trimpath -ldflags "-s -w"`.
- Cogent: confirmed, no change. `core build` (cogentcore.org/core v0.5.27 `cmd/build.go`, `cmd/config/config.go`) adds `-trimpath` (default true except for `core run`) and `-s -w` unless `-debug`.
- Wails: confirmed, no change. `build/darwin/Taskfile.yml` production build uses `-tags production -trimpath -buildvcs=false -ldflags="-w -s"`.

**B2. gzip on every web server.** A stdlib copy of `core/server/server.go` `gzipMiddleware` (plus an `Unwrap` for `http.ResponseController`, a bypass for `text/event-stream` and for upgrade requests, and a per-app skip function):
- Wails: new `gzip_web.go` (`//go:build server`) in `apps/wails`, `apps/wails-fit`, `apps/wails-fitcap`; `main_web.go` `assetMiddleware` wraps its handler with it. The streamed CSV `/export/<id>` is not compressed (as core skips `/api/export`).
- GoTTH family: new `cmd/web/gzip.go` in gotth, gotth-fit, gotth-fitcap, gotth-glaze, miqt-gotth; `cmd/web/main.go` wraps `srv.Handler()`. `/export` (CSV) and `/export/*` (SSE progress) are not compressed.
- Fyne and Cogent already serve through `core/server`, which compresses.

**B3. One definition of web "ready"** (harness change in core, see D3).
- New `core/metrics_js.go` (`//go:build js`) and `core/metrics_native.go` (`//go:build !js`) define `core.SinceStart()`. Natively it is `time.Since(ProcessStart)` (unchanged). In js/wasm it is `performance.now()`, the time since `performance.timeOrigin` (navigation start), read through `syscall/js`.
- `core/bench/bench.go` emits `ready` from `core.SinceStart()` instead of `time.Since(core.ProcessStart)`. This affects the Go wasm apps (Fyne, Cogent web) only; before, their "ready" excluded the wasm download and compile.
- Confirmed: Wails `frontend/src/bench/bench.ts` and GoTTH `static/bench/bench.js` already use `performance.now()` for web "ready" (navigation start); desktop still uses the process start.

**B4. Wails `scrollSteps` on an absolute schedule** (`frontend/src/bench/bench.ts` in wails, wails-fit, wails-fitcap): step k waits until `due = start + (k+1)*16` (no sleep when late, only a Svelte `tick()`), as in `core/bench` and GoTTH `bench.js`. Before, each step slept a fixed 16 ms after its own work, so slow steps stretched the run.

**B5. Web renderer memory** (`bench/web.mjs`). Each 250 ms sample asks the browser CDP session for `SystemInfo.getProcessInfo`, takes the pids of type `renderer` and `GPU`, and sums their `ps` RSS. New output fields: `peak_renderer_rss_mb` (covers wasm linear memory), `renderer_pids`, `peak_gpu_rss_mb`, `memory_sampling_error`. `peak_js_heap_mb` is kept and labelled as JS heap only (CDP `JSHeapUsedSize`, excludes wasm memory). `performance.measureUserAgentSpecificMemory` was not used: it needs a cross-origin-isolated page, which none of the apps serve.

**B6. Bundle size per web target** (`bench/measure.py`, `web()`): `bundle_dir_kb` and `wasm_files` are recorded inside each target's result right after that target's build and server start, not once at the end (when `web-remote` had overwritten `bench-web`'s `bin/web`).

**B7. Incremental UI marker** (`bench/measure.py`, `incremental()`): uses `id.split('-')[0]` for the wails and gotth checks, and miqt-gotth also gets `views/marker.templ`. Before, only the exact ids `wails` and `gotth` measured a UI marker, so variants, gotth-glaze and miqt-gotth had no UI incremental build.

**B8. Web skip** (`bench/measure.py`, `main()`): web is skipped for every id whose prefix is `miqt` except `miqt-gotth`, which has GoTTH web targets. Before, only the exact id `miqt` was skipped.

**B9. LOC counts Wails Taskfiles** (`tools/loc/main.go`): the top-level `build/` dir is walked, but only files that count as build scripts (`Taskfile.yml`, Makefiles) are counted there; the generated templates and assets under `build/` (plists, nsis, nfpm, Dockerfile, icons, `config.yml`) are skipped. Other `build` dirs are still skipped.

## C. Provenance and robustness

**C1. Provenance in metrics.json** (`bench/measure.py`, `bench/web.mjs`).
- `provenance`: time recorded, sha256 tree hash of `apps/<id>` and of `core/` (sorted relative paths plus file sha256, excluding `bin`, `node_modules`, `dist`, `.git`, `.task`), the inherited `GGUIX_*` env, the `GGUIX_*` env the harness passes, and the names scrubbed.
- `artifacts.<step>`: path, sha256, size and mtime of the exercised binary for cold_build, incremental, sizes, idle_rss and bench (sizes also gets a tree hash of the `.app`; bench also records its build rc and time).
- `web.<target>.provenance`: served dir (`bin/web`, `frontend/dist` or `static`), sha256 of `index.html` and of every `.wasm`/`.js` file in it, the server binary artifact for Wails and GoTTH-family, and start/end timestamps.
- `web.mjs` prints `chrome_version`, `dpr` and `viewport`.
- `step_times.<step>`: start and end timestamps (UTC ISO) for every step.

**C2. Env scrub** (`bench/measure.py`, `env()`): every inherited `GGUIX_*` var is dropped except `GGUIX_DPR`, which defaults to `2`; the harness then sets what it means to (`GGUIX_PREFS_DIR`, and per step `GGUIX_BENCH`, `GGUIX_BENCH_EXPORT`). The node web driver now gets the same env.

**C3. Web port checks.**
- `bench/measure.py` `web()`: fails a target if anything listens on the port before start (`lsof -iTCP:<port> -sTCP:LISTEN`); after the port opens, checks that every listener pid is in the server's process group; fails if the server is ready in under 0.3 s.
- GoTTH family `cmd/web/main.go` (gotth, gotth-fit, gotth-fitcap, gotth-glaze, miqt-gotth): when `GOTTH_PORT` or `PORT` is set, the server binds only that port (it may still try `[::1]` on the same port) and exits with "port N is busy" instead of moving to the next port.

**C4. Cogent package and stray outputs.**
- `apps/cogent`, `apps/cogent-fit`, `apps/cogent-fitcap` Makefiles: `package` removes a stale root `GGUIX-cogent.app` first and no longer ignores a `core pack` failure (the `-` prefix is gone); it removes the root binaries `cogent` and `$(notdir $(CURDIR))`. `clean` also removes the root `.app` and those binaries.
- Deleted stale build outputs (checked with `file` as Mach-O or PE executables / a bundle, not sources): `apps/cogent-fitcap/GGUIX-cogent.app`, `apps/cogent-fitcap/cogent`, `apps/cogent-fitcap/cogent-fitcap`, `apps/miqt/miqt`, `apps/miqt/miqt.exe`, `apps/miqt-fit/miqt`, `apps/miqt-fit/miqt.exe`, and also `apps/miqt-fitcap/miqt` (same kind of stray output). The MIQT `clean` targets now remove `miqt` and `miqt.exe`.

**C5. Wails `make test`** (wails, wails-fit, wails-fitcap Makefiles): removes `bin/test-go.log`, `bin/test-vitest.json` and `frontend/e2e/results.json` at the start, and before scoring fails with a `GGUIX_TESTS passed=0 failed=1` line if any is missing or empty.

**C6. Cosmetic**: the line-1 comment in `apps/fyne-fit/Makefile` and `apps/fyne-fitcap/Makefile` now names its own dir.

## D3. Core change (harness, not app)

`core/metrics_js.go`, `core/metrics_native.go` (new) and the `ready` line in `core/bench/bench.go` (B3) are bench timing only. No change to data, latency, the pager or any other core behaviour. They count as a harness change, not an app change.

## E. Addenda

**E1. MyGo** (`bench/measure.py`, 2026-11-14): id prefix `mygo` gets the Wails treatment where the structure is the same: the shadcn-svelte vendor dir is excluded from LOC, `frontend/src/lib/marker.ts` is timed as the UI marker, and before the plain `go build` the generated client and `frontend/dist` are produced untimed (`mygo generate`, then `npm run build`). Web measurement is skipped, since MyGo has no browser or server target.

**E2. MyGo native UI** (id `mygoui`, `bench/measure.py`): measured like the Fyne and Cogent apps (Go marker only; web from `bin/web`), and like them it also gets the `web-remote` web target.

**E3. Browser frame probe** (`bench/web.mjs`, new `bench/web_frames.py`, 2026-11-14): MyGo's browser build reported a 0.2 ms worst scroll gap from the in-app probe, shorter than one frame, so that probe does not see its drawing. `web.mjs` now also records `requestAnimationFrame` timestamps and stamps every `GGUIX_METRIC` line with `performance.now()`, and reports `browser_frames` (frame count, max and p95 gap) for the `scroll` and `cards_scroll` windows. `web_frames.py` runs only this, for the Go WASM apps (mygoui, fyne, cogent; `bench-web` and `web-remote`), and writes `results/<id>/web_frames.json`; `metrics.json` and the article's published web numbers are untouched. Single run, Chrome, DPR 2, like the article's web runs.

**E4. Custom-rendering line counts** (`results/mygo/loc_args.json`, `results/mygoui/loc_args.json`): added so `tools/loc` counts these apps' custom-rendering files like the others (mygo: the same four Svelte files as Wails; mygoui: `cells.go`). mygoui's card layout is inside `view.go` (`card`, about 38 lines) and is not in the count. Only the `loc` step was re-run.

**E5. go-gui** (id `gogui`, `bench/measure.py`, 2026-11-14): measured like Fyne, Cogent and MyGo native UI (Go marker only; web from `bin/web`), including `web-remote`. Browser frame gaps come from the E3 probe inside the normal web step. Style: blind passes `results/judge-addenda2/` (ten apps) and `results/judge-addenda3/` (eleven, with `gogui-fit`, used in the addendum). `gogui-fit` measured the same way.
