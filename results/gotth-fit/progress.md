# GoTTH progress

All brief items are implemented. See capabilities.md for gaps (split divider, column resize, native menu bar) and the intermittent desktop bench stall.
make targets verified here: deps, build, package, run (launches), test (12 passed), bench, cgoprobe, cross-linux, cross-windows, web (starts on [::1]:8090 because 127.0.0.1:8090 is used by a foreign process).

Second pass (data layer for the toolkit, see adapter.md): the list is now a virtual scroller over server-rendered page blocks with a server-side page cache (`adapter_data.go`). make test: 12 scenarios passed, plus the adapter unit tests. make bench completes with all eight events (placeholder steps 0 to 3 instead of 185).
