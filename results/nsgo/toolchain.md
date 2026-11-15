# Toolchain: NSGo

| Name | Version | Install command | Size | Needed for | cgo-related |
|---|---|---|---|---|---|
| Go | go1.27.3 darwin/arm64 (already installed) | n/a | n/a | everything | no |
| github.com/nsgo-dev/nsgo | `v0.0.0-20261114190341-2345b2fa5aba` (commit `2345b2f`) | `go get github.com/nsgo-dev/nsgo@2345b2f` (pulled in by `go mod tidy`) | module source in the module cache (tens of MB; the repository carries docs, a website and examples) | build, runtime (linked) | no: AppKit, GTK 4 and the DOM are reached with purego and `syscall/js` |
| `nsgo` CLI (`github.com/nsgo-dev/nsgo/cmd/nsgo`) | same module | `go get -tool github.com/nsgo-dev/nsgo/cmd/nsgo@2345b2f`, run as `go tool nsgo` | built on first use from the module cache | dev, package, web | no |
| github.com/ebitengine/purego | v0.11.3 (NSGo's own dependency) | `go mod tidy` after importing it in `identity_desktop.go` | small | Linux build (window identity) | no (purego with `CGO_ENABLED=0`) |
| esbuild, goja, brotli, x/image, x/net, x/text, yaml (indirect) | `esbuild v0.28.2`, `goja 2026-10-27`, `brotli v1.3.0`, ... | pulled by the NSGo module | module cache | build of the `nsgo` tool (TSX support); not linked into the app except `x/image`, `x/text`, `x/sys`, `x/net` | no |
| `wasm_exec.js` | from the Go 1.27.3 GOROOT | none (the tool copies it into the site) | 20 KB | web | no |
| Docker image `gguix-linux` (harness) | existing | `linux/dx.sh build` (already built) | 3.8 GB | Linux runtime check only | n/a |
| `libxkbcommon-tools`, `gnome-themes-extra`, `xdg-desktop-portal(-gtk/-gnome)` | in the rebuilt `gguix-linux` image | provided by the image | n/a | Linux runtime check | no |

No global tool was installed on the Mac. No Homebrew package, no Node, no Xcode component was needed. The only third-party modules in `go.mod` are NSGo's own dependencies.

Observed: nothing in the macOS build needed cgo (`CGO_ENABLED=0 go build` and `nsgo build -target macos_arm64` both work). `make cgoprobe` builds the desktop target with cgo on. The macOS binary links `libSystem`, `libresolv` and `CoreFoundation` only (`otool -L`); AppKit and friends are opened at run time. The Linux binary built with `CGO_ENABLED=0` is still "dynamically linked" (`ld-linux`, `libc`, `libdl`, `libpthread`, no GTK in `ldd`); GTK 4 is loaded at run time with purego (see `linux.md`).

## Clean machine steps

Mac with only Go, Homebrew and the Xcode Command Line Tools:

```sh
git clone <repo> && cd <repo>/apps/nsgo
make deps                 # go mod download
make run                  # desktop app (bin/gguix-nsgo)
make package              # optional: bin/GGUIX-nsgo.app
make web                  # web app: http://localhost:8090
make web-remote           # web app with the remote core
```

The first `go tool nsgo ...` call compiles the CLI (a few seconds).
