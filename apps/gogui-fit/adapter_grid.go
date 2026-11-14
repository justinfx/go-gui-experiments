package main

// adapter_grid.go connects go-gui's DataGrid to rowStore through a
// datagrid.DataGridDataSource. DataGrid's source mode is offset paging:
// the grid holds one page of rows, virtualizes the rows of that page and
// shows its own pager row (prev/next, "Rows a-b/total", jump) for the
// rest. There is no continuous scroll over the whole result.

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-gui-org/go-gui/gui"
	"github.com/go-gui-org/go-gui/gui/datagrid"

	"gguix/core"
)

// Column IDs. The sort field of each sortable column is in columns.
var gridColIDs = []string{"id", "sku", "name", "category", "price", "qty", "stock", "status", "rating", "updated", "featured"}

// qColID carries the non-sort part of the core query (search, filters)
// plus a refresh revision through GridQueryState, so the grid's own
// query signature changes exactly when the core query does.
const qColID = "__q"

// gridSource is the DataGridDataSource. It translates the grid's request
// back into a core.Query and asks the store for the page.
type gridSource struct{ st *rowStore }

func (g *gridSource) Capabilities() datagrid.GridDataCapabilities {
	return datagrid.GridDataCapabilities{SupportsOffsetPagination: true, SupportsNumberedPages: true, RowCountKnown: true}
}

func (g *gridSource) MutateData(datagrid.GridMutationRequest) (datagrid.GridMutationResult, error) {
	return datagrid.GridMutationResult{RowCount: -1}, nil
}

func (g *gridSource) FetchData(req datagrid.GridDataRequest) (datagrid.GridDataResult, error) {
	start, end := 0, gridPageRows
	if p, ok := req.Page.(datagrid.GridOffsetPageReq); ok {
		start, end = p.StartIndex, p.EndIndex
	}
	end = min(end, start+gridPageRows)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		t := time.NewTicker(10 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if req.Signal.IsAborted() {
					cancel()
					return
				}
			}
		}
	}()
	items, total, err := g.st.Fetch(ctx, coreQueryOf(req.Query), start, end, req.RequestID)
	if err != nil {
		return datagrid.GridDataResult{}, err
	}
	rows := make([]datagrid.GridRow, len(items))
	for i, it := range items {
		rows[i] = gridRow(it)
	}
	return datagrid.GridDataResult{Rows: rows, RowCount: total, ReceivedCount: len(rows), HasMore: start+len(rows) < total}, nil
}

// gridRow is the cell text of one item. Stock, status and rating are the
// text fallbacks for the custom cells DataGrid cannot draw.
func gridRow(it core.Item) datagrid.GridRow {
	feat := ""
	if it.Featured {
		feat = "✓"
	}
	return datagrid.GridRow{ID: strconv.FormatInt(it.ID, 10), Cells: map[string]string{
		"id":       strconv.FormatInt(it.ID, 10),
		"sku":      it.SKU,
		"name":     it.Name,
		"category": it.Category.String(),
		"price":    core.FormatPrice(it.PriceCents),
		"qty":      strconv.Itoa(int(it.Quantity)),
		"stock":    stockText(it.Quantity),
		"status":   it.Status.String(),
		"rating":   starsText(it.Rating),
		"updated":  core.FormatUpdated(it.UpdatedAt),
		"featured": feat,
	}}
}

// stockText is the stock bar as text: 8 block cells filled by
// StockFraction, then the quantity.
func stockText(q int32) string {
	n := int(core.StockFraction(q)*8 + 0.5)
	return strings.Repeat("█", n) + strings.Repeat("░", 8-n) + " " + strconv.Itoa(int(q))
}

// starsText is the five stars as glyphs (a half star is ⯪) and FormatRating.
func starsText(r float32) string {
	f, h, e := core.Stars(r)
	return strings.Repeat("★", f) + strings.Repeat("⯪", h) + strings.Repeat("☆", e) + " " + core.FormatRating(r)
}

// gridQueryOf is the GridQueryState the grid is given for a core query.
func gridQueryOf(q core.Query, rev int) datagrid.GridQueryState {
	gq := datagrid.GridQueryState{}
	if q.SortBy != "" {
		dir := datagrid.GridSortAsc
		if q.Desc {
			dir = datagrid.GridSortDesc
		}
		gq.Sorts = []datagrid.GridSort{{ColID: colForSort(q.SortBy), Dir: dir}}
	}
	vals := core.Query{Search: q.Search, Categories: q.Categories, Status: q.Status}.Values()
	vals.Set("rev", strconv.Itoa(rev))
	gq.Filters = []datagrid.GridFilter{{ColID: qColID, Op: "equals", Value: vals.Encode()}}
	return gq
}

// coreQueryOf reverses gridQueryOf (in the fetch goroutine).
func coreQueryOf(gq datagrid.GridQueryState) core.Query {
	var q core.Query
	for _, f := range gq.Filters {
		if f.ColID == qColID {
			if v, err := url.ParseQuery(f.Value); err == nil {
				v.Del("rev")
				if qq, err := core.QueryFromValues(v); err == nil {
					q = qq
				}
			}
		}
	}
	q.SortBy, q.Desc = core.SortID, false
	if len(gq.Sorts) > 0 {
		if f, ok := sortForCol(gq.Sorts[0].ColID); ok {
			q.SortBy, q.Desc = f, gq.Sorts[0].Dir == datagrid.GridSortDesc
		}
	}
	return q
}

func colForSort(f core.SortField) string {
	for i, c := range columns {
		if c.field == f && c.title != "Stock" {
			return gridColIDs[i]
		}
	}
	return "id"
}

func sortForCol(id string) (core.SortField, bool) {
	for i, c := range gridColIDs {
		if c == id && columns[i].title != "Stock" {
			return columns[i].field, true
		}
	}
	return "", false
}

// gridColumns builds the DataGrid columns. fill is the width the last
// column gets so the columns fill the pane (DataGrid has no stretch
// column; its widths are cached once, so the last column is pinned to
// the fill width through MinWidth = MaxWidth).
func gridColumns(fill float32) []datagrid.GridColumnCfg {
	out := make([]datagrid.GridColumnCfg, len(columns))
	for i, c := range columns {
		w := c.w
		col := datagrid.GridColumnCfg{
			ID: gridColIDs[i], Title: c.title, Width: gui.SomeF(w), MinWidth: gui.SomeF(50),
			Resizable: true, Sortable: c.title != "Stock", Align: c.align,
		}
		if i == len(columns)-1 {
			fw := max(w, fill)
			col.Width, col.MinWidth, col.MaxWidth, col.Resizable = gui.SomeF(fw), gui.SomeF(fw), gui.SomeF(fw), false
		}
		out[i] = col
	}
	return out
}

// gridCellFormat colors the text fallbacks: the stock bar in the fill
// color (low-stock color under 0.2), the status as the badge text on the
// badge background (whole cell, no pill), the stars in the star color.
func (a *app) gridCellFormat(row datagrid.GridRow, _ int, col datagrid.GridColumnCfg, value string, _ *gui.Window) datagrid.GridCellFormat {
	switch col.ID {
	case "stock":
		q, _ := strconv.Atoi(row.Cells["qty"])
		return datagrid.GridCellFormat{HasTextColor: true, TextColor: hexColor(a.pal.StockFillFor(int32(q)))}
	case "status":
		s, err := core.ParseStatus(value)
		if err != nil {
			return datagrid.GridCellFormat{}
		}
		b := a.pal.Badge(s)
		return datagrid.GridCellFormat{HasBGColor: true, BGColor: hexColor(b.Background), HasTextColor: true, TextColor: hexColor(b.Text)}
	case "rating":
		return datagrid.GridCellFormat{HasTextColor: true, TextColor: hexColor(a.pal.StarFilled)}
	}
	return datagrid.GridCellFormat{}
}
