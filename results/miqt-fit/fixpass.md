
## Linux fix (2026-11-09)

Tray notification passed a nil `*QIcon` to `ShowMessage3`, which MIQT dereferences, so no notification was sent. Now passes `qt.NewQIcon()`. Found by the Linux runtime check; our app bug, not MIQT.
