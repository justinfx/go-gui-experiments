# Handoff: Wails GGUIX app (paused at coordinator request)

## State of the code
- Go side builds: `go vet ./...` and `go vet -tags server ./...` pass in apps/wails.
- Frontend does NOT build yet: `frontend/src/lib/state.svelte.ts` imports `./form.svelte` (FormModel) which does not exist, and no Svelte components beyond the cells and VirtualScroller exist.

## Done
- wails3 v3.1.0 installed; project from template `svelte`, flattened into apps/wails; go.mod rewritten (go 1.27, replace gguix/core).
- npm: pinned svelte, tailwind, vite, bits-ui, tanstack virtual, vitest 5.1.0, playwright 1.65.0, @testing-library/svelte, jsdom; shadcn-svelte 1.8.0 init (preset vega, neutral, needs pty script /tmp/ptyrun.py) and all requested components added.
- Go: marker.go, view.go (DTOs, palette), service.go (ItemService: AppInfo, Page, Get, Update, FindRow, LoadPrefs, SavePrefs, CopyRow, ImportNotes, HostInfo, Reveal*, Quit, SetMenuState, PrefsFlushed, Palette), export.go (jobs, StartExport/CancelExport, web serveExport), service_bench.go, platform*.go, main.go, main_desktop.go, menu_desktop.go, main_web.go (server mode, /export/<id> stream). Bindings generate OK (`wails3 generate bindings -clean=true -ts -i`).
- Frontend: lib/pager.ts (port of 4.6.1), api.ts, scaled-scroll.ts, components/VirtualScroller.svelte, cells (StatusBadge, Stars, StockBar), palette.svelte.ts, state.svelte.ts, marker.ts.
- results/wails/friction.md has 3 entries so far.

## Next steps
1. Write lib/form.svelte.ts (FormModel: loaded, fields name/category/price/quantity/status/rating tenths/featured/restock/notes, errors, dirty, saving, load, reset, toEdit, setErrors).
2. Components: App.svelte, main.ts (import app.css, ModeWatcher, Toaster), Toolbar, CategoryFilter (popover), DataTable (VirtualScroller, 37 px rows, resizable columns), CardList + Card (104 px item), DetailForm, StatusBar, HostDialog, AboutDialog, confirm AlertDialog, WebMenuBar (web only), bench/bench.ts (8 metrics, EmitMetric, Quit).
3. Remove build/ios, build/android, their Taskfile includes; set build/config.yml product info; Makefile (all targets in 3.2), README, set wails build task for server tags, rename package output to GGUIX-wails.app.
4. Tests: Go tests for ItemService, Vitest (pager + components), Playwright against server mode (channel chrome) for T1-T12; `make test` must end with GGUIX_TESTS line.
5. Verify notifications service starts on unbundled binary; cgoprobe, cross-windows, cross-linux (Docker image uses golang:1.26, core needs 1.27: expect problems).
6. Write toolchain.md, capabilities.md, finish friction.md summary.
Known notes for friction: tanstack svelte-virtual peer dep is svelte ^4; 1M rows exceed browser max element height (handled with scaled-scroll); Wails has no reveal-in-Finder API (core host used).

## Update (after resume)
Done: all frontend components, bench.ts, Makefile, scripts/gguix_tests.py, Go/Vitest/Playwright tests. `make test` ends with `GGUIX_TESTS passed=33 failed=0 not_feasible=0`.
Remaining: run make bench / run / package on desktop and fix issues, cgoprobe, cross builds, README, toolchain.md, capabilities.md, friction summary.
Note: port 8090 is occupied by other processes on this host; tests use 8191.
Update: all results files written; remaining item is a desktop bench rerun on an unlocked screen.
