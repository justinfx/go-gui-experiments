# Gophics: prediction before building (2026-11-21)

Written before any code, from the site, the repo README and the earlier
addenda results. To be compared with the measured results afterwards.

| Metric (desktop, macOS) | Prediction | Basis |
|---|---|---|
| Ready (ms) | 350 to 500 | WebGPU device setup on top of a self-drawn start like go-gui (277) and MyGo native UI (236) |
| Jump to row 500k (ms) | 80 to 110 | every app with a virtualized table landed at 69 to 146 |
| Table scroll, max frame gap (ms) | 15 to 30 | GPU self-drawn apps: 11 to 40; a young renderer, so mid-range |
| Switch to cards (ms) | 100 to 150 | no documented card or list widget; hand-built like go-gui (117) |
| Sort (ms) | 250 to 280 | core-bound; every app but MIQT and IUP-Go sat at 241 to 280 |
| Peak memory (MB) | 250 to 400 | GPU buffers and glyph atlases, between go-gui (279) and NSGo (369) |
| Cold build (s) | 20 to 35 | a large pure-Go WebGPU stack compiled from source |
| Blind style (1 to 5) | 2 to 3 | stock theme, agent-built from the spec |

| Metric (browser, Chrome) | Prediction | Basis |
|---|---|---|
| Download, gzip (MB) | 3.5 to 4.0 | the site's own 3.4 to 3.9 MB figure |
| Ready (ms) | 900 to 1,400 | WASM plus WebGPU adapter setup; the WASM builds ranged 720 to 1,453 |
| Table scroll, max frame gap (ms) | 50 to 120 | between MyGo native UI (67) and go-gui (151) |

Other calls:

- **cgo:** `CGO_ENABLED=0` builds on macOS and Linux. GPU driver libraries
  still load at run time.
- **Linux container:** starts only after adding Mesa lavapipe (Vulkan in
  software). About 8 of 12 checks pass; native menus, notifications and
  Reveal are the likely failures.
- **Hands-on:** more framework-limit write-ups than fixes. Likely gaps: text
  input polish (IME, selection), file dialogs, notifications, menus.
- **Spec completeness:** some native features stubbed or left out, written
  up as framework limits.
- **Overall:** fast on the desktop, close to go-gui, but not MyGo native UI;
  the weakest on native features and maturity of any addendum.

## Style ranking (added 2026-11-21, still before any code)

Blind pass, one score per app, compared with the closest alternatives:

| App | Measured blind | Predicted place for Gophics |
|---|---|---|
| MyGo native UI | 5 | above Gophics |
| NSGo | 4 | above Gophics |
| Gophics | (predicted 3) | |
| go-gui | 2 | at or below Gophics |
| Fyne | 2 | at or below Gophics |
| Cogent | 1 | below Gophics |

- **Blind:** 3 out of 5, so third of the six self-drawn and pure-Go apps.
  A modern flat default theme should beat go-gui and Fyne's stock look, but
  the hand-built cards and table cells keep it under MyGo native UI and NSGo.
- **Pairwise against go-gui and Fyne:** a narrow win or a tie.
- **Author's hands-on score:** 3 out of 5, below NSGo (4) and above IUP-Go (2).

## Scorecard (measured 2026-11-21, after fix pass 2)

| Metric | Predicted | Measured | Call |
|---|---|---|---|
| Ready (ms) | 350 to 500 | 399 | hit |
| Jump (ms) | 80 to 110 | 88 | hit |
| Table scroll max gap (ms) | 15 to 30 | 17.8 | hit |
| Switch to cards (ms) | 100 to 150 | 68 | miss (faster) |
| Sort (ms) | 250 to 280 | 284 | near miss |
| Peak memory (MB) | 250 to 400 | 332 | hit |
| Cold build (s) | 20 to 35 | 8.8 | miss (much faster) |
| Web download, gzip (MB) | 3.5 to 4.0 | 4.5 | miss (larger) |
| Web ready (ms) | 900 to 1,400 | 681 | miss (faster) |
| Web table max gap (ms) | 50 to 120 | 68 | hit |
| cgo-free desktop and Linux | yes | yes | hit |
| Linux needs lavapipe added | yes | already in image | miss |
| Author style | 3 | 4, then 5 after fixes | miss |
| Blind style | 3 | 3 | hit |
