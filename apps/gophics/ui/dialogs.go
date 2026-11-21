package ui

import (
	"runtime"
	"runtime/debug"
	"slices"

	"github.com/doug/gophics/geom"
	"github.com/doug/gophics/layout"
	"github.com/doug/gophics/theme"
	"github.com/doug/gophics/widget"

	"gguix/core"
)

// BuildMarker is set by main (marker.go lives in package main).
var BuildMarker = "?"

func (s *App) showDialog(title string, content widget.Widget) {
	if s.dialogDismiss != nil {
		s.dialogDismiss()
	}
	s.DialogTitle = title
	var dismiss func()
	dismiss = theme.ShowDialog(s.ctx, content)
	s.dialogDismiss = func() {
		dismiss()
		s.DialogTitle = ""
		s.dialogDismiss = nil
	}
}

// CloseDialog dismisses the open dialog, if any.
func (s *App) CloseDialog() {
	if s.dialogDismiss != nil {
		s.dialogDismiss()
	}
}

func buttonsRow(children ...widget.Widget) widget.Widget {
	all := append([]widget.Widget{widget.Spacer()}, children...)
	return widget.Row(all...)
}

// confirmDiscard asks "Discard unsaved changes?" and runs then on Discard.
func (s *App) confirmDiscard(then func()) {
	s.pendingDiscard = then
	s.showDialog("Discard unsaved changes?", widget.Sized{W: 320, Child: widget.Column(
		theme.Heading("Discard unsaved changes?"),
		gapH(16),
		buttonsRow(
			theme.Button{Label: "Cancel", OnTap: s.act(s.CancelDiscard)}, gapW(8),
			theme.Button{Label: "Discard", Primary: true, OnTap: s.act(s.ConfirmDiscard)},
		),
	)})
}

// ConfirmDiscard is the dialog's Discard button.
func (s *App) ConfirmDiscard() {
	then := s.pendingDiscard
	s.pendingDiscard = nil
	s.CloseDialog()
	if then != nil {
		s.Form = formFrom(s.Item) // drop the edits first so the guard passes
		then()
	}
}

// CancelDiscard is the dialog's Cancel button.
func (s *App) CancelDiscard() {
	s.pendingDiscard = nil
	s.CloseDialog()
}

// ShowMessage shows a simple error/info dialog.
func (s *App) ShowMessage(title, body string) {
	s.MessageBody = body
	s.showDialog(title, widget.Sized{W: 360, Child: widget.Column(
		theme.Heading(title),
		gapH(8),
		theme.Body(body),
		gapH(16),
		buttonsRow(theme.Button{Label: "OK", Primary: true, OnTap: s.act(s.CloseDialog)}),
	)})
}

func frameworkVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, d := range bi.Deps {
			if d.Path == "github.com/doug/gophics" {
				return "Gophics " + d.Version
			}
		}
	}
	return "Gophics (unknown version)"
}

// AboutLines is the About dialog content.
func AboutLines() []string {
	return []string{Title, frameworkVersion(), runtime.Version(), "BuildMarker " + BuildMarker}
}

// ShowAbout opens the About dialog.
func (s *App) ShowAbout() {
	lines := []widget.Widget{}
	for i, l := range AboutLines() {
		if i == 0 {
			lines = append(lines, theme.Heading(l), gapH(6))
		} else {
			lines = append(lines, theme.Label(l), gapH(4))
		}
	}
	lines = append(lines, gapH(12), buttonsRow(theme.Button{Label: "OK", Primary: true, OnTap: s.act(s.CloseDialog)}))
	col := widget.Column(lines...)
	col.CrossAlign = layout.CrossStart
	s.showDialog("About GGUIX", widget.Sized{W: 360, Child: col})
}

// HostRows is the host panel content (label, value), notes appended.
func (s *App) HostRows() [][2]string {
	out := [][2]string{}
	for _, r := range s.Host.Rows() {
		out = append(out, [2]string{r.Label, r.Value})
	}
	for _, n := range s.Host.Notes {
		out = append(out, [2]string{"Note", n})
	}
	return out
}

// ShowHost opens the Host Info dialog.
func (s *App) ShowHost() {
	th := s.themeFor(s.ctx)
	rows := []widget.Widget{theme.Heading("Host Info"), gapH(10)}
	for _, r := range s.HostRows() {
		rows = append(rows, widget.Padding{Insets: geom.Insets{Bottom: 4}, Child: widget.Row(
			widget.Sized{W: 140, Child: widget.Text{Value: r[0], Size: th.Type.Label, Color: th.Muted}},
			widget.Expand(widget.Text{Value: r[1], Size: th.Type.Label, Color: th.Text, Wrap: true}),
		)})
	}
	avail := s.Host.Available()
	var revealCfg, revealExp func()
	if avail {
		revealCfg = s.act(s.RevealConfig)
		if s.LastExport != "" {
			revealExp = s.act(s.RevealLastExport)
		}
	}
	btns := buttonsRow(
		disabled(!avail, theme.Button{Label: "Reveal Config Folder", OnTap: revealCfg}), gapW(8),
		disabled(!avail || s.LastExport == "", theme.Button{Label: "Reveal Last Export", OnTap: revealExp}), gapW(8),
		theme.Button{Label: "Close", Primary: true, OnTap: s.act(s.CloseDialog)},
	)
	if !avail {
		btns = theme.Tooltip{Message: "Not available in the browser", Child: btns}
		rows = append(rows, gapH(6), widget.Text{Value: "Reveal actions: Not available in the browser", Size: th.Type.Caption, Color: th.Muted})
	}
	rows = append(rows, gapH(14), btns)
	col := widget.Column(rows...)
	col.CrossAlign = layout.CrossStretch
	s.showDialog("Host Info", widget.Sized{W: 560, Child: col})
}

// ShowCategories opens the category filter popup under its button.
func (s *App) ShowCategories() {
	ov, ok := s.ctx.Of[widget.Overlay]()
	if !ok {
		return
	}
	var tok widget.OverlayToken
	closed := false
	closeFn := func() {
		if !closed {
			closed = true
			tok.Dismiss()
		}
	}
	th := s.themeFor(s.ctx)
	tok = ov.ShowFrom(s.ctx, widget.Provide[theme.Theme]{Value: th, Child: widget.Stack{Children: []widget.Widget{
		widget.Interactive{Gestures: widget.Gestures{OnTap: closeFn}, Child: widget.Fill{}},
		widget.Padding{Insets: geom.Insets{Left: s.catsAnchor.X, Top: s.catsAnchor.Y},
			Child: widget.Align{X: 0, Y: 0, Child: catPopup{app: s}}},
	}}})
}

type catPopup struct{ app *App }

func (c catPopup) CreateState() widget.State { return &catPopupState{} }

type catPopupState struct {
	widget.StateBase[catPopup]
}

func (p *catPopupState) Build(ctx widget.Ctx) widget.Widget {
	s := p.W().app
	th := theme.Of(ctx)
	items := []widget.Widget{}
	for _, c := range core.AllCategories {
		c := c
		items = append(items, theme.Checkbox{Checked: slices.Contains(s.Query.Categories, c), Label: c.String(),
			OnChange: func(bool) {
				s.SetState(func() { s.ToggleCategory(c) })
				p.SetState(func() {})
			}}, gapH(2))
	}
	items = append(items, gapH(6), theme.Button{Label: "Clear", OnTap: func() {
		s.SetState(s.ClearCategories)
		p.SetState(func() {})
	}})
	col := widget.Column(items...)
	col.CrossAlign = layout.CrossStart
	return widget.Decorated{Color: th.Elevated, Radius: th.Radius, BorderColor: th.Border, BorderWidth: 1,
		Child: widget.Padding{All: 12, Child: widget.Sized{W: 180, Child: col}}}
}
