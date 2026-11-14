# Toolchain: MyGo native UI (mygoui)

| Tool | Version | Install | Size | Needed for | cgo related |
|---|---|---|---|---|---|
| Go | go1.27.3 darwin/arm64 | already installed | n/a | everything | no |
| Xcode Command Line Tools | already installed | `xcode-select --install` | n/a | linking and signing on macOS (the `codesign` call in `mygo build`) | no |
| MyGo module and CLI | v0.9.0 | `go.mod` `tool github.com/egoist/mygo/cmd/mygo` then `go mod download` | about 40 MB of module cache with its dependencies | dev, build, web, cross | no (MyGo builds with `CGO_ENABLED=0`, calls the system through purego) |
| Browser (Chrome) | already installed | n/a | n/a | viewing `make web` | no |

Nothing else: no Node, npm, Bun, Docker or system libraries were used. The web target is built by the same CLI (`mygo build -platform web`) and served by `core/cmd/server`.

## Clean machine steps

Desktop:

```sh
# Mac with Go, Homebrew and the Xcode Command Line Tools
cd apps/mygoui
make deps
make run
```

Web (in-process core), then with the remote core:

```sh
make web            # http://localhost:8090
make web-remote     # same, with the core API (-tags remote)
```
