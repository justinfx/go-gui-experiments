package main

import (
	"image/color"
	"reflect"
	"strconv"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"gguix/core"
)

type column struct {
	title string
	width float32
	align fyne.TextAlign
	sort  core.SortField
}

var columns = []column{
	{"ID", 80, fyne.TextAlignTrailing, core.SortID},
	{"SKU", 110, fyne.TextAlignLeading, core.SortSKU},
	{"Name", 260, fyne.TextAlignLeading, core.SortName},
	{"Category", 110, fyne.TextAlignLeading, core.SortCategory},
	{"Price", 100, fyne.TextAlignTrailing, core.SortPrice},
	{"Qty", 80, fyne.TextAlignTrailing, core.SortQuantity},
	{"Stock", 120, fyne.TextAlignLeading, core.SortQuantity},
	{"Status", 120, fyne.TextAlignLeading, core.SortStatus},
	{"Rating", 110, fyne.TextAlignLeading, core.SortRating},
	{"Updated", 150, fyne.TextAlignLeading, core.SortUpdated},
	{"Featured", 80, fyne.TextAlignCenter, core.SortFeatured},
}

// Column indexes.
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
)

// ---------------------------------------------------------------- label metrics

var (
	rowMinCache  fyne.Size
	rowMinKey    [2]float32
	rowMinCached bool
)

// labelMin is the minimum size of a default widget.Label with one line of
// text. Table cells and headers use it so the row height is the framework's
// default one.
func labelMin() fyne.Size {
	key := [2]float32{themeSize(theme.SizeNameText), themeSize(theme.SizeNameInnerPadding)}
	if !rowMinCached || key != rowMinKey {
		rowMinCache = widget.NewLabel("Wy").MinSize()
		rowMinKey, rowMinCached = key, true
	}
	return rowMinCache
}

// ---------------------------------------------------------------- table cell

// tableCell is the one cell widget of the table. It shows plain text for
// most columns and the custom stock bar, status badge and stars for three.
type tableCell struct {
	widget.BaseWidget
	row, col int
	loaded   bool
	selected bool
	item     core.Item
}

func newTableCell() *tableCell {
	c := &tableCell{}
	c.ExtendBaseWidget(c)
	return c
}

// Set fills the cell for column col. A nil item is a placeholder.
func (c *tableCell) Set(row, col int, it *core.Item, selected bool) {
	c.row, c.col, c.selected = row, col, selected
	c.loaded = it != nil
	if it != nil {
		c.item = *it
	}
	c.Refresh()
}

// Text returns the plain text shown in a text column ("…" for placeholders).
func (c *tableCell) Text() string {
	if !c.loaded {
		return "…"
	}
	it := c.item
	switch c.col {
	case colID:
		return strconv.FormatInt(it.ID, 10)
	case colSKU:
		return it.SKU
	case colName:
		return it.Name
	case colCategory:
		return it.Category.String()
	case colPrice:
		return core.FormatPrice(it.PriceCents)
	case colQty:
		return strconv.Itoa(int(it.Quantity))
	case colUpdated:
		return core.FormatUpdated(it.UpdatedAt)
	case colFeatured:
		if it.Featured {
			return "✓"
		}
		return ""
	}
	return ""
}

// custom reports whether the column is drawn with custom widgets.
func (c *tableCell) custom() bool {
	return c.loaded && (c.col == colStock || c.col == colStatus || c.col == colRating)
}

func (c *tableCell) MinSize() fyne.Size {
	m := labelMin()
	return fyne.NewSize(m.Width, m.Height)
}

func (c *tableCell) CreateRenderer() fyne.WidgetRenderer {
	r := &tableCellRenderer{
		c:     c,
		sel:   canvas.NewRectangle(color.Transparent),
		txt:   canvas.NewText("", color.Black),
		qty:   canvas.NewText("", color.Black),
		bar:   newStockBar(),
		badge: newPill(pillStatus, false),
		stars: newStarRow(true),
	}
	r.update()
	return r
}

type tableCellRenderer struct {
	c     *tableCell
	sel   *canvas.Rectangle
	txt   *canvas.Text
	qty   *canvas.Text
	bar   *stockBar
	badge *pill
	stars *starRow
}

func (r *tableCellRenderer) update() {
	c := r.c
	r.sel.FillColor = themeColor(theme.ColorNameSelection)
	r.sel.CornerRadius = themeSize(theme.SizeNameSelectionRadius)
	r.sel.Hidden = !c.selected
	fg := themeColor(theme.ColorNameForeground)
	size := themeSize(theme.SizeNameText)
	r.txt.Color, r.txt.TextSize = fg, size
	r.qty.Color, r.qty.TextSize = fg, size
	custom := c.custom()
	r.txt.Hidden = custom
	r.qty.Hidden = !(custom && c.col == colStock)
	r.bar.Hidden = !(custom && c.col == colStock)
	r.badge.Hidden = !(custom && c.col == colStatus)
	r.stars.Hidden = !(custom && c.col == colRating)
	if custom {
		switch c.col {
		case colStock:
			r.bar.SetQty(c.item.Quantity)
			r.qty.Text = strconv.Itoa(int(c.item.Quantity))
		case colStatus:
			r.badge.SetStatus(c.item.Status)
		case colRating:
			r.stars.SetRating(c.item.Rating)
		}
	} else {
		r.txt.Text = c.Text()
	}
}

func (r *tableCellRenderer) Layout(s fyne.Size) {
	c := r.c
	pad := themeSize(theme.SizeNamePadding) * 2
	r.sel.Move(fyne.NewPos(0, 0))
	r.sel.Resize(s)
	center := func(h float32) float32 { return (s.Height - h) / 2 }
	if c.custom() {
		switch c.col {
		case colStock:
			r.bar.Resize(r.bar.MinSize())
			r.bar.Move(fyne.NewPos(pad/2, center(r.bar.MinSize().Height)))
			x := pad/2 + stockBarW + pad/2
			txt := fitText(r.qty.Text, r.qty.TextSize, fyne.TextStyle{}, s.Width-x-2)
			r.qty.Text = txt
			ts := r.qty.MinSize()
			r.qty.Resize(ts)
			r.qty.Move(fyne.NewPos(x, center(ts.Height)))
		case colStatus:
			r.badge.Resize(r.badge.MinSize())
			r.badge.Move(fyne.NewPos(pad, center(r.badge.MinSize().Height)))
		case colRating:
			r.stars.Resize(r.stars.MinSize())
			r.stars.Move(fyne.NewPos(pad, center(r.stars.MinSize().Height)))
		}
		return
	}
	col := columns[0]
	if c.col >= 0 && c.col < len(columns) {
		col = columns[c.col]
	}
	if !c.loaded {
		col.align = fyne.TextAlignLeading
	}
	text := fitText(c.Text(), r.txt.TextSize, fyne.TextStyle{}, s.Width-2*pad)
	r.txt.Text = text
	ts := r.txt.MinSize()
	r.txt.Resize(ts)
	x := pad
	switch col.align {
	case fyne.TextAlignTrailing:
		x = s.Width - pad - ts.Width
	case fyne.TextAlignCenter:
		x = (s.Width - ts.Width) / 2
	}
	r.txt.Move(fyne.NewPos(x, center(ts.Height)))
}

func (r *tableCellRenderer) MinSize() fyne.Size { return r.c.MinSize() }

func (r *tableCellRenderer) Refresh() {
	r.update()
	r.Layout(r.c.Size())
	r.sel.Refresh()
	r.txt.Refresh()
	r.qty.Refresh()
	r.bar.Refresh()
	r.badge.Refresh()
	r.stars.Refresh()
}

func (r *tableCellRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.sel, r.txt, r.qty, r.bar, r.badge, r.stars}
}

func (r *tableCellRenderer) Destroy() {}

// ---------------------------------------------------------------- header

// headerCell is a tappable column header; tapping sorts by the column.
type headerCell struct {
	widget.Label
	col int
	tap func(col int)
}

func newHeaderCell(tap func(col int)) *headerCell {
	h := &headerCell{tap: tap, col: -1}
	h.Text = "Wy"
	h.TextStyle = fyne.TextStyle{Bold: true}
	h.Truncation = fyne.TextTruncateEllipsis
	h.ExtendBaseWidget(h)
	return h
}

func (h *headerCell) Tapped(*fyne.PointEvent) {
	if h.tap != nil && h.col >= 0 {
		h.tap(h.col)
	}
}

// ---------------------------------------------------------------- table

// gridTable is the data table: widget.Table with keyboard row movement and
// access to the scroll offset.
type gridTable struct {
	widget.Table
	u        *ui
	lastOffY float32
}

var tableOffsetField, tableOffsetOK = func() (uintptr, bool) {
	f, ok := reflect.TypeOf(widget.Table{}).FieldByName("offset")
	if !ok || f.Type != reflect.TypeOf(fyne.Position{}) {
		return 0, false
	}
	return f.Offset, true
}()

func newGridTable(u *ui) *gridTable {
	t := &gridTable{u: u}
	t.Length = func() (int, int) {
		n := u.pager.Total()
		if n < 0 {
			n = 0
		}
		return n, len(columns)
	}
	t.CreateCell = func() fyne.CanvasObject { return newTableCell() }
	t.UpdateCell = u.updateTableCell
	t.ShowHeaderRow = true
	t.CreateHeader = func() fyne.CanvasObject { return newHeaderCell(u.sortByColumn) }
	t.UpdateHeader = u.updateTableHeader
	t.OnSelected = func(id widget.TableCellID) {
		t.Unselect(id)
		u.selectIndex(id.Row)
	}
	t.ExtendBaseWidget(t)
	for i, c := range columns {
		t.SetColumnWidth(i, c.width)
	}
	return t
}

// offsetY is the vertical scroll offset. widget.Table has no getter, so the
// private field is read; if that is not possible the last offset set by
// this app is used.
func (t *gridTable) offsetY() float32 {
	if tableOffsetOK {
		p := (*fyne.Position)(unsafe.Add(unsafe.Pointer(&t.Table), tableOffsetField))
		return p.Y
	}
	return t.lastOffY
}

// pitch is the distance between two rows.
func (t *gridTable) pitch() float32 {
	return labelMin().Height + themeSize(theme.SizeNamePadding)
}

func (t *gridTable) bodyHeight() float32 {
	h := t.Size().Height - labelMin().Height
	if h <= 0 {
		h = 600
	}
	return h
}

// visibleRange returns the first and last row index currently on screen.
func (t *gridTable) visibleRange() (int, int) {
	p := t.pitch()
	off := t.offsetY()
	first := int(off / p)
	last := int((off+t.bodyHeight()-1)/p)
	if last < first {
		last = first
	}
	return first, last
}

func (t *gridTable) scrollToRow(row int) {
	if row < 0 {
		row = 0
	}
	t.lastOffY = float32(row) * t.pitch()
	t.ScrollToOffset(fyne.NewPos(0, t.lastOffY))
}

// ensureVisible scrolls the minimum amount that shows row.
func (t *gridTable) ensureVisible(row int) {
	first, last := t.visibleRange()
	rows := last - first
	if row < first {
		t.scrollToRow(row)
	} else if row > last-1 {
		t.scrollToRow(row - rows + 1)
	}
}

func (t *gridTable) TypedKey(ev *fyne.KeyEvent) {
	switch ev.Name {
	case fyne.KeyDown:
		t.u.moveSelection(1)
	case fyne.KeyUp:
		t.u.moveSelection(-1)
	case fyne.KeyPageDown:
		t.u.moveSelection(10)
	case fyne.KeyPageUp:
		t.u.moveSelection(-10)
	}
}

func (t *gridTable) ScrollToTop() {
	t.lastOffY = 0
	t.Table.ScrollToTop()
}
