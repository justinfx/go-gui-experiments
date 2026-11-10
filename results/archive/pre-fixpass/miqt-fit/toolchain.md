# Toolchain (MIQT)

| Tool | Version | Install | Size | Needed for | cgo related |
|---|---|---|---|---|---|
| Qt 6 (Homebrew `qt`) | 6.12.0 | `brew install qt` | about 1 GB with deps | build, runtime (dynamic), package | yes |
| pkgconf | 2.5.1 | `brew install pkgconf` | small | build | yes |
| Xcode CLT (clang) | preinstalled | `xcode-select --install` | n/a | build | yes |
| miqt module | v0.15.0 | `go get github.com/mappu/miqt@v0.15.0` | about 100 MB source in module cache | build | yes |
| miqt-docker | v0.15.0 | `go install github.com/mappu/miqt/cmd/miqt-docker@v0.15.0` | 4.7 MB | cross | no |
| macdeployqt | ships with qt | `/opt/homebrew/opt/qt/bin/macdeployqt` | n/a | package | no |
| Docker Desktop | 29.7.2 | preinstalled | n/a | cross | no |
| Docker images | linux64-go1.27-qt6.12-dynamic, win64-cross-go1.26-qt6.5-static | built by miqt-docker | several GB each | cross | yes |

Binary sizes: `bin/gguix-miqt` 31 MB stripped. App bundle 30 MB before macdeployqt, 111 MB after.

## Clean machine steps
Desktop:
1. `xcode-select --install`
2. `brew install go qt pkgconf`
3. `cd apps/miqt && make deps && make run` (first compile of the bindings is slow)

Web: not supported.
