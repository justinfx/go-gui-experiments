package main

// Custom-rendered cells shared by the table and the cards view (brief
// sections 5.3 and 5.10). Colors come from gguix/core's palette (4.5.1).

import (
	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/matcolor"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"

	gc "gguix/core"
)

// isDark reports whether the current color scheme is dark.
func isDark() bool { return matcolor.SchemeIsDark }

func palette() gc.Palette { return gc.Colors(isDark()) }

// selectable makes a cell or card part pass clicks on to the row selection.
func selectable(w core.Widget, fn func()) {
	wb := w.AsWidget()
	wb.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Clickable, abilities.Selectable)
	})
	wb.OnSelect(func(e events.Event) {
		e.SetHandled()
		fn()
	})
}

// stockBar is the custom stock bar: a rounded 80x8 track with a fill.
type stockBar struct {
	core.Frame
	Qty    int32
	Loaded bool
}

const stockBarW = 80

func (s *stockBar) Init() {
	s.Frame.Init()
	s.Styler(func(st *styles.Style) {
		st.Min.Set(units.Dp(stockBarW), units.Dp(8))
		st.Max.Set(units.Dp(stockBarW), units.Dp(8))
		st.Grow.Set(0, 0)
		st.Align.Self = styles.Center
		st.Padding.Zero()
		st.Gap.Zero()
		st.Border.Radius = sides.NewValues(units.Dp(4))
		st.Background = colors.Uniform(gc.HexColor(palette().StockTrack))
		st.SetAbilities(false, abilities.Hoverable)
	})
	tree.AddChild(s, func(w *core.Frame) {
		w.Styler(func(st *styles.Style) {
			f := float32(gc.StockFraction(s.Qty))
			w.SetProperty("fraction", gc.StockFraction(s.Qty))
			st.Min.Set(units.Dp(stockBarW*f), units.Dp(8))
			st.Max.Set(units.Dp(stockBarW*f), units.Dp(8))
			st.Grow.Set(0, 0)
			st.Padding.Zero()
			st.Border.Radius = sides.NewValues(units.Dp(4))
			st.Background = colors.Uniform(gc.HexColor(palette().StockFillFor(s.Qty)))
		})
	})
}

// Fraction returns the fill fraction currently shown.
func (s *stockBar) Fraction() float64 { return gc.StockFraction(s.Qty) }

// FillColor returns the fill color currently used.
func (s *stockBar) FillColor() color.RGBA { return gc.HexColor(palette().StockFillFor(s.Qty)) }

// set updates the bar for a quantity.
func (s *stockBar) set(qty int32) { s.Qty = qty }

// pill is a rounded badge or chip with palette colors.
type pill struct {
	core.Text
	// Colors picks the badge colors from the current palette.
	Colors func(p gc.Palette) gc.BadgeColors
	// Hidden removes the pill from the layout.
	Hidden bool
}

func (p *pill) Init() {
	p.Text.Init()
	p.SetType(core.TextLabelMedium)
	p.Styler(func(s *styles.Style) {
		s.SetAbilities(false, abilities.Selectable, abilities.DoubleClickable, abilities.TripleClickable, abilities.LongPressable, abilities.Slideable)
		s.Padding.Set(units.Dp(1), units.Dp(8))
		s.Border.Radius = sides.NewValues(units.Dp(500))
		s.Align.Self = styles.Center
		s.Grow.Set(0, 0)
		s.SetTextWrap(false)
		if p.Hidden {
			s.Display = styles.DisplayNone
		}
		if p.Colors != nil {
			c := p.Colors(palette())
			s.Background = colors.Uniform(gc.HexColor(c.Background))
			s.Color = colors.Uniform(gc.HexColor(c.Text))
		}
	})
}

// BadgeColors returns the colors in use now (for tests).
func (p *pill) BadgeColors() (bg, fg color.RGBA) {
	return colors.ToUniform(p.Styles.Background), colors.ToUniform(p.Styles.Color)
}

func statusPill(w *pill, st gc.Status) {
	w.Colors = func(pal gc.Palette) gc.BadgeColors { return pal.Badge(st) }
	w.SetText(st.String())
}

func featuredPill(w *pill) {
	w.Colors = func(pal gc.Palette) gc.BadgeColors { return pal.Featured }
	w.SetText("Featured")
}

// starsW draws five 12 dp stars for a rating (full, half, empty).
type starsW struct {
	core.Canvas
	Full, Half, Empty int
}

const starSize = 12

func (w *starsW) Init() {
	w.Canvas.Init()
	w.Draw = w.draw
	w.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(starSize*5), units.Dp(starSize))
		s.Max.Set(units.Dp(starSize*5), units.Dp(starSize))
		s.Grow.Set(0, 0)
		s.Align.Self = styles.Center
		s.SetAbilities(false, abilities.Hoverable)
	})
}

func (w *starsW) set(r float32) {
	w.Full, w.Half, w.Empty = gc.Stars(r)
}

// starPoints returns the outline of a star in the unit square.
func starPoints() []math32.Vector2 {
	const outer, inner = 0.5, 0.2
	pts := make([]math32.Vector2, 0, 10)
	for i := 0; i < 10; i++ {
		r := float32(outer)
		if i%2 == 1 {
			r = inner
		}
		a := float64(i)*math32.Pi/5 - math32.Pi/2
		pts = append(pts, math32.Vec2(0.5+r*float32(math32.Cos(float32(a))), 0.5+r*float32(math32.Sin(float32(a)))))
	}
	return pts
}

// clipLeft clips a polygon to x <= cut (Sutherland-Hodgman, one edge).
func clipLeft(in []math32.Vector2, cut float32) []math32.Vector2 {
	var out []math32.Vector2
	for i := range in {
		a, b := in[i], in[(i+1)%len(in)]
		ain, bin := a.X <= cut, b.X <= cut
		if ain {
			out = append(out, a)
		}
		if ain != bin {
			t := (cut - a.X) / (b.X - a.X)
			out = append(out, math32.Vec2(cut, a.Y+t*(b.Y-a.Y)))
		}
	}
	return out
}

func (w *starsW) draw(pc *paint.Painter) {
	pal := palette()
	filled := gc.HexColor(pal.StarFilled)
	empty := gc.HexColor(pal.StarEmpty)
	base := starPoints()
	shape := func(i int, pts []math32.Vector2, c color.RGBA) {
		if len(pts) < 3 {
			return
		}
		for k, p := range pts {
			x, y := (float32(i)+p.X)/5, p.Y
			if k == 0 {
				pc.MoveTo(x, y)
			} else {
				pc.LineTo(x, y)
			}
		}
		pc.Close()
		pc.Fill.Color = colors.Uniform(c)
		pc.Stroke.Color = nil
		pc.Draw()
	}
	for i := 0; i < 5; i++ {
		switch {
		case i < w.Full:
			shape(i, base, filled)
		case i < w.Full+w.Half:
			shape(i, base, empty)
			shape(i, clipLeft(base, 0.5), filled)
		default:
			shape(i, base, empty)
		}
	}
}

// ratingRow is a stars widget followed by the rating text.
type ratingRow struct {
	core.Frame
	Stars *starsW
	Text  *core.Text
}

func (r *ratingRow) Init() {
	r.Frame.Init()
	r.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Gap.X.Dp(4)
		s.Align.Items = styles.Center
		s.Grow.Set(0, 0)
		s.Padding.Zero()
	})
	r.Stars = tree.New[starsW](r)
	r.Text = core.NewText(r).SetType(core.TextBodyMedium)
	r.Text.Styler(func(s *styles.Style) {
		s.SetAbilities(false, abilities.Selectable, abilities.DoubleClickable, abilities.TripleClickable, abilities.LongPressable, abilities.Slideable)
	})
}

func (r *ratingRow) set(rating float32, ok bool) {
	r.Stars.SetState(!ok, states.Invisible)
	if !ok {
		r.Text.SetText("\u2026")
		return
	}
	r.Stars.set(rating)
	r.Text.SetText(gc.FormatRating(rating))
}

// stockRow is a stock bar with a text to its right (table) or left (card).
type stockRow struct {
	core.Frame
	Bar   *stockBar
	Text  *core.Text
	label func(qty int32) string
}

func (r *stockRow) Init() {
	r.Frame.Init()
	r.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Gap.X.Dp(6)
		s.Align.Items = styles.Center
		s.Grow.Set(0, 0)
		s.Padding.Zero()
	})
}

// build adds the children; it is called once after the widget is made.
func (r *stockRow) build(left bool, label func(int32) string) {
	r.label = label
	if left {
		r.Text = core.NewText(r).SetType(core.TextBodyMedium)
		r.Bar = tree.New[stockBar](r)
	} else {
		r.Bar = tree.New[stockBar](r)
		r.Text = core.NewText(r).SetType(core.TextBodyMedium)
	}
	r.Text.Styler(func(s *styles.Style) {
		s.SetAbilities(false, abilities.Selectable, abilities.DoubleClickable, abilities.TripleClickable, abilities.LongPressable, abilities.Slideable)
	})
}

func (r *stockRow) set(qty int32, ok bool) {
	r.Bar.SetState(!ok, states.Invisible)
	if !ok {
		r.Bar.Qty = 0
		r.Text.SetText("\u2026")
		return
	}
	r.Bar.set(qty)
	if r.label != nil {
		r.Text.SetText(r.label(qty))
	}
}
