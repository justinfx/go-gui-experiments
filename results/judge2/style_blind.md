# Blind style judgement (judge2)

Inputs: `shots_blind/*.png`, made from `results/screenshots/*-desktop.png` with the top 64 px (window title bar) and bottom 80 px (status bar with user and host name) cropped by `sips`. Labels were drawn at random; the mapping (`label_map.json`) was opened only after the scores below were written. Seven shots: five base apps plus the two GoTTH shell variants.

Limit: each current capture shows one state only (dark theme, table view, nothing selected). Light mode, cards view and selection highlight could not be judged, so this pass scores dark table and form polish only. In-app menu bars (F, W) remain visible.

| Label | Score | Notes |
|---|---|---|
| F | 2 | Material-like pills. Form fields have no visible boxes, Rating value clipped at the right edge ("0.("), Stock header clipped, stray "1" in Notes, Qty left aligned, radios stacked vertically and dimmed. Looks unfinished. |
| J | 3 | Clean, gridded, legible table with right-aligned numbers. Strong blue Table toggle. Stock column clipped to "St" by the split, disabled buttons nearly invisible, form labels and fields well aligned. |
| K | 3 | Native-desktop look, crisp table. Form fields small and low contrast, labels greyed, large dead space below the buttons, odd scrollbar corner. Coherent but plain. |
| N | 4 | Modern neutral (shadcn-like) look, segmented control, consistent inputs, radios on one line. SKU values truncate ("SKU-00000...") inconsistently. |
| V | 4 | Same design as N. Status radios wrap to two lines, Rating value missing, and a light scrollbar track shows on the dark table. |
| W | 4 | Same design as N plus a menu bar row. Same radio wrap and SKU truncation. |
| Z | 4 | Same design as V without the scrollbar glitch. Same radio wrap and SKU truncation. |

Mapping, opened after scoring: F=cogent, J=fyne, K=miqt, N=wails, V=gotth-glaze, W=miqt-gotth, Z=gotth. On macOS Qt 6.12; on Linux Qt 6.4 miqt-gotth loses some styling (`results/miqt-gotth/linux.md:36-38`), which these macOS shots do not show.
