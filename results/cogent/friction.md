# Friction log: Cogent Core

Chronological. Times are local wall clock on 2026-11-08.

### 14:24 Brief install command for the CLI does not work
- Step: Installing the `core` CLI as the brief section 6.2 says.
- Command/API: `go install cogentcore.org/core/cmd/core@v0.5.27`
- Outcome: `module cogentcore.org/core@v0.5.27 found, but does not contain package cogentcore.org/core/cmd/core`. In v0.5.27 the `main` package of the CLI lives in the module root (`cmd.go`), and `cmd/` is a library package.
- Time lost: 2
- Category: docs-gap
- Resolution: The framework's own install page says `go install cogentcore.org/core@main`; pinned form is `go install cogentcore.org/core@v0.5.27`. That worked (about 11 s, installs `~/go/bin/core`, 34 MB).

### 14:25 No bootstrap/template command in the CLI
- Step: Looking for the "framework's own recommended project bootstrap (its CLI template)".
- Command/API: `core help` lists setup, build, run, pack, install, generate, changed, pull, log, release, next-release, update. There is no `init`/`new`.
- Outcome: The documented bootstrap is the hello-world tutorial (`core.NewBody()`, widgets, `b.RunMainWindow()`) plus `go mod init`. I used exactly that, then `go get cogentcore.org/core@v0.5.27` and `go mod tidy`.
- Time lost: 1
- Category: docs-gap
- Resolution: Recorded what was run in README.md. `core setup` was not needed on this machine (Xcode CLT already present); not run.

### 14:27 wasm build needs go.sum entries for web-only dependencies
- Step: First `core build wasm` of the hello world.
- Command/API: `core build wasm -o /tmp/hbw`
- Outcome: `missing go.sum entry for module providing package github.com/hack-pad/hackpadfs`. `go get` from a darwin host only records darwin-reachable deps.
- Time lost: 2
- Category: build-error
- Resolution: `go mod tidy` (tidy considers all GOOS values) fixed it. After that `core build wasm` works (bundle 26.2 MB, gzip 5.9 MB).

### 14:28 `core build` has no way to pass Go build tags
- Step: Planning `web-remote` (`-tags remote`) and `cgoprobe` (`-tags cgoprobe`) with the CLI.
- Command/API: `core build -h` (only a `-debug` flag that adds the `debug` tag); `cmd/web/build.go` hardcodes `-tags mvmnosrc` for wasm.
- Outcome: The CLI cannot build the wasm target with extra tags. GOFLAGS `-tags` would be overridden by the CLI's explicit `-tags`.
- Time lost: 5
- Category: framework-limitation
- Resolution: Makefile puts a tiny `go` wrapper script first in PATH for the `core build wasm` step that appends the extra tag to the `-tags` argument. Recorded in capabilities.md.

### 14:45 core.Table cannot be backed by a Pager
- Step: Choosing the table widget (brief 6.2).
- Command/API: `core.Table`, `core.ListBase`, `core.Lister` (read in `core/table.go`, `core/list.go`).
- Outcome: Table binds to a reflected slice of structs, sorts that slice itself and builds columns from struct fields. Using it would need 1,000,000 materialized structs.
- Time lost: 30 (reading source; the docs only show small slices)
- Category: framework-limitation
- Resolution: Subclassed `core.ListBase` (the virtualization, scrollbar, selection and key navigation base of List and Table) with a zero-size `[]struct{}` of length Total and my own header, cells and column handles (`table.go`). Same idea for the cards view (`cards.go`).

### 15:00 No native menu bar, native file dialogs, OS notifications or edit menu
- Step: Menus, export dialog, notifications (brief 5.5, 5.6).
- Command/API: grep of the module for NSMenu, NSOpenPanel, notification APIs: none.
- Outcome: Menus are framework-drawn buttons in the window, file choosing uses the framework `FilePicker` dialog, the export-complete message is a snackbar, and the Edit menu sends synthetic key chords to the last focused text widget (TextField cut/copy/paste are unexported).
- Time lost: 20
- Category: framework-limitation
- Resolution: Implemented the fallbacks and recorded them as gaps.

### 15:10 Canvas uses normalized 0-1 coordinates
- Step: Drawing stars.
- Command/API: `core.Canvas.Draw`.
- Outcome: Shapes must be specified per widget size; a 60 x 12 dp canvas with 1/5 wide cells gives square stars. Worked after one attempt.
- Time lost: 5
- Category: api-confusion
- Resolution: Star geometry is computed in unit cells and scaled by 1/5 in x.

### 16:50 go test / offscreen window renders the whole app
- Step: Visual check and tests.
- Command/API: `Body.RenderImage(fn...)` (offscreen driver, used automatically in test binaries).
- Outcome: Works well: scenarios T1 to T12 run against a real widget tree and I could save PNGs. Needed `AsyncLock`/`AsyncUnlock` (via `a.async`) around every access from the test goroutine.
- Time lost: 0
- Category: docs-gap
- Resolution: none needed.

### 17:03 Real desktop window cannot open while the Mac is locked
- Step: First run of the desktop binary.
- Command/API: `./bin/gguix-cogent`; `screencapture`.
- Outcome: Panic `index out of range [0] with length 0` in `system/driver/base.(*AppMulti).Screen` while the display was asleep, later the process ran but the screen was at the login window (a screenshot showed the lock screen). Passwords must not be typed by me, so no interactive desktop run could be shown. Verification is by offscreen tests and bench.
- Time lost: 15
- Category: toolchain
- Resolution: unresolved; `make bench` and `make run` are untested with a visible window. The bench script was run through the offscreen window (test `TestBenchOffscreen`, opt-in) and in the browser.

### 17:05 Framework crash in Text restyle
- Step: Bench run, sort while scrolled.
- Command/API: `Widget.SetSelected(false)` inside a cell Updater (`Restyle` -> `Text` styler -> `shaped.Lines.UpdateStyle`).
- Outcome: nil pointer dereference in `shaped.RunBase.SetFromStyle` when a restyle ran on a Text whose content had just changed.
- Time lost: 10
- Category: runtime-bug
- Resolution: Removed the `SetSelected` call (my cells do not use that state).

### 17:08 core pack bugs on desktop
- Step: `make package`.
- Command/API: `core pack`, `core pack -o bin/pack`.
- Outcome: With any `-o` other than `.` it fails with `cp: <name>: No such file or directory` (it copies `filepath.Base(Output)` from the current directory). With `-o .` the `.app` is built, then it stops because `pip install dmgbuild` is missing (not installed on purpose). The executable inside the bundle is named `GGUIX-cogent.app`.
- Time lost: 12
- Category: build-error
- Resolution: Makefile runs `core pack -o .` ignoring the dmg error and moves the bundle to `bin/GGUIX-cogent.app`. Not launched (locked display).

### 17:12 core build -o semantics
- Step: `make build` output path.
- Command/API: `core build -o bin/x`.
- Outcome: `-o` is a directory; the binary is written to `<o>/<base of o>`.
- Time lost: 3
- Category: docs-gap
- Resolution: Makefile builds into `bin/_b/gguix-cogent/` and moves the file to `bin/gguix-cogent`.

### 17:15 Port 8090 busy and shared browser console
- Step: `make web-remote`.
- Command/API: core server `-addr :8090`.
- Outcome: Another process on this machine already listened on 8090 and the browser pane's console held old messages from another app. I tested on 8091 (`make web-remote PORT=8091`) and used a fresh browser tab.
- Time lost: 5
- Category: toolchain
- Resolution: The Makefile `PORT` variable can be overridden.

### 17:20 Status bar did not refresh in wasm
- Step: Browser check of `web-remote`.
- Command/API: `Text.SetText` followed by `NeedsRender`.
- Outcome: The status text stayed "Loading..." because the text width changed and no layout was requested. Fixed with `NeedsLayout` when the text changes (not re-verified in the browser after the fix).
- Time lost: 5
- Category: api-confusion
- Resolution: see above.

### 17:35 Cross builds
- Step: `make cross-windows`, `make cross-linux` (brief 7).
- Command/API: `core build -t windows/amd64`, `core build -t linux/amd64`, then a Docker golang:1.25 container with `GOTOOLCHAIN=go1.27.3` and `go build`.
- Outcome: The CLI route fails for both (cgo needs a target C toolchain; the docs say cross-compiling between desktop platforms is not yet supported). The Docker linux/amd64 build produced `bin/linux/gguix-cogent` (41.6 MB) in 6 m 45 s. Windows via Docker was not attempted.
- Time lost: 25
- Category: framework-limitation
- Resolution: Linux PASS via Docker, Windows recorded as FAIL.

## Summary

- Wall time to first window: about 2 h 30 min after starting (14:24 start; the first offscreen render of the full UI was around 16:55, with a pause in between for a usage reset). No window was ever shown on the real screen because the Mac was locked.
- Wall time to feature complete (all make targets, tests passing, bench metrics on offscreen and web-remote): about 3 h 15 min from the start (17:35), same caveats.
- Entries per category: docs-gap 4, api-confusion 2, toolchain 2, build-error 2, runtime-bug 1, framework-limitation 4, core-issue 0.
- Three biggest pain points:
  1. The table and list widgets only bind to reflected slices; building a Pager-backed table meant subclassing `core.ListBase` and re-implementing the header and column handling (about an hour of source reading).
  2. Missing native pieces: no menu bar, file dialogs, notifications or edit menu, so everything is framework-drawn or a fallback.
  3. The `core` CLI: no way to pass build tags, `core pack` path bugs and a dmg step that needs pip, and no cross compilation for desktop targets.

### Fix pass 2026-11-09: minimum window size
- Step: fix pass gap 1 (min window 1024 x 640).
- Command/API: type-assert `system.Window` to `*desktop.Window`, call `Glw.SetSizeLimits(1024, 640, glfw.DontCare, glfw.DontCare)`; glfw promoted from indirect to direct in go.mod.
- Outcome: builds, tests pass. The earlier "no documented API" note was wrong: the field is exported (cogentcore.org/core@v0.5.27/system/driver/desktop/window.go:27-31). An automated test is not possible: glfw on macOS must run on the process main thread, which `go test` does not give (crash in Cocoa), and programmatic SetSize on a hidden window is not clamped by the limits (prints 300 200), so only user drag-resizing exercises it.
- Time lost: 15
- Category: docs-gap
- Resolution: implemented; manual drag check pending.
