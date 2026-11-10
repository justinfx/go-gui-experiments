
## Linux fix (2026-11-09)

Quit menu item (Ctrl+Q) deadlocked: it called `system.TheApp.QuitReq()` while the render lock was held. Now `go system.TheApp.QuitReq()`, matching Cogent's own Quit (`core/menu.go:298`). Found by the Linux runtime check; the bug was in our app, not Cogent.
