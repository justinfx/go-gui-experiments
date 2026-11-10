# {{NAME}} app, fix pass: close gaps that rested on wrong facts

`apps/{{ID}}/` is a finished app built from `briefs/{{ID}}.md`. A fact-check of the exact module sources found that some gaps recorded in `results/{{ID}}/capabilities.md` assumed an API or component did not exist when it does. Close only those gaps, listed below. Everything in the original brief still applies (core unchanged, target-code marking, tests, Make targets, behaviour).

## Gaps to close
{{GAPS}}

## Rules
- Change only what these gaps need. Do not touch the data path, paging, or bench code.
- Use the framework's own API or component where one exists; hand-written cgo against a native handle is fine where that is the only route, marked as desktop target code.
- Do NOT run `make bench`, `bench-web`, cross builds, or anything long-running: benchmarks are being measured on this machine. `make build`, `make test` and brief manual launches are fine.
- All tests must pass. Add tests where practical.

## Results
- Write `results/{{ID}}/fixpass.md`: for each gap, what you changed, which API/component, files, and whether verified (how). Also note anything that still could not be done and why, with evidence (source path or doc).
- Update the matching rows in `results/{{ID}}/capabilities.md` and append a dated entry to `results/{{ID}}/friction.md`.
Reply with a short summary.

## Screenshots (mandatory)

Capture only the app window, never the full screen or desktop. Use `bench/shot.sh <process-name> <out.png>` (macOS: `screencapture -l <windowid>`; Linux: `import -window <id>`). Plain `screencapture` or `screencapture -x` without `-l` is forbidden: it records the user's other windows. If no window is found, skip the screenshot and say so.
