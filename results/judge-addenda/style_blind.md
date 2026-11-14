# Blind style judgement for the addenda (2026-11-14)

Same method as `results/judge2/style_blind.md`. Inputs: `shots_blind/*.png`, made from `results/screenshots/<id>-desktop.png` with the top 64 px (title bar) and bottom 80 px (status bar) cropped by `sips`. All nine desktop apps were scored together in one pass, with fresh random labels, by a judge that saw only the images. The mapping (`label_map.json`) was sealed outside the judge's reach and opened only after scoring. The two MyGo shots were taken on 2026-11-14 in the same state as the originals (dark theme, table view, nothing selected, 1280 x 800 window, `GGUIX_SHOT_IDENTITY=demo@workstation`).

Limit, as in judge2: one state only (dark, table, nothing selected).

| Label | App | Score | judge2 score | Notes (judge's, condensed) |
|---|---|---|---|---|
| B | mygo | 4 | | Clean, rounded controls, segmented toggle; partial SKU truncation; loose top-aligned form labels. |
| E | miqt | 3 | 3 | Full SKUs, readable bars; washed-out labels, small fixed-width fields, heavy focus ring. |
| G | wails | 4 | 4 | Near-identical to B. |
| H | fyne | 3 | 3 | Solid grid; Stock header clipped to "St", faint buttons, Clear cut off. |
| L | miqt-gotth | 3 | 4 | Radios wrap, rating slider offset, bright scrollbar track. |
| P | gotth-glaze | 3 | 4 | Same as L without the scrollbar glitch. |
| R | cogent | 2 | 2 | Clipped Stock header and rating value, borderless fields, stray "1". |
| X | gotth | 4 | 4 | Same design as L and P, subtler scrollbar. |
| Y | mygoui | 5 | | Most consistent: column dividers, full SKUs, aligned labels, single-line radios, primary Save in a footer, Quantity stepper, rating readout. Empty Category dropdown, nothing clipped. |

Calibration: the five base apps and gotth score the same as in judge2. gotth-glaze and miqt-gotth drop from 4 to 3 here; the judge ranked their scrollbar and wrap issues below gotth's. MyGo scores are relative to this pass.
