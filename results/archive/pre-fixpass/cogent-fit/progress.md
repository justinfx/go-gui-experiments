# Progress

Everything in the brief is implemented or recorded as a gap. Remaining caveats: no window was ever shown on a real display (the Mac was locked), so `make run`, `make bench` and `make package` output were not seen on screen; the status-bar relayout fix was not re-checked in the browser; Windows cross build is a recorded FAIL.


Second pass (data layer): `core.Pager` replaced by `rowModel` (`adapter_model.go`, `adapter_ui.go`). Tests 13/13, `make bench` completes, wasm builds with and without `remote` succeed. See `adapter.md`. The Mac was still without a visible window session, `make web` could not bind :8090 because another process held it, so the web serve step was not re-run (builds only).
