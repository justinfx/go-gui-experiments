# GGUIX: one app, five Go GUI approaches

This repository holds everything used to produce the results in the article
"[One Go App, Served Five Ways](https://justinfx.com/2026/11/10/go-gui-five-ways/)". The same application was
built five ways, on one shared Go core, and every build was measured by the same
harness.

| id | Stack | Web route |
|---|---|---|
| `fyne` | Fyne v2.8.2 | Go WASM, or WASM UI plus a Go server (`web-remote`) |
| `cogent` | Cogent Core v0.5.27 | Go WASM, or `web-remote` |
| `wails` | Wails v3.1.0, Svelte 5, Tailwind v4, shadcn-svelte | Wails server mode |
| `gotth` | templ, htmx, Tailwind, shadcn-templ; webview_go desktop shell | Go web server |
| `miqt` | MIQT v0.15.0 on Qt 6 | none (desktop only) |

Variants, used for follow-up questions:

- `<id>-fit` and `<id>-fitcap`: each app with a data layer suited to its toolkit
  (`-fitcap` keeps the shared core's limit of 2 requests in flight).
- `gotth-glaze`: the GoTTH app in a desktop shell built with `CGO_ENABLED=0`.
- `miqt-gotth`: the GoTTH web UI inside Qt WebEngine via MIQT.

## Layout

| Path | What it is |
|---|---|
| `SPEC.md` | The application spec every app implements (v1.2) |
| `EVALUATION_BRIEF.md` | Method summary and known caveats |
| `briefs/` | Per-framework briefs given to the implementing agents, generated from the spec by `tools/mkbrief` |
| `core/` | Shared Go module (no dependencies): 1M-row data service, pager, prefs, host info, HTTP client/server, bench driver, cgo probe |
| `apps/<id>/` | Each implementation |
| `bench/` | Measurement harness (`measure.sh`, `measure.py`, `web.mjs`, `shot.sh`) |
| `linux/` | Docker harness for the Linux function checks (Ubuntu 24.04, X11 and Wayland) |
| `tools/` | `mkbrief` and `loc` |
| `results/` | Raw metrics, notes and reports, see below |
| `addenda/` | Options suggested after the article, tested the same way (also on the [wiki](https://github.com/justinfx/go-gui-experiments/wiki/Addenda)) |

## How the apps were built

Each app was written by an AI coding agent (Claude Code, Sonnet) working alone from
its own brief. The agents were not given the judging criteria. Bugs found later,
mostly on Linux, were fixed afterwards; each fix is recorded in
`results/<id>/fixpass.md`.

## Running it

Every app exposes the same make targets:

```
deps dev build package run web web-remote test bench bench-web
cgoprobe cross-windows cross-linux clean
```

Unsupported targets exit with status 3. To build and run one app:

```bash
cd apps/fyne && make deps build run
```

To measure one app with the harness (macOS, screen awake and unlocked):

```bash
bench/measure.sh fyne
```

Results go to `results/<id>/metrics.json`. Toolchain needs per app are in
`results/<id>/toolchain.md`.

## Results

- `results/REPORT.md`: the full results report.
- `results/<id>/metrics.json`: raw measurements per app.
- `results/<id>/{toolchain,friction,capabilities}.md`: the implementing agent's notes.
  Corrections to them are in `results/ERRATA.md`.
- `results/LINUX.md` and `results/<id>/linux.md`: Linux function checks.
- `results/judge2/`: rubric scores; style judged blind.
- `results/DEVIATIONS.md` and `results/HARNESS_CHANGES.md`: every method change and
  evaluator error, kept on purpose.
- `results/archive/`: superseded data, kept for the record. Do not cite it.

Screenshots are not included in the repository.

## License

- Code (`core/`, `apps/`, `bench/`, `linux/`, `tools/`): MIT, see `LICENSE`.
- Docs, spec, briefs and results: CC BY 4.0, see `LICENSE-docs.md`.

### Third-party code

Some apps include vendored third-party code, which keeps its own license:

| Component | Where | License |
|---|---|---|
| [shadcn-templ](https://github.com/axadrn/shadcn-templ) v2.0.0 components and bundled JS | `apps/gotth*/components/`, `apps/gotth*/assets/js/`, `apps/miqt-gotth/` (same) | MIT, Copyright (c) 2023 Axel Adrian |
| [htmx](https://htmx.org) 2.0.11 and the SSE extension | `apps/gotth*/static/htmx/`, `apps/miqt-gotth/static/htmx/` | 0BSD |
| [Lucide](https://lucide.dev) icon data (via shadcn-templ) | `apps/gotth*/components/icon/` | ISC |
| [shadcn-svelte](https://shadcn-svelte.com) components | `apps/wails*/frontend/src/lib/components/ui/` | MIT |

The GoTTH apps patch the vendored shadcn-templ `components/baseui/lifecycle.js`
for WebKit performance (see `results/DEVIATIONS.md`).
