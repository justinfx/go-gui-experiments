//go:build !js

package main

import qt "github.com/mappu/miqt/qt6"

// Data layer for MIQT, part 2: one item model over the rowStore. The Table
// and the Cards view are both views on this model (the card list shows column
// 0 and paints the whole row with its delegate) and they share one selection
// model, so a row is counted, reset, selected and refreshed exactly once.
// data() only reads the store (no locks, never blocks); rows without data
// answer with the placeholder. Models emit dataChanged for the rows on screen.

// tableModel is a QAbstractTableModel over the rowStore.
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
			it, ok := a.rows.Row(idx.Row())
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
