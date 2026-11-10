# GGUIX Linux runtime harness

Runs the seven desktop apps (fyne, cogent, wails, gotth, miqt, gotth-glaze, miqt-gotth) on Linux inside a native `linux/arm64` Docker container
(Apple Silicon, no emulation) to check that they start and which native integrations work. It does not measure timings or visual sharpness
(software rendering in a container is not representative). Results: `results/LINUX.md` and `results/<id>/linux.md`.

## Files

| File | Where it runs | Purpose |
|---|---|---|
| `Dockerfile` | host (build) | Ubuntu 24.04 arm64, Go 1.27.3 tarball, Node LTS, GL/X11/Wayland dev libs, GTK3 + GTK4, WebKitGTK 4.1 and 6.0, Qt 6.4 base + WebEngine + WebChannel, Xvfb, openbox, weston, sway, xdotool, ImageMagick, xclip, wl-clipboard, dunst, dbus tools, xdg-desktop-portal, Tailwind (linux-arm64) |
| `dx.sh` | host | `dx.sh build` builds the image; `dx.sh <cmd>` runs a throwaway container with the right mounts |
| `run.sh` | container | `run.sh <x11\|wayland\|sway> <app-id> [cmd...]` starts a session (session bus, display server, WM, dunst, `dbus-monitor` log) and runs the command in it |
| `prep.sh` | container | copies `apps/<id>` and `core/` into `/work` so Linux builds never overwrite macOS binaries or `node_modules` |
| `lib.sh` | container, sourced | `waitwin`, `shotwin` (one window only), `dbus_grep`; sway helpers `swwait`, `swshot` (grim, window rectangle only), `swclick`, `swkey` |
| `wlinput.py` | container | persistent virtual keyboard and pointer for sway (Wayland wire protocol, real evdev keycodes) |
| `sni-watcher.py` | container | fake `org.kde.StatusNotifierWatcher` to emulate a KDE session for Qt tray notifications |
| `probe-start.sh` | container | start a binary, report whether it is still alive (used for weston headless) |
| `shim/xdg-open` | container | logs what an app asks `xdg-open` to open (no file manager in the container) |

## Mounts

Repo at `/repo` (rw), `~/go/pkg/mod` at `/gomod` (GOMODCACHE, rw), volume `gguix-gocache` at `/gocache` (Linux GOCACHE), volume `gguix-gopath` at `/root/go` (installed tools such as `core`, `wails3`, `templ`), volume `gguix-work` at `/work` (scratch copies from `prep.sh`).

## Session modes

- `x11`: Xvfb + openbox. `xdotool`, `xclip`, `import -window` work.
- `wayland`: weston `--backend=headless --renderer=pixman`. Real Wayland clients, but weston headless advertises **no `wl_seat`**: GLFW-based apps (Fyne, Cogent) segfault in `glfwInit` and GTK prints seat assertions. No input injection, no capture. Only useful as a "does it crash" probe.
- `sway`: sway on the wlroots headless backend, floating windows. `grim` crops to the window rectangle, `wlinput.py` injects keys and clicks, `wl-paste` reads the clipboard. This is the working Wayland column.

## Typical use

```sh
linux/dx.sh build                                   # once
docker run -d --name gguixs --platform linux/arm64 --shm-size=1g \
  -v "$PWD":/repo -v "$HOME/go/pkg/mod":/gomod -v gguix-gocache:/gocache -v gguix-work:/work -v gguix-gopath:/root/go \
  gguix-linux sleep infinity                        # long-lived container for interactive sessions
docker exec gguixs /repo/linux/prep.sh fyne
docker exec -w /work/apps/fyne gguixs make build
docker exec -d gguixs /repo/linux/run.sh sway fyne sleep infinity      # session; env is written to results/fyne/linux/sway/env.sh
docker exec -w /work/apps/fyne gguixs bash -c '. /repo/results/fyne/linux/sway/env.sh; . /repo/linux/lib.sh; ./bin/gguix-fyne & swwait "GGUIX" 20; swshot "GGUIX" /repo/results/fyne/linux/x.png'
```

Per-session logs land in `results/<id>/linux/<mode>/`: `dbus.log` (all session-bus traffic, grep `member=Notify`), `dunst.log` (shown notifications), `xdg-open.log`, `display.log`, `app.log`.

## Sandbox note (read before running the web-engine apps)

WebKitGTK 6.0 (GTK4: Wails default build, glaze when `libwebkitgtk-6.0` is installed) starts a bubblewrap sandbox that needs user namespaces, which Docker's default seccomp profile blocks. Qt WebEngine's Chromium sandbox refuses to run as root.
By default `dx.sh` relaxes nothing, so those paths fail in this container (see the per-app reports). Two explicit opt-ins exist and weaken isolation of the app under test, so they are off by default:
`GGUIX_RELAX_SECCOMP=1` (adds `--security-opt seccomp=unconfined`) and `GGUIX_NO_QT_SANDBOX=1` (`QTWEBENGINE_DISABLE_SANDBOX=1`). Better: re-check those two cases on real Linux hardware as a normal user.

## Known container-only workarounds

- `webkit2gtk-4.0.pc` is aliased to the 4.1 pkg-config file because `webview_go` still asks for the 4.0 module name, which Ubuntu 24.04 no longer ships.
- Google Chrome has no Linux arm64 build, so Playwright tests that use `channel: "chrome"` (Wails) need Playwright's bundled Chromium (done in the scratch copy only).
- The Makefiles of the GoTTH apps download a macOS Tailwind binary; `bin/tools/tailwindcss` is symlinked to the Linux one in the scratch copy.
