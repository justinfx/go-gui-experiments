# mygoui fix pass

## 2026-11-14: web port (evaluator fix)

The Makefile hardcoded `-addr :8090` for `web`, `web-remote` and `bench-web`, so the harness's `SERVE_PORT` was ignored and the web step failed with "port 8097 never opened". It now uses `SERVE_PORT := 8090`, overridable, as the Fyne Makefile does. No app code changed. Only the web step was re-run (`--only web`); the desktop steps of the same full run stand.

## 2026-11-14: Linux build (app bug, found by the Linux check)

`make build` globbed `$(OUT)/linux-*/GGUIX-mygoui`, but MyGo writes `gguix-mygoui` on Linux, so the build exited 1 after `mygo build` succeeded. Fixed to the lowercase name (`results/mygoui/linux/makefile-linux-exe-name.patch`). macOS path unchanged; no app code changed; the macOS measurements stand.
