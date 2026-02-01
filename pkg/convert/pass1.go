// Package convert provides Pass1 conversion functionality for go-cpc-image.
// This file contains color reduction, HSL adjustment, contrast, and bit-depth reduction.
package convert

import (
	"math"

	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
	"github.com/ikari-pl/go-cpc-image/pkg/cpc"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SetLumiSat modifies luminosity and saturation of RGB values.
// This replicates the C# SetLumiSat function exactly.
func SetLumiSat(lumi, satur float32, r, v, b *float32) {
	min := float32(math.Min(float64(*r), math.Min(float64(*v), float64(*b))))
	max := float32(math.Max(float64(*r), math.Max(float64(*v), float64(*b))))
	dif := max - min
	var hue float32 = 0

	if max > min {
		if *v == max {
			hue = (*b-*r)/dif*60.0 + 120.0
		} else if *b == max {
			hue = (*r-*v)/dif*60.0 + 240.0
		} else {
			hue = (*v-*b)/dif*60.0
			if *b > *v {
				hue += 360.0
			}
		}
		if hue < 0 {
			hue += 360.0
		}
	}

	hue *= 255.0 / 360.0
	sat := satur * (dif / max) * 255.0
	bri := lumi * max

	*r = bri
	*v = bri
	*b = bri

	if sat != 0 {
		max = bri
		dif = bri * sat / 255.0
		min = bri - dif
		h := hue * 360.0 / 255.0

		if h < 60.0 {
			*r = max
			*v = h*dif/60.0 + min
			*b = min
		} else if h < 120.0 {
			*r = -(h-120.0)*dif/60.0 + min
			*v = max
			*b = min
		} else if h < 180.0 {
			*r = min
			*v = max
			*b = (h-120.0)*dif/60.0 + min
		} else if h < 240.0 {
			*r = min
			*v = -(h-240.0)*dif/60.0 + min
			*b = max
		} else if h < 300.0 {
			*r = (h-240.0)*dif/60.0 + min
			*v = min
			*b = max
		} else if h <= 360.0 {
			*r = max
			*v = min
			*b = -(h-360.0)*dif/60.0 + min
		} else {
			*r = 0
			*v = 0
			*b = 0
		}
	}
}

// GetNumColorPixelCpc returns the CPC color index (0-26) closest to the given pixel.
// This implements both old method (thresholds) and new method (distance calculation).
func GetNumColorPixelCpc(prm *Param, p bitmap.RgbColor) int {
	colorIndex := 0

	if !prm.NewMethod {
		// Old method: use RGB thresholds for 3x3x3 = 27 color cube
		rIndex := 0
		if p.R > byte(prm.CstR2) {
			rIndex = 6
		} else if p.R > byte(prm.CstR1) {
			rIndex = 3
		}

		vIndex := 0
		if p.V > byte(prm.CstV2) {
			vIndex = 18
		} else if p.V > byte(prm.CstV1) {
			vIndex = 9
		}

		bIndex := 0
		if p.B > byte(prm.CstB2) {
			bIndex = 2
		} else if p.B > byte(prm.CstB1) {
			bIndex = 1
		}

		colorIndex = rIndex + vIndex + bIndex
	} else {
		// New method: find closest color by weighted Euclidean distance
		oldDist := 0x7FFFFFFF
		for i := 0; i < 27; i++ {
			s := cpc.CpcRgbPalette[i]
			dr := int(s.R) - int(p.R)
			dv := int(s.V) - int(p.V)
			db := int(s.B) - int(p.B)
			dist := dr*dr*prm.CoefR + dv*dv*prm.CoefV + db*db*prm.CoefB

			if dist < oldDist {
				oldDist = dist
				colorIndex = i
				if dist == 0 {
					break
				}
			}
		}
	}

	return colorIndex
}

// GetPixel retrieves and processes a pixel with all transformations applied.
// This includes averaging, bit-depth reduction, color channel adjustments,
// contrast, luminosity/saturation, and optional dithering.
func GetPixel(source *bitmap.DirectBitmap, xPix, yPix, Tx int, prm *Param, pct int) bitmap.RgbColor {
	var p bitmap.RgbColor

	// Smoothing (averaging) if enabled
	if prm.Smoothing {
		var r, v, b float32 = 0, 0, 0
		for i := 0; i < Tx; i++ {
			pixel := source.GetPixelColor(xPix+i, yPix)
			r += float32(pixel.R)
			v += float32(pixel.V)
			b += float32(pixel.B)
		}
		p.R = byte(r / float32(Tx))
		p.V = byte(v / float32(Tx))
		p.B = byte(b / float32(Tx))
	} else {
		p = source.GetPixelColor(xPix, yPix)
	}

	// Bit depth reduction
	switch prm.BitsRGB {
	case 12: // 12 bits (4 bits per component)
		p.R = (p.R >> 4) * 17
		p.V = (p.V >> 4) * 17
		p.B = (p.B >> 4) * 17
	case 9: // 9 bits (3 bits per component)
		p.R = (p.R >> 5) * 36
		p.V = (p.V >> 5) * 36
		p.B = (p.B >> 5) * 36
	case 6: // 6 bits (2 bits per component)
		p.R = (p.R >> 6) * 85
		p.V = (p.V >> 6) * 85
		p.B = (p.B >> 6) * 85
	}

	// Palette reduction masks
	m1 := byte(0)
	if prm.ReductPal1 {
		m1 |= 0x11
	}
	if prm.ReductPal3 {
		m1 |= 0x22
	}

	m2 := byte(0xFF)
	if prm.ReductPal2 {
		m2 &= 0xEE
	}
	if prm.ReductPal4 {
		m2 &= 0xDD
	}

	// Apply masks and color channel percentages
	p.R = MinMaxByte(float64((p.R|m1)&m2) * float64(prm.PctRed) / 100.0)
	p.V = MinMaxByte(float64((p.V|m1)&m2) * float64(prm.PctGreen) / 100.0)
	p.B = MinMaxByte(float64((p.B|m1)&m2) * float64(prm.PctBlue) / 100.0)

	// Apply contrast and luminosity/saturation if pixel is not black
	if p.R != 0 || p.V != 0 || p.B != 0 {
		r := float32(contrastTable[p.R])
		v := float32(contrastTable[p.V])
		b := float32(contrastTable[p.B])

		// Apply luminosity and saturation adjustments
		if prm.PctLumi != 100 || prm.PctSat != 100 {
			var lumiAdj float32
			if prm.PctLumi > 100 {
				lumiAdj = float32(100+(prm.PctLumi-100)*2) / 100.0
			} else {
				lumiAdj = float32(prm.PctLumi) / 100.0
			}
			satAdj := float32(prm.PctSat) / 100.0
			SetLumiSat(lumiAdj, satAdj, &r, &v, &b)
		}

		p.R = MinMaxByte(float64(r))
		p.V = MinMaxByte(float64(v))
		p.B = MinMaxByte(float64(b))
	}

	// Apply dithering matrix if enabled
	if pct > 0 {
		var chosen bitmap.RgbColor
		if prm.CpcPlus {
			// CPC+ mode: 4096 colors (4 bits per RGB component)
			chosen = bitmap.NewRgbColor((p.R>>4)*17, (p.V>>4)*17, (p.B>>4)*17)
		} else {
			// Standard CPC: 27 colors
			chosen = cpc.CpcRgbPalette[GetNumColorPixelCpc(prm, p)]
		}

		// Apply dithering using existing dither system
		// Create a wrapper to match the dither.Bitmap interface
		wrapper := &bitmapWrapper{source}
		p = DoDitherFull(wrapper, xPix, yPix, Tx, p, chosen, prm.DiffErr)
	}

	return p
}

// bitmapWrapper adapts bitmap.DirectBitmap to the dither.Bitmap interface
type bitmapWrapper struct {
	*bitmap.DirectBitmap
}

func (w *bitmapWrapper) SetPixel(x, y int, color bitmap.RgbColor) {
	w.DirectBitmap.SetPixelColor(x, y, color)
}

// ConvertPass1 performs the first conversion pass:
// - Reduces palette to CPC colors
// - Fills frequency table (colorFrequency)
// - Applies dithering if requested
func ConvertPass1(source *bitmap.DirectBitmap, prm *Param) {
	// Clear frequency table
	for i := range colorFrequency {
		for j := range colorFrequency[i] {
			colorFrequency[i][j] = 0
		}
	}

	// Set up dithering
	ditherParam := DitherParam{
		CpcPlus: prm.CpcPlus,
		Pct:     prm.Pct,
		Method: prm.Method,
		DiffErr: prm.DiffErr,
	}
	pct := SetMatDither(ditherParam)

	var p, chosen, n1, n2 bitmap.RgbColor
	var colorIndex int

	// Calculate screen dimensions
	width := prm.GetScreenWidth()
	height := prm.GetScreenHeight()

	// Increment step: 4 for true color dithering, 2 otherwise
	stepY := 2
	if prm.TrueColorDither {
		stepY = 4
	}

	// Process pixels
	for yPix := 0; yPix < height; yPix += stepY {
		Tx := cpc.PixelWidth(prm.VirtualMode, yPix, prm.YEgx)

		for xPix := 0; xPix < width; xPix += Tx {
			for yy := 0; yy < stepY; yy += 2 {
				p = GetPixel(source, xPix, yy+yPix, Tx, prm, pct)

				// Find closest CPC color
				if prm.CpcPlus {
					// CPC+ mode: 4096 colors
					chosen = bitmap.NewRgbColor((p.R>>4)*17, (p.V>>4)*17, (p.B>>4)*17)
					colorIndex = (int(chosen.V>>4) << 8) | (int(chosen.B>>4) << 4) | int(chosen.R>>4)
				} else {
					// Standard CPC: 27 colors
					colorIndex = GetNumColorPixelCpc(prm, p)
					chosen = cpc.CpcRgbPalette[colorIndex]
				}

				// Increment frequency counter
				colorFrequency[colorIndex][(yPix+yy)>>1]++

				// Set pixel if not using true color dithering
				if !prm.TrueColorDither {
					if prm.SetPalCpc {
						source.SetPixelColor(xPix, yy+yPix, chosen)
					} else {
						source.SetPixelColor(xPix, yy+yPix, p)
					}
				}
			}

			// True color dithering processing
			if prm.TrueColorDither {
				// Average of 2 pixels
				p0 := GetPixel(source, xPix, yPix, Tx, prm, pct)
				p1 := GetPixel(source, xPix, yPix+2, Tx, prm, pct)

				r := int(p0.R) + int(p1.R)
				v := int(p0.V) + int(p1.V)
				b := int(p0.B) + int(p1.B)

				if prm.CpcPlus {
					// CPC+ true color processing
					n1 = bitmap.NewRgbColor(byte(r>>1), byte(v>>1), byte(b>>1))
					n2 = bitmap.NewRgbColor(
						byte(min(255, (r>>3)*8)),
						byte(min(255, (v>>3)*8)),
						byte(min(255, (b>>3)*8)),
					)
				} else {
					// Standard CPC true color processing using thresholds
					n1Index := 0
					if r > prm.CstR3 {
						n1Index += 6
					} else if r > prm.CstR1 {
						n1Index += 3
					}
					if v > prm.CstV3 {
						n1Index += 18
					} else if v > prm.CstV1 {
						n1Index += 9
					}
					if b > prm.CstB3 {
						n1Index += 2
					} else if b > prm.CstB1 {
						n1Index += 1
					}

					n2Index := 0
					if r > prm.CstR4 {
						n2Index += 6
					} else if r > prm.CstR2 {
						n2Index += 3
					}
					if v > prm.CstV4 {
						n2Index += 18
					} else if v > prm.CstV2 {
						n2Index += 9
					}
					if b > prm.CstB4 {
						n2Index += 2
					} else if b > prm.CstB2 {
						n2Index += 1
					}

					n1 = cpc.CpcRgbPalette[n1Index]
					n2 = cpc.CpcRgbPalette[n2Index]
				}

				// Set alternating pattern
				if (xPix & Tx) == 0 {
					source.SetPixelColor(xPix, yPix, n1)
					source.SetPixelColor(xPix, yPix+2, n2)
				} else {
					source.SetPixelColor(xPix, yPix, n2)
					source.SetPixelColor(xPix, yPix+2, n1)
				}
			}
		}
	}
}