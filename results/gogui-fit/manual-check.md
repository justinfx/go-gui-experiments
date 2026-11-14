# go-gui second pass (DataGrid) hands-on check (macOS, packaged .app, 2026-11-14)

Done by the author on `apps/gogui-fit/bin/GGUIX-gogui.app`. Reported:

- Scroll performance itself is fast.
- The UI is "pretty wonky":
  - A gap under the horizontal table scrollbar still shows table content through it.
  - The Stock text bars have small vertical lines at random places (block-character fallback, see `adapter.md`).
  - Random rating stars render as boxes (the half-star glyph used in the text fallback is missing from the font).
  - In the cards view, the vertical scrollbar does not move when scrolling with the mouse wheel.
- Overall impression (author): "seems like its a lot of work to try and make it look good".
