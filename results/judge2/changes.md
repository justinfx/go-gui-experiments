# Score changes from results/judge/scores.md

| Factor | App | Old | New | Why |
|---|---|---|---|---|
| Runtime requirements | gotth | 4 | 3 | Linux fix adds `zenity` and a clipboard tool as run-time dependencies; webview_go needs `webkit2gtk-4.0` (`webview.go:9`), absent on Ubuntu 24.04 (`results/LINUX.md`). |
| Native widgets | fyne | 2 | 3 | Old "no min size" was wrong (`internal/driver/glfw/window_desktop.go:300`, ERRATA); native D-Bus notification on Linux (`app/app_xdg.go:80-81`, LINUX.md). |
| Native widgets | gotth | 2 | 3 | "No menus or dialogs" was wrong (ERRATA, `webview.go:83,173`); native macOS menu user-verified; Linux dialogs, notification and clipboard now work (fixpass.md, LINUX.md). Old penalty was an app bug, now fixed. |
| Performance | fyne | 3 | 4 | Re-graded by perceptual bands: all actions instant or responsive, no stutter. Memory alone no longer costs a point. |
| Performance | wails | 4 | 5 | Same bands as cogent (instant jump, no stutter); tie. |
| Performance | gotth | 2 | 3 | Max frame gaps (86 ms scroll, 76 ms cards) are below visible stutter; placeholders and a ~1 s cards switch remain. |
| Style appeal | cogent | 3 | 2 | New blind pass on current captures: clipped rating and Stock header, unboxed form fields, stray "1" in Notes. |
| Style appeal | wails | 5 | 4 | New blind pass: polished but ties with gotth (same shadcn look) and shows SKU truncation. |

Unchanged scores whose reasons changed:
- cogent native widgets 2: min-size gap closed (user-verified), but no native menu, dialogs or Linux notification remain framework limits.
- wails native widgets 4: reveal gap was ours (ERRATA) and is fixed; Linux menu, dialogs and notification are native.
- miqt native widgets 5: notification nil-icon was our bug, fixed. miqt performance 2: base app only; miqt-fitcap shows Qt itself is smooth.
- cogent Ctrl+Q deadlock on Linux was our bug, fixed; no score effect.
- Desktop-web reach gotth 4: overhead rose from 15.5% to 17.9% with the native shell; still 4.

New in judge2: -fitcap performance scores and a GoTTH desktop-shell comparison (gotth-glaze, miqt-gotth); see scores.md.

## Clean re-run (2026-11-10)

| Factor | App | Old | New | Why |
|---|---|---|---|---|
| Performance | fyne | 4 | 5 | Same bands as cogent now that cogent's jump is 113 ms; no stutter, no placeholders. |
| Performance | wails | 5 | 4 | Absolute pacing (B4) exposes scroll gaps: max 91 ms median, one run over 100 ms. At the edge of visible stutter. |
| Desktop plus web reach | cogent | 5 | 4 | Web ready is 1.45 s once the wasm download and compile count (B3): a noticeable wait. |
| Performance (variant) | fyne-fitcap | 4 | 5 | Follows fyne base; same bands. |
| Performance (variant) | wails-fitcap | 5 | 3 | Visible stutter in all 5 runs (max gap 127 to 183 ms). |

## Hands-on check (2026-11-10)

| Factor | App | Old | New | Why |
|---|---|---|---|---|
| Performance | wails | 4 | 5 | Author's hands-on check: 91 ms isolated gaps not perceived as stutter; controls responsive. |
| Performance (variant) | wails-fitcap | 3 | 5 | Same check: 140 ms isolated gaps felt the same as base Wails. |

## Hands-on check, gotth-fitcap and base MIQT (2026-11-10)

| App | Factor | Before | After | Why |
|---|---|---|---|---|
| gotth-fitcap | Performance (variant) | 2 | 4 | Author's hands-on check: only minor lag on fast scroll, controls kept up; acceptable but not as good as Wails. Measured 321 ms repeated gaps did not read as stutter. Scrollbar rendering bug keeps it below 5. |
| miqt | Performance | 2 | 2 | Hands-on check confirmed: real lag on fast scroll and scrollbar drag, and the scrollbar itself lagged. |
