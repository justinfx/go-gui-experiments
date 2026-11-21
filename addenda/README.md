# Addenda

Options suggested after [One Go App, Served Five Ways](https://justinfx.com/2026/11/10/go-gui-five-ways/)
was published. Each one was built against the same spec and shared core and
measured with the same harness, then compared with the article's results.

| Option | Compared with | A lot of data? | cgo? | Web, native, or both? |
|---|---|---|---|---|
| [MyGo](mygo.md), webview mode | Wails | like Wails | not needed | desktop only |
| [MyGo](mygo.md), native UI mode | Fyne, Cogent | fastest measured, desktop and browser | not needed | both (WASM) |
| [go-gui](gogui.md) | MyGo native UI, Fyne, Cogent | fast, but its grid pages and a polished table needs custom work | no (Linux, Windows); yes on macOS | both (WASM) |
| [NSGo](nsgo.md) | MyGo native UI, go-gui, Fyne, Cogent | yes on the desktop; the browser shows loading rows on fast scrolls | not needed, any desktop | both (native controls, DOM in the browser) |
| [IUP-Go](iupgo.md) | MIQT, MyGo native UI, NSGo, Fyne | yes with its drawn grid (plain look); the native table stutters | yes by default; or a prebuilt C library | both, but the browser build needs Emscripten |
| [Gophics](gophics.md) | MyGo native UI, NSGo, go-gui, Fyne | yes, desktop and browser, no loading rows | not needed, any desktop | both, self-drawn everywhere; browser needs WebGPU |

To add one: build it from a brief, collect everything in
`briefs/ADDENDUM_RESULTS_CHECKLIST.md`, write the page from
[TEMPLATE.md](TEMPLATE.md), and add a row here.
