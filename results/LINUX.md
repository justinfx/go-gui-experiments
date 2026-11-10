# Linux runtime check (container), all seven desktop apps

Native `linux/arm64` Ubuntu 24.04 container on Apple Silicon (Go 1.27.3, Qt 6.4.2, GTK 3.24 / 4.14, WebKitGTK 2.52, weston 13, sway 1.9). Harness in `linux/` (see `linux/README.md`).
Per-app evidence (commands, log excerpts, screenshots, library file:line citations): `results/<id>/linux.md` and `results/<id>/linux/`.
No file under `apps/` or `core/` was changed. Where a change was tried it was in the scratch copy under `/work` and is recorded as a patch.

## Summary table

"X11" is Xvfb + openbox. "Wayland" is sway headless (native `xdg_shell`, no XWayland), the only Wayland compositor here that accepts injected input. Y = works, N = does not work, P = partial. Details and caveats in the per-app files.

| App | Build (native arm64) | `make test` | Starts, X11 | Starts, Wayland | Menu bar | Shortcuts | File dialogs | Notification (`Notifications.Notify`) | Clipboard | Reveal |
|---|---|---|---|---|---|---|---|---|---|---|
| fyne | Y | Y (12/12) | Y | Y | in-window, framework-drawn | Y | Fyne-drawn in-window dialog (not GTK/portal) | **Y** (D-Bus from the library) | Y | `xdg-open <parent dir>`, no select |
| cogent | Y | Y (12/12) | Y | Y | in-window buttons | P: **Ctrl+Q deadlocks the app (app bug)** | Cogent-drawn in-window picker | **N** (no API; in-app snackbar) | Y | `xdg-open <parent dir>`, no select |
| wails (GTK3 build) | Y (default GTK4 build also compiles) | Y as 34/34 with Chromium; 20/34 with the configured Chrome (no arm64 Chrome) | Y | Y | **native GtkMenuBar** | Y | **native GTK3 chooser** (not portal) | **Y** (D-Bus from the library) | Y | `xdg-open` via `OpenFileManager`, folder or parent, no select |
| wails (default GTK4 build) | Y | n/a | **not runnable in container** (sandbox) | not runnable | not verified | not verified | not verified | not verified | not verified | not verified |
| gotth (webview_go) | Y, only with a container pkg-config alias (`webkit2gtk-4.0`) | Y (11/11, 1 not feasible) | Y | Y | in-page | Y | **N (app gap): `osascript` not found, error dialog** | **N** (unreachable, uses `osascript`) | P: works only through the page Clipboard API, native `pbcopy` path fails silently | `xdg-open <config dir>`; last export disabled (no export possible) |
| miqt (Qt Widgets) | Y (2.5 min) | Y (18/18) | Y | Y | native QMenuBar in window | Y | Qt `QFileDialog` (not GTK/portal) | **N by default**; **app bug** (nil icon) even with a tray watcher | Y | `xdg-open file://<dir>` (Qt) and `xdg-open <parent dir>` (core), no select |
| gotth-glaze (CGO_ENABLED=0) | Y | Y (11/11, 1 not feasible) | Y (GTK3 path only) | Y (GTK3 path only) | in-page (lib has no Linux menu) | Y | **real GTK3 `GtkFileChooserNative`** | **N** (lib returns `ErrUnsupported`) | P: native path unsupported, works through the page Clipboard API | `xdg-open <parent dir>`, also for the config folder (opens its parent); no select |
| miqt-gotth (Qt WebEngine) | Y (25 s warm) | Y (11/11, 1 not feasible) | Y, **only with Chromium sandbox disabled** | Y, same | native QMenuBar | Y | Qt `QFileDialog` | P: only with a tray watcher (KDE), none on stock GNOME | Y | `xdg-open <dir>` and `<parent dir>`, no select |

weston headless column (all apps): Fyne and Cogent crash in `glfwInit` with SIGSEGV; GTK apps (gotth, wails GTK3) and Qt apps stay alive but print seat assertions or cannot be inspected. weston headless advertises no `wl_seat`; GLFW dereferences it at
`github.com/go-gl/glfw/v3.4/glfw@v0.1.0-pre.2/glfw/src/wl_init.c:856`. This is a container limit, not an app failure.

## Update: three fixes re-checked on Linux (X11 and sway)

| Fix | Result |
|---|---|
| cogent (+fit, +fitcap): `go system.TheApp.QuitReq()` | PASS: Ctrl+Q exits cleanly on X11 and sway; no visible File > Quit exists on Linux (Quit item only built in the macOS menu). fit/fitcap build and `make test` 13/13 |
| miqt (+fit, +fitcap): `qt.NewQIcon()` | PASS: with `sni-watcher.py`, one `Notify` reaches the bus and dunst on X11 and sway. fit/fitcap build and `make test` 18/18 |
| gotth (+fit, +fitcap): zenity, beeep, atotto/clipboard | PASS: Export dialog opens and writes the CSV, Import dialog opens, `Notify` arrives, Copy Row reaches xclip/wl-paste. New runtime dependency: `zenity` (or qarma/matedialog) plus xclip/xsel or wl-clipboard. fit/fitcap build and `make test` pass |

The ranked list below is the original finding text; items 1, 2 and 3 are now FIXED (History kept in the per-app files).

## Notable failures, ranked by how much they matter on a real Linux desktop

1. **gotth desktop shell is macOS-only for dialogs, notifications and clipboard.** `cmd/desktop/native_osa.go` runs `osascript` and `pbcopy` on every OS (build tag `!(darwin && native)`). On Linux Export CSV and Import Notes show `exec: "osascript": executable file not found`, so exporting is impossible and the notification can never fire. Copy works only because the page also calls `navigator.clipboard`. App gap (webview_go has no dialog/notification API).
2. **cogent: Ctrl+Q / File > Quit freezes the process on Linux (app bug).** `menus_desktop.go:45` calls `system.TheApp.QuitReq()` from inside the event handler while Cogent already holds the render-context lock (`core/renderwindow.go:484`), then `CloseReq` tries to take it again (`:158`). Cogent's own menu uses `go TheApp.QuitReq()` (`core/menu.go:298`). One-line fix verified in the scratch copy (`results/cogent/linux/quit-deadlock.patch`). Hidden on macOS because the system app menu handles Quit.
3. **miqt: notification silently dropped (app bug + platform).** `ui_desktop.go:351` passes a Go `nil` as the `QIcon` to `ShowMessage3`, which MIQT dereferences as a null reference (`gen_qsystemtrayicon.go:270`, `gen_qsystemtrayicon.cpp:331`); with a tray watcher present nothing is sent. Fixed in the scratch copy (`results/miqt/linux/notify-nil-icon.patch`). Independently, without a `StatusNotifierWatcher` (stock GNOME) Qt tray messages are unavailable and the app returns silently.
4. **miqt-gotth looks broken-ish on Ubuntu's Qt 6.4**: Chromium 102 does not support `color-mix()` used by the Tailwind v4 stylesheet, so dialogs and some buttons lose their styling (`results/miqt-gotth/linux/x11-host-dialog.png`). A Linux build needs a newer Qt WebEngine.
5. **glaze: reveal config folder opens the parent** (`native/openurl/openurl_linux.go:26-28` always opens `filepath.Dir`; the app passes a directory). Notification and native clipboard are unimplemented on Linux in that library (`notify_other.go:1,5`, `clipboard_linux.go:3,28`).
6. **Nobody can select the file on reveal.** Every app ends in `xdg-open` of a directory; none calls `org.freedesktop.FileManager1.ShowItems`. Wails documents it (`internal/fileexplorer/fileexplorer_linux.go:77`), glaze documents it, core's helper just opens the parent. Degraded vs macOS `open -R` in all seven.
7. Wails v3.1.0's default Linux backend is GTK4/WebKitGTK 6.0, whose sandbox could not start in this container (see below); the GTK3 backend needs `-tags gtk3`.
8. `webview_go` pins `webkit2gtk-4.0` (`webview.go:9`), which Ubuntu 24.04 and Debian 13 do not ship; a stock `go build` fails there.

Framework vs app vs container, at a glance: Fyne notification and Wails notification/dialogs/menu are framework-native and good; Cogent has no native dialog/menu/notification (framework limits) plus one app bug; miqt problems are app bugs plus Qt tray dependence; gotth problems are an app gap; glaze limits are library gaps (notify, clipboard) plus one app-level reveal choice; container limits are the weston seat, sandboxes, arm64 Chrome.

## Method and limits

- Method: each app was built natively in the container following its Makefile (extra steps only where unavoidable: `go install` of the CLI tools, a Linux Tailwind in place of the macOS one, the `webkit2gtk-4.0` pkg-config alias), `make test` was run, then the app was launched under X11 (Xvfb + openbox, `xdotool`, `xclip`) and Wayland (sway headless, `wlinput.py` virtual keyboard and pointer, `wl-paste`, `grim` cropped to the window rectangle). A session bus with dunst and a `dbus-monitor` log recorded every `org.freedesktop.Notifications.Notify` and any `FileManager1` or portal call; a logging `xdg-open` shim recorded reveal requests.
- Screenshots are of the app window only (`import -window <id>` on X11, `grim -g` of the window rectangle on Wayland), saved under `results/<id>/linux/`. Popup menus are separate override-redirect windows and appear as black rectangles in the parent window capture.
- **Timings and visual sharpness were NOT measured.** The container renders in software (llvmpipe, lavapipe), at an unrepresentative resolution and with no compositor; numbers from it would say nothing about real hardware, so none are reported.
- Container limits that shaped the results: no user namespaces under Docker's default seccomp (blocks WebKitGTK 6.0's bubblewrap and Chromium's sandbox), root user (Chromium refuses `--no-sandbox`-less root), no file manager, no GNOME/KDE shell or portal backends, no `StatusNotifierWatcher`, weston headless without a seat, Chrome unavailable on arm64.
- **Sandbox relaxation disclosure:** the miqt-gotth results were obtained with `QTWEBENGINE_DISABLE_SANDBOX=1` (set when the test container was created; with it unset the app shows no window). I did not relax seccomp and did not disable WebKitGTK's sandbox; the Wails GTK4 build and glaze's GTK4 path were therefore not run. glaze was run on its GTK3/WebKitGTK 4.1 path by moving `libwebkitgtk-6.0` aside in the throwaway container. Two relaxations were attempted and refused by the tool's safety checks; they are now opt-in switches in `linux/dx.sh`, off by default.
- Qt is the distro 6.4.2, not the 6.12 used on macOS; GTK and WebKit are Ubuntu's.

## Needs checking on real Linux hardware

1. Wails default GTK4/WebKitGTK 6.0 build (could not start here) and glaze's GTK4 path: start, menu, dialogs, notification, clipboard.
2. miqt-gotth with the Chromium sandbox enabled as a normal user, and with Qt 6.7+ WebEngine for the CSS rendering.
3. GNOME and KDE: Qt tray notifications (needs a StatusNotifierWatcher), GTK file chooser vs portal, global menu, real notification servers.
4. What a real file manager does with the arguments that Wails and Qt pass (Wails runs the default `inode/directory` handler's Exec line; whether Nautilus/Dolphin selects the file is not observed).
5. Native Wayland on a real compositor: Fyne/GLFW and Cogent decorations and `app_id` (Cogent sets none), IME, fractional scaling, drag and drop (Cogent has Wayland-specific DnD limits in its source), clipboard persistence after the app loses focus.
6. Hardware GL/Vulkan paths (Fyne, Cogent) and WebKit/Chromium GPU rendering; the container used llvmpipe and lavapipe.
7. Fyne with `-tags flatpak` or a portal-capable setup (the portal file chooser and notification paths exist only there), and the Cogent Ctrl+Q fix and miqt notify fix applied to the real apps.
8. Ctrl+Q on X11 for Wails and glaze (verified only on Wayland for those two).
