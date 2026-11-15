package main

import (
	"fmt"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"

	gx "gguix/core"
)

// The custom rendering of the stock bar, the status badge, the stars, the
// featured chip and the category tile. Each is a View (or Text) built from
// NSGo's own primitives: there is no custom cell mechanism beyond the
// DataTable RenderCell hook, which returns a normal element tree. Colors are
// the section 4.5.1 palette, read inside StyleFuncs from cx.Dark() so a
// theme change restyles the live nodes.

const (
	barWidth  = 80
	barHeight = 8
)

// itemFn returns the item a custom node shows, false for a placeholder.
type itemFn func() (gx.Item, bool)

// colorOf converts a palette hex string to an NSGo color.
func colorOf(hex string) core.Color {
	c := gx.HexColor(hex)
	return core.RGB(uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B))
}

// stockBar is the 80 x 8 track with its fill. With a label it also shows
// the quantity text: to the right in the table, to the left in a card.
func stockBar(cx *core.Scope, it itemFn, labelLeft, labelRight bool, testID string) core.Node {
	qty := func() int32 {
		if i, ok := it(); ok {
			return i.Quantity
		}
		return 0
	}
	fill := core.View{
		TestID: testID + "-fill",
		StyleFunc: func() core.Style {
			pal := gx.Colors(cx.Dark())
			q := qty()
			return core.Style{
				Width:        pt(float32(barWidth * gx.StockFraction(q))),
				Height:       pt(barHeight),
				CornerRadius: barHeight / 2,
				Background:   colorOf(pal.StockFillFor(q)),
			}
		},
	}
	track := core.View{
		TestID: testID + "-track",
		StyleFunc: func() core.Style {
			pal := gx.Colors(cx.Dark())
			return core.Style{
				Width: pt(barWidth), Height: pt(barHeight), CornerRadius: barHeight / 2,
				Background: colorOf(pal.StockTrack), Overflow: core.Hidden, FlexShrink: 0,
			}
		},
		Children: []core.Node{fill},
	}
	children := []core.Node{}
	if labelLeft {
		children = append(children, core.Text{TestID: testID + "-label", ContentFunc: func() string { return gx.StockLabel(qty()) },
			Style: core.Style{FontSize: 12}, StyleFunc: ui.Colored(ui.ColorsOf(cx).Muted)})
	}
	children = append(children, track)
	if labelRight {
		children = append(children, core.Text{TestID: testID + "-qty", ContentFunc: func() string { return fmt.Sprint(qty()) },
			Style: core.Style{FontSize: 13}})
	}
	return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8}, Children: children}
}

// statusBadge is the rounded pill with the status text.
func statusBadge(cx *core.Scope, it itemFn, testID string) core.Node {
	status := func() gx.Status {
		if i, ok := it(); ok {
			return i.Status
		}
		return gx.Active
	}
	return core.View{
		TestID: testID,
		StyleFunc: func() core.Style {
			b := gx.Colors(cx.Dark()).Badge(status())
			return core.Style{
				Padding: core.Symmetric(2, 8), CornerRadius: 999, Background: colorOf(b.Background),
				AlignSelf: core.AlignStart, FlexShrink: 0,
			}
		},
		Children: []core.Node{core.Text{
			TestID:      testID + "-text",
			ContentFunc: func() string { return status().String() },
			StyleFunc: func() core.Style {
				b := gx.Colors(cx.Dark()).Badge(status())
				return core.Style{FontSize: 12, FontWeight: core.FontWeightMedium, Color: colorOf(b.Text)}
			},
			MaxLines: 1,
		}},
	}
}

// featuredChip is the Featured chip of a card.
func featuredChip(cx *core.Scope) core.Node {
	return core.View{
		TestID: "featured-chip",
		StyleFunc: func() core.Style {
			b := gx.Colors(cx.Dark()).Featured
			return core.Style{Padding: core.Symmetric(2, 8), CornerRadius: 999, Background: colorOf(b.Background), FlexShrink: 0}
		},
		Children: []core.Node{core.Text{
			Content: "Featured", MaxLines: 1,
			StyleFunc: func() core.Style {
				b := gx.Colors(cx.Dark()).Featured
				return core.Style{FontSize: 12, FontWeight: core.FontWeightMedium, Color: colorOf(b.Text)}
			},
		}},
	}
}

const starSize = 12

// stars is five 12 px stars (glyphs) followed by the rating text. A half
// star is a filled glyph clipped to its left half over an empty one.
func stars(cx *core.Scope, it itemFn, testID string) core.Node {
	rating := func() float32 {
		if i, ok := it(); ok {
			return i.Rating
		}
		return 0
	}
	star := func(i int) core.Node {
		kind := func() string {
			full, half, _ := gx.Stars(rating())
			switch {
			case i < full:
				return "full"
			case i < full+half:
				return "half"
			}
			return "empty"
		}
		glyph := func(filled bool) core.Style {
			pal := gx.Colors(cx.Dark())
			c := pal.StarEmpty
			if filled {
				c = pal.StarFilled
			}
			return core.Style{FontSize: starSize, Color: colorOf(c), Width: pt(starSize), TextAlign: core.TextAlignCenter}
		}
		return core.View{
			TestID:    fmt.Sprintf("%s-star%d", testID, i),
			LabelFunc: kind,
			Style:     core.Style{Width: pt(starSize), Height: pt(starSize + 2), FlexShrink: 0},
			Children: []core.Node{
				core.Text{Content: "★", StyleFunc: func() core.Style {
					return glyph(kind() == "full")
				}},
				core.View{
					StyleFunc: func() core.Style {
						st := core.Style{Position: core.Absolute, Top: pt(0), Left: pt(0), Height: pt(starSize + 2),
							Width: pt(starSize / 2), Overflow: core.Hidden}
						if kind() != "half" {
							st.Display = core.DisplayNone
						}
						return st
					},
					Children: []core.Node{core.Text{Content: "★", StyleFunc: func() core.Style { return glyph(true) }}},
				},
			},
		}
	}
	row := []core.Node{}
	for i := 0; i < 5; i++ {
		row = append(row, star(i))
	}
	return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 0}, Children: []core.Node{
		core.View{TestID: testID, Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter}, Children: row},
		core.Text{TestID: testID + "-text", ContentFunc: func() string { return " " + gx.FormatRating(rating()) }, Style: core.Style{FontSize: 13}},
	}}
}

// categoryTile is the 48 x 48 rounded tile with the category initial.
func categoryTile(cx *core.Scope, it itemFn) core.Node {
	cat := func() gx.Category {
		if i, ok := it(); ok {
			return i.Category
		}
		return 0
	}
	return core.View{
		TestID: "tile",
		StyleFunc: func() core.Style {
			pal := gx.Colors(cx.Dark())
			return core.Style{Width: pt(48), Height: pt(48), CornerRadius: 10, FlexShrink: 0,
				Background: colorOf(pal.Tile(cat())), AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter}
		},
		Children: []core.Node{core.Text{
			ContentFunc: func() string { return gx.CategoryInitial(cat()) },
			StyleFunc: func() core.Style {
				return core.Style{FontSize: 22, FontWeight: core.FontWeightBold, Color: colorOf(gx.Colors(cx.Dark()).TileText)}
			},
		}},
	}
}
