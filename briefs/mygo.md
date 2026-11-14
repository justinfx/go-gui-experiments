# MyGo: port the Wails app to MyGo

This brief tests a reader's suggestion: start with Wails, then have an agent port
the app to [MyGo](https://mygo.run) (`github.com/egoist/mygo`, v0.9.0, MIT). Use
MyGo's **web frontend mode** (a page in the OS webview, Go services bound to a
generated TypeScript client). Do not use its native `ui` package.

`apps/mygo/` starts as an exact copy of `apps/wails/` (Wails v3, Svelte 5,
Tailwind v4, shadcn-svelte). Read `briefs/wails.md` (the original brief; every
functional requirement in it still applies) and `results/wails/capabilities.md`,
`fixpass.md`, `friction.md`.

## Setup

```bash
cp -R apps/wails apps/mygo
rm -rf apps/mygo/bin apps/mygo/frontend/node_modules apps/mygo/frontend/dist
```

## The change

Replace Wails with MyGo in `apps/mygo/`: the Go entry points, service binding,
events, menus, dialogs, notifications, clipboard, window options, and the
frontend's calls into Go (swap the Wails runtime and generated bindings for
`mygo-runtime` and the client from `mygo generate`). Keep the Svelte components,
styling, shared `core/` usage, paging and bench code as close to `apps/wails` as
the port allows, so the comparison isolates the framework. Record every place
where it could not stay the same, and why.

Relevant MyGo API (verify each against the module source in `~/go/pkg/mod` before
relying on it, and record evidence paths):

- Binding: `mygo.Bind`, `mygo.BindAs`; `mygo generate` writes the TS client.
- Events: `mygo.NewEvent`, `Broadcast`, `Emit`; frontend `mygo-runtime` (`call`, `on`, `event`).
- Menus: `mygo.NewMenu`, `mygo.MenuItem{Label, Accelerator, Click, Role, Submenu}`, `App.SetMenu`.
- Dialogs: `mygo.Dialog.Open` / `Save` with `FileFilter`.
- Notifications: `mygo.NewNotification(...).Show()` (macOS: packaged apps only; note what that means for the harness).
- Clipboard: `mygo.Clipboard.WriteText`. Reveal: `mygo.Shell.ShowItemInFolder`.
- Window: `WindowOptions` with `MinWidth`, `MinHeight`, `StateKey` (size restore).
- Config: `mygo.config.ts` (`defineConfig`), CLI `mygo build`, `mygo dev`, `mygo doctor`.

MyGo's docs use Bun. Bun is not installed on this machine. Try npm/npx first
(`npx mygo-cli`, npm scripts). If MyGo truly requires Bun, install it with
Homebrew, and record that in `toolchain.md` and `friction.md`.

## Web target

MyGo documents no browser or server deployment. Do not build one. Make `web`,
`web-remote` and `bench-web` exit 3, and record what you found about web support
(including any `*_js.go` files in the module) with evidence in `capabilities.md`.

## Rules

- `core/` unchanged. All tests must pass. Keep the Make targets (deps, dev, build,
  package, run, test, bench, cgoprobe, cross-linux, cross-windows, clean) working,
  with exit 3 for unsupported targets.
- The desktop binary must be `bin/gguix-mygo` (the harness launches it directly, with
  `GGUIX_BENCH=1` and `GGUIX_BENCH_EXPORT`), and it must print the same
  `GGUIX_METRIC` lines as `apps/wails`.
- `make build` must rebuild after a change to `marker.go` or
  `frontend/src/lib/marker.ts` (the harness times both). Keep both files.
- `cgoprobe`: same check as the Wails app (can a cgo core run in the desktop build).
  MyGo defaults to `CGO_ENABLED=0`; note what turning cgo on requires.
- Do NOT run `make bench`, cross builds or other long jobs; benchmarks are measured
  separately. `make build`, `make test` and brief launches are fine.
- Never use em-dashes in what you write.

## Results

In `results/mygo/`: `capabilities.md` (same format as Wails'), `friction.md` (dated
entries; mark each as a framework gap or your own mistake), `toolchain.md`, and
`port.md` (what changed vs Wails, files touched, lines changed, what is native vs
fallback, limits with evidence). Reply with a short summary.

## Screenshots (mandatory)

Capture only the app window, never the full screen or desktop. Use
`bench/shot.sh <process-name> <out.png>`. Plain `screencapture` or
`screencapture -x` without `-l` is forbidden: it records the user's other windows.
If no window is found, skip the screenshot and say so. Save into
`results/mygo/screenshots/`.
