# Shell: MIQT v0.15.0 + QtWebEngine hosting GoTTH

## Design
`cmd/desktop/main.go` starts the unchanged GoTTH server on `127.0.0.1:0`, creates a `QApplication` (with `AA_ShareOpenGLContexts`, required by WebEngine), a `QMainWindow` whose central widget is a `QWebEngineView`, and loads the server URL. `gotth.Options.NativeMenu` is true, so the page omits its in-page menu bar. Closing the window or the Quit action ends `QApplication_Exec`, then `srv.Flush()` saves prefs.

## Files
| File | Role |
|---|---|
| `cmd/desktop/main.go` | startup, server, event loop |
| `cmd/desktop/window.go` | window, WebEngine page, QWebChannel bridge, menu bar |
| `cmd/desktop/native.go` | `gotth.Native` with Qt: QFileDialog, QClipboard, QSystemTrayIcon |
| `cmd/desktop/qwebchannel.js` | Qt's client library (embedded, injected) |
| `cmd/desktop/appnap_desktop.*` | kept from gotth (macOS App Nap guard, not a webview feature) |
| `desktop/Info.plist` | bundle metadata |
| `Makefile` | desktop block replaced; cross targets exit 3 |

Removed vs gotth: `native_shim.{go,m}`, `native_osa.go`, `native_types.go`, `desktop/cross.sh`, the `-tags native` targets, the webview_go dependency. Server, views, JS, tests, core: unchanged.

## What is native
Window and its min size, QMenuBar (items, shortcuts, radio groups, role-tagged About and Quit), file open and save dialogs, clipboard, notification route. The page content is still HTML in Chromium, so every control inside the window is web-drawn, as in gotth.

## Bridge routes
- Go to JS: `QWebEnginePage.RunJavaScriptWithScriptSource` calls `GGUIX.actions[name]`.
- JS to Go for menu state: QWebChannel (see friction.md for the QLineEdit property trick).
- JS to Go for dialogs, copy, quit, window size: the existing fetch routes into the server, because they need the Go server anyway.

## Limits and evidence
- Native Go slots are impossible with MIQT (no moc), hence the property hack. Evidence: friction.md.
- Cross builds: not feasible. `~/go/pkg/mod/github.com/mappu/miqt@v0.15.0/docker/linux64-go1.27-qt6.12-dynamic.Dockerfile` installs base Qt only (`aqt install-qt linux desktop 6.12.0 linux_gcc_64`, no `-m qtwebengine`); `win64-cross-go1.26-qt6.5-static.Dockerfile` (the image the MIQT app uses) is a static MinGW Qt with no WebEngine. `grep -i webengine` over those Dockerfiles finds nothing. The Make targets print the reason and exit 3. The images were not run.
- Notification: the call succeeds, banner display not observed. A tray icon flashes in the menu bar. UNUserNotificationCenter would need a Go-callable Objective-C shim, which MIQT does not provide.
- Bundle is 335 MB (WebEngine) against 111 MB for the widget MIQT app. The packaged app is ad hoc signed only.
- macdeployqt prints QtVirtualKeyboard rpath errors; harmless here.
- Not exercised: keyboard shortcuts, Import Notes dialog, Copy Row, window size restore, bench run (not allowed).
- Marking: all of `cmd/desktop/` and `desktop/` are desktop code; the Makefile desktop block carries `gguix:target desktop`.
