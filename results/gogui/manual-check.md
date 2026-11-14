# go-gui hands-on check (macOS, packaged .app, 2026-11-14)

Done by the author on `bin/GGUIX-gogui.app`. Reported:

- Scroll performance was good.
- No usable vertical scrollbar on Table or Cards at the default window size. The vertical scrollbar sits at the far right of the horizontally scrolled content: scrolling the horizontal bar fully right shows a small nub, and the window had to be widened a lot before it worked as a vertical scrollbar.
- A small strip of table content draws over (past) the horizontal scrollbar.
- Detail form fields overflow the pane.
- The last table column does not stretch to fill the width.

Classification: all four are app layout (the agent put the vertical list inside a wider horizontal scroll area and gave the form fixed widths), not missing framework features; recorded as is, consistent with the article's apps.

The rest of the checklist all worked (author: "Yes, all worked"): cards view, sort and search, edit/save/revert/copy row, native save and open dialogs, export progress, cancel and completion notification, Reveal, Light/Dark/System theme, and size/view/theme restore after relaunch.
