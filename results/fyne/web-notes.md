
## Web sharpness check at 2x (2026-11-09)

`screenshots/fyne-web-dpr1.png` vs `fyne-web-dpr2.png` (page-only Playwright captures via bench/webshot.mjs). At 2x Fyne web text is crisp; the blur seen earlier came from the 1x bench context, not from Fyne. Separate visible defect found: the web build shows a missing-glyph box in place of the sort arrow in the ID header ("ID ?"); the desktop build renders "ID ▲". Cause not yet checked.
