# Manual perception check (2026-11-10)

Done by the author on the macOS desktop build (`bin/gguix-wails`, release build from the clean re-run), with only this app open. Purpose: check whether the benchmark's scroll gaps (max 140 ms, range 127 to 183 ms over 5 runs) are something a person perceives as a problem.

Input: a mouse with a free-spinning wheel, plus dragging the scrollbar.

- **Gentle wheel scrolling:** smooth ("everything is beautiful").
- **Wheel flicked as fast as it goes:** still scrolls acceptably. An occasional flicker is visible, which matches the measured gaps, but it is not perceived as a problem given how aggressive the input was.
- **Fast scrollbar drag:** the table shows blank rows while far-away pages load. Judged normal, since the data has to be fetched.
- **Controls:** never lag. The UI stays responsive; only the data catches up.
- **Card view:** very acceptable.

Reading: the benchmark measured gaps that could be noticeable to someone, and an occasional flicker is visible on an aggressive fling. To the author they do not stand out. This is one person's perception on one input device, so the measured gaps stay in the report alongside it.
