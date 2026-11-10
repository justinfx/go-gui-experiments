# Toolchain (MIQT + GoTTH)

Same as `results/miqt/toolchain.md` plus GoTTH's tools (templ v0.3.1100, shadcn-templ v2.0.0, Tailwind v4.3.5 standalone in `bin/tools`, htmx files, all from `make deps`).

| Tool | Version | Install | Needed for | cgo related |
|---|---|---|---|---|
| Qt 6 | 6.12.0 | `brew install qt` | build, runtime, package | yes |
| qtwebengine, qtwebchannel (Homebrew formulae, pulled in by `qt`) | 6.12.0 | with `qt` | `QtWebEngineWidgets`, `QtWebChannel` frameworks under `/opt/homebrew/opt/qtwebengine` and `qtwebchannel` | yes |
| pkgconf | 2.5.1 | `brew install pkgconf` | build (`pkg-config Qt6WebEngineWidgets` is the cgo flag of `qt6/webengine`) | yes |
| miqt | v0.15.0 | `go get github.com/mappu/miqt@v0.15.0` (go.mod) | build | yes |
| macdeployqt | ships with qt | `/opt/homebrew/opt/qt/bin/macdeployqt` | package | no |
| miqt-docker | not used | | cross is not feasible, see capabilities.md | |

Evidence: `pkg-config --modversion Qt6WebEngineWidgets Qt6WebChannel` prints 6.12.0 twice; `otool -L bin/gguix-miqt` lists QtWebEngineWidgets, QtWebEngineCore and QtWebChannel; `~/go/pkg/mod/github.com/mappu/miqt@v0.15.0/qt6/{webengine,webchannel}` exist.
Binary: `bin/gguix-miqt` 40 MB (stripped; MIQT alone was 31 MB). Bundle after macdeployqt: 335 MB.

## Clean machine steps
1. `xcode-select --install`
2. `brew install go qt pkgconf`
3. `cd apps/miqt-gotth && make deps && make run`

Web: use `make web` (unchanged from gotth).
