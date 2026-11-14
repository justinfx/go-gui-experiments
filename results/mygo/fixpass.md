
## 2026-11-14: bench launch (evaluator fix)

The harness launches `bin/gguix-mygo` directly. `make build` had copied the executable out of `build/out/darwin-arm64/gguix-mygo.app`. A web-frontend MyGo app on macOS opens no window and runs no page when its executable runs outside the `.app` bundle (same bytes, `cmp` identical): no `GGUIX_METRIC` lines, under the harness or `make bench`. The agent's own bench check ran the bundled executable. Fix: `make build` now makes `bin/gguix-mygo` a symlink to the bundled executable. No app code changed. The first measurement run was discarded (`results/archive/mygo-discarded-2026-11-14/`).
