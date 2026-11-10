# Wails fix pass (2026-11-09)

## Gap 1: Reveal in Finder on desktop
- Changed: `Reveal` in `apps/wails/platform_desktop.go` now calls `p.app.Env.OpenFileManager(path, selectFile)` (Wails v3.1.0 `EnvironmentManager`, source `pkg/application/environment_manager.go`) instead of `host.RevealInFileManager`. `selectFile` is true when the path is a file (reveal after export) and false for a directory (Host panel config folder). The `host` import stays for `host.Current`. Web is unchanged (`errUnsupported`).
- Verified: `go build` and `make test` pass (34 checks). The real Finder reveal was not exercised (no GUI launch in this pass).

## Gap 2: Split view
- Changed: the custom pointer-drag divider in `apps/wails/frontend/src/App.svelte` is replaced by shadcn-svelte `resizable` (`ResizablePaneGroup`, `ResizablePane`, `ResizableHandle`), added with `npx shadcn-svelte@1.8.0 add resizable -y` (files in `frontend/src/lib/components/ui/resizable/`, new dependency `paneforge`). Default 65/35, minimums 30 and 15 percent.
- Verified: new e2e test T13 in `frontend/e2e/gguix.spec.ts` drags the handle and checks the left pane shrinks; full `make test` passes (14 Playwright, 34 total).
- Column resize stays custom: Resizable lays out sibling panes by percentage, while the table needs per-column pixel widths feeding a `<colgroup>` in a virtualized table. Code is in `DataTable.svelte`.

## Not done
Nothing outstanding for these gaps beyond the unverified visual Finder reveal.

## Manual check by the user (2026-11-09)

- Reveal after export: Finder opens with the exported file selected. PASS.
