# GGUIX with MIQT

Qt 6 widgets from Go through MIQT v0.15.0 (dynamic link against Homebrew Qt 6.12.0). Desktop only.

## Prerequisites
A Mac with Xcode Command Line Tools, Homebrew and Go 1.27.3.

## From a clean machine
```
xcode-select --install
brew install go qt pkgconf     # `make deps` does qt and pkgconf for you
cd apps/miqt
make deps                      # brew qt + pkgconf, miqt-docker v0.15.0, go mod download
make build                     # first build compiles the Qt bindings: 2 to 10 minutes
make run
```
Bootstrap used: `go get github.com/mappu/miqt@v0.15.0`, import `github.com/mappu/miqt/qt6`. MIQT has no project template.

## Targets
- `make dev`: incremental build then run (no hot reload exists).
- `make build`: `bin/gguix-miqt`. `make package`: `bin/GGUIX-miqt.app`, made self-contained with macdeployqt.
- `make test`: offscreen Qt tests, ends with the `GGUIX_TESTS passed=... failed=... not_feasible=...` line.
- `make bench`: bench mode, prints the eight GGUIX_METRIC lines.
- `make cgoprobe`: builds and tests with `-tags cgoprobe`.
- `make cross-linux`, `make cross-windows`: miqt-docker (needs Docker running; slow under emulation).
- `make web`, `make web-remote`, `make bench-web`: print UNSUPPORTED. Note: GNU make reports a failing recipe as exit status 2, so the required exit status 3 is only visible as the recipe's own status (the message is printed).
- `make clean`.
