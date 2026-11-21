# GGUIX on Gophics

The GGUIX reference app built with [Gophics](https://github.com/doug/gophics) (pinned at commit `8d36097`, `v0.0.0-20261103180631-8d360971eb89`). Pure Go, `CGO_ENABLED=0`, GPU-drawn widgets on Metal (macOS), Vulkan (Linux), DX12 (Windows) and WebGPU (browser).

## Prerequisites

- Go 1.27.3
- For `web` / `web-remote`: a browser with WebGPU (Chrome)
- Nothing else. No cgo, no SDK, no system libraries at build time.

## Bootstrap from a clean machine

```sh
cd apps/gophics
make deps        # installs the gophics CLI at the pinned version into ./.tools, downloads modules
make run         # builds bin/gguix-gophics and opens the window
```

How the project was created (for reference, already done):

```sh
GOBIN=$PWD/.tools go install github.com/doug/gophics/cmd/gophics@8d36097
.tools/gophics create -p desktop,web -module gguix/apps/gophics gophics
go mod edit -require=github.com/doug/gophics@v0.0.0-20261103180631-8d360971eb89
# plus: require gguix/core v0.0.0, replace gguix/core => ../../core
```

## Targets

| Target | What it does |
|---|---|
| `make deps` | CLI + modules (idempotent) |
| `make dev` | `gophics dev -p desktop .` (rebuild + relaunch on save, state kept) |
| `make build` | `bin/gguix-gophics` via `gophics build -p desktop` |
| `make package` | `bin/GGUIX-gophics.app` via `gophics package` (unsigned, no .dmg) |
| `make run` | build and run |
| `make web` | WASM build into `bin/web/`, served by `core/cmd/server` on http://localhost:8090 |
| `make web-remote` | same with `-tags remote` (data, prefs, host info from the server) |
| `make test` | headless tests T1..T12 (`app.NewHeadless`), ends with `GGUIX_TESTS ...` |
| `make bench` | desktop bench (`GGUIX_BENCH=1`), prints `GGUIX_METRIC` lines, quits |
| `make bench-web` | serves the web build; open http://localhost:8090/?bench=1 and read the console |
| `make cgoprobe` | rebuilds each target with `-tags cgoprobe` and prints `CGOPROBE` lines |
| `make cross-windows` / `make cross-linux` | `GOOS=... CGO_ENABLED=0 go build` into `bin/cross/` |
| `make clean` | removes `bin/` |

## Layout

- `main.go`, `marker.go`: shared entry point; `main_bench.go`: bench hook.
- `ui/`: the app. `*_desktop.go` / `*_web.go` are target-specific, `driver_bench.go` is the bench driver, `app_test.go` the tests.
- `bench/cgoprobe.sh`, `run_tests.sh`: scripts behind `make cgoprobe` and `make test`.
