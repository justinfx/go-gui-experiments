# Capabilities: go-gui v0.96.0

| Requirement | Status | How | Notes |
|---|---|---|---|
| Window title, size, min size, size restore | native | `WindowCfg{Width,Height,MinWidth,MinHeight}`, size saved from `EventResized` | |
| Search field with 250 ms debounce | framework-drawn | `gui.Input` + `time.AfterFunc` posted via `QueueCommand` | |
| Category popup | framework-drawn | Button + floating `Column` with `Checkbox` and Clear | closes by clicking the button again, not on outside click |
| Status filter | framework-drawn | `gui.Select` | |
| Go-to-row | framework-drawn | `gui.Input` (digits only via `PreTextChange`) + Go button, Enter | |
| View switcher | framework-drawn | `gui.SegmentedControl` | |
| Theme dropdown | framework-drawn | `gui.Select`, synced with View > Theme | |
| Split view | framework-drawn | `gui.Splitter`, draggable | detail pane content can overflow at narrow widths |
| Table | framework-drawn | header row of ghost `Button`s + `gui.VirtualList` (fixed row height 28) driven by `core.Pager` | `Table`/`DataGrid` could not use the pager (friction) |
| Column resize | gap | fixed widths; horizontal scroll container | no resize in a hand-built header |
| Sort headers with arrows | framework-drawn | header buttons, ` ▲`/` ▼` in label | |
| Virtualization | framework-drawn | `VirtualList`, scrollbar spans Total | visible range read in an `AmendLayout` from `ScrollVerticalOffset` |
| Keyboard Up/Down | framework-drawn | `OnKeyDown` on focusable list box | |
| Stock bar cell | framework-drawn | nested containers with fill color and radius (`cells.go`) | |
| Status badge | framework-drawn | rounded container + colored text (`cells.go`) | |
| Stars | framework-drawn | `gui.DrawCanvas`, triangles via `FilledPolygon`, half star clipped | |
| Card item | framework-drawn | `VirtualList` item = row container (tile, columns, chips) | height 96 + 8 gap |
| Detail form: text fields | framework-drawn | `gui.Input` (read-only for ID/SKU/Updated) | |
| Category dropdown | framework-drawn | `gui.Select` | |
| Quantity spin box | framework-drawn | `gui.NumericInput` (Up/Down steps, min/max) | |
| Status radios | framework-drawn | `gui.RadioButtonGroupRow` | |
| Rating slider | framework-drawn | `gui.Slider` step 0.1 + label | |
| Featured | framework-drawn | `gui.Checkbox` | |
| Restock date picker | framework-drawn | `gui.InputDate` + Clear | |
| Notes multi-line | framework-drawn | `gui.Input{Mode: InputMultiline}` | |
| Confirm dialog | framework-drawn | `w.Dialog(DialogCustom)` with Discard/Cancel | `NativeConfirmDialog` has fixed button labels |
| Clipboard | native | `w.SetClipboard` | |
| File dialogs | native | `w.NativeSaveDialog`, `w.NativeOpenDialog` | |
| Notifications | native | `w.NativeNotification`, status-bar fallback on failure | |
| Menus | native (macOS) | `App.SetNativeMenubar`, `IncludeEditMenu` | Copy Row in File (Edit menu fixed); checked items, not radio |
| Shortcuts | native | `w.RegisterCommands` (Global), referenced by menu `CommandID` | |
| Theme modes, live system switch | native | `SetTheme(ThemeLight/ThemeDark)`, `FollowSystemAppearance` + `OnSystemAppearance` | |
| UI-thread mechanism | native | `Window.QueueCommand` (+ `InvalidateLayout` from pager goroutine) | |
| Native prefs facility (unused) | n/a | `gui.LoadSettings`/`SaveSettings` (JSON under UserConfigDir, localStorage on web) | core `PrefsStore` used instead |
| Reveal in file manager | fallback | `host.RevealInFileManager` | go-gui has no reveal API |
| Host panel desktop | native | `host.Current`, all fields, actions enabled | |
| Host panel web | fallback | `Source: unavailable`, actions disabled with note | build verified, not run in a browser here |
| Host panel web-remote | fallback | `server-process` via `/api/host`, actions disabled | build verified only |
| Export desktop | native | save dialog, background goroutine, progress/cancel | |
| Export web | framework-drawn + syscall/js | in-memory CSV, Blob download | not exercised in browser this session |
| Web target | works (build) | `GOOS=js` build, `gogui.js` + `wasm_exec.js`; backend is Canvas2D, not WebGL | `make web` built and served |
| Web-remote target | works (build) | `-tags remote` | |
| Package | native | go-gui `cmd/buildapp` | produced `bin/GGUIX-gogui.app` |
| cgoprobe desktop | PASS | | |
| cgoprobe web | FAIL | no cgo on wasm (expected) | |
| cgoprobe web-remote | PASS | client without tag, server with cgoprobe | server run not verified this session |
| cross-linux | PASS | `CGO_ENABLED=0`, no extra tools | ~35 MB |
| cross-windows | PASS | `CGO_ENABLED=0`, no extra tools | ~26 MB |
| T1-T12 | pass | `gui.NewTestWindow`, `TestRender`, `TestClick`, layout walk, plus model calls | T10 checks fill width/color via layout shapes; badge colors via palette |

## Target notes

Desktop-only code is in `main_desktop.go`, `platform_desktop.go` (native dialogs, notifications, reveal) and `menu_desktop.go` (native menubar). Web-only code is `main_web.go`, `platform_web.go` (Blob download, file input via syscall/js, no menubar) and `web/index.html`. Everything else (model, view, cells, commands, bench driver) is shared, because go-gui renders the same widget tree on both. The Makefile marks desktop and web blocks.
