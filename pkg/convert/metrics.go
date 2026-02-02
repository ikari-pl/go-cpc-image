// Package convert provides image comparison metrics for auto-optimization.
package convert

import (
	"math"

	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
)

// bicubicWeight returns the Mitchell-Netravali bicubic weight for distance t.
func bicubicWeight(t float64) float64 {
	t = math.Abs(t)
	if t >= 2 {
		return 0
	}
	// Mitchell-Netravali with B=1/3, C=1/3
	const B, C = 1.0 / 3.0, 1.0 / 3.0
	t2 := t * t
	t3 := t2 * t
	if t < 1 {
		return ((12 - 9*B - 6*C) * t3 +
			(-18 + 12*B + 6*C) * t2 +
			(6 - 2*B)) / 6
	}
	return ((-B - 6*C) * t3 +
		(6*B + 30*C) * t2 +
		(-12*B - 48*C) * t +
		(8*B + 24*C)) / 6
}

// downsample4x returns a new bitmap that is 1/4 the size using bicubic resampling.
func downsampleNx(src *bitmap.DirectBitmap, factor int) *bitmap.DirectBitmap {
	sw := src.Width()
	sh := src.Height()
	dw := sw / factor
	dh := sh / factor
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}

	dst := bitmap.NewDirectBitmap(dw, dh)

	for dy := 0; dy < dh; dy++ {
		srcY := (float64(dy) + 0.5) * float64(sh) / float64(dh)
		for dx := 0; dx < dw; dx++ {
			srcX := (float64(dx) + 0.5) * float64(sw) / float64(dw)

			var r, g, b, wSum float64
			iy0 := int(math.Floor(srcY)) - 1
			ix0 := int(math.Floor(srcX)) - 1

			for ky := 0; ky < 4; ky++ {
				sy := iy0 + ky
				if sy < 0 {
					sy = 0
				} else if sy >= sh {
					sy = sh - 1
				}
				wy := bicubicWeight(srcY - float64(sy) - 0.5)

				for kx := 0; kx < 4; kx++ {
					sx := ix0 + kx
					if sx < 0 {
						sx = 0
					} else if sx >= sw {
						sx = sw - 1
					}
					wx := bicubicWeight(srcX - float64(sx) - 0.5)
					w := wx * wy
					if w == 0 {
						continue
					}

					c := src.GetPixelColor(sx, sy)
					r += float64(c.R) * w
					g += float64(c.V) * w
					b += float64(c.B) * w
					wSum += w
				}
			}

			if wSum > 0 {
				r /= wSum
				g /= wSum
				b /= wSum
			}
			clamp := func(v float64) byte {
				if v < 0 {
					return 0
				}
				if v > 255 {
					return 255
				}
				return byte(v + 0.5)
			}
			dst.SetPixelColor(dx, dy, bitmap.NewRgbColor(clamp(r), clamp(g), clamp(b)))
		}
	}
	return dst
}

// mseAt computes mean squared error between two same-sized bitmaps.
func mseAt(a, b *bitmap.DirectBitmap) float64 {
	w := a.Width()
	h := a.Height()
	n := w * h
	if n == 0 {
		return 0
	}
	var sum float64
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c1 := a.GetPixelColor(x, y)
			c2 := b.GetPixelColor(x, y)
			dr := float64(c1.R) - float64(c2.R)
			dg := float64(c1.V) - float64(c2.V)
			dbl := float64(c1.B) - float64(c2.B)
			sum += dr*dr + dg*dg + dbl*dbl
		}
	}
	return sum / float64(n*3)
}

// ComputeMSE blends colour accuracy (8x downscale) with spatial detail (2x downscale).
// The 8x pass is dither-tolerant and measures overall colour fidelity.
// The 2x pass rewards modes with finer pixel resolution (mode 1, mode 2)
// by preserving enough spatial detail to penalise blocky wide pixels.
// Blend: 50% colour + 50% detail.
func ComputeMSE(a, b *bitmap.DirectBitmap) float64 {
	if a.Width() != b.Width() || a.Height() != b.Height() {
		return 1e18
	}

	// Colour accuracy at 8x (dither-tolerant)
	colourMSE := mseAt(downsampleNx(a, 8), downsampleNx(b, 8))

	// Spatial detail at 4x (rewards sharp pixels, still averages most dithering)
	detailMSE := mseAt(downsampleNx(a, 4), downsampleNx(b, 4))

	return 0.5*colourMSE + 0.5*detailMSE
}
