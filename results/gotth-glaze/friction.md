# GoTTH + glaze friction log

Shell swap on a copy of `apps/gotth`. Dates are 2026-11-09.

### 09:40 Read the addendum and the glaze / native sources
- Step: checking every API before relying on it
- Command/API: read `~/go/pkg/mod/github.com/crgimenes/glaze@v0.0.83/{webview_common.go,dialog.go,menu/*.go,README.md}` and `native@v0.1.18/{notify,clipboard,openurl}`
- Outcome: all needed APIs exist: `glaze.New`, `SetTitle`, `SetSize` with `HintNone` / `HintMin`, `Eval`, `Bind`, `Dispatch`, `OpenFile`, `SaveFile`, `menu.Set` with `Selector`, `notify.Send`, `clipboard.WriteText`, `openurl.Reveal`. Both modules were already in the module cache, so `go get` needed no network.
- Time lost: 0
- Category: docs-gap (none: the READMEs and doc comments were accurate)
- Resolution: none needed

### 09:44 glaze.New returns an error
- Step: porting `main.go`
- Command/API: `webview.New(false)` became `glaze.New(false)` which returns `(WebView, error)`
- Outcome: one extra error check. `HintNone` and `HintMin` carry over unchanged.
- Time lost: 0
- Category: api-confusion
- Resolution: `log.Fatal` on error

### 09:48 menu.Item has no checked state and no mutators
- Step: porting the radio items (Table / Cards, System / Light / Dark) and the enabled flags of the NSMenu shim
- Command/API: `menu.Item` fields: Title, Shortcut, OnClick, Selector, Services, Submenu, Separator, Disabled
- Outcome: nothing can be updated after `menu.Set`. The shell rebuilds the whole menu from `gguixMenuState` and draws the check as a leading mark in the title. `Options.Dispatch` must stay nil for the first `Set` (before `Run`) or the process hangs without a window, and must be `w.Dispatch` for later calls; both are in the `Options` doc comment.
- Time lost: 4
- Category: framework-limitation
- Resolution: rebuild on state change; verified through System Events that clicking View > Cards moved the check mark.

### 09:55 Windows shortcuts and Linux menu bar are not provided by glaze/menu
- Step: reading `menu_windows.go` and `menu_linux.go`
- Command/API: `menu.Set`
- Outcome: Windows draws a menu bar but ignores `Shortcut`; Linux draws no bar and installs only shortcuts (and skips `Selector` items). Documented in the package comment and README.
- Time lost: 2
- Category: framework-limitation
- Resolution: on Linux `installNativeMenu` returns false so the page menubar shows and `static/app.js` shortcuts apply (no doubled handlers). On Windows the page keydown handler provides the shortcuts. Neither platform was run.

### 10:02 App Nap guard needed C before
- Step: replacing `appnap_desktop.m`
- Command/API: `-[NSProcessInfo beginActivityWithOptions:reason:]` through `github.com/ebitengine/purego/objc`
- Outcome: 10 lines of Go; purego is already a glaze dependency, now a direct one. Effect on bench throttling not measured.
- Time lost: 3
- Category: toolchain
- Resolution: `cmd/desktop/appnap_darwin.go`

### 10:08 Reveal is outside gotth.Native
- Step: the addendum asks for `openurl` for reveal, but reveal goes through `host.RevealInFileManager` (core, unchanged) in `handlers_desktop.go`
- Command/API: `openurl.Reveal`
- Outcome: added `Server.reveal`, which uses an optional `Reveal(string) error` method on the shell's `Native` and falls back to the core helper. This is the only edit outside `cmd/desktop` and the Makefile, cross script, README.
- Time lost: 3
- Category: core-issue (shared helper not replaceable from the shell)
- Resolution: optional interface, as above

### 10:15 Cross builds no longer need Docker
- Step: `desktop/cross.sh`
- Command/API: `CGO_ENABLED=0 GOOS=windows|linux go build ./cmd/desktop`
- Outcome: both compile first time (also arm64). gotth needed Docker, mingw and a header symlink for windows and two attempts for linux. Rewrote `cross.sh` to a plain loop over web and desktop. The script was not run (compile checks only, as instructed).
- Time lost: 0
- Category: toolchain
- Resolution: none needed

### 10:20 cgoprobe with CGO_ENABLED=0
- Step: `make cgoprobe`
- Command/API: `CGO_ENABLED=0 go build -tags cgoprobe ./cmd/desktop`
- Outcome: web PASS, desktop FAIL (`../../core/checksum_nocgo.go:8:9: undefined: cgoprobe_requires_cgo_but_cgo_is_unavailable_for_this_target`), as expected because the desktop shell now ships cgo-free. An extra `desktop-cgo` line (cgo on) PASSes. web-export-test PASS.
- Time lost: 0
- Category: core-issue (expected)
- Resolution: reported, not worked around

### 10:25 Bash safety check refused harmless commands
- Step: editing the Makefile and writing result files with sed, python and heredocs containing `$$(`
- Outcome: the tool refused several commands as possible dangerous removals though none removed anything.
- Time lost: 5
- Category: toolchain
- Resolution: used the Edit and Write tools instead.

### 10:30 Checking behaviour on macOS
- Step: launched `bin/gguix-gotth` with a temp prefs dir and drove it with System Events (accessibility worked this time)
- Outcome: menu bar GGUIX/File/Edit/View/Help present; View > Cards click ran the page action and moved the check; File > Export CSV opened a dialog window (2 windows), Escape closed it; window title `GGUIX - GoTTH`, size 1280 x 832 with title bar. A probe program showed clipboard write/read round trip, `notify.Send` nil, `openurl.Reveal` nil. Notification banner and Finder reveal were not looked at; Import dialog file selection not exercised.
- Time lost: 0
- Category: runtime-bug (none found)
- Resolution: none needed

## Summary

- Entries per category: toolchain 3, framework-limitation 2, api-confusion 1, core-issue 2, docs-gap 1 (none needed), runtime-bug 1 (none found).
- Pain points: no checked state or mutation in `glaze/menu`; Windows shortcuts and Linux menu bar absent; reveal not reachable through the shell interface.
