# Friction log: Fyne

Times are wall clock (NZDT, 2026-11-08). The session was paused for about 2.5 hours between 14:55 and 17:00 for usage limits; that gap is not counted anywhere below.

### 14:24 Bootstrap
- Step: install CLI, create module.
- Command/API: `go install fyne.io/tools/cmd/fyne@v1.7.4`; `go mod init gguix/apps/fyne`; `go get fyne.io/fyne/v2@v2.8.2`
- Outcome: CLI installed in 13 s. `go build` then failed with many "missing go.sum entry" errors because `go get` alone does not record the transitive modules.
- Time lost: 2
- Category: toolchain
- Resolution: `go mod tidy`.

### 14:28 Reading the Table API
- Step: designing the table.
- Command/API: `widget.Table` source in the module cache.
- Outcome: no getter for the scroll offset or the visible row range, no per-row selection (only a selected cell), `OnHighlighted` also fires on mouse hover (so it cannot drive keyboard selection), header cells are plain widgets (sorting needs a custom tappable header).
- Time lost: 15
- Category: framework-limitation
- Resolution: scroll offset read with reflect + unsafe from the private `offset` field (fallback to the last offset the app set); keyboard Up/Down by subclassing the table and overriding `TypedKey`; row highlight drawn by the app's own cell using the theme's selection color, and the table's own single-cell marker removed with `Unselect` in `OnSelected`.

### 14:30 Custom cells
- Step: stock bar, badge, stars.
- Command/API: `canvas.Rectangle` (CornerRadius), `canvas.ArbitraryPolygon` (new in 2.8), `canvas.Text`.
- Outcome: works first time, including a concave star polygon and a half star as a second polygon. Table has one `CreateCell` for all columns, so a single multi-purpose cell widget is used.
- Time lost: 0
- Category: docs-gap (ArbitraryPolygon has no docs besides the source comment; limit 16 vertices)
- Resolution: n/a

### 14:36 Nil pointer panic at first run
- Step: first launch.
- Command/API: `Pager.SetQuery` called from a widget callback while the UI was still being built.
- Outcome: `Pager.notify` is called synchronously inside `SetQuery`; `fyne.Do` on the main thread runs inline in Fyne 2.8, so the refresh ran before the status bar existed.
- Time lost: 3
- Category: api-confusion
- Resolution: suppress flag during construction. All UI-thread hops go through one `u.do` helper (fyne.Do) so tests can queue them.

### 14:40 Test driver
- Step: writing the T1 to T12 tests.
- Command/API: `test.LaidOutObjects`
- Outcome: it calls `CreateRenderer` again, so it never sees the live cells of a table. Needed my own walker using `test.WidgetRenderer` (the cached renderer).
- Time lost: 10
- Category: api-confusion
- Resolution: custom `walk` helper in harness_test.go.

### 14:55 Screenshots
- Step: checking the look.
- Command/API: `screencapture`
- Outcome: "could not create image from display" (no screen recording permission at that time). Used `Canvas().Capture()` from the test driver (software renderer) for visual checks.
- Time lost: 5
- Category: toolchain
- Resolution: later permitted by the coordinator.

### 17:02 Packaging
- Step: `make package`.
- Command/API: `fyne package --target darwin --name GGUIX-fyne --app-id io.gguix.fyne --icon Icon.png --source-dir .`
- Outcome: worked (24 s). The tool rewrites FyneApp.toml (bumps Build) and writes the .app into the current directory, so the Makefile moves it into bin/. `fyne package` needs an icon file; none is generated.
- Time lost: 2
- Category: docs-gap
- Resolution: generated a 512 px PNG with a throwaway Go program.

### 17:03 Web packaging and port conflict
- Step: `fyne package --target wasm`.
- Command/API: `fyne package --target wasm [--tags remote] --name gguix-fyne --icon Icon.png --source-dir .` (the `--help` text for `--target` does not list wasm, but `package.go` accepts `wasm`; output goes to `./wasm/`). `fyne serve` is the other route (it serves, but only builds a release package and cannot take `--tags`).
- Outcome: build works, 40 MB uncompressed .wasm. Port 8090 was already in use by another process on this machine (another agent's app), so web checks were run on 8091. The Fyne wasm runtime prints two harmless errors about creating cache directories ("mkdir fyne/: not implemented on js").
- Time lost: 6
- Category: docs-gap
- Resolution: port override by running the same commands on 8091; Makefile keeps 8090 per the brief (`SERVE_PORT` variable).

### 17:05 Web import and dialogs
- Step: reading `dialog/file_wasm.go`.
- Command/API: `dialog.NewFileOpen` / `NewFileSave` on js.
- Outcome: both are `TODO #2737/#2738` stubs that return without showing anything.
- Time lost: 5
- Category: framework-limitation
- Resolution: `Import Notes...` on the web uses a `syscall/js` file input; export uses a Blob download.

### 17:06 Browser pane shared with another agent
- Step: running the web bench in the built-in browser pane.
- Outcome: the pane tab was navigated away by another session twice. Switched to headless Google Chrome (`--headless=new --enable-logging=stderr`) and read the console lines from its log.
- Time lost: 8
- Category: toolchain
- Resolution: headless Chrome with SwiftShader.

### 17:07 Cross builds
- Step: `fyne-cross linux` and `windows`.
- Command/API: `fyne-cross linux -arch=amd64 -app-id io.gguix.fyne -name gguix-fyne`
- Outcome: attempt 1 failed: the image has Go 1.26.10 with `GOTOOLCHAIN=local`, module needs 1.27.3. Attempt 2 with `-env GOTOOLCHAIN=auto`: failed, `replacement directory ../../core does not exist` (fyne-cross copies only the app directory into the container). Attempt 3: `go mod vendor` first and `-env GOFLAGS=-mod=vendor`: both PASS.
- Time lost: 12 (including 7 GB of image pulls)
- Category: toolchain
- Resolution: temporary vendor directory (removed afterwards by the Makefile).

### 17:10 Bench scroll rate
- Step: first `make bench`.
- Outcome: all eight events. The 200-step scroll took 7.2 s instead of 3.2 s (each step costs about 36 ms to render at 1M rows with custom cells); not a defect, recorded as a result. In the browser (software GL) it was far slower, see capabilities.md.
- Time lost: 0
- Category: runtime-bug (none, informational)
- Resolution: n/a

## Summary

- Wall time to first window: about 15 minutes of work (14:24 to 14:37 after fixing one panic).
- Wall time to feature complete: about 3.5 hours of calendar time including the 2.5 hour pause, about 1 hour of working time.
- Entries by category: toolchain 4, api-confusion 2, framework-limitation 2, docs-gap 3 (some entries carry two), build-error 0, runtime-bug 0, core-issue 0.
- No third-party dependencies were added to the app module.
- Biggest pain points:
  1. `widget.Table` lacks public scroll offset, row selection and header interaction hooks; all had to be built around it (reflect on a private field).
  2. Fyne's file dialogs are Fyne-drawn on macOS and non-functional under WASM; no native dialogs.
  3. Cross compilation with a module that has a `replace` outside the app directory needs a vendor workaround, and the web target is very slow with software rendering.
