> An addendum to [One Go App, Served Five Ways](https://justinfx.com/2026/11/10/go-gui-five-ways/).
> Same spec, same shared core, same harness. Other addenda: [[Home]].

**Stack** [go-gui](https://go-gui.com/) v0.96.0 ·
**Compared with** MyGo native UI mode, Fyne, Cogent · **Measured** 2026-11-14, same machine as the article

## At a glance

> **Verdict:** fast, but not ready for a large-data table app. Its stock table
> shows data in pages of 5,000 rows instead of one continuous scroll, and a polished table needs a lot of custom work.
>
> **Stronger than the comparators at:** desktop speed (close to MyGo native
> UI, ahead of Fyne and Cogent on scroll, jump and memory); no cgo on Linux or
> Windows; a smaller browser download than Fyne.
>
> **Weaker at:** the table (pages of 5,000 rows, text-only cells, visual
> glitches in both builds); style (2 and 4 out of 5, against MyGo native
> UI's 5); browser scroll (more stutter than MyGo native UI); no menu bar on
> Linux; maturity (v0.96.0, breaking changes in recent minor versions).
>
> | | |
> |---|---|
> | **A lot of data?** | Fast to scroll, but the stock table shows pages of 5,000 rows; continuous scrolling means building your own. |
> | **Need cgo?** | Not on Linux or Windows; yes on macOS (Metal). |
> | **Web, native, or both?** | Both from one Go UI codebase, with visible scroll stutter in the browser. |

The rest of this page is the evidence for the box above.

## What it is

A Go GUI framework that draws its own widgets on the GPU (Metal on macOS, GL on
Linux and Windows) and also builds to WASM for the browser, like Fyne, Cogent
and MyGo's native UI mode. No webview mode, no server mode.

| Pass | Table | Built by |
|---|---|---|
| First (`gogui`) | hand-built: button header plus go-gui's `VirtualList` | an agent working from the spec, like the article's apps |
| Second (`gogui-fit`) | go-gui's `DataGrid` with a data source over the core, pages of 5,000 rows | an agent under the article's second-pass rules, told to use `DataGrid` and fit the layout |

## Results

Desktop, median of 5 runs. Bold is the best figure in the row.

| | gogui | gogui-fit | mygoui | fyne | cogent |
|---|---|---|---|---|---|
| Ready (ms) | 277 | 340 | **236** | 572 | 555 |
| Jump to row 500k (ms) | 85 | 73 | **69** | 146 | 113 |
| Table scroll, max frame gap (ms) | 17 | 16 | **13** | 40 | 19 |
| Switch to cards (ms) | 117 | 58 | **51** | 104 | 112 |
| Sort (ms) | 262 | 259 | **241** | 254 | 262 |
| Peak memory (MB) | 279 | 267 | **219** | 1412 | 513 |
| Cold build (s) | **12.7** | 13.1 | 13.7 | 19.1 | 23.5 |
| Style, blind (1 to 5, one pass) | 2 | 4 | **5** | 3 | 1 |

Browser (Chrome, single run, frame gaps timed by the browser for all).

| | gogui | gogui-fit | mygoui | fyne | cogent |
|---|---|---|---|---|---|
| Download, gzip (MB) | 6.5 | 6.8 | **4.0** | 12.4 | 6.9 |
| Ready (ms) | **720** | 732 | 921 | 830 | 1,453 |
| Table scroll, max frame gap (ms) | 151 | 167 | **67** | 217 | 317 |
| Cards scroll, max frame gap (ms) | 151 | 133 | **33** | 135 | 717 |

**Hands-on (macOS).** Menus, shortcuts, native dialogs, notification, Reveal,
theme and state restore all worked, and scrolling was fast in both passes. The
tables were not:

- First pass: the vertical scrollbar sat past the right edge of a wider
  horizontal scroll area, the detail form overflowed, the last column did not
  stretch.
- Second pass: the table pages (a scrollbar per 5,000-row page, prev/next and
  Jump below it, arrow keys stop at the page edge). Stock bars and star ratings
  are text, with stray lines in the bars and some stars drawn as boxes. Table
  content shows through a gap under the horizontal scrollbar, and the cards
  scrollbar does not follow the mouse wheel.

**First pass**

[![go-gui first pass, dark theme, table view](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/gogui-desktop.png)](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/gogui-desktop.png)

**Second pass (`DataGrid`)**

[![go-gui second pass with DataGrid, dark theme, table view](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/gogui-fit-desktop.png)](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/gogui-fit-desktop.png)

**First pass, browser**

[![go-gui first pass in Chrome, dark theme, table view](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/gogui-web.png)](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/gogui-web.png)

**Second pass, browser**

[![go-gui second pass in Chrome, dark theme, table view](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/gogui-fit-web.png)](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/gogui-fit-web.png)


## The three questions

**A lot of data?** Fast to scroll on the desktop, close to MyGo native UI. But
the stock `DataGrid` pages large data rather than scrolling it, and a
continuous table means building your own.

**Need cgo?** Not on Linux or Windows: only libEGL at run time (plus the
Wayland libraries on Wayland), X11 in pure Go. On macOS it needs cgo for Metal.
The article's cgo test core (a core that needs cgo) runs on the desktop, and in
the browser through `web-remote`, where the core runs on a Go server.

**Web, native, or both?** Both from one Go UI codebase, at about half of Fyne's
download size, with visible scroll stutter in the browser. On Linux (the
article's container check, X11 and Wayland) it starts on both, native Wayland
by default, but the menu bar is macOS only, with no in-window fallback, so the
app has no menus and Ctrl+Q does nothing. File dialogs run zenity or kdialog if
installed, otherwise go-gui's own in-window browser; notifications go through
`notify-send`. Real GNOME/KDE sessions and the GPU path were not tested. It is
at v0.96.0, released daily to weekly, with breaking changes in recent minor
versions.

---

Code: [`apps/gogui/`](https://github.com/justinfx/go-gui-experiments/tree/main/apps/gogui), [`apps/gogui-fit/`](https://github.com/justinfx/go-gui-experiments/tree/main/apps/gogui-fit) · Full results:
[`results/gogui/`](https://github.com/justinfx/go-gui-experiments/tree/main/results/gogui), [`results/gogui-fit/`](https://github.com/justinfx/go-gui-experiments/tree/main/results/gogui-fit) ·
Method: [`results/HARNESS_CHANGES.md`](https://github.com/justinfx/go-gui-experiments/blob/main/results/HARNESS_CHANGES.md) (E5), [`results/DEVIATIONS.md`](https://github.com/justinfx/go-gui-experiments/blob/main/results/DEVIATIONS.md), style pass [`results/judge-addenda3/`](https://github.com/justinfx/go-gui-experiments/tree/main/results/judge-addenda3)
