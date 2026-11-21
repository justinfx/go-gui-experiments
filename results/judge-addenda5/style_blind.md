# Blind style judgement, fifth addenda pass (2026-11-21)

Same method as `results/judge-addenda4/style_blind.md`: fourteen desktop apps (adds `gophics`), inputs `results/screenshots/<id>-desktop.png` cropped 64 px top and 80 px bottom (PIL), fresh random labels, Sonnet judge given only a copy of the images, then two re-checked (E vs P, G vs L: unchanged). Gophics shot in the same state as the others (dark, table view, nothing selected, `GGUIX_SHOT_IDENTITY=demo@workstation`).

| Label | App | Score | Notes (judge's, condensed) |
|---|---|---|---|
| Q | mygoui | 5 | Cleanest: aligned, nothing truncated, radios on one row, stepper. |
| T | nsgo | 4 | Polished and consistent; Stock column clipped at the grid edge. |
| L | mygo | 4 | Clean and modern; SKU truncated. |
| G | wails | 4 | Like L; SKU truncated, form tight. |
| E | miqt-gotth | 3 | SKU truncated, wrapped radios, slider misplaced. |
| J | gotth-glaze | 3 | Nearly the same as E. |
| P | gotth | 3 | Menu bar; same defects as E. |
| V | gogui-fit | 3 | Heavy gray grid looks dated, uneven alignment. |
| W | gophics | 3 | Distinctive terracotta toggle, clean grid; the empty edit panel looks washed out, category arrow misplaced. |
| M | cogent | 2 | Right-side clipping, heavy gray, dated arrows. |
| X | iupgo | 2 | Dated bordered grid, mixed controls. |
| Y | fyne | 2 | Stock clipped, faint buttons, panel overlaps grid. |
| C | miqt | 1 | Mismatched control sizes, tiny fields, bottom cut off. |
| N | gogui | 1 | Most clipping and misalignment. |

Read scores as a ranking within this pass; several comparators moved one point from pass 4 (wails and mygo up; iupgo, gogui and miqt down). Author hands-on: Gophics 4 before fixes, 5 after fix pass 1.

## Pairwise re-check (Gophics against NSGo, row selected)

The full pass shot every app with nothing selected, and Gophics dims its empty form by hand (no disabled state), which the judge read as washed out. A fresh Sonnet judge compared both apps with row 42 selected (dark, table), in two blind pairs with sides swapped (`pair/`, `pair/pair_map.json`):

| Pair | Scores | Result | Reason (condensed) |
|---|---|---|---|
| Gophics (left) vs NSGo (right) | 3 vs 4 | NSGo | Gophics: oversized controls, triangle dropdown and salmon accents read heavy, label baselines off, no row dividers. NSGo: crisp, native feel, row dividers, tidy form; Stock column clipped. |
| NSGo (left) vs Gophics (right) | 4 vs 3 | NSGo | Same reasons, consistent across sides. |

Result: the selected state does not change the score. Gophics stays at 3, one below NSGo.

## Opus check with a control (row selected, dark table plus light cards)

A fresh Opus judge scored three blind apps on an absolute scale (ties allowed), each shown as the dark table and the light cards with row 42 selected (`opus/`). MyGo native UI was included as a control: if it did not come out on top, the judge would be suspect.

| App | Score | Strengths (condensed) | Weaknesses (condensed) |
|---|---|---|---|
| mygoui | 4.5 | Bold selection, even form rows, pinned actions, consistent in both themes | No row separators, Restock has no field, gap above actions |
| nsgo | 4 | Native and modern, accent toggle, row dividers | Stock column clipped, narrow Category, weak dark selection |
| gophics | 3 | Distinct warm identity, pill toggle, consistent rounded inputs | Monospace-looking SKU dashes, crude ▼ dropdown glyphs, label baselines off, uneven spacing, bold Category text, orange accents clash with blue stock bars, last card clipped |

The control held on ranking (mygoui first) but not on absolute score (4.5, not its usual 5), so read the Opus pass as a ranking. It matches the Sonnet passes: Gophics 3, one below NSGo.
