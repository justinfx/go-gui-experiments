# MIQT progress (final, 17:31)

All brief items done. Open: cross-windows FAIL (go1.26 image vs go 1.27 module), cross-linux FAIL by timebox (emulated amd64 build produced nothing in 28 min). Both recorded in capabilities.md and friction.md. Code builds; make test gives GGUIX_TESTS passed=18 failed=0 not_feasible=0.

Second pass (data layer): pager replaced by rowStore + shared model (results/miqt/adapter.md). make test: GGUIX_TESTS passed=18 failed=0 not_feasible=0. make bench completes.
