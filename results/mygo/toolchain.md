# Toolchain: MyGo v0.9.0

Host: macOS 26.6.2, Apple Silicon, Go 1.27.3.

| Name | Version | Install | Needed for | cgo related |
|---|---|---|---|---|
| MyGo module | github.com/egoist/mygo v0.9.0 | `go get` (via go.mod; go.mod's go line moves to 1.27.3) | runtime and library | no: purego, `CGO_ENABLED=0` by default; the binary links only libSystem, libresolv, CoreFoundation, Security |
| mygo CLI | mygo-cli 0.9.0 (npm) | `npm install -D mygo-cli@0.9.0` in frontend/ (native binary from the `@egoist/mygo-cli-darwin-arm64` optional package) | generate, dev, build, doctor | no |
| mygo-runtime | 0.9.0 (npm) | `npm install mygo-runtime@0.9.0` | frontend runtime | no |
| Bun | not installed, not needed | | | |
| Node / npm | 26.10 / 11.19 | existing | the CLI evaluates mygo.config.ts with Node 22.6+ when Bun is absent | no |
| Xcode Command Line Tools / clang | not needed to build | | only for the cgoprobe check | |
| Frontend packages | same as apps/wails, minus @wailsio/runtime and the Wails vite plugin | `npm ci` | build, test | no |

Bun was not required. `npx mygo-cli` and the local `node_modules/.bin/mygo` both work; `mygo doctor` lists Bun as optional.

## Clean machine steps

```sh
brew install node
cd apps/mygo && make deps   # npm ci (includes the mygo CLI)
make build                  # bin/gguix-mygo (about 10.9 MB), .app in build/out/darwin-arm64
```

The CLI path matters: `mygo.config.ts` is evaluated from the project root, so `import { defineConfig } from "mygo-cli"` fails when mygo-cli is installed in `frontend/node_modules`. The config here is a plain object.
