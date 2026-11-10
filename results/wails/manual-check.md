# Manual perception check (2026-11-10)

Done by the author right after the wails-fitcap check (`results/wails-fitcap/manual-check.md`), same inputs (free-spinning mouse wheel, scrollbar drag), on the release build from the clean re-run, with only this app open.

Result: "Same feel" as wails-fitcap. Gentle scrolling is smooth; an aggressive flick shows at most an occasional flicker that is not perceived as a problem; a fast scrollbar drag shows blank rows while pages load, judged normal; the controls stay responsive.

Benchmark for comparison: scroll max frame gap 91 ms median (68 to 108), 64 blank-row steps (wails-fitcap: 140 ms, about 1).
