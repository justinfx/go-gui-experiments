# Hands-on check: gotth-fitcap (2026-11-10)

Author, macOS, same machine as the harness, free-spinning mouse wheel. Desktop build launched alone (`apps/gotth-fitcap/bin/GGUIX-gotth.app`, WKWebView).

Author's observations:
- Fast wheel scrolling: only the most minor lag.
- Bug: the table's scrollbar did not render. Dragging in its area (the divider) still worked.
- Dragging there was much more responsive than base MIQT.

Harness numbers this is checked against (`results/gotth-fitcap/metrics.json`, median of 5): table scroll max gap 321 ms, p95 over 300 ms; switch to cards 454 ms.

Conclusion: the measured repeated gaps of about 300 ms were not perceived as stutter in hand use; base MIQT's 250 ms gaps were (`results/miqt/manual-check.md`). The harness's fixed-pace scroll does not separate these two cases. The missing scrollbar is a functional bug in this variant; not checked whether base gotth has it.
