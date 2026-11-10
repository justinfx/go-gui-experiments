package main

// The cards view (brief section 5.10): a virtualized core.ListBase whose
// items are custom card frames built from child widgets.

import (
	"fmt"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"

	gc "gguix/core"
)

const (
	cardHeight = 96
	cardGap    = 8
)

// cardsList is the cards view.
type cardsList struct {
	core.ListBase
	app        *App
	backing    []struct{}
	gridStyled bool
}

func (c *cardsList) Init() {
	c.ListBase.Init()
	c.SetReadOnly(true)
	c.SelectedIndex = -1
	c.SetSlice(&c.backing)
	c.OnSelect(func(e events.Event) { // arrow keys
		c.app.userSelect(c.SelectedIndex)
	})
	c.Updater(func() {
		c.UpdateStartIndex()
		if lg := c.ListGrid; lg != nil && !c.gridStyled {
			c.gridStyled = true
			lg.Styler(func(s *styles.Style) { s.Gap.Set(units.Dp(cardGap)) })
			lg.Restyle()
		}
		c.visibleChanged()
	})
	c.Makers.Normal[0] = func(p *tree.Plan) {
		ls := c.This.(core.Lister)
		ls.UpdateSliceSize()
		c.UpdateStartIndex()
		c.MakeGrid(p, func(p *tree.Plan) {
			for i := range c.VisibleRows {
				ls.MakeRow(p, i)
			}
		})
	}
}

func (c *cardsList) UpdateSliceSize() int {
	n := 0
	if c.app != nil {
		n = max(c.app.pager.Total(), 0)
	}
	if len(c.backing) != n {
		c.backing = make([]struct{}, n)
	}
	c.SliceSize = n
	return n
}

func (c *cardsList) UpdateMaxWidths()                                        {}
func (c *cardsList) HasStyler() bool                                         { return false }
func (c *cardsList) StyleRow(w core.Widget, idx, fidx int)                   {}
func (c *cardsList) StyleValue(w core.Widget, s *styles.Style, row, col int) {}
func (c *cardsList) NewAt(idx int)                                           {}
func (c *cardsList) DeleteAt(idx int)                                        {}
func (c *cardsList) RowWidgetNs() (nWidgPerRow, idxOff int)                  { return 1, 0 }

func (c *cardsList) visibleRange() (first, last int) {
	first = c.StartIndex
	last = first + max(c.VisibleRows, 1) - 1
	if c.SliceSize > 0 {
		last = min(last, c.SliceSize-1)
	}
	return
}

func (c *cardsList) visibleChanged() {
	if c.app == nil || c.app.currentView() != viewCards {
		return
	}
	first, last := c.visibleRange()
	c.app.pager.SetVisible(first, last)
	c.app.visibleMoved()
}

func (c *cardsList) setSelected(idx int) {
	c.SelectedIndex = idx
	c.ResetSelectedIndexes()
	if idx >= 0 {
		c.SelectIndex(idx)
	}
	c.Restyle()
	c.NeedsRender()
}

func (c *cardsList) scrollTop(idx int) {
	idx = max(0, min(idx, max(c.SliceSize-1, 0)))
	c.StartIndex = idx
	c.UpdateStartIndex()
	c.syncThumb()
	c.visibleChanged()
	c.Update()
}

func (c *cardsList) syncThumb() {
	lg := c.ListGrid
	if lg == nil || !lg.HasScroll[math32.Y] || lg.Scrolls[math32.Y] == nil {
		return
	}
	mx, _, _ := lg.ScrollValues(math32.Y)
	rht := mx / float32(max(c.SliceSize, 1))
	lg.Scrolls[math32.Y].SetValue(float32(c.StartIndex) * rht)
}

func (c *cardsList) focusList() {
	if c.CanFocus() && !c.ContainsFocus() {
		c.SetFocusQuiet()
	}
}

func (c *cardsList) rowItem(i int) (it gc.Item, ok, invis bool) {
	si, _, inv := c.SliceIndex(i)
	if inv {
		return gc.Item{}, false, true
	}
	it, ok = c.app.pager.Row(si)
	return it, ok, false
}

func (c *cardsList) MakeRow(p *tree.Plan, i int) {
	tree.AddAt(p, fmt.Sprintf("card-%d", i), func(w *cardWidget) {
		w.build(c, i)
	})
}

// cardWidget is one card.
type cardWidget struct {
	core.Frame
	list *cardsList
	row  int

	Tile     *core.Frame
	TileText *core.Text
	Name     *core.Text
	Sub      *core.Text
	Notes    *core.Text
	Chips    *core.Frame
	Status   *pill
	Featured *pill
	Right    *core.Frame
	Price    *core.Text
	Rating   *ratingRow
	Stock    *stockRow
	Mid      *core.Frame

	item gc.Item
	ok   bool
}

func noTextSelect(w *core.Text) {
	w.Styler(func(s *styles.Style) {
		s.SetAbilities(false, abilities.Selectable, abilities.DoubleClickable, abilities.TripleClickable, abilities.LongPressable, abilities.Slideable)
	})
}

func (cw *cardWidget) build(l *cardsList, row int) {
	cw.list, cw.row = l, row
	cw.SetProperty(core.ListRowProperty, row)
	cw.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Grow.Set(1, 0)
		s.Min.Y.Dp(cardHeight)
		s.Max.Y.Dp(cardHeight)
		s.Padding.Set(units.Dp(12))
		s.Gap.Set(units.Dp(12))
		s.Align.Items = styles.Center
		s.Border.Style.Set(styles.BorderSolid)
		s.Border.Radius = sides.NewValues(units.Dp(8))
		s.Border.Color.Set(colors.Scheme.Outline)
		s.Border.Width.Set(units.Dp(1))
		if cw.row+l.StartIndex == l.SelectedIndex {
			s.Border.Width.Set(units.Dp(2))
			s.Border.Color.Set(colors.Scheme.Primary.Base)
		}
	})
	selectable(cw, func() {
		l.focusList()
		l.app.userSelect(l.StartIndex + cw.row)
	})

	// category tile
	cw.Tile = core.NewFrame(cw)
	cw.Tile.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(48))
		s.Max.Set(units.Dp(48))
		s.Grow.Set(0, 0)
		s.Padding.Zero()
		s.Justify.Content = styles.Center
		s.Align.Items = styles.Center
		s.Border.Radius = sides.NewValues(units.Dp(8))
		s.Background = colors.Uniform(gc.HexColor(palette().CategoryTiles[cw.item.Category%8]))
	})
	cw.TileText = core.NewText(cw.Tile).SetType(core.TextTitleLarge)
	cw.TileText.Styler(func(s *styles.Style) {
		s.Font.Weight = rich.Bold
		s.Color = colors.Uniform(gc.HexColor(palette().TileText))
		s.SetAbilities(false, abilities.Selectable, abilities.DoubleClickable, abilities.TripleClickable, abilities.LongPressable, abilities.Slideable)
	})

	// middle column
	cw.Mid = core.NewFrame(cw)
	cw.Mid.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 0)
		s.Gap.Set(units.Dp(1))
		s.Padding.Zero()
		s.Overflow.X = styles.OverflowHidden
		s.Min.X.Dp(0)
	})
	cw.Name = core.NewText(cw.Mid).SetType(core.TextBodyLarge)
	cw.Name.Styler(func(s *styles.Style) {
		s.Font.Weight = rich.Bold
		s.SetTextWrap(false)
	})
	noTextSelect(cw.Name)
	cw.Sub = core.NewText(cw.Mid).SetType(core.TextSupporting)
	cw.Sub.Styler(func(s *styles.Style) { s.SetTextWrap(false) })
	noTextSelect(cw.Sub)
	cw.Notes = core.NewText(cw.Mid).SetType(core.TextSupporting)
	cw.Notes.Styler(func(s *styles.Style) { s.SetTextWrap(false) })
	noTextSelect(cw.Notes)
	cw.Chips = core.NewFrame(cw.Mid)
	cw.Chips.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Gap.Set(units.Dp(6))
		s.Padding.Zero()
		s.Grow.Set(0, 0)
	})
	cw.Status = tree.New[pill](cw.Chips)
	cw.Featured = tree.New[pill](cw.Chips)
	featuredPill(cw.Featured)

	// right column
	cw.Right = core.NewFrame(cw)
	cw.Right.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Min.X.Dp(160)
		s.Max.X.Dp(160)
		s.Grow.Set(0, 0)
		s.Gap.Set(units.Dp(2))
		s.Padding.Zero()
		s.Align.Items = styles.End
	})
	cw.Price = core.NewText(cw.Right).SetType(core.TextTitleMedium)
	cw.Price.Styler(func(s *styles.Style) {
		s.Font.Weight = rich.Bold
		s.Text.Align = 2 // end
	})
	noTextSelect(cw.Price)
	cw.Rating = tree.New[ratingRow](cw.Right)
	cw.Stock = tree.New[stockRow](cw.Right)
	cw.Stock.build(true, gc.StockLabel)

	cw.Updater(func() { cw.set() })
}

// set fills the card from its row (placeholder when not loaded).
func (cw *cardWidget) set() {
	it, ok, invis := cw.list.rowItem(cw.row)
	cw.item, cw.ok = it, ok
	cw.SetState(invis, states.Invisible)
	cw.Sub.SetState(!ok, states.Invisible)
	cw.Notes.SetState(!ok, states.Invisible)
	cw.Chips.SetState(!ok, states.Invisible)
	cw.Right.SetState(!ok, states.Invisible)
	cw.Tile.SetState(!ok, states.Invisible)
	if !ok {
		cw.Name.SetText("Loading…")
		return
	}
	cw.TileText.SetText(gc.CategoryInitial(it.Category))
	cw.Name.SetText(it.Name)
	cw.Sub.SetText(it.SKU + " · " + it.Category.String())
	cw.Notes.SetText(gc.NotesPreview(it.Notes))
	statusPill(cw.Status, it.Status)
	cw.Featured.Hidden = !it.Featured
	cw.Price.SetText(gc.FormatPrice(it.PriceCents))
	cw.Rating.set(it.Rating, true)
	cw.Stock.set(it.Quantity, true)
}
