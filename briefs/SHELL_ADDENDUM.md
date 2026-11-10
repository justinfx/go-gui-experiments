# {{NAME}}: a different desktop shell for the GoTTH app

`apps/{{ID}}/` starts as an exact copy of `apps/gotth/` (templ + htmx + Tailwind + shadcn-templ, webview_go desktop shell). Read `briefs/gotth.md` (the original brief, all of which still applies) and `results/gotth/capabilities.md`, `fixpass.md`.

## The change
Replace only the desktop shell (`cmd/desktop/` and its Make targets) with: {{SHELL}}

Keep the Go server, templ views, htmx, JS, data path, paging and bench code identical to `apps/gotth`, so the comparison isolates the shell. The browser (web) target stays exactly as GoTTH has it. Remove the `-tags native` webview_go shim and the osascript/pbcopy helpers from this copy if the new shell replaces them.

## What the shell must provide (desktop)
Window title, 1280 x 800, min 1024 x 640, restore saved size; a native menu bar with the brief's menus, items and shortcuts, wired to the same `GGUIX.actions[...]` the page uses (skip the in-page menubar when the native one is present, as the `native` tag variant did); native open (.txt) and save (CSV) dialogs; the export-complete OS notification; clipboard for Copy Row; Reveal Last Export; the Host panel with desktop host info. Mark target-specific code as the brief requires.

## Rules
- `core/` unchanged. All tests must pass; keep the Make targets (build, run, test, bench, bench-web, package, cross-linux, cross-windows, cgoprobe) working, with exit 3 for unsupported targets.
- The desktop binary must be named `bin/gguix-{{BIN}}` (the harness looks for it).
- Do NOT run `make bench`, `bench-web`, cross builds or other long jobs; benchmarks are being measured on this machine. `make build`, `make test` and brief launches are fine.
- Verify every API you rely on against the module source in ~/go/pkg/mod before claiming it exists or doesn't. Record evidence paths.
- Never use em-dashes in what you write.

## Results
In `results/{{ID}}/`: `capabilities.md` (same format as gotth's), `friction.md` (dated entries), `toolchain.md`, `shell.md` (design, files, what changed vs gotth, what is native vs fallback, limits with evidence). Reply with a short summary.

## Screenshots (mandatory)

Capture only the app window, never the full screen or desktop. Use `bench/shot.sh <process-name> <out.png>` (macOS: `screencapture -l <windowid>`; Linux: `import -window <id>`). Plain `screencapture` or `screencapture -x` without `-l` is forbidden: it records the user's other windows. If no window is found, skip the screenshot and say so.
