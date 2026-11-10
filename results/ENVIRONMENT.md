# Environment

- Machine: Apple M1 Max, 64 GB RAM, macOS 26.6.2 (arm64)
- Go 1.27.3, Node 26.10.0, Docker Desktop (for cross builds), Google Chrome (Playwright 1.65.0, `channel: chrome`, headed)
- Frameworks:
  - fyne: fyne.io/fyne/v2 v2.8.2
  - cogent: cogentcore.org/core v0.5.27
  - wails: github.com/wailsapp/wails/v3 v3.1.0 (Svelte 5, Tailwind v4, shadcn-svelte)
  - miqt: github.com/mappu/miqt v0.15.0 with Homebrew Qt 6.12.0
  - gotth: github.com/a-h/templ v0.3.1100, webview_go 20240831, htmx, Tailwind, shadcn-templ
- Measurement: `bench/measure.sh <id>`, run sequentially on 2026-11-08 with the screen unlocked. Windows were sometimes on another display or occluded; per-app run-to-run spread was checked for throttling and was tight.
