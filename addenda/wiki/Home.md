Companion wiki for [One Go App, Served Five Ways](https://justinfx.com/2026/11/10/go-gui-five-ways/). The code, harness and raw results are in this repository; see the [README](https://github.com/justinfx/go-gui-experiments#readme).

## Addenda

Options suggested after the article was published. Each one is built against the same spec and shared core, measured with the same harness on the same machine, and compared with the article's closest options.

| Option | Compared with | A lot of data? | Needs cgo? | Web, native, or both? |
|---|---|---|---|---|
| [MyGo, webview mode](Addendum-MyGo) | Wails | like Wails | no | desktop only |
| [MyGo, native UI mode](Addendum-MyGo) | Fyne, Cogent | fastest measured, desktop and browser | no | both (WASM) |
| [go-gui](Addendum-go-gui) | MyGo native UI, Fyne, Cogent | fast, but its grid pages and a polished table needs custom work | no (Linux, Windows); yes on macOS | both (WASM) |

Suggestions are welcome as [issues](https://github.com/justinfx/go-gui-experiments/issues). The template each addendum follows is [`addenda/TEMPLATE.md`](https://github.com/justinfx/go-gui-experiments/blob/main/addenda/TEMPLATE.md).
