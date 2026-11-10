# Capabilities (MIQT, Qt 6.12.0, miqt v0.15.0)

| requirement | status | how it was done | notes |
|---|---|---|---|
| Window title, 1280x800, min 1024x640, restore size | native | QMainWindow, size saved in prefs | |
| Search field + 250 ms debounce | native | QLineEdit + single-shot QTimer | |
| Category filter button + checkbox popup + Clear | native | QPushButton menu with QWidgetAction(QCheckBox) | menu stays open while ticking |
| Status filter | native | QComboBox | |
| Go to row | native | QLineEdit + QIntValidator, QPushButton, Return | clamps |
| View switcher | native | two checkable QPushButtons in exclusive QButtonGroup | |
| Split view, draggable divider | native | QSplitter | |
| Table | native | QTableView + QAbstractTableModel on rowStore (adapter_*_desktop.go) | page arrivals marshalled with mainthread.Start and coalesced per 4 ms; see adapter.md |
| Column resize | native | QHeaderView default | |
| Sort indicator in header text | native | model HeaderData returns title + arrow | |
| Virtualization | native | model rowCount = Total, uniform row height | 1M rows scroll fine |
| Zero results text | framework-drawn | paintEvent override draws "No items match." | |
| Stock bar, Status badge, Stars cells | native | QStyledItemDelegate subclass (OnPaint) on columns 6,7,8, no cell widgets | files delegates_desktop.go; base paint draws selection, custom drawn on top; placeholder "…" via model text |
| Cards view item | native | QListView on the table's model (column 0) + QStyledItemDelegate (OnPaint, OnSizeHint), 96 px cards, 8 px gap | Placeholder card shows "Loading…". Muted text uses PlaceholderText role, border Mid, accent Highlight |
| ID/SKU/Updated read-only text | native | QLabel | |
| Name, Price | native | QLineEdit | |
| Category | native | QComboBox | |
| Quantity | native | QSpinBox 0..1,000,000 | |
| Status radio group | native | QRadioButton x3 in QButtonGroup | |
| Rating slider + label | native | QSlider 0..50 tenths, QLabel | |
| Featured | native | QCheckBox | |
| Restock date + Clear | fallback | QDateEdit with calendar popup; cleared state is the minimum date 1999-12-31 shown as "None" via special value text | QDateEdit has no empty state |
| Notes | native | QPlainTextEdit fixed to 4 lines | |
| Inline field errors | native | QLabel under each field | Error color: BrightText is not red in macOS palettes and Qt has no error role, so the platform system red (light 215,0,21 / dark 255,69,58) is used |
| Dirty state, Save, Revert, Copy Row, Import Notes | native | see form_desktop.go | |
| Confirm dialog | native | QMessageBox with Discard/Cancel buttons | |
| Clipboard | native | QGuiApplication.Clipboard().SetText | |
| File dialogs | native | QFileDialog static GetSaveFileName/GetOpenFileName (NSSavePanel/NSOpenPanel) | not exercised in tests (hooks) |
| Export background job, progress, cancel | native | goroutine + core.Export, mainthread.Start | |
| OS notification | native (unverified visually) | QSystemTrayIcon.ShowMessage with tray shown briefly; status bar message always shown too | not verified that macOS displays it for an unbundled binary; offscreen has no tray so test uses the in-app text |
| Menus | native | QMenuBar, native macOS bar, About/Quit with menu roles | |
| Shortcuts | native | QAction shortcuts "Ctrl+..." (Cmd on macOS) | Edit menu forwards Cut/Copy/Paste/Select All to the focused field |
| Theme modes | native via workaround | QStyleHints property "colorScheme" set through QObject.SetProperty (no typed setter in binding) | verified on cocoa |
| Live system theme switching | native, not verified | palette change event repaints custom cells; System reads palette lightness | could not toggle OS appearance |
| UI-thread mechanism | native | qt6/mainthread Start/Wait | |
| Framework prefs facility (unused) | n/a | QSettings would be used | core PrefsStore used instead |
| Host panel (desktop) | native | QDialog, host.Current, Source row "native" | Reveal Config Folder via QDesktopServices.OpenUrl; Reveal Last Export via core host.RevealInFileManager (Qt cannot select a file) |
| About dialog | native | QMessageBox plain text, 4 lines | |
| Status bar | native | QStatusBar with permanent widgets | |
| Web, web-remote, bench-web | gap | prints UNSUPPORTED | no supported route found in MIQT docs |
| cgoprobe desktop | native | `make cgoprobe` | CGOPROBE desktop PASS |
| cross-windows | gap | miqt-docker win64-qt6-static, extra tools: Docker | FAIL: image go1.26 cannot build the go 1.27 module, toolchain download fails with TLS error in the container; 3 attempts, see friction.md |
| cross-linux | see below | miqt-docker linux64-go1.27-qt6.12, Docker amd64 emulation | FAIL (timebox): image built, but the amd64-emulated go build had no output after 28 minutes (17:02 to 17:30), killed at the 30 minute limit; no binary. Try on a native amd64 host or with more time |
| Packaging | native | manual .app skeleton + macdeployqt | 30 MB before, 111 MB after; macdeployqt printed rpath errors for QtVirtualKeyboard plugins, bundle still links its own frameworks |
| T1 to T12 | native | app_test.go | all pass, plus 6 extra scenarios (go-to, filters, export complete, import notes, prefs restore, about); mouse events not synthesized (no qtest), selection and signals driven directly |
| Bench | native | driver_bench.go, `make bench` | all 8 events; numbers vary with machine load |
