# MIQT + GoTTH capabilities

Status values: native, framework-drawn, fallback, gap, not feasible.
Everything not listed here (all in-page controls, windowed lists, form, theme, prefs, tests, web target) is unchanged from `results/gotth/capabilities.md`, because the Go server, templ views, htmx and JS are the same. Files are relative to `apps/miqt-gotth/`. Verified items say how; "not exercised" means compiled but not driven.

## Section 5 requirements that the shell changes

| Requirement | Status | How it was done | Notes |
|---|---|---|---|
| Window title `GGUIX - GoTTH` | native | `QMainWindow.SetWindowTitle` (`cmd/desktop/window.go`) | verified by System Events window name |
| Window 1280 x 800, min 1024 x 640, restore saved size | native | `Resize(prefs.WindowWidth, WindowHeight)`, `SetMinimumSize2(1024, 640)`; `static/desktop/desktop.js` (unchanged) still posts the page size to `/prefs/window` | initial window measured 1280 x 832 including the 32 px title bar. Restore on relaunch not exercised |
| Menu bar, shortcuts | native | `QMenuBar` with `QAction`, `QKeySequence` ("Ctrl+" is Cmd on macOS), `QActionGroup` radio items, About/Quit role-tagged (`window.go` `buildMenus`). Items run `GGUIX.actions[name]` through `QWebEnginePage.RunJavaScriptWithScriptSource`; Cut/Copy/Paste/Select All use `QWebEngineView.TriggerPageAction` | the page hides its in-page menu bar (`NativeMenu: true`). Verified by clicking View > Theme > Dark, View > Cards, File > Export CSV... and App > Quit through System Events. Keyboard shortcuts not exercised |
| Menu state (view, theme radios, Export enabled, Reveal enabled) | native, via QWebChannel | page calls `gguixMenuState`, which writes the `text` property of a registered `QLineEdit`; Go observes `OnTextChanged` | verified in the log: theme and view changes, export running and finished all reach Go. See shell.md for why a QLineEdit |
| Import Notes open dialog (.txt) | native | `QFileDialog_GetOpenFileName4` on the GUI thread through `mainthread.Wait` | Qt uses NSOpenPanel on macOS. Not exercised (same call path as save) |
| Export save dialog (CSV) | native | `QFileDialog_GetSaveFileName4` | panel appeared as a native "Export CSV" window and Return saved 1,000,002 lines |
| OS notification `Export complete` | native route, delivery not verified | `QSystemTrayIcon.ShowMessage3`; icon created on first use, hidden 8 s later | no error from the call. Whether macOS showed a banner was not observed. A tray icon appears briefly in the menu bar |
| Copy Row | native | `QGuiApplication_Clipboard().SetText` via `/native/copy` | not exercised; page also tries the Clipboard API |
| Reveal Last Export, Reveal Config | native (core helper) | `host.RevealInFileManager` in `handlers_desktop.go`, unchanged | |
| Host panel | native | unchanged (`host.Current`) | |
| Threading mechanism | native | `qt6/mainthread.Wait` and `Start` from HTTP handler goroutines | dialogs block only the calling handler |
| Native prefs facility | n/a | unchanged, core `PrefsStore` | |

## Section 6 (shell specific)

| Requirement | Status | How | Notes |
|---|---|---|---|
| In-process Go server on a random 127.0.0.1 port, shown in QtWebEngine | native | `cmd/desktop/main.go`, `QWebEngineView.SetUrl` | |
| JS to Go bridge | native (QWebChannel) plus the existing HTTP routes | `qt6/webchannel` `RegisterObject`, `qwebchannel.js` injected with `QWebEngineScript` at document creation | HTTP routes remain the bridge for dialogs, copy, quit, as in gotth |
| `package` builds the .app | fallback | `make package`: binary, Info.plist, `macdeployqt`, a symlink for the WebEngine helper, ad hoc codesign | 335 MB. Packaged app launches and the bridge works |

## cgoprobe

| Target | Result | Notes |
|---|---|---|
| web server | PASS | |
| desktop shell | PASS | `make cgoprobe` |
| web export test | PASS | |

## Cross builds

| Target | Route | Result | Notes |
|---|---|---|---|
| linux/amd64 desktop | none | not feasible (exit 3) | `docker/linux64-go1.27-qt6.12-dynamic.Dockerfile` installs `aqt install-qt linux desktop 6.12.0 linux_gcc_64` with no `-m qtwebengine` module and no WebEngine dependencies |
| windows/amd64 desktop | none | not feasible (exit 3) | the Windows images are static MinGW Qt builds (`win64-cross-go1.26-qt6.5-static.Dockerfile` downloads a "staticFull" Qt 6.5.3 archive); none mention WebEngine, and QtWebEngine cannot be built with MinGW. Not built here, so "no WebEngine" is from the Dockerfiles, not from running the image |
| web server | plain `go build` | not part of this target; the Make targets exit 3 for both |

## Tests

`make test`: `GGUIX_TESTS passed=12 failed=0 not_feasible=0` (the test suite is the gotth suite, unchanged).

## Bench

Not run (benchmarks are measured separately).
