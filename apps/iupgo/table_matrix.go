//go:build matrix && !js

package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/gen2brain/iup-go/iup"

	"gguix/core"
)

// The Matrix variant of the table (build tags `ctrl,matrix`): an IupMatrixEx
// in callback mode. Unlike IupTable it reports its scroll position
// (SCROLLTOP_CB, ORIGIN), can be scrolled to an exact top row (ORIGIN) and
// lets the app draw the cell (DRAW_CB), so the stock bar, the status pill
// with its text and the stars are drawn directly, without images. It is a
// drawn control, not a native table.

// tableView is the state around the IupMatrixEx.
type tableView struct {
	ih     iup.Ihandle
	total  int
	top    int // 0-based top body row
	rowH   int // body row height in pixels
	hdrH   int // title row height
	keyNav bool
	// Row cache for VALUE_CB and DRAW_CB.
	rowIdx int
	rowIt  core.Item
	rowOK  bool
	fonts  fontSet
}

const matrixRowH = 24
const matrixHdrH = 26

func (a *app) buildTable() {
	t := &a.ui.tbl
	t.rowIdx = -1
	t.rowH, t.hdrH = matrixRowH, matrixHdrH
	t.ih = a.newMatrixWidget()
}

// newMatrixWidget creates a configured, unmapped IupMatrixEx.
func (a *app) newMatrixWidget() iup.Ihandle {
	ih := iup.MatrixEx()
	set(ih, "NUMCOL", strconv.Itoa(len(tableCols)), "NUMLIN", "0", "NUMCOL_VISIBLE", "3", "NUMLIN_VISIBLE", "10",
		"EXPAND", "YES", "RESIZEMATRIX", "YES", "HIDEFOCUS", "YES", "FLAT", "NO", "SCROLLBAR", "YES",
		"USETITLESIZE", "NO", "WIDTH0", "0")
	iup.SetAttributeId(ih, "RASTERHEIGHT", 0, matrixHdrH)
	iup.SetAttributeId(ih, "RASTERWIDTH", 0, 0)
	set(ih, "RASTERHEIGHTDEF", strconv.Itoa(matrixRowH), "HEIGHTDEF", "")
	for i, c := range tableCols {
		iup.SetAttributeId(ih, "RASTERWIDTH", i+1, c.width)
		al := "ALEFT"
		switch c.align {
		case "ARIGHT":
			al = "ARIGHT"
		case "ACENTER":
			al = "ACENTER"
		}
		iup.SetAttributeId(ih, "ALIGNMENT", i+1, al)
	}
	iup.On(ih, iup.MatrixValueCB, func(ih iup.Ihandle, lin, col int) string { return a.matrixValue(lin, col) })
	iup.On(ih, iup.MatrixDrawCB, func(ih iup.Ihandle, lin, col, x1, x2, y1, y2 int) int {
		return a.matrixDraw(lin, col, x1, x2, y1, y2)
	})
	iup.On(ih, iup.ClickCB, func(ih iup.Ihandle, lin, col int, status string) int {
		switch {
		case lin == 0 && col >= 1:
			a.setSort(tableCols[col-1].sort)
		case lin >= 1 && !a.ui.inSel && lin-1 != a.selRow:
			a.requestSelect(lin - 1)
		}
		return iup.DEFAULT
	})
	// The matrix also calls ENTERITEM_CB when it gets the focus, so only a
	// move made with the keyboard selects.
	iup.On(ih, iup.KAnyCB, func(ih iup.Ihandle, k int) int {
		switch k {
		case iup.K_UP, iup.K_DOWN, iup.K_PGUP, iup.K_PGDN:
			a.ui.tbl.keyNav = true
		}
		return iup.CONTINUE
	})
	iup.On(ih, iup.EnterItemCB, func(ih iup.Ihandle, lin, col int) int {
		if a.ui.tbl.keyNav && lin >= 1 {
			a.ui.tbl.keyNav = false
			if !a.ui.inSel && lin-1 != a.selRow {
				a.requestSelect(lin - 1)
			}
		}
		return iup.DEFAULT
	})
	iup.On(ih, iup.ScrollTopCB, func(ih iup.Ihandle, lin, col int) int {
		a.ui.tbl.top = max(lin-1, 0)
		a.matrixRange()
		return iup.DEFAULT
	})
	iup.On(ih, iup.ResizeMatrixCB, func(ih iup.Ihandle, w, h int) int {
		a.matrixRange()
		return iup.DEFAULT
	})
	return ih
}

// matrixReplace swaps in a fresh matrix with n lines at the top. Scrolling a
// deep matrix back to the top and shrinking a large one are slow in place
// (measured about 0.4 s for 500,000 to 1,000,000 lines), a new one is cheap.
func (a *app) matrixReplace(n int) {
	u := &a.ui
	t := &u.tbl
	old := t.ih
	nt := a.newMatrixWidget()
	for i := range tableCols {
		if w := iup.GetIntId(old, "RASTERWIDTH", i+1); w > 0 {
			iup.SetAttributeId(nt, "RASTERWIDTH", i+1, w)
		}
	}
	set(nt, "NUMLIN", strconv.Itoa(n))
	iup.Insert(u.zbox, u.crd.ih, nt)
	iup.Map(nt)
	t.ih = nt
	t.top, t.rowIdx = 0, -1
	a.uiEmptyPage()
	iup.Detach(old)
	iup.Destroy(old)
	iup.Refresh(u.zbox)
}

// matrixRange derives the visible rows from the top row and the control
// height, and tells the pager.
func (a *app) matrixRange() {
	t := &a.ui.tbl
	if a.viewMode != 0 {
		return
	}
	_, _, h := iup.GetInt2(t.ih, "RASTERSIZE")
	if h <= 0 {
		return
	}
	n := max((h-t.hdrH+t.rowH-1)/t.rowH, 1)
	a.visH = n
	last := t.top + n - 1
	if a.ui.total > 0 {
		last = min(last, a.ui.total-1)
	}
	a.setVisible(t.top, last)
}

func (a *app) tableRow(row int) (core.Item, bool) {
	t := &a.ui.tbl
	if row != t.rowIdx {
		t.rowIdx = row
		t.rowIt, t.rowOK = a.pager.Row(row)
	}
	return t.rowIt, t.rowOK
}

// matrixValue is VALUE_CB: titles (line 0) and the cell text.
func (a *app) matrixValue(lin, col int) string {
	if col < 1 {
		return ""
	}
	if lin == 0 {
		return headerTitle(col-1, a)
	}
	it, ok := a.tableRow(lin - 1)
	if !ok {
		return placeholder
	}
	return cellText(it, col)
}

func (a *app) tableValue(row, col int) string { return a.matrixValue(row+1, col) }

// matrixDraw is DRAW_CB. It paints the selection and the custom cells; for
// the text columns it returns IGNORE so the matrix draws the text itself.
func (a *app) matrixDraw(lin, col, x1, x2, y1, y2 int) int {
	if lin < 1 || col < 1 {
		return iup.IGNORE
	}
	t := &a.ui.tbl
	ih := t.ih
	row := lin - 1
	surface := globalRGB("TXTBGCOLOR")
	if row == a.selRow {
		sel := mixRGB(globalRGB("ACCENTCOLOR"), surface, 0.35)
		set(ih, "DRAWSTYLE", "FILL", "DRAWCOLOR", rgbStr(sel))
		iup.DrawRectangle(ih, x1, y1, x2, y2)
	}
	it, ok := a.tableRow(row)
	if !ok || (col != colStock && col != colStatus && col != colRating) {
		return iup.IGNORE
	}
	sp := a.cellSpecFor(it, col)
	fg := globalRGB("TXTFGCOLOR")
	cy := (y1 + y2) / 2
	switch col {
	case colStock:
		bx := x1 + 4
		by := cy - barH/2
		set(ih, "DRAWSTYLE", "FILL", "DRAWCOLOR", hexRGB(sp.Track))
		iup.DrawRoundedRectangle(ih, bx, by, bx+barW-1, by+barH-1, 4)
		if fw := int(math.Round(sp.Fraction * barW)); fw > 0 {
			set(ih, "DRAWCOLOR", hexRGB(sp.Fill))
			iup.DrawRoundedRectangle(ih, bx, by, bx+fw-1, by+barH-1, 4)
		}
		a.matrixText(ih, sp.Label, bx+barW+8, cy, fg)
	case colStatus:
		tw, th := iup.DrawGetTextSize(ih, sp.Text)
		pw, ph := tw+16, 18
		px, py := x1+4, cy-ph/2
		set(ih, "DRAWSTYLE", "FILL", "DRAWCOLOR", hexRGB(sp.BadgeBG))
		iup.DrawRoundedRectangle(ih, px, py, px+pw-1, py+ph-1, 9)
		set(ih, "DRAWFONT", "", "DRAWCOLOR", hexRGB(sp.BadgeFG), "DRAWTEXTALIGNMENT", "ALEFT")
		iup.DrawText(ih, sp.Text, px+8, cy-th/2, tw, th)
	case colRating:
		drawStarsAt(ih, sp.Full, sp.Half, sp.StarFilled, sp.StarEmpty, x1+4, cy-starSize/2)
		a.matrixText(ih, sp.Text, x1+4+starsW+6, cy, fg)
	}
	return iup.DEFAULT
}

func (a *app) matrixText(ih iup.Ihandle, s string, x, cy int, c [3]uint8) {
	tw, th := iup.DrawGetTextSize(ih, s)
	set(ih, "DRAWFONT", "", "DRAWCOLOR", rgbStr(c), "DRAWTEXTALIGNMENT", "ALEFT")
	iup.DrawText(ih, s, x, cy-th/2, tw, th)
}

func (a *app) tableSetTotal(total int) {
	t := &a.ui.tbl
	n := max(total, 0)
	if n == t.total && total >= 0 {
		return
	}
	shrink := n < t.total-5000
	t.total = n
	t.rowIdx = -1
	t.top = 0
	if shrink {
		a.matrixReplace(n)
	} else {
		set(t.ih, "NUMLIN", strconv.Itoa(n))
	}
	a.matrixRange()
}

func (a *app) tableRedraw(force bool) {
	if a.viewMode != 0 {
		return
	}
	a.ui.tbl.rowIdx = -1
	set(a.ui.tbl.ih, "REDRAW", "ALL")
}

// tableScrollTo makes row the top row: ORIGIN scrolls exactly.
func (a *app) tableScrollTo(row int) {
	t := &a.ui.tbl
	if a.ui.total <= 0 {
		return
	}
	if row == 0 && t.top > 2000 {
		a.matrixReplace(a.ui.total)
		a.matrixRange()
		return
	}
	t.top = row
	set(t.ih, "ORIGIN", fmt.Sprintf("%d:1", row+1))
	a.matrixRange()
}

// tableSelect shows row as the selected row.
func (a *app) tableSelect(row int) {
	u := &a.ui
	t := &u.tbl
	u.inSel = true
	defer func() { u.inSel = false }()
	t.rowIdx = -1
	if row >= 0 {
		set(t.ih, "FOCUSCELL", fmt.Sprintf("%d:1", row+1))
	}
	set(t.ih, "REDRAW", "ALL")
}

func (a *app) tableTheme() {}

func (a *app) prewarmCellImages() {}

func (a *app) tableInvalidate() { a.ui.tbl.rowIdx = -1 }

func (a *app) tableHeaders() { set(a.ui.tbl.ih, "REDRAW", "ALL") }

func (a *app) tableCellText(row, col int) string { return a.matrixValue(row+1, col) }

func (a *app) tableHeaderTitle(col int) string { return a.matrixValue(0, col) }

func (a *app) tableSortClick(col int) { a.setSort(tableCols[col-1].sort) }

var _ = strings.TrimSpace
