# GoTTH toolchain

Every tool and system dependency beyond `go` itself. Sizes are approximate disk use.

| Name | Exact version | Install command | Size | Needed for | cgo related |
|---|---|---|---|---|---|
| templ CLI | v0.3.1100 | `go install github.com/a-h/templ/cmd/templ@v0.3.1100` | 20 MB binary | dev, build, test, web | no |
| shadcn-templ CLI | v2.0.0 | `go install github.com/axadrn/shadcn-templ/v2/cmd/shadcn-templ@v2.0.0` | 16 MB binary | project setup, `add` components, `bundle` after editing component JS | no |
| Tailwind standalone CLI | v4.3.5 | downloaded by `make deps` into `bin/tools/tailwindcss` from the GitHub release (`tailwindcss-macos-arm64`) | 96 MB | dev, build, test, web | no |
| htmx | 2.0.11 | vendored file `static/htmx/htmx.min.js` (fetched from jsDelivr by `make deps`, committed) | 52 KB | runtime | no |
| htmx-ext-sse | 2.2.4 | vendored file `static/htmx/sse.js` | 9 KB | runtime | no |
| shadcn-templ components | generated from registry (style `base-nova`, base color `neutral`) | `shadcn-templ add <names>` (network: https://shadcn-templ.com) | 6,000 lines of templ plus JS in `components/` | build, runtime | no |
| webview_go | v0.0.0-20240831120633-6173450d4dd6 | `go get` (module cache) | 1 MB source, compiled C++ | desktop build, runtime | yes (cgo, C++ and the system WebKit framework) |
| Xcode Command Line Tools (clang 21, WebKit headers) | installed (`/Library/Developer/CommandLineTools`) | `xcode-select --install` | n/a | desktop build | yes |
| macOS `osascript`, `pbcopy`, `open` | system | none | n/a | desktop runtime: save and open dialogs, notifications, clipboard, reveal | no |
| Node | 26.10.0 | `brew install node` | n/a | only `tests/browser/t9.test.mjs` (no npm packages) | no |
| Google Chrome | installed | none | n/a | only test T9 (headless, DevTools protocol) | no |
| Docker Desktop | 29.7.2 | none | n/a | only `make cross-linux` and `make cross-windows` (desktop shell) | yes (it supplies the C toolchains) |

No npm packages are needed. Nothing in the app requires npm.

Go module dependencies beyond the core: `github.com/a-h/templ`, `github.com/Oudwins/tailwind-merge-go` (used by the shadcn-templ `utils.CN`, comes from the template), `github.com/webview/webview_go`, and `golang.org/x/*` pulled in by the templ and shadcn-templ tools. The two `tool` lines in `go.mod` come from the shadcn-templ template.

## Clean machine steps

From a Mac with only Go, Homebrew and the Xcode Command Line Tools:

```sh
# 1. get the workspace (gguix-work with core/ and apps/gotth/), then:
cd gguix-work/apps/gotth
make deps        # installs templ and shadcn-templ with go install, downloads Tailwind and the htmx files
make run         # generates templ and CSS, builds cmd/desktop, opens the webview window
```

Web app:

```sh
make web         # generates, builds cmd/web, serves http://localhost:8090
```

Optional: `make test` (needs Node and Chrome for T9), `make bench`, `make package`.
Run `make deps` first on a fresh checkout: `bin/tools` and the vendored htmx files are not in git.
