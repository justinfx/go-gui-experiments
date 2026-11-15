# GGUIX reference app: NSGo

The GGUIX reference application built with [NSGo](https://github.com/nsgo-dev/nsgo) (native widgets from Go: AppKit on macOS, GTK 4 on Linux, DOM in WebAssembly), pinned at commit `2345b2f` (pseudo-version `v0.0.0-20261114190341-2345b2fa5aba`). Module `gguix/apps/nsgo`, with `core` replaced by `../../core`.

## Prerequisites

- Go 1.27.3 (the only requirement; no Homebrew packages, no Node, no extra CLI installs).
- macOS: nothing else. NSGo reaches AppKit through purego, so `CGO_ENABLED=0` builds work and the Xcode Command Line Tools are not needed to build this app.
- Linux (to run): GTK 4 and Pango/Cairo shared libraries on the machine (NSGo loads them at run time), plus a desktop portal (`xdg-desktop-portal` with a `gtk` or other FileChooser backend) for the file dialogs.

## From a clean machine

```sh
git clone <repo> && cd <repo>/apps/nsgo
make deps        # go mod download (the nsgo CLI is a `tool` directive in go.mod)
make run         # builds bin/gguix-nsgo and starts it
make web         # builds the web site into bin/web and serves http://localhost:8090
```

How the project was bootstrapped (already done, kept in the repo): `go mod init gguix/apps/nsgo`, `go get -tool github.com/nsgo-dev/nsgo/cmd/nsgo@2345b2f`, then `go run github.com/nsgo-dev/nsgo/cmd/nsgo@v0.0.0-20261114190341-2345b2fa5aba init -template go -name gguix-nsgo -module gguix/apps/nsgo .` (run before the `go.mod` was created, because `init` refuses a directory that already has one). The template's `app.go` was replaced by this app.

## Make targets

| Target | What it does |
|---|---|
| `deps` | `go mod download` |
| `dev` | `go tool nsgo dev` (hot reload: rebuild and restart on save) |
| `build` | `CGO_ENABLED=0 go build -o bin/gguix-nsgo .` |
| `package` | `go tool nsgo build -target macos_arm64`, then copies the `.app` to `bin/GGUIX-nsgo.app` |
| `run` | build, then run |
| `web` | `go tool nsgo build -target web`, copies the site to `bin/web`, serves it with `core/cmd/server` on `SERVE_PORT` (8090) |
| `web-remote` | same with `GOFLAGS=-tags=remote`; the page talks to the core server's HTTP API |
| `test` | `go test` (12 scenarios through `nsgotest`); last line `GGUIX_TESTS passed=<n> failed=<n> not_feasible=<n>` |
| `bench` | desktop bench mode (`GGUIX_BENCH=1`) |
| `bench-web` | builds and serves the web target; open `http://localhost:8090/?bench=1` |
| `cgoprobe` | rebuilds each target with `-tags cgoprobe` |
| `cross-windows`, `cross-linux` | `GOOS=... CGO_ENABLED=0 go build` |
| `clean` | removes `bin/` |

## Layout

`model.go` (state, pager, prefs, selection; shared), `view.go` (toolbar, table, cards, form, status bar, dialogs; shared), `cells.go` (stock bar, badge, stars, chip, tile; shared), `export.go`, `platform.go`, `platform_desktop.go`, `menu_desktop.go`, `main_desktop.go`, `platform_web.go`, `install_web.go`, `main_web.go`, `bench_bench.go`, `app_test.go`, `marker.go`. Target-specific files end in `_desktop.go` or `_web.go`; the Makefile marks its desktop and web blocks.
