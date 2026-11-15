//go:build !matrix && !js

package main

import (
	"testing"

	"gguix/core"
)

// nativeCellChecks looks at the images the Table variant shows in the custom cells.
func nativeCellChecks(t *testing.T, it core.Item) {
	p := core.Colors(ta.dark)
	bar := ta.cellImageFor(it, colStock)
	fill := core.HexColor(p.StockFillFor(it.Quantity))
	n := 0
	for x := 0; x < barW; x++ {
		c := bar.img.NRGBAAt(x, imgH/2)
		if c.A == 255 && c.R == fill.R && c.G == fill.G && c.B == fill.B {
			n++
		}
	}
	if want := int(core.StockFraction(it.Quantity)*barW + 0.5); n < want-4 || n > want {
		t.Errorf("bar fill pixels %d want about %d", n, want)
	}
	st := ta.cellImageFor(it, colStatus)
	bg := core.HexColor(p.Badge(it.Status).Background)
	if c := st.img.NRGBAAt(3, imgH/2); c.R != bg.R || c.G != bg.G || c.B != bg.B {
		t.Errorf("badge pixel %v want %v", c, bg)
	}
	if name := ta.tableImage(0, colStock); name != bar.name {
		t.Errorf("IMAGE_CB name %q", name)
	}
}
