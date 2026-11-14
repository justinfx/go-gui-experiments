# GGUIX - MyGo (native UI)

The GGUIX reference app built with MyGo v0.9.0's native UI (package `ui`, drawn by MyGo, no webview).

## Prerequisites

A Mac with Go 1.27.3 (the version in `go.mod`) and the Xcode Command Line Tools. Homebrew is only needed to install Go. MyGo's CLI is pinned by the `tool` line in `go.mod`, so no npm, Bun or other tool is needed. Nothing needs cgo (`CGO_ENABLED=0`).

## Bootstrap from a clean machine

The project started from MyGo's native template:

```sh
go run github.com/egoist/mygo/cmd/mygo@v0.9.0 init -template native boot
```

The result was moved here with module `gguix/apps/mygoui` (see `go.mod`; `mygo.json` has `"webview": false`).

```sh
cd apps/mygoui
make deps        # go mod download, builds the pinned mygo CLI
make run         # build bin/gguix-mygoui and start it
```

## Targets

| Target | What it does |
|---|---|
| `make deps` | Downloads modules and builds the pinned `mygo` CLI. |
| `make dev` | `go tool mygo dev` (rebuild and restart on change). |
| `make build` | `go tool mygo build -skip-dmg`, bare executable copied to `bin/gguix-mygoui`. |
| `make package` | `.app` bundle at `bin/GGUIX-mygoui.app`. |
| `make run` | Build and run. |
| `make web` | `go tool mygo build -platform web`, copied to `bin/web`, served on http://localhost:8090 by `core/cmd/server`. |
| `make web-remote` | Same with `-tags remote`, served with the core API. |
| `make test` | `go test`, prints `GGUIX_TESTS passed=<n> failed=<n> not_feasible=<n>`. |
| `make bench` | Desktop bench mode (`GGUIX_BENCH=1`), prints the eight `GGUIX_METRIC` events. |
| `make bench-web` | Builds and serves the web target; open `http://localhost:8090/?bench=1` and read the console. |
| `make cgoprobe` | Rebuilds each target with `-tags cgoprobe`. |
| `make cross-windows`, `make cross-linux` | `mygo build -platform windows/amd64` or `linux/amd64`. |
| `make clean` | Removes `bin/`, `build/`, `.mygo/`. |

Note: after `make build` the bare executable is replaced (not overwritten in place), because macOS kills a re-copied signed binary that keeps its old inode.

## Layout

Shared code: `app.go`, `view.go`, `cells.go`, `export.go`. Desktop only: `main_desktop.go`, `menu_desktop.go`, `platform_desktop.go`, `platform_shortcuts_desktop.go`. Web only: `main_web.go`, `platform_web.go`. Bench: `driver_bench.go`. Tests: `app_test.go`.
