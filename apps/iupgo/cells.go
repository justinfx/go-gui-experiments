package main

import (
	"image"
	"image/color"
	"math"
)

// Rasterizers for the table's custom cells. IUP's Table can show one image
// per cell (IMAGE_CB) but cannot draw into a cell, so the stock bar, the
// status badge swatch and the stars are small RGBA images made here with
// the standard library and registered with IupImage.

const (
	barW, barH   = 80, 8
	imgH         = 20 // cell image height; the shape is centered in it
	starSize     = 12
	starGap      = 2
	starsW       = 5*starSize + 4*starGap
	badgeW       = 30
	badgeH       = 16
	supersamples = 4
)

func nrgba(c color.RGBA) color.NRGBA { return color.NRGBA{c.R, c.G, c.B, 255} }

// blend composites src with coverage cov over the pixel (non-premultiplied).
func blend(img *image.NRGBA, x, y int, src color.NRGBA, cov float64) {
	if cov <= 0 || x < 0 || y < 0 || x >= img.Rect.Dx() || y >= img.Rect.Dy() {
		return
	}
	i := img.PixOffset(x, y)
	sa := cov * float64(src.A) / 255
	da := float64(img.Pix[i+3]) / 255
	oa := sa + da*(1-sa)
	if oa <= 0 {
		return
	}
	mix := func(s, d uint8) uint8 {
		return uint8(math.Round((float64(s)*sa + float64(d)*da*(1-sa)) / oa))
	}
	img.Pix[i+0] = mix(src.R, img.Pix[i+0])
	img.Pix[i+1] = mix(src.G, img.Pix[i+1])
	img.Pix[i+2] = mix(src.B, img.Pix[i+2])
	img.Pix[i+3] = uint8(math.Round(oa * 255))
}

// fillShape draws every pixel of the box whose supersampled coverage by
// inside is above zero.
func fillShape(img *image.NRGBA, x0, y0, x1, y1 int, c color.NRGBA, inside func(px, py float64) bool) {
	n := supersamples
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			hit := 0
			for sy := 0; sy < n; sy++ {
				for sx := 0; sx < n; sx++ {
					if inside(float64(x)+(float64(sx)+0.5)/float64(n), float64(y)+(float64(sy)+0.5)/float64(n)) {
						hit++
					}
				}
			}
			blend(img, x, y, c, float64(hit)/float64(n*n))
		}
	}
}

// pillInside reports whether (px,py) is inside a fully rounded rectangle.
func pillInside(x0, y0, w, h float64) func(px, py float64) bool {
	r := math.Min(w, h) / 2
	return func(px, py float64) bool {
		if px < x0 || px > x0+w || py < y0 || py > y0+h {
			return false
		}
		cx := math.Max(x0+r, math.Min(px, x0+w-r))
		cy := math.Max(y0+r, math.Min(py, y0+h-r))
		return math.Hypot(px-cx, py-cy) <= r
	}
}

// renderBar draws the stock bar: a track with a fill of the given
// fraction, 80x8 px with rounded ends, centered vertically in an 80x20
// image.
func renderBar(frac float64, track, fill color.RGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, barW, imgH))
	y0 := (imgH - barH) / 2
	fillShape(img, 0, y0, barW, y0+barH, nrgba(track), pillInside(0, float64(y0), barW, barH))
	fw := int(math.Round(frac * barW))
	if fw > 0 {
		fillShape(img, 0, y0, fw, y0+barH, nrgba(fill), pillInside(0, float64(y0), float64(fw), barH))
	}
	return img
}

// renderBadge draws the status swatch: a pill in the badge background with
// a dot in the badge text color.
func renderBadge(bg, fg color.RGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, badgeW, imgH))
	y0 := (imgH - badgeH) / 2
	fillShape(img, 0, y0, badgeW, y0+badgeH, nrgba(bg), pillInside(0, float64(y0), badgeW, badgeH))
	d := 6.0
	cx, cy := float64(badgeW)/2, float64(imgH)/2
	fillShape(img, int(cx-d), int(cy-d), int(cx+d)+1, int(cy+d)+1, nrgba(fg), func(px, py float64) bool {
		return math.Hypot(px-cx, py-cy) <= d/2
	})
	return img
}

// starPoly is a five-point star of the given outer radius.
func starPoly(cx, cy, ro float64) [][2]float64 {
	ri := ro * 0.42
	pts := make([][2]float64, 0, 10)
	for i := 0; i < 10; i++ {
		r := ro
		if i%2 == 1 {
			r = ri
		}
		a := -math.Pi/2 + float64(i)*math.Pi/5
		pts = append(pts, [2]float64{cx + r*math.Cos(a), cy + r*math.Sin(a)})
	}
	return pts
}

func inPoly(pts [][2]float64, px, py float64) bool {
	in := false
	for i, j := 0, len(pts)-1; i < len(pts); j, i = i, i+1 {
		if (pts[i][1] > py) != (pts[j][1] > py) &&
			px < (pts[j][0]-pts[i][0])*(py-pts[i][1])/(pts[j][1]-pts[i][1])+pts[i][0] {
			in = !in
		}
	}
	return in
}

// renderStars draws five 12 px stars: full, half (left half filled), empty.
func renderStars(full, half, empty int, filled, emptyC color.RGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, starsW, imgH))
	for i := 0; i < 5; i++ {
		x0 := i * (starSize + starGap)
		cx, cy := float64(x0)+starSize/2.0, float64(imgH)/2
		poly := starPoly(cx, cy, starSize/2.0)
		y0 := (imgH-starSize)/2 - 1
		box := func(c color.NRGBA, inside func(px, py float64) bool) {
			fillShape(img, x0, y0, x0+starSize, y0+starSize+2, c, inside)
		}
		switch {
		case i < full:
			box(nrgba(filled), func(px, py float64) bool { return inPoly(poly, px, py) })
		case i == full && half > 0:
			box(nrgba(emptyC), func(px, py float64) bool { return inPoly(poly, px, py) })
			box(nrgba(filled), func(px, py float64) bool { return px < cx && inPoly(poly, px, py) })
		default:
			box(nrgba(emptyC), func(px, py float64) bool { return inPoly(poly, px, py) })
		}
	}
	_ = empty
	return img
}
