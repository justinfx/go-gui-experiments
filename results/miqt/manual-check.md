# Hands-on check: base MIQT (2026-11-10)

Author, macOS, same machine as the harness, free-spinning mouse wheel and scrollbar. App launched alone (`apps/miqt/bin/GGUIX-miqt.app`).

Author's observations:
- Gentle wheel scrolling: fine.
- Fast wheel spin: really lagging.
- Fast scrollbar drag: really lagging.
- Switching to card mode: a short lag of about a second.
- Card view: the same fast-scroll problems as the table.
- Unlike Wails, the controls themselves lagged when dragged (the scrollbar did not keep up), not only the data.

Harness numbers this is checked against (`results/miqt/metrics.json`, median of 5): table scroll max gap 251 ms, p95 229 ms; cards scroll max gap 428 ms; switch to cards 928 ms.

Conclusion: the repeated gaps of 250 ms or more are perceived as a problem in normal fast scrolling. The "visible stutter" band holds for base MIQT. miqt-fitcap (same Qt, different data layer) was not checked by hand.
