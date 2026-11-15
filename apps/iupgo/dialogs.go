package main

import (
	"fmt"

	"github.com/gen2brain/iup-go/iup"

	"gguix/core/host"
)

// hostRows is the label/value list of the host panel.
func (a *app) hostRows() [][2]string {
	i := a.info
	or := func(s string) string {
		if s == "" {
			return "(unavailable)"
		}
		return s
	}
	rows := [][2]string{
		{"Source", string(i.Source)},
		{"Username", or(i.Username)},
		{"Full name", or(i.FullName)},
		{"UID", or(i.UID)},
		{"Home directory", or(i.HomeDir)},
		{"Hostname", or(i.Hostname)},
		{"OS / Arch", or(i.OS + " / " + i.Arch)},
		{"PID", or(fmt.Sprint(i.PID))},
		{"Config directory", or(i.ConfigDir)},
		{"Locale", or(i.Locale)},
	}
	for _, n := range i.Notes {
		rows = append(rows, [2]string{"Note", n})
	}
	return rows
}

// hostDialog builds the host panel dialog and returns it.
func (a *app) hostDialog() iup.Ihandle {
	u := &a.ui
	var kids []iup.Ihandle
	for _, r := range a.hostRows() {
		l := iup.Label(r[0])
		set(l, "ALIGNMENT", "ARIGHT:ATOP", "FONTSTYLE", "Bold")
		v := iup.Label(r[1])
		set(v, "ALIGNMENT", "ALEFT:ATOP", "WORDWRAP", "NO")
		kids = append(kids, l, v)
	}
	grid := iup.GridBox(kids...)
	set(grid, "NUMDIV", "2", "ORIENTATION", "HORIZONTAL", "GAPLIN", "4", "GAPCOL", "12", "SIZECOL", "-1", "SIZELIN", "-1")
	reveal := iup.Button("Reveal Config Folder")
	u.hostReveal = reveal
	u.hostRevealExp = iup.Button("Reveal Last Export")
	closeBtn := iup.Button("Close")
	var dlg iup.Ihandle
	tip := ""
	if !a.plat.HostActions {
		tip = "Not available in the browser"
		set(reveal, "ACTIVE", "NO", "TIP", tip)
		set(u.hostRevealExp, "ACTIVE", "NO", "TIP", tip)
	} else if a.exp.lastPath == "" {
		set(u.hostRevealExp, "ACTIVE", "NO")
	}
	iup.On(reveal, iup.ActionCB, func(iup.Ihandle) int {
		if a.plat.Reveal != nil {
			if err := a.plat.Reveal(a.info.ConfigDir); err != nil {
				a.showError(err.Error())
			}
		}
		return iup.DEFAULT
	})
	iup.On(u.hostRevealExp, iup.ActionCB, func(iup.Ihandle) int { a.revealLastExport(); return iup.DEFAULT })
	iup.On(closeBtn, iup.ActionCB, func(iup.Ihandle) int {
		if iup.GetAttribute(dlg, "MODAL") == "YES" {
			return iup.CLOSE
		}
		iup.Hide(dlg)
		iup.Destroy(dlg)
		u.hostReveal, u.hostRevealExp = 0, 0
		return iup.DEFAULT
	})
	buttons := iup.Hbox(iup.Fill(), reveal, u.hostRevealExp, closeBtn)
	set(buttons, "GAP", "8")
	box := iup.Vbox(grid, buttons)
	set(box, "MARGIN", "14x12", "GAP", "12")
	dlg = iup.Dialog(box)
	set(dlg, "TITLE", "Host Info", "DIALOGFRAME", "YES")
	iup.SetAttributeHandle(dlg, "PARENTDIALOG", u.dlg)
	iup.SetAttributeHandle(dlg, "DEFAULTESC", closeBtn)
	return dlg
}

func (a *app) revealLastExport() {
	if a.exp.lastPath == "" || a.plat.Reveal == nil {
		return
	}
	if err := a.plat.Reveal(a.exp.lastPath); err != nil {
		a.showError(err.Error())
	}
}

var _ = host.ErrUnavailable
