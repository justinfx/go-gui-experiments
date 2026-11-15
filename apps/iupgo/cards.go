package main

import (
	"fmt"
	"math"
	"strconv"

	"github.com/gen2brain/iup-go/iup"

	"gguix/core"
)

const (
	cardH    = 96
	cardGap  = 8
	cardStep = cardH + cardGap
	cardPad  = 12
	rightCol = 160
)

// cardsView is the state around the cards canvas. The list is a canvas with
// a vertical scrollbar whose unit is one card: IUP has no virtual list with
// custom items (IupList draws text only), so the app draws the visible
// cards itself with the IupDraw functions.
type cardsView struct {
	ih    iup.Ihandle
	total int
	top   float64 // POSY: the top card, fractional while scrolling
	h     int     // cards that fit
	w, ph int     // canvas size in pixels
	font  fontSet
}

// cardSpec is what one card shows, independent of the toolkit.
type cardSpec struct {
	Row         int
	Placeholder bool
	Selected    bool
	Name        string
	Sub         string
	Notes       string
	Initial     string
	Tile        string
	Status      string
	BadgeBG     string
	BadgeFG     string
	Featured    bool
	FeatBG      string
	FeatFG      string
	Price       string
	Full, Half  int
	Empty       int
	StarFilled  string
	StarEmpty   string
	Rating      string
	StockLabel  string
	StockFrac   float64
	StockTrack  string
	StockFill   string
}

func (a *app) cardSpecFor(row int) cardSpec {
	sp := cardSpec{Row: row, Selected: row == a.selRow}
	it, ok := a.pager.Row(row)
	if !ok {
		sp.Placeholder = true
		sp.Name = "Loading…"
		return sp
	}
	p := a.pal
	f, h, e := core.Stars(it.Rating)
	b := p.Badge(it.Status)
	sp.Name = it.Name
	sp.Sub = it.SKU + " · " + it.Category.String()
	sp.Notes = core.NotesPreview(it.Notes)
	sp.Initial = core.CategoryInitial(it.Category)
	sp.Tile = p.Tile(it.Category)
	sp.Status, sp.BadgeBG, sp.BadgeFG = it.Status.String(), b.Background, b.Text
	if it.Featured {
		sp.Featured, sp.FeatBG, sp.FeatFG = true, p.Featured.Background, p.Featured.Text
	}
	sp.Price = core.FormatPrice(it.PriceCents)
	sp.Full, sp.Half, sp.Empty = f, h, e
	sp.StarFilled, sp.StarEmpty = p.StarFilled, p.StarEmpty
	sp.Rating = core.FormatRating(it.Rating)
	sp.StockLabel = core.StockLabel(it.Quantity)
	sp.StockFrac = core.StockFraction(it.Quantity)
	sp.StockTrack, sp.StockFill = p.StockTrack, p.StockFillFor(it.Quantity)
	return sp
}

func (a *app) buildCards() {
	c := &a.ui.crd
	c.h = 8
	c.ih = iup.Canvas()
	set(c.ih, "SCROLLBAR", "VERTICAL", "EXPAND", "YES", "RASTERSIZE", "200x200", "BORDER", "NO",
		"YMIN", "0", "YMAX", "1", "DY", "1", "POSY", "0", "LINEY", "1")
	iup.On(c.ih, iup.ActionCB, func(iup.Ihandle) int { a.cardsDraw(); return iup.DEFAULT })
	iup.On(c.ih, iup.ScrollCB, func(ih iup.Ihandle, op int, x, y float64) int {
		c.top = iup.GetDouble(ih, "POSY")
		a.cardsRange() // tell the pager which cards are visible now
		iup.Update(ih)
		return iup.DEFAULT
	})
	iup.On(c.ih, iup.ResizeCB, func(ih iup.Ihandle, w, h int) int {
		a.cardsResized()
		return iup.DEFAULT
	})
	iup.On(c.ih, iup.ButtonCB, func(ih iup.Ihandle, button, pressed, x, y int, status string) int {
		if pressed == 1 && button == iup.BUTTON1 {
			iup.SetFocus(ih)
			if row, ok := a.cardAt(y); ok {
				a.requestSelect(row)
			}
		}
		return iup.DEFAULT
	})
	iup.On(c.ih, iup.KAnyCB, func(ih iup.Ihandle, k int) int {
		switch k {
		case iup.K_UP:
			a.moveSelection(-1)
			return iup.IGNORE
		case iup.K_DOWN:
			a.moveSelection(1)
			return iup.IGNORE
		}
		return iup.CONTINUE
	})
}

// cardAt maps a y coordinate in the canvas to a row.
func (a *app) cardAt(y int) (int, bool) {
	c := &a.ui.crd
	first := int(math.Floor(c.top))
	off := (c.top - float64(first)) * cardStep
	fy := float64(y) + off
	i := int(fy) / cardStep
	if int(fy)%cardStep >= cardH {
		return 0, false
	}
	row := first + i
	if row < 0 || row >= max(a.ui.total, 0) {
		return 0, false
	}
	return row, true
}

func (a *app) cardsResized() {
	c := &a.ui.crd
	_, w, h := iup.GetInt2(c.ih, "RASTERSIZE")
	c.w, c.ph = w, h
	a.cardsRange()
}

// cardsRange updates DY and the pager's visible range for the canvas size.
func (a *app) cardsRange() {
	c := &a.ui.crd
	if c.ph <= 0 {
		return
	}
	n := int(math.Ceil(float64(c.ph) / cardStep))
	c.h = max(n, 1)
	dy := math.Min(float64(c.ph)/cardStep, math.Max(float64(a.ui.total), 1))
	set(c.ih, "DY", fmt.Sprintf("%g", dy))
	if a.viewMode == 1 {
		first := int(math.Floor(c.top))
		a.visH = c.h
		a.setVisible(first, first+c.h-1)
	}
}

func (a *app) cardsSetTotal(total int) {
	c := &a.ui.crd
	n := max(total, 0)
	if c.total == n && total >= 0 {
		return
	}
	c.total = n
	c.top = 0
	set(c.ih, "YMAX", strconv.Itoa(max(n, 1)), "POSY", "0")
	a.cardsRange()
	iup.Update(c.ih)
}

func (a *app) cardsRedraw() {
	if a.viewMode == 1 {
		iup.Update(a.ui.crd.ih)
	}
}

func (a *app) cardsScrollTo(row int) {
	c := &a.ui.crd
	total := a.ui.total
	if total <= 0 {
		return
	}
	c.top = float64(min(row, max(total-1, 0)))
	set(c.ih, "POSY", fmt.Sprintf("%g", c.top))
	a.cardsRange()
	iup.Update(c.ih)
}

func (a *app) cardsTheme() {
	a.cardsFonts()
}

// ---- drawing ------------------------------------------------------------

type fontSet struct {
	regular, bold, price, tile string
	lh                         int
}

func (a *app) cardsFonts() {
	face := iup.GetGlobal("DEFAULTFONTFACE")
	size := iup.GetGlobalInt("DEFAULTFONTSIZE")
	if face == "" {
		face = "Sans"
	}
	if size == 0 {
		size = 10
	}
	a.ui.crd.font = fontSet{
		regular: fmt.Sprintf("%s, %d", face, size),
		bold:    fmt.Sprintf("%s, Bold %d", face, size),
		price:   fmt.Sprintf("%s, Bold %d", face, size+4),
		tile:    fmt.Sprintf("%s, Bold %d", face, size+6),
	}
}

func rgbStr(c [3]uint8) string { return fmt.Sprintf("%d %d %d", c[0], c[1], c[2]) }

func hexRGB(hex string) string {
	c := core.HexColor(hex)
	return fmt.Sprintf("%d %d %d", c.R, c.G, c.B)
}

func mixRGB(fg, bg [3]uint8, t float64) [3]uint8 {
	var o [3]uint8
	for i := range o {
		o[i] = uint8(math.Round(float64(fg[i])*t + float64(bg[i])*(1-t)))
	}
	return o
}

func globalRGB(name string) [3]uint8 {
	r, g, b := iup.GetGlobalRGB(name)
	return [3]uint8{r, g, b}
}

func (a *app) cardsDraw() {
	c := &a.ui.crd
	ih := c.ih
	if c.font.regular == "" {
		a.cardsFonts()
	}
	iup.DrawBegin(ih)
	defer iup.DrawEnd(ih)
	w, h := iup.DrawGetSize(ih)
	if w != c.w || h != c.ph {
		c.w, c.ph = w, h
		defer a.cardsRange()
	}
	list := globalRGB("DLGBGCOLOR")
	set(ih, "DRAWSTYLE", "FILL", "DRAWCOLOR", rgbStr(list))
	iup.DrawRectangle(ih, 0, 0, w, h)
	first := int(math.Floor(c.top))
	off := int(math.Round((c.top - float64(first)) * cardStep))
	for i := 0; ; i++ {
		y := i*cardStep - off
		if y >= h {
			break
		}
		row := first + i
		if row >= a.ui.total {
			break
		}
		a.drawCard(ih, a.cardSpecFor(row), 4, y, w-8)
	}
}

func (a *app) drawCard(ih iup.Ihandle, sp cardSpec, x, y, w int) {
	c := &a.ui.crd
	fg := globalRGB("TXTFGCOLOR")
	surface := globalRGB("TXTBGCOLOR")
	muted := mixRGB(fg, surface, 0.6)
	border := mixRGB(fg, surface, 0.2)
	accent := globalRGB("ACCENTCOLOR")

	set(ih, "DRAWSTYLE", "FILL", "DRAWCOLOR", rgbStr(surface))
	iup.DrawRoundedRectangle(ih, x, y, x+w-1, y+cardH-1, 8)
	set(ih, "DRAWSTYLE", "STROKE", "DRAWCOLOR", rgbStr(border), "DRAWLINEWIDTH", "1")
	if sp.Selected {
		set(ih, "DRAWCOLOR", rgbStr(accent), "DRAWLINEWIDTH", "2")
	}
	iup.DrawRoundedRectangle(ih, x, y, x+w-1, y+cardH-1, 8)
	set(ih, "DRAWLINEWIDTH", "1", "DRAWSTYLE", "FILL", "DRAWTEXTALIGNMENT", "ALEFT", "DRAWTEXTELLIPSIS", "NO", "DRAWTEXTCLIP", "NO")

	text := func(font string, col [3]uint8, s string, tx, ty, tw, th int, align string, ellipsis bool) {
		el := "NO"
		if ellipsis {
			el = "YES"
		}
		set(ih, "DRAWFONT", font, "DRAWCOLOR", rgbStr(col), "DRAWTEXTALIGNMENT", align, "DRAWTEXTELLIPSIS", el, "DRAWTEXTCLIP", el)
		iup.DrawText(ih, s, tx, ty, tw, th)
	}
	set(ih, "DRAWFONT", c.font.regular)
	_, _, lh := iup.DrawGetTextMetrics(ih)
	lh = min(max(lh, 12), 17)
	cx := x + cardPad
	if sp.Placeholder {
		text(c.font.bold, fg, sp.Name, cx, y+cardPad, w-2*cardPad, lh+2, "ALEFT", true)
		return
	}
	// Category tile.
	set(ih, "DRAWSTYLE", "FILL", "DRAWCOLOR", hexRGB(sp.Tile))
	iup.DrawRoundedRectangle(ih, cx, y+cardPad, cx+47, y+cardPad+47, 8)
	text(c.font.tile, [3]uint8{255, 255, 255}, sp.Initial, cx, y+cardPad+16, 48, 20, "ACENTER", false)

	// Middle column.
	mx := cx + 48 + cardPad
	mw := x + w - cardPad - rightCol - cardPad - mx
	ly := y + 10
	text(c.font.bold, fg, sp.Name, mx, ly, mw, lh+2, "ALEFT", true)
	text(c.font.regular, muted, sp.Sub, mx, ly+lh+2, mw, lh+2, "ALEFT", true)
	text(c.font.regular, muted, sp.Notes, mx, ly+2*(lh+2), mw, lh+2, "ALEFT", true)
	cy := ly + 3*(lh+2) + 2
	set(ih, "DRAWFONT", c.font.regular)
	chip := func(label, bg, fgc string, cx int) int {
		tw, th := iup.DrawGetTextSize(ih, label)
		cw := tw + 16
		set(ih, "DRAWSTYLE", "FILL", "DRAWCOLOR", hexRGB(bg))
		iup.DrawRoundedRectangle(ih, cx, cy, cx+cw-1, cy+17, 9)
		text(c.font.regular, hexTriple(fgc), label, cx+8, cy+(18-th)/2, tw, th, "ALEFT", false)
		return cx + cw + 6
	}
	nx := chip(sp.Status, sp.BadgeBG, sp.BadgeFG, mx)
	if sp.Featured {
		chip("Featured", sp.FeatBG, sp.FeatFG, nx)
	}

	// Right column.
	rx := x + w - cardPad
	text(c.font.price, fg, sp.Price, rx-rightCol, y+cardPad-2, rightCol, 24, "ARIGHT", false)
	set(ih, "DRAWFONT", c.font.regular)
	rw, _ := iup.DrawGetTextSize(ih, sp.Rating)
	sx := rx - rw - 6 - starsW
	sy := y + 42
	drawStarsAt(ih, sp.Full, sp.Half, sp.StarFilled, sp.StarEmpty, sx, sy)
	text(c.font.regular, fg, sp.Rating, rx-rw, sy-1, rw, lh+2, "ARIGHT", false)
	by := y + 68
	set(ih, "DRAWSTYLE", "FILL", "DRAWCOLOR", hexRGB(sp.StockTrack))
	iup.DrawRoundedRectangle(ih, rx-barW, by, rx-1, by+barH-1, 4)
	if fw := int(math.Round(sp.StockFrac * barW)); fw > 0 {
		set(ih, "DRAWCOLOR", hexRGB(sp.StockFill))
		iup.DrawRoundedRectangle(ih, rx-barW, by, rx-barW+fw-1, by+barH-1, 4)
	}
	lw, _ := iup.DrawGetTextSize(ih, sp.StockLabel)
	text(c.font.regular, muted, sp.StockLabel, rx-barW-6-lw, by-4, lw, lh, "ARIGHT", false)
}

func drawStarsAt(ih iup.Ihandle, full, half int, filledHex, emptyHex string, x, y int) {
	for i := 0; i < 5; i++ {
		sx := x + i*(starSize+starGap)
		poly := starPoly(float64(sx)+starSize/2.0, float64(y)+starSize/2.0, starSize/2.0)
		pts := make([]int, 0, 20)
		for _, p := range poly {
			pts = append(pts, int(math.Round(p[0])), int(math.Round(p[1])))
		}
		set(ih, "DRAWSTYLE", "FILL")
		switch {
		case i < full:
			set(ih, "DRAWCOLOR", hexRGB(filledHex))
			iup.DrawPolygon(ih, pts, len(poly))
		case i == full && half > 0:
			set(ih, "DRAWCOLOR", hexRGB(emptyHex))
			iup.DrawPolygon(ih, pts, len(poly))
			set(ih, "DRAWCOLOR", hexRGB(filledHex))
			iup.DrawSetClipRect(ih, sx, y-1, sx+starSize/2, y+starSize+1)
			iup.DrawPolygon(ih, pts, len(poly))
			iup.DrawResetClip(ih)
		default:
			set(ih, "DRAWCOLOR", hexRGB(emptyHex))
			iup.DrawPolygon(ih, pts, len(poly))
		}
	}
}

func hexTriple(hex string) [3]uint8 {
	c := core.HexColor(hex)
	return [3]uint8{c.R, c.G, c.B}
}
