# MIQT + GoTTH friction log

Times are local on 2026-11-09 (NZDT).

### 09:41 No Go slots for QWebChannel
- Step: Choose the JS to Go route.
- Command/API: `webchannel.QWebChannel.RegisterObject`, Qt headers and miqt sources.
- Outcome: QWebChannel exposes the slots and properties of a QObject's meta object. MIQT generates bindings but a Go type cannot declare a Q_INVOKABLE slot (that needs moc). Only existing Qt classes can be registered.
- Time lost: 5
- Category: framework-limitation
- Resolution: register a hidden `QLineEdit` and let JS write its `text` property, which has a notify signal; Go observes `OnTextChanged`. Messages carry a counter prefix because an unchanged value does not emit. Works, but it is a hack and not a typed call.

### 09:41 qwebchannel.js is not reachable from an http page
- Step: Load the client library.
- Command/API: `find /opt/homebrew -name qwebchannel.js`
- Outcome: the file lives in the qtwebchannel Cellar (`share/qt6/webchannel/`); `qrc:///qtwebchannel/qwebchannel.js` is only reachable from qrc pages, and the page is served over http.
- Time lost: 3
- Category: docs-gap
- Resolution: copied into `cmd/desktop/qwebchannel.js`, embedded with `go:embed`, injected through `QWebEngineScript` at DocumentCreation in the main world, together with a small glue script.

### 09:42 Qt WebEngine startup noise
- Step: First run.
- Outcome: about 75 lines "Property 'x' of object 'QLineEdit' has no notify signal ..." on stderr, one per non-notifying QWidget property of the registered object. Harmless. NSKeyBindingManager lines as in the MIQT app.
- Time lost: 0
- Category: runtime-bug
- Resolution: ignored.

### 09:42 A failed sed produced a stray empty file
- Step: Editing with BSD `sed -i` without an empty suffix argument created `cmd/desktop/indow.go`, which broke the build.
- Time lost: 2
- Category: tooling
- Resolution: removed the file.

### 09:45 Driving the menus without synthetic input
- Step: Verify menus.
- Command/API: System Events through `osascript` (accessibility was available this time).
- Outcome: menu clicks, the save panel and Quit worked, so the native menu, dialog, bridge and quit were exercised on a bare binary.
- Time lost: 0
- Category: toolchain

### 09:48 Packaged app fails: QtWebEngineProcess cannot find QtWebChannel
- Step: `make package`, then launch the bundle.
- Command/API: `macdeployqt`, `bin/GGUIX-miqt-gotth.app/Contents/MacOS/gguix-miqt`
- Outcome: dyld error in the helper: it resolves `@executable_path/../Frameworks/QtWebChannel.framework` inside `QtWebEngineProcess.app`, which has no `Frameworks` directory. macdeployqt also prints about 28 "Cannot resolve rpath QtVirtualKeyboard" errors for QML plugins that are not used; the build still completes.
- Time lost: 8
- Category: toolchain
- Resolution: the Makefile adds a `Frameworks` symlink inside the helper (`../../../../../../`) and re-signs ad hoc. The packaged app then loads the page and the bridge works.

### 09:46 Export file location
- Step: Save panel opened in the process working directory, so a verification export landed in `apps/miqt-gotth/gguix-export.csv` (deleted afterwards). Qt's static `QFileDialog` takes the dir argument; the shell passes the default name only.
- Time lost: 0
- Category: api-confusion
