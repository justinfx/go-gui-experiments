package main

import (
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"gguix/core"
)

// Custom-rendered pieces shared by the table cells and the cards. They are
// the only places where colors other than the theme's own are used, and the
// colors come from core.Colors (brief section 4.5.1).

const (
	stockBarW = 80
	stockBarH = 8
	starSize  = 12
	starGap   = 2
)

func hex(s string) color.Color { return core.HexColor(s) }

// ---------------------------------------------------------------- stock bar

type stockBar struct {
	widget.BaseWidget
	qty int32
}

func newStockBar() *stockBar {
	b := &stockBar{}
	b.ExtendBaseWidget(b)
	return b
}

func (b *stockBar) SetQty(q int32) {
	b.qty = q
	b.Refresh()
}

// Fraction is the filled fraction of the bar.
func (b *stockBar) Fraction() float64 { return core.StockFraction(b.qty) }

// FillHex is the fill color in use.
func (b *stockBar) FillHex() string { return core.Colors(isDark()).StockFillFor(b.qty) }

// TrackHex is the track color in use.
func (b *stockBar) TrackHex() string { return core.Colors(isDark()).StockTrack }

func (b *stockBar) CreateRenderer() fyne.WidgetRenderer {
	r := &stockBarRenderer{b: b, track: canvas.NewRectangle(color.Transparent), fill: canvas.NewRectangle(color.Transparent)}
	r.apply()
	return r
}

type stockBarRenderer struct {
	b           *stockBar
	track, fill *canvas.Rectangle
}

func (r *stockBarRenderer) apply() {
	p := core.Colors(isDark())
	r.track.FillColor = hex(p.StockTrack)
	r.fill.FillColor = hex(p.StockFillFor(r.b.qty))
	r.track.CornerRadius = stockBarH / 2
}

func (r *stockBarRenderer) Layout(s fyne.Size) {
	y := (s.Height - stockBarH) / 2
	r.track.Move(fyne.NewPos(0, y))
	r.track.Resize(fyne.NewSize(stockBarW, stockBarH))
	w := float32(r.b.Fraction()) * stockBarW
	r.fill.Move(fyne.NewPos(0, y))
	r.fill.Resize(fyne.NewSize(w, stockBarH))
	r.fill.CornerRadius = minf(stockBarH/2, w/2)
	r.fill.Hidden = w <= 0
}

func (r *stockBarRenderer) MinSize() fyne.Size { return fyne.NewSize(stockBarW, stockBarH) }

func (r *stockBarRenderer) Refresh() {
	r.apply()
	r.Layout(r.b.Size())
	r.track.Refresh()
	r.fill.Refresh()
}

func (r *stockBarRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.track, r.fill}
}

func (r *stockBarRenderer) Destroy() {}

// ---------------------------------------------------------------- pill

type pillKind int

const (
	pillStatus pillKind = iota
	pillFeatured
)

// pill is a fully rounded badge or chip.
type pill struct {
	widget.BaseWidget
	kind   pillKind
	status core.Status
	text   string
	small  bool
}

func newPill(kind pillKind, small bool) *pill {
	p := &pill{kind: kind, small: small}
	p.ExtendBaseWidget(p)
	return p
}

func (p *pill) SetStatus(s core.Status) {
	p.kind, p.status, p.text = pillStatus, s, s.String()
	p.Refresh()
}

func (p *pill) SetFeatured() {
	p.kind, p.text = pillFeatured, "Featured"
	p.Refresh()
}

// Text is the label of the pill.
func (p *pill) Text() string { return p.text }

// Colors are the background and text colors in use.
func (p *pill) Colors() core.BadgeColors {
	pal := core.Colors(isDark())
	if p.kind == pillFeatured {
		return pal.Featured
	}
	return pal.Badge(p.status)
}

func (p *pill) textSize() float32 {
	if p.small {
		return themeSize(theme.SizeNameCaptionText)
	}
	return themeSize(theme.SizeNameText)
}

func (p *pill) CreateRenderer() fyne.WidgetRenderer {
	r := &pillRenderer{p: p, bg: canvas.NewRectangle(color.Transparent), txt: canvas.NewText("", color.Black)}
	r.apply()
	return r
}

type pillRenderer struct {
	p   *pill
	bg  *canvas.Rectangle
	txt *canvas.Text
}

func (r *pillRenderer) apply() {
	c := r.p.Colors()
	r.bg.FillColor = hex(c.Background)
	r.txt.Color = hex(c.Text)
	r.txt.Text = r.p.text
	r.txt.TextSize = r.p.textSize()
}

func (r *pillRenderer) Layout(s fyne.Size) {
	r.bg.Move(fyne.NewPos(0, 0))
	r.bg.Resize(s)
	r.bg.CornerRadius = s.Height / 2
	ts := r.txt.MinSize()
	r.txt.Resize(ts)
	r.txt.Move(fyne.NewPos((s.Width-ts.Width)/2, (s.Height-ts.Height)/2))
}

func (r *pillRenderer) MinSize() fyne.Size {
	ts := fyne.MeasureText(r.p.text, r.p.textSize(), fyne.TextStyle{})
	return fyne.NewSize(ts.Width+16, ts.Height+4)
}

func (r *pillRenderer) Refresh() {
	r.apply()
	r.Layout(r.p.Size())
	r.bg.Refresh()
	r.txt.Refresh()
}

func (r *pillRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.bg, r.txt} }
func (r *pillRenderer) Destroy()                      {}

// ---------------------------------------------------------------- stars

// starPoints returns the 10 vertices of a five pointed star in clockwise
// order, scaled to the unit square.
func starPoints() [10]fyne.Position {
	var raw [10][2]float64
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for k := 0; k < 10; k++ {
		rad := 0.5
		if k%2 == 1 {
			rad = 0.5 * 0.4
		}
		ang := (-90 + 36*float64(k)) * math.Pi / 180
		x, y := rad*math.Cos(ang), rad*math.Sin(ang)
		raw[k] = [2]float64{x, y}
		minX, maxX = math.Min(minX, x), math.Max(maxX, x)
		minY, maxY = math.Min(minY, y), math.Max(maxY, y)
	}
	var out [10]fyne.Position
	for k := range raw {
		out[k] = fyne.NewPos(float32((raw[k][0]-minX)/(maxX-minX)), float32((raw[k][1]-minY)/(maxY-minY)))
	}
	return out
}

var starShape = starPoints()

func newStarPolygon(points []fyne.Position) *canvas.ArbitraryPolygon {
	return &canvas.ArbitraryPolygon{Points: points, NormalizedPoints: true, FillColor: color.Transparent}
}

// starRow draws five stars (full, half, empty) and optionally the rating text.
type starRow struct {
	widget.BaseWidget
	full, half, empty int
	text              string
	withText          bool
}

func newStarRow(withText bool) *starRow {
	s := &starRow{withText: withText, empty: 5}
	s.ExtendBaseWidget(s)
	return s
}

func (s *starRow) SetRating(r float32) {
	s.full, s.half, s.empty = core.Stars(r)
	s.text = core.FormatRating(r)
	s.Refresh()
}

// Counts returns the number of full, half and empty stars drawn.
func (s *starRow) Counts() (int, int, int) { return s.full, s.half, s.empty }

func (s *starRow) CreateRenderer() fyne.WidgetRenderer {
	r := &starRowRenderer{s: s, txt: canvas.NewText("", color.Black)}
	for i := range r.stars {
		r.stars[i] = newStarPolygon(starShape[:])
	}
	left := []fyne.Position{starShape[5], starShape[6], starShape[7], starShape[8], starShape[9], starShape[0]}
	r.halfStar = newStarPolygon(left)
	r.apply()
	return r
}

type starRowRenderer struct {
	s        *starRow
	stars    [5]*canvas.ArbitraryPolygon
	halfStar *canvas.ArbitraryPolygon
	txt      *canvas.Text
}

func (r *starRowRenderer) apply() {
	p := core.Colors(isDark())
	for i, st := range r.stars {
		if i < r.s.full {
			st.FillColor = hex(p.StarFilled)
		} else {
			st.FillColor = hex(p.StarEmpty)
		}
	}
	r.halfStar.FillColor = hex(p.StarFilled)
	r.halfStar.Hidden = r.s.half == 0
	r.txt.Text = r.s.text
	r.txt.Color = themeColor(theme.ColorNameForeground)
	r.txt.TextSize = themeSize(theme.SizeNameText)
	r.txt.Hidden = !r.s.withText
}

func (r *starRowRenderer) Layout(s fyne.Size) {
	y := (s.Height - starSize) / 2
	for i, st := range r.stars {
		st.Move(fyne.NewPos(float32(i)*(starSize+starGap), y))
		st.Resize(fyne.NewSize(starSize, starSize))
	}
	r.halfStar.Move(fyne.NewPos(float32(r.s.full)*(starSize+starGap), y))
	r.halfStar.Resize(fyne.NewSize(starSize, starSize))
	ts := r.txt.MinSize()
	r.txt.Resize(ts)
	r.txt.Move(fyne.NewPos(5*starSize+4*starGap+6, (s.Height-ts.Height)/2))
}

func (r *starRowRenderer) MinSize() fyne.Size {
	w := float32(5*starSize + 4*starGap)
	h := float32(starSize)
	if r.s.withText {
		ts := fyne.MeasureText("5.0", themeSize(theme.SizeNameText), fyne.TextStyle{})
		w += 6 + ts.Width
		h = maxf(h, ts.Height)
	}
	return fyne.NewSize(w, h)
}

func (r *starRowRenderer) Refresh() {
	r.apply()
	r.Layout(r.s.Size())
	for _, st := range r.stars {
		st.Refresh()
	}
	r.halfStar.Refresh()
	r.txt.Refresh()
}

func (r *starRowRenderer) Objects() []fyne.CanvasObject {
	objs := make([]fyne.CanvasObject, 0, 7)
	for _, st := range r.stars {
		objs = append(objs, st)
	}
	return append(objs, r.halfStar, r.txt)
}

func (r *starRowRenderer) Destroy() {}

// ---------------------------------------------------------------- category tile

type tile struct {
	widget.BaseWidget
	cat core.Category
}

func newTile() *tile {
	t := &tile{}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tile) SetCategory(c core.Category) {
	t.cat = c
	t.Refresh()
}

func (t *tile) CreateRenderer() fyne.WidgetRenderer {
	r := &tileRenderer{t: t, bg: canvas.NewRectangle(color.Transparent), txt: canvas.NewText("", color.White)}
	r.txt.TextStyle = fyne.TextStyle{Bold: true}
	r.txt.TextSize = 22
	r.bg.CornerRadius = 10
	r.apply()
	return r
}

type tileRenderer struct {
	t   *tile
	bg  *canvas.Rectangle
	txt *canvas.Text
}

func (r *tileRenderer) apply() {
	p := core.Colors(isDark())
	r.bg.FillColor = hex(p.Tile(r.t.cat))
	r.txt.Color = hex(p.TileText)
	r.txt.Text = core.CategoryInitial(r.t.cat)
}

func (r *tileRenderer) Layout(s fyne.Size) {
	r.bg.Move(fyne.NewPos(0, 0))
	r.bg.Resize(s)
	ts := r.txt.MinSize()
	r.txt.Resize(ts)
	r.txt.Move(fyne.NewPos((s.Width-ts.Width)/2, (s.Height-ts.Height)/2))
}

func (r *tileRenderer) MinSize() fyne.Size { return fyne.NewSize(48, 48) }

func (r *tileRenderer) Refresh() {
	r.apply()
	r.Layout(r.t.Size())
	r.bg.Refresh()
	r.txt.Refresh()
}

func (r *tileRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.bg, r.txt} }
func (r *tileRenderer) Destroy()                      {}
