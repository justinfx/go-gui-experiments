# Wails v3.1.0 on Linux (container run, linux/arm64, Ubuntu 24.04)

Harness: `linux/`. Logs and screenshots: `results/wails/linux/` (`x11/` Xvfb+openbox, `sway/` sway headless Wayland, `wayland/` weston headless).
Timings and visual sharpness not measured.

## Two Linux builds exist (read this first)

Wails v3.1.0 defaults to **GTK4 + WebKitGTK 6.0** on Linux (`pkg/application/linux_cgo.go:1` build tag `linux && cgo && !gtk3 ...`, pkg-config `gtk4 webkitgtk-6.0` at :18).
The GTK3 + WebKitGTK 4.1 backend is behind the `gtk3` build tag (`pkg/application/linux_cgo_gtk3.go:1,41`).
The Makefile's `make build` (`wails3 task build`) therefore produces the GTK4 binary.

| Build | Compiles | Runs in this container |
|---|---|---|
| default `make build` (GTK4, WebKitGTK 6.0) | PASS (10.7 MB) | **NO.** `bwrap: Creating new namespace failed: Operation not permitted` -> `Failed to fully launch dbus-proxy` -> SIGTRAP inside `webkit_web_view_evaluate_javascript` (`wayland/app-gtk4.log`). CONTAINER limit: WebKitGTK 6.0 launches its bubblewrap sandbox at start, which needs user namespaces that Docker's default seccomp blocks. I was not permitted to relax seccomp or disable the sandbox, so the GTK4 build is UNVERIFIED at run time |
| `wails3 task build EXTRA_TAGS=gtk3` (GTK3, WebKitGTK 4.1) | PASS | yes. Every runtime result below is for this GTK3 build (binary `bin/gguix-wails-gtk3` in the scratch copy; no repo change) |

## Build and tests

| Step | Result |
|---|---|
| `make deps` equivalent: `go install wails3@v3.1.0`, `npm ci` | PASS |
| `make build` | PASS (`make-build.log`); frontend built by Vite, then `go build -tags production` |
| `make test` as shipped | Go tests pass, but the 14 Playwright e2e tests FAIL: `Chromium distribution 'chrome' is not found at /opt/google/chrome/chrome` because `playwright.config.ts` uses `channel: "chrome"` and Google Chrome has no Linux arm64 build. `GGUIX_TESTS passed=20 failed=14` (`make-test.log`). Container/arch limit, not an app bug |
| `make test` with Playwright's bundled Chromium (the `channel` option removed in the scratch copy only) | `GGUIX_TESTS passed=34 failed=0` (`make-test-chromium.log`). These tests drive the server-mode (web) build, not the desktop shell |

## Runtime matrix (GTK3 build)

| Check | X11 (Xvfb + openbox) | Wayland (sway headless) | Wayland (weston headless) |
|---|---|---|---|
| Starts, shows a window | yes, 1280x800, title `GGUIX - Wails`, WM_CLASS `gguix-wails-gtk3` (`x11-start.png`) | yes, native `xdg_shell`, `app_id=gguix-wails-gtk3` (`wayland-start.png`) | process alive but GDK `gdk_seat_get_keyboard: assertion 'GDK_IS_SEAT (seat)' failed` (no seat); window not verified. Container limit |
| Menu bar | **native GtkMenuBar** above the web view (window child at +1+22, menu popups are separate X windows): `linux_cgo_gtk3.go:796` (`gtk_menu_bar_new`), `:1485`. The page hides its own menu | same | n/a |
| Keyboard shortcuts | menu accelerators fire: Ctrl+2 (cards), Ctrl+E (native save dialog), Ctrl+Shift+C, Ctrl+I. Caveat: an open menu popup grabs the keyboard, so a Ctrl+E sent while the File menu was open did nothing | same set fire; Ctrl+Q exits the process | n/a |
| Open/save dialog | **native GTK3 `GtkFileChooserDialog`** (separate dialog window, `_NET_WM_WINDOW_TYPE_DIALOG`, not a portal; no `org.freedesktop.portal.FileChooser` call): the library calls `gtk_file_chooser_dialog_new` (`linux_cgo_gtk3.go:155-156`) (`x11-export-dialog.png`, `wayland-export-dialog.png`). Export wrote 118,342,650 bytes | same | n/a |
| Notification | **YES**: `org.freedesktop.Notifications.Notify`, app_name `GGUIX - Wails`, summary `Export complete`, body `1,000,000 rows written`, actions `default`, hint `x-notification-id`; dunst displayed it (`x11/dbus.log` line 1012, `x11/dunst.log`). No bundle needed on Linux. Library: `services/notifications/notifications_linux.go:48-49,278` (D-Bus), authorization is a stub that returns true (`:133-135`) | same call arrives (`sway/dunst.log`) | n/a |
| Clipboard copy | `xclip -o -selection clipboard` returns the CSV row | `wl-paste` returns the row | n/a |
| Reveal in file manager | `app.Env.OpenFileManager`: Reveal Config -> `xdg-open /tmp/prefs-w` (the folder itself), Reveal Last Export -> `xdg-open /work/apps/wails/bin` (parent dir of the file); no `FileManager1`; file not selected | same (`xdg-open /tmp/prefs-ws`, `/work/apps/wails`) | n/a |

## Findings

1. Default Linux build is GTK4/WebKitGTK 6.0 (framework default, see top). On a stock Ubuntu/Debian/Fedora desktop this needs `libgtk-4-dev libwebkitgtk-6.0-dev` to build and the sandbox works there; the container could not exercise it.
2. Dialogs, notifications and the menu bar are all native on Linux with no extra code in the app, the most complete native integration among the web-view apps.
3. Reveal never selects the file. The library's Linux code states it: `internal/fileexplorer/fileexplorer_linux.go:77` ("The linux fallback explorer opening does not support file selection") and for a file it opens the containing directory (`:82-87`). It first tries the default `inode/directory` handler's `.desktop` Exec line (`:19-63`); in the container no file manager is installed so the `xdg-open` fallback ran, and what a real Nautilus/Dolphin does with the path argument was not observed. Unlike glaze, a directory is opened itself (not its parent).
4. Menu popups cannot be screenshotted window-only in Xvfb (the popup is a separate override-redirect window; the main-window capture shows a black rectangle where it overlaps). Not a defect in the app.

## Not verified / needs real hardware

GTK4/WebKitGTK 6.0 desktop build (container sandbox limit), real GNOME/KDE file chooser and portal behaviour, whether the file manager Exec route selects the file, hardware accelerated WebKit, HiDPI.

## Relaxed retest (2026-11-09, user-approved)

Run with `--security-opt seccomp=unconfined` only (`linux/retest-relaxed.sh`). Still fails: `bwrap: Creating new namespace failed: Operation not permitted`, then `Failed to fully launch dbus-proxy`. Seccomp alone is not enough for WebKitGTK 6.0 bubblewrap in Docker Desktop. A further relaxation probe (AppArmor unconfined, CAP_SYS_ADMIN) was refused by the safety checks and not run. The GTK4 path remains unverified in the container; check on real Linux.

## Sandbox-on retest as a normal user (2026-11-09, user-approved)

With `seccomp=unconfined` plus `systempaths=unconfined` (Docker masks parts of /proc, which made bwrap fail with "Can't mount proc on /newroot/proc"), and the app run as an unprivileged user with its own XDG_RUNTIME_DIR (`linux/retest-webkit-user.sh`): the WebKitGTK 6.0 sandbox starts, no bwrap errors, the window opens and the process stays alive. The window content captured black, also with LIBGL_ALWAYS_SOFTWARE=1 and WEBKIT_DISABLE_DMABUF_RENDERER=1, so rendering on the GTK4 path is NOT verified in the container (likely Xvfb/Mesa, since the GTK3 path rendered normally). Check rendering on real Linux.
