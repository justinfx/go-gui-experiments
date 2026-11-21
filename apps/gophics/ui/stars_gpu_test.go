//go:build gophics_gpu

package ui

import (
	"image"
	"testing"

	"gguix/core"
)

// TestStarsClippedGPU guards fixpass issue 1: with the Rating column cut by
// the split, no star pixel may land in the form area on the GPU backend.
// Run with: go test -tags gophics_gpu -run StarsClipped ./ui
func TestStarsClippedGPU(t *testing.T) {
	hs := newHarness(t, 0)
	hs.ui(func() { hs.a.split = 1000.0 / 1280 })
	img := hs.h.RenderGPU()
	if img == nil {
		t.Skip("no GPU adapter")
	}
	pal := core.Colors(hs.a.IsDark())
	sc := img.Bounds().Dx() / 1280
	form := img.(interface {
		SubImage(image.Rectangle) image.Image
	}).SubImage(image.Rect(1006*sc, 100*sc, 1100*sc, 700*sc))
	if hasPixel(form, colorOf(pal.StarFilled)) || hasPixel(form, colorOf(pal.StarEmpty)) {
		t.Fatal("star pixels drawn over the form area")
	}
}
