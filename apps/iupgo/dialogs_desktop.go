//go:build !js

package main

import (
	"fmt"
	"runtime"

	"github.com/gen2brain/iup-go/iup"
)

func (a *app) openAbout() {
	msg := fmt.Sprintf("GGUIX - IUP-Go\nIUP %s (iup-go main cbcdf09)\n%s\nBuildMarker %s",
		iup.Version(), runtime.Version(), BuildMarker)
	d := iup.MessageDlg()
	set(d, "TITLE", "About GGUIX", "DIALOGTYPE", "INFORMATION", "BUTTONS", "OK", "VALUE", msg)
	set(d, "PARENTDIALOG", "")
	iup.SetAttributeHandle(d, "PARENTDIALOG", a.ui.dlg)
	iup.Popup(d, iup.CENTERPARENT, iup.CENTERPARENT)
	iup.Destroy(d)
}

func (a *app) openHost() {
	dlg := a.hostDialog()
	iup.Popup(dlg, iup.CENTERPARENT, iup.CENTERPARENT)
	iup.Destroy(dlg)
	a.ui.hostReveal, a.ui.hostRevealExp = 0, 0
}
