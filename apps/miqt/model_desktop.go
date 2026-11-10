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

// tableModel is a QAbstractTableModel over the pager.
type tableModel struct {
	*qt.QAbstractTableModel
	a     *App
	shown int
}

func newTableModel(a *App) *tableModel {
	m := &tableModel{QAbstractTableModel: qt.NewQAbstractTableModel(), a: a}
	m.OnRowCount(func(parent *qt.QModelIndex) int {
		if parent.IsValid() {
			return 0
		}
		return m.shown
	})
	m.OnColumnCount(func(parent *qt.QModelIndex) int {
		if parent.IsValid() {
			return 0
		}
		return len(columns)
	})
	m.OnData(func(idx *qt.QModelIndex, role int) *qt.QVariant {
		if !idx.IsValid() {
			return qt.NewQVariant()
		}
		col := idx.Column()
		switch qt.ItemDataRole(role) {
		case qt.DisplayRole:
			it, ok := a.pager.Row(idx.Row())
			if !ok {
				return qt.NewQVariant14(placeholder)
			}
			return qt.NewQVariant14(cellText(it, col))
		case qt.TextAlignmentRole:
			return qt.NewQVariant4(int(columns[col].align | qt.AlignVCenter))
		}
		return qt.NewQVariant()
	})
	m.OnHeaderData(func(super func(int, qt.Orientation, int) *qt.QVariant, section int, o qt.Orientation, role int) *qt.QVariant {
		if o != qt.Horizontal || section < 0 || section >= len(columns) {
			return super(section, o, role)
		}
		switch qt.ItemDataRole(role) {
		case qt.DisplayRole:
			return qt.NewQVariant14(m.headerText(section))
		case qt.TextAlignmentRole:
			return qt.NewQVariant4(int(columns[section].align | qt.AlignVCenter))
		}
		return super(section, o, role)
	})
	return m
}

// headerText is the column title plus the sort indicator of the active column.
func (m *tableModel) headerText(col int) string {
	t := columns[col].title
	if col == m.a.sortCol && m.a.query.SortBy == columns[col].sort {
		if m.a.query.Desc {
			return t + " ▼"
		}
		return t + " ▲"
	}
	return t
}

func (m *tableModel) headersChanged() {
	m.HeaderDataChanged(qt.Horizontal, 0, len(columns)-1)
}

// reset drops all rows (new query) and sets the row count.
func (m *tableModel) reset(total int) {
	m.BeginResetModel()
	m.shown = total
	m.EndResetModel()
}

// sync grows or shrinks the row count without disturbing the scroll position.
func (m *tableModel) sync(total int) {
	switch {
	case total > m.shown:
		m.BeginInsertRows(qt.NewQModelIndex(), m.shown, total-1)
		m.shown = total
		m.EndInsertRows()
	case total < m.shown:
		m.BeginRemoveRows(qt.NewQModelIndex(), total, m.shown-1)
		m.shown = total
		m.EndRemoveRows()
	}
}

// changed tells views that rows lo..hi may have new data.
func (m *tableModel) changed(lo, hi int) {
	if m.shown == 0 {
		return
	}
	hi = min(hi, m.shown-1)
	if lo > hi {
		return
	}
	m.DataChanged(m.Index(lo, 0, qt.NewQModelIndex()), m.Index(hi, len(columns)-1, qt.NewQModelIndex()))
}

// cardModel is a QAbstractListModel over the same pager.
type cardModel struct {
	*qt.QAbstractListModel
	a     *App
	shown int
}

func newCardModel(a *App) *cardModel {
	m := &cardModel{QAbstractListModel: qt.NewQAbstractListModel(), a: a}
	m.OnRowCount(func(parent *qt.QModelIndex) int {
		if parent.IsValid() {
			return 0
		}
		return m.shown
	})
	m.OnData(func(idx *qt.QModelIndex, role int) *qt.QVariant {
		if !idx.IsValid() {
			return qt.NewQVariant()
		}
		if qt.ItemDataRole(role) == qt.DisplayRole {
			if it, ok := a.pager.Row(idx.Row()); ok {
				return qt.NewQVariant14(it.Name)
			}
			return qt.NewQVariant14("Loading…")
		}
		return qt.NewQVariant()
	})
	return m
}

func (m *cardModel) reset(total int) {
	m.BeginResetModel()
	m.shown = total
	m.EndResetModel()
}

func (m *cardModel) sync(total int) {
	switch {
	case total > m.shown:
		m.BeginInsertRows(qt.NewQModelIndex(), m.shown, total-1)
		m.shown = total
		m.EndInsertRows()
	case total < m.shown:
		m.BeginRemoveRows(qt.NewQModelIndex(), total, m.shown-1)
		m.shown = total
		m.EndRemoveRows()
	}
}

func (m *cardModel) changed(lo, hi int) {
	if m.shown == 0 {
		return
	}
	hi = min(hi, m.shown-1)
	if lo > hi {
		return
	}
	m.DataChanged(m.Index(lo, 0, qt.NewQModelIndex()), m.Index(hi, 0, qt.NewQModelIndex()))
}
