# MyGo v0.9.0, web-frontend mode (apps/mygo) on Linux (container run, linux/arm64, Ubuntu 24.04)

Harness: `linux/`. Logs and screenshots: `results/mygo/linux/` (`x11/` Xvfb+openbox, `sway/` sway headless Wayland, `wayland/` weston headless).
Timings and visual sharpness not measured. Screenshots are of the app window only (popup menus on X11 are separate windows and were captured by their own window id).

## How it runs on Linux (read this first)

MyGo builds with `CGO_ENABLED=0`; the binary (10.1 MB, `ldd`: only libc, libdl, libpthread) opens GTK 3, GDK, Cairo, GLib/GIO and WebKitGTK at run time through purego
(`internal/linux/ffi.go:545-566` for `libgtk-3.so.0`, `libgdk-3.so.0`, `libcairo.so.2`, `libgio-2.0.so.0`; `:902-911` for `libwebkit2gtk-4.1.so.0`, falling back to `4.0`).
It uses the GTK3 / WebKitGTK 4.1 stack, not GTK4 / WebKitGTK 6.0, so the bubblewrap sandbox problem that blocks Wails' default GTK4 build in this container did not occur: the page rendered with default Docker seccomp.
A target machine needs `libgtk-3-0`, `libwebkit2gtk-4.1-0` (and the usual GLib/Cairo) installed; nothing is linked at build time, so a missing library shows up as a start-up error, not a build error.

## Build and tests

| Step | Result |
|---|---|
| `make deps` (`npm ci`, `go mod download`) in `/work/apps/mygo` | PASS (the mygo CLI is the npm optional package `@egoist/mygo-cli-linux-arm64`, installed by `npm ci`) |
| `make build` (`mygo build -skip-dmg`, native linux/arm64) | PASS, no changes needed (`make-build.log`). Output `build/out/linux-arm64/gguix-mygo` (ELF aarch64) plus `.deb`, `.rpm`, `.tar.gz`, `.desktop`, `PKGBUILD`, `install.sh`; the Makefile's glob (`build/out/*/gguix-mygo`) finds it and `bin/gguix-mygo` is a symlink |
| `make test` as shipped | Go tests and vitest pass, but the 14 Playwright e2e tests FAIL: `Chromium distribution 'chrome' is not found at /opt/google/chrome/chrome` (`frontend/playwright.config.ts:18` uses `channel: "chrome"`; Google Chrome has no Linux arm64 build). `GGUIX_TESTS passed=20 failed=14` (`make-test.log`). Container/arch limit, not an app bug; identical to Wails |
| `make test` with Playwright's bundled Chromium (`channel` removed in the scratch copy only, `playwright-no-chrome-channel.patch`; `npx playwright install chromium`) | `GGUIX_TESTS passed=34 failed=0 not_feasible=0` (`make-test-chromium.log`). These tests drive the test-only HTTP harness (`main_e2e.go`), not the desktop shell |

## Runtime matrix

| Check | X11 (Xvfb + openbox) | Wayland (sway headless) | Wayland (weston headless) |
|---|---|---|---|
| Starts, shows a window | yes, 1280x800 frame, title `GGUIX - MyGo`, WM_CLASS `gguix-mygo` (`x11-start.png`) | yes, native `xdg_shell`, `app_id=gguix-mygo`, 1280x800 (`wayland-start.png`) | process alive after 15 s, but GDK prints `gdk_seat_get_keyboard: assertion 'GDK_IS_SEAT (seat)' failed` and more seat assertions (`wayland/app.log`); window not inspectable. Container limit (weston headless has no `wl_seat`) |
| Menu bar | **native GtkMenuBar** above the web view: GGUIX, File, Edit, View, Help; popups are separate X windows (`x11-file-menu-popup.png`). Library: `gtk_menu_bar_new` bound at `internal/linux/ffi.go:755`, built in `internal/linux/menu.go:221-262`. Linux gets the app's first menu too ("GGUIX" with About and Quit) | same; the popup is part of the window rectangle | n/a |
| Keyboard shortcuts | Ctrl+2 (cards, `x11-ctrl2-cards.png`), Ctrl+1, Ctrl+I (host dialog, `x11-host-dialog.png`), Ctrl+E (save dialog), Ctrl+O (import dialog), Ctrl+Shift+C (copy row), Ctrl+Q (process exits) all fire | same set fire, Ctrl+Q exits the process. Ctrl+O (import) and Ctrl+Shift+C / Copy Row only act when a row is selected (observed on X11: Ctrl+O did nothing until a row was clicked; app behaviour, not a Linux difference that was checked) | n/a |
| Open/save dialog | **native GTK3 `GtkFileChooserNative`**, a separate dialog window (`_NET_WM_WINDOW_TYPE_DIALOG`, transient for the main window), title `Save` for export and `Import Notes` for import (`x11-export-dialog.png`, `x11-import-dialog.png`, `wayland-export-dialog.png`, `wayland-import-dialog.png`). No `org.freedesktop.portal.FileChooser` method call appears on the bus (the four matches in `dbus.log` are GTK's own name watches). The container has no portal backend, so the GTK dialog is what a user without a portal gets. Library: `gtk_file_chooser_native_new` (`ffi.go:776`), `internal/linux/dialog.go`. Export wrote 118,342,650 bytes | same | n/a |
| Notification | **YES**: `org.freedesktop.Notifications.Notify`, app_name `GGUIX - MyGo`, summary `Export complete`, body `1,000,000 rows written`; dunst displayed it (`x11/dbus.log`, `x11/dunst.log`). Sent from the library over D-Bus (`internal/linux/system.go:714`), no bundle needed (the macOS-only bundle requirement in the app, `platform_desktop.go:63-70`, does not apply) | same call arrives (`sway/dbus.log`, `sway/dunst.log`) | n/a |
| Clipboard copy | Copy Row button and Ctrl+Shift+C: `xclip -o -selection clipboard` returns `13,SKU-0000013,Heavy Steel Umbrella,Electronics,1638.41,755,Active,5.0,false,,2025-04-26T10:12:00Z,` (the clipboard was cleared between the two) | `wl-paste` returns the CSV row for both (row 3) | n/a |
| Reveal in file manager | `mygo.Shell.ShowItemInFolder`: a `org.freedesktop.FileManager1.ShowItems` call is made with `file:///tmp/prefs-mg` (Reveal Config Folder) and `file:///work/apps/mygo/gguix-export.csv` (Reveal Last Export), i.e. the file itself, so a real file manager that implements FileManager1 (Nautilus, Dolphin, Nemo) selects it. The container has no such service (`ServiceUnknown` error in `dbus.log`), so the library then falls back to opening the parent directory through GIO (`g_app_info_launch_default_for_uri`), which does not go through `xdg-open`; nothing was launched and the shim log stays empty. Library: `internal/linux/system.go:201-208` | same (`file:///tmp/prefs-mgs`, `file:///work/apps/mygo/gguix-export.csv`) | n/a |

## Findings

1. Native integration is the most complete in this set: native GTK menu bar, GTK file chooser, D-Bus notification, GTK clipboard, and a reveal call that can select the file. All of it comes from the library with no Linux-specific app code beyond what the macOS build already used.
2. Reveal is better than every other app here: it calls `FileManager1.ShowItems` first (file selected on a real file manager) and only opens the parent folder if that fails. Whether Nautilus/Dolphin select the file was not observed (no file manager in the container).
3. The page content is plain WebKitGTK 4.1 rendering (Svelte UI unchanged from apps/wails). Fonts and form controls look different from the macOS WKWebView build (`x11-start.png`), as expected for an unmodified web UI.
4. Runtime dependency (not a build dependency): GTK 3 and WebKitGTK 4.1 packages must be present on the user's machine. This is the "CGO means runtime linkage" question answered as: no cgo, but dlopen of the system GTK/WebKit stack.
5. Extra step, not a failure: the `mygo` CLI's Playwright e2e needs a Chromium that exists for arm64 (see Build and tests). On x86-64 Linux with Chrome installed the shipped command works unchanged.

## Not verified / needs real hardware

Real GNOME/KDE file chooser and portal path (a portal backend would take over the chooser), whether a real file manager selects the file on reveal, hardware-accelerated WebKit, HiDPI and fractional scaling, tray/AppIndicator (not used by the app), Wayland on a real compositor with decorations, IME, window size restore (`StateKey`) across launches.
