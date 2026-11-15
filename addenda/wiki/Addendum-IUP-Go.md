> An addendum to [One Go App, Served Five Ways](https://justinfx.com/2026/11/10/go-gui-five-ways/).
> Same spec, same shared core, same harness. Other addenda: [[Home]].

**Stack** [IUP-Go](https://github.com/gen2brain/iup-go) at `main` commit `cbcdf09` ·
**Compared with** MIQT, MyGo native UI mode, NSGo, Fyne · **Measured** 2026-11-15, same machine as the article

## At a glance

> **Verdict:** if you want native widgets without Qt, IUP-Go is a lighter,
> much faster alternative to MIQT on the desktop, with a far simpler build. It
> can also target the browser, unlike MIQT, but that build is the most involved
> of any option measured. It has the plainest look and the most bugs found by
> hand of the addenda apps.
>
> **Stronger than the comparators at:** memory (205 MB, the lowest
> measured); table scroll with IUP's drawn grid control (the smoothest
> measured); against MIQT, much faster jump, scroll, cards and sort, a 13 s cold
> build against 113 s, and one small C library to ship instead of 80 MB of Qt.
>
> **Weaker at:** style (the author's hands-on 2 out of 5; blind, a tie with
> Fyne and MIQT, so no step up in polish over MIQT); a slow start (about 1 s)
> and slower sort than the other native options; the platform's own table
> stutters on fast scrolls, while the smooth grid looks basic; the browser build
> needs a separate C-to-WASM toolchain; the Go binding never frees the C
> strings it creates (a memory leak).
>
> | | |
> |---|---|
> | **A lot of data?** | Yes with IUP's drawn grid, at the cost of a plain look; the platform's own table stutters. |
> | **Need cgo?** | By default yes; with `CGO_ENABLED=0` it loads a prebuilt IUP library at run time. |
> | **Web, native, or both?** | Both, but the browser build needs its own toolchain and uses a different table. |

The rest of this page is the evidence for the box above.

## What it is

Go bindings for IUP, a C toolkit that wraps each platform's own widgets:
Cocoa on macOS, GTK 3 on Linux and Win32 on Windows. The browser target
compiles IUP to WASM with Emscripten (a C-to-WASM compiler) and runs the app in
a web worker (a background thread of the page). Built from
`main`, not the latest tag, by an agent from the spec like the article's
apps, then fixed after the author's hands-on check.

It has two table controls, and the app builds with either:

| Build | Table | Scroll, max frame gap | Look |
|---|---|---|---|
| default (`TABLE=matrix`) | IupMatrixEx, a grid whose cells the app draws | 8 ms | a plain grid |
| `TABLE=native` | IupTable, the platform's own table | about 143 ms (one rough run) | the native table |

## Results

Desktop, median of 5 runs, default build. Bold is the best figure in the row.

| | iupgo | miqt | mygoui | nsgo | fyne |
|---|---|---|---|---|---|
| Ready (ms) | 1,023 | 443 | **236** | 498 | 572 |
| Jump to row 500k (ms) | 92 | 666 | **69** | 78 | 146 |
| Table scroll, max frame gap (ms) | **8** | 251 | 13 | 11 | 40 |
| Switch to cards (ms) | 130 | 929 | **51** | 59 | 104 |
| Sort (ms) | 497 | 622 | **241** | 272 | 254 |
| Peak memory (MB) | **205** | 702 | 219 | 369 | 1412 |
| Cold build (s) | 13.4 | 112.6 | 13.7 | **8.0** | 19.1 |
| Style, blind (1 to 5, one pass) | 3 | 2 | **5** | 4 | 2 |

The blind pass put IUP-Go one point above Fyne and MIQT; a side-by-side
re-check scored all three 3, a tie. Browser (Chrome, single run): ready in
526 ms, 8.2 MB of files. Frame gaps were not measured: the app runs in a web
worker, where the harness's frame timer cannot see it. MIQT has no browser build.

**Hands-on (macOS).** Table scrolling with the grid was very smooth; jump,
sort, search, export and Revert worked. After two fix passes:

- Fixed: cards stuck on "Loading..." past the first pages, the Table/Cards
  selector losing its selection (macOS grouped it with the form's status
  radios), a cut-off hint in the form, a black date field in the light
  theme, and the busy cursor staying on after export.
- Left: the Search and Go to row boxes keep their text at the top (IUP's
  macOS text field has no vertical alignment), widget borders are low
  contrast (no border color setting), and IUP's macOS notification call
  blocks the app for up to 2 s.
- The native table looked much better than the grid, with nicer stock bars
  and stars, but lagged and showed loading rows when the scrollbar was dragged.
- The author's style score was 2 out of 5, slightly below Cogent: a basic
  look, and the agent could not get close to the other apps from the same
  spec.

**Desktop, drawn grid (default)**

[![IUP-Go with the IupMatrixEx grid, dark theme, table view](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/iupgo-desktop.png)](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/iupgo-desktop.png)

**Desktop, cards**

[![IUP-Go, light theme, card view with a row selected](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/iupgo-cards.png)](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/iupgo-cards.png)

**Browser (IUP's web table)**

[![IUP-Go in Chrome, dark theme, table view](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/iupgo-web.png)](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/iupgo-web.png)


## The three questions

**A lot of data?** Yes with the drawn grid (IupMatrixEx): the smoothest table scroll
measured and fast jumps, with the app drawing every cell. The native table
looks better but stutters under fast scrolling; the cost is its layout and
cell callbacks per step. Sort and search are slower than the other native
options.

**Need cgo?** By default yes, for the C toolkit. With `CGO_ENABLED=0` the
binding loads a prebuilt IUP library at run time instead, so the build needs no
C compiler but the program still ships a C library. The article's cgo test core (a core
that needs cgo) runs on the desktop, and in the browser through `web-remote`,
where the core runs on a Go server.

**Web, native, or both?** Both, but not from one simple build. The browser
target needs Emscripten (about 2 GB installed), server headers that turn on
cross-origin isolation (the WASM build needs shared memory), and uses IUP's web table, not the grid. On Linux (the
article's container check, X11 and sway) all 12 checks pass with GTK 3; the
grid draws the stock bars and status pills there, while images in the native
table do not show on GTK.

---

Code: [`apps/iupgo/`](https://github.com/justinfx/go-gui-experiments/tree/main/apps/iupgo) · Full results: [`results/iupgo/`](https://github.com/justinfx/go-gui-experiments/tree/main/results/iupgo) ·
Method: [`results/HARNESS_CHANGES.md`](https://github.com/justinfx/go-gui-experiments/blob/main/results/HARNESS_CHANGES.md) (E6 to E8), style pass [`results/judge-addenda4/`](https://github.com/justinfx/go-gui-experiments/tree/main/results/judge-addenda4)
