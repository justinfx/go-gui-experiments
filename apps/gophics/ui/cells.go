package ui

import (
	"math"

	"github.com/doug/gophics/geom"
	"github.com/doug/gophics/paint"
	"github.com/doug/gophics/theme"
	"github.com/doug/gophics/widget"

	"gguix/core"
)

// Custom-rendered elements (sections 5.3 and 5.10). Each is a stateless
// widget composed of framework primitives (Decorated, Text) or a
// widget.Canvas for the star shapes, with the exact section 4.5.1 colors.

// StockBar is an 80x8 rounded track with a fill of StockFraction(qty).
type StockBar struct {
	Qty int32
	Pal core.Palette
}

// Fraction is the fill fraction drawn.
func (b StockBar) Fraction() float64 { return core.StockFraction(b.Qty) }

// FillHex is the fill color used.
func (b StockBar) FillHex() string { return b.Pal.StockFillFor(b.Qty) }

func (b StockBar) Build(widget.Ctx) widget.Widget {
	track, fill := colorOf(b.Pal.StockTrack), colorOf(b.FillHex())
	frac := float32(b.Fraction())
	return widget.Canvas{W: 80, H: 8, Draw: func(c paint.Canvas, sz geom.Size) {
		c.FillRRect(geom.RectXYWH(0, 0, 80, 8), 4, track)
		if frac > 0 {
			c.FillRRect(geom.RectXYWH(0, 0, 80*frac, 8), 4, fill)
		}
	}}
}

// StatusBadge is a fully rounded pill with the status text.
type StatusBadge struct {
	Status core.Status
	Pal    core.Palette
	Size   float32
}

// Colors returns the badge background and text hex colors.
func (b StatusBadge) Colors() core.BadgeColors { return b.Pal.Badge(b.Status) }

// Text is the badge label.
func (b StatusBadge) Text() string { return b.Status.String() }

func (b StatusBadge) Build(widget.Ctx) widget.Widget {
	bc := b.Colors()
	return pill(b.Text(), bc, b.Size)
}

func pill(label string, bc core.BadgeColors, size float32) widget.Widget {
	if size == 0 {
		size = 11
	}
	return widget.Decorated{Color: colorOf(bc.Background), Radius: 999,
		Child: widget.Padding{Insets: geom.InsetsSymmetric(8, 2),
			Child: widget.Text{Value: label, Size: size, Color: colorOf(bc.Text)}}}
}

// FeaturedChip is the "Featured" pill in the featured colors.
type FeaturedChip struct {
	Pal  core.Palette
	Size float32
}

func (f FeaturedChip) Build(widget.Ctx) widget.Widget {
	return pill("Featured", f.Pal.Featured, f.Size)
}

// Stars draws five 12 px stars: Stars(rating) full, half, empty.
type Stars struct {
	Rating float32
	Pal    core.Palette
}

// Counts returns (full, half, empty).
func (s Stars) Counts() (int, int, int) { return core.Stars(s.Rating) }

// starPaths holds one retained star per slot (12 px apart plus 1 px gap), so
// no transform layer is needed.
var starPaths, halfStarPaths = starSet(false), starSet(true)

// starSet builds the five stars. With left set, each star is cut to its left
// half by clamping x to the star's center line: the star is symmetric about
// that line, so the clamped polygon is exactly the left half. This avoids a
// nested clip, which the GPU path fill did not honor (fixpass issue 1).
func starSet(left bool) (out [5]*paint.Path) {
	for i := range out {
		p := paint.NewPath()
		cx, cy, ro, ri := float32(i)*13+6, float32(6.3), float32(6), float32(2.5)
		for k := 0; k < 10; k++ {
			r := ro
			if k%2 == 1 {
				r = ri
			}
			a := -math.Pi/2 + float64(k)*math.Pi/5
			pt := geom.Pt{X: cx + r*float32(math.Cos(a)), Y: cy + r*float32(math.Sin(a))}
			if left {
				pt.X = min(pt.X, cx)
			}
			if k == 0 {
				p.MoveTo(pt)
			} else {
				p.LineTo(pt)
			}
		}
		out[i] = p.Close()
	}
	return out
}

func (s Stars) Build(widget.Ctx) widget.Widget {
	full, half, _ := s.Counts()
	on, off := colorOf(s.Pal.StarFilled), colorOf(s.Pal.StarEmpty)
	return widget.Canvas{W: 5*12 + 4*1, H: 12, Draw: func(c paint.Canvas, sz geom.Size) {
		for i, p := range starPaths {
			switch {
			case i < full:
				c.FillPath(p, on)
			case i == full && half > 0:
				c.FillPath(p, off)
				c.FillPath(halfStarPaths[i], on)
			default:
				c.FillPath(p, off)
			}
		}
	}}
}

// CategoryTile is the 48x48 rounded square with the white initial.
type CategoryTile struct {
	Cat core.Category
	Pal core.Palette
}

func (t CategoryTile) Build(widget.Ctx) widget.Widget {
	return widget.Decorated{Color: colorOf(t.Pal.Tile(t.Cat)), Radius: 8,
		Child: widget.Sized{W: 48, H: 48, Child: widget.Center(widget.Text{
			Value: core.CategoryInitial(t.Cat), Font: theme.FontBold, Size: 20, Color: colorOf(t.Pal.TileText)})}}
}
