# GGUIX - GoTTH

The GGUIX reference application built with GoTTH: Go, templ, Tailwind and htmx, with
shadcn-templ components. One Go server renders everything. It runs as a web
target on `http://localhost:8090` and, inside a glaze window (pure Go, no cgo), as a
macOS desktop app.

## Prerequisites

- macOS on Apple Silicon (Intel works with the x64 Tailwind binary, picked by `make deps`)
- Homebrew and the Xcode Command Line Tools (`xcode-select --install`)
- Go 1.27.3 (`brew install go` gives the current release; the module needs go 1.27)
- Optional for `make test` scenario T9: Node 22 or newer and Google Chrome
- Nothing extra for the cross builds: the desktop shell is pure Go (CGO_ENABLED=0), no Docker or C toolchain

No npm packages are used. htmx and htmx-ext-sse are vendored under `static/htmx/`.

## Bootstrap from a clean machine

```sh
git clone <repo> gguix-work && cd gguix-work/apps/gotth   # or copy the workspace
make deps      # templ v0.3.1100, shadcn-templ v2.0.0 (go install), Tailwind v4.3.5 into bin/tools, htmx files
make run       # desktop app
make web       # web app on http://localhost:8090
```

How the project itself was created (already done, recorded for reproduction):

```sh
go install github.com/a-h/templ/cmd/templ@v0.3.1100
go install github.com/axadrn/shadcn-templ/v2/cmd/shadcn-templ@v2.0.0
cd apps && shadcn-templ init gotth --template templ --base-color neutral --silent
# module renamed to gguix/apps/gotth, then, one at a time:
shadcn-templ add button input select checkbox radio-group slider dialog alert-dialog popover \
  progress calendar toast card badge table textarea label field native-select toggle-group menubar
```

## Targets

| Target | What it does |
|---|---|
| `make deps` | Installs the tools above. Safe to run again. |
| `make dev` | Hot reload: Tailwind watch plus `templ generate --watch` proxying to the app on :8090 (browser on :7331). Static files are read from disk (`GOTTH_DEV=1`). |
| `make build` | Desktop binary `bin/gguix-gotth`. |
| `make package` | `bin/GGUIX-gotth.app`. glaze has no packaging tool, so the bundle (binary plus `desktop/Info.plist`) is assembled by the Makefile. |
| `make run` | Build and run the desktop app. |
| `make web` | Build and run the web server on http://localhost:8090. If another program holds 127.0.0.1:8090 the server uses `[::1]:8090`, which `http://localhost:8090` also reaches, then the next ports. |
| `make web-remote` | Same as `web`, with a note that the core already runs on the server. |
| `make test` | Go tests T1 to T12 (T9 drives Chrome through `tests/browser/t9.test.mjs`). Ends with `GGUIX_TESTS passed=<n> failed=<n> not_feasible=<n>`. |
| `make bench` | Desktop app in bench mode (`GGUIX_BENCH=1`), prints the eight `GGUIX_METRIC` lines and quits. |
| `make bench-web` | Serves the web target; open `http://localhost:8090/?bench=1` and read the browser console. |
| `make cgoprobe` | Rebuilds the web server and the desktop shell with `-tags cgoprobe`. |
| `make cross-windows`, `make cross-linux` | `CGO_ENABLED=0 go build` of the web server and the desktop shell (glaze needs no C toolchain). |
| `make clean` | Removes `bin/` (except `bin/tools`) and generated files. |

## Layout

```
marker.go            BuildMarker (shared)
server.go            routes, prefs, handlers for the list, form and theme (shared)
export.go            export jobs, SSE progress, web download (shared)
handlers_desktop.go  native dialogs, reveal, quit (desktop only, routes registered in desktop mode)
metric_bench.go      bench metric sink (bench)
server_test.go       T1 to T12 (test)
views/               templ components (shared); views/desktop is the in-page menu bar (desktop)
static/app.js        shared client script; static/web, static/desktop, static/bench per target
cmd/web              web server (web)
cmd/desktop          webview shell and macOS helpers (desktop)
components/, assets/ shadcn-templ output (shared)
```

Target-specific code is marked as described in the brief: whole files by suffix or directory,
and `gguix:target` / `gguix:end` comment blocks inside shared files. Everything under
`cmd/desktop` and `desktop/` is desktop code; everything under `cmd/web` is web code.

## Environment variables

`GGUIX_ROWS`, `GGUIX_LATENCY_MS`, `GGUIX_EXPORT_RPS`, `GGUIX_PREFS_DIR` (from the core),
`GGUIX_BENCH`, `GGUIX_BENCH_EXPORT`, `GOTTH_DEV=1` (serve static files from disk),
`GOTTH_LOG=1` (log every request), `GOTTH_PORT` (web port, default 8090).
