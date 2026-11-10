# MIQT + GoTTH in Qt WebEngine on Linux (container run, linux/arm64, Ubuntu 24.04, Qt 6.4.2 / Chromium 102)

Harness: `linux/`. Logs and screenshots: `results/miqt-gotth/linux/` (`x11/` Xvfb+openbox, `sway/` sway headless Wayland, `wayland/` weston headless).
Timings and visual sharpness not measured.

## IMPORTANT: sandbox relaxation used for these results

Every runtime result below was obtained with `QTWEBENGINE_DISABLE_SANDBOX=1` in the container environment (the app log prints `Sandboxing disabled by user.`). That variable was set when the
long-lived test container was created, before I had decided against relaxing sandboxes; I only found out it was active when the log showed it. With the variable removed
(`env -u QTWEBENGINE_DISABLE_SANDBOX`) the app starts but never shows a window: `ERROR:zygote_host_impl_linux.cc(90) Running as root without --no-sandbox is not supported`
(`x11/app-sandbox-on.log`). So in this root-in-Docker container Qt WebEngine cannot run with its sandbox on; `linux/dx.sh` no longer sets it by default (opt-in: `GGUIX_NO_QT_SANDBOX=1`). Treat this app's runtime
results as "sandbox disabled, container only"; re-check on a real desktop session as a normal user.

## Build and tests

| Step | Result |
|---|---|
| `make deps` equivalent | `templ v0.3.1100` via `go install`; Linux Tailwind linked into `bin/tools` in the scratch copy (the Makefile downloads a macOS Tailwind, same as gotth) |
| `make build` (`templ`, Tailwind, `go build ./cmd/desktop`, cgo, Qt6Widgets + WebEngineWidgets + WebChannel) | PASS, 39 MB, 25 s with a warm cache (`make-build.log`). The capabilities note said a Linux build was "not feasible" with the stock miqt-docker image (no WebEngine module); with the Ubuntu packages (`qt6-webengine-dev`, `qt6-webchannel-dev`) it builds fine |
| `make test` | PASS, `GGUIX_TESTS passed=11 failed=0 not_feasible=1` (T9 browser test needs Node + Chrome) (`make-test.log`) |

## Runtime matrix (sandbox disabled, see above)

| Check | X11 (Xvfb + openbox) | Wayland (sway headless) | Wayland (weston headless) |
|---|---|---|---|
| Starts, shows a window | yes, 1280x800, title `GGUIX - GoTTH`, WM_CLASS `gguix-miqt` (`x11-start.png`); WebEngine runs in-process GPU on llvmpipe (`GL Type: desktop`) | yes, native `xdg_shell`, `app_id=gguix-miqt` (`wayland-start.png`) | process alive 12 s; window not verified |
| Menu bar | native Qt `QMenuBar` in the window (page hides its in-page menu); popups are separate Qt windows | same | n/a |
| Keyboard shortcuts | Ctrl+E (Qt save dialog), Ctrl+2/Ctrl+1, Ctrl+Shift+C, Ctrl+I, Ctrl+Q (exits) fire | same | n/a |
| Open/save dialog | **Qt `QFileDialog`** (separate window `Export CSV`, `x11-export-dialog.png`, `wayland-export-dialog.png`), not GTK/portal. Export wrote 118,342,650 bytes | same | n/a |
| Notification | **Conditional.** Without a StatusNotifierWatcher: 0 `Notify` calls (QSystemTrayIcon unavailable). With the emulated watcher (`linux/sni-watcher.py`, the KDE case): `org.freedesktop.Notifications.Notify`, app_name `GGUIX`, summary `Export complete`, body `1,000,000 rows written`, hint `urgency=0`; dunst showed it (`x11/dbus.log`, `x11/dunst.log`) | without watcher 0 calls (same code) | n/a |
| Clipboard copy | `xclip -o -selection clipboard` returns the CSV row | `wl-paste` returns the row | n/a |
| Reveal in file manager | `xdg-open /tmp/prefs-mg` (config folder itself) and `xdg-open /work/apps/miqt-gotth` (parent dir of the export); no `FileManager1`, not selected (core `host.RevealInFileManager`, `core/host/native.go:55-76`) | same (`xdg-open /tmp/prefs-mgs`, `/work/apps/miqt-gotth`) | n/a |

## Findings

1. **Degraded rendering from the old Chromium in Qt 6.4 (container-independent).** Ubuntu's `libQt6WebEngineCore` identifies as `QtWebEngine/6.4.2 Chrome/102.0.5005.177`. The app's Tailwind v4 stylesheet (`assets/css/output.css`) contains `color-mix` and `@layer`;
   `color-mix()` needs Chromium 111+ (from general browser-support knowledge, not verified in library source). The screenshots show it: segmented Table/Cards control and Save button lose their styling, and the Host Info dialog has no
   background panel and is mispositioned (`x11-host-dialog.png`). The macOS build used Qt 6.12 (a much newer Chromium), so this is a Linux-distro-Qt-version effect: a Linux target needs a current Qt (6.7+ or so) or a CSS build that avoids these features.
2. Native Qt dialogs, clipboard and shortcuts work through the same Go `qtNative` as on macOS (`cmd/desktop/native.go`), so the dialog gap seen in gotth does not exist here.
3. Notification here uses the correct non-nil icon (`ShowMessage3(title, body, qt.NewQIcon(), 5000)`, `cmd/desktop/native.go`), unlike the miqt Widgets app; it still depends on a tray watcher being present (KDE yes, stock GNOME no), otherwise `Notify` returns an error and nothing is shown.
4. Reveal opens a folder only; it cannot select the file (same core helper).
5. stderr is noisy: `Property 'redoAvailable'' of object 'QLineEdit' has no notify signal ...` repeated from the QWebChannel registration of a `QLineEdit` (shell.md explains why a QLineEdit is used). Harmless.
6. Qt WebEngine needs a large runtime install on the target (Qt6 WebEngine, NSS, ALSA etc.); the binary is not self-contained.

## Not verified / needs real hardware

Sandbox-enabled run as a normal user on a real desktop, Qt 6.7+ rendering, KDE tray, GNOME behaviour, hardware GL, portal file dialogs, HiDPI.

## Sandbox-on retest (2026-11-09, user-approved)

Run as an unprivileged user with `QTWEBENGINE_DISABLE_SANDBOX` unset and `seccomp=unconfined` (`linux/retest-relaxed.sh`). The window opens and the app stays alive; the log has no "Sandboxing disabled" line. Qt passes `--disable-setuid-sandbox` itself (it uses the namespace sandbox). Window-only screenshot: `linux/relaxed-x11-start.png`. Earlier results ran with the sandbox disabled as root; this run shows the app works with it on.
