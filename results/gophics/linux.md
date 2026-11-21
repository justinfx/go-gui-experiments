# Gophics `8d36097` (apps/gophics) on Linux (container run, linux/arm64, Ubuntu 24.04)

Harness: `linux/`. Helpers: `results/gophics/linux/check-x11.sh` (start, identity, shortcuts) and `check-full.sh` (dialogs, notification, Reveal, light/dark, clipboard), both run in each mode. Logs and app-window-only screenshots: `results/gophics/linux/x11/`, `results/gophics/linux/sway/`. Timings and visual sharpness not measured. Dataset: `GGUIX_ROWS=1000`. The status bar's `user@host` label and the Host dialog's user and host rows are blanked in the screenshots (container hostname).

**Total: 9 of 12 checks pass** (both modes). Gaps: window identity, menu bar, system light/dark.

## How it runs on Linux (read this first)

Gophics draws everything with its own pure-Go WebGPU implementation. On Linux the wgpu layer picked the **Vulkan** backend on Mesa lavapipe (`llvmpipe (LLVM 20.1.2)`, `type=CPU`), with a GLES/EGL instance also probed (`x11/app.log`, `GOPHICS_GPU_LOG=info`). Libraries are opened at run time through goffi (pure-Go dlopen), not linked:

- `ldd` lists only `libc`, `libdl`, `libpthread` and the loader (`x11/ldd.log`); the binary is a static-feeling `CGO_ENABLED=0` build.
- Loaded at run time (`x11/maps.log`, `sway/maps.log`): `libvulkan.so.1` plus every Mesa ICD it enumerates (`libvulkan_lvp` used), `libEGL`/`libEGL_mesa`/`libGLdispatch`, `libLLVM`, `libdrm*`, and on X11 `libX11`, `libX11-xcb`, `libxcb*`; on sway the Wayland client libraries.
- Subprocesses, per the framework source: `zenity`/`kdialog` for file dialogs (`shell/desktop/filepicker_unix.go`), `notify-send` for notifications (`notify_unix.go`). Reveal uses the core helper (`xdg-open` of the parent folder).

## Build and tests

| Step | Result |
|---|---|
| `CGO_ENABLED=0 go build` in the container | PASS, ELF aarch64 (`x11/file.log`) |
| `make test` | not re-run in the container (passes on macOS, 12/12, headless, no display needed) |

## Runtime matrix

| # | Check | X11 (Xvfb + openbox) | Wayland (sway headless) | Result |
|---|---|---|---|---|
| 1 | Starts, renders | yes, 1280x800, `GGUIX - Gophics` (`x11/start.png`) | yes (`sway/start.png`) | PASS |
| 2 | Window class / app_id | `WM_CLASS "gogpu", "GoGPU"` (`x11/window.log`); `app.Config.AppID` does not reach it | `app_id gogpu` (`sway/tree.json`, `sway/*-props.log`) | gap |
| 3 | Vulkan device | lavapipe (llvmpipe), Vulkan backend | same | PASS |
| 4 | Menu bar | none; gogpu has no Linux `SetMenu` backend (only darwin and windows) | none | gap |
| 5 | Shortcuts | Ctrl+2, Ctrl+Shift+C, Ctrl+I, Ctrl+O, Ctrl+E, Ctrl+Q (`x11/ctrlq.log`) | Ctrl+Shift+C, Ctrl+O, Ctrl+E fired (`check-full.sh`) | PASS |
| 6 | Clipboard (Copy Row) | `xclip` returns item 7's CSV row (`x11/clipboard-full.log`) | `wl-paste` returns it (`sway/clipboard-full.log`) | PASS |
| 7 | Open dialog (Import Notes) | zenity (GTK 4 chooser), `.txt` filter (`x11/import-dialog.png`); `/tmp/notes.txt` loads into Notes, `Unsaved changes` (`x11/after-import.png`) | same, native Wayland zenity (`sway/import-dialog.png`, `sway/after-import.png`) | PASS |
| 8 | Folder chooser (Export) | in-app name prompt (`x11/export-name-dialog.png`), then `zenity --directory` (`x11/export-folder-dialog.png`); wrote `gguix-export.csv`, 1,002 lines (`x11/export-files.log`) | same (`sway/export-folder-dialog.png`, `sway/export-files.log`) | PASS |
| 9 | Notification | `notify-send` reached `org.freedesktop.Notifications.Notify`: `Export complete`, `1,000 rows written`, shown by dunst (`x11/notify.log`, `x11/dunst-full.log`); `app_name` is `notify-send` | same (`sway/notify.log`) | PASS |
| 10 | Reveal (`xdg-open` shim) | Reveal Last Export: `xdg-open /tmp/export-out`; Reveal Config Folder: `xdg-open /tmp/prefs-gp` (`x11/xdg-open-full.log`, `x11/host-reveal.png`) | same (`sway/xdg-open-full.log`) | PASS |
| 11 | System dark at launch | portal `color-scheme` = 1 (`x11/portal-dark.log`) left the app **light** (`x11/dark-portal-at-start.png`); `GTK_THEME=Adwaita:dark` in the env gave dark (`x11/dark-gtktheme-at-start.png`) | same (`sway/dark-portal-at-start.png`, `sway/dark-gtktheme-at-start.png`) | gap |
| 12 | Live light/dark switch | flipping `color-scheme` while running: stays light (`x11/live-dark.png`) | same (`sway/live-dark.png`) | gap |

## Findings

1. **Runs CGO-free on both X11 and Wayland with Vulkan on lavapipe.** Classification: works.
2. **File dialogs, notification and Reveal work through subprocesses** (`zenity`, `notify-send`, `xdg-open`), with no portal or D-Bus code in the app. The target needs `zenity` (or `kdialog`) and `notify-send` installed. Classification: works, with a runtime dependency on those tools.
3. **The zenity dialogs are separate top-level windows**, not transient for the app window (`WM_TRANSIENT_FOR` not set, type NORMAL, title `zenity`; `x11/*-props.log`). On a real desktop they can fall behind the app. Classification: framework gap.
4. **Notifications are attributed to `notify-send`, not the app** (empty icon, `app_name notify-send`). Classification: framework gap (no `--app-name`).
5. **System light/dark ignores the portal and gsettings.** Gophics' Linux `detectDarkMode` reads only `GTK_THEME` and KDE's `kdeglobals`; neither GNOME's `color-scheme` nor Ubuntu's `gtk-theme` setting reaches it, and nothing watches for changes. A stock GNOME or Ubuntu dark toggle leaves the app light. Classification: framework gap.
6. **Window class and app_id are `gogpu`.** Classification: framework gap; no Config field sets them, so a `.desktop` file cannot match the window.
7. No menu bar on Linux; every action is reachable from toolbar buttons or shortcuts except About and Reload/Focus Search (shortcut only, About has none). Classification: framework gap plus app choice.
8. **Disabled styling (Opacity) not visible** on Save/Revert in `x11/ctrl2-cards.png` while the form was clean (macOS showed them dimmed). Classification: not isolated (renderer difference or capture timing).
9. In one X11 run, an area the zenity dialog had covered stayed black after it closed until the next repaint (not kept as evidence; later runs were clean). Classification: not isolated (lavapipe, no compositor).
10. Harness notes: typing a path into the GTK 4 chooser races its inline completion under `xdotool type` (`notes.txtxt`, which the app correctly reported as `Import failed`); `check-full.sh` pastes the path instead. `xdotool`/`import` capture of a window rectangle can include an overlapping zenity window.

## Not verified / needs real hardware

kdialog path (KDE), dialog stacking under a real window manager, a real GNOME/KDE session's theme, real GPU drivers, HiDPI, IME.
