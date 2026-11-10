# Cogent Core fix pass

## Gap 1: minimum window size 1024 x 640
- Change: in `platformInit` (OnShow) the system window is type-asserted to `*desktop.Window` and `Glw.SetSizeLimits(1024, 640, glfw.DontCare, glfw.DontCare)` is called (`applyMinSize`). Desktop only (`main_desktop.go`, `//go:build !js`, within the existing desktop target code). `go.mod`: `github.com/go-gl/glfw/v3.4/glfw` is now a direct dependency.
- API: `cogentcore.org/core/system/driver/desktop.Window.Glw` (`cogentcore.org/core@v0.5.27/system/driver/desktop/window.go:27-31`), `glfw.Window.SetSizeLimits`.
- Verified: `make build` and `go test .` pass. Resizing below the limit was NOT verified on a real window. A glfw unit test crashes under `go test` on macOS (Cocoa needs the main thread), and a standalone hidden-window program showed programmatic `SetSize(300, 200)` is not clamped (the limit applies to interactive resizing), so there is no automated proof. Manual check: run `make run` and drag a window edge below 1024 x 640.

## Linux fix (2026-11-09)

Quit menu item (Ctrl+Q) deadlocked: it called `system.TheApp.QuitReq()` while the render lock was held. Now `go system.TheApp.QuitReq()`, matching Cogent's own Quit (`core/menu.go:298`). Found by the Linux runtime check; the bug was in our app, not Cogent.

## Manual check by the user (2026-11-09)

- Window minimum size: dragging the corner stops at the minimum. PASS.
