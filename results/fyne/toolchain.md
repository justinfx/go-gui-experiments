# Toolchain: Fyne

| Tool | Version | Install command | Approx size | Needed for | cgo related |
|---|---|---|---|---|---|
| Go | go1.27.3 darwin/arm64 | already installed | n/a | all | no |
| Xcode Command Line Tools (clang, macOS frameworks) | already installed | `xcode-select --install` | about 1.5 GB | dev, build, runtime (Fyne desktop driver is cgo: GLFW, Cocoa, OpenGL) | yes |
| fyne CLI | v1.7.4 | `go install fyne.io/tools/cmd/fyne@v1.7.4` | 20 MB binary, 13 s | package (.app), web (WASM packaging) | no |
| Fyne library | v2.8.2 | `go get fyne.io/fyne/v2@v2.8.2` then `go mod tidy` | module cache about 60 MB | all | desktop build uses cgo |
| python3 (`http.server`) | 3.14 here (any 3) | ships with the Command Line Tools / Homebrew | n/a | web serving (`make web`) | no |
| Google Chrome | already installed | n/a | n/a | web runtime and bench-web (headless used for the web bench) | no |
| Docker Desktop | 29.7.2 | already installed | n/a | cross builds | no |
| fyne-cross | v1.6.3 | `go install github.com/fyne-io/fyne-cross@latest` | 5.8 MB binary | cross-windows, cross-linux | no |
| fyne-cross docker images | `fyneio/fyne-cross-images:linux` and `:windows` | pulled automatically by fyne-cross | 4.17 GB and 2.97 GB | cross | no |

Global installs made: the fyne CLI and fyne-cross into `$(go env GOPATH)/bin`, and the two Docker images. fyne-cross is a third party tool that the brief's section 7 route requires; it is not an app dependency. The app module has no third-party dependencies beyond what Fyne itself pulls in.

## Clean machine steps

From a Mac with only Go, Homebrew and the Xcode Command Line Tools:

```sh
cd apps/fyne
export PATH="$PATH:$(go env GOPATH)/bin"
make deps          # go install fyne.io/tools/cmd/fyne@v1.7.4, go mod download
make run           # desktop app (first build compiles cgo code, about 1 to 2 minutes)
```

Web app:

```sh
make web           # fyne package --target wasm, serves bin/web on http://localhost:8090 (python3 http.server)
# or, with the core server:
make web-remote    # builds with --tags remote, serves with core/cmd/server on :8090
```

Other: `make package` (bin/GGUIX-fyne.app), `make test`, `make bench`.
Cross builds need Docker Desktop running: `make cross-linux`, `make cross-windows` (they run `go mod vendor` temporarily, see capabilities.md).
