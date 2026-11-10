package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"gguix/core"
)

const (
	cardH       = 96
	cardGap     = 8
	cardPad     = 12
	cardRightW  = 160
	cardRowH    = 18
	cardItemH   = cardH + cardGap/2 // plus the list's own padding gives the 8 px gap
	cardMinW    = 420
	cardCorner  = 8
	cardTileW   = 48
	cardMidGap  = 12
	cardLineGap = 2
)

// cardItem is one card of the cards view.
type cardItem struct {
	widget.BaseWidget
	loaded   bool
	selected bool
	item     core.Item
}

func newCardItem() *cardItem {
	c := &cardItem{}
	c.ExtendBaseWidget(c)
	return c
}

// Set fills the card. A nil item is a placeholder card.
func (c *cardItem) Set(it *core.Item, selected bool) {
	c.selected = selected
	c.loaded = it != nil
	if it != nil {
		c.item = *it
	}
	c.Refresh()
}

func (c *cardItem) MinSize() fyne.Size { return fyne.NewSize(cardMinW, cardItemH) }

func (c *cardItem) CreateRenderer() fyne.WidgetRenderer {
	r := &cardRenderer{
		c:        c,
		frame:    canvas.NewRectangle(color.Transparent),
		tile:     newTile(),
		name:     canvas.NewText("", color.Black),
		meta:     canvas.NewText("", color.Black),
		notes:    canvas.NewText("", color.Black),
		badge:    newPill(pillStatus, true),
		featured: newPill(pillFeatured, true),
		price:    canvas.NewText("", color.Black),
		stars:    newStarRow(true),
		stockTxt: canvas.NewText("", color.Black),
		bar:      newStockBar(),
	}
	r.featured.SetFeatured()
	r.frame.CornerRadius = cardCorner
	r.name.TextStyle = fyne.TextStyle{Bold: true}
	r.price.TextStyle = fyne.TextStyle{Bold: true}
	r.update()
	return r
}

type cardRenderer struct {
	c        *cardItem
	frame    *canvas.Rectangle
	tile     *tile
	name     *canvas.Text
	meta     *canvas.Text
	notes    *canvas.Text
	badge    *pill
	featured *pill
	price    *canvas.Text
	stars    *starRow
	stockTxt *canvas.Text
	bar      *stockBar
}

func (r *cardRenderer) update() {
	c := r.c
	r.frame.FillColor = themeColor(theme.ColorNameBackground)
	if c.selected {
		r.frame.StrokeColor = themeColor(theme.ColorNamePrimary)
		r.frame.StrokeWidth = 2
	} else {
		r.frame.StrokeColor = themeColor(theme.ColorNameSeparator)
		r.frame.StrokeWidth = 1
	}
	fg := themeColor(theme.ColorNameForeground)
	muted := themeColor(theme.ColorNamePlaceHolder)
	size := themeSize(theme.SizeNameText)
	r.name.Color, r.name.TextSize = fg, size
	r.meta.Color, r.meta.TextSize = muted, size
	r.notes.Color, r.notes.TextSize = muted, size
	r.price.Color, r.price.TextSize = fg, size*1.35
	r.stockTxt.Color, r.stockTxt.TextSize = fg, size

	full := c.loaded
	r.tile.Hidden, r.meta.Hidden, r.notes.Hidden = !full, !full, !full
	r.badge.Hidden = !full
	r.featured.Hidden = !(full && c.item.Featured)
	r.price.Hidden, r.stars.Hidden, r.stockTxt.Hidden, r.bar.Hidden = !full, !full, !full, !full
	if !full {
		r.name.Text = "Loading…"
		return
	}
	it := c.item
	r.name.Text = it.Name
	r.meta.Text = it.SKU + " · " + it.Category.String()
	r.notes.Text = core.NotesPreview(it.Notes)
	r.tile.SetCategory(it.Category)
	r.badge.SetStatus(it.Status)
	r.price.Text = core.FormatPrice(it.PriceCents)
	r.stars.SetRating(it.Rating)
	r.stockTxt.Text = core.StockLabel(it.Quantity)
	r.bar.SetQty(it.Quantity)
}

func (r *cardRenderer) Layout(s fyne.Size) {
	c := r.c
	top := float32(cardGap) / 4 // centre the 96 px card in the item
	r.frame.Move(fyne.NewPos(0, top))
	r.frame.Resize(fyne.NewSize(s.Width, cardH))

	leftX := float32(cardPad)
	midX := leftX + cardTileW + cardMidGap
	rightX := s.Width - cardPad - cardRightW
	midW := rightX - cardMidGap - midX
	y0 := top + cardPad

	place := func(o fyne.CanvasObject, x, y, w float32) {
		sz := o.MinSize()
		o.Resize(sz)
		o.Move(fyne.NewPos(x, y+(cardRowH-sz.Height)/2))
		_ = w
	}

	if !c.loaded {
		r.name.Text = fitText("Loading…", r.name.TextSize, r.name.TextStyle, midW)
		place(r.name, midX, y0, midW)
		return
	}

	r.tile.Resize(fyne.NewSize(cardTileW, cardTileW))
	r.tile.Move(fyne.NewPos(leftX, top+(cardH-cardTileW)/2))

	r.name.Text = fitText(c.item.Name, r.name.TextSize, r.name.TextStyle, midW)
	place(r.name, midX, y0, midW)
	r.meta.Text = fitText(c.item.SKU+" · "+c.item.Category.String(), r.meta.TextSize, r.meta.TextStyle, midW)
	place(r.meta, midX, y0+cardRowH+cardLineGap, midW)
	r.notes.Text = fitText(core.NotesPreview(c.item.Notes), r.notes.TextSize, r.notes.TextStyle, midW)
	place(r.notes, midX, y0+2*(cardRowH+cardLineGap), midW)

	chipY := y0 + 3*(cardRowH+cardLineGap)
	bs := r.badge.MinSize()
	r.badge.Resize(bs)
	r.badge.Move(fyne.NewPos(midX, chipY+(cardRowH-bs.Height)/2))
	if c.item.Featured {
		fs := r.featured.MinSize()
		r.featured.Resize(fs)
		r.featured.Move(fyne.NewPos(midX+bs.Width+6, chipY+(cardRowH-fs.Height)/2))
	}

	// Right column, right aligned.
	rightEdge := s.Width - cardPad
	ps := r.price.MinSize()
	r.price.Resize(ps)
	r.price.Move(fyne.NewPos(rightEdge-ps.Width, y0))
	ss := r.stars.MinSize()
	r.stars.Resize(ss)
	r.stars.Move(fyne.NewPos(rightEdge-ss.Width, y0+ps.Height+2))
	stockY := y0 + ps.Height + 2 + ss.Height + 4
	bsz := r.bar.MinSize()
	r.bar.Resize(bsz)
	r.bar.Move(fyne.NewPos(rightEdge-bsz.Width, stockY+(cardRowH-bsz.Height)/2))
	ts := r.stockTxt.MinSize()
	r.stockTxt.Resize(ts)
	r.stockTxt.Move(fyne.NewPos(rightEdge-bsz.Width-6-ts.Width, stockY+(cardRowH-ts.Height)/2))
}

func (r *cardRenderer) MinSize() fyne.Size { return r.c.MinSize() }

func (r *cardRenderer) Refresh() {
	r.update()
	r.Layout(r.c.Size())
	r.frame.Refresh()
	r.tile.Refresh()
	for _, t := range []*canvas.Text{r.name, r.meta, r.notes, r.price, r.stockTxt} {
		t.Refresh()
	}
	r.badge.Refresh()
	r.featured.Refresh()
	r.stars.Refresh()
	r.bar.Refresh()
}

func (r *cardRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.frame, r.tile, r.name, r.meta, r.notes, r.badge, r.featured, r.price, r.stars, r.stockTxt, r.bar}
}

func (r *cardRenderer) Destroy() {}

// ---------------------------------------------------------------- list

// cardList is widget.List with Up/Down moving the app's selection.
type cardList struct {
	widget.List
	u *ui
}

func newCardList(u *ui) *cardList {
	l := &cardList{u: u}
	l.Length = func() int {
		n := u.pager.Total()
		if n < 0 {
			n = 0
		}
		return n
	}
	l.CreateItem = func() fyne.CanvasObject { return newCardItem() }
	l.UpdateItem = u.updateCard
	l.HideSeparators = true
	l.OnSelected = func(id widget.ListItemID) {
		l.Unselect(id)
		u.selectIndex(id)
	}
	l.ExtendBaseWidget(l)
	return l
}

func (l *cardList) pitch() float32 {
	return cardItemH + themeSize(theme.SizeNamePadding)
}

func (l *cardList) visibleRange() (int, int) {
	p := l.pitch()
	off := l.GetScrollOffset()
	h := l.Size().Height
	if h <= 0 {
		h = 600
	}
	first := int(off / p)
	last := int((off + h - 1) / p)
	if last < first {
		last = first
	}
	return first, last
}

func (l *cardList) scrollToRow(row int) {
	if row < 0 {
		row = 0
	}
	l.ScrollToOffset(float32(row) * l.pitch())
}

func (l *cardList) ensureVisible(row int) {
	first, last := l.visibleRange()
	rows := last - first
	if row < first {
		l.scrollToRow(row)
	} else if row > last-1 {
		l.scrollToRow(row - rows + 1)
	}
}

func (l *cardList) TypedKey(ev *fyne.KeyEvent) {
	switch ev.Name {
	case fyne.KeyDown:
		l.u.moveSelection(1)
	case fyne.KeyUp:
		l.u.moveSelection(-1)
	case fyne.KeyPageDown:
		l.u.moveSelection(5)
	case fyne.KeyPageUp:
		l.u.moveSelection(-5)
	}
}
