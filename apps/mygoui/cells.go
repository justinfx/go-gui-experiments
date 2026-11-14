package main

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"

	"gguix/core"
)

// Custom-rendered pieces shared by the table cells and the cards. They are
// built from ordinary elements (boxes with a background and a radius) and a
// painted star shape, in the section 4.5.1 colors for the current theme.

const (
	barW, barH = 80, 8
	starSize   = 12
)

func hex(s string) ui.Color { return ui.Hex(s) }

// stockBar draws the 80 x 8 track and fill for qty.
func (a *app) stockBar(c *ui.Context, qty int32) {
	frac := float32(core.StockFraction(qty))
	pal := a.pal
	ui.Box(c).Size(barW, barH).Radius(barH / 2).Background(hex(pal.StockTrack)).Shrink(0).
		Role(ui.RoleImage).Label(fmt.Sprintf("stock track")).Children(func() {
		ui.Box(c).Size(barW*frac, barH).Radius(barH / 2).Background(hex(pal.StockFillFor(qty))).
			Role(ui.RoleImage).Label("stock fill")
	})
}

// stockCell is the table's Stock cell: the bar and the quantity.
func (a *app) stockCell(c *ui.Context, qty int32) {
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		a.stockBar(c, qty)
		ui.Text(c, fmt.Sprint(qty)).SingleLine()
	})
}

// badge is the status pill.
func (a *app) badge(c *ui.Context, s core.Status) {
	b := a.pal.Badge(s)
	ui.Box(c).Row().AlignItems(ui.Center).Padding(1, 8).Radius(999).Background(hex(b.Background)).Shrink(0).
		Role(ui.RoleImage).Label("status badge").Children(func() {
		ui.Text(c, s.String()).SingleLine().TextColor(hex(b.Text)).FontSize(c.Theme().FontSize - 1)
	})
}

// chip is the Featured chip.
func (a *app) chip(c *ui.Context) {
	b := a.pal.Featured
	ui.Box(c).Row().AlignItems(ui.Center).Padding(1, 8).Radius(999).Background(hex(b.Background)).Shrink(0).
		Role(ui.RoleImage).Label("featured chip").Children(func() {
		ui.Text(c, "Featured").SingleLine().TextColor(hex(b.Text)).FontSize(c.Theme().FontSize - 1)
	})
}

var starPath = func() *ui.Path {
	var p ui.Path
	// A five point star in the unit square.
	for i := 0; i < 10; i++ {
		r := 0.5
		if i%2 == 1 {
			r = 0.2
		}
		ang := -math.Pi/2 + float64(i)*math.Pi/5
		x, y := float32(0.5+r*math.Cos(ang)), float32(0.5+r*math.Sin(ang))
		if i == 0 {
			p.MoveTo(x, y)
		} else {
			p.LineTo(x, y)
		}
	}
	p.Close()
	return &p
}()

func starAt(r ui.Rect) *ui.Path {
	var p ui.Path
	for i := 0; i < 10; i++ {
		rad := 0.5
		if i%2 == 1 {
			rad = 0.21
		}
		ang := -math.Pi/2 + float64(i)*math.Pi/5
		x := r.X + r.W*float32(0.5+rad*math.Cos(ang))
		y := r.Y + r.H*float32(0.52+rad*math.Sin(ang))
		if i == 0 {
			p.MoveTo(x, y)
		} else {
			p.LineTo(x, y)
		}
	}
	p.Close()
	return &p
}

// star draws one 12 px star: kind is "full", "half" (left half filled) or
// "empty".
func (a *app) star(c *ui.Context, kind string) {
	on, off := hex(a.pal.StarFilled), hex(a.pal.StarEmpty)
	ui.Box(c).Size(starSize, starSize).Shrink(0).Role(ui.RoleImage).Label("star " + kind).Draw(func(p *ui.Painter, r ui.Rect) {
		path := starAt(r)
		switch kind {
		case "full":
			p.FillPath(path, on)
		case "half":
			p.FillPath(path, off)
			p.Clip(ui.Rect{X: r.X, Y: r.Y, W: r.W / 2, H: r.H}, 0, func() { p.FillPath(path, on) })
		default:
			p.FillPath(path, off)
		}
	})
}

// stars is the rating cell: five stars, a space and the rating text.
func (a *app) stars(c *ui.Context, rating float32) {
	full, half, empty := core.Stars(rating)
	ui.Row(c).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Gap(1).AlignItems(ui.Center).Children(func() {
			for i := 0; i < full; i++ {
				a.star(c, "full")
			}
			for i := 0; i < half; i++ {
				a.star(c, "half")
			}
			for i := 0; i < empty; i++ {
				a.star(c, "empty")
			}
		})
		ui.Text(c, " "+core.FormatRating(rating)).SingleLine()
	})
}

// tile is the 48 x 48 category tile with the initial in white.
func (a *app) tile(c *ui.Context, cat core.Category) {
	ui.Box(c).Size(48, 48).Radius(8).Background(hex(a.pal.Tile(cat))).Shrink(0).Center().
		Role(ui.RoleImage).Label("category tile").Children(func() {
		ui.Text(c, core.CategoryInitial(cat)).Bold().FontSize(20).TextColor(hex(a.pal.TileText)).TextAlign(ui.Center)
	})
}
