package main

import (
	"os"
	"testing"

	"fyne.io/fyne/v2/test"

	"gguix/core/memory"
)

// TestScreenshots writes PNGs of the main states when GGUIX_SHOT_DIR is set.
func TestScreenshots(t *testing.T) {
	if os.Getenv("GGUIX_SHOT_DIR") == "" {
		t.Skip("set GGUIX_SHOT_DIR to write screenshots")
	}
	h := newHarness(t, memory.Options{Rows: 1000})
	h.loaded()
	h.shot("table-light")
	cells := h.tableCells()
	h.tapAt(cells[1][colName], 5, 5)
	h.pump()
	h.shot("table-selected")
	test.Tap(h.u.cardsBtn)
	h.pump()
	h.loaded()
	h.shot("cards-light")
	h.u.setThemeMode("dark")
	h.pump()
	h.shot("cards-dark")
	test.Tap(h.u.tableBtn)
	h.pump()
	h.loaded()
	h.shot("table-dark")
}
