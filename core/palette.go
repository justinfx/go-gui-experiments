package core

import (
	"image/color"
	"strconv"
	"strings"
)

// BadgeColors are the background and text colors of a pill badge or chip.
type BadgeColors struct {
	Background string
	Text       string
}

// Palette holds the exact colors (hex strings like "#2563EB") for the
// custom-rendered elements: status badges, stock bar, stars, featured chip
// and category tiles.
type Palette struct {
	Active, Backordered, Discontinued BadgeColors
	StockTrack                        string
	StockFill                         string
	StockFillLow                      string // used when StockLow(qty)
	StarFilled                        string
	StarEmpty                         string
	Featured                          BadgeColors
	CategoryTiles                     [8]string // indexed by Category
	TileText                          string    // initial letter on a category tile
}

var lightPalette = Palette{
	Active:        BadgeColors{"#DCFCE7", "#166534"},
	Backordered:   BadgeColors{"#FEF3C7", "#92400E"},
	Discontinued:  BadgeColors{"#E5E7EB", "#374151"},
	StockTrack:    "#E5E7EB",
	StockFill:     "#2563EB",
	StockFillLow:  "#DC2626",
	StarFilled:    "#F59E0B",
	StarEmpty:     "#D1D5DB",
	Featured:      BadgeColors{"#EDE9FE", "#5B21B6"},
	CategoryTiles: [8]string{"#2563EB", "#16A34A", "#EA580C", "#64748B", "#0D9488", "#DC2626", "#CA8A04", "#DB2777"},
	TileText:      "#FFFFFF",
}

var darkPalette = Palette{
	Active:        BadgeColors{"#14532D", "#BBF7D0"},
	Backordered:   BadgeColors{"#78350F", "#FDE68A"},
	Discontinued:  BadgeColors{"#374151", "#E5E7EB"},
	StockTrack:    "#374151",
	StockFill:     "#60A5FA",
	StockFillLow:  "#F87171",
	StarFilled:    "#FBBF24",
	StarEmpty:     "#4B5563",
	Featured:      BadgeColors{"#4C1D95", "#DDD6FE"},
	CategoryTiles: lightPalette.CategoryTiles,
	TileText:      "#FFFFFF",
}

// Colors returns the light or dark palette.
func Colors(dark bool) Palette {
	if dark {
		return darkPalette
	}
	return lightPalette
}

// Badge returns the badge colors for a status.
func (p Palette) Badge(s Status) BadgeColors {
	switch s {
	case Backordered:
		return p.Backordered
	case Discontinued:
		return p.Discontinued
	default:
		return p.Active
	}
}

// StockFillFor returns the stock bar fill color for a quantity.
func (p Palette) StockFillFor(qty int32) string {
	if StockLow(qty) {
		return p.StockFillLow
	}
	return p.StockFill
}

// Tile returns the category tile color.
func (p Palette) Tile(c Category) string {
	if !c.Valid() {
		return p.CategoryTiles[0]
	}
	return p.CategoryTiles[c]
}

// HexColor converts "#RRGGBB" or "#RRGGBBAA" to color.RGBA. Invalid input
// returns opaque black.
func HexColor(hex string) color.RGBA {
	h := strings.TrimPrefix(hex, "#")
	if len(h) != 6 && len(h) != 8 {
		return color.RGBA{A: 255}
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return color.RGBA{A: 255}
	}
	if len(h) == 6 {
		return color.RGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}
	}
	return color.RGBA{R: uint8(v >> 24), G: uint8(v >> 16), B: uint8(v >> 8), A: uint8(v)}
}
