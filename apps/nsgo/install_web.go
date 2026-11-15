//go:build js

package main

import (
	"github.com/nsgo-dev/nsgo/desktop"

	"github.com/nsgo-dev/nsgo/core"
)

// installPlatform wires what the browser target has: the web hooks and the
// keyboard shortcuts (there is no menu bar in a page).
func (a *app) installPlatform(cx *core.Scope) {
	a.plat = webHooks(cx)
	desktop.SetTitle(cx, "GGUIX - NSGo")
	cx.OnKey("mod+e", a.startExport)
	cx.OnKey("mod+o", a.beginImport)
	cx.OnKey("mod+shift+c", a.copyRow)
	cx.OnKey("mod+1", func() { a.setView(0) })
	cx.OnKey("mod+2", func() { a.setView(1) })
	cx.OnKey("mod+r", func() { a.rebuildQuery(true) })
	cx.OnKey("mod+i", func() { a.hostOpen = true; a.rDlg.Bump() })
}
