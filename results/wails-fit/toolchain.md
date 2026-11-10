# Toolchain: Wails v3.1.0

Host: macOS 26.6.2, Apple Silicon. Go 1.27.3 (given).

| Name | Exact version | Install command | Approx. size | Needed for | cgo related |
|---|---|---|---|---|---|
| wails3 CLI | v3.1.0 | `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0` | 57 MB binary (about 250 MB of module cache downloads) | dev, build, package, web, cross | no (the CLI itself) |
| Wails library | github.com/wailsapp/wails/v3 v3.1.0 | `go mod download` (via go.mod) | in the module cache | build, runtime | yes on macOS (Cocoa/WebKit through cgo), also the notifications service |
| Xcode Command Line Tools (clang, WebKit headers) | as installed | `xcode-select --install` | about 1 GB | build (desktop) | yes: cgo C toolchain |
| Node.js / npm | 26.10.0 / 11.19.1 | `brew install node` | about 100 MB | build, dev, test | no |
| svelte | 5.57.3 | `npm ci` in frontend/ | in node_modules (194 MB total) | build | no |
| tailwindcss, @tailwindcss/vite | 4.3.5 | npm | | build | no |
| vite, @sveltejs/vite-plugin-svelte | 8.3.6, 7.3.1 | npm | | build, dev | no |
| shadcn-svelte CLI | 1.8.0 | `npx shadcn-svelte@1.8.0 ...` (not needed after bootstrap) | small | bootstrap only | no |
| bits-ui | 2.20.6 | npm | | build | no |
| @tanstack/svelte-virtual | 3.13.52 (peer dependency says svelte ^4) | npm | | build | no |
| @wailsio/runtime | 3.1.0 | npm (template) | | runtime | no |
| mode-watcher, svelte-sonner, @internationalized/date, @lucide/svelte, tailwind-variants 3.3.1, tw-animate-css, @fontsource-variable/inter, cn 0.5.2, svelte-check 4.7.8, typescript 5.9.3 | as listed | added by the shadcn-svelte CLI or the Wails template | | build | no |
| vitest | 5.1.0 | npm | | test | no |
| @testing-library/svelte | 5.4.2 | npm | | test | no |
| jsdom | 31.0.0 (extra, required by vitest component tests) | npm | | test | no |
| @playwright/test | 1.65.0 | npm (uses the installed Google Chrome, no browser download) | | test | no |
| Google Chrome | installed on host | already present | | test | no |
| python3 | 3.14.7 | ships with the host | | test (total the results) | no |
| Docker Desktop | 29.7.2 | already present | `wails-cross` image about 4 GB | cross-linux | no |
| Wails cross image (`wails-cross`) | built from build/docker/Dockerfile.cross, edited to `golang:1.27-trixie` | `wails3 task setup:docker` | several GB, 5 to 10 min | cross-linux | yes (GTK and WebKitGTK in the image) |
| go-toast and other Go modules | pulled by Wails | go.mod | | build (Windows) | no |

Notes
- Task (the build runner, `task`) is built into wails3 (`wails3 task ...`); no separate install.
- Windows cross builds need no extra tool: Wails v3 on Windows does not use cgo, so `GOOS=windows go build` works from macOS.

## Clean machine steps

From a Mac with only Go 1.27.3, Homebrew and the Xcode Command Line Tools:

```sh
brew install node                       # Node 26, npm 11
git clone <repo> && cd <repo>/apps/wails
make deps                               # wails3 v3.1.0 + npm ci in frontend/ + go mod download
make run                                # desktop app (ad-hoc signed dev .app bundle in bin/)
make web                                # web app: http://localhost:8090 (port must be free)
```

Tests additionally need Google Chrome (`make test`), and cross builds need Docker Desktop (`make cross-linux`).
