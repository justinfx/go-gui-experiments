# GoTTH + glaze (pure Go, `CGO_ENABLED=0`) on Linux (container run, linux/arm64, Ubuntu 24.04)

Harness: `linux/`. Logs and screenshots: `results/gotth-glaze/linux/` (`x11/` Xvfb+openbox, `sway/` sway headless Wayland).
Timings and visual sharpness not measured.

## Build and tests

| Step | Result |
|---|---|
| `make build` (`templ`, Tailwind, `CGO_ENABLED=0 go build ./cmd/desktop`) | PASS, 13 MB ELF aarch64. No C toolchain or GTK headers are needed; glaze `dlopen`s the libraries at run time. Same Tailwind caveat as gotth (Makefile downloads a macOS Tailwind; a Linux binary was linked in the scratch copy) |
| `make test` | PASS, `GGUIX_TESTS passed=11 failed=0 not_feasible=1` (T9 needs Node + Chrome) (`make-build-test.log`) |

## WebKit stack selection (important for what was tested)

glaze picks the stack at run time: if `libwebkitgtk-6.0.so.4` can be loaded it uses GTK4 + WebKitGTK 6.0, otherwise GTK3 + WebKitGTK 4.1/4.0
(`github.com/crgimenes/glaze@v0.0.83/webview_linux.go:226-248`). The image has both. With 6.0 present the app dies at start in this container:
`bwrap: Creating new namespace failed: Operation not permitted` -> `Failed to fully launch dbus-proxy` -> SIGTRAP (`x11-webkit6-bwrap-crash.log`).
That is a CONTAINER limit (Docker's default seccomp blocks user namespaces that WebKit's bubblewrap sandbox needs; I was not permitted to relax
seccomp or switch the sandbox off, so I did not). All results below are for the GTK3 + WebKitGTK 4.1 path, obtained by moving
`libwebkitgtk-6.0.so.4*` aside inside the throwaway container. The GTK4 path of glaze is therefore UNVERIFIED (see LINUX.md).

## Runtime matrix (GTK3 + WebKitGTK 4.1 path)

| Check | X11 (Xvfb + openbox) | Wayland (sway headless) |
|---|---|---|
| Starts, shows a window | yes, 1280x800, title `GGUIX - GoTTH`, WM_CLASS `gguix-gotth` (`x11-start.png`) | yes, native `xdg_shell`, `app_id=gguix-gotth` (`wayland-start.png`) |
| Menu bar | in-page menubar; the shell deliberately skips `glaze/menu` on Linux (`cmd/desktop/menu.go:88`), the lib draws no bar there | same |
| Keyboard shortcuts | page keydown: Ctrl+E, Ctrl+2, Ctrl+Shift+C, Ctrl+I fire; Ctrl+Q exits (verified on Wayland; on X11 the first Ctrl+Q was sent while a dialog was open and did not exit, not retried) | same |
| Open/save dialog | **real GTK3 `GtkFileChooserNative`**, a separate top-level window `Export CSV` / `Import Notes` (not a portal, no `org.freedesktop.portal.FileChooser` call; the portal is not running) (`x11-export-dialog.png`, `x11-import-dialog.png`, `wayland-export-dialog.png`). Export wrote `gguix-export.csv` (118,342,650 bytes) | same, with GTK client-side header bar. Import dialog on Wayland was not confirmed (the click did not open it; X11 did) |
| Notification | **NONE on Linux**: 0 `org.freedesktop.Notifications.Notify` calls and dunst showed nothing after a completed export. The shell swallows `ErrUnsupported` so the user sees only the in-page toast | same |
| Clipboard copy | `xclip` returns the CSV row, but via the page's `navigator.clipboard`, NOT the native path (see 3) | `wl-paste` returns the row, same caveat |
| Reveal in file manager | `xdg-open` only. Reveal Last Export -> `xdg-open /work/apps/gotth-glaze` (parent folder of the file, file not selected); Reveal Config Folder -> `xdg-open /tmp` which is the PARENT of the config dir `/tmp/prefs-z` (see 4); no `FileManager1` call | same (`xdg-open /tmp`, `xdg-open /work/apps/gotth-glaze`) |

## Findings

1. Native file dialogs work on Linux with no C code (purego into GTK): `github.com/crgimenes/glaze@v0.0.83/dialog_linux.go:1-4` ("GtkFileChooserNative + gtk_native_dialog_run"),
   binding at `dialog_linux.go:91`. This is the best dialog result among the web-view apps (gotth, wails untested, miqt-gotth untested here).
2. Notifications are a gap on Linux (framework): `github.com/crgimenes/native@v0.1.18/notify/notify_other.go:1,5` (`//go:build !darwin`, `send` returns `ErrUnsupported`). Matches the capabilities note.
3. Native clipboard is a gap on Linux (framework): `native@v0.1.18/clipboard/clipboard_linux.go:3-4,28` ("NOT YET IMPLEMENTED", `writeText` returns `ErrUnsupported`).
   Copy still works because `static/desktop/desktop.js:17-20` also calls `navigator.clipboard.writeText` after the native POST; the native error is ignored.
4. Reveal on Linux is lossy and slightly wrong for folders: `native@v0.1.18/openurl/openurl_linux.go:26-28` `revealFile` always opens `filepath.Dir(absPath)`. Our
   `handleRevealConfig` passes a directory, so Linux opens its parent (`/tmp` instead of `/tmp/prefs-z`), whereas core `host.RevealInFileManager` (`core/host/native.go:70-73`)
   handles directories correctly. App-level fix: use the core helper for directories. Neither selects a file; macOS `open -R` does.
5. Linux has no `glaze/menu` bar, so the in-page menu is the only menu; consistent with the capabilities note.
6. The pure-Go claim holds at build time (no cgo), but the binary still needs GTK3 + WebKitGTK (4.1 or 4.0) or GTK4 + WebKitGTK 6.0 installed at run time.

## Not verified / needs real hardware

GTK4/WebKitGTK 6.0 path (blocked here by the container sandbox limit), xdg-desktop-portal file chooser behaviour (portal absent), Wayland import dialog, a real file manager, HiDPI.

## Relaxed retest (2026-11-09, user-approved)

Run with `--security-opt seccomp=unconfined` only (`linux/retest-relaxed.sh`). Still fails: `bwrap: Creating new namespace failed: Operation not permitted`, then `Failed to fully launch dbus-proxy`. Seccomp alone is not enough for WebKitGTK 6.0 bubblewrap in Docker Desktop. A further relaxation probe (AppArmor unconfined, CAP_SYS_ADMIN) was refused by the safety checks and not run. The GTK4 path remains unverified in the container; check on real Linux.

## Sandbox-on retest as a normal user (2026-11-09, user-approved)

With `seccomp=unconfined` plus `systempaths=unconfined` (Docker masks parts of /proc, which made bwrap fail with "Can't mount proc on /newroot/proc"), and the app run as an unprivileged user with its own XDG_RUNTIME_DIR (`linux/retest-webkit-user.sh`): the WebKitGTK 6.0 sandbox starts, no bwrap errors, the window opens and the process stays alive. The window content captured black, also with LIBGL_ALWAYS_SOFTWARE=1 and WEBKIT_DISABLE_DMABUF_RENDERER=1, so rendering on the GTK4 path is NOT verified in the container (likely Xvfb/Mesa, since the GTK3 path rendered normally). Check rendering on real Linux.
