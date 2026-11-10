# Blind style judgement: out-of-box style appeal

Criterion: polished, modern, consistent in both themes (5 = polished on every surface, 1 = dated or inconsistent). Based only on screenshots in shots_blind/ (a sample of table and cards views per label, both themes).

| Label | Score | Row 1 selected visible | Legibility | Alignment/spacing | Dark mode | Desktop vs web |
|---|---|---|---|---|---|---|
| P | 2 | yes (strong blue) | good in table; form labels near-invisible in light | dense, consistent grid lines | decent | inconsistent (web dark has light buttons) |
| Q | 3 | no | good | native-ish but cramped form fields, clipped Name text | good, native | desktop only |
| R | 4 | faint | good | clean, minor label/field misalignment in form | very good | consistent |
| S | 3 | yes (clear) | good | Qty left-aligned, clipped rating and Stock column, stray "1" in notes | good | consistent |
| T | 4 | faint | good | clean, generous spacing, Status radios wrap | very good | consistent |

**P (2).** Strong, unmistakable selection highlight and a tidy gridded table, but theming is broken in places. In light mode the toolbar inputs and Save/Revert stay dark and the form labels are almost invisible; in web dark the buttons and inputs render light. Spacing is fine, but the cross-theme inconsistency makes it feel unpolished.

**Q (3).** Looks like a native macOS app: clean, legible, and both themes are coherent. However, the selected row is not visibly highlighted, the form fields are small and uneven with the Name value clipped ("eluxe Silicone Lamp"), and the detail pane has a lot of dead space. No web build to compare.

**R (4).** Modern shadcn-like look with a neutral palette, pill segmented control and consistent inputs; dark mode is crisp and matches light. Selection on row 1 is only a faint grey tint. SKU values truncate inconsistently in the table and form labels sit slightly above their fields. Desktop and web are nearly identical.

**S (3).** Material-style theme with a clear selected row and card, applied consistently across desktop and web and both themes. Polish suffers from layout flaws: the Stock column header and rating value "3.7" are clipped, Qty is left-aligned, the Notes field shows a stray line number "1", and the vertical status radios make the form uneven. Looks modern but not finished.

**T (4).** Very similar to R and just as clean: modern neutral styling, good spacing, excellent dark mode, consistent desktop and web, and a nice outlined selected card. Selected table row is faint. Minor issues are truncated SKUs and the Status radios wrapping onto a second line.
