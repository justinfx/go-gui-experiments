# MIQT friction log

Times are local wall clock on 2026-11-08 (NZDT). Work started at 14:24.

### 14:24 Qt not installed
- Step: Prerequisites check.
- Command/API: `brew list --versions qt pkgconf`
- Outcome: pkgconf 2.5.1 was present, qt was not. `brew install qt` upgraded about 30 dependencies first, then installed Qt 6.12.0 (bottle, no source build).
- Time lost: 3 (ran in the background while I read docs)
- Category: toolchain
- Resolution: `brew install qt` finished. pkg-config finds Qt6 without extra PKG_CONFIG_PATH (`pkg-config --modversion Qt6Widgets` gives 6.12.0).

### 14:26 Bootstrap
- Step: Create the module as the brief requires.
- Command/API: `go get github.com/mappu/miqt@v0.15.0` in apps/miqt (go.mod written by hand first, with the `replace gguix/core => ../../core` line). MIQT has no project template or scaffolding CLI; the README bootstrap is just `go get` and `go build -ldflags '-s -w'`.
- Outcome: OK, go.mod gained `github.com/mappu/miqt v0.15.0 // indirect` until the first import.
- Time lost: 0
- Category: docs-gap
- Resolution: none needed.

### 14:27 No QStyleHints color scheme binding, no qtest binding
- Step: Checking the Qt API surface I need against the v0.15.0 sources in the module cache.
- Command/API: `grep ColorScheme $(go env GOMODCACHE)/github.com/mappu/miqt@v0.15.0/qt6/*.go`, `ls qt6 | grep -i test`
- Outcome: `QStyleHints::setColorScheme` / `colorScheme` are not in the Go binding (QStyleHints exposes only older properties). There is no `qt6/qtest` package; only `gen_qtestsupport_gui.go` / `gen_qtestsupport_widgets.go` with QTest::QTouchEventSequence.
- Time lost: 2
- Category: framework-limitation
- Resolution: see the theme entry below for the workaround attempts.

### 14:33 First compile and first window
- Step: Compile the Qt bindings, run the app.
- Command/API: `go build` with qt6, qt6/mainthread
- Outcome: first build took 2m11s on a loaded machine (README promised about 10). The app compiled with only 5 trivial API-name errors (overload suffixes such as NewQFont5, NewQPalette7, NewQIntValidator4, StandardIcon needing 3 args). The window appeared on the first run.
- Time lost: 5
- Category: api-confusion
- Resolution: read gen_*.go signatures. Overload numbering is arbitrary and needs grep.

### 14:36 macOS stderr noise
- Step: Running the app.
- Command/API: any run
- Outcome: `NSKeyBindingManager: Bad key binding selector` lines on stderr at start. Harmless.
- Time lost: 0
- Category: runtime-bug
- Resolution: ignored, filtered in bench output.

### 14:38 Theme override
- Step: Light/Dark override.
- Command/API: `QGuiApplication_StyleHints().QObject.SetProperty("colorScheme", NewQVariant4(n))`
- Outcome: attempt 1 (typed setter) does not exist in the binding. Attempt 2 (Q_PROPERTY through setProperty) works on cocoa: 2 dark, 1 light, 0 system. Palette lightness follows.
- Time lost: 6
- Category: framework-limitation
- Resolution: property route.

### 14:41 Testing without qtest
- Step: Automated UI tests.
- Command/API: TestMain runs the Qt event loop on the main thread, tests run in a goroutine and reach widgets through `mainthread.Wait`. Delegates keep a paint log so T10/T11 check what was really painted after `Grab()`.
- Outcome: works, 18 scenarios in about 3 s. No QTest binding, so selection is driven through QItemSelectionModel and header clicks through the SectionClicked signal, not synthetic mouse events. Needed `QGuiApplication_SetQuitOnLastWindowClosed(false)` or the loop exits after the first test window closes.
- Time lost: 5
- Category: framework-limitation
- Resolution: as above.

### 14:44 miqt-docker cross builds
- Step: cross-linux and cross-windows.
- Command/API: `miqt-docker linux64-go1.27-qt6.12 go build ...`, `miqt-docker win64-qt6-static -windows-build --tags=windowsqtstatic`
- Outcome: (1) Docker images are amd64 only, needed `DOCKER_DEFAULT_PLATFORM=linux/amd64` (emulated, slow). (2) The module replaces `../../core`, outside the mounted go.mod directory, so core is mounted with `-v core:/core:ro` through MIQTDOCKER_FLAGS. (3) miqt-docker passes `-t` and fails without a terminal ("cannot attach stdin to a TTY-enabled container"); I wrapped docker with desktop/docker-notty.sh via the DOCKER variable. (4) The only Qt 6 static Windows image has Go 1.26, the module needs 1.27; the toolchain download inside the container fails with a TLS certificate error, and GOTOOLCHAIN=local says core requires go >= 1.27.
- Time lost: about 40 (image builds 5 and 18 min)
- Category: toolchain
- Resolution: Linux see capabilities.md. Windows: unresolved after 3 attempts, recorded as gap.

### 17:05 make exit status
- Step: `make web` must exit 3.
- Command/API: recipe `exit 3`
- Outcome: GNU make reports exit status 2 for any failing recipe.
- Time lost: 2
- Category: toolchain
- Resolution: unresolved, documented in README.

### 17:08 Screen capture
- Step: Look at the packaged app.
- Command/API: `screencapture -x`
- Outcome: captured image was fully black (display asleep or locked), so no visual check of the real app. Visuals verified through `QWidget::grab()` images on the offscreen platform.
- Time lost: 3
- Category: toolchain
- Resolution: not resolved.

## Summary
- Wall time to first window: about 12 minutes (14:24 start, window run at 14:35).
- Wall time to feature complete: about 25 minutes of coding (tests green at 14:44), then packaging, docs and cross-build attempts.
- Entries per category: toolchain 4, api-confusion 1, framework-limitation 2, runtime-bug 1, docs-gap 1, build-error 0, core-issue 0.
- Biggest pain points: (1) Docker cross builds (tty, platform, core mount, Go toolchain mismatch); (2) no QTest and no setColorScheme binding; (3) overload-numbered API names that require grepping generated files.
