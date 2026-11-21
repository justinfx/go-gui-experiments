package ui

import (
	"github.com/doug/gophics/geom"
	"github.com/doug/gophics/layout"
	"github.com/doug/gophics/theme"
	"github.com/doug/gophics/widget"

	"gguix/core"
)

func (s *App) cardsView(th theme.Theme) widget.Widget {
	total := s.Pager.Total()
	if total == 0 {
		return widget.Center(widget.Text{Value: "No items match.", Size: th.Type.Body, Color: th.Muted})
	}
	list := widget.LazyList{
		Count:           max(total, 0),
		EstimatedExtent: CardExtent,
		Controller:      s.cardsCtl,
		Build: func(i int) widget.Widget {
			s.syncVisible()
			it, ok := s.Pager.Row(i)
			return widget.Padding{Insets: geom.Insets{Left: 8, Right: 8, Bottom: CardGap},
				Child: Card{Item: it, Loaded: ok, Selected: ok && it.ID == s.SelectedID,
					OnTap: s.act(func() { s.SelectRow(i) })}}
		},
	}
	return widget.Padding{Insets: geom.Insets{Top: 8},
		Child: widget.LayoutObserver{OnLayout: s.onViewport, Child: list}}
}

// Card is the custom card item for the cards view (section 5.10).
type Card struct {
	Item     core.Item
	Loaded   bool
	Selected bool
	OnTap    func()
}

func (c Card) Build(ctx widget.Ctx) widget.Widget {
	th := theme.Of(ctx)
	pal := core.Colors(th.Dark)
	border, bw := th.Border, float32(1)
	if c.Selected {
		border, bw = th.Primary, 2
	}
	var content widget.Widget
	if !c.Loaded {
		content = widget.Align{X: 0, Y: 0.5, Child: widget.Text{Value: "Loading…", Size: th.Type.Label, Color: th.Muted}}
	} else {
		it := c.Item
		chips := []widget.Widget{StatusBadge{Status: it.Status, Pal: pal, Size: th.Type.Caption}}
		if it.Featured {
			chips = append(chips, gapW(6), FeaturedChip{Pal: pal, Size: th.Type.Caption})
		}
		mid := widget.Column(
			widget.Text{Value: it.Name, Font: theme.FontBold, Size: th.Type.Label, Color: th.Text, Ellipsis: true},
			muted(th, it.SKU+" · "+it.Category.String(), th.Type.Caption),
			muted(th, core.NotesPreview(it.Notes), th.Type.Caption),
			gapH(2),
			widget.Row(chips...),
		)
		mid.CrossAlign = layout.CrossStart
		mid.MainAlign = layout.MainSpaceBetween
		right := widget.Column(
			widget.Text{Value: core.FormatPrice(it.PriceCents), Font: theme.FontBold, Size: th.Type.Heading, Color: th.Text},
			row(Stars{Rating: it.Rating, Pal: pal}, gapW(4), widget.Text{Value: core.FormatRating(it.Rating), Size: th.Type.Caption, Color: th.Text}),
			row(widget.Text{Value: core.StockLabel(it.Quantity), Size: th.Type.Caption, Color: th.Muted}, gapW(6), StockBar{Qty: it.Quantity, Pal: pal}),
		)
		right.CrossAlign = layout.CrossEnd
		right.MainAlign = layout.MainSpaceBetween
		r := widget.Row(
			widget.Align{X: 0, Y: 0.5, Child: CategoryTile{Cat: it.Category, Pal: pal}},
			gapW(12),
			widget.Expand(mid),
			gapW(12),
			widget.Sized{W: 160, Child: right},
		)
		r.CrossAlign = layout.CrossStretch
		content = r
	}
	return widget.Interactive{
		Gestures: widget.Gestures{OnTap: c.OnTap},
		Child: widget.Decorated{Color: th.Surface, Radius: th.Radius, BorderColor: border, BorderWidth: bw,
			Child: widget.Sized{H: CardH, Child: widget.Padding{All: 12, Child: content}}},
	}
}
