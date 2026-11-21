# Gophics hands-on notes (macOS, author, 2026-11-21)

## 1. Table
- Aggressive wheel scrolling: smooth, very minor flashes, no placeholders.
- No visible vertical scrollbar; dragging works only on a hidden spot found by clicking around.
- Horizontal scrollbar hidden, appears when dragging in the right spot.
- Jump and sort: work great.
- Bug: rating stars draw over the property form when the column sits just behind the form's left edge (no clipping).

## 2. Cards
- Wheel scrolling smooth, look great. Jump to 500k works. Icons vary correctly.
- Table/Cards switcher keeps its selection.
- Bug: when a card is selected, the card itself stays put but its inner content shifts up slightly.

## 3. Form
- Edit, Revert (before save), Save and validation errors all work. Restock date fine. Light and dark themes work.
- Style note: table header dividers look basic, thick block lines.

## 4. Native features
- Shortcuts do not fire while a text field has focus.
- Export asks for a folder (no save-as path). Progress clears; notification works.
- Reveal, Copy Row, Escape on Host dialog: work.
- Bug: table row selection highlight is so light it is nearly invisible.

## Style
- Author: 4 out of 5 before fixes; possibly 5 once the bugs are fixed. Ask again after the fix pass.

## Recheck after fix pass 1
- All reported issues fixed (author). Export now asks a file name in an in-app dialog, then a native folder; not native, but acceptable.
- Author style: 5 out of 5 ("really clean and appealing").
- Web: table header bleed into the form fixed in fix pass 2 (author, Chrome).
