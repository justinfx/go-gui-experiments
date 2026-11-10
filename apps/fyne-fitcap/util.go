package main

import (
	"image/color"
	"math"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// themeColor returns a color of the theme that is currently in effect.
func themeColor(name fyne.ThemeColorName) color.Color {
	s := fyne.CurrentApp().Settings()
	return s.Theme().Color(name, s.ThemeVariant())
}

// themeSize returns a size of the theme that is currently in effect.
func themeSize(name fyne.ThemeSizeName) float32 {
	return fyne.CurrentApp().Settings().Theme().Size(name)
}

// isDark reports whether the theme in effect (including a forced variant)
// is a dark one, judged by the luminance of its background color.
func isDark() bool {
	r, g, b, _ := themeColor(theme.ColorNameBackground).RGBA()
	lum := (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 65535
	return lum < 0.5
}

func textWidth(s string, size float32, style fyne.TextStyle) float32 {
	return fyne.MeasureText(s, size, style).Width
}

// fitText shortens s with an ellipsis so that it is at most maxW wide.
func fitText(s string, size float32, style fyne.TextStyle, maxW float32) string {
	if maxW <= 0 {
		return ""
	}
	if textWidth(s, size, style) <= maxW {
		return s
	}
	r := []rune(s)
	lo, hi := 0, len(r)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if textWidth(string(r[:mid])+"…", size, style) <= maxW {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return strings.TrimRight(string(r[:lo]), " ") + "…"
}

func maxf(a, b float32) float32 {
	return float32(math.Max(float64(a), float64(b)))
}

func minf(a, b float32) float32 {
	return float32(math.Min(float64(a), float64(b)))
}
