package main

import (
	"math"
	"strconv"

	"github.com/go-gui-org/go-gui/gui"

	"gguix/core"
)

// Custom-rendered pieces shared by the table cells and the cards, in the
// section 4.5.1 colors for the current theme. The stock bar and the
// badges are ordinary go-gui containers (fill color plus radius); the
// stars are a DrawCanvas with filled polygons.

const (
	barW, barH float32 = 80, 8
	starPx     float32 = 12
	starGap    float32 = 1
)

// stockSpec is what the stock bar shows: the fill fraction and colors.
type stockSpec struct {
	frac        float32
	track, fill string
}

func (a *app) stockSpec(qty int32) stockSpec {
	return stockSpec{float32(core.StockFraction(qty)), a.pal.StockTrack, a.pal.StockFillFor(qty)}
}

// stockBar draws the 80 x 8 track with its fill. row >= 0 gives the
// fill an ID (stock-fill-<row>) so tests can measure it.
func (a *app) stockBar(qty int32, row int) gui.View {
	s := a.stockSpec(qty)
	content := []gui.View{}
	if s.frac > 0 {
		id := ""
		if row >= 0 {
			id = "stockfill" + strconv.Itoa(row)
		}
		content = append(content, gui.Row(gui.ContainerCfg{
			ID: id, Width: barW * s.frac, Height: barH, Sizing: gui.FixedFixed, Radius: gui.RadiusPx(barH / 2),
			Color: hexColor(s.fill), SizeBorder: gui.NoBorder, Padding: gui.NoPadding,
		}))
	}
	return gui.Row(gui.ContainerCfg{
		Width: barW, Height: barH, Sizing: gui.FixedFixed, Radius: gui.RadiusPx(barH / 2),
		Color: hexColor(s.track), SizeBorder: gui.NoBorder, Padding: gui.NoPadding, Content: content,
	})
}

// pill is a fully rounded chip with text.
func (a *app) pill(w *gui.Window, text, bg, fg string) gui.View {
	ts := w.Theme().TextStyleBodySmall
	ts.Color = hexColor(fg)
	return gui.Row(gui.ContainerCfg{
		Padding: gui.PadVH(1, 8), Radius: gui.RadiusPx(999), Color: hexColor(bg), SizeBorder: gui.NoBorder,
		VAlign: gui.VAlignMiddle, Sizing: gui.FitFit,
		Content: []gui.View{gui.Text(gui.TextCfg{Text: text, TextStyle: ts})},
	})
}

func (a *app) badge(w *gui.Window, s core.Status) gui.View {
	b := a.pal.Badge(s)
	return a.pill(w, s.String(), b.Background, b.Text)
}

// starTris is a 5-point star around (cx, cy) as 10 triangles fanned from
// the center, so no concave polygon has to be tessellated.
func starTris(cx, cy, r float32) [][]float32 {
	pts := make([][2]float32, 10)
	for k := 0; k < 10; k++ {
		rr := r
		if k%2 == 1 {
			rr = r * 0.45
		}
		ang := -math.Pi/2 + float64(k)*math.Pi/5
		pts[k] = [2]float32{cx + rr*float32(math.Cos(ang)), cy + rr*float32(math.Sin(ang))}
	}
	out := make([][]float32, 0, 10)
	for k := 0; k < 10; k++ {
		p, q := pts[k], pts[(k+1)%10]
		out = append(out, []float32{cx, cy, p[0], p[1], q[0], q[1]})
	}
	return out
}

// clipLeft keeps the part of polygon poly with x <= xm.
func clipLeft(poly []float32, xm float32) []float32 {
	var out []float32
	n := len(poly) / 2
	for i := 0; i < n; i++ {
		x0, y0 := poly[2*i], poly[2*i+1]
		x1, y1 := poly[2*((i+1)%n)], poly[2*((i+1)%n)+1]
		in0, in1 := x0 <= xm, x1 <= xm
		if in0 {
			out = append(out, x0, y0)
		}
		if in0 != in1 {
			t := (xm - x0) / (x1 - x0)
			out = append(out, xm, y0+t*(y1-y0))
		}
	}
	return out
}

// starsWithText is the five stars and the FormatRating text.
func (a *app) starsWithText(r float32, row int, kind string) gui.View {
	full, half, empty := core.Stars(r)
	on, off := hexColor(a.pal.StarFilled), hexColor(a.pal.StarEmpty)
	ver := uint64(full*100+half*10+empty) | uint64(len(a.pal.StarFilled))<<16
	if a.isDark() {
		ver |= 1 << 20
	}
	id := ""
	if row >= 0 {
		id = "stars" + kind + strconv.Itoa(row)
	}
	canvas := gui.DrawCanvas(gui.DrawCanvasCfg{
		ID: id, Width: 5*(starPx+starGap) - starGap, Height: starPx, Sizing: gui.FixedFixed, Version: ver,
		OnDraw: func(dc *gui.DrawContext) {
			for k := 0; k < 5; k++ {
				cx := float32(k)*(starPx+starGap) + starPx/2
				tris := starTris(cx, starPx/2+0.5, starPx/2)
				for _, t := range tris {
					c := off
					if k < full {
						c = on
					}
					dc.FilledPolygon(t, c)
					if k == full && half == 1 {
						if l := clipLeft(t, cx); len(l) >= 6 {
							dc.FilledPolygon(l, on)
						}
					}
				}
			}
		},
	})
	return gui.Row(gui.ContainerCfg{
		Padding: gui.NoPadding, Spacing: gui.SpacingSmall, SizeBorder: gui.NoBorder, VAlign: gui.VAlignMiddle,
		Content: []gui.View{canvas, gui.Text(gui.TextCfg{Text: core.FormatRating(r)})},
	})
}
