package ui

import (
	"fmt"

	"github.com/doug/gophics/geom"
	"github.com/doug/gophics/layout"
	"github.com/doug/gophics/paint"
	"github.com/doug/gophics/theme"
	"github.com/doug/gophics/widget"

	"gguix/core"
)

var statusOptions = []string{"All statuses", "Active", "Backordered", "Discontinued"}
var themeOptions = []string{"System", "Light", "Dark"}
var themeValues = []string{core.ThemeSystem, core.ThemeLight, core.ThemeDark}

func row(children ...widget.Widget) widget.Flex {
	r := widget.Row(children...)
	r.CrossAlign = layout.CrossCenter
	return r
}

func gapW(w float32) widget.Widget { return widget.Sized{W: w} }
func gapH(h float32) widget.Widget { return widget.Sized{H: h} }

// disabled renders child dimmed; its callbacks must already be no-ops.
func disabled(off bool, child widget.Widget) widget.Widget {
	if !off {
		return child
	}
	return widget.Opacity{Alpha: 0.45, Child: child}
}

func (s *App) act(fn func()) func() { return func() { s.SetState(fn) } }

// CategoriesLabel is the category filter button text.
func (s *App) CategoriesLabel() string {
	if n := len(s.Query.Categories); n > 0 {
		return fmt.Sprintf("Categories (%d)", n)
	}
	return "Categories"
}

func (s *App) toolbar(th theme.Theme) widget.Widget {
	statusSel := 0
	if s.Query.Status != nil {
		statusSel = int(*s.Query.Status) + 1
	}
	themeSel := 0
	for i, v := range themeValues {
		if v == s.Theme {
			themeSel = i
		}
	}
	viewSel := 0
	if s.View == core.ViewCards {
		viewSel = 1
	}
	var exportTap func()
	if !s.Exporting {
		exportTap = s.act(s.StartExport)
	}

	search := widget.WithKey{Key: s.searchGen, Child: widget.Sized{W: 280, Child: Field{OnKeyPreview: s.fieldKeyPreview,
		Value:       s.Search,
		Placeholder: "Search name or SKU",
		Autofocus:   s.searchGen > 0,
		OnChange:    func(v string) { s.SetState(func() { s.SetSearchText(v) }) },
	}}}
	cats := widget.LayoutObserver{
		OnLayout: func(r geom.Rect) { s.catsAnchor = geom.Pt{X: r.Min.X, Y: r.Max.Y + 4} },
		Child:    theme.Button{Label: s.CategoriesLabel(), OnTap: s.act(s.ShowCategories)},
	}
	status := widget.Sized{W: 160, Child: Select{
		Options: statusOptions, Selected: statusSel,
		OnChange: func(i int) {
			s.SetState(func() {
				if i <= 0 {
					s.SetStatusFilter(nil)
				} else {
					st := core.Status(i - 1)
					s.SetStatusFilter(&st)
				}
			})
		},
	}}
	goTo := widget.Sized{W: 110, Child: Field{OnKeyPreview: s.fieldKeyPreview,
		Value: s.GoToText, Placeholder: "Go to row",
		OnChange: func(v string) { s.SetState(func() { s.GoToText = v }) },
		OnSubmit: func(string) { s.SetState(s.GoToSubmit) },
	}}
	views := widget.Sized{W: 170, Child: theme.Segmented{
		Options: []string{"Table", "Cards"}, Selected: viewSel,
		OnChange: func(i int) {
			s.SetState(func() {
				if i == 1 {
					s.SetView(core.ViewCards)
				} else {
					s.SetView(core.ViewTable)
				}
			})
		},
	}}
	themes := widget.Sized{W: 120, Child: Select{
		Options: themeOptions, Selected: themeSel,
		OnChange: func(i int) { s.SetState(func() { s.SetTheme(themeValues[i]) }) },
	}}
	return widget.Padding{Insets: geom.InsetsSymmetric(12, 8), Child: row(
		search, gapW(8), cats, gapW(8), status, gapW(8), goTo, gapW(4),
		theme.Button{Label: "Go", OnTap: s.act(s.GoToSubmit)}, gapW(8), views,
		widget.Spacer(),
		disabled(s.Exporting, theme.Button{Label: "Export CSV", OnTap: exportTap}), gapW(8), themes,
	)}
}

func (s *App) mainArea(th theme.Theme) widget.Widget {
	return widget.LayoutBuilder{Build: func(cs layout.Constraints) widget.Widget {
		w := cs.Max.W
		if w <= 0 || w > 1e6 {
			w = 1280
		}
		left := w * s.split
		var list widget.Widget
		if s.View == core.ViewCards {
			list = s.cardsView(th)
		} else {
			list = s.tableView(th)
		}
		divider := widget.Interactive{
			Gestures: widget.Gestures{
				Cursor:   resizeCursor,
				DragAxis: widget.DragAny,
				OnDrag: func(_, d geom.Pt) {
					s.SetState(func() {
						s.split = min(max(s.split+d.X/w, 0.3), 0.8)
					})
				},
			},
			Child: widget.Fill{Color: th.Border, Child: widget.Sized{W: 5}},
		}
		r := widget.Row(
			widget.Sized{W: left, Child: list},
			divider,
			widget.Expand(s.formView(th)),
		)
		r.CrossAlign = layout.CrossStretch
		return r
	}}
}

func (s *App) statusBar(th theme.Theme) widget.Widget {
	total := s.Pager.Total()
	left := "Loading…"
	if total >= 0 {
		left = core.FormatCount(total) + " items"
		if total > 0 && s.Last >= s.First {
			left += fmt.Sprintf(" · rows %d-%d", s.First+1, s.Last+1)
		}
	}
	st := s.Pager.Stats()
	left += fmt.Sprintf(" · last fetch %d ms · %d pages cached", st.LastFetch.Milliseconds(), st.CachedPages)
	if s.Message != "" {
		left += "   " + s.Message
	}
	items := []widget.Widget{widget.Text{Value: left, Size: th.Type.Label, Color: th.Text}, widget.Spacer()}
	if s.Exporting {
		frac := float32(0)
		if s.ExportTotal > 0 {
			frac = float32(s.ExportDone) / float32(s.ExportTotal)
		}
		items = append(items,
			widget.Sized{W: 200, Child: theme.Progress{Value: frac, Label: "Exporting"}}, gapW(8),
			widget.Text{Value: fmt.Sprintf("%d%%", int(frac*100)), Size: th.Type.Label, Color: th.Text}, gapW(8),
			theme.Button{Label: "Cancel", OnTap: s.act(s.CancelExport)}, gapW(16),
		)
	}
	items = append(items,
		widget.Text{Value: s.Host.Identity(), Size: th.Type.Label, Color: th.Muted}, gapW(8),
		theme.Button{Label: "Host", OnTap: s.act(s.ShowHost)},
	)
	return widget.Padding{Insets: geom.InsetsSymmetric(12, 4), Child: row(items...)}
}

// StatusText returns the left status bar text (tests).
func (s *App) StatusText() string {
	total := s.Pager.Total()
	if total < 0 {
		return ""
	}
	return core.FormatCount(total) + " items"
}

func muted(th theme.Theme, s string, size float32) widget.Widget {
	return widget.Text{Value: s, Size: size, Color: th.Muted, Ellipsis: true}
}

func colorOf(hex string) paint.Color {
	c := core.HexColor(hex)
	return paint.Color{R: float32(c.R) / 255, G: float32(c.G) / 255, B: float32(c.B) / 255, A: float32(c.A) / 255}
}
