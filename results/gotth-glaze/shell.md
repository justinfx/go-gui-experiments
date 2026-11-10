# GoTTH + glaze: the desktop shell

## Design
`cmd/desktop` runs the unchanged GoTTH server in process on `127.0.0.1:0` and shows it in a `glaze.WebView`. Everything the webview lacks goes through the existing `gotth.Native` interface (`native.go` in the app package), now implemented with pure-Go packages. No hand-written C or Objective-C. `CGO_ENABLED=0` is used by `make build`.

## Files
| File | Role |
|---|---|
| `cmd/desktop/main.go` | `glaze.New`, title, size (`HintNone` from prefs, `HintMin` 1024 x 640), Navigate, Run, Flush |
| `cmd/desktop/menu.go` | `glaze/menu` menu bar, actions via `WebView.Eval` into `GGUIX.actions[...]`, `gguixMenuState` binding |
| `cmd/desktop/native.go` | `glazeNative`: `SaveFile` / `OpenFile`, `notify.Send`, `clipboard.WriteText`, `openurl.Reveal` |
| `cmd/desktop/appnap_darwin.go`, `appnap_other.go` | App Nap guard through purego `objc` |
| `handlers_desktop.go` | `Server.reveal` (optional `Reveal` on `Native`, else core helper) |
| `Makefile`, `desktop/cross.sh`, `README.md`, `go.mod` | build with `CGO_ENABLED=0`, `-tags native` targets removed, cgoprobe lines, cross route, dependencies |

## Changed versus gotth
Removed: `native_shim.go`, `native_shim.m`, `native_osa.go`, `native_types.go`, `appnap_desktop.go`, `appnap_desktop.m`, `build-native`, `package-native`, `run-native`, `bench-native`, webview_go. Server, views, JS, tests, core: untouched (`core/` is the same directory; `make test` gives `GGUIX_TESTS passed=12 failed=0 not_feasible=0`).

## Native versus fallback
| Feature | Mechanism | Status |
|---|---|---|
| Menu bar | `glaze/menu` | native on macOS; Windows bar without shortcuts; Linux in-page fallback |
| Open / save dialogs | `WebView.OpenFile` / `SaveFile` (`dialog_*.go` per OS) | native on all three (compiled for Windows and Linux only) |
| Notification | `native/notify` | native on macOS; no backend elsewhere, page toast |
| Clipboard | `native/clipboard` | native (macOS tested; Windows and Linux backends exist) |
| Reveal | `native/openurl` | native; Linux opens the folder only |
| Resize persistence | page posts its size | fallback (same as gotth) |
| Host panel | core `host.Current` | unchanged |

## Evidence
- API checks: `~/go/pkg/mod/github.com/crgimenes/glaze@v0.0.83/webview_common.go` (WebView interface), `dialog.go`, `menu/menu.go`, `menu/menu_windows.go` lines 9 to 11 (shortcuts ignored), `menu/menu_linux.go` header (no bar); `~/go/pkg/mod/github.com/crgimenes/native@v0.1.18/notify/notify.go` and `notify_other.go`, `openurl/openurl.go`, `clipboard/clipboard.go`.
- Build: `CGO_ENABLED=0 go build ./cmd/desktop` for darwin/arm64 (run), windows/amd64, linux/amd64, windows/arm64, linux/arm64 (compile only). `go vet ./cmd/desktop` clean for darwin and linux.
- cgoprobe: `make cgoprobe` printed `CGOPROBE web PASS`, `CGOPROBE desktop FAIL ../../core/checksum_nocgo.go:8:9: undefined: cgoprobe_requires_cgo_but_cgo_is_unavailable_for_this_target`, `CGOPROBE desktop-cgo PASS`, `CGOPROBE web-export-test PASS`. The desktop FAIL is the expected result with cgo off.
- Runtime on macOS (arm64): System Events listed the menu bar items and menu item names before and after a View > Cards click; the Export CSV menu item produced a second window and Escape dismissed it; title and size read back.

## Limits
- Windows: menu shortcuts not wired by glaze (page JS covers them); not run.
- Linux: no menu bar; in-page menubar used; not run.
- The macOS app menu title is the process name (`gguix-gotth`), not `GGUIX`; the first top-level item title is replaced by macOS, as documented in `menu.Item.Title`.
- Menu rebuild on every state change; a menu open at that moment is not tested.
- Notification banner, Finder reveal and Import file selection were not looked at; Bench was not run.
