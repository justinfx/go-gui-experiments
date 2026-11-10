package main

import (
	"context"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	gc "gguix/core"

	"gguix/core/host"
)

// hostActionsOn reports whether the reveal actions run on the user's machine.
func (a *App) hostActionsOn() bool { return a.hostInfo.Source == host.SourceNative }

const notInBrowser = "Not available in the browser"

// showHost opens the Host panel (brief section 5.11).
func (a *App) showHost() { a.hostBody().RunDialog(a.body) }

// hostBody builds the Host panel dialog body.
func (a *App) hostBody() *core.Body {
	d := core.NewBody("Host Info")
	grid := core.NewFrame(d)
	grid.Styler(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = 2
		s.Gap.Set(units.Dp(16), units.Dp(4))
	})
	for _, r := range a.hostInfo.Rows() {
		l := core.NewText(grid).SetType(core.TextBodyMedium).SetText(r.Label)
		l.Styler(func(s *styles.Style) { s.Font.Weight = rich.Bold })
		core.NewText(grid).SetType(core.TextBodyMedium).SetText(r.Value)
	}
	for _, n := range a.hostInfo.Notes {
		core.NewText(d).SetType(core.TextSupporting).SetText(n)
	}
	d.AddBottomBar(func(bar *core.Frame) {
		cfg := core.NewButton(bar).SetText("Reveal Config Folder").SetType(core.ButtonOutlined)
		cfg.OnClick(func(e events.Event) { a.revealConfig() })
		last := core.NewButton(bar).SetText("Reveal Last Export").SetType(core.ButtonOutlined)
		last.OnClick(func(e events.Event) { a.revealLastExport() })
		on := a.hostActionsOn()
		cfg.SetEnabled(on && a.hostInfo.ConfigDir != "")
		last.SetEnabled(on && a.exp.lastPath != "")
		if !on {
			cfg.SetTooltip(notInBrowser)
			last.SetTooltip(notInBrowser)
		}
		d.AddOK(bar).SetText("Close")
	})
	return d
}

func (a *App) revealConfig() {
	if a.hostActionsOn() && a.hostInfo.ConfigDir != "" {
		if err := host.RevealInFileManager(a.hostInfo.ConfigDir); err != nil {
			core.ErrorDialog(a.body, err, "Reveal")
		}
	}
}

func (a *App) revealLastExport() {
	if a.hostActionsOn() && a.exp.lastPath != "" {
		if err := host.RevealInFileManager(a.exp.lastPath); err != nil {
			core.ErrorDialog(a.body, err, "Reveal")
		}
	}
}

// afterSave refreshes cached data after an edit and re-finds the row.
func (a *App) afterSave(saved gcItem) {
	a.pager.Invalidate()
	id := saved.ID
	q := a.query
	go func() {
		row, ok, _ := findRow(q, a.svc, id)
		a.async(func() {
			if a.selID != id {
				return
			}
			if ok {
				a.selRow = row
			} else {
				a.selRow = -1
			}
			a.syncViewSelection()
		})
	}()
}

type gcItem = gc.Item

func findRow(q gc.Query, svc gc.DataService, id int64) (int, bool, error) {
	return gc.FindRow(context.Background(), svc, q, id, 5000)
}
