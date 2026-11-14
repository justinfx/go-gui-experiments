> An addendum to [One Go App, Served Five Ways](https://justinfx.com/2026/11/10/go-gui-five-ways/).
> Same spec, same shared core, same harness. Other addenda: [[Home]].

**Suggested in** [r/golang](https://www.reddit.com/r/golang/s/AKasb7nJ4P) · **Stack** [MyGo](https://mygo.run) v0.9.0 ·
**Compared with** Wails (webview mode), Fyne and Cogent (native UI mode) · **Measured** 2026-11-14, same machine as the article

## At a glance

> **Verdict:** in native UI mode, the strongest self-drawn option measured so
> far, on the desktop and in the browser. Webview mode is on par with Wails but
> desktop only.
>
> **Stronger than the comparators at:** native UI mode is fastest on nearly
> every desktop row and has the smoothest browser scroll of any Go WASM build
> (ahead of Fyne and Cogent), with the top blind style score; webview mode
> builds faster and uses less memory than Wails; neither mode needs cgo.
>
> **Weaker at:** webview mode has no browser target (Wails can serve its UI);
> maturity (v0.9.0, pre-1.0, no stated stability policy, a shorter history than
> Fyne and Cogent); GTK, WebKit and GPU libraries must be on the user's machine.
>
> | | |
> |---|---|
> | **A lot of data?** | Native UI mode is the fastest measured; webview mode scrolls like Wails, stutter included. |
> | **Need cgo?** | No, in either mode; a cgo core still runs on the desktop. |
> | **Web, native, or both?** | Native UI mode reaches both from one codebase; webview mode is desktop only. |

The rest of this page is the evidence for the box above.

## What it is

A pure-Go desktop framework with two modes, which one app can mix. **Webview
mode** shows a web page in the system webview, with Go services bound to a
generated TypeScript client, like Wails. **Native UI mode** draws the UI itself
on the GPU, like Fyne and Cogent, and also builds to WASM for the browser.

| Mode | What draws the UI | Web target | Built by |
|---|---|---|---|
| Webview (`mygo`) | the system webview | none | an agent porting the article's Wails app |
| Native UI (`mygoui`) | MyGo, on the GPU | WASM | an agent working from the spec, like the article's apps |

## Results

**Webview mode (`mygo`)**

[![MyGo webview mode, dark theme, table view](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/mygo-desktop.png)](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/mygo-desktop.png)

**Native UI mode (`mygoui`)**

[![MyGo native UI mode, dark theme, table view](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/mygoui-desktop.png)](https://raw.githubusercontent.com/wiki/justinfx/go-gui-experiments/images/mygoui-desktop.png)

Desktop, median of 5 runs. Bold is the best figure in the row.

| | mygo | wails | mygoui | fyne | cogent |
|---|---|---|---|---|---|
| Ready (ms) | 628 | 623 | **236** | 572 | 555 |
| Jump to row 500k (ms) | 84 | 90 | **69** | 146 | 113 |
| Table scroll, max frame gap (ms) | 71 | 91 | **13** | 40 | 19 |
| Blank-row steps while scrolling | 54 | 64 | 23 | **0** | 24 |
| Switch to cards (ms) | 191 | 194 | **51** | 104 | 112 |
| Sort (ms) | 280 | 280 | **241** | 254 | 262 |
| Peak memory (MB) | 641 | 824 | **219** | 1412 | 513 |
| Cold build (s) | 15.1 | 39.2 | **13.7** | 19.1 | 23.5 |
| Style, blind (1 to 5) | 4 | 4 | **5** | 3 | 2 |

Browser (Chrome, single run). Frame gaps here are timed by the browser for all
three apps, so they compare with each other, not with the article's web table.

| | mygoui | fyne | cogent |
|---|---|---|---|
| Download, gzip (MB) | **4.0** | 12.4 | 6.9 |
| Ready (ms) | 921 | **830** | 1,453 |
| Jump (ms) | **54** | 187 | 138 |
| Table scroll, max frame gap (ms) | **67** | 217 | 317 |
| Cards scroll, max frame gap (ms) | **33** | 135 | 717 |

In a hands-on check, both modes had every native feature working (menus,
shortcuts, dialogs, notification, clipboard, Reveal, state restore), and felt
the same except scrolling, which was flawless in native UI mode.

Desktop, median of 5 runs. Bold is the best figure in the row.

| | mygo | wails | mygoui | fyne | cogent |
|---|---|---|---|---|---|
| Ready (ms) | 628 | 623 | **236** | 572 | 555 |
| Jump to row 500k (ms) | 84 | 90 | **69** | 146 | 113 |
| Table scroll, max frame gap (ms) | 71 | 91 | **13** | 40 | 19 |
| Blank-row steps while scrolling | 54 | 64 | 23 | **0** | 24 |
| Switch to cards (ms) | 191 | 194 | **51** | 104 | 112 |
| Sort (ms) | 280 | 280 | **241** | 254 | 262 |
| Peak memory (MB) | 641 | 824 | **219** | 1412 | 513 |
| Cold build (s) | 15.1 | 39.2 | **13.7** | 19.1 | 23.5 |
| Style, blind (1 to 5) | 4 | 4 | **5** | 3 | 2 |

Browser (Chrome, single run). Frame gaps here are timed by the browser for all
three apps, so they compare with each other, not with the article's web table.

| | mygoui | fyne | cogent |
|---|---|---|---|
| Download, gzip (MB) | **4.0** | 12.4 | 6.9 |
| Ready (ms) | 921 | **830** | 1,453 |
| Jump (ms) | **54** | 187 | 138 |
| Table scroll, max frame gap (ms) | **67** | 217 | 317 |
| Cards scroll, max frame gap (ms) | **33** | 135 | 717 |

In a hands-on check, both modes had every native feature working (menus,
shortcuts, dialogs, notification, clipboard, Reveal, state restore), and felt
the same except scrolling, which was flawless in native UI mode.

## The three questions

**A lot of data?** Webview mode behaves like Wails, including the WebKit scroll
stutter. Native UI mode is the fastest app measured on the desktop so far, and
the only Go WASM build that scrolls without visible stutter in the browser.

**Need cgo?** Neither mode needs it, and a cgo core still runs on the desktop.
In the browser, as with Fyne and Cogent, a cgo core cannot run inside WASM but
works with `web-remote`, where the core runs on a Go server.
GTK, WebKit and GPU libraries are loaded at launch rather than linked, so they
must be installed on the user's machine, and a missing one fails at launch,
not at build.

**Web, native, or both?** Webview mode is desktop only: unlike Wails, it has no
server mode, so the same UI cannot be served to a browser. Native UI mode
reaches both from one Go UI codebase, at a third of Fyne's download size. An app
can mix the two modes, but only its native UI windows can go to the browser.
On Linux (the article's container check, X11 and Wayland) both modes start,
with native GTK menus and dialogs, notifications and clipboard. Reveal selects
the file in the file manager, which no article app did. Real GNOME/KDE sessions
and the Linux GPU path were not tested. MyGo is at v0.9.0 with no stated
stability policy; Fyne and Cogent have longer release histories.

---

Code: [`apps/mygo/`](https://github.com/justinfx/go-gui-experiments/tree/main/apps/mygo), [`apps/mygoui/`](https://github.com/justinfx/go-gui-experiments/tree/main/apps/mygoui) · Full results:
[`results/mygo/`](https://github.com/justinfx/go-gui-experiments/tree/main/results/mygo), [`results/mygoui/`](https://github.com/justinfx/go-gui-experiments/tree/main/results/mygoui) ·
Method changes: [`results/HARNESS_CHANGES.md`](https://github.com/justinfx/go-gui-experiments/blob/main/results/HARNESS_CHANGES.md) (E1 to E4), [`results/DEVIATIONS.md`](https://github.com/justinfx/go-gui-experiments/blob/main/results/DEVIATIONS.md)
