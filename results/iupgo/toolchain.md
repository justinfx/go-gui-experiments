# Toolchain: IUP-Go

| Tool | Version | Install | Size | Needed for | cgo-related |
|---|---|---|---|---|---|
| Go | 1.27.3 | preinstalled | n/a | all | no |
| Xcode Command Line Tools | preinstalled | `xcode-select --install` | n/a | build (cgo, Cocoa) | yes |
| iup-go/iup | main cbcdf09 (v0.32.1-0.20261114161355-cbcdf09dde04) | `go get github.com/gen2brain/iup-go/iup@cbcdf09` | a few MB of sources in the module cache | build | yes (bundled C sources) |
| purego | v0.10.1 | pulled by iup-go (used by the `CGO_ENABLED=0` mode) | small | build | no |
| iupkg | cbcdf09 | `go run github.com/gen2brain/iup-go/cmd/iupkg@cbcdf09` (no install) | downloads pkcs12, go-macho, x/crypto, circl | package | no |
| emsdk / Emscripten | emsdk main, emcc 7.0.0 | `git clone --depth 1 https://github.com/emscripten-core/emsdk.git ~/emsdk && cd ~/emsdk && ./emsdk install latest && ./emsdk activate latest` | 1.9 GB in `~/emsdk` (1.6 GB `upstream`, 196 MB `node`) | web (the IUP wasm module) | no |
| Writable copy of the iup module | cbcdf09 | `cp -R <iup module dir> ~/iupwasm/repo/iup` (the `make` rule does it) | 81 MB in `~/iupwasm`, built module `iup.wasm` 0.65 MB + `iup.js` 0.33 MB | web | no |
| Google Chrome (Playwright, `channel: chrome`) | installed | n/a | n/a | web test (service worker, screenshots) | no |
| Docker image `gguix-linux` | existing | `linux/dx.sh build` | 3.8 GB | Linux check | n/a |
| `libxkbcommon-tools` (container only) | 1.6.0 | `apt-get install libxkbcommon-tools` inside the throwaway container | small | Linux check (virtual keyboard under sway) | no |

Timings: the first cgo build of IUP took about 10 s of wall time here (warm Go build cache for other apps). emsdk clone + install + activate took about a minute on this network (start 09:40, done by 09:41). The emcc build of the IUP module (`build-wasm.sh -O -m`) took 13 s. The Linux GTK3 cgo build in the container took a few minutes the first time.

The emsdk and the writable module copy are global installs outside the repo (`~/emsdk`, `~/iupwasm`).

## Clean machine steps

Desktop (macOS, only Go, Homebrew and the Xcode Command Line Tools installed):

```
cd apps/iupgo
make deps
make run
```

Web (adds git and about 2 GB of disk):

```
cd apps/iupgo
make emsdk-install          # clones and installs emsdk into ~/emsdk
make web                    # builds the IUP module once (bash build-wasm.sh -O -m), then the app, serves http://localhost:8090
make web-remote             # same with -tags remote, served by the core server with its API
make bench-web              # same as web; open http://localhost:8090/?bench=1
```

Notes: `build-wasm.sh` must run under bash on macOS (its `#!/bin/sh` line breaks in `sh` there; the Makefile calls `bash`). The Go module cache is read-only and the script writes next to itself and expects the repo layout `repo/iup/external/wasm`, so the Makefile builds from a copy in `~/iupwasm`.
