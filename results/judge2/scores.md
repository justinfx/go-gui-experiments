# GGUIX judge2 scores

Scores 1 to 5, same rubric as `results/judge/scores.md`. Evidence is the current `results/<id>/metrics.json` (desktop: median of 5; web: single run at DPR 2), notes in `results/<id>/`, `results/LINUX.md`, and `results/ERRATA.md` (errata override notes). Library claims are checked in `~/go/pkg/mod` and cited as file:line.

Perception bands used for performance: <=100 ms instant, <=1 s responsive, >1 s noticeable wait, frame gaps >100 ms visible stutter. Same band = tie. Web numbers are single runs, so only a large gap ranks.

Linux note: Linux was checked for function only (`results/LINUX.md`). Timings, sharpness and GTK4 rendering (Wails default backend, glaze GTK4 path) were NOT verified on Linux.

## Base apps

| Factor | fyne | cogent | wails | gotth | miqt |
|---|---|---|---|---|---|
| 1 Bootstrap and build loop | 4 | 4 | 3 | 3 | 3 |
| 2 Runtime requirements | 5 | 5 | 4 | 3 | 3 |
| 3 Toolchain footprint | 4 | 4 | 2 | 3 | 2 |
| 4 Native desktop widgets | 3 | 2 | 4 | 3 | 5 |
| 5 Desktop plus web reach | 3 | 4 | 4 | 4 | N/A |
| 6 cgo-core tolerance | 4 | 4 | 5 | 5 | 3 |
| 7 Code clarity | 4 | 3 | 2 | 3 | 3 |
| 8 Testability | 4 | 4 | 4 | 3 | 4 |
| 9 Performance with large data | 5 | 5 | 5 | 3 | 2 |
| 10 Style appeal (blind) | 3 | 2 | 4 | 4 | 3 |
| 11 Native host access | 4 | 4 | 4 | 4 | 4 |
| 12 Custom rendering | 3 | 4 | 5 | 4 | 3 |

### Factor group ranking
- Build and ship (1 to 3): fyne and cogent lead (pure Go, no external runtime); wails and miqt trail (npm, Qt).
- Native integration (4, 11): miqt, then wails; gotth and fyne now mid; cogent last.
- Reach and core (5, 6): cogent for shared-code reach; wails and gotth for cgo-core tolerance.
- Code and tests (7, 8, 12): fyne for clarity; wails for custom rendering and test breadth.
- User-perceived quality (9, 10): fyne, cogent and wails tie on performance (wails' fast-scroll gaps were not perceived as stutter in a hands-on check); wails and gotth lead blind style.

## Justifications

### 1 Bootstrap and build loop
- fyne 4: plain `go build` works, cold 20.3 s, incremental 2.3 s, no-op 0.18 s (`results/fyne/metrics.json`). No hot reload.
- cogent 4: plain build works, cold 23.6 s, incremental 2.3 s, no-op 1.8 s (`results/cogent/metrics.json`); all within the responsive-to-short-wait range.
- wails 3: plain `go build` fails without the frontend (`plain_go_build_from_clean.rc=1`), cold 41.4 s; Vite HMR helps (`results/wails/metrics.json`, `toolchain.md`).
- gotth 3: fastest cold build (9.6 s) but `templ generate` needed first (`plain_go_build_from_clean.rc=1`, `results/gotth/metrics.json`).
- miqt 3: cold 110.8 s (bindings compile), incremental 1.9 s, plain build works (`results/miqt/metrics.json`).

### 2 Runtime requirements
- fyne 5: 33 MB bundle, no external dylibs (`results/fyne/metrics.json`); runs on X11 and Wayland (`results/LINUX.md`).
- cogent 5: 27 MB bundle, no external dylibs; runs on X11 and Wayland.
- wails 4: 11.6 MB bundle, needs the system WebView (WebKitGTK on Linux). The default GTK4 build starts with the WebKit sandbox on in the container retest, but its rendering could not be captured, so GTK4 rendering is unverified (`results/LINUX.md`).
- gotth 3 (was 4): 16.8 MB bundle, but on Linux it now also needs `zenity` and xclip/xsel or wl-clipboard at run time, and webview_go pins `webkit2gtk-4.0` (`webview_go webview.go:9`), which Ubuntu 24.04 does not ship (`results/LINUX.md` items 8 and fix table).
- miqt 3: 113 MB bundle with 80 MB of Qt frameworks (`results/miqt/metrics.json`).

### 3 Toolchain footprint
Unchanged from judge: fyne 4 and cogent 4 (Go-installed CLIs), gotth 3 (templ, shadcn-templ, Tailwind binary; Node only for T9), wails 2 (Node, npm), miqt 2 (Qt, pkgconf, C++). Sources: `results/<id>/toolchain.md`.

### 4 Native desktop widgets
- fyne 3 (was 2): controls are framework-drawn, but the earlier "no min size" was wrong: the glfw driver sets size limits from content MinSize (`fyne v2.8.2 internal/driver/glfw/window_desktop.go:300`, ERRATA). On Linux notifications go over D-Bus from the library (`fyne v2.8.2 app/app_xdg.go:80-81`) and worked (`results/LINUX.md`). Dialogs are Fyne-drawn outside flatpak (`dialog/file_xdg_flatpak.go` exists; not tried).
- cogent 2: no native menu bar, Cogent-drawn file picker, no notification on Linux (`results/LINUX.md`). Min size now set via the exported glfw handle (`cogentcore.org/core@v0.5.27 system/driver/desktop/window.go:27-31`) and user-verified (`results/cogent/fixpass.md`), so that gap is gone, but the rest are framework limits.
- wails 4: native GtkMenuBar, native GTK3 chooser and D-Bus notification on Linux (`results/LINUX.md`); controls are HTML. Reveal now uses `OpenFileManager` (`wails v3.1.0 pkg/application/environment_manager.go:52`), user-verified on macOS (`results/wails/fixpass.md`).
- gotth 3 (was 2): webview_go exposes the native window (`webview_go webview.go:83,173`); a `-tags native` macOS menu bar was added and user-verified (`results/gotth/fixpass.md`). On Linux, real zenity dialogs, notification and clipboard now work (`results/LINUX.md` fix table). Menu bar is still in-page on Linux.
- miqt 5: real Qt widgets, QMenuBar, QFileDialog. Notification nil-icon bug was ours and is fixed (`results/miqt/fixpass.md`); on stock GNOME tray notifications need a StatusNotifierWatcher (Qt platform dependence, `results/LINUX.md`).

### 5 Desktop plus web reach
- fyne 3: 13.7% overhead, but the web build is 39.6 MB transfer and has visible stutter at DPR 2 (scroll p95 218 ms, max 398 ms; sort 1.2 s) (`results/fyne/metrics.json` web.bench-web).
- cogent 5: 12.2% overhead, 6.9 MB transfer, web scroll p95 5.7 ms, jump 132 ms (`results/cogent/metrics.json`).
- wails 4: 11.3% overhead and 0.8 MB transfer with smooth web, but unmarked `isWeb` branches understate overhead (judge sample check, still present).
- gotth 4: the web is its native target (1.4 MB), overhead 17.9% now that the desktop shell grew (590 desktop lines).
- miqt N/A: no browser target (Android and WebEngine exist, ERRATA).

### 6 cgo-core tolerance
Unchanged: wails and gotth 5 (server-side core, cgoprobe PASS on web), fyne and cogent 4 (web FAIL, web-remote PASS), miqt 3 (desktop only). `cgoprobe` in each `metrics.json`.

### 7 Code clarity
Unchanged: fyne 4 (95-line longest func, single language), cogent 3 (181-line `newDetailForm`, Makefile marker errors), wails 2 (four languages, unmarked branches), gotth 3 (templ+JS windowing; now an Objective-C shim behind a tag), miqt 3 (138-line func, 3 marker errors). `loc` in each `metrics.json`.

### 8 Testability
Unchanged: wails 4 (34 checks incl. Playwright; 34/34 on Linux with Chromium), fyne 4, cogent 4, miqt 4 (12, 12, 18), gotth 3 (T9 needs Node and Chrome). `tests` in `metrics.json`, `results/LINUX.md`.

### 9 Performance with large data (desktop, macOS medians)
- cogent 5: jump 60 ms, scroll p95 4 ms/max 19 ms, cards switch 111 ms, sort 262 ms. All instant or responsive, no stutter.
- wails 5: superseded numbers removed. Final: scroll max 91 ms after the harness pacing fix, restored to 5 by the hands-on check. See "Clean re-run re-check" and "Hands-on calibration" below.
- fyne 5: superseded numbers removed. Final: jump 146 ms, scroll max 40 ms, no placeholders. See "Clean re-run re-check" below.
- gotth 3 (was 2): no visible stutter (scroll max 86 ms, cards max 76 ms), jump 231 ms, but 155 placeholder steps while scrolling and a cards switch of 978 ms, right at the edge of a noticeable wait.
- miqt 2: scroll max 254 ms and cards max 436 ms are visible stutter, jump 652 ms, cards switch 908 ms. This is our adapter, not Qt: the same toolkit with a different data layer (miqt-fitcap) has scroll max 16 ms (see variants). Kept at 2 because it is the shipped base app.

### 10 Style appeal
From `style_blind.md`: wails 4, gotth 4, fyne 3, miqt 3, cogent 2. Dark table view only.

### 11 Native host access
All 4. Identity works on desktop and labels server-process on web (`results/<id>/capabilities.md`). No app can select the file on reveal on Linux (`results/LINUX.md` item 6; wails source says the Linux fallback cannot select, `wails v3.1.0 internal/fileexplorer/fileexplorer_linux.go:77`). gotth's Linux reveal of the last export is no longer disabled after the export fix.

### 12 Custom rendering
Unchanged: wails 5 (100 lines, smooth cards), cogent 4 (550 lines, smooth), gotth 4 (90 lines, placeholders), fyne 3 (581 lines), miqt 3 (300 lines, janky cards in base app; smooth in fitcap).

## Variants

### Data-layer variants (-fitcap, 2 requests in flight), desktop performance score

| App | Score | Evidence (`results/<id>-fitcap/metrics.json`) |
|---|---|---|
| fyne-fitcap | 5 | jump 121 ms, scroll max 26 ms, 0 placeholders, cards switch 65 ms. Same bands as base. |
| cogent-fitcap | 5 | jump 80 ms, scroll max 19 ms, 0 placeholders (base 24), cards switch 54 ms. |
| wails-fitcap | 5 | hands-on check: not perceived as stutter (`results/wails-fitcap/manual-check.md`). Measured: max gap 140 ms (127 to 183, all 5 runs over 100 ms). Otherwise instant: jump 90 ms, cards switch 105 ms, about 1 placeholder step. 36 of 36 tests pass after the test was updated to the cap (`results/wails-fitcap/fixpass.md`). |
| gotth-fitcap | 4 | hands-on check: only the most minor lag on fast scroll, dragging much more responsive than base MIQT; acceptable, not as good as Wails (`results/gotth-fitcap/manual-check.md`). Measured: scroll p95 and max 321 ms, placeholders gone, cards switch 454 ms. Not 5: a little lag remains, and the table scrollbar did not render (dragging its area works). Was 2 before the hands-on check. |
| miqt-fitcap | 4 | scroll max 16 ms, cards max 5 ms (no stutter), jump 183 ms, but cards switch 981 ms and sort 505 ms. Shows Qt is not the base app's bottleneck. |

### Desktop shells for GoTTH

| Factor | gotth (webview_go) | gotth-glaze | miqt-gotth |
|---|---|---|---|
| Build loop | 3 | 4 | 2 |
| Runtime requirements | 3 | 4 | 1 |
| Toolchain | 3 | 4 | 2 |
| Native integration | 3 | 3 | 4 |
| Cross builds | 4 | 5 | 1 |
| Performance | 3 | 3 | 2 |
| Linux function | 3 | 3 | 2 |

- gotth-glaze: `CGO_ENABLED=0`, cold 8.3 s, incremental 0.8 s, 13.8 MB bundle (`results/gotth-glaze/metrics.json`); cross builds without Docker (`capabilities.md`). Native macOS menu; Linux has no menu bar from glaze (`glaze@v0.0.83 menu/menu_linux.go:1-3`), Windows ignores shortcuts (`menu/menu_windows.go:9-11`). Linux file dialogs are real GtkFileChooserNative (`glaze@v0.0.83 dialog_linux.go:1,91`), verified on X11 and sway, GTK3 path only. Notification unsupported off macOS (`native@v0.1.18 notify/notify_other.go:1,5`); clipboard unsupported on Linux (`clipboard/clipboard_linux.go:26,28`) but has a Win32 backend on Windows (`clipboard/clipboard_windows.go`); reveal opens the parent folder (`openurl/openurl_linux.go:26-28`). Performance equals gotth (scroll max 86 ms, cards switch 974 ms). It stands for a feasible cgo-free shell, not a product pick: the library gaps above would need filling.
- miqt-gotth: native QMenuBar, QFileDialog and clipboard on every OS (`capabilities.md`), but 342 MB bundle, 132 s cold build, no cross route (Dockerfiles lack WebEngine, `shell.md`), and the slowest shell: scroll max 282 ms and cards max 209 ms (visible stutter), cards switch 1.9 s (noticeable wait) (`results/miqt-gotth/metrics.json`). On Linux it runs with the Chromium sandbox on as a normal user (container retest, `results/miqt-gotth/linux.md`, "Sandbox-on retest"), and Ubuntu's Qt 6.4 WebEngine misrenders the Tailwind CSS (`results/LINUX.md`, `results/miqt-gotth/linux.md:36-38`).

## Clean re-run re-check (2026-11-10)

Evidence above was refreshed against the clean re-run (`results/DEVIATIONS.md`, "Clean re-run"). Scores changed only where a number crossed a perception band. Current evidence for the changed and borderline scores:

- 9 fyne 5 (was 4): jump 146 ms, scroll max 40 ms, cards switch 104 ms, sort 254 ms, 0 placeholders. These are now the same bands as cogent (cogent's jump rose to 113 ms), so they tie. Peak memory 1.4 GB is still the real cost; it is not a perceived delay.
- 9 cogent 5: jump 113 ms, scroll max 19 ms (p95 3.3 ms), cards switch 112 ms, sort 262 ms; 24 placeholder steps.
- 9 wails 4 (was 5): scroll max gap 91 ms median (68 to 108; one run of 5 over 100 ms), p95 54 ms, 64 placeholder steps. Jump 90 ms, cards switch 194 ms, sort 280 ms. The old 22 ms max gap was a harness artifact: before B4 the scroll ran at relative pacing, so slow frames stretched the run instead of showing as gaps (`results/HARNESS_CHANGES.md`). At the edge of visible stutter, so one point below fyne and cogent.
- 9 gotth 3, miqt 2: unchanged bands (gotth scroll max 92 ms, cards switch 969 ms; miqt scroll max 251 ms, cards max 428 ms).
- 5 cogent 4 (was 5): the web build is smooth (scroll max 15 ms) but ready is 1,453 ms, a noticeable wait, now that B3 counts the wasm download and compile. Transfer 6.9 MB, overhead 12.3%.
- 5 fyne 3: 12.4 MB gzipped transfer (B2), web scroll max 126 ms (visible stutter), sort 825 ms, ready 830 ms. Overhead 13.7%.
- 5 wails 4, gotth 4: wails 0.26 MB, smooth, ready 179 ms, overhead 10.3%; gotth 0.16 MB, web scroll max 152 ms and cards switch 1,024 ms, overhead 19.5%.
- 2 runtime: bundles are now release builds: fyne 23.0 MB, cogent 26.5 MB (measured by hand, see DEVIATIONS), wails 11.4 MB, gotth 11.1 MB, miqt 111 MB (78 MB Qt). No score changes.
- 3 cogent 4 kept: `make package` fails on a machine without `pip` on PATH, because `core pack` runs `pip install dmgbuild` (DEVIATIONS). It affects packaging only; build, run and tests need nothing extra. Noted, not scored down.
- 1 build loop: cold fyne 19.1 s, cogent 23.5 s, wails 39.2 s, gotth 9.4 s, miqt 113 s; incremental 1.5, 2.2, 2.5, 1.6, 2.9 s. Same scores. Plain `go build` works for wails and gotth once their codegen step has run (A4).
- Shells: gotth-glaze 3 (scroll max 83 ms, cards switch 970 ms; cold 8.1 s; 9.5 MB bundle), miqt-gotth 2 (scroll max 280 ms, cards switch 1,859 ms; 335 MB bundle; cold 133 s). Unchanged.

Older numbers in the justifications above this section are from the pre-audit run; where they differ, this section wins.

## Hands-on calibration (2026-11-10)

The author used base Wails and wails-fitcap by hand, one at a time (`results/wails/manual-check.md`, `results/wails-fitcap/manual-check.md`). Isolated worst gaps of 91 to 140 ms during an aggressive wheel flick showed as an occasional flicker, not perceived as a problem; controls stayed responsive; blank rows on a fast scrollbar drag were judged normal. Both felt the same.

- 9 wails 5 (was 4 after the clean re-run): restored. The gaps are measured, but not perceived as stutter.
- wails-fitcap 5 (was 3): same reason; jump 90 ms and cards switch 105 ms are instant.
- Rule used from here on: isolated gaps up to about 200 ms in the fast-scroll test do not by themselves cost a band. Repeated gaps of 250 ms or more (p95 above 200 ms) still count as visible stutter until checked by hand. gotth-fitcap (p95 321 ms) and miqt (p95 229 ms) are unchanged on that rule.
