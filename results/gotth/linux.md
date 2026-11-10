# GoTTH desktop shell (webview_go + WebKitGTK) on Linux (container run, linux/arm64, Ubuntu 24.04)

Harness: `linux/`. Logs and screenshots: `results/gotth/linux/` (`x11/` Xvfb+openbox, `sway/` sway headless Wayland,
`wayland/` weston headless). Timings and visual sharpness not measured. Build used a container-only alias, see "Container workarounds".

## Build and tests

| Step | Result |
|---|---|
| `make deps` equivalent | `templ v0.3.1100` installed with `go install`; the Makefile downloads a **macOS** Tailwind binary (`TAILWIND_ASSET` is `macos-arm64`/`macos-x64` whatever the OS), so a Linux Tailwind v4.3.5 was symlinked into `bin/tools/` in the scratch copy. Makefile limitation, not changed in the repo |
| `make build` (`templ generate`, Tailwind, `go build ./cmd/desktop`, cgo) | PASS (13 MB ELF aarch64), but only with the pkg-config alias below (`make-build.log`) |
| `make test` | PASS, `GGUIX_TESTS passed=11 failed=0 not_feasible=1` (T9 needs Node + Chrome; skipped: `server_test.go:564`) (`make-test.log`) |

## Runtime matrix

| Check | X11 (Xvfb + openbox) | Wayland (sway headless) | Wayland (weston headless) |
|---|---|---|---|
| Starts, shows a window | yes, 1280x800, title `GGUIX - GoTTH`, WM_CLASS `gguix-gotth` (`x11-start.png`) | yes, native `xdg_shell`, `app_id=gguix-gotth` (`wayland-start.png`) | process stays alive, but GDK prints `gdk_seat_get_keyboard: assertion 'GDK_IS_SEAT (seat)' failed` (no seat in headless weston); not checked further. Container limit |
| Menu bar | in-page shadcn menubar (GGUIX File Edit View Help in the page); `installNativeMenu` returns false (`cmd/desktop/native_osa.go`) | same | n/a |
| Keyboard shortcuts | page keydown handler: Ctrl+E, Ctrl+2, Ctrl+Shift+C, Ctrl+I fire; Ctrl+Q exits the process | same | n/a |
| Open/save dialog | **BROKEN**: Export CSV and Import Notes show an in-page `Error: exec: "osascript": executable file not found in $PATH` (`x11-ctrl-e.png`, `x11-import-notes.png`). No dialog of any kind | same (`wayland-ctrl-e.png`) | n/a |
| Notification | **Not reachable**: an export cannot start (save dialog fails). The completion path is `osaNative.Notify` = `osascript display notification` (`cmd/desktop/native_osa.go`), so it would fail too. 0 `Notify` calls on the bus | same | n/a |
| Clipboard copy | works: `xclip -o -selection clipboard` returns the CSV row. NOT through the native path: `/native/copy` runs `pbcopy`, which does not exist; the page's `navigator.clipboard.writeText` (second call in `static/desktop/desktop.js:17-20`) is what worked, and the "Row copied" flash shows regardless | works: `wl-paste` returns the row | n/a |
| Reveal in file manager | Reveal Config Folder -> `xdg-open /tmp/prefs-g` (`x11/xdg-open.log`); no `FileManager1`; Reveal Last Export is disabled because no export can finish | `xdg-open /tmp/prefs-gs` | n/a |

## Findings

1. **App gap (not framework): the default desktop shell is macOS-only for dialogs, notifications and clipboard.**
   `cmd/desktop/native_osa.go` (build tag `!(darwin && native)`, so it is also the Linux build) implements `SaveDialog`, `OpenDialog`,
   `Notify` with `osascript` and `CopyText` with `pbcopy`. On Linux the export and import features are unusable and notifications can never
   fire. webview_go itself offers no dialog, notification or menu API, so a Linux implementation would have to be written
   (zenity/portal for dialogs, `notify-send` or D-Bus for notifications, `xclip`/`wl-copy` for the clipboard). Compare `gotth-glaze`,
   which uses glaze's native GTK dialogs on Linux.
2. webview_go requires the `webkit2gtk-4.0` pkg-config module: `github.com/webview/webview_go@v0.0.0-20240831120633-6173450d4dd6/webview.go:9`
   (`#cgo linux ... pkg-config: gtk+-3.0 webkit2gtk-4.0`). Ubuntu 24.04 and Debian 13 ship only 4.1, so a stock `go build` fails there
   (the same reason the earlier cross build had to use Debian bookworm). The bundled header already prefers 4.1 at run time
   (`libs/webview/include/webview.h:1418-1419`), so the compile-time name is the only blocker. Framework limit (stale pin).
3. Reveal: core `host.RevealInFileManager` runs `xdg-open <dir>` on Linux (`core/host/native.go:55-76`); it opens the folder and cannot select a file.
4. The clipboard result above hides a latent bug: the native path fails silently (error discarded) and the UI still says "Row copied".
5. The WebKitGTK sandbox (bwrap) started fine in this container for the 4.1 build (no relaxation needed). The Wails (GTK4, WebKitGTK 6.0) app was different, see its report.

## Container workarounds (not app changes)

- `webkit2gtk-4.0.pc` is a copy of the 4.1 pc file (Dockerfile). Source and Makefile untouched.

## Not verified / needs real hardware

Behaviour with a real desktop (GNOME/KDE), the xdg-desktop-portal GTK file chooser, hardware-accelerated WebKit, HiDPI, IME.

## Re-check after fix (shell now uses ncruces/zenity v0.10.15, gen2brain/beeep v0.11.3, atotto/clipboard v0.1.4)

PASS on X11 and sway for the base app. Fresh `prep.sh` copy, build and `make test` (11 pass, 1 not feasible) pass. gotth-fit and gotth-fitcap: build PASS, `make test` same result each.
Runtime dependency added: the `zenity` binary must be installed on the target. Source check: zenity looks for `qarma`, `zenity`, then `matedialog` with `exec.LookPath` (`github.com/ncruces/zenity@v0.10.15/internal/zenutil/run_unix.go:21-22`; kdialog is not in that list); atotto/clipboard needs `wl-copy`+`wl-paste` when `WAYLAND_DISPLAY` is set, else `xclip` or `xsel` (`clipboard@v0.1.4/clipboard_unix.go:50-75`); beeep sends via D-Bus and falls back to `notify-send` (`beeep@v0.11.3/notify_unix.go`). The image now installs `zenity`.

| Check | X11 | Wayland (sway) |
|---|---|---|
| Export opens a dialog and writes a CSV | PASS: zenity GTK save dialog, window `Export CSV` (`x11-export-dialog-zenity.png`); file written, 118,342,650 bytes | PASS: dialog toplevel `app_id=zenity` (`wayland-export-dialog-zenity.png`), file written |
| Import opens a dialog | PASS: window `Import Notes` (`x11-import-dialog-zenity.png`); selecting a file and the resulting import was not completed in my script, so only the dialog opening is verified | not run |
| Notification | PASS: `Notifications.Notify`, app_name `DefaultAppName`, `Export complete`, `1,000,000 rows written`; dunst displayed it | PASS, same |
| Copy Row | PASS: `xclip -o` returns the row (the page Clipboard API also runs, so this does not isolate the native path) | PASS: `wl-paste` returns the row (same caveat) |
| Reveal | unchanged (core `xdg-open <dir>`), not re-run | not re-run |

Observation: beeep uses the fixed app name `DefaultAppName` unless `beeep.AppName` is set, so notifications are attributed to "DefaultAppName" on the desktop.
History: the original finding 1 above (osascript/pbcopy on Linux, export and import broken, notification unreachable) was the earlier failure; status is now FIXED for dialogs, notification and clipboard.
