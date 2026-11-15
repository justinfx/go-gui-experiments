> An addendum to [One Go App, Served Five Ways](https://justinfx.com/2026/11/10/go-gui-five-ways/).
> Same spec, same shared core, same harness. Other addenda: [[Home]].

**Stack** [NSGo](https://github.com/nsgo-dev/nsgo) at commit `2345b2f` (no releases yet) ·
**Compared with** MyGo native UI mode, go-gui, Fyne, Cogent · **Measured** 2026-11-15, same machine as the article

## At a glance

> **Verdict:** the most complete new entry so far. Fast, polished, and no cgo
> anywhere, but very young, and its table needs hand-built pieces.
>
> **Stronger than the comparators at:** no cgo on any desktop, macOS
> included; desktop speed close to MyGo native UI; second-best blind style
> (4 out of 5, behind MyGo native UI's 5); the fastest cold build; a smaller
> browser download than go-gui, Fyne and Cogent.
>
> **Weaker at:** maturity (first commit 2026-10-21, no releases); the table
> (header sort built by hand, and the card view rebuilt as a one-column
> table so it can scroll past about row 160,000); memory (369 MB, above MyGo native UI and go-gui); fast
> wheel scrolling flashes briefly; on Linux it follows the system dark mode
> only through the GTK theme name.
>
> | | |
> |---|---|
> | **A lot of data?** | Yes on the desktop: smooth to row 1,000,000. The browser shows loading rows during fast scrolls. |
> | **Need cgo?** | No, on macOS, Linux or Windows. Native libraries load at run time. |
> | **Web, native, or both?** | Both from one Go UI codebase: native controls on the desktop, the page's own elements in the browser. |

The rest of this page is the evidence for the box above.

## What it is

A Go UI framework that drives each platform's own toolkit without cgo:
AppKit on macOS, GTK 4 on Linux and Direct2D on Windows, all loaded at run
time through purego (a way to call C libraries from Go without cgo), plus the browser's DOM through WASM. Apps are built with
its own `nsgo` tool. One pass, built by an agent from the spec like the
article's apps, then fixed after the author's hands-on check.

## Results

Desktop, median of 5 runs. Bold is the best figure in the row.

| | nsgo | mygoui | gogui | fyne | cogent |
|---|---|---|---|---|---|
| Ready (ms) | 498 | **236** | 277 | 572 | 555 |
| Jump to row 500k (ms) | 78 | **69** | 85 | 146 | 113 |
| Table scroll, max frame gap (ms) | **11** | 13 | 17 | 40 | 19 |
| Switch to cards (ms) | 59 | **51** | 117 | 104 | 112 |
| Sort (ms) | 272 | **241** | 262 | 254 | 262 |
| Peak memory (MB) | 369 | **219** | 279 | 1412 | 513 |
| Cold build (s) | **8.0** | 13.7 | 12.7 | 19.1 | 23.5 |
| Style, blind (1 to 5, one pass) | 4 | **5** | 2 | 2 | 1 |

Browser (Chrome, single run, frame gaps timed by the browser for all).

| | nsgo | mygoui | gogui | fyne | cogent |
|---|---|---|---|---|---|
| Download, gzip (MB) | 4.2 | **4.0** | 6.5 | 12.4 | 6.9 |
| Ready (ms) | 940 | 921 | **720** | 830 | 1,453 |
| Table scroll, max frame gap (ms) | **51** | 67 | 151 | 217 | 317 |
| Cards scroll, max frame gap (ms) | 49 | **33** | 151 | 135 | 717 |

In the browser, the table showed rows still loading for much of the scroll
test.

**Hands-on (macOS).** Cards at row 500,000, export, Reveal, themes and
dialogs worked. After one fix pass:

- Fast wheel scrolling flashes a little and stops at once. The app now
  prefetches more rows; the short catch-up left after the wheel stops is
  NSGo's own momentum (inertia) scrolling.
- Escape did not close the Host dialog while a text field had focus (NSGo on
  macOS keeps Escape for the field). The app now handles Escape while a dialog
  is open.
- The Table/Cards switch had almost no selected look; it is now a filled
  toggle.
- The author's own style score was 4 out of 5: close to Fyne and Wails in
  places, ahead of Cogent, with good-looking sliders and buttons.

**Desktop**

[![NSGo, dark theme, table view](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/nsgo-desktop.png)](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/nsgo-desktop.png)

**Desktop, cards**

[![NSGo, light theme, card view with a row selected](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/nsgo-cards.png)](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/nsgo-cards.png)

**Browser**

[![NSGo in Chrome, dark theme, table view](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/nsgo-web.png)](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/nsgo-web.png)


## The three questions

**A lot of data?** Yes on the desktop: table scroll is the smoothest measured
and jump and sort are close to MyGo native UI. Two pieces were built by hand:
header sort (the stock sort control is a small chevron button), and the cards,
because NSGo's list loses position precision past about row 160,000 (it lays
out in 32-bit floats). The cards are now a one-column table, which handles
large scroll ranges.

**Need cgo?** No. `CGO_ENABLED=0` builds on all three desktops, macOS
included, and the Linux binary links only libc; GTK 4, Pango and Cairo load at
run time. The article's cgo test core (a core that needs cgo) runs on the
desktop, and in the browser through `web-remote`, where the core runs on a Go
server.

**Web, native, or both?** Both from one Go UI codebase, with a small browser
download and less browser stutter than the other single-codebase options. On
Linux (the article's container check, X11 and sway) all 12 checks pass. File
dialogs and opening files need the desktop portal service
(xdg-desktop-portal), and fail silently without it. It
follows the system dark mode through the GTK theme name; on GTK before 4.20
it ignores GNOME's newer dark-style setting on its own. NSGo has no option for
the Linux app ID (what the window manager and launcher match on), so the app
sets it itself. The project started on 2026-10-21 and has no releases.

---

Code: [`apps/nsgo/`](https://github.com/justinfx/go-gui-experiments/tree/main/apps/nsgo) · Full results: [`results/nsgo/`](https://github.com/justinfx/go-gui-experiments/tree/main/results/nsgo) ·
Method: [`results/HARNESS_CHANGES.md`](https://github.com/justinfx/go-gui-experiments/blob/main/results/HARNESS_CHANGES.md) (E6 to E8), style pass [`results/judge-addenda4/`](https://github.com/justinfx/go-gui-experiments/tree/main/results/judge-addenda4)
