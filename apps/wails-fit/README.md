# GGUIX - Wails

GGUIX reference application built with Wails v3.1.0 (Go backend, Svelte 5 + shadcn-svelte frontend in a system webview). The same Go service and frontend also run in Wails server mode as a web app.

## Prerequisites (macOS, Apple Silicon)

- Go 1.27.3 (`go version` must say go1.27.3 darwin/arm64)
- Homebrew, Node 26 with npm 11 (`brew install node`)
- Xcode Command Line Tools (`xcode-select --install`)
- Google Chrome (only for `make test`; Playwright drives the installed Chrome, nothing is downloaded)
- Docker Desktop (only for `cross-windows` and `cross-linux`)
- `python3` (ships with the Command Line Tools; used to total the test results)

## From a clean machine

```sh
cd apps/wails
make deps        # installs wails3 v3.1.0 into $(go env GOPATH)/bin and runs npm ci in frontend/
make run         # builds and runs the desktop app (as an ad-hoc signed dev .app bundle)
make web         # server mode on http://localhost:8090
```

The Makefile adds `$(go env GOPATH)/bin` to PATH itself. If you call `wails3` by hand, add it to your own PATH.

### How the project was bootstrapped (already done, for the record)

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0
wails3 init -n gguix-wails -t svelte -d apps/wails -mod gguix/apps/wails   # template "svelte" (Svelte + TypeScript + Vite)
cd frontend
npm install --save-exact svelte@5.57.3 tailwindcss@4.3.5 @tailwindcss/vite@4.3.5 @sveltejs/vite-plugin-svelte@7.3.1 vite@8.3.6 bits-ui@2.20.6 @tanstack/svelte-virtual@3.13.52
npx shadcn-svelte@1.8.0 init --preset vega --base-color neutral ...        # needs a terminal (interactive prompts)
npx shadcn-svelte@1.8.0 add -y button input select checkbox radio-group slider label textarea dialog alert-dialog dropdown-menu popover progress calendar table card badge sonner
npm install --save-dev --save-exact vitest@5.1.0 @playwright/test@1.65.0
npm install --save-dev @testing-library/svelte jsdom
```

## Make targets

| Target | What it does |
|---|---|
| `deps` | Installs wails3 and the npm packages. Idempotent. |
| `dev` | `wails3 dev`: Vite hot reload for the frontend, rebuild and restart for Go changes. |
| `build` | Desktop binary `bin/gguix-wails`. |
| `package` | `bin/GGUIX-wails.app` (ad-hoc signed) with `wails3 task package`, renamed. |
| `run` | Builds and runs the desktop app (inside `bin/gguix-wails.dev.app`, so notifications work). |
| `web` | Builds the server-mode binary (`-tags server`) and serves http://localhost:8090. |
| `web-remote` | Prints a NOTE, then the same as `web` (the core already runs on the server). |
| `test` | Go tests, Vitest, and Playwright against the server-mode build; ends with the `GGUIX_TESTS` line. |
| `bench` | Desktop app with `GGUIX_BENCH=1`; prints the eight `GGUIX_METRIC` lines. The window must be on an unlocked, visible screen (a locked or occluded WKWebView throttles timers). |
| `bench-web` | Serves the web target; open http://localhost:8090/?bench=1 and read the browser console. |
| `cgoprobe` | Rebuilds desktop and web with `-tags cgoprobe`; prints `CGOPROBE <target> PASS\|FAIL`. |
| `cross-windows`, `cross-linux` | Docker based cross builds via the template tasks (outcomes in results/wails/capabilities.md). |
| `clean` | Removes `bin/` and generated frontend files. |

## Layout

- `main.go`, `service*.go`, `export.go`, `view.go`: shared Go code (the `ItemService` bound to the frontend).
- `*_desktop.go`: desktop only (window, native menu, dialogs, notifications). `*_web.go`: web (server mode) only.
- `frontend/src`: Svelte app. `lib/data/rowstore.svelte.ts` is the data layer (lazy row model, see `results/wails/adapter.md`), `adapter_pages.go` is its Go batch endpoint, `lib/scaled-scroll.ts` maps scroll positions for lists taller than a browser allows, `bench/` is the bench script.
- `frontend/e2e`: Playwright scenarios T1 to T12 (server mode, Chrome).
- Port 8090 must be free for `make web`; the Playwright run uses 8191.
