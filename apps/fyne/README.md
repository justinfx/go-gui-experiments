# GGUIX - Fyne

The GGUIX reference application built with Fyne v2.8.2 (`fyne.io/fyne/v2`), CLI `fyne.io/tools v1.7.4`.
Module `gguix/apps/fyne`, depends on `gguix/core` through `replace gguix/core => ../../core`.

## Prerequisites (clean Mac)

1. Xcode Command Line Tools: `xcode-select --install`
2. Homebrew (https://brew.sh)
3. Go 1.27.3: `brew install go` (or the official installer). Make sure `$(go env GOPATH)/bin` is on `PATH`.
4. Optional: Python 3 (for the simple static server used by `make web`), Docker Desktop (cross builds), Google Chrome or any browser (web targets), Node (not needed).

Fyne's desktop driver uses cgo with the macOS system frameworks, so only the Command Line Tools are needed. No extra libraries.

## Bootstrap from a clean machine

```sh
cd apps/fyne
make deps      # go install fyne.io/tools/cmd/fyne@v1.7.4 and go mod download
make run       # build bin/gguix-fyne and start it
```

The project was bootstrapped with `go mod init gguix/apps/fyne`, `go get fyne.io/fyne/v2@v2.8.2` and `go mod tidy`.

## Make targets

| Target | What it does |
|---|---|
| `deps` | Install the pinned `fyne` CLI and download modules. Idempotent. |
| `dev` | `go run .` (Fyne has no hot reload). |
| `build` | `bin/gguix-fyne` desktop binary. |
| `package` | `bin/GGUIX-fyne.app` using `fyne package --target darwin`. |
| `run` | Build and run. |
| `web` | `fyne package --target wasm`, moved to `bin/web/`, served with `python3 -m http.server` on port 8090 (override with `SERVE_PORT=`). In-process core. |
| `web-remote` | Same with `--tags remote`, served by `cd ../../core && go run ./cmd/server -static ../apps/fyne/bin/web -addr :8090`. |
| `test` | All automated tests (Fyne test driver). Last line is `GGUIX_TESTS passed=<n> failed=<n> not_feasible=<n>`. |
| `bench` | Desktop bench mode (`GGUIX_BENCH=1`, temp prefs dir, export to a temp file). Prints eight `GGUIX_METRIC` events. |
| `bench-web` | Builds and serves the web target; open `http://localhost:8090/?bench=1` and read the browser console. |
| `cgoprobe` | Rebuilds each target with `-tags cgoprobe` and prints `CGOPROBE <target> PASS\|FAIL <reason>`. |
| `cross-windows`, `cross-linux` | `fyne-cross` (Docker) attempts, see `results/fyne/capabilities.md`. |
| `clean` | Remove `bin/` and generated files. |

## Notes

- Preferences go through the core `PrefsStore` (`$GGUIX_PREFS_DIR` overrides the directory).
- Environment overrides from core apply: `GGUIX_ROWS`, `GGUIX_LATENCY_MS`, `GGUIX_EXPORT_RPS`; `GGUIX_REMOTE_URL` with the `remote` tag on native builds.
- Source layout: `*_desktop.go` desktop only, `*_web.go` web only, `*_bench.go` bench driver, `*_test.go` tests, everything else shared.
