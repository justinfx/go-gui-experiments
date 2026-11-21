# Friction log: Gophics (apps/gophics)

Framework: `github.com/doug/gophics` `v0.0.0-20261103180631-8d360971eb89` (commit `8d36097`). Module cache path below is written as `$GOMODCACHE/github.com/doug/gophics@v0.0.0-20261103180631-8d360971eb89` (shortened to `gophics@8d36097`).

### 14:57 Fetch the pinned framework
- Step: resolve the pseudo-version for commit `8d36097`
- Command/API: `go get github.com/doug/gophics@8d36097` (in a scratch module)
- Outcome: resolved to `v0.0.0-20261103180631-8d360971eb89`. No problems.
- Time lost: 0
- Category: toolchain
- Resolution: used that pseudo-version in `go.mod`

### 14:59 Bootstrap with the framework CLI
- Step: project setup as the README recommends (`gophics create`)
- Command/API: `GOBIN=apps/gophics/.tools go install github.com/doug/gophics/cmd/gophics@8d36097`, then `gophics create -p desktop,web -module gguix/apps/gophics gophics` in a scratch dir and copied the result into `apps/gophics/`
- Outcome: generated `go.mod`, `main.go`, `ui/app.go`, `README.md`, `.gitignore`. The printed next steps say the dependency is "not yet published" and suggest a `replace` to a local checkout, which is out of date: `go mod edit -require=github.com/doug/gophics@v0.0.0-20261103180631-8d360971eb89 && go mod tidy` works directly against the module proxy.
- Time lost: 2
- Category: docs-gap
- Resolution: added the require by hand; the template app builds with `CGO_ENABLED=0 go build` (20.7 MB binary)

### 15:00 Survey the widget catalog against the brief
- Step: map each required control to a Gophics widget before writing code
- Command/API: `go doc github.com/doug/gophics/theme`, `go doc .../widget`, `go doc .../shell`, `skills/gophics/SKILL.md`, `internal/capgen/README.md` (capability matrix)
- Outcome: present: `theme.Table`, `widget.LazyList` + `ScrollController`, `theme.Field`, `Dropdown`, `Segmented`, `Slider` (0..1 only), `Radio`, `Checkbox`, `ShowDatePicker`, `ShowDialog`, `ShowMenu`, `ShowSnackbar`, `Progress`, `ctx.Menus()` (native NSMenu), `ctx.FilePicker()`, `ctx.FolderPicker()`, `ctx.Notifier()`, `ctx.Clipboard()`, `ctx.WindowControl()`. Missing: split container, spin box, any `Disabled` state on theme controls (only `widget.TextField.Disabled`), menu shortcuts and checked menu items (`shell.MenuItem` has neither), a global shortcut/keymap API (keys reach only the focused widget), minimum window size, programmatic quit, a reveal-in-Finder API, a save dialog that returns a path (`FilePicker.Save` takes the bytes up front).
- Time lost: 0
- Category: framework-limitation
- Resolution: recorded per row in capabilities.md with the fallbacks used

### 15:04 theme.Table cannot be driven programmatically
- Step: table view
- Command/API: `theme.Table` (`gophics@8d36097/theme/table.go`)
- Outcome: it wraps `widget.LazyList` but exposes no `ScrollController`, no visible-range callback, no column resize, and draws the sort arrow as a separate text run. Go-to-row, restoring the selection, scroll-to-top on query change and the bench scroll steps all need the offset.
- Time lost: 5
- Category: framework-limitation
- Resolution: built the table from the same parts theme.Table uses (`widget.LazyList` with a `Controller`, header row, the theme's hairline and selected-row tint `Lerp(Bg, Primary, 0.10)`). Column resize added with `Interactive.OnDrag` grips.

### 15:05 LazyList offsets are float32 sums over every row
- Step: sizing 1,000,000 rows
- Command/API: `widget.LazyList` (`gophics@8d36097/widget/lazylist.go`): Build loops `for i := 0; i < Count; i++ { y += height(i) }` in float32 on every rebuild and keeps a `[]float32` of Count heights
- Outcome: the theme row height (Body 15 + 18 = 33 px) would accumulate float32 rounding past 2^24 px (about row 508,000). Each scroll rebuild also walks all rows.
- Time lost: 5
- Category: framework-limitation
- Resolution: table rows pinned to 32 px (exact in float32) and cards to 96+8 = 104 px (multiple of 8). Recorded as a deviation from "framework default row height".

### 15:12 App aborts at first frame: setAccessibilityDisclosing
- Step: first desktop run
- Command/API: `theme.Dropdown` (status filter, theme picker, form category)
- Outcome: `*** Terminating app due to uncaught exception 'NSInvalidArgumentException', reason: '-[GoGPUA11yElement setAccessibilityDisclosing:]: unrecognized selector'`, SIGABRT. Dropdown sets `SemInfo.Expanded`, and the macOS bridge (`internal/gfx/gogpu/internal/platform/darwin/a11y.go:357`) sends `setAccessibilityDisclosing:` to an `NSAccessibilityElement` subclass, which this macOS (26.6) does not implement. The headless tests did not catch it (no AppKit). There is no switch to turn the accessibility bridge off.
- Time lost: 6
- Category: runtime-bug
- Resolution: replaced every `theme.Dropdown` with `ui.Select` (a `theme.Button` that opens `theme.ShowMenu`), `ui/dropdown.go`. The app then ran the full bench.

### 15:15 web-remote exits on first frame (wasm deadlock)
- Step: `web-remote` in Chrome
- Command/API: `PrefsStore.Save` (remote: HTTP PUT) called from a debounced `Ctx.Post` callback
- Outcome: Go program exited (`exit code: 2`) with a stack through `remote.(*Client).Save` <- `app.(*core).drainPosted` <- `shell/web.(*window).Invalidate.func1` <- `syscall/js.handleEvent`. A blocking fetch on the js event goroutine deadlocks the wasm runtime. Not documented in Gophics' web notes; `Ctx.Lifetime` docs only say to start work in a goroutine.
- Time lost: 5
- Category: api-confusion
- Resolution: prefs Load moved to `main` before `app.Run`, Save runs on a goroutine. Same rule as the pager (already async).

### 15:16 Web bench export used a file path
- Step: `?bench=1` in Chrome
- Command/API: `bench.ExportPath()` returns a temp path on js too
- Outcome: `export: open /tmp/gguix-bench-export.csv: not implemented on js`
- Time lost: 2
- Category: core-issue
- Resolution: the driver passes "" on js (marked `gguix:target web` in `ui/driver_bench.go`); all eight metrics then print. Brief already says the web CSV is discarded, so this is a minor core rough edge.

### 15:18 Linux container run
- Step: section 12 checks, `results/gophics/linux/check-x11.sh`
- Command/API: `linux/dx.sh` + `run.sh x11|sway`
- Outcome: builds and runs on both; window class `gogpu` / app_id `gogpu` (no Config field to set it); theme controls' Opacity dimming did not show on the Linux renderer (Save/Revert look enabled when clean).
- Time lost: 4
- Category: framework-limitation
- Resolution: recorded in linux.md

## Fix pass 1 (2026-11-21)

### 15:46 Hands-on fixes: clip leak, scrollbars, shortcuts in fields, export name
- Step: FIX1.md issues 1 to 7
- Command/API: `paint.Canvas.PushClip` inside `widget.Canvas`; `widget.Scroll` scrollbar; `widget.TextField.OnKeyPreview`; `shell.FilePicker.Save`
- Outcome: (1) a nested `PushClip` in the half-star draw escaped the table viewport's clip on the GPU backend only (CPU headless render was clean; reproduced with `Headless.RenderGPU` under tag `gophics_gpu`). (2) the vertical scrollbar was painted at the right edge of the 1,320 px table content, off screen; the overlay bar also fades when idle with no option to keep it. (4) `widget.Fill` expands to its constraints, so the 1 px grip line filled the 6 px grip. (6) keys go only to the focused widget, and `theme.Field` hides `OnKeyPreview`. (7) `FilePicker.Save` never reports the chosen path.
- Time lost: 25
- Category: framework-limitation
- Resolution: half star drawn from a clamped half path (no clip); list at visible width plus a horizontal `Scroll` strip and `widget.Transform` shift; `Decorated` divider; `ui.Field` (theme.Field's Build plus OnKeyPreview); in-app file name prompt then folder chooser. Idle scrollbar fade and native save-with-path recorded as gaps. Details in `fixpass.md`.

## Summary

- Wall time to first window: about 16 minutes (14:57 start, 15:13 first window with the full UI; the template window was never run separately).
- Wall time to feature complete: about 22 minutes (15:19), plus results writing.
- Entries per category: docs-gap 1, api-confusion 1, toolchain 1, build-error 0, runtime-bug 1, framework-limitation 5, core-issue 1 (fix pass 1 added one framework-limitation entry).
- Biggest pain points:
  1. `theme.Dropdown` aborts the macOS app through the accessibility bridge (`setAccessibilityDisclosing:`), invisible to headless tests.
  2. The widget catalog is thin for desktop data apps: no split view, spin box, disabled state, menu shortcuts or checked items, global key handling, save-dialog path or reveal API; `theme.Table` has no scroll controller.
  3. `LazyList` walks every row in float32 on each rebuild, which needs a power-of-two row height to stay exact at 1M rows.
