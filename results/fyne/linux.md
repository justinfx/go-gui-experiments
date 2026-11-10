# Fyne v2.8.2 on Linux (container run, linux/arm64, Ubuntu 24.04)

Harness: `linux/` (Dockerfile, run.sh, lib.sh, wlinput.py). Logs and screenshots: `results/fyne/linux/`
(`x11/` = Xvfb+openbox session, `sway/` = sway headless Wayland session, `wayland/` = weston headless session).
Timings and visual sharpness were not measured (software rendering in a container).

## Build and tests

| Step | Result |
|---|---|
| `make build` (plain `go build`, cgo, no tags) | PASS, ELF aarch64, no changes needed |
| `make test` | PASS, `GGUIX_TESTS passed=12 failed=0 not_feasible=0` (`make-test.log`) |

## Runtime matrix

| Check | X11 (Xvfb + openbox) | Wayland (sway headless) | Wayland (weston headless) |
|---|---|---|---|
| Starts, shows a window | yes, 1280x800, title `GGUIX - Fyne`, WM_CLASS `GGUIX - Fyne` (`x11-start.png`) | yes, native `xdg_shell` toplevel, `app_id=io.gguix.fyne`, no XWayland (`wayland-*.png`) | NO: SIGSEGV in `glfwInit` (container limit, see below) |
| Menu bar | in-window, Fyne-drawn (File Edit View Help in the top strip); no global menu export | same | n/a |
| Keyboard shortcuts | Ctrl+E (export dialog), Ctrl+2 (cards), Ctrl+1, Ctrl+Shift+C (copy row), Ctrl+I (host), Ctrl+Q (quit, process exits) all fire via xdotool | same set fire via wlinput virtual keyboard | n/a |
| Open/save dialog | Fyne-drawn in-window `Save File` dialog (`x11-export-dialog.png`); not GTK, not a portal | same (`wayland-export-dialog.png`) | n/a |
| Notification | `org.freedesktop.Notifications.Notify` arrives; app_name `io.gguix.fyne`, summary `Export complete`, body `1,000,000 rows written`; dunst displayed it (`x11/dbus.log` line 745, `x11/dunst.log`) | same call arrives (`sway/dbus.log` line 745, `sway/dunst.log`) | n/a |
| Clipboard copy | `xclip -o -selection clipboard` returns the CSV row (`1,SKU-0000001,Deluxe Silicone Lamp,...`) | `wl-paste` returns the row | n/a |
| Reveal in file manager | `xdg-open /tmp/prefs-fy2` (config) and `xdg-open /root` (parent dir of the last export); no `org.freedesktop.FileManager1` call; the file is NOT selected | identical | n/a |

## Evidence

Notification call as seen by `dbus-monitor` (`results/fyne/linux/x11/dbus.log`):

```
method call ... destination=org.freedesktop.Notifications ... member=Notify
   string "io.gguix.fyne"   uint32 0   string ""
   string "Export complete" string "1,000,000 rows written"
```

Reveal (xdg-open shim log, `x11/xdg-open.log`): `xdg-open /tmp/prefs-fy2`, `xdg-open /root`. The shim only records the argument,
because the container has no file manager. Export wrote 1,000,002 lines (118,342,650 bytes) to `/root/gguix-export.csv`.

## Findings (framework limit vs app vs container)

1. Notifications: native on Linux. Fyne sends the D-Bus Notify call itself, `fyne.io/fyne/v2@v2.8.2/app/app_xdg.go:62-82`
   (session bus, `org.freedesktop.Notifications.Notify`). With `-tags flatpak` it uses the portal instead (`app_xdg.go:63`).
2. Menu bar: framework limit, in-window only. Native menus are compiled only on macOS: `internal/driver/glfw/menu_other.go:1`
   (`//go:build !darwin || no_native_menus`) is the non-mac implementation.
3. File dialogs: framework-drawn on Linux. `dialog/file_xdg_notflatpak.go:5-11` returns `false` from `fileOpenOSOverride` and
   `fileSaveOSOverride` (no OS override), so Fyne's own dialog is used. The portal (OS) override exists only in the flatpak build,
   `dialog/file_xdg_flatpak.go:70`. A normal Linux user therefore gets a non-GTK dialog, unlike a native GTK/Qt app.
4. Reveal: our app calls core `host.RevealInFileManager`, which on Linux runs `xdg-open <parent dir>` (`core/host/native.go:55-76`).
   It cannot select the file. This is a core/app limit, not Fyne: Fyne has no reveal API. Degraded vs macOS (`open -R` selects).
5. Wayland: Fyne chooses the platform at runtime through GLFW (`internal/driver/glfw/loop_desktop.go:19-36`); the default build (no
   `wayland`/`x11` tags) supports both (`window_x11wayland.go:1`), no build tag was needed.
6. weston headless cannot host this app. weston 13 headless advertises no `wl_seat` (the global list in
   `wayland/app.log` has compositor, shm, xdg_wm_base... and no `wl_seat`), and GLFW dereferences the null seat:
   `github.com/go-gl/glfw/v3.4/glfw@v0.1.0-pre.2/glfw/src/wl_init.c:856` `wl_seat_get_version(_glfw.wl.seat)` (SIGSEGV addr=0x40 inside
   `glfwInit`). A real compositor always has a seat, so this is a CONTAINER limit (headless weston has no input devices), with a latent
   GLFW robustness bug. sway headless was used for the Wayland column instead; input there is injected with `linux/wlinput.py`
   (zwp_virtual_keyboard_v1 and zwlr_virtual_pointer_v1).

## Not verified / needs real hardware

- Hardware GL (the container used llvmpipe), HiDPI/fractional scaling, IME, real compositor window decorations (sway here has none), real GNOME/KDE notification servers, and a real file manager for reveal.
