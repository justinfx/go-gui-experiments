# MIQT (Qt 6 Widgets via cgo) on Linux (container run, linux/arm64, Ubuntu 24.04, Qt 6.4.2)

Harness: `linux/`. Logs and screenshots: `results/miqt/linux/` (`x11/` Xvfb+openbox, `sway/` sway headless Wayland, `wayland/` weston headless).
Timings and visual sharpness not measured. The Linux Qt here is the distro's 6.4.2 (MIQT supports Qt 6.4+, `github.com/mappu/miqt@v0.15.0/README.md:13`), not the 6.12 the macOS
build used, so Qt-version behaviour differences are possible.

## Build and tests

| Step | Result |
|---|---|
| `make build` (`go build`, cgo, `qt6-base-dev`) | PASS, 30 MB, 2 min 25 s cold on 8 cores (`make-build.log`). Makefile `QTBIN` uses `brew`, unused on Linux; nothing changed |
| `make test` (`QT_QPA_PLATFORM=offscreen`) | PASS, `GGUIX_TESTS passed=18 failed=0 not_feasible=0` (`make-test.log`) |
| `make cross-linux` equivalent (native here) | the earlier amd64-emulated cross build timed out at 30 min; a native arm64 build takes about 2.5 min, so that was an emulation problem, not a MIQT one |

## Runtime matrix

| Check | X11 (Xvfb + openbox) | Wayland (sway headless) | Wayland (weston headless) |
|---|---|---|---|
| Starts, shows a window | yes, 1280x800, title `GGUIX - MIQT`, WM_CLASS `gguix-miqt` (`x11-start.png`) | yes, native `xdg_shell`, `app_id=gguix-miqt`, `QT_QPA_PLATFORM=wayland` (`wayland-start.png`) | process alive for 12 s; window not verified (no screenshot or input possible) |
| Menu bar | native Qt `QMenuBar` drawn inside the window (File Edit View Help); Qt has no global-menu service in this session, menu popups are separate Qt windows | same | n/a |
| Keyboard shortcuts | Ctrl+E (export dialog), Ctrl+2/Ctrl+1, Ctrl+Shift+C, Ctrl+I, Ctrl+Q (process exits) all fire | same | n/a |
| Open/save dialog | **Qt's own `QFileDialog`** (a separate top-level window `Export CSV`, 629x415, Qt-widget styled, not GTK, not a portal: 0 `portal.FileChooser` calls) (`x11-export-dialog.png`, `wayland-export-dialog.png`). Export wrote 118,342,650 bytes | same | n/a |
| Notification | **NONE by default.** 0 `Notifications.Notify` calls. Cause chain below (no tray watcher, then an app bug) | same | n/a |
| Clipboard copy | `xclip -o -selection clipboard` returns the CSV row | `wl-paste` returns the row | n/a |
| Reveal in file manager | Reveal Config -> `QDesktopServices.OpenUrl` -> `xdg-open file:///tmp/prefs-m` (Qt shelled out to xdg-open; no portal `OpenURI`); Reveal Last Export -> core helper -> `xdg-open /root` (parent dir, not selected); no `FileManager1` | same (`xdg-open file:///tmp/prefs-ms`, `xdg-open /root`) | n/a |

## Findings

1. **Notification is silently missing on a stock Linux session (two causes).**
   a. `QSystemTrayIcon.isSystemTrayAvailable()` is false unless a `org.kde.StatusNotifierWatcher` is on the session bus (our dbus log shows Qt introspecting it and getting `The name org.kde.StatusNotifierWatcher was not provided by any .service files`, `x11/dbus.log` lines 45-48). KDE provides one; stock GNOME does not (needs the AppIndicator extension). `App.osNotify` returns without a message in that case (`apps/miqt/ui_desktop.go:343-345`), so the user gets only the status-bar text. Platform dependence, partly app design (a direct `org.freedesktop.Notifications` call would not need the tray).
   b. **App bug**: even with a watcher present (emulated by `linux/sni-watcher.py`; tray item registered), no `Notify` is sent, because `ui_desktop.go:351` passes a Go `nil` as the `QIcon` argument: `a.tray.ShowMessage3(title, body, nil, 5000)`. MIQT forwards `icon.cPointer()` (`qt6/gen_qsystemtrayicon.go:270`) and the C++ wrapper dereferences it (`gen_qsystemtrayicon.cpp:331`, `self->showMessage(..., *icon, ...)`), a null reference. With `qt.NewQIcon()` instead (patch `notify-nil-icon.patch`, applied only to the scratch copy) one `Notifications.Notify` call per export arrives (counted on the bus) and dunst shows it. The miqt-gotth app already uses `qt.NewQIcon()`. This path was unverifiable on macOS (the capability note said "unverified visually").
2. Dialogs are Qt widgets rather than GTK/portal dialogs: on GNOME a user sees a Qt-styled dialog without the GTK places sidebar and recent files. Using the xdg-desktop-portal Qt platform theme would change that; it was not installed or tested here.
3. Reveal Config via Qt works with a `file://` URL and opens the folder itself; Reveal Last Export (core helper) opens the parent directory only. Neither selects a file.
4. The Linux binary needs the Qt 6 shared libraries installed on the target (`libqt6widgets6` etc.); it is not self-contained.
5. weston headless: Qt starts but cannot be inspected (no seat, no capture) so it is only a "does not crash" result. Container limit.

## Not verified / needs real hardware

Qt 6.12 (the macOS version) on Linux, KDE with a real tray watcher, GNOME global menu / appmenu, portal file dialogs, fractional scaling on Wayland, hardware GL (the Widgets app does not need it).

## Re-check after fix (apps/miqt now passes `qt.NewQIcon()` to `ShowMessage3`)

PASS. Fresh `prep.sh` copy, build and `make test` (18/18) pass. With `linux/sni-watcher.py` running, one `org.freedesktop.Notifications.Notify` call (app_name `GGUIX`, `Export complete`, `1,000,000 rows written`) reaches the session bus and dunst shows it, on X11 and on sway. Without a watcher (stock GNOME) the notification is still skipped, by design (`ui_desktop.go:343-345`). miqt-fit and miqt-fitcap: build PASS, `make test` 18/18 each.
History: the original finding 1b above (nil icon, no Notify even with a watcher) was the earlier failure; status is now FIXED. Finding 1a (no tray watcher, no notification) is unchanged.
