
## wails-fitcap test fix (2026-11-09)

One vitest case still expected the uncapped single batch. Updated to the capped behaviour (one page per call, at most 2 in flight); 36 of 36 pass.
