package ui

import (
	"strconv"

	"github.com/doug/gophics/geom"
	"github.com/doug/gophics/layout"
	"github.com/doug/gophics/paint"
	"github.com/doug/gophics/shell"
	"github.com/doug/gophics/theme"
	"github.com/doug/gophics/widget"

	"gguix/core"
)

const resizeCursor = shell.CursorMove

// Column describes one table column.
type Column struct {
	Title string
	Field core.SortField
	Width float32
	Align float32 // 0 left, 0.5 center, 1 right
}

// Columns is the table layout (section 5.3).
var Columns = []Column{
	{"ID", core.SortID, 80, 1},
	{"SKU", core.SortSKU, 110, 0},
	{"Name", core.SortName, 260, 0},
	{"Category", core.SortCategory, 110, 0},
	{"Price", core.SortPrice, 100, 1},
	{"Qty", core.SortQuantity, 80, 1},
	{"Stock", core.SortQuantity, 120, 0},
	{"Status", core.SortStatus, 120, 0},
	{"Rating", core.SortRating, 110, 0},
	{"Updated", core.SortUpdated, 150, 0},
	{"Featured", core.SortFeatured, 80, 0.5},
}

func (s *App) colWidth(c int) float32 {
	if s.colW == nil {
		s.colW = make([]float32, len(Columns))
		for i, col := range Columns {
			s.colW[i] = col.Width
		}
	}
	return s.colW[c]
}

func (s *App) tableWidth() float32 {
	var w float32
	for i := range Columns {
		w += s.colWidth(i)
	}
	return w
}

// HeaderLabel is the header text, with the sort arrow on the active column.
func (s *App) HeaderLabel(c int) string {
	col := Columns[c]
	sort := s.Query.SortBy
	if sort == "" {
		sort = core.SortID
	}
	if col.Title != "Stock" && col.Field == sort {
		if s.Query.Desc {
			return col.Title + " ▼"
		}
		return col.Title + " ▲"
	}
	return col.Title
}

func cell(w, align float32, child widget.Widget) widget.Widget {
	return widget.Sized{W: w, Child: widget.Align{X: align, Y: 0.5,
		Child: widget.Padding{Insets: geom.InsetsSymmetric(7, 0), Child: child}}}
}

func (s *App) tableView(th theme.Theme) widget.Widget {
	headers := make([]widget.Widget, len(Columns))
	for c := range Columns {
		c := c
		label := widget.Interactive{
			Gestures: widget.Gestures{OnTap: s.act(func() {
				f := Columns[c].Field
				if Columns[c].Title == "Stock" {
					f = core.SortQuantity
				}
				s.HeaderClick(f)
			})},
			Child: widget.Sized{W: s.colWidth(c) - 6, H: headerH, Child: widget.Align{X: Columns[c].Align, Y: 0.5,
				Child: widget.Padding{Insets: geom.InsetsSymmetric(7, 0),
					Child: widget.Text{Value: s.HeaderLabel(c), Size: th.Type.Label, Color: th.Muted, Ellipsis: true}}}},
		}
		grip := widget.Interactive{
			Gestures: widget.Gestures{
				Cursor: resizeCursor, DragAxis: widget.DragHorizontal,
				OnDrag: func(_, d geom.Pt) {
					s.SetState(func() { s.colW[c] = max(s.colWidth(c)+d.X, 40) })
				},
			},
			Child: widget.Sized{W: 6, H: headerH, Child: widget.Align{X: 1, Y: 0.5,
				Child: widget.Decorated{Color: th.Border, Child: widget.Sized{W: 1, H: 16}}}},
		}
		headers[c] = widget.Row(label, grip)
	}
	header := widget.Row(headers...)

	total := s.Pager.Total()
	var body widget.Widget
	switch {
	case total == 0:
		body = widget.Center(widget.Text{Value: "No items match.", Size: th.Type.Body, Color: th.Muted})
	default:
		body = widget.LazyList{
			Count:           max(total, 0),
			EstimatedExtent: TableRowH,
			Controller:      s.tableCtl,
			Build: func(i int) widget.Widget {
				s.syncVisible()
				return s.tableRow(th, i)
			},
		}
	}
	// The vertical list spans the visible width so its scrollbar sits at the
	// visible right edge (fixpass issue 2). Header and rows are shifted by the
	// horizontal offset, which a separate horizontal Scroll strip at the bottom
	// owns (its scrollbar is the table's horizontal bar).
	tw := s.tableWidth()
	hstrip := widget.Scroll{Axis: layout.Horizontal, Controller: s.hCtl,
		OnOffset: func(off, _ float32) {
			if off != s.hOff {
				s.SetState(func() { s.hOff = off })
			}
			if s.hdrCtl.Offset() != off {
				s.hdrCtl.JumpTo(off)
			}
		},
		Child: widget.Sized{W: tw, H: hBarH}}
	col := widget.Column(
		// The header sits in its own horizontal Scroll so its viewport clips
		// it to the table pane (fix pass 2: it bled over the form on WebGPU).
		// Its offset and the strip's are kept in step both ways.
		widget.Scroll{Axis: layout.Horizontal, Controller: s.hdrCtl,
			OnOffset: func(off, _ float32) {
				if s.hCtl.Offset() != off {
					s.hCtl.JumpTo(off)
				}
			},
			Child: widget.Sized{W: tw, Child: header}},
		widget.Fill{Color: th.Border, Child: widget.Sized{H: 1}},
		widget.Expand(widget.LayoutObserver{OnLayout: s.onViewport, Child: body}),
		hstrip,
	)
	col.CrossAlign = layout.CrossStretch
	return col
}

// hBarH is the height of the horizontal scroll strip under the table.
const hBarH = 10

// hShift offsets a full-width table row by the horizontal scroll offset; the
// enclosing viewport clips what falls outside.
func (s *App) hShift(child widget.Widget) widget.Widget {
	return widget.Transform{T: paint.Transform{TX: -s.hOff}, Child: child}
}

func (s *App) onViewport(r geom.Rect) {
	s.listTop = r.Min.Y
	if r.Dy() > 0 && r.Dy() != s.viewportH {
		s.SetState(func() {
			s.viewportH = r.Dy()
			s.setVisibleFrom(s.First)
		})
	}
}

func (s *App) tableRow(th theme.Theme, i int) widget.Widget {
	it, ok := s.Pager.Row(i)
	cells := make([]widget.Widget, len(Columns))
	for c, col := range Columns {
		w := s.colWidth(c)
		if !ok {
			cells[c] = cell(w, col.Align, widget.Text{Value: "…", Size: th.Type.Label, Color: th.Muted})
			continue
		}
		cells[c] = cell(w, col.Align, s.cellContent(th, c, it))
	}
	var bg paint.Color
	if ok && it.ID == s.SelectedID {
		bg = th.Selection // the theme's selection color
	}
	r := widget.Row(cells...)
	r.CrossAlign = layout.CrossStretch
	return widget.Interactive{
		Gestures: widget.Gestures{OnTap: s.act(func() { s.SelectRow(i) })},
		Child: widget.Decorated{Color: bg, Child: widget.Sized{H: TableRowH,
			Child: s.hShift(r)}},
	}
}

func (s *App) cellContent(th theme.Theme, c int, it core.Item) widget.Widget {
	text := func(v string) widget.Widget {
		return widget.Text{Value: v, Size: th.Type.Label, Color: th.Text, Ellipsis: true}
	}
	pal := core.Colors(th.Dark)
	switch Columns[c].Title {
	case "ID":
		return text(strconv.FormatInt(it.ID, 10))
	case "SKU":
		return text(it.SKU)
	case "Name":
		return text(it.Name)
	case "Category":
		return text(it.Category.String())
	case "Price":
		return text(core.FormatPrice(it.PriceCents))
	case "Qty":
		return text(strconv.Itoa(int(it.Quantity)))
	case "Stock":
		return row(StockBar{Qty: it.Quantity, Pal: pal}, gapW(6), text(strconv.Itoa(int(it.Quantity))))
	case "Status":
		return StatusBadge{Status: it.Status, Pal: pal, Size: th.Type.Caption}
	case "Rating":
		return row(Stars{Rating: it.Rating, Pal: pal}, gapW(4), text(core.FormatRating(it.Rating)))
	case "Updated":
		return text(core.FormatUpdated(it.UpdatedAt))
	case "Featured":
		if it.Featured {
			return text("✓")
		}
		return text("")
	}
	return nil
}
