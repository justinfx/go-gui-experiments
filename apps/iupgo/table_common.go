package main

import (
	"strconv"

	"gguix/core"
)

// colDef is one table column.
type colDef struct {
	title string
	width int
	align string
	sort  core.SortField
}

var tableCols = []colDef{
	{"ID", 80, "ARIGHT", core.SortID},
	{"SKU", 110, "ALEFT", core.SortSKU},
	{"Name", 260, "ALEFT", core.SortName},
	{"Category", 110, "ALEFT", core.SortCategory},
	{"Price", 100, "ARIGHT", core.SortPrice},
	{"Qty", 80, "ARIGHT", core.SortQuantity},
	{"Stock", 120, "ALEFT", core.SortQuantity},
	{"Status", 120, "ALEFT", core.SortStatus},
	{"Rating", 110, "ALEFT", core.SortRating},
	{"Updated", 150, "ALEFT", core.SortUpdated},
	{"Featured", 80, "ACENTER", core.SortFeatured},
}

// Column indexes (1-based, as IUP counts them).
const (
	colID = iota + 1
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

const placeholder = "…"

// cellSpec is what a custom cell shows, independent of the toolkit.
type cellSpec struct {
	Kind string // "stock", "status", "rating" or ""
	// Stock bar.
	Fraction float64
	Track    string
	Fill     string
	Label    string
	// Status badge.
	Text    string
	BadgeBG string
	BadgeFG string
	// Stars.
	Full, Half, Empty int
	StarFilled        string
	StarEmpty         string
}

func cellText(it core.Item, col int) string {
	switch col {
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
	case colQty, colStock:
		return strconv.Itoa(int(it.Quantity))
	case colStatus:
		return it.Status.String()
	case colRating:
		return core.FormatRating(it.Rating)
	case colUpdated:
		return core.FormatUpdated(it.UpdatedAt)
	case colFeatured:
		if it.Featured {
			return "✓"
		}
	}
	return ""
}

// cellSpecFor describes the custom cell content of a loaded row.
func (a *app) cellSpecFor(it core.Item, col int) cellSpec {
	p := a.pal
	switch col {
	case colStock:
		return cellSpec{Kind: "stock", Fraction: core.StockFraction(it.Quantity), Track: p.StockTrack,
			Fill: p.StockFillFor(it.Quantity), Label: strconv.Itoa(int(it.Quantity))}
	case colStatus:
		b := p.Badge(it.Status)
		return cellSpec{Kind: "status", Text: it.Status.String(), BadgeBG: b.Background, BadgeFG: b.Text}
	case colRating:
		f, h, e := core.Stars(it.Rating)
		return cellSpec{Kind: "rating", Full: f, Half: h, Empty: e, StarFilled: p.StarFilled,
			StarEmpty: p.StarEmpty, Text: core.FormatRating(it.Rating)}
	}
	return cellSpec{}
}
