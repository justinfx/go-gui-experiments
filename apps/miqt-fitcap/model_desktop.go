//go:build !js

package main

import (
	"strconv"

	"gguix/core"

	qt "github.com/mappu/miqt/qt6"
)

type columnSpec struct {
	title  string
	width  int
	align  qt.AlignmentFlag
	sort   core.SortField
	custom bool
}

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

var columns = []columnSpec{
	{"ID", 80, qt.AlignRight, core.SortID, false},
	{"SKU", 110, qt.AlignLeft, core.SortSKU, false},
	{"Name", 260, qt.AlignLeft, core.SortName, false},
	{"Category", 110, qt.AlignLeft, core.SortCategory, false},
	{"Price", 100, qt.AlignRight, core.SortPrice, false},
	{"Qty", 80, qt.AlignRight, core.SortQuantity, false},
	{"Stock", 120, qt.AlignLeft, core.SortQuantity, true},
	{"Status", 120, qt.AlignLeft, core.SortStatus, true},
	{"Rating", 110, qt.AlignLeft, core.SortRating, true},
	{"Updated", 150, qt.AlignLeft, core.SortUpdated, false},
	{"Featured", 80, qt.AlignHCenter, core.SortFeatured, false},
}

// columnForSort returns the first column that sorts by f.
func columnForSort(f core.SortField) int {
	for i, c := range columns {
		if c.sort == f {
			return i
		}
	}
	return 0
}

// cellText is the text shown in a plain column. Custom columns return ""
// when loaded (their delegate draws them) and the placeholder otherwise.
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
	case colQty:
		return strconv.Itoa(int(it.Quantity))
	case colUpdated:
		return core.FormatUpdated(it.UpdatedAt)
	case colFeatured:
		if it.Featured {
			return "✓"
		}
	}
	return ""
}

const placeholder = "…"
