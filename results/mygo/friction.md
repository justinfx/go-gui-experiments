# Friction log: MyGo v0.9.0 (port of apps/wails)

Dated 2026-11-14. Each entry says whether it is a framework gap or my own mistake.

### 11:30 Module path and Go version
- Step: `go get github.com/egoist/mygo@v0.9.0`. Raised go.mod from 1.27 to 1.27.3 and bumped golang.org/x/text. Category: framework requirement (mygo needs go 1.27.3). Time lost: 0.

### 11:33 No Bun needed
- Step: Brief expected Bun. npm install of `mygo-cli` gave a working `mygo` binary; `mygo doctor` marks Bun optional. Outcome: no Bun. Category: none.

### 11:40 mygo.config.ts cannot import mygo-cli from a frontend/ subfolder
- Step: `mygo generate` with `import { defineConfig } from "mygo-cli"`.
- Outcome: ERR_MODULE_NOT_FOUND, because the config sits at the Go module root and mygo-cli is in `frontend/node_modules`. MyGo's template puts package.json at the module root; `frontend/` is half supported (bindings default `frontend/src/mygo.ts`, config.go:60).
- Resolution: export a plain object, losing the types. Category: framework gap (layout assumption), minor. Time lost: 4.

### 11:45 Frontend embedding is done by mygo build, not go build
- Step: First tried `//go:embed all:frontend/dist` plus `mygo.SetFrontend`, which makes `go test` fail without a built dist and needs dist before `mygo generate`.
- Resolution: dropped it; `mygo build` embeds through a go build overlay (cmd/mygo/embed.go). Consequence: plain `go build` makes a binary with no frontend, so the harness binary must come from `mygo build`. Category: my own mistake first, then a framework design point. Time lost: 6.

### 11:50 mygo build outputs a bundle, not a bare binary
- Step: Harness needs `bin/gguix-mygo`. `mygo build` writes `build/out/darwin-arm64/gguix-mygo.app` (also runs vite build on every call, about 5 s total).
- Resolution: the Makefile copies `Contents/MacOS/gguix-mygo` to bin/; it runs outside the bundle. Linux layout not checked. Category: framework gap (no plain-binary output), minor.

### 11:55 Playwright cannot reach Go: no browser mode
- Step: The Wails e2e suite runs against server mode. MyGo has none.
- Resolution: wrote a test-only HTTP harness (`main_e2e.go`, about 120 lines) and `e2e/shim.js` faking `window.mygo`. T12 changed to expect a desktop host (native Source). This is the largest deviation from the Wails app and tests less (no MyGo IPC). Category: framework gap. Time lost: 25.

### 12:10 Generated client naming
- Step: Methods become lower camel case and Go's `in` parameter becomes `in_`. The app's `Api` type uses Go names, so `api.ts` maps with a Proxy (`lowerFirst`). Category: framework convention, no loss.

### 12:12 Events: typed events only; no bare names
- Step: Wails registered event names with `RegisterEvent`; MyGo needs one `NewEvent[T]` per name. `emitEvent(name, data)` became a switch. Payload comes unwrapped. Void events use `struct{}` (client type `{}`). Category: framework design, no gap.

### 12:15 Notifications need a bundle
- Step: `mygo.NewNotification.Show` needs an app bundle on macOS (`needsApp`, `IsPackaged`). Guarded with `App.IsPackaged()`. Category: platform limit, same as Wails.

### 12:20 Stray binary in the app folder
- Step: Running `go build` without `-o` in the module root wrote a 14 MB binary named wails (copied from apps/wails). Removed. Category: my own mistake.

### 12:25 make web exits 2, not 3, through make
- The recipe exits 3 and prints UNSUPPORTED, but `make` reports 2 for its own failure. Same behavior as the Wails Makefile pattern. Category: not a framework issue.
