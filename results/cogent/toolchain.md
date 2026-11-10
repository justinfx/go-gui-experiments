# Toolchain: Cogent Core v0.5.27

Beyond `go` (1.27.3 darwin/arm64), Homebrew and the Xcode Command Line Tools, which were already present.

| Tool | Version | Install command | Approx. size | Needed for | cgo related |
|---|---|---|---|---|---|
| `core` CLI | v0.5.27 | `go install cogentcore.org/core@v0.5.27` | 34 MB binary at `~/go/bin/core` | dev, build, package, web, cross | no (it is pure Go; the apps it builds use cgo) |
| `cogentcore.org/core` module | v0.5.27 | `go get cogentcore.org/core@v0.5.27` (via `make deps`: `go mod download`) | about 52 MB source in the module cache | build, runtime | no |
| `github.com/oliverbestmann/webgpu` and its `libs-darwin` etc. | v1.37.0 / pseudo-versions | pulled in by the framework (WebGPU renderer) | the module family is large (about 550 MB in this shared module cache, all platforms' native libraries are downloaded by `go mod download`) | build, runtime (desktop renders with WebGPU, native static libraries linked in) | yes (cgo, links the wgpu native library) |
| glfw (`github.com/go-gl/glfw`) | pulled in by the framework | module download | small | build, runtime | yes (cgo, Cocoa) |
| Xcode Command Line Tools (clang) | already installed | `xcode-select --install` | n/a | build (cgo) | yes |
| Python 3 (macOS built-in) | system | none | n/a | not needed by the app (Makefile only uses sh, go and core) | no |
| `pip install dmgbuild` | not installed | would be needed by `core pack` for the `.dmg` | n/a | package (dmg only) | no |

The wasm target needs nothing else: `core build wasm` only runs `go build` for `js/wasm`. The `js` platform (GoScript plus bun) was not used.

Nothing else was installed globally. `core setup` was not needed on this machine.

## Clean machine steps

From a Mac with only Go, Homebrew and the Xcode Command Line Tools:

Desktop app:

```sh
export PATH="$PATH:$HOME/go/bin"
cd apps/cogent
make deps          # installs the core CLI v0.5.27, downloads modules
make run           # core build, then runs bin/gguix-cogent
```

Web app:

```sh
cd apps/cogent
make web           # core build wasm -o bin/web, serves http://localhost:8090
make web-remote    # same with -tags remote, served by core/cmd/server with the real API
```

The first desktop build takes about 30 s and the wasm build about 20 s on this machine (Apple Silicon, 10 cores).
