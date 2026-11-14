# Toolchain: go-gui

| Name | Version | Install command | Size | Needed for | cgo-related |
|---|---|---|---|---|---|
| github.com/go-gui-org/go-gui | v0.96.0 | `go get github.com/go-gui-org/go-gui@v0.96.0` | module source ~60 MB in module cache | build, runtime (linked) | yes on macOS (Metal backend is Objective-C) |
| github.com/go-gui-org/go-glyph | v1.26.5 (indirect, imported for `TypefaceBold`) | pulled by go-gui | small | build | no (pure Go text) |
| Xcode Command Line Tools | already installed | `xcode-select --install` | ~1.5 GB | build (macOS) | yes |
| go-gui `cmd/buildapp` | v0.96.0 | none, run with `go run github.com/go-gui-org/go-gui/cmd/buildapp@v0.96.0` | in module cache | package | no |
| `wasm_exec.js` | from Go 1.27.3 GOROOT | none | 20 KB | web | no |
| `gogui.js` | from go-gui module (`gui/backend/web`) | none | small | web | no |

No global tools were installed. Linux and Windows cross builds needed nothing extra.

## Clean machine steps

```
# Mac with Go, Homebrew and Xcode CLT
git clone <repo> && cd <repo>/apps/gogui
make deps
make build
./bin/gguix-gogui          # desktop app
make web                   # then open http://localhost:8090
```
