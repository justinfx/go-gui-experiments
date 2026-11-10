# Progress: Fyne GGUIX app

Status: complete against the brief's definition of done.

- All make targets exist; build, run, package, test, bench, web, web-remote, cgoprobe, cross-linux, cross-windows verified (web runs used port 8091 because 8090 was taken by another process).
- make test: GGUIX_TESTS passed=12 failed=0 not_feasible=0
- make bench prints all 8 events (desktop and web in headless Chrome, web-remote too).
- Results: toolchain.md, friction.md, capabilities.md in this folder.
- Not verified: a real-screen screenshot of the desktop app (the screen capture came back black, display locked); visuals were checked with Fyne test-driver captures.

## Second pass (data layer)

- `core.Pager` replaced by `rowStore` (`apps/fyne/adapter_store.go`); see `adapter.md`.
- make test: GGUIX_TESTS passed=12 failed=0 not_feasible=0; make bench prints all 8 events; web target serves.
