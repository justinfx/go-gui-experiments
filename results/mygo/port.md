# Port notes: apps/wails to apps/mygo

## What stayed the same
`core/` is untouched. Svelte components, styling, paging, scaled scroll, bench code, `view.go`, `export.go`, `service_test.go` and `marker.go` / `marker.ts` are unchanged. The desktop binary is `bin/gguix-mygo` and prints the same `GGUIX_METRIC` lines (verified with a bench launch).

## Files changed (diff vs apps/wails, about 620 changed lines, excluding lockfiles and the generated client)
- Go: `service.go` (events via `mygo.NewEvent`, service name method dropped, framework string), `main.go`, `main_desktop.go`, `menu_desktop.go`, `platform_desktop.go` (rewritten against `mygo`), new `backend.go` (`openBackend` moved out), new `main_e2e.go` (tag `e2e`, test-only), `go.mod`. Removed: `main_web.go`, `platform_web.go`, `gzip_web.go`, `Taskfile.yml`, `build/` (Wails templates).
- Config: new `mygo.config.ts` and `resources/icon.png`; `.gitignore`, `Makefile`, `README.md`.
- Frontend: `api.ts` (generated client plus name mapping), `state.svelte.ts` (events import and unwrapped payloads), `Dialogs.svelte` (title), `index.html`, `vite.config.ts` (no Wails plugin), `package.json` (mygo-runtime, mygo-cli), `playwright.config.ts`, `e2e/gguix.spec.ts` (shim init script, T12), new `e2e/shim.js`, generated `src/mygo.ts` (362 lines, replaces the bindings folder).
- Go line count: 1642 vs 1718 (the web files are gone, the harness is added).

## Native vs fallback
Native through MyGo: window options, menu, file dialogs, error dialog, clipboard, reveal, notification (bundle only), window state, quit hook, events and bindings. Fallback: none in the desktop app. The browser path no longer exists (web code branches remain in the frontend but never run, since the target is always "desktop"). The e2e harness is a test fallback only.

## Limits, with evidence
- No browser or server mode (see capabilities.md). `Bind` is a no-op under js/wasm (`ipc_js.go`).
- Reveal returns no error (`Shell.ShowItemInFolder` has no result).
- Notifications are skipped when not running from a bundle (harness case).
- The frontend is embedded by `mygo build` only (`cmd/mygo/embed.go`).
- Not verified: native menu appearance and behavior, dialogs, clipboard, reveal, StateKey restore, notification from a bundle, Linux and Windows builds, `make package` launch.

## Results
`make test`: `GGUIX_TESTS passed=34 failed=0 not_feasible=0`. `make build` about 5 s. `make cgoprobe`: desktop PASS. Bench launch: all eight metric lines. Screenshot: `screenshots/main-system-dark.png` (app window only, via `bench/shot.sh`).
