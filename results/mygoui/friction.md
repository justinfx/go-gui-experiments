# Friction log: MyGo native UI (mygoui)

Times are wall clock on the build Mac (the day started at 12:00).

### 12:00 Reading the docs
- Step: finding the native UI API for table, list, form widgets, testing and browsers.
- Command/API: read `docs/ui/*.md` under `$(go env GOMODCACHE)/github.com/egoist/mygo@v0.9.0`, `go doc`, grep of `ui/api_gen.go`.
- Outcome: the per-widget docs are accurate and complete. `go doc github.com/egoist/mygo/ui` lists almost no constructors (they are generated into `api_gen.go`), so grep was needed to list them.
- Time lost: 3
- Category: docs-gap
- Resolution: grep `^func [A-Z]` in `ui/api_gen.go`.

### 12:04 Bootstrap
- Step: project template.
- Command/API: `go run github.com/egoist/mygo/cmd/mygo@v0.9.0 init -template native boot`, copied into `apps/mygoui`, module renamed to `gguix/apps/mygoui`, `replace gguix/core => ../../core`, `"webview": false` added to `mygo.json`, template `main.go` and `main_test.go` removed.
- Outcome: works. `go mod tidy` dropped the `gguix/core` require while no code imported it, so it was re-added by hand.
- Time lost: 2
- Category: toolchain
- Resolution: wrote the require line manually.

### 12:07 Table header sort arrow
- Step: section 5.3 wants ` ▲`/` ▼` inside the header label.
- Command/API: `ui.TableColumn{Sortable}`, `ListState.Sort`.
- Outcome: the table draws its own chevron next to the title for the sorted column, so the label text would show two arrows.
- Time lost: 4
- Category: framework-limitation
- Resolution: the app keeps `ListState.Sort` pointing at a scratch `SortOrder`, reads the clicked column from it after the table is built, resets it, does the ascending/descending toggle itself and puts the arrow glyph in the title. The table never shows its own arrow.

### 12:09 Selected card text turned white
- Step: cards view, selected card.
- Command/API: `ui.List` with `ListState.Selected`.
- Outcome: the list row paints a chosen row in the accent color and sets the text color to `AccentText`, so the card's text (which inherits it) went white on the card surface.
- Time lost: 3
- Category: api-confusion
- Resolution: set `TextColor(theme.Text)` on the card box. The accent row background still shows behind the card, its corners match the card radius so only the 2 px accent border is visible.

### 12:10 Re-copied binary killed by macOS
- Step: `make build` copies the bare executable out of the `.app`.
- Command/API: `cp .../Contents/MacOS/GGUIX-mygoui bin/gguix-mygoui` over an existing file.
- Outcome: the second copy over the same path was killed with SIGKILL (exit 137) and showed no window, with no message.
- Time lost: 6
- Category: toolchain
- Resolution: `rm -f` the target before copying (in the Makefile).

### 12:11 Tests could not see off-screen cells
- Step: T10 custom cells in the table.
- Command/API: `ui.Tester.Find("status badge")`.
- Outcome: at 1280 px the Status column is scrolled out of the 65% pane, so `Find` returned a zero-size box. The tester window is wider in the test.
- Time lost: 3
- Category: api-confusion
- Resolution: tester size 2400 x 800 in the test harness.

### 12:12 Web bench export
- Step: bench on the web target.
- Command/API: `bench.Run` passes a file path to `StartExport`.
- Outcome: `os.Create` fails under js/wasm ("not implemented on js"), so the export step errored.
- Time lost: 2
- Category: runtime-bug
- Resolution: the bench driver passes an empty path on js, which makes the app write into `io.Discard`.

### 12:12 Port 8090 busy
- Step: `make web`.
- Command/API: `core/cmd/server -addr :8090`.
- Outcome: another process already listened on 8090 (not started by this app). I did not stop it and checked the web target on :8191 with the same server instead.
- Time lost: 1
- Category: toolchain
- Resolution: none needed for the check, the Makefile still uses 8090 as the brief says.

No `core-issue` entries. No third-party dependency was added beyond what the template generated.

## Summary

- Wall time to first window: about 8 minutes (12:08, from the first screenshot).
- Wall time to feature complete (desktop, tests, bench, web target built and benched in a browser): about 14 minutes (12:14).
- Entries per category: docs-gap 1, api-confusion 2, toolchain 3, build-error 0, runtime-bug 1, framework-limitation 1, core-issue 0.
- Biggest pain points: (1) the silent SIGKILL of a re-copied signed binary; (2) the table's built-in sort arrow versus the required label text; (3) list selection styling forcing the accent text color onto custom rows.
