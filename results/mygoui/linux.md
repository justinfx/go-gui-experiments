# MyGo v0.9.0, native UI mode (apps/mygoui) on Linux (container run, linux/arm64, Ubuntu 24.04)

Harness: `linux/`. Logs and screenshots: `results/mygoui/linux/` (`x11/` Xvfb+openbox, `sway/` sway headless Wayland, `wayland/` weston headless).
Timings and visual sharpness not measured. Screenshots are of the app window only (X11 popup menus were captured by their own window id).

## How it runs on Linux (read this first)

No webview: `mygo.json` has `"webview": false`. MyGo builds with `CGO_ENABLED=0` (binary 9.8 MB, `ldd`: only libc, libdl, libpthread). At run time the library opens GTK 3, GDK, Cairo and GLib/GIO through purego
(`internal/linux/ffi.go:545-566`), creates a `GtkWindow` for the title bar, menu bar and dialogs, and draws the UI itself into a surface. Without a GPU it falls back to CPU drawing and logs
`mygo: native UI draws without the GPU: there is none` (`internal/linux/glprobe.go:41`); this is expected in the container (llvmpipe, no `/dev/dri`) and the UI rendered correctly (`x11-start.png`).
A target machine needs GTK 3 installed (no WebKitGTK is needed for this mode).

## Build and tests

| Step | Result |
|---|---|
| `make deps` | PASS (`go mod download`, the CLI is pinned by the `tool` line in `go.mod`) |
| `make build` as shipped | **FAILS on Linux (app Makefile bug)**: `mygo build` succeeds and writes `build/linux-arm64/gguix-mygoui`, then the Makefile prints `mygo build produced no executable` and exits 1. `apps/mygoui/Makefile:26` globs `$(OUT)/linux-*/GGUIX-mygoui` (capital letters, the name in `mygo.json`) but MyGo writes the lower-case executable `gguix-mygoui` on Linux (the macOS bundle keeps the capitalised name). Only the Linux branch of the glob is wrong; nothing in the app code is affected |
| `make build` with the glob fixed in the scratch copy (`makefile-linux-exe-name.patch`, one word) | PASS (`make-build.log`), ELF aarch64, `bin/gguix-mygoui`. Also outputs `.deb`, `.rpm`, `.tar.gz`, `.desktop`, `PKGBUILD`, `install.sh` under `build/linux-arm64/`. A stale `build/linux-amd64/` from an earlier `make cross-linux` would also match `linux-*` and win `head -1`, so the glob should name the architecture or the stale directory be removed (I removed it in the scratch copy) |
| `make test` (`run_tests.sh`, `go test`, the `ui.NewTester` harness, no browser) | PASS, `GGUIX_TESTS passed=12 failed=0 not_feasible=0` (`make-test.log`) |

## Runtime matrix

| Check | X11 (Xvfb + openbox) | Wayland (sway headless) | Wayland (weston headless) |
|---|---|---|---|
| Starts, shows a window | yes, 1280x800 frame, title `GGUIX - MyGo`, WM_CLASS `gguix-mygoui` (`x11-start.png`) | yes, native `xdg_shell`, `app_id=gguix-mygoui`, 1280x800 (`wayland-start.png`) | process alive after 12 s, but GDK prints `gdk_seat_get_keyboard: assertion 'GDK_IS_SEAT (seat)' failed` and more seat assertions (`wayland/app.log`); window not inspectable. Container limit (no `wl_seat`) |
| Menu bar | **native GtkMenuBar** with File, Edit, View, Help (the in-window strip is a real GTK widget; its popups are separate X windows, `x11-file-menu-popup.png`; on sway the popup is inside the window rectangle, `wayland-file-menu.png`). File has Export CSV, Import Notes, Reveal Last Export and Quit. Library: `internal/linux/ffi.go:755`, `menu.go:221-262` | same | n/a |
| Keyboard shortcuts | Ctrl+2 (cards, `x11-ctrl2-cards.png`), Ctrl+1, Ctrl+I (host dialog), Ctrl+E (save dialog), Ctrl+O (import dialog), Ctrl+Shift+C (copy row), **Ctrl+Q exits** | Ctrl+2, Ctrl+1, Ctrl+I, Ctrl+E, Ctrl+O, Ctrl+Shift+C fire. **Ctrl+Q does NOT quit** (see finding 1); File > Quit by mouse does quit | n/a |
| Open/save dialog | **native GTK3 `GtkFileChooserNative`**, separate dialog window (`_NET_WM_WINDOW_TYPE_DIALOG`, transient for the main window), titles `Export CSV` and `Import Notes`, filter `CSV` / `Text` (`x11-export-dialog.png`, `x11-import-dialog.png`, `wayland-export-dialog.png`, `wayland-import-dialog.png`). No portal method call on the bus (the container has no portal backend). Export wrote 118,342,650 bytes. The `wayland-import-dialog.png` crop includes a dunst toast from the earlier notification | same | n/a |
| Notification | **YES**: `org.freedesktop.Notifications.Notify`, app_name `GGUIX-mygoui`, summary `Export complete`, body `1,000,000 rows written`; dunst displayed it (`x11/dbus.log`, `x11/dunst.log`). Sent by the library, `internal/linux/system.go:714`; the app's `mygo.NotificationsSupported()` check passes (`platform_desktop.go:41-45`), so the in-window message fallback was not needed | same call arrives (`sway/dbus.log`, `sway/dunst.log`) | n/a |
| Clipboard copy | Copy Row button and Ctrl+Shift+C: `xclip -o -selection clipboard` returns `3,SKU-0000003,Rugged Aluminum Wallet,Kitchen,1142.70,396,Active,5.0,false,,2025-12-03T14:22:00Z,Batteries not included. Batteries not included.` (cleared between the two); status bar shows `Row copied` (`x11-copy-row.png`) | `wl-paste` returns the row for both (`wayland-copy-row.png`) | n/a |
| Reveal in file manager | `mygo.Shell.ShowItemInFolder`: `org.freedesktop.FileManager1.ShowItems` is called with `file:///tmp/prefs-mu` (Reveal Config Folder) and `file:///work/apps/mygoui/gguix-export.csv` (Reveal Last Export), the file itself, so a real file manager would select it. No FileManager1 service in the container (`ServiceUnknown`), so the library falls back to opening the parent directory through GIO, not `xdg-open` (shim log empty). `internal/linux/system.go:201-208` | same (`file:///tmp/prefs-mus2`, `file:///work/apps/mygoui/gguix-export.csv`) | n/a |

## Findings (framework gap vs app bug vs container limit)

1. **Ctrl+Q does not quit under Wayland (sway), works under X11. Classification: framework gap (MyGo GTK backend), cause not isolated.** The shortcut appears only as the default accelerator of `{Role: RoleQuit}` (`apps/mygoui/menu_desktop.go:34`, label and key from `roleDefaults`, `menu.go:795`). File > Quit by mouse quits cleanly on sway, and the same Ctrl+E/O/I/1/2 accelerators work there, so the quit path itself is fine. MyGo handles role accelerators of native-UI windows specially in the key-press handler (`internal/linux/menu.go:139-146`: `roleKeys` lookup, then `gtkWindowPropagateKeyEvent`), a branch the web-frontend app should not take (it has no `w.surface`, my reading, not tested); apps/mygo's Ctrl+Q on sway works. A scratch rebuild with an explicit `Accelerator: "CmdOrCtrl+Q"` on the item made no difference, so it is not an app-side omission. I could not read the key event further (GTK built without `GTK_DEBUG`). Needs a check on real Wayland (GNOME/KDE) to rule out the headless virtual keyboard; other chords from the same keyboard work.
2. **`make build` fails on Linux, app Makefile bug** (see Build and tests). A user on Linux gets a built binary in `build/linux-arm64/` but a failing make target and no `bin/gguix-mygoui`.
3. Native menu bar, GTK file chooser, D-Bus notification, clipboard and reveal-with-select all come from the library and need no app code beyond what macOS uses. The `platformHooks` split in the app (`platform_desktop.go`) carries over unchanged.
4. Render path: no GPU in the container, so MyGo reported a CPU draw. Whether the GPU path (OpenGL/EGL via the GTK surface) works was not exercised.

## Not verified / needs real hardware

GPU drawing path, HiDPI and fractional scaling, a real GNOME/KDE session (portal chooser, global menu, Ctrl+Q on Wayland), a real file manager's handling of `ShowItems`, window size restore across launches, IME, drag and drop.
