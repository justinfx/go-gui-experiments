# go-gui app, second pass: a data layer built for the toolkit

`briefs/gogui.md` in this workspace is the original brief, and `apps/gogui-fit/` holds a finished app built from it. All of `briefs/gogui.md` still applies, except for the changes below.

## What changes

The first app had to reach the data through the prescribed paging layer: section 4.6 of `briefs/gogui.md`, or its frontend or server-rendered equivalent. In this pass you replace that layer with whatever data layer best suits go-gui: the design an experienced go-gui developer would choose for showing a large, slow, remote dataset in a table and a card list. Examples:
- a lazily fetching model in the toolkit's own data-model style
- batched or coalesced change notifications
- a different virtualisation or windowing strategy
- moving work between threads, or between server and client
- reshaping how the view asks for rows

Use the toolkit's idioms, not a general-purpose design.

You may change any app code on the data path and in the view-update logic that the new layer needs: models, views, delegates, cell and card renderers, and update scheduling. Keep everything else as it is.

## What stays fixed

- `core/` must not be modified. All row data must come from the core data service (`Page`, `Get`, `Update` and so on), with its simulated latency left unchanged. In the web-remote route it comes through the remote client.
- **Fetch limits:**
  - no more than 200 rows per service call
  - no more than 10,000 rows held in memory at once
  - never preload or enumerate the whole dataset
- **Bench script:** keep its steps, metric names and the definition of "loaded" exactly as in section 9 of `briefs/gogui.md`. A row that has no data yet must still show as a placeholder and count as not loaded.
- **Unchanged behaviour:**
  - all functional behaviour, the Make targets, tests and the target-code marking rules
  - all tests must still pass
  - the web routes must still work where the first app had them

## Process

1. Read the existing app and find where its data path loses time: extra refreshes, blocking calls, re-layouts, too many or too few fetches, and similar.
2. Design and implement the new data layer. Put it in clearly named files (for example `adapter*.go`, or `data/` / `lib/data/`) so it can be found.
3. Iterate with `make bench`, at most three rounds. Other heavy jobs may share this machine and the screen may be locked, so treat timings as direction only. Prefer signals that don't depend on timing: placeholder steps, the number of view refreshes, and the number of service calls.
4. Run `make test` and confirm the web targets still start.

## Results

Update the existing results files and add `results/gogui-fit/adapter.md` covering:
- **Design:** the design you chose and why it suits go-gui, including the toolkit APIs it relies on.
- **Before and after:** each data-path change compared with the first app, with the reason for it.
- **Files:** which files make up the data layer.
- **Limits:** anything you wanted to do but couldn't within the fixed rules above, and why.
- **Benchmarks:** your own bench observations, labelled as indicative only.

Definition of done: the app builds, all tests pass, `make bench` completes, and `adapter.md` is written. Reply with a short summary: the design, the files changed, the test result, and any problems.

## Screenshots (mandatory)

Capture only the app window, never the full screen or desktop. Use `bench/shot.sh <process-name> <out.png>` (macOS: `screencapture -l <windowid>`; Linux: `import -window <id>`). Plain `screencapture` or `screencapture -x` without `-l` is forbidden: it records the user's other windows. If no window is found, skip the screenshot and say so.

## go-gui specifics (added for this pass)

The first app (`apps/gogui`) did not use go-gui's own table. It built one from a header row of buttons plus `VirtualList`, and the hands-on check found its UX poor: the vertical scrollbar sits past the right edge of a wider horizontal scroll area, content draws over the horizontal scrollbar, the last column does not stretch, and the detail form overflows its pane.

In this pass you MUST build the table on go-gui's `datagrid` package (`github.com/go-gui-org/go-gui/gui/datagrid`, v0.96.0), with a `DataGridDataSource` implementation that fetches from the core service through the same fetch limits. Verify the API in the module source and `examples/data_grid_data_source/` before using it. Find out whether DataGrid can scroll continuously over all 1,000,000 rows with virtualized rows, or only shows numbered pages; prefer continuous scrolling if it exists. Cells DataGrid cannot draw (stock bar, status badge, stars) fall back to text with `CellFormat` colors; record each fallback. Use DataGrid's own sorting hooks where they map to the core's sort. Keep the cards view working (VirtualList is fine there) and fix the detail form so it fits its pane and the table's columns fill the width. Record honestly, as a framework gap, anything the spec needs that DataGrid cannot do.

Paths: app `apps/gogui-fit/` (already a copy of `apps/gogui`), results `results/gogui-fit/`. Keep the binary name `bin/gguix-gogui`. Run `make bench` at most three times; nothing else will be measuring meanwhile. Never use em-dashes; no absolute `/Users/...` paths in files. Do not commit. Screenshots into `results/gogui-fit/screenshots/`.
