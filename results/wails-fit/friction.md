# Friction log: Wails v3.1.0

Session start: 14:23 (2026-11-08, NZDT). Entries are chronological.

### 14:24 Install the wails3 CLI
- Step: Bootstrap per brief section 6.3.
- Command/API: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0`
- Outcome: Worked, about 33 s including module downloads. `wails3 version` prints v3.1.0.
- Time lost: 0
- Category: toolchain
- Resolution: none needed.

### 14:25 wails3 init creates a nested directory
- Step: Create the project in `apps/wails`.
- Command/API: `wails3 init -n gguix-wails -t svelte -d apps/wails -mod gguix/apps/wails ...` (template name `svelte`, described as "Svelte + TypeScript + Vite")
- Outcome: `-d` is the parent directory: the project was created in `apps/wails/gguix-wails/`, not in `apps/wails/`. Also `-productname` and similar flags were accepted but `build/config.yml` still has the "My Product" placeholders (the generated Taskfile and plists use them). The generated `go.mod` says `go 1.26.0` and the Docker cross image uses golang:1.26, which is older than the core's `go 1.27`.
- Time lost: 3
- Category: api-confusion
- Resolution: moved the files up one level by hand, rewrote go.mod (go 1.27, require/replace for gguix/core), edited build/config.yml.

### 14:27 shadcn-svelte init is interactive and needs a preset
- Step: `npx shadcn-svelte@1.8.0 init` with base color neutral.
- Command/API: `npx shadcn-svelte@1.8.0 init --base-color neutral --css src/app.css ...`
- Outcome: With stdin closed or piped the command hangs or exits silently at an interactive "choose a preset" prompt (the `--base-color` flag alone is not enough in 1.8.0; presets are `nova, vega, maia, lyra, mira, luma`...). Later prompts (overwrite files, update CSS) also need a TTY. The first background attempt had to be killed.
- Time lost: 12
- Category: docs-gap
- Resolution: found the preset list by reading the CLI's dist; used `--preset vega` (Lucide icons, Inter, neutral, the classic look) and drove the remaining prompts with a small Python pty script that presses Enter. `add -y` for components needed the same pty trick. The generated `src/lib/utils.ts` imports a package named `cn` (added to devDependencies by the CLI) instead of clsx and tailwind-merge directly.

### 14:35 tanstack svelte-virtual declares a Svelte 4 peer dependency
- Step: Install `@tanstack/svelte-virtual@3.13.52` next to svelte 5.57.3.
- Command/API: `npm install --save-exact ... @tanstack/svelte-virtual@3.13.52`
- Outcome: Installed without an error; its peerDependency is `svelte ^4.2.20`. It works because it only uses `svelte/store`, but the Svelte 5 runes style is not supported (store syntax `$virtualizer`).
- Time lost: 0
- Category: docs-gap
- Resolution: used the store API; wrapped it in `VirtualScroller.svelte`.

### 17:00 One million rows are taller than a browser can scroll
- Step: Table (37 px rows) and cards (104 px) at `Total * rowHeight`, as the brief says.
- Command/API: tanstack virtualizer with `getTotalSize()`
- Outcome: Chrome and WebKit cap element height near 33.5 million px; 1M cards are 104 million px. Scrolling past the cap is impossible, and `scrollTop` precision cannot address single rows.
- Time lost: 40
- Category: framework-limitation
- Resolution: `scaled-scroll.ts` maps scrollTop to a virtual offset when the height exceeds 30 million px and remembers the exact offset of programmatic jumps. tanstack's own `scrollToIndex` read-back logic is bypassed (custom `observeElementOffset`, `observeElementRect`, `scrollToFn`).

### 17:02 Svelte effect loop in the virtual scroller
- Step: Bench run in headless Chrome showed `effect_update_depth_exceeded` and a 13 s cards scroll.
- Command/API: `$virtualizer.setOptions(...)` inside `$effect` (setOptions writes the store the effect subscribes to).
- Outcome: Infinite effect re-runs.
- Time lost: 10
- Category: api-confusion
- Resolution: use `get(virtualizer)` inside `untrack`. Cards scroll then took 3.4 s for 200 steps.

### 17:13 Notifications service stops the whole app outside a bundle
- Step: Run `bin/gguix-wails` directly.
- Command/API: `notifications.New()` registered as a service.
- Outcome: `error starting service ... notifications require a valid bundle identifier` and the app exits. The template's `run` task avoids this by running a `.dev.app` bundle.
- Time lost: 10
- Category: runtime-bug
- Resolution: register the notification service only when the executable path contains `.app/Contents/MacOS/`.

### 17:15 Desktop bench stalls when the screen is locked
- Step: `make bench` on the desktop.
- Command/API: `GGUIX_BENCH=1 wails3 task run`
- Outcome: The Mac was at the lock screen (screenshots black, no way to unlock without a password, which I must not enter). The WKWebView is occluded: `requestAnimationFrame` never fired and timers stalled. After replacing rAF with `setTimeout`, `ready` and `jump` printed and the `scroll` step (200 timer steps) never finished in 25 minutes. The same script completes in headless Chrome against the server build.
- Time lost: 60
- Category: framework-limitation
- Resolution: unresolved here; must be rerun on an unlocked desktop. Web bench numbers are in the summary.

### 17:25 Cross builds
- Step: `cross-windows` and `cross-linux`.
- Command/API: `wails3 task build GOOS=windows ARCH=amd64`, then `GOOS=linux`.
- Outcome: Windows: native Go cross compile, 12.2 MB, no Docker needed. Linux: (1) generated Dockerfile uses golang:1.26 but go.mod needs 1.27 (`GOTOOLCHAIN=local`); (2) after editing to golang:1.27-trixie: `gcc: unrecognized option -m64`, because the image is arm64 on Apple Silicon and linux/amd64 uses plain gcc; (3) building the image with `--platform linux/amd64` and running the template entrypoint directly produced `bin/gguix-wails-linux-amd64` (11.4 MB). The image build takes about 5 minutes the first time (several GB).
- Time lost: 35
- Category: toolchain
- Resolution: `make cross-linux` does attempt 3.

### 17:35 Package name on a case-insensitive file system
- Step: Rename `gguix-wails.app` to `GGUIX-wails.app`.
- Command/API: `rm -rf bin/GGUIX-wails.app && mv ...`
- Outcome: The rm deleted the fresh bundle (same path on APFS).
- Time lost: 3
- Category: build-error
- Resolution: rename through a temporary name; re-sign ad hoc.

### 17:40 Other notes
- Template files for iOS and Android (Go sources and Taskfiles) were removed; the generated `build/config.yml` product fields were edited by hand and `wails3 update build-assets` rerun.
- Port 8090 is held by unrelated processes on this host; tests use 8191.
- `wails3 init` prints no hint that `-d` is the parent directory.
- Wails has no reveal-in-Finder API and no preferences API (core helpers used).
- Additional dependencies beyond the brief: `jsdom` (needed by vitest component tests), `mode-watcher`, `svelte-sonner`, `@internationalized/date`, `@lucide/svelte`, `tailwind-variants`, `tw-animate-css`, `@fontsource-variable/inter`, `cn`, `svelte-check` (all from the template or the shadcn-svelte CLI), and `python3` for totalling test results.

## Summary

- Wall time to first window: about 17:13 on the session clock (desktop window loads and answers Page calls; first browser render of the web target about 17:05). The session clock includes a usage pause between roughly 14:40 and 16:55, so active time to first window was about 40 minutes.
- Wall time to feature complete: about 17:45 (same clock; about 1 hour 20 minutes of active work).
- Entries per category: toolchain 2, docs-gap 2, api-confusion 2, build-error 1, runtime-bug 1, framework-limitation 2, core-issue 0 (not counting the "other notes" entry).
- Three biggest pain points: (1) the 33.5 million pixel element height cap forced a custom scroll mapping; (2) the desktop bench could not complete on a locked screen, and the notification service kills the app outside a bundle; (3) cross-linux needed an amd64 Docker image on Apple Silicon plus a Go version fix in the generated Dockerfile, and shadcn-svelte's init needs a TTY and a preset.

Web bench (headless Chrome, 1,000,000 rows, 50 ms latency, server mode): ready 252 ms, jump 83 ms, scroll 3629 ms total (max gap 10.9 ms, 49 placeholder steps), cards_scroll switch 185 ms and 3356 ms total (0 placeholder steps), sort 312 ms, search 129 ms (49,898 rows), export 49,898 rows in 521 ms.


## Second pass (data layer)

### 22:15 Task skipped the rebuild after Go and frontend edits
- Step: first `wails3 task run` bench after adding `PageBatch` showed the old numbers and no new log line.
- Command/API: `GGUIX_BENCH=1 wails3 task run`
- Outcome: the run task reused the previous binary and bindings, so `frontend/bindings` had no `PageBatch`. `wails3 task build` regenerated the bindings and rebuilt both sides.
- Time lost: 5
- Category: toolchain
- Resolution: always run `wails3 task build` (or `make build`) before comparing bench numbers.
