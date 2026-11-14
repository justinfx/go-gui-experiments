# Blind style judgement, second addenda pass (2026-11-14)

Same method as `results/judge-addenda/style_blind.md`. Inputs: `shots_blind/*.png`, from `results/screenshots/<id>-desktop.png` with the top 64 px and bottom 80 px cropped by `sips`. All ten desktop apps scored together, fresh random labels, judge saw only the images. `label_map.json` was kept outside the judge's reach until scoring was done. The go-gui shot was taken in the same state as the others (dark, table view, nothing selected, 1280 x 800 requested, `GGUIX_SHOT_IDENTITY=demo@workstation`).

| Label | App | Score | Previous pass | Notes (judge's, condensed) |
|---|---|---|---|---|
| A | gogui | 2 | | Clean table, but the form overflows the right edge (fields run off, "Disconti" and the date Clear button clipped); header size differs from rows; faint checkbox. |
| B | gotth-glaze | 4 | 3 | Polished; SKU truncated, radios wrap, slider above label. |
| E | mygoui | 5 | 5 | Most consistent: aligned labels, pinned actions, no clipping. |
| K | fyne | 3 | 3 | Stock header clipped to "St", faint disabled buttons. |
| Q | miqt | 2 | 3 | Small hard-bordered fields, dimmed form, stray focus ring. |
| R | mygo | 4 | 4 | Clean; SKU truncated, uneven form spacing. |
| T | cogent | 2 | 2 | Clipped Stock header and rating value, stray "1", borderless fields. |
| U | wails | 4 | 4 | Near-identical to R. |
| V | miqt-gotth | 3 | 3 | Bright scrollbar track, wrap issues. |
| X | gotth | 4 | 4 | Polished; same issues as B. |

Calibration: Fyne, Cogent, Wails, gotth, MyGo and MyGo native UI score the same as before; gotth-glaze and miqt each moved by one. go-gui's score comes from the app's form layout (the agent's fixed-width form inside the split pane), not a missing framework feature; it is recorded as is, as the article's apps were, with no style fix pass.
