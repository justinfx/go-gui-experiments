# GGUIX: IUP-Go

The GGUIX reference app built with IUP-Go (`github.com/gen2brain/iup-go/iup`, `main` commit `cbcdf09`), which wraps native controls (Cocoa on macOS, GTK3 on Linux, Win32 on Windows).

## Prerequisites

- Go 1.27.3
- macOS: Xcode Command Line Tools (cgo compiles IUP's bundled C and Objective-C sources)
- Linux: a C compiler and `libgtk-3-dev`
- Web: git, about 2 GB of disk for emsdk, Google Chrome to try it (the page needs a service worker for cross-origin isolation, so a plain `file://` or non-localhost origin will not work)

## Bootstrap from a clean machine

IUP-Go has no project template. This module was created with:

```
go mod init gguix/apps/iupgo     # then the core require/replace lines
go get github.com/gen2brain/iup-go/iup@cbcdf09
go mod tidy
```

Then, from `apps/iupgo`:

```
make deps     # go mod download
make build    # bin/gguix-iupgo (cgo build)
make run
```

## Targets

| Target | What it does |
|---|---|
| `deps` | `go mod download` |
| `dev` | `go run .` (no hot reload) |
| `build` | `bin/gguix-iupgo`, cgo build. `TABLE=matrix` (default, IupMatrixEx with drawn cells) or `TABLE=native` (IupTable); applies to dev, build, run, test, bench, package |
| `package` | `bin/GGUIX-iupgo.app` via `iupkg` (`go run github.com/gen2brain/iup-go/cmd/iupkg@cbcdf09`) |
| `run` | build and run |
| `emsdk-install` | clones and installs emsdk into `~/emsdk` (about 2 GB, one time) |
| `web` | builds the IUP wasm module once (from a copy in `~/iupwasm`), the Go app for js/wasm into `bin/web`, serves it on `:8090` with the core server |
| `web-remote` | same with `-tags remote`, the core server provides the API |
| `bench-web` | same as `web`; open `/?bench=1` |
| `test` | real-toolkit tests (window mapped, never shown), prints `GGUIX_TESTS ...` |
| `bench` | bench mode (`GGUIX_BENCH=1`) |
| `cgoprobe` | `-tags cgoprobe` build of the desktop target |
| `cross-windows`, `cross-linux` | `CGO_ENABLED=0` builds (IUP-Go loads a bundled prebuilt library at run time) |
| `clean` | remove `bin/` |
