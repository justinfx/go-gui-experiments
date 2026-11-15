# NSGo (`2345b2f`) on Linux (container run, linux/arm64, Ubuntu 24.04)

Harness: `linux/` (not modified), with the rebuilt image (xkbcli, gnome-themes-extra, xdg-desktop-portal and its gtk and gnome backends present; nothing installed ad hoc). Logs, scripts and app-window-only screenshots: `results/nsgo/linux/` (`x11/` Xvfb + openbox, `sway/` sway headless Wayland). Scripts: `x11-run.sh` (shortcuts, clipboard, Escape, Import and Export dialogs, notification), `x11-run2.sh` (identity, Reveal, theme, system theme, Ctrl+Q), `sway-run.sh`. No timings. Screenshots are of the app window or of one dialog window.

The session's `run.sh` now starts the portal itself (harness change, `linux/run.sh`): `xdg-desktop-portal` with the gtk backend as default (FileChooser, OpenURI) and the gnome backend for Settings (it runs settings-only outside gnome-shell). The xdg-desktop-portal-gtk 1.15 backend no longer provides Settings. With that, `org.freedesktop.portal.Settings` reads `color-scheme` and reports changes live (`x11/portal-before.txt`, `x11/portal-after.txt`: 0, then 1).

## How it runs on Linux

NSGo's Linux backend draws with GTK 4, Pango and Cairo loaded at run time with purego (no cgo). The window is a normal GTK window with a GTK menu bar inside it; stock bars, stars and cards are NSGo views. `make build` is `CGO_ENABLED=0 go build` (with NSGo's identity ldflags); the binary's `ldd` lists only `libc`, `libdl`, `libpthread` and the loader (`ldd.log`). `/proc/<pid>/maps` shows 94 shared objects (`x11-maps.log`, `sway-maps.log`): `libgtk-4`, `libcairo*`, `libpango*`, `libharfbuzz`, `libfontconfig`, `libgio`/`libglib`/`libgobject`, `libgraphene`, `libepoxy`, the X11 libraries, `libEGL`/`libgallium` (llvmpipe) and `libdconfsettings`. File dialogs, Open and notifications use D-Bus (desktop portal, `org.freedesktop.Notifications`).

The target needs GTK 4 with Pango/Cairo, a portal backend with a FileChooser (and OpenURI for NSGo's `OpenFile`), and a notification daemon.

## Build and tests

| Step | Result |
|---|---|
| `make build` in the container | PASS, ELF aarch64 (`make-build.log`) |
| `make test` (headless) | PASS, `GGUIX_TESTS passed=12 failed=0 not_feasible=0` (`make-test.log`) |

## Runtime matrix

| Check | X11 (Xvfb + openbox) | Wayland (sway headless) |
|---|---|---|
| Starts and renders | yes, 1280 x 800, `GGUIX - NSGo`, GTK menu bar (`x11-start.png`). The container's default font is a serif face, so labels are wider than on macOS | yes, native Wayland, `xdg_shell` (`sway-start.png`) |
| Window identity | `WM_CLASS` is `dev.gguix.nsgo` (`x11/winfo.txt`). Before the review pass it was empty: NSGo sets no program name, so the app sets it (`identity_desktop.go`, `g_set_prgname` through purego) | `app_id` is `dev.gguix.nsgo` (`sway/tree.txt`), previously `GTK Application` |
| Menu bar | File, Edit, View, Help (GTK) | same |
| Shortcuts | Ctrl+2 (cards, `x11-ctrl2-cards.png`), Ctrl+1, Ctrl+I, Ctrl+E, Ctrl+O, Ctrl+Shift+C fired. Ctrl+Q: exited in an earlier run, **still running** in the last `x11-run2.sh` run (`x11/ctrlq.txt`); xdotool sends keys to the focused window and the focus after the popup clicks was not verified, so this is inconclusive | the same set fired; Ctrl+Q exited (`sway/ctrlq.txt`) |
| Escape | closes the Host dialog (`x11-host-dialog.png`, `x11-host-after-escape.png`) | closes it (`sway-host-dialog.png`, `sway-host-after-escape.png`). My first attempt sent the wrong key name to `wlinput.py` (`esc`, not `Escape`) |
| Save dialog (Export) | the portal's GTK chooser, a separate dialog window, `WM_TRANSIENT_FOR` the app window, type DIALOG, name pre-filled `gguix-export.csv` (`x11-export-dialog.png`, `x11/export-dialog-props.txt`). Typing `/tmp/export-out/out.csv` wrote it: 2,341,616 bytes for 20,000 rows (`x11/export-files.txt`) | not run |
| Open dialog (Import Notes) | opens filtered to `.txt` (`x11-import-dialog.png`); after typing `/tmp/notes.txt` (`x11-import-dialog-typed.png`) the Notes field holds `hello notes from a text file` and the form shows `Unsaved changes` (`x11-after-import.png`, kept). A first attempt with fast typing produced `notes.txtxt` through the entry's completion and the app showed the error dialog `open /tmp/notes.txtxt: no such file or directory`, which is correct behavior | not run |
| Dialog failure mode | without a working FileChooser backend (the default session config here) the call fails and the app sees a cancel: nothing is shown. This is NSGo's behavior (a portal error is not surfaced). Not seen with the gtk backend | |
| Notification | yes: `org.freedesktop.Notifications.Notify`, `app_name` `dev.gguix.nsgo` (the id), summary `Export complete`, body `20,000 rows written`, shown by dunst (`x11/notify.txt`, `x11/dunst.log`). The name was empty before the identity ldflags | not run |
| Clipboard | `xclip -o` returns the item's CSV row (`x11/clipboard.txt`) | `wl-paste` returns it (`sway/clipboard.txt`) |
| Reveal | Reveal Last Export (core helper): `xdg-open /tmp/export-out`. Reveal Config Folder: NSGo's `cx.OpenFile` on a folder needs the `OpenURI` portal, which is missing (`x11/dbus.log`: `No such interface org.freedesktop.portal.OpenURI`) and reported nothing; the app now falls back to the core helper and ran `xdg-open /tmp/prefs-gx` (`x11/xdg-open.log`, `x11-host-after-reveal.png`) | not run |
| Theme | Dark from the select works and `prefs.json` has `"theme":"dark"` (`x11-theme-dark.png`, `x11/prefs-after-dark.json`) | not run |
| System light/dark | **follows the GTK theme, not `color-scheme` alone.** NSGo reads GTK's `gtk-application-prefer-dark-theme` and treats a theme name containing "dark" as dark (`internal/platform/linux/env.go`); it uses `gtk-interface-color-scheme` only on GTK 4.20 and later. This image has GTK 4.14.5 (`sway/gtk4-version.txt`). Setting only `color-scheme prefer-dark` (what the stock GNOME toggle writes) left the window light, at launch and live (`x11-dark-at-start.png`, `x11-system-dark.png`, `sway-system-dark.png`). Setting `gtk-theme Adwaita-dark` as well (what Ubuntu's toggle also does, with Yaru-dark) gave a dark window at launch, and switching back to Adwaita turned it light live (`sway-dark-theme-at-start.png`, `sway-light-theme-live.png`, script `sway-dark-theme.sh`) | X11: `color-scheme` only, light (no XSETTINGS daemon to carry the theme name, not tested further) |

## Findings (framework vs app vs container)

1. **File dialogs and Open need a working desktop portal and fail silently without one. Framework behavior.** With the gtk backend they are correct (transient, pre-filled name, `.txt` filter).
2. **`cx.OpenFile` on a folder reports nothing when the OpenURI portal is missing.** The app falls back to the core helper's `xdg-open`.
3. **Window identity is not set by NSGo on Linux.** Worked around in the app (ldflags plus `g_set_prgname`). This adds `github.com/ebitengine/purego` as a direct dependency (it is NSGo's own, same version).
4. **GLib-GObject-CRITICAL warnings on stderr** (`g_value_get_int: assertion 'G_VALUE_HOLDS_INT (value)' failed`), a few per interaction (`x11/app.log`). The app works. Not isolated.
5. **System light/dark follows the GTK theme name; on GTK before 4.20 it ignores `color-scheme` alone.** Framework behavior: a stock GNOME dark toggle on an older GTK leaves the app light, Ubuntu's toggle (which also swaps the theme) works.
6. Container limits: Mesa llvmpipe only, a serif default font, no gnome-shell; the harness' `wlinput.py` key names differ from xdotool's.

## Image gaps for the coordinator

Everything needed was in the image. The portal Settings gap was a session setup gap, fixed in `linux/run.sh` (see above).

## Not verified / needs real hardware

GPU drawing, HiDPI, a real GNOME or KDE session (theme following, portal behavior, notification attribution), the `nsgo build` Linux packages (`.deb`, AppImage) and their desktop entry, window size restore across launches, IME, Weston headless (not run).
