# Capabilities: MyGo v0.9.0 (web frontend mode, port of apps/wails)

Status values: native / framework-drawn / fallback / gap / not feasible. "Verified" means exercised by a test or a run on this machine (macOS 26.6.2, desktop unlocked). Native menu, native dialogs, OS notification, Finder reveal and window size restore were built against the MyGo source and docs but NOT exercised: no automated way was used to drive native UI, and notifications need an app bundle (see below). Evidence paths are under `~/go/pkg/mod/github.com/egoist/mygo@v0.9.0/`.

| Requirement | Status | How it was done | Notes |
|---|---|---|---|
| Window title, 1280x800, min 1024x640 | native | `mygo.WindowOptions{Title, Width, Height, MinWidth, MinHeight}` in `main_desktop.go` | Screenshot: `screenshots/main-system-dark.png`. |
| Restore saved window size | native | `WindowOptions.StateKey: "main"` (window_state.go); the prefs size from core is still written via `Window.OnResize` | StateKey restore not exercised across two launches. |
| Frontend (search, categories, status, go-to-row, views, split, table, cards, custom cells, form, status bar) | framework-drawn | Unchanged Svelte code from apps/wails | Bench and screenshot show it running in WKWebView. |
| Go services bound to frontend | native | `mygo.Bind(svc)`; `mygo generate` writes `frontend/src/mygo.ts` (types, `ItemService` object, `events`) | Struct types and doc comments carry through. Verified by build, vitest and the Go-service e2e. |
| Events Go to UI | native | `mygo.NewEvent[T]` + `Broadcast` (events.go); frontend `events.exportProgress.on(...)` | Payload arrives directly, not wrapped in `{data}`. The harness (`main_e2e.go`) does not test MyGo's event path. Desktop export progress ran in the bench (export metric). |
| Native menu | native | `mygo.NewMenu`, `MenuItem{Label, Accelerator, Role, Type: MenuItemRadio, Submenu, Click}`, `App.SetMenu` | Radio checks and enabled state updated through `SetChecked`/`SetEnabled`. Not visually verified. |
| Save dialog (CSV) / open dialog (.txt) | native | `mygo.Dialog.Save` / `Open` with `FileFilter`, `Parent` = window | Bench mode bypasses the save dialog. Not exercised. |
| Error dialog | native | `mygo.Dialog.Error` | Not exercised. |
| Clipboard | native | `mygo.Clipboard.WriteText` | Not exercised. |
| Reveal in Finder | native | `mygo.Shell.ShowItemInFolder(path)` | Returns nothing, so errors cannot be reported. Not exercised. |
| OS notification | native (bundle only) | `mygo.NewNotification(...).Show()` | On macOS it needs an app bundle (`App.IsPackaged`). The harness launches the bare `bin/gguix-mygo`, so the app skips the notification there (returns an error) and the UI toast shows instead. Works only from `bin/GGUIX-mygo.app`; not tried. |
| Quit with prefs flush | native | `App.OnBeforeQuit` + `PreventDefault`, frontend ack through a bound call, then `App.Quit` | Not exercised. |
| Theme live following | native | mode-watcher in the webview | Unchanged. |
| Host panel | native | `host.Current`; reveal via `Shell` | |
| Web target (server mode, browser) | gap | Not supported. `make web`, `web-remote`, `bench-web` print UNSUPPORTED and exit 3 | Evidence: the module has `*_js.go` files (`backend_js.go`, `bridge_js.go`, `ipc_js.go`, `channel_js.go`, `protocol_js.go`, `updater_js.go`, `window_state_js.go`, `single_instance_js.go`, `typescript_js.go`, `random_js.go`, `locale_file_js.go`). They are for native UI (`GOOS=js GOARCH=wasm`, drawn with WebGL). `backend_js.go` returns `web.New()`; `bridge_js.go` says "Browsers show no web pages ... and so have no bridge"; `ipc_js.go` says `Bind`, `BindAs` and `Use` "bind nothing". `AGENTS.md` scopes browsers to native UI. A web-frontend app cannot call Go from a browser. |
| Browser fallback of the frontend | fallback (tests only) | `main_e2e.go` (tag `e2e`) serves the built frontend and the real `ItemService` over HTTP, with `e2e/shim.js` standing in for `window.mygo` | Test-only, not a MyGo feature or a shipped target. It tests the app, not MyGo's IPC. |
| T1 to T13 | pass | Playwright against the harness, desktop platform faked (T12 expects Source native, Reveal enabled) | 14 Playwright tests pass. |
| Go and vitest tests | pass | `go test`, `vitest` | `GGUIX_TESTS passed=34 failed=0 not_feasible=0`. |
| Bench, desktop | works | `GGUIX_BENCH=1` run of `bin/gguix-mygo`; all metric lines (ready 670 ms, jump, scroll, cards_scroll, sort, search, export, done) printed in about 20 s | Single launch, not a benchmark. |
| cgoprobe desktop | PASS | `CGO_ENABLED=1 GOFLAGS=-tags=cgoprobe mygo build`, then the export tests with the C checksum | MyGo sets `CGO_ENABLED=0` only when the variable is empty (cmd/mygo/proc.go:74). Turning cgo on means exporting it (and the clang toolchain); GOFLAGS carries extra tags through `mygo build`. |
| cgoprobe web | n/a | no web target | |
| Cross build windows / linux | not run | `make cross-windows` / `cross-linux` call `mygo build -platform ...`; not run by instruction | MyGo's own docs say Linux needs WebKitGTK at run time; Linux and Windows packages are produced by `mygo build` natively (CGO_ENABLED=0). |
| Package | native | `mygo build -skip-dmg`, copied to `bin/GGUIX-mygo.app` (ad hoc signed by mygo) | .app is 10.5 MB; not launched. |
