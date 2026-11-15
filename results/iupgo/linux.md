# IUP-Go (main cbcdf09) on Linux (container run, linux/arm64, Ubuntu 24.04, GTK3)

Harness: `linux/`, helper scripts `results/iupgo/linux/build.sh`, `run-x11.sh`, `run-sway.sh`, `dbg-img.sh`. The rebuilt image already has `xkbcli` and `gnome-themes-extra`; nothing was installed per container. Logs and screenshots: `results/iupgo/linux/` (`x11/`, `sway/`). No timings. Screenshots are of the app window only (dialogs by their own rectangle on sway; the File menu popup by a crop of the app window's area).

## How it runs on Linux

IUP-Go's default Linux backend is GTK3, linked at build time (cgo). `make build` is a plain `go build`; the binary is 9.5 MB and `ldd` lists the GTK3 stack (`libgtk-3`, `libgdk-3`, `libpango`, `libcairo`, `libglib`, `libX11`, `libwayland-client/-cursor/-egl`, `libxkbcommon`, `libepoxy`, `libatk`; `ldd.log`, `x11/maps.log`, `sway/maps.log`). The target needs the GTK3 runtime. Native dialogs, notifications and the clipboard come from GTK and D-Bus, not helper programs.

## Runtime matrix

| Check | x11 (Xvfb + openbox) | sway (headless Wayland) |
|---|---|---|
| Builds | yes (9.5 MB) | same binary |
| Starts and renders | yes, 1278x777, title `GGUIX - IUP-Go` (`x11/start.png`) | yes, 1280x800 (`sway/start.png`) |
| Window class / app_id | `WM_CLASS "gguix-iupgo", "Gguix-iupgo"` (`x11/wmclass.log`) | `app_id gguix-iupgo` (`sway/appid.log`) |
| Menus | File menu opens from a click (openbox shows the popup as an extra window, `x11/windows-with-menu.log`) | File menu with Export CSV Ctrl+E, Import Notes Ctrl+O, Reveal Last Export disabled until an export (`sway/menu-file.png`) |
| Shortcuts | Ctrl+Shift+C, Ctrl+2, Ctrl+E, Ctrl+I work; **Ctrl+Q exits** (`x11/quit.log`) | same; **Ctrl+Q exits** (`sway/quit.log`) |
| File dialog | native GTK save dialog (`x11/export-dialog.png`); export wrote 347,958 bytes (`x11/export-file.log`) | same (`sway/export-dialog.png`) |
| Notification | dunst "Export complete" (`x11/dunst.log`) | dunst "Export complete / 3,000 rows written" (`sway/dunst.log`) |
| Clipboard | `xclip -o` returned the CSV row (`x11/clipboard.txt`) | `wl-paste` returned the CSV row (`sway/clipboard.txt`) |
| Reveal | `xdg-open /tmp/pf-x11` and `xdg-open /tmp` (parent folder of the export, file not selected) (`x11/xdg-open.log`) | `xdg-open /tmp/pf-sway` and `/tmp` (`sway/xdg-open.log`) |
| Host panel | Source native (`x11/host-dialog.png`) | same (`sway/host-dialog.png`) |
| Cards view | renders correctly (`x11/cards.png`) | renders correctly (`sway/cards.png`) |
| System light/dark live | not demonstrated | not demonstrated: `gsettings set color-scheme prefer-dark` changed nothing (`sway/system-dark.png`); no portal settings service in the container |
| `go test` (no GTK_THEME) | 12/12 for the Matrix default and 12/12 for the native Table (`test-theme-default.log`) | n/a |

## Findings

1. **T7.** With `APPEARANCE=DARK`, IUP's `DARKMODE` stayed NO on GTK3 in the container, even with Adwaita-dark installed and 300 ms of main loop; it only turns YES when `GTK_THEME=Adwaita:dark` is set at start (seen earlier). The app now sets its own palette from the user's explicit choice, so T7 checks the palette and prefs and only logs `DARKMODE` (`test-theme-default.log`: `NO`). The native GTK widgets turning dark on a switch was not checked.
2. **Table cell images missing on GTK (native Table only).** The default build now uses IupMatrixEx, whose `DRAW_CB` draws the bars, pills (text inside) and stars directly; it shows them on GTK under x11 and sway (`sway/start.png`). For the native Table (`TABLE=native`): Exact data in friction.md: `IMAGE_CB` is called and returns valid image names (80x20, 30x20, 68x20, 32 bpp), nothing is drawn. Cause not isolated in the GTK driver source. The Cards view (canvas drawing) shows the same bars, badges and stars correctly.
3. **Visible range.** The status bar still read "rows 1-20" style values in the later runs; the earlier off-by-two was not seen again.
4. **Form wider than its pane at 1280 px** (Discontinued label clipped).
5. Image gaps: none needed from the image for these checks. `gguix-session` is not on the PATH in the rebuilt image; `/repo/linux/run.sh` was used instead.

## Not verified

Menu items beyond File, dark theme on GTK, live system theme on a real desktop, IME, HiDPI.
