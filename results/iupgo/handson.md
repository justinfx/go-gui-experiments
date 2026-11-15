# Hands-on (macOS, author, 2026-11-15)

## MatrixEx table (default)

1. Table: very smooth scrolling, impressive. At aggressive wheel speed it tops out at a maximum scroll rate but keeps performing well. Scrollbars and cell content look good.
2. Jump, sort, search: work fine.
3. Cards: broken. A few pages down, all cards sit in "Loading..." state. No color variation in the card icons.
4. Detail form: Save works. Revert works (enables on edit, before Save; the earlier report expected it to undo a Save).
5. Export: the busy (spinner) cursor did not stop at 100% until the notification was clicked (or a timer ran out). Light theme: black background in the Restock date widget; low-contrast widget borders.
6. Style 2 of 5, slightly worse than Cogent. A very basic look, light-theme bugs, basic table styling. Unclear whether better styling is possible, but the agent did not get close to the other apps from the same spec.

## Native table (`TABLE=native`)

1. Table: slow scrolling fine. Fast wheel scrolling flashes a little and lags slightly. Dragging the scrollbar lags and shows placeholders.
2. Cell content: stock bars, status pills and stars look nice.
3. Look: much better than the MatrixEx grid. The horizontal and vertical scrollbars float over the table, so content shows past the outer edge of each scrollbar (possibly the native table's style).
4. Sort and search: about the same as MatrixEx.

## Re-check after fix pass

- Cards load at row 500k. Identical tiles were the restored Category sort (same in NSGo); spec-level UX issue: cards follow the table's sort with no sort indication in the cards view.
- Clicking an item makes the Table/Cards selector lose its selected state.
- Detail form shows cut-off "Select an item to edi" after clicking a row.
- Mouse wheel scrolling fine.
- Light theme Restock date field: fixed.
- Export busy cursor clears at 100%: fixed.
- View selector keeps its state after clicking a row: fixed.
- Clipped "Select an item to edit" hint: fixed.
