# {{NAME}} app, second pass: a data layer built for the toolkit

`BRIEF.md` in this workspace is the original brief, and `apps/{{ID}}/` holds a finished app built from it. All of `BRIEF.md` still applies, except for the changes below.

## What changes

The first app had to reach the data through the prescribed paging layer: section 4.6 of `BRIEF.md`, or its frontend or server-rendered equivalent. In this pass you replace that layer with whatever data layer best suits {{NAME}}: the design an experienced {{NAME}} developer would choose for showing a large, slow, remote dataset in a table and a card list. Examples:
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
- **Bench script:** keep its steps, metric names and the definition of "loaded" exactly as in section 9 of `BRIEF.md`. A row that has no data yet must still show as a placeholder and count as not loaded.
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

Update the existing results files and add `results/{{ID}}/adapter.md` covering:
- **Design:** the design you chose and why it suits {{NAME}}, including the toolkit APIs it relies on.
- **Before and after:** each data-path change compared with the first app, with the reason for it.
- **Files:** which files make up the data layer.
- **Limits:** anything you wanted to do but couldn't within the fixed rules above, and why.
- **Benchmarks:** your own bench observations, labelled as indicative only.

Definition of done: the app builds, all tests pass, `make bench` completes, and `adapter.md` is written. Reply with a short summary: the design, the files changed, the test result, and any problems.

## Screenshots (mandatory)

Capture only the app window, never the full screen or desktop. Use `bench/shot.sh <process-name> <out.png>` (macOS: `screencapture -l <windowid>`; Linux: `import -window <id>`). Plain `screencapture` or `screencapture -x` without `-l` is forbidden: it records the user's other windows. If no window is found, skip the screenshot and say so.
