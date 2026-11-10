# GoTTH + glaze toolchain

Same tools as `results/gotth/toolchain.md` (templ v0.3.1100, shadcn-templ v2.0.0, Tailwind v4.3.5, htmx 2.0.11, htmx-ext-sse 2.2.4, Node and Chrome for T9) except for the desktop shell.

| Name | Exact version | Install command | Needed for | cgo related |
|---|---|---|---|---|
| glaze | github.com/crgimenes/glaze v0.0.83 | `go get` (module cache) | desktop window, dialogs, `glaze/menu` | no (purego, dlopen at run time) |
| native | github.com/crgimenes/native v0.1.18 | `go get` | notify, clipboard, openurl | no |
| purego | github.com/ebitengine/purego v0.11.3 | pulled in by both; direct import for the App Nap guard | desktop | no |

Removed compared with gotth: `github.com/webview/webview_go`, the Xcode Command Line Tools as a desktop build requirement (clang, WebKit headers), and Docker with mingw / libgtk-3-dev / libwebkit2gtk-4.0-dev for the cross builds. `go build` with `CGO_ENABLED=0` is the whole desktop build. The Command Line Tools are still handy for `git` and `make`.

Run-time requirements of the desktop binary: macOS WKWebView (system); Windows WebView2 runtime (glaze finds it through the registry); Linux WebKitGTK (4.0, 4.1 or 6.0) plus GTK3 or GTK4 loadable by the dynamic linker. Linked libraries of the macOS binary (`otool -L`): libSystem, libresolv, CoreFoundation, Security only; WebKit, AppKit and UserNotifications are dlopened.

Binary size (macOS arm64, `CGO_ENABLED=0`): `bin/gguix-gotth` about 14 MB.

Clean machine steps: as gotth (`make deps`, `make run`), with Go 1.27.3; the Xcode CLT and Docker lines are not needed for the desktop build.
