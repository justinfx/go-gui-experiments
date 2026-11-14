# GGUIX - MyGo

GGUIX reference application ported from `apps/wails` to MyGo v0.9.0 (github.com/egoist/mygo) in its web frontend mode: Go services bound to a generated TypeScript client, Svelte 5 + shadcn-svelte in the system webview. No browser or server target exists.

## Prerequisites (macOS, Apple Silicon)

- Go 1.27.3, Node 26 with npm 11 (no Bun needed), Google Chrome (only for `make test`), Xcode Command Line Tools are not needed (MyGo is built with CGO_ENABLED=0).

## Commands

```sh
cd apps/mygo
make deps    # npm ci in frontend/ (installs the mygo CLI from the mygo-cli npm package)
make build   # mygo build: vite build, mygo generate, go build, copied to bin/gguix-mygo
make run     # build and run
make dev     # mygo dev (Vite dev server + live reload)
make test    # go test, vitest, Playwright against the test-only harness (main_e2e.go)
make package # bin/GGUIX-mygo.app
```

`make web`, `web-remote` and `bench-web` exit 3 (UNSUPPORTED).
