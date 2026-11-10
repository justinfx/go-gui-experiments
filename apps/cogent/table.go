package main

// The virtualized table (brief section 5.3). It is built on core.ListBase,
// the base of Cogent Core's List and Table widgets, which owns virtualization,
// the scrollbar, row selection and arrow-key navigation. core.Table binds to a
// reflected slice of structs and sorts that slice itself, so it cannot sit on
// a core.Pager. rowsTable keeps the ListBase machinery but backs it with a
// zero-size []struct{} whose length is the pager's Total (no row data), draws
// every cell from core.Pager rows, and sends header clicks to the data service.

import (
	"fmt"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/tree"

	gc "gguix/core"
)

const (
	colID = iota
	colSKU
	colName
	colCategory
	colPrice
	colQty
	colStock
	colStatus
	colRating
	colUpdated
	colFeatured
	nCols
)

type colSpec struct {
	title string
	width float32 // dp
	align text.Aligns
	sort  gc.SortField
}

var columns = [nCols]colSpec{
	colID:       {"ID", 80, text.End, gc.SortID},
	colSKU:      {"SKU", 110, text.Start, gc.SortSKU},
	colName:     {"Name", 260, text.Start, gc.SortName},
	colCategory: {"Category", 110, text.Start, gc.SortCategory},
	colPrice:    {"Price", 100, text.End, gc.SortPrice},
	colQty:      {"Qty", 80, text.End, gc.SortQuantity},
	colStock:    {"Stock", 120, text.Start, gc.SortQuantity},
	colStatus:   {"Status", 120, text.Start, gc.SortStatus},
	colRating:   {"Rating", 110, text.Start, gc.SortRating},
	colUpdated:  {"Updated", 150, text.Start, gc.SortUpdated},
	colFeatured: {"Featured", 80, text.Center, gc.SortFeatured},
}

// rowsTable is the table view.
type rowsTable struct {
	core.ListBase
	app     *App
	header  *core.Frame
	widths  [nCols]float32 // current column widths, dp
	backing []struct{}
	lastVis [2]int
}

func (t *rowsTable) Init() {
	t.ListBase.Init()
	for i := range columns {
		t.widths[i] = columns[i].width
	}
	t.lastVis = [2]int{-1, -1}
	t.SetReadOnly(true)
	t.SelectedIndex = -1
	t.SetSlice(&t.backing)
	t.OnSelect(func(e events.Event) { // arrow keys
		t.app.userSelect(t.SelectedIndex)
	})
	t.Updater(func() {
		t.UpdateStartIndex()
		t.visibleChanged()
	})
	t.Makers.Normal[0] = func(p *tree.Plan) {
		ls := t.This.(core.Lister)
		ls.UpdateSliceSize()
		t.UpdateStartIndex()
		t.makeHeader(p)
		t.MakeGrid(p, func(p *tree.Plan) {
			for i := range t.VisibleRows {
				ls.MakeRow(p, i)
			}
		})
	}
}

// UpdateSliceSize sizes the zero-size backing slice to the pager's total.
func (t *rowsTable) UpdateSliceSize() int {
	n := 0
	if t.app != nil {
		n = max(t.app.pager.Total(), 0)
	}
	if len(t.backing) != n {
		t.backing = make([]struct{}, n)
	}
	t.SliceSize = n
	return n
}

func (t *rowsTable) UpdateMaxWidths()                                        {}
func (t *rowsTable) HasStyler() bool                                         { return false }
func (t *rowsTable) StyleRow(w core.Widget, idx, fidx int)                   {}
func (t *rowsTable) StyleValue(w core.Widget, s *styles.Style, row, col int) {}
func (t *rowsTable) NewAt(idx int)                                           {}
func (t *rowsTable) DeleteAt(idx int)                                        {}

func (t *rowsTable) RowWidgetNs() (nWidgPerRow, idxOff int) { return nCols, 0 }

// ColumnWidthDots implements core.ColumnWidther.
func (t *rowsTable) ColumnWidthDots(col int) float32 {
	if col < 0 || col >= nCols {
		return 0
	}
	return t.widths[col] * t.Styles.UnitContext.Dots(units.UnitDp)
}

// visibleRange returns the first and last visible row index.
func (t *rowsTable) visibleRange() (first, last int) {
	first = t.StartIndex
	last = first + max(t.VisibleRows, 1) - 1
	if t.SliceSize > 0 {
		last = min(last, t.SliceSize-1)
	}
	return
}

// visibleChanged tells the pager which rows this view shows.
func (t *rowsTable) visibleChanged() {
	if t.app == nil || t.app.currentView() != viewTable {
		return
	}
	first, last := t.visibleRange()
	t.lastVis = [2]int{first, last}
	t.app.pager.SetVisible(first, last)
	t.app.visibleMoved()
}

// rowItem returns the item shown at widget row i.
func (t *rowsTable) rowItem(i int) (it gc.Item, ok, invis bool) {
	si, _, inv := t.SliceIndex(i)
	if inv {
		return gc.Item{}, false, true
	}
	it, ok = t.app.pager.Row(si)
	return it, ok, false
}

// setSelected sets the displayed selection (-1 for none).
func (t *rowsTable) setSelected(idx int) {
	t.SelectedIndex = idx
	t.ResetSelectedIndexes()
	if idx >= 0 {
		t.SelectIndex(idx)
	}
	t.Restyle()
	t.NeedsRender()
}

// scrollTop makes row idx the top visible row.
func (t *rowsTable) scrollTop(idx int) {
	idx = max(0, min(idx, max(t.SliceSize-1, 0)))
	t.StartIndex = idx
	t.UpdateStartIndex()
	t.syncThumb()
	t.visibleChanged()
	t.Update()
}

// syncThumb moves the scrollbar thumb to StartIndex.
func (t *rowsTable) syncThumb() {
	lg := t.ListGrid
	if lg == nil || !lg.HasScroll[math32.Y] || lg.Scrolls[math32.Y] == nil {
		return
	}
	mx, _, _ := lg.ScrollValues(math32.Y)
	rht := mx / float32(max(t.SliceSize, 1))
	lg.Scrolls[math32.Y].SetValue(float32(t.StartIndex) * rht)
}

func (t *rowsTable) focusTable() {
	if t.CanFocus() && !t.ContainsFocus() {
		t.SetFocusQuiet()
	}
}

// headerText returns the header label with the sort indicator.
func (t *rowsTable) headerText(c int) string {
	s := columns[c].title
	if t.app != nil && t.app.sortCol == c {
		if t.app.query.Desc {
			s += " ▼"
		} else {
			s += " ▲"
		}
	}
	return s
}

func (t *rowsTable) makeHeader(p *tree.Plan) {
	tree.AddAt(p, "header", func(w *core.Frame) {
		t.header = w
		core.ToolbarStyles(w)
		w.FinalStyler(func(s *styles.Style) {
			s.Padding.Zero()
			s.Grow.Set(0, 0)
			s.Gap.Zero()
		})
		w.Maker(func(p *tree.Plan) {
			for c := range nCols {
				tree.AddAt(p, fmt.Sprintf("head-%d", c), func(b *core.Button) {
					b.SetType(core.ButtonAction)
					b.Styler(func(s *styles.Style) {
						s.Justify.Content = styles.Start
					})
					b.OnClick(func(e events.Event) { t.app.sortByColumn(c) })
					b.Updater(func() {
						b.SetText(t.headerText(c))
						b.Tooltip = columns[c].title + " (click to sort)"
					})
				})
				t.makeColumnHandle(p, c)
			}
		})
	})
}

// makeColumnHandle adds the draggable handle after the header of column c.
func (t *rowsTable) makeColumnHandle(p *tree.Plan, c int) {
	tree.AddAt(p, fmt.Sprintf("width-%d", c), func(w *core.Handle) {
		w.SetTooltip("Drag to set the width of the column, double click to reset it")
		w.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.SetAbilities(false, abilities.Focusable)
			s.SetAbilities(true, abilities.DoubleClickable)
		})
		w.FinalStyler(func(s *styles.Style) {
			s.Min.X.Dp(3)
			s.Min.Y.Em(1.5)
			s.Margin.SetHorizontal(units.Dp(2))
		})
		var start float32
		w.On(events.SlideStart, func(e events.Event) {
			start = t.widths[c] * t.Styles.UnitContext.Dots(units.UnitDp)
		})
		w.On(events.SlideMove, func(e events.Event) {
			e.SetHandled()
			dots := start + float32(e.Pos().X-e.StartPos().X)
			t.widths[c] = max(dots/t.Styles.UnitContext.Dots(units.UnitDp), 24)
			t.Restyle()
			t.NeedsLayout()
		})
		w.OnDoubleClick(func(e events.Event) {
			t.widths[c] = columns[c].width
			t.Restyle()
			t.NeedsLayout()
		})
	})
}

// SizeFinal aligns the header columns with the body columns.
func (t *rowsTable) SizeFinal() {
	t.ListBase.SizeFinal()
	sg := t.ListGrid
	sh := t.header
	if sg == nil || sh == nil || sh.NumChildren() != 2*nCols || sg.NumChildren() < nCols {
		return
	}
	gap := sg.Styles.Gap.X.Dots
	setWidth := func(w *core.WidgetBase, width float32) {
		ksz := &w.Geom.Size
		d := width - ksz.Actual.Total.X
		ksz.Actual.Total.X += d
		ksz.Actual.Content.X += d
		ksz.Alloc.Total.X = ksz.Actual.Total.X
		ksz.Alloc.Content.X = ksz.Actual.Content.X
	}
	for c := range nCols {
		hi := 2 * c
		handle := core.AsWidget(sh.Child(hi + 1)).Geom.Size.Actual.Total.X
		cw := core.AsWidget(sg.Child(c)).Geom.Size.Alloc.Total.X + gap
		setWidth(core.AsWidget(sh.Child(hi)), cw-handle)
	}
	gsz := &sg.Geom.Size
	ksz := &sh.Geom.Size
	ksz.Actual.Total.X = gsz.Actual.Total.X
	ksz.Actual.Content.X = gsz.Actual.Content.X
	ksz.Alloc.Total.X = gsz.Alloc.Total.X
	ksz.Alloc.Content.X = gsz.Alloc.Content.X
}

// initCell sets up the list properties, styles and selection of a cell.
func (t *rowsTable) initCell(w core.Widget, i, col int) {
	wb := w.AsWidget()
	wb.SetProperty(core.ListRowProperty, i)
	wb.SetProperty(core.ListColProperty, col)
	selectable(w, func() {
		t.focusTable()
		t.app.userSelect(t.StartIndex + i)
	})
	wb.Updater(func() {
		_, _, invis := t.rowItem(i)
		wb.SetState(invis, states.Invisible)
	})
}

func (t *rowsTable) MakeRow(p *tree.Plan, i int) {
	for c := range nCols {
		name := fmt.Sprintf("cell-%d-%d", i, c)
		switch c {
		case colStock:
			tree.AddAt(p, name, func(w *stockRow) {
				w.build(false, func(q int32) string { return fmt.Sprint(q) })
				t.initCell(w, i, c)
				w.Updater(func() {
					it, ok, _ := t.rowItem(i)
					w.set(it.Quantity, ok)
				})
			})
		case colStatus:
			tree.AddAt(p, name, func(w *statusCell) {
				t.initCell(w, i, c)
				w.Updater(func() {
					it, ok, _ := t.rowItem(i)
					w.set(it.Status, ok)
				})
			})
		case colRating:
			tree.AddAt(p, name, func(w *ratingRow) {
				t.initCell(w, i, c)
				w.Updater(func() {
					it, ok, _ := t.rowItem(i)
					w.set(it.Rating, ok)
				})
			})
		default:
			tree.AddAt(p, name, func(w *core.Text) {
				t.initCell(w, i, c)
				w.SetType(core.TextBodyMedium)
				w.Styler(func(s *styles.Style) {
					s.SetTextWrap(false)
					s.Text.Align = columns[c].align
					s.Grow.Set(1, 0)
					s.Align.Self = styles.Center
				})
				w.Updater(func() {
					it, ok, _ := t.rowItem(i)
					w.SetText(cellText(c, it, ok))
				})
			})
		}
	}
}

// cellText returns the text of a plain text cell.
func cellText(c int, it gc.Item, ok bool) string {
	if !ok {
		return "…"
	}
	switch c {
	case colID:
		return fmt.Sprint(it.ID)
	case colSKU:
		return it.SKU
	case colName:
		return it.Name
	case colCategory:
		return it.Category.String()
	case colPrice:
		return gc.FormatPrice(it.PriceCents)
	case colQty:
		return fmt.Sprint(it.Quantity)
	case colUpdated:
		return gc.FormatUpdated(it.UpdatedAt)
	case colFeatured:
		if it.Featured {
			return "✓"
		}
		return ""
	}
	return ""
}

// statusCell is a status pill in a table cell.
type statusCell struct {
	core.Frame
	Pill *pill
}

func (s *statusCell) Init() {
	s.Frame.Init()
	s.Styler(func(st *styles.Style) {
		st.Direction = styles.Row
		st.Align.Items = styles.Center
		st.Padding.Zero()
		st.Grow.Set(0, 0)
	})
	s.Pill = tree.New[pill](s)
}

func (s *statusCell) set(st gc.Status, ok bool) {
	if !ok {
		s.Pill.Colors = nil
		s.Pill.SetText("…")
		return
	}
	statusPill(s.Pill, st)
}
