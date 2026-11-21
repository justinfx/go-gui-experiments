# Gophics preflight (2026-11-21)

Lessons from earlier runs, checked before the build agent starts:

- [x] Brief generated from the go-gui brief; no judging terms, no other app's code.
- [x] Prediction kept outside the agent workspace (`results/gophics-prediction.md`).
- [x] Agent workspace `~/src/gguix-ws-gophics` holds only BRIEF.md, core/, linux/, bench/shot.sh and the Linux template; grep found no prediction, judge, rubric, email, hostname or home path.
- [x] Linux image already has mesa-vulkan-drivers (lavapipe).
- [ ] Before measuring: delete stray binaries, nothing else running, temp GGUIX_PREFS_DIR.
- [ ] Before screenshots: GGUIX_SHOT_IDENTITY set, app window only (bench/shot.sh, bench/shot-web.mjs).
- [ ] Before committing: grep for home path, email, hostname; no binaries.
- [ ] No GUI launches while the author does the hands-on check.
