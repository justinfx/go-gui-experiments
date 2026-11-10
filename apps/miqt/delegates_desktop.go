//go:build !js

package main

import (
	"math"
	"strconv"

	"gguix/core"

	qt "github.com/mappu/miqt/qt6"
)

const (
	cardHeight    = 96
	cardGap       = 8
	cardRowHeight = cardHeight + cardGap
	barWidth      = 80
	barHeight     = 8
	starSize      = 12
	starGap       = 2
)

// paintRecord is what a delegate last drew for a cell. Tests read it to
// check the drawn values against the section 4.5.1 palette.
type paintRecord struct {
	Kind       string // "stock", "status", "rating", "card"
	Dark       bool
	Fraction   float64
	TrackHex   string
	FillHex    string
	Qty        int32
	BadgeText  string
	BadgeBG    string
	BadgeFG    string
	Full, Half int
	Empty      int
	RatingText string
	StarOn     string
	StarOff    string
	Name       string
	Price      string
	TileHex    string
	Featured   bool
	Selected   bool
	StockLabel string
	Notes      string
}

func qcolor(hex string) *qt.QColor {
	c := core.HexColor(hex)
	return qt.NewQColor11(int(c.R), int(c.G), int(c.B), int(c.A))
}

func fillRounded(p *qt.QPainter, x, y, w, h int, radius float64, c *qt.QColor) {
	if w <= 0 || h <= 0 {
		return
	}
	p.SetPenWithStyle(qt.NoPen)
	p.SetBrush(qt.NewQBrush3(c))
	p.DrawRoundedRect2(x, y, w, h, radius, radius)
}

func drawText(p *qt.QPainter, x, y, w, h int, align qt.AlignmentFlag, c *qt.QColor, s string) {
	p.SetPen(c)
	p.DrawText7(x, y, w, h, int(align|qt.AlignVCenter)|int(qt.TextSingleLine), s)
}

// starPath builds a five-point star inside a size x size box at (x, y).
func starPath(x, y, size float64) *qt.QPainterPath {
	cx, cy := x+size/2, y+size/2
	outer, inner := size/2, size/2*0.42
	path := qt.NewQPainterPath()
	for i := 0; i < 10; i++ {
		r := outer
		if i%2 == 1 {
			r = inner
		}
		ang := -math.Pi/2 + float64(i)*math.Pi/5
		px, py := cx+r*math.Cos(ang), cy+r*math.Sin(ang)
		if i == 0 {
			path.MoveTo2(px, py)
		} else {
			path.LineTo2(px, py)
		}
	}
	path.CloseSubpath()
	return path
}

// drawStars draws five stars starting at (x, y) and returns the x after them.
func drawStars(p *qt.QPainter, x, y int, rating float32, pal core.Palette) (full, half, empty int) {
	full, half, empty = core.Stars(rating)
	on, off := qcolor(pal.StarFilled), qcolor(pal.StarEmpty)
	for i := 0; i < 5; i++ {
		sx := float64(x + i*(starSize+starGap))
		path := starPath(sx, float64(y), starSize)
		switch {
		case i < full:
			p.FillPath(path, qt.NewQBrush3(on))
		case i == full && half == 1:
			p.FillPath(path, qt.NewQBrush3(off))
			p.Save()
			p.SetClipRect2(int(sx), y, starSize/2, starSize)
			p.FillPath(path, qt.NewQBrush3(on))
			p.Restore()
		default:
			p.FillPath(path, qt.NewQBrush3(off))
		}
	}
	return full, half, empty
}

// drawStockBar draws the 80x8 track and fill at (x, y) (top-left).
func drawStockBar(p *qt.QPainter, x, y int, qty int32, pal core.Palette) float64 {
	frac := core.StockFraction(qty)
	fillW := int(math.Round(frac * barWidth))
	fillW = max(fillW, 0)
	fillColor := qcolor(pal.StockFillFor(qty))
	fillRounded(p, x, y, barWidth, barHeight, barHeight/2, qcolor(pal.StockTrack))
	if fillW > 0 {
		fillRounded(p, x, y, fillW, barHeight, math.Min(barHeight/2, float64(fillW)/2), fillColor)
	}
	return frac
}

// badgeSize returns the width and height of a status pill for text.
func badgeSize(fm *qt.QFontMetrics, text string) (int, int) {
	return fm.HorizontalAdvance(text) + 16, fm.Height() + 4
}

func paintBadge(p *qt.QPainter, x, cy int, text string, fm *qt.QFontMetrics, c core.BadgeColors) int {
	w, h := badgeSize(fm, text)
	y := cy - h/2
	fillRounded(p, x, y, w, h, float64(h)/2, qcolor(c.Background))
	drawText(p, x, y, w, h, qt.AlignHCenter, qcolor(c.Text), text)
	return w
}

// ---------------------------------------------------------------- table

// cellDelegate paints the Stock, Status and Rating cells.
type cellDelegate struct {
	*qt.QStyledItemDelegate
	a   *App
	log map[[2]int]paintRecord
}

func newCellDelegate(a *App) *cellDelegate {
	d := &cellDelegate{QStyledItemDelegate: qt.NewQStyledItemDelegate(), a: a, log: map[[2]int]paintRecord{}}
	d.OnPaint(func(super func(*qt.QPainter, *qt.QStyleOptionViewItem, *qt.QModelIndex), p *qt.QPainter, opt *qt.QStyleOptionViewItem, idx *qt.QModelIndex) {
		// The base class draws the selection background and, for rows that are
		// not loaded yet, the placeholder text.
		super(p, opt, idx)
		it, ok := a.pager.Row(idx.Row())
		if !ok {
			return
		}
		d.paintCell(p, opt, idx.Row(), idx.Column(), it)
	})
	return d
}

func (d *cellDelegate) paintCell(p *qt.QPainter, opt *qt.QStyleOptionViewItem, row, col int, it core.Item) {
	dark := d.a.isDark()
	pal := core.Colors(dark)
	r := opt.Rect()
	x, y, w, h := r.X(), r.Y(), r.Width(), r.Height()
	selected := opt.State()&qt.QStyle__State_Selected != 0
	textColor := opt.Palette().Text().Color()
	if selected {
		textColor = opt.Palette().HighlightedText().Color()
	}
	rec := paintRecord{Dark: dark}
	p.Save()
	p.SetRenderHint2(qt.QPainter__Antialiasing, true)
	p.SetClipRectWithQRect(r)
	switch col {
	case colStock:
		rec.Kind = "stock"
		rec.Qty = it.Quantity
		rec.TrackHex = pal.StockTrack
		rec.FillHex = pal.StockFillFor(it.Quantity)
		rec.Fraction = drawStockBar(p, x+8, y+(h-barHeight)/2, it.Quantity, pal)
		drawText(p, x+8+barWidth+6, y, w-8-barWidth-6, h, qt.AlignLeft, textColor, strconv.Itoa(int(it.Quantity)))
	case colStatus:
		rec.Kind = "status"
		bc := pal.Badge(it.Status)
		text := it.Status.String()
		rec.BadgeText, rec.BadgeBG, rec.BadgeFG = text, bc.Background, bc.Text
		paintBadge(p, x+8, y+h/2, text, opt.FontMetrics(), bc)
	case colRating:
		rec.Kind = "rating"
		rec.StarOn, rec.StarOff = pal.StarFilled, pal.StarEmpty
		rec.Full, rec.Half, rec.Empty = drawStars(p, x+8, y+(h-starSize)/2, it.Rating, pal)
		rec.RatingText = core.FormatRating(it.Rating)
		tx := x + 8 + 5*(starSize+starGap) + 4
		drawText(p, tx, y, w-(tx-x), h, qt.AlignLeft, textColor, rec.RatingText)
	}
	p.Restore()
	if len(d.log) > 4000 {
		clear(d.log)
	}
	d.log[[2]int{row, col}] = rec
}

// ---------------------------------------------------------------- cards

// cardDelegate paints one card per item.
type cardDelegate struct {
	*qt.QStyledItemDelegate
	a   *App
	log map[int]paintRecord
}

func newCardDelegate(a *App) *cardDelegate {
	d := &cardDelegate{QStyledItemDelegate: qt.NewQStyledItemDelegate(), a: a, log: map[int]paintRecord{}}
	d.OnSizeHint(func(super func(*qt.QStyleOptionViewItem, *qt.QModelIndex) *qt.QSize, opt *qt.QStyleOptionViewItem, idx *qt.QModelIndex) *qt.QSize {
		return qt.NewQSize2(max(200, d.a.cards.Viewport().Width()), cardRowHeight)
	})
	d.OnPaint(func(super func(*qt.QPainter, *qt.QStyleOptionViewItem, *qt.QModelIndex), p *qt.QPainter, opt *qt.QStyleOptionViewItem, idx *qt.QModelIndex) {
		d.paintCard(p, opt, idx.Row())
	})
	return d
}

func (d *cardDelegate) paintCard(p *qt.QPainter, opt *qt.QStyleOptionViewItem, row int) {
	dark := d.a.isDark()
	pal := core.Colors(dark)
	r := opt.Rect()
	cx, cy, cw, ch := r.X()+2, r.Y(), r.Width()-4, cardHeight
	selected := opt.State()&qt.QStyle__State_Selected != 0
	qp := opt.Palette()
	text := qp.Text().Color()
	muted := qp.PlaceholderText().Color()

	p.Save()
	p.SetRenderHint2(qt.QPainter__Antialiasing, true)
	p.SetClipRectWithQRect(r)

	// frame: surface, then a 1 px border or a 2 px accent border when selected
	bw := 1.0
	border := qp.Mid().Color()
	if selected {
		bw = 2
		border = qp.Highlight().Color()
	}
	p.SetPenWithPen(qt.NewQPen4(qt.NewQBrush3(border), bw))
	p.SetBrush(qt.NewQBrush3(qp.Base().Color()))
	p.DrawRoundedRect(qt.NewQRectF4(float64(cx)+bw/2, float64(cy)+bw/2, float64(cw)-bw, float64(ch)-bw), 8, 8)

	rec := paintRecord{Kind: "card", Dark: dark, Selected: selected}
	it, ok := d.a.pager.Row(row)
	pad := 12
	ix, iy := cx+pad, cy+pad
	iw, ih := cw-2*pad, ch-2*pad
	bold := qt.NewQFont5(opt.Font())
	bold.SetBold(true)
	if !ok {
		p.SetFont(bold)
		drawText(p, ix, iy, iw, 20, qt.AlignLeft, text, "Loading…")
		rec.Name = "Loading…"
		p.Restore()
		d.remember(row, rec)
		return
	}

	// category tile
	tile := pal.CategoryTiles[it.Category]
	fillRounded(p, ix, iy+(ih-48)/2, 48, 48, 8, qcolor(tile))
	p.SetFont(bold)
	drawText(p, ix, iy+(ih-48)/2, 48, 48, qt.AlignHCenter, qcolor(pal.TileText), core.CategoryInitial(it.Category))
	rec.TileHex = tile

	// right column
	rightW := 160
	rx := ix + iw - rightW
	big := qt.NewQFont5(opt.Font())
	big.SetBold(true)
	big.SetPointSizeF(big.PointSizeF() + 3)
	p.SetFont(big)
	rec.Price = core.FormatPrice(it.PriceCents)
	drawText(p, rx, iy, rightW, 24, qt.AlignRight, text, rec.Price)
	p.SetFont(opt.Font())
	fm := opt.FontMetrics()
	lh := fm.Height()
	ratingText := core.FormatRating(it.Rating)
	rtw := fm.HorizontalAdvance(ratingText)
	starsW := 5*(starSize+starGap) - starGap
	sx := rx + rightW - rtw - 4 - starsW
	rec.Full, rec.Half, rec.Empty = drawStars(p, sx, iy+28+(lh-starSize)/2, it.Rating, pal)
	rec.RatingText = ratingText
	rec.StarOn, rec.StarOff = pal.StarFilled, pal.StarEmpty
	drawText(p, rx+rightW-rtw, iy+28, rtw, lh, qt.AlignRight, text, ratingText)
	bx := rx + rightW - barWidth
	rec.Qty = it.Quantity
	rec.TrackHex, rec.FillHex = pal.StockTrack, pal.StockFillFor(it.Quantity)
	rec.Fraction = drawStockBar(p, bx, iy+50+(lh-barHeight)/2, it.Quantity, pal)
	rec.StockLabel = core.StockLabel(it.Quantity)
	drawText(p, rx, iy+50, rightW-barWidth-6, lh, qt.AlignRight, muted, rec.StockLabel)

	// middle column
	mx := ix + 48 + 12
	mw := rx - 12 - mx
	p.SetFont(bold)
	rec.Name = it.Name
	drawText(p, mx, iy, mw, lh+2, qt.AlignLeft, text, opt.FontMetrics().ElidedText(it.Name, qt.ElideRight, mw))
	p.SetFont(opt.Font())
	drawText(p, mx, iy+lh+2, mw, lh, qt.AlignLeft, muted, it.SKU+" · "+it.Category.String())
	rec.Notes = core.NotesPreview(it.Notes)
	drawText(p, mx, iy+2*lh+2, mw, lh, qt.AlignLeft, muted, fm.ElidedText(rec.Notes, qt.ElideRight, mw))
	chipY := iy + 3*lh + 2 + (ih-3*lh-2)/2
	bc := pal.Badge(it.Status)
	rec.BadgeText, rec.BadgeBG, rec.BadgeFG = it.Status.String(), bc.Background, bc.Text
	bwid := paintBadge(p, mx, chipY, rec.BadgeText, fm, bc)
	if it.Featured {
		rec.Featured = true
		paintBadge(p, mx+bwid+6, chipY, "Featured", fm, pal.Featured)
	}
	p.Restore()
	d.remember(row, rec)
}

func (d *cardDelegate) remember(row int, rec paintRecord) {
	if len(d.log) > 2000 {
		clear(d.log)
	}
	d.log[row] = rec
}
