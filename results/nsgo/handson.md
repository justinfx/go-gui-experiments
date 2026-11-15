# Hands-on (macOS, author, 2026-11-15)

1. Cards at row 500k: aligned and correct.
2. Table: slow scrolling good. Very fast wheel scrolling flashes a lot, and after the wheel stops the view briefly keeps catching up. Header sort arrow works.
3. Detail form: Save worked. Revert works (enables on edit, before Save; the earlier report expected it to undo a Save).
4. Export CSV and Reveal Last Export worked. Theme switching and dialogs fine. Escape does not close the Host dialog on macOS (it does on Linux X11 and sway).
5. Style 4 of 5: close to Fyne and Wails in some areas, better than Cogent. The Table/Cards switch has little active-state distinction; sliders and the Save button look good.

## Re-check after fix pass

- Escape closes the Host dialog. Table scrolling better: a little acceptable flashing under intense wheel scrolling, stops instantly.
- Identical card tiles were the restored Category sort, not a bug (spec-level UX issue, see iupgo).
