# GGUIX: go-gui

The GGUIX reference app built with go-gui v0.96.0 (immediate-mode, GPU drawn:
Metal on macOS, GL on Linux/Windows, Canvas2D/WASM in the browser).

## Prerequisites

- Go 1.27.3
- macOS: Xcode Command Line Tools (the Metal backend is Objective-C, so cgo is on).
- Linux/Windows desktop builds are cgo-free.

## Bootstrap from a clean machine

go-gui has no project template; its README says `go get github.com/go-gui-org/go-gui`.
This module was created with:

```
go mod init gguix/apps/gogui   # then the core require/replace lines
go get github.com/go-gui-org/go-gui@v0.96.0
go mod tidy
```

Then, from `apps/gogui`:

```
make deps     # go mod download
make build    # bin/gguix-gogui
make run
```

## Targets

| Target | What it does |
|---|---|
| `deps` | `go mod download` |
| `dev` | `go run .` (no hot reload in go-gui) |
| `build` | `bin/gguix-gogui` |
| `package` | `bin/GGUIX-gogui.app` via go-gui's `cmd/buildapp` (ad-hoc signed) |
| `run` | build and run |
| `web` | WASM build into `bin/web`, served by the core server on :8090 |
| `web-remote` | same with `-tags remote`, the core server provides the API |
| `test` | headless tests (`gui.NewTestWindow`), prints `GGUIX_TESTS ...` |
| `bench` / `bench-web` | bench mode (desktop / browser with `?bench=1`) |
| `cgoprobe` | `-tags cgoprobe` builds per target |
| `cross-windows` / `cross-linux` | `CGO_ENABLED=0` cross builds |
| `clean` | remove `bin/` |
