# {{NAME}}

> An addendum to [One Go App, Served Five Ways](https://justinfx.com/2026/11/10/go-gui-five-ways/).
> Same spec, same shared core, same harness. Other addenda: [index](README.md).

**Suggested by** {{link}} · **Stack** {{framework + version, UI layer, shell}} ·
**Compared with** {{one or more article options}} · **Measured** {{date}}

## At a glance

> **Verdict:** {{One sentence: who should pick it, ranked against the
> comparators, in terms a reader would notice.}}
>
> **Stronger than the comparators at:** {{two to four short items, each naming
> who it beats, e.g. "desktop scroll (smoother than Fyne and Cogent)"}}
>
> **Weaker at:** {{two to four short items, same form, including maturity or a
> platform hole if it matters}}
>
> | | |
> |---|---|
> | **A lot of data?** | {{one line}} |
> | **Need cgo?** | {{one line}} |
> | **Web, native, or both?** | {{one line}} |

The rest of this page is the evidence for the box above.

## What it is

{{Two or three sentences: what the project is, and for each comparator the one
way it differs. If the option has more than one mode (for example a webview mode
and a self-drawn mode), give each mode its own row below and its own results
column, compared with the article options closest to that mode.}}

| Mode | Where Go runs | What draws the UI | Web target | Compared with |
|---|---|---|---|---|
| {{mode}} | {{desktop / server / browser}} | {{WebView / own renderer / native widgets}} | {{yes / no / how}} | {{article options}} |

## Results

Comparator figures are from the article's `results/REPORT.md`. Bold marks the
best figure in the article for that row.

Use one column per mode built and one per comparator; drop rows that do not
apply to any of them.

| | {{id}} | {{comparator}} | {{id-mode2}} | {{comparator 2}} | Article best |
|---|---|---|---|---|---|
| Ready, desktop (ms) | | | | | **443** (miqt) |
| Jump to row 500k (ms) | | | | | **90** (wails) |
| Table scroll, max frame gap (ms) | | | | | **19** (cogent) |
| Switch to cards (ms) | | | | | **104** (fyne) |
| Sort (ms) | | | | | **254** (fyne) |
| Peak memory (MB) | | | | | **513** (cogent) |
| Web initial transfer, gzip (KB) | | | | | **156** (gotth) |
| Cold build (s) | | | | | **9.4** (gotth) |

## The three questions

**A lot of data?** {{A short paragraph, in terms of what the user notices.}}

**Need cgo?** {{Can a cgo core run, and what must be on the machine at run time.}}

**Web, native, or both?** {{What it reaches, and what reaching both costs.}}

With more than one mode, answer each question per mode where the answers differ.
Platform notes (Linux check, runtime libraries, maturity) go under the question
they bear on. There is no separate summary or "worth knowing" section: the box at
the top is the summary.

---

Code: [`apps/{{id}}/`](../apps/{{id}}/) · Full results: [`results/{{id}}/`](../results/{{id}}/)
