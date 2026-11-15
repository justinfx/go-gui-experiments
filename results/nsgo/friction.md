# Friction log: NSGo

Times are wall clock on the build day (approximate, the log was kept as the work went).

### 09:12 Bootstrap: `nsgo init` refuses an existing go.mod
- Step: following the project's recommended setup in a directory that already needed `go.mod` (`replace gguix/core`).
- Command/API: `go get -tool github.com/nsgo-dev/nsgo/cmd/nsgo@2345b2f`, then `go tool nsgo init -template go -name gguix-nsgo -module gguix/apps/nsgo .`
- Outcome: `error init: go.mod exists`. After moving `go.mod` away, `go tool nsgo` no longer existed (the tool directive lives in the go.mod). Ran the CLI from the module cache instead: `go run github.com/nsgo-dev/nsgo/cmd/nsgo@v0.0.0-20261114190341-2345b2fa5aba init ...`, then restored my `go.mod` (the template's says `go 1.26.0`).
- Time lost: 5
- Category: docs-gap
- Resolution: ran `init` before creating `go.mod`; kept the template's `nsgo.yaml` (edited name, id, targets) and discarded its `app.go`.

### 09:15 Plain core.Text is not themed
- Step: first run of the template.
- Command/API: `core.Text{Content: ...}` in a dark system appearance.
- Outcome: black text on a dark background. Only `ui` components take theme colors; `core.Text` inherits nothing until a parent sets `Color`.
- Time lost: 3
- Category: api-confusion
- Resolution: the root view sets `Color: ui.ColorsOf(cx).Text()` and `Background` in a `StyleFunc`; its text children inherit them.

### 09:20 Table cell content collapsed to zero height
- Step: text cells in `DataTable.RenderCell`.
- Command/API: `core.Text{Style: {FlexGrow: 1, FlexBasis: 0, MinWidth: 0}}` copied from the table's own default cell.
- Outcome: empty cells. The RenderCell wrapper is a column, so `FlexBasis: 0` gives height 0 there (the table's own cell sits in a row).
- Time lost: 5
- Category: api-confusion
- Resolution: `Width: core.Pct(100)` instead.

### 09:22 Header click does not sort
- Step: section 5.3 says clicking a header sorts.
- Command/API: `datatable.Column.Sortable`, `OnSortChange`, `RenderHeaderCell`.
- Outcome: only the small sort button (chevrons) that `Sortable` adds asks for a sort, and it cycles unsorted, ascending, descending. A header press selects the column (and Left/Right keys do too), so using column selection as a sort trigger would sort on arrow keys.
- Time lost: 15 (reading `base/datatable/datatable.go` and `view.go`)
- Category: framework-limitation
- Resolution: `RenderHeaderCell` returns a pressable view that calls the app's own sort (asc, then toggle) and shows ` ▲`/` ▼` in the label; `Sortable` is off. Column drag-to-move still works because the table claims the gesture only on move.

### 09:30 base/datatable has no look
- Step: first table render.
- Command/API: `datatable.DataTable` with no `Styles`.
- Outcome: no padding, no row lines, no selection highlight. The look is the app's job.
- Time lost: 5
- Category: docs-gap
- Resolution: used the gallery's own `Styles` for the table (`examples/gallery/datatable.go`), built from `ui.ColorsOf` only. No custom colors beyond theme roles.

### 09:35 SaveFile wants the data first
- Step: export with the native save dialog, then a background export to the chosen path.
- Command/API: `cx.SaveFile(core.SaveOptions{Name, MIME})`.
- Outcome: SaveFile copies a file or bytes to the place the user chose; there is no "ask for a path" call. With neither Path nor Data it saves an empty file and returns the chosen path.
- Time lost: 10
- Category: api-confusion
- Resolution: use that empty file as the placeholder and write the export over it; Cancel deletes it. On the web SaveFile takes the finished bytes (a download).

### 09:38 nsgotest panic: "a commit missed changes it should have sent"
- Step: going to row 500,000 in the Cards view in a headless test.
- Command/API: `base.List` (`core.List`) over 1,000,000 fixed 96 pt cards.
- Outcome: nsgotest's commit invariant check panics with 1 pt layout differences. It does not at 100,000 rows (10.4 million pt) and does at 200,000 (20.8 million pt): float32 positions are exact only up to 2^24 = 16.7 million. `core.List` does not scale its scroll range; the DataTable does (`base/datatable/layout.go`, `maxExtent`).
- Time lost: 15, plus 15 in the review pass
- Category: framework-limitation
- Resolution: review pass: Cards rebuilt on a single-column `DataTable` (104 pt rows, `RenderCell` card, 1 pt invisible header). No panic at rows 200,000 and 500,000. Deviation from "list widget" recorded in `capabilities.md`.

### 09:45 nsgotest default surface is a phone
- Step: first headless test run.
- Command/API: `nsgotest.Start`.
- Outcome: the surface is 390 x 844 and the model renders the phone layout (narrow table).
- Time lost: 3
- Category: docs-gap
- Resolution: `app.Resize(2600, 900)` in the test harness.

### 09:50 Header selector for Press
- Step: T2.
- Command/API: `app.Press("role:columnheader Name")`.
- Outcome: the press lands on the header cell, not on my inner pressable view, so nothing sorted.
- Time lost: 5
- Category: api-confusion
- Resolution: `role:columnheader >> text:Name`.

### 10:20 web-remote Cancel did not stop the export
- Step: review pass, streaming the export from the server into a picked file.
- Command/API: `remote.Client.Export(ctx, ...)` in js/wasm.
- Outcome: Cancel set the context but the download carried on to 100%. Go's js/wasm transport (`net/http/roundtrip_js.go`) aborts the fetch on context cancel only while waiting for headers; `streamReader.Read` ignores it.
- Time lost: 20
- Category: core-issue (the core's client relies on the context)
- Resolution: the app wraps the writer (`ctxWriter` in `export.go`) so a write after Cancel fails with `context.Canceled`, which ends the client's read loop.

### 10:35 Linux window identity is not set by NSGo
- Step: review pass; looked for an app-id option in `desktop`, `nsgo.Run` and `internal/platform/linux`.
- Command/API: none found; `core.App` (ldflags `core.appID`) only feeds notifications and the launcher entry.
- Outcome: bare `go build` gave an empty `WM_CLASS`, `app_id` `GTK Application` and an empty notification name. Passing NSGo's own ldflags fixed the notification name only.
- Time lost: 20
- Category: framework-limitation
- Resolution: `identity_desktop.go` sets the GLib program name through purego before the window is made; `github.com/ebitengine/purego v0.11.3` (NSGo's own dependency) is now a direct requirement of the app's `go.mod`.

### 10:40 Linux dialogs: typed paths and key names
- Step: automating the portal chooser.
- Command/API: `xdotool type` into the GTK location entry; `wlinput.py key Escape`.
- Outcome: fast typing garbled the path through the entry's completion (`notes.txtxt`, which the app reported as an error dialog, correctly); `wlinput.py` calls Escape `esc`.
- Time lost: 15
- Category: toolchain (test automation only)
- Resolution: slower typing plus Delete before Return; `esc`.

### 10:05 Linux: file dialogs and Open need a working portal
- Step: Linux runtime check.
- Command/API: `cx.SaveFile`, `cx.PickFiles`, `cx.OpenFile` (GtkFileDialog and GtkFileLauncher, through `org.freedesktop.portal.*`).
- Outcome: in the harness' GNOME-flavored session the FileChooser portal has no backend, the call fails, and the app sees a cancelled dialog with no error (nothing is shown to the user). With a `portals.conf` that prefers the `gtk` backend the dialogs appear, are transient for the app window and work (Save pre-fills `gguix-export.csv`). The portal then stopped answering in some later runs (`No such interface FileChooser`), which made the import dialog flaky to automate. `OpenURI` was never available (Reveal Config Folder through `cx.OpenFile` did nothing, silently).
- Time lost: 25
- Category: framework-limitation (silent failure) plus a container limit (portal)
- Resolution: unresolved for the app; recorded in `linux.md`. The harness was not changed (portal preference is set in the session's own `XDG_CONFIG_HOME` by my script).

### 10:15 Linux: window identity, Escape, console noise
- Step: Linux runtime check.
- Command/API: window class, `ui.Dialog` Escape, stderr.
- Outcome: `WM_CLASS` empty on X11, `app_id` `GTK Application` on Wayland (fixed in the review pass, see above). Escape did close the Host dialog once the right key was sent (my first test used the wrong key name). Constant `GLib-GObject-CRITICAL ... g_value_get_int` lines on stderr.
- Time lost: 5
- Category: runtime-bug
- Resolution: identity fixed; the console noise is unresolved, recorded.

### 11-15 macOS hands-on fixes
- Step: author hands-on pass on macOS (see handson.md).
- Escape and the Host dialog: `internal/platform/macos` turns Escape into "back" (which closes a modal) only when no text field is being edited (`isBack` returns `!editing`). A click on the Host button does not take the keyboard focus from the Search field, so with the Search field focused Escape never reached the dialog. Linux has no such rule. Fix, using NSGo's own API: while any dialog is open, a `cx.OnKeyWith("escape", core.KeyOptions{When: core.KeyAlways}, ...)` binding closes it, and `cx.DismissKeyboard()` takes the focus out of the field so native back works too. Checked on macOS with the Search field focused: Escape closes the Host dialog. Category: framework-limitation (modals do not take focus from a text field on macOS).
- Table/Cards switch: `ui.Tabs` (segmented) marks the selected tab with a low-contrast raised thumb. Replaced with `ui.ToggleGroup` of two `ui.ToggleButton`, which fills the selected one with the accent. Screenshot: shots/fix-toolbar.png.
- Revert never active: not reproduced. The `Disabled` closure is the same as Save's and the test harness shows Save and Revert both enable on the first edit and both disable after Save or after Revert (assertions added to TestT6Save). Not fixed beyond that; likely a visual reading of the dimmed Secondary button, unconfirmed. Category: not-reproduced.
- Fast wheel scrolling flashes: simulated the pager against a 1,000,000 row in-memory service at 60 visible-range updates per second. With the old options (2 in flight, 50 row prefetch) every frame had unloaded rows at 20,000 rows/s; with 4 in flight, 400 row prefetch and 40 cached pages, 0 of 60 frames did at 20,000 rows/s and 0 at 3,000. At 100,000 rows/s both still show placeholders in every frame (less than 2 pages of lookahead cannot cover it). After the wheel stops the pager was loaded within microseconds in both, so the "keeps catching up" tail is not the app's paging: it is the table's momentum (inertia) scrolling that NSGo's macOS wheel handling continues after the wheel stops, and the visible range it reports. The app cannot change that. Category: framework-limitation (inertia) plus an app-side prefetch tune.
- Search and Go-to-row entries: checked by screenshot; the text and placeholders are vertically centered and both are 28 pt tall, the same as the buttons and selects. No change.

## Summary

- Wall time to first window: about 5 minutes (the template ran at 09:15; this app's first full window at about 09:27).
- Wall time to feature complete (macOS, web, tests): about 70 minutes; Linux runtime check about another 45.
- Entries per category: docs-gap 3, api-confusion 4, toolchain 1, build-error 0, runtime-bug 1, framework-limitation 4 (one also container-related), core-issue 1.
- Three biggest pain points: (1) `core.List` is not exact past 2^24 points (Cards had to move onto the DataTable); (2) no first-class header-click sort in the table and no `ui` styled table (the look and the cells are assembled by hand); (3) Linux integration is portal-dependent and fails silently (file dialogs, Reveal), with no window identity.
- One dependency line was added to `go.mod`: `github.com/ebitengine/purego v0.11.3`, already a dependency of NSGo at that version, used directly in `identity_desktop.go` (Linux window identity).
