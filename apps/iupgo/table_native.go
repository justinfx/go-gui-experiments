//go:build !matrix

package main

import (
	"fmt"
	"image"
	"sort"
	"strconv"

	"github.com/gen2brain/iup-go/iup"

	"gguix/core"
)

// tableView is the state around the IupTable.
type tableView struct {
	ih    iup.Ihandle
	total int
	// seen collects the rows VALUE_CB asked for since the last flush; the
	// table has no scroll position attribute, so the visible range is
	// inferred from these requests.
	seen   []int
	flush  iup.Ihandle
	full   bool // next flush follows a full REDRAW, so it measures the whole view
	hint   int  // row just scrolled to, -1 none
	winLo  int
	winHi  int
	imgs   map[imgKey]*cellImage
	selLin int
	// Row cache for VALUE_CB: the 11 cells of a row are asked for one after
	// another, so the pager is consulted once per row.
	rowIdx int
	rowIt  core.Item
	rowOK  bool
	// loadedAtDraw is true when every visible row was loaded at the last
	// REDRAW, so a later pager change that touches no visible row can skip it.
	loadedAtDraw          bool
	drawnFirst, drawnLast int
}

// cellImage is an image shown in a table cell, kept with the data it was
// made from so tests can check what is on screen.
type cellImage struct {
	name string
	kind string // "bar", "badge", "stars"
	img  *image.NRGBA
	ih   iup.Ihandle
	spec cellSpec
}

func (a *app) buildTable() {
	t := &a.ui.tbl
	t.hint = -1
	t.imgs = map[imgKey]*cellImage{}
	t.rowIdx = -1
	t.ih = a.newTableWidget()
	t.flush = iup.Timer()
	set(t.flush, "TIME", "25")
	iup.On(t.flush, iup.TimerActionCB, func(ih iup.Ihandle) int {
		set(ih, "RUN", "NO")
		a.tableFlush()
		return iup.DEFAULT
	})
}

// newTableWidget creates a configured, unmapped IupTable.
func (a *app) newTableWidget() iup.Ihandle {
	ih := iup.Table()
	set(ih, "VIRTUALMODE", "YES", "NUMCOL", strconv.Itoa(len(tableCols)), "NUMLIN", "0",
		"SHOWIMAGE", "YES", "FITIMAGE", "NO", "USERRESIZE", "YES", "SORTABLE", "YES",
		"SELECTIONMODE", "SINGLE", "STRETCHLAST", "NO", "FOCUSFEEDBACK", "NO",
		"VISIBLECOLUMNS", "3", "VISIBLELINES", "10", "EXPAND", "YES")
	for i, c := range tableCols {
		iup.SetAttributeId(ih, "TITLE", i+1, headerTitle(i, a))
		iup.SetAttributeId(ih, "WIDTH", i+1, c.width)
		iup.SetAttributeId(ih, "ALIGNMENT", i+1, c.align)
	}
	iup.On(ih, iup.TableValueCB, func(ih iup.Ihandle, lin, col int) string { return a.tableValue(lin-1, col) })
	iup.On(ih, iup.TableImageCB, func(ih iup.Ihandle, lin, col int) string { return a.tableImage(lin-1, col) })
	iup.On(ih, iup.TableSortCB, func(ih iup.Ihandle, col int) int {
		if col >= 1 && col <= len(tableCols) {
			a.setSort(tableCols[col-1].sort)
		}
		return iup.IGNORE
	})
	iup.On(ih, iup.EnterItemCB, func(ih iup.Ihandle, lin, col int) int {
		if !a.ui.inSel && lin-1 != a.selRow {
			a.requestSelect(lin - 1)
		}
		return iup.DEFAULT
	})
	return ih
}

// shrinkLimit is how many rows a NUMLIN decrease may remove in place.
// Removing rows from a native table is slow (about 3.4 us per row on Cocoa,
// 3.4 s for 1,000,000 rows, measured here), while adding rows and creating
// a table are cheap, so a large decrease swaps in a fresh table instead.
const shrinkLimit = 5000

// tableReplace swaps the table widget for a new one with n rows.
func (a *app) tableReplace(n int) {
	u := &a.ui
	t := &u.tbl
	old := t.ih
	nt := a.newTableWidget()
	for i := range tableCols {
		if w := iup.GetIntId(old, "WIDTH", i+1); w > 0 {
			iup.SetAttributeId(nt, "WIDTH", i+1, w)
		}
	}
	set(nt, "NUMLIN", strconv.Itoa(n))
	iup.Insert(u.zbox, u.crd.ih, nt)
	iup.Map(nt)
	t.ih = nt
	t.selLin = 0
	iup.SetHandle("gxtable", nt)
	a.uiEmptyPage()
	iup.Detach(old)
	iup.Destroy(old)
	iup.Refresh(u.zbox)
}

// tableValue is VALUE_CB: the text of one cell, taken from the pager.
func (a *app) tableValue(row, col int) string {
	it, ok := a.tableRow(row)
	if !ok {
		return placeholder
	}
	return cellText(it, col)
}

// tableRow fetches a row from the pager once for all its cells.
func (a *app) tableRow(row int) (core.Item, bool) {
	t := &a.ui.tbl
	if row != t.rowIdx {
		t.rowIdx = row
		t.rowIt, t.rowOK = a.pager.Row(row)
		a.tableSeen(row)
	}
	return t.rowIt, t.rowOK
}

// tableImage is IMAGE_CB: the image of one cell ("" for none).
func (a *app) tableImage(row, col int) string {
	if col != colStock && col != colStatus && col != colRating {
		return ""
	}
	it, ok := a.tableRow(row)
	if !ok {
		return ""
	}
	if ci := a.cellImageFor(it, col); ci != nil {
		return ci.name
	}
	return ""
}

// imgKey identifies a cell image without building a string.
type imgKey struct {
	kind byte // 's' stock, 'b' badge, 'r' rating
	dark bool
	a, b int
}

// cellImageFor returns (creating on first use) the registered image for a
// custom cell. The cache is keyed by value so the hot path allocates nothing.
func (a *app) cellImageFor(it core.Item, col int) *cellImage {
	var k imgKey
	k.dark = a.dark
	switch col {
	case colStock:
		k.kind, k.a = 's', min(int(it.Quantity), 1000)
	case colStatus:
		k.kind, k.a = 'b', int(it.Status)
	case colRating:
		f, h, _ := core.Stars(it.Rating)
		k.kind, k.a, k.b = 'r', f, h
	default:
		return nil
	}
	t := &a.ui.tbl
	if ci, ok := t.imgs[k]; ok {
		return ci
	}
	sp := a.cellSpecFor(it, col)
	theme := "l"
	if a.dark {
		theme = "d"
	}
	var name string
	var im *image.NRGBA
	switch k.kind {
	case 's':
		name = fmt.Sprintf("gxbar-%s-%d", theme, k.a)
		im = renderBar(sp.Fraction, core.HexColor(sp.Track), core.HexColor(sp.Fill))
	case 'b':
		name = fmt.Sprintf("gxbadge-%s-%d", theme, k.a)
		im = renderBadge(core.HexColor(sp.BadgeBG), core.HexColor(sp.BadgeFG))
	default:
		name = fmt.Sprintf("gxstars-%s-%d-%d", theme, k.a, k.b)
		im = renderStars(sp.Full, sp.Half, sp.Empty, core.HexColor(sp.StarFilled), core.HexColor(sp.StarEmpty))
	}
	ih := iup.ImageRGBA(im.Bounds().Dx(), im.Bounds().Dy(), im.Pix)
	iup.SetHandle(name, ih)
	ci := &cellImage{name: name, kind: sp.Kind, img: im, ih: ih, spec: sp}
	t.imgs[k] = ci
	return ci
}

// ---- visible range inference -------------------------------------------

// tableSeen records a VALUE_CB request. IupTable has no scroll position and
// no scroll callback, but the native table asks for exactly the cells it
// draws, so a short timer turns the requests into the visible range.
func (a *app) tableSeen(row int) {
	t := &a.ui.tbl
	if n := len(t.seen); n > 0 && t.seen[n-1] == row {
		return
	}
	if len(t.seen) < 8192 {
		t.seen = append(t.seen, row)
	}
	if len(t.seen) == 1 {
		set(t.flush, "RUN", "YES")
	}
}

func (a *app) tableFlush() {
	t := &a.ui.tbl
	if len(t.seen) == 0 || a.viewMode != 0 {
		t.seen = t.seen[:0]
		return
	}
	rows := append([]int(nil), t.seen...)
	t.seen = t.seen[:0]
	sort.Ints(rows)
	type span struct{ lo, hi int }
	var cl []span
	for _, r := range rows {
		if n := len(cl); n > 0 && r-cl[n-1].hi <= 2 {
			cl[n-1].hi = r
		} else if n == 0 || r != cl[n-1].hi {
			cl = append(cl, span{r, r})
		}
	}
	best := cl[0]
	for _, c := range cl[1:] {
		if t.hint >= 0 {
			if c.lo <= t.hint+a.visH && c.hi >= t.hint && !(best.lo <= t.hint+a.visH && best.hi >= t.hint) {
				best = c
			}
			continue
		}
		if c.hi-c.lo > best.hi-best.lo {
			best = c
		}
	}
	t.hint = -1
	total := a.pager.Total()
	lo, hi := best.lo, best.hi
	span1 := hi - lo + 1
	switch {
	case t.full:
		t.full = false
		a.visH = max(span1, 1)
	case lo <= a.visLast+1 && hi >= a.visFirst-1:
		// Scrolled by a few rows: keep the height, follow the new edge.
		if lo < a.visFirst {
			hi = min(a.visLast, lo+a.visH-1)
		} else if hi > a.visLast {
			lo = max(a.visFirst, hi-a.visH+1)
		} else {
			lo, hi = a.visFirst, a.visLast
		}
	default:
		a.visH = max(a.visH, span1)
		hi = lo + a.visH - 1
	}
	if total > 0 {
		hi = min(hi, total-1)
	}
	a.setVisible(lo, hi)
}

func (a *app) tableSetTotal(total int) {
	t := &a.ui.tbl
	n := max(total, 0)
	if n == t.total && total >= 0 {
		return
	}
	if n < t.total-shrinkLimit {
		t.total = n
		t.seen = t.seen[:0]
		a.tableReplace(n)
		return
	}
	t.total = n
	t.seen = t.seen[:0]
	set(t.ih, "NUMLIN", strconv.Itoa(n))
}

func (a *app) tableRedraw(force bool) {
	t := &a.ui.tbl
	if a.viewMode != 0 {
		return
	}
	// A pager change that touches no visible row needs no redraw: the last
	// redraw already showed every visible row loaded.
	loaded := a.ui.total > 0 && a.pager.RangeLoaded(a.visFirst, min(a.visLast, a.ui.total-1))
	if !force && loaded && t.loadedAtDraw && t.drawnFirst == a.visFirst && t.drawnLast == a.visLast {
		return
	}
	t.loadedAtDraw, t.drawnFirst, t.drawnLast = loaded, a.visFirst, a.visLast
	if force {
		t.loadedAtDraw = false // the data behind loaded rows may change next
	}
	t.rowIdx = -1
	t.full = true
	set(t.ih, "REDRAW", "YES")
}

// tableScrollTo makes row the top visible row. IupTable only offers SHOW,
// which scrolls the minimum distance, so the row below the view is shown
// first; showing the target then moves it to the top edge.
func (a *app) tableScrollTo(row int) {
	t := &a.ui.tbl
	total := a.ui.total
	if total <= 0 {
		return
	}
	if row <= a.visFirst {
		// Moving up: the minimum-distance scroll puts the row at the top.
		set(t.ih, "SHOW", fmt.Sprintf("%d:1", row+1))
	} else {
		// Moving down: show the row that should be last in the view, which
		// puts the target at the top. One scroll instead of two halves the
		// layout work per step; the range inference corrects a one row error.
		set(t.ih, "SHOW", fmt.Sprintf("%d:1", min(total, row+a.visH)))
	}
	t.seen = t.seen[:0]
	t.hint = row
	a.visFirst, a.visLast = row, min(row+a.visH-1, total-1)
	a.pager.SetVisible(a.visFirst, a.visLast)
	set(t.flush, "RUN", "NO")
}

// tableSelect shows row as the selected line (row < 0 clears it).
func (a *app) tableSelect(row int) {
	t := &a.ui.tbl
	u := &a.ui
	u.inSel = true
	defer func() { u.inSel = false }()
	if row < 0 {
		if t.selLin > 0 {
			iup.SetAttributeId(t.ih, "SELECTED", t.selLin, "NO")
			t.selLin = 0
		}
		return
	}
	t.selLin = row + 1
	set(t.ih, "FOCUSCELL", fmt.Sprintf("%d:", row+1))
}

// tableTheme re-applies what depends on the theme (the cell images are
// registered per theme, so only a redraw is needed).
func (a *app) tableTheme() {}

// sortFromHeader is what SORT_CB does for a header click.
func (t *tableView) sortFromHeader(a *app, col int) {
	a.setSort(tableCols[col-1].sort)
}

// prewarmCellImages creates the badge and star images before the window is
// mapped (there are only 3 and 11 of them per theme).
func (a *app) prewarmCellImages() {
	for _, s := range core.AllStatuses {
		a.cellImageFor(core.Item{Status: s}, colStatus)
	}
	for r := 0; r <= 10; r++ {
		a.cellImageFor(core.Item{Rating: float32(r) / 2}, colRating)
	}
}

// tableInvalidate drops the cached row and the "already drawn" state after a
// query or data change.
func (a *app) tableInvalidate() {
	a.ui.tbl.rowIdx = -1
	a.ui.tbl.loadedAtDraw = false
}

// tableHeaders shows the sort arrow in the column titles.
func (a *app) tableHeaders() {
	for i := range tableCols {
		iup.SetAttributeId(a.ui.tbl.ih, "TITLE", i+1, headerTitle(i, a))
	}
}

// tableCellText is the text a cell shows (used by the tests).
func (a *app) tableCellText(row, col int) string {
	return iup.GetAttributeId2(a.ui.tbl.ih, "IDVALUE", row+1, col)
}

// tableHeaderTitle is the title of column col (1-based).
func (a *app) tableHeaderTitle(col int) string {
	return iup.GetAttributeId(a.ui.tbl.ih, "TITLE", col)
}

// tableSortClick is what a click on a column header does.
func (a *app) tableSortClick(col int) { a.setSort(tableCols[col-1].sort) }
