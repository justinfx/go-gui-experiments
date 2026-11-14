# Addenda

Options suggested after [One Go App, Served Five Ways](https://justinfx.com/2026/11/10/go-gui-five-ways/)
was published. Each one was built against the same spec and shared core and
measured with the same harness, then compared with the article's results.

| Option | Compared with | A lot of data? | cgo? | Web, native, or both? |
|---|---|---|---|---|
| [MyGo](mygo.md), webview mode | Wails | like Wails | not needed | desktop only |
| [MyGo](mygo.md), native UI mode | Fyne, Cogent | fastest measured, desktop and browser | not needed | both (WASM) |

To add one: build it from a brief, collect everything in
`briefs/ADDENDUM_RESULTS_CHECKLIST.md`, write the page from
[TEMPLATE.md](TEMPLATE.md), and add a row here.
