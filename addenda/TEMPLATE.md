# {{NAME}}

> An addendum to [One Go App, Served Five Ways](https://justinfx.com/2026/11/10/go-gui-five-ways/).
> Same spec, same shared core, same harness. Other addenda: [index](README.md).

**Suggested by** {{link}} · **Stack** {{framework + version, UI layer, shell}} ·
**Compared with** {{one or more article options}} · **Measured** {{date}}

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

**A lot of data?** {{One or two sentences, in terms of what the user notices.}}

**Need cgo?** {{Can a cgo core run, and what must be on the machine at run time.}}

**Web, native, or both?** {{What it reaches, and what reaching both costs.}}

With more than one mode, answer each question per mode where the answers differ.

## Worth knowing

{{Optional. Only gaps or friction a reader would want before choosing it.
Delete the section if there is nothing.}}

---

Code: [`apps/{{id}}/`](../apps/{{id}}/) · Full results: [`results/{{id}}/`](../results/{{id}}/)
