# Cogent Core v0.5.27 on Linux (container run, linux/arm64, Ubuntu 24.04)

Harness: `linux/`. Logs and screenshots: `results/cogent/linux/` (`x11/` Xvfb+openbox, `sway/` sway headless Wayland,
`wayland/` weston headless). Timings and visual sharpness not measured.

## Build and tests

| Step | Result |
|---|---|
| `make deps` equivalent: `go install cogentcore.org/core@v0.5.27` (Linux `core` CLI) | PASS |
| `make build` (`core build`, wgpu + glfw, cgo) | PASS, only `-Wunused-result` C warnings from `dnd_linux.c` (`make-build.log`) |
| `make test` | PASS, `GGUIX_TESTS passed=12 failed=0 not_feasible=0` (`make-test.log`) |

## Runtime matrix

| Check | X11 (Xvfb + openbox) | Wayland (sway headless) | Wayland (weston headless) |
|---|---|---|---|
| Starts, shows a window | yes, 1280x800, title `GGUIX - Cogent Core` (`x11-start.png`); rendering is wgpu on software Vulkan (lavapipe) | yes, native `xdg_shell`, but `app_id` is empty (`wayland-start.png`) | NO: SIGSEGV in `glfwInit` (container limit, same as Fyne) |
| Menu bar | in-window row of `core.Button` menus (File Edit View Help); no native/global menu | same | n/a |
| Keyboard shortcuts | Ctrl+E (export dialog), Ctrl+2/Ctrl+1 (view), Ctrl+Shift+C (copy row), Ctrl+I (host) fire. **Ctrl+Q DEADLOCKS the app** (see bug) | same; Ctrl+Q deadlocks | n/a |
| Open/save dialog | Cogent-drawn in-window `core.FilePicker` (title `Export CSV`), not GTK/Qt/portal (`x11-export-dialog.png`). Single X window only | same (`wayland-export-dialog.png`) | n/a |
| Notification | NONE. 0 `Notify` calls on the bus, dunst shows nothing after an export (1,000,000 rows, `/root/gguix-export.csv` written). The app uses an in-app snackbar | same, 0 calls | n/a |
| Clipboard copy | `xclip -o -selection clipboard` returns `8,SKU-0000008,Compact Maple Hammer,...` | `wl-paste` returns `4,SKU-0000004,Compact Copper Bowl,...` | n/a |
| Reveal in file manager | `xdg-open /tmp/prefs-c` and `xdg-open /root` (parent of last export); no `FileManager1`; file not selected | identical (`sway/xdg-open.log`) | n/a |

## BUG found (app, not framework): Ctrl+Q / File > Quit freezes the app on Linux

Symptom (X11 and Wayland): Ctrl+Q leaves the process alive at ~28% CPU and the window stops responding to clicks. Evidence: SIGQUIT goroutine
dump in `results/cogent/linux/x11/app9.log`. The event-loop goroutine is blocked in `sync.Mutex.Lock` at
`cogentcore.org/core@v0.5.27/core/renderwindow.go:158` (the close-request func does `rc.Lock()`), reached from our
`gguix/apps/cogent/menus_desktop.go:45` (`system.TheApp.QuitReq()`) -> `base/app.go:168` -> `app_multi.go:129` `QuitClean` -> `CloseReq`.
The event handler already holds the same render-context lock (`core/renderwindow.go:484`, `handleEvent` does `rc.Lock()` around `mainHandleEvent` at :511),
so the call self-deadlocks. Cogent's own Quit menu item avoids it by quitting from a goroutine: `core/menu.go:298` `go TheApp.QuitReq()`.
On macOS the standard app-menu Quit never reaches this code, so the bug did not show there.

Verified fix (applied only to the scratch copy in the container, NOT to `apps/cogent`; patch in `quit-deadlock.patch`): prefix the call with `go`
(`go system.TheApp.QuitReq()`); after rebuilding, Ctrl+Q exits the process cleanly (`exited`). Because the rules say to avoid app changes, the
repo was left untouched; the one-line change is recommended.

## Other findings

1. No notification API in v0.5.27 (capabilities.md already says gap); confirmed on Linux, nothing is sent.
2. Reveal is core `host.RevealInFileManager` (`core/host/native.go:55-76`): `xdg-open` on the directory, cannot select the file (degraded vs `open -R`).
3. The window sets no Wayland `app_id` (sway tree: `app_id: ""`), which hurts taskbar/desktop-file matching on a real Wayland desktop. Observation only; library cause not traced.
4. weston headless: no `wl_seat` -> `wl_init.c:856` null deref in GLFW (`github.com/go-gl/glfw/v3.4/glfw@v0.1.0-pre.1.0.20260628091122-0bd588dc30cf/glfw/src/wl_init.c:856`). Container limit.
5. Harness note: on Wayland the first click after the pointer enters a window can be dropped by Cogent (a click sent right after the pointer moved in did not select a row; a second click did). Not investigated further; may be a harness timing artefact.

## Not verified / needs real hardware

Hardware Vulkan/GPU path (lavapipe was used), HiDPI scaling, IME (`ime_linux.go`), drag and drop, real desktop notification servers (nothing is sent anyway), real file manager.

## Re-check after fix (apps/cogent now uses `go system.TheApp.QuitReq()`)

PASS. Fresh `prep.sh` copy, `make build` and `make test` (12/12) pass; Ctrl+Q exits the process cleanly on X11 and on sway (no hang). cogent-fit and cogent-fitcap: build PASS, `make test` 13/13 each.
Note: on Linux the app builds the Quit item only inside the macOS-only "GGUIX" menu (`menus_desktop.go:44-47`, `if mac`), so there is no visible File > Quit item on Linux; only the Ctrl+Q shortcut exists, so "File > Quit" could not be exercised (`x11-file-menu-fixed.png` shows File: Export CSV, Import Notes, Reveal Last Export).
History: the original finding above (Ctrl+Q / File > Quit freezes the app, SIGQUIT dump, `quit-deadlock.patch`) stands as the record of the earlier failure; status is now FIXED.
