
## Linux fix (2026-11-09)

The desktop shell used osascript and pbcopy on every OS, so on Linux and Windows export, import and notifications failed. Replaced with cgo-free cross-platform libraries: ncruces/zenity v0.10.15 (dialogs), gen2brain/beeep v0.11.3 (notification), atotto/clipboard v0.1.4 (copy). The `-tags native` macOS shim is unchanged. Our app bug, not a GoTTH limit.
