# Blind style judgement, fourth addenda pass (2026-11-15)

Same method as `results/judge-addenda3/style_blind.md`: thirteen desktop apps (adds `nsgo`, `iupgo`), inputs `results/screenshots/<id>-desktop.png` cropped 64 px top and 80 px bottom, fresh random labels, Sonnet judge given only a copy of the images, one image viewed and noted at a time, then two re-checked (B, X: unchanged). NSGo and IUP-Go shot in the same state as the others (dark, table view, nothing selected, `GGUIX_SHOT_IDENTITY=demo@workstation`).

| Label | App | Score | Notes (judge's, condensed) |
|---|---|---|---|
| R | mygoui | 5 | Cleanest: no truncation, Stock visible, radios on one line, pinned action footer. |
| B | nsgo | 4 | Polished rounded controls, consistent form; last table column clipped at the pane edge. |
| G | miqt-gotth | 3 | SKU truncated, bright scrollbar track, wrapped radios, slider off its label. |
| J | gotth | 3 | Same family as G; SKU truncated, wrapped radios, slider misaligned. |
| L | wails | 3 | Radios on one line, but labels sit high, cramped ID/SKU rows, SKU truncated. |
| M | iupgo | 3 | Tidy right-aligned form, but heavy dated grid borders, Stock off-screen, mismatched toolbar heights, boxy date picker. |
| Q | gotth-glaze | 3 | Near-identical to G and J. |
| T | mygo | 3 | Same family as L. |
| X | gogui-fit | 3 | Contained stacked form and pager, but heavy grid, blocky bars, tiny dim labels. |
| A | fyne | 2 | Status column clipped, big gaps in form, near-invisible disabled buttons. |
| E | miqt | 2 | Tiny mixed-style inputs, narrow Name field, misaligned header dividers. |
| N | gogui | 2 | Detail form overflows the right edge. |
| H | cogent | 1 | Mixed toolbar styles, clipped header and rating, stray "1" in Notes. |

Calibration: top and bottom held from pass 3 (mygoui first, cogent last, gogui 2); gogui-fit dropped from 4 to 3. Read scores as a ranking within this pass. Author hands-on (`results/<id>/handson.md`): NSGo 4, IUP-Go 2 (the judge saw only the dark table; the author also used the light theme and the MatrixEx look).

## Pairwise re-check (IUP-Go against Fyne and MIQT)

The full pass gave IUP-Go 3 against 2 for Fyne and MIQT, while noting a dated grid and an off-screen Stock column for IUP-Go, the same kind of defect it marked Fyne down for. A fresh Sonnet judge compared the cropped images in two blind pairs (IUP-Go right in one, left in the other), told to weigh a dated, basic look as seriously as layout defects:

| Pair | Scores | Result | Reason (condensed) |
|---|---|---|---|
| Fyne (left) vs IUP-Go (right) | 3 vs 3 | tie | Fyne: modern but Status clipped, weak form affordances. IUP-Go: complete form, but heavy bordered grid, boxed date picker, hidden trailing column read as dated. |
| IUP-Go (left) vs MIQT (right) | 3 vs 3 | tie | IUP-Go: consistent form, dated grid, Stock off-screen. MIQT: all columns and modern bars, but dated, mismatched form inputs. |

Use for the addendum: on a static screenshot IUP-Go ties Fyne and MIQT, not above them. The author's hands-on score (2) also covers the light theme and use.
