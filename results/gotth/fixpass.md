# GoTTH fix pass (2026-11-09)

Files are relative to `apps/gotth/`.

## 1. Native menu bar, file dialogs, notification (macOS, build tag `native`)

Behind the build tag `darwin && native`. The default `make build` / `make run` is unchanged (in-page shadcn menubar, osascript dialogs and notification, pbcopy). `make build-native`, `run-native`, `package-native` and `bench-native` (not run) build and use `go build -tags native`.

- API: `webview.Window() unsafe.Pointer` (webview_go `webview.go:83,173`) returns the NSWindow. `Bind` is used so the page can push state to the menu.
- Files: `cmd/desktop/native_shim.go` (cgo, `//export gguixGoMenuAction`, `shimNative`), `cmd/desktop/native_shim.m` (Objective-C), `cmd/desktop/native_osa.go` (the old osascript/pbcopy code, now the default for everything except `darwin && native`), `cmd/desktop/native_types.go`, `cmd/desktop/main.go`.
- Menu: NSMenu main menu with App (About GGUIX, Quit Cmd+Q), File (Export CSV, Import Notes, Reveal Last Export), Edit (standard Cut/Copy/Paste/Select All via first-responder selectors, Copy Row Cmd+Shift+C), View (Table Cmd+1 and Cards Cmd+2 with check state, Theme submenu System/Light/Dark, Reload Cmd+R, Focus Search Cmd+F), Help (Host Info Cmd+I). Each item calls `GGUIX.actions[name]()` through `Eval`, the same actions the in-page menu and keyboard use. View/theme checkmarks, Export disabled while running and Reveal Last Export enabled after an export are synced from `static/desktop/desktop.js` through the bound `gguixMenuState`. With the native menu the page does not render the in-page menubar (`Options.NativeMenu`, `data-native-menu`) and `static/app.js` does not handle the Cmd shortcuts itself.
- Dialogs: NSOpenPanel (`.txt`) and NSSavePanel (default `gguix-export.csv`) shown as sheets on the webview NSWindow; the server goroutine waits on a semaphore so the page stays responsive.
- Clipboard: NSPasteboard (replaces pbcopy in this build).
- Notification: `UNUserNotificationCenter` when the process has a bundle identifier (`make package-native`, uses `desktop/Info.plist`), because it raises an exception for a bare binary. For an unbundled binary it uses the deprecated `NSUserNotificationCenter`, which accepts a bare process. This replaces osascript `display notification`.
- Other platforms: the default build files compile everywhere (as before); `native_shim.*` are excluded by the tag. `go build ./cmd/...` and `go build -tags native ./cmd/desktop` both succeed on this Mac; non-darwin was not rebuilt (no cross builds per the rules), but nothing darwin-only is in the default file set except the existing `appnap_desktop.*`.
- Verified: both variants compile and `go vet` clean; both binaries launch and stay running for several seconds. NOT verified: the menu, dialogs, shortcuts and notification were not exercised interactively. System Events accessibility is denied for osascript here and the only screen capture available was of the whole display (not usable), so there is no visual or automated evidence of menu rendering, dialog sheets, key-equivalent precedence over the webview, or notification delivery (including whether NSUserNotificationCenter still delivers on macOS 26 for a bare binary). Treat the shim as unverified at runtime.

## 2. Split view with draggable divider (web and desktop, default build)

- Component: shadcn-templ `resizable`, added with `shadcn-templ add resizable` (`components/resizable/`, bundle regenerated to `assets/js/shadcn-templ-e6355cae522da787.js`; the local `lifecycle.js` edit was skipped by the CLI, so it is preserved).
- `views/page.templ`: `PanelGroup` > `Panel` (65%, min 25%) / `Handle` / `Panel` (35%, min 20%), replacing the fixed flex split. Element ids `list-pane`, `divider`, `detail-pane` kept.
- Verified in Chrome-class browser pane against the web server: default 665/358 px at 1024 wide, and a mouse drag of the divider moved it to 511/512 px. Desktop uses the same page; not dragged inside the webview. Layout is not persisted (not required).

## 3. Column resize (web and desktop, default build)

Feasible, done. The shadcn table parts render `<th>`/`<td>`, and the header and body tables already use `table-layout: fixed` with a `<colgroup>`, so a width change is `<col>` width plus table width.

- `views/list.templ`: a `colGrip` span (`data-col-resize`) in each header cell; `static/app.js`: pointer drag, 40 px minimum, widths kept in memory and re-applied after every list swap (`applyColWidths`), the grip stops the click so it does not trigger sort.
- Verified in the browser pane: dragging the SKU edge changed it from 110 to 187 px in both header and body tables, widths survived a sort-triggered region swap, and the sort click was not triggered by the drag. Widths are not persisted across launches. Test: `ui_test.go`.

## Tests

`make test`: `GGUIX_TESTS passed=12 failed=0`. Added `ui_test.go` (split and grip markup, native-menu page omits the in-page menubar); run with `go test -run 'TestSplit|TestNativeMenu' .` (the GGUIX_TESTS line counts only the T-numbered scenarios).

## Linux fix (2026-11-09)

The desktop shell used osascript and pbcopy on every OS, so on Linux and Windows export, import and notifications failed. Replaced with cgo-free cross-platform libraries: ncruces/zenity v0.10.15 (dialogs), gen2brain/beeep v0.11.3 (notification), atotto/clipboard v0.1.4 (copy). The `-tags native` macOS shim is unchanged. Our app bug, not a GoTTH limit.

## Manual check by the user (2026-11-09)

- `-tags native` build: macOS menu bar shows GGUIX, File, Edit, View, Help; View > Cards switches the view. PASS.
