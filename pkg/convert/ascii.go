// Package convert provides ASCII art conversion functionality for CPC modes.
// ASCII modes use 4x4 trame patterns and different resolutions (ASC0/ASC1/ASC2).
package convert

import (
	"math"

	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
	"github.com/ikari-pl/go-cpc-image/pkg/cpc"
)

// PatternM1 represents a 4x4 trame pattern for Mode 1 ASCII art.
// This is equivalent to the C# PatternM1 class.
type PatternM1 struct {
	Pattern [4][4]byte // 4x4 pattern of pen numbers
}

// NewPatternM1 creates a new empty 4x4 trame pattern.
func NewPatternM1() *PatternM1 {
	return &PatternM1{}
}

// SetPix sets a pixel in the trame pattern.
func (t *PatternM1) SetPix(x, y int, pen byte) {
	if x >= 0 && x < 4 && y >= 0 && y < 4 {
		t.Pattern[y][x] = pen
	}
}

// GetPix gets a pixel from the trame pattern.
func (t *PatternM1) GetPix(x, y int) byte {
	if x >= 0 && x < 4 && y >= 0 && y < 4 {
		return t.Pattern[y][x]
	}
	return 0
}

// IsSame checks if this trame is identical to another trame.
func (t *PatternM1) IsSame(other *PatternM1, maxPens int) bool {
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if t.Pattern[y][x] != other.Pattern[y][x] {
				return false
			}
		}
	}
	return true
}

// AsciiConverter handles ASCII art conversion modes (ASC0, ASC1, ASC2).
type AsciiConverter struct {
	params  *Settings
	cpcPlus bool
}

// NewAsciiConverter creates a new ASCII art converter.
func NewAsciiConverter(params *Settings, cpcPlus bool) *AsciiConverter {
	return &AsciiConverter{
		params:  params,
		cpcPlus: cpcPlus,
	}
}

// ConvertAscii performs ASCII art conversion (ASC0/ASC1/ASC2 modes).
// These modes use different pixel groupings and averaging.
func (conv *AsciiConverter) ConvertAscii(source *bitmap.DirectBitmap, dest *ImageCpc, maxPen int, colorTable *[16][272]bitmap.RgbColor) {
	stepY := 8 // Process 8 pixels vertically at a time
	width := dest.Width
	height := dest.Height

	for y := 0; y < height; y += stepY {
		// Calculate horizontal step based on current line and mode
		tx := cpc.PixelWidth(dest.Mode, y, 0) // yEgx is 0 for ASCII modes

		for x := 0; x < width; x += tx {
			oldDist := 0x7FFFFFFF
			var r, v, b int

			// Average color over the 8-pixel vertical block (every 2 pixels)
			for j := 0; j < stepY; j += 2 {
				pix := source.GetPixelColor(x, y+j)
				r += int(pix.R)
				v += int(pix.V)
				b += int(pix.B)
			}

			// Average the accumulated values (divide by 4 since we took 4 samples)
			avgColor := bitmap.NewRgbColor(
				byte(r>>2), // r / 4
				byte(v>>2), // v / 4
				byte(b>>2), // b / 4
			)

			// Find the best matching pen
			chosen := 0
			for i := 0; i < maxPen; i++ {
				c := colorTable[i][y>>1]
				if c.R == 0 && c.V == 0 && c.B == 0 && i > 0 {
					continue // Skip invalid colors
				}

				// Calculate squared Euclidean distance with coefficients
				rDiff := int(c.R) - int(avgColor.R)
				vDiff := int(c.V) - int(avgColor.V)
				bDiff := int(c.B) - int(avgColor.B)
				dist := rDiff*rDiff*conv.params.CoefR +
				       vDiff*vDiff*conv.params.CoefV +
				       bDiff*bDiff*conv.params.CoefB

				if dist < oldDist {
					chosen = i
					oldDist = dist
					if dist == 0 {
						break // Exact match found
					}
				}
			}

			// Apply Y offset for import draw mode if needed
			offsetY := 0
			if conv.params.ModeImpDraw && height == 544 {
				offsetY = 2
			}

			// Set all pixels in the block to the chosen pen
			for j := 0; j < stepY; j += 2 {
				dest.SetPixelCpc(x, y+offsetY+j, chosen, tx)
			}
		}
	}
}

// ConvertAscUt performs ASCII conversion with precalculated trame patterns in Mode 1.
// This uses 4x4 trame patterns stored in trameM1 for dithering.
func (conv *AsciiConverter) ConvertAscUt(source *bitmap.DirectBitmap, dest *ImageCpc, colorTable *[16][272]bitmap.RgbColor, trameM1 *[16][4][4]byte) {
	width := dest.Width
	height := dest.Height

	for y := 0; y <= height-8; y += 8 {
		for x := 0; x < width; x += 8 {
			chosen := 0
			oldDist := 0x7FFFFFFF

			// Test all 16 precalculated trame patterns
			for i := 0; i < 16; i++ {
				var r1, v1, b1, r2, v2, b2 int

				// Calculate average colors for source and trame pattern
				for ym := 0; ym < 4; ym++ {
					for xm := 0; xm < 4; xm++ {
						// Get source pixel color
						pix := source.GetPixelColor(x+(xm<<1), y+(ym<<1))
						r1 += int(pix.R)
						v1 += int(pix.V)
						b1 += int(pix.B)

						// Get trame pattern color
						pen := trameM1[i][xm][ym]
						c := colorTable[pen][ym+(y>>1)]
						r2 += int(c.R)
						v2 += int(c.V)
						b2 += int(c.B)
					}
				}

				// Calculate Manhattan distance between averaged colors
				dist := int(math.Abs(float64(r1-r2)))*conv.params.CoefR +
				       int(math.Abs(float64(v1-v2)))*conv.params.CoefV +
				       int(math.Abs(float64(b1-b2)))*conv.params.CoefB

				if dist < oldDist {
					chosen = i
					oldDist = dist
					if dist == 0 {
						break // Exact match found
					}
				}
			}

			// Apply the chosen trame pattern to the destination
			for ym := 0; ym < 4; ym++ {
				for xm := 0; xm < 4; xm++ {
					pen := trameM1[chosen][xm][ym]
					dest.SetPixelCpc(x+(xm<<1), y+(ym<<1), int(pen), 2)
				}
			}
		}
	}
}

// ConvertPattern calculates automatic 4x4 matrix patterns in Mode 1.
// This generates custom trame patterns based on the source image.
func (conv *AsciiConverter) ConvertPattern(source *bitmap.DirectBitmap, dest *ImageCpc, lstTrame *[]*PatternM1) int {
	// First pass: calculate palette with virtual mode 1
	oldMode := conv.params.VirtualMode
	conv.params.VirtualMode = 1

	// This would normally call ConvertPass1 and FindBestColors
	// For now, we'll assume the palette has been set up

	width := dest.Width
	height := dest.Height
	trameCount := 0

	for y := 0; y < height; y += 8 {
		for x := 0; x < width; x += 8 {
			locTrame := NewPatternM1()

			// Generate trame pattern for this 8x8 block
			for matY := 0; matY < 4; matY++ {
				for matX := 0; matX < 4; matX++ {
					pix := source.GetPixelColor(x+(matX<<1), y+(matY<<1))
					oldDist := 0x7FFFFFFF
					chosen := 0

					// Find best matching pen from the 4-color Mode 1 palette
					for i := 0; i < 4; i++ {
						// This would get color from dest.bitmapCpc.GetColorPal(i)
						// For now, we'll use a placeholder calculation
						c := bitmap.NewRgbColor(byte(i*85), byte(i*85), byte(i*85))

						rDiff := int(c.R) - int(pix.R)
						vDiff := int(c.V) - int(pix.V)
						bDiff := int(c.B) - int(pix.B)
						dist := rDiff*rDiff*conv.params.CoefR +
						       vDiff*vDiff*conv.params.CoefV +
						       bDiff*bDiff*conv.params.CoefB

						if dist < oldDist {
							chosen = i
							oldDist = dist
							if dist == 0 {
								break
							}
						}
					}

					locTrame.SetPix(matX, matY, byte(chosen))
				}
			}

			// Check if this trame pattern already exists
			found := false
			for _, t := range *lstTrame {
				if t != nil && t.IsSame(locTrame, 4) {
					found = true
					break
				}
			}

			if !found {
				*lstTrame = append(*lstTrame, locTrame)
				trameCount++
			}
		}
	}

	// Restore original mode
	conv.params.VirtualMode = oldMode
	return trameCount
}

// GetDefaultPatternM1 returns the default 16 trame patterns for Mode 1.
// Returns preset 0 from TramesAscUt. Index order is [pattern][x][y].
func GetDefaultPatternM1() [16][4][4]byte {
	return CopyTrame(0)
}