// Package cpc provides CPC hardware constants and functions.
package cpc

import "github.com/ikari-pl/go-cpc-image/pkg/bitmap"

// CPC luminance levels - 3 levels per RGB channel for 27 total colors
const (
	Lum0 = 0x00
	Lum1 = 0x66
	Lum2 = 0xFF
)

// CpcRgbPalette contains the 27 standard CPC colors.
// Each color uses 3 luminance levels (Lum0, Lum1, Lum2) for R, V (green), B channels.
var CpcRgbPalette = [27]bitmap.RgbColor{
	bitmap.NewRgbColor(Lum0, Lum0, Lum0), // 0: Black
	bitmap.NewRgbColor(Lum0, Lum0, Lum1), // 1: Blue
	bitmap.NewRgbColor(Lum0, Lum0, Lum2), // 2: Bright Blue
	bitmap.NewRgbColor(Lum1, Lum0, Lum0), // 3: Red
	bitmap.NewRgbColor(Lum1, Lum0, Lum1), // 4: Magenta
	bitmap.NewRgbColor(Lum1, Lum0, Lum2), // 5: Mauve
	bitmap.NewRgbColor(Lum2, Lum0, Lum0), // 6: Bright Red
	bitmap.NewRgbColor(Lum2, Lum0, Lum1), // 7: Purple
	bitmap.NewRgbColor(Lum2, Lum0, Lum2), // 8: Bright Magenta
	bitmap.NewRgbColor(Lum0, Lum1, Lum0), // 9: Green
	bitmap.NewRgbColor(Lum0, Lum1, Lum1), // 10: Cyan
	bitmap.NewRgbColor(Lum0, Lum1, Lum2), // 11: Sky Blue
	bitmap.NewRgbColor(Lum1, Lum1, Lum0), // 12: Yellow
	bitmap.NewRgbColor(Lum1, Lum1, Lum1), // 13: White
	bitmap.NewRgbColor(Lum1, Lum1, Lum2), // 14: Pastel Blue
	bitmap.NewRgbColor(Lum2, Lum1, Lum0), // 15: Orange
	bitmap.NewRgbColor(Lum2, Lum1, Lum1), // 16: Pink
	bitmap.NewRgbColor(Lum2, Lum1, Lum2), // 17: Pastel Magenta
	bitmap.NewRgbColor(Lum0, Lum2, Lum0), // 18: Bright Green
	bitmap.NewRgbColor(Lum0, Lum2, Lum1), // 19: Sea Green
	bitmap.NewRgbColor(Lum0, Lum2, Lum2), // 20: Bright Cyan
	bitmap.NewRgbColor(Lum1, Lum2, Lum0), // 21: Lime
	bitmap.NewRgbColor(Lum1, Lum2, Lum1), // 22: Pastel Green
	bitmap.NewRgbColor(Lum1, Lum2, Lum2), // 23: Pastel Cyan
	bitmap.NewRgbColor(Lum2, Lum2, Lum0), // 24: Bright Yellow
	bitmap.NewRgbColor(Lum2, Lum2, Lum1), // 25: Pastel Yellow
	bitmap.NewRgbColor(Lum2, Lum2, Lum2), // 26: Bright White
}

// CpcVGA is the VGA to CPC color mapping string
const CpcVGA = "TDU\\X]LEMVFW^@_NGORBSZY[JCK"

// PaletteColor returns the RGB color value for the given palette index.
// For CPC+ mode, it converts 12-bit GRB to RGB. For standard CPC, uses CpcRgbPalette table.
func PaletteColor(c int, cpcPlus bool) int {
	if cpcPlus {
		// CPC+ 12-bit color: 0x0VBR where V=green(vert), B=blue, R=red (each 4 bits)
		r := (c & 0x0F) * 17         // Red: bits 3-0
		g := ((c & 0xF00) >> 8) * 17 // Green: bits 11-8
		b := ((c & 0xF0) >> 4) * 17  // Blue: bits 7-4
		return b + (g << 8) + (r << 16)
	}

	// Standard CPC 27-color palette
	if c < 0 || c >= 27 {
		c = 0
	}
	return CpcRgbPalette[c].GetColor()
}

// GetColor returns an RgbColor for the given palette index or CPC+ color value.
func GetColor(c int, cpcPlus bool) bitmap.RgbColor {
	if c >= 0xFFFF {
		return bitmap.RgbColor{}
	}

	if cpcPlus {
		// CPC+ 12-bit format: 0x0VBR where V=green, B=blue, R=red
		r := uint8((c & 0x0F) * 17)          // Red: bits 3-0
		g := uint8(((c & 0xF00) >> 8) * 17)  // Green: bits 11-8
		b := uint8(((c & 0xF0) >> 4) * 17)   // Blue: bits 7-4
		return bitmap.NewRgbColor(r, g, b)
	}

	// Standard CPC palette
	if c < 0 || c >= 27 {
		c = 0
	}
	return CpcRgbPalette[c]
}

// GetPenColor finds the pen number for a given color in a bitmap.
// This searches the current palette to find the matching pen.
func GetPenColor(bmp *bitmap.DirectBitmap, x, y int, palette [16]int, cpcPlus bool) int {
	col := bmp.GetPixelColor(x, y)

	if cpcPlus {
		// CPC+ mode: match 4-bit precision per channel
		for pen := 0; pen < 16; pen++ {
			if palette[pen] == 0xFFFF {
				continue
			}
			// Extract VBR nibbles from palette entry (0x0VBR)
			palG := (palette[pen] >> 8) & 0x0F
			palB := (palette[pen] >> 4) & 0x0F
			palR := palette[pen] & 0x0F

			// Compare with pixel color (4-bit precision)
			if (col.V>>4) == uint8(palG) && (col.R>>4) == uint8(palR) && (col.B>>4) == uint8(palB) {
				return pen
			}
		}
	} else {
		// Standard CPC mode: exact color match
		for pen := 0; pen < 16; pen++ {
			if palette[pen] == 0xFFFF || palette[pen] >= 27 {
				continue
			}
			fixedCol := CpcRgbPalette[palette[pen]]
			if fixedCol.R == col.R && fixedCol.V == col.V && fixedCol.B == col.B {
				return pen
			}
		}
	}

	return 0 // Default to pen 0 if no match found
}

// DefaultPalette returns the default CPC palette used in most modes.
var DefaultPalette = [16]int{1, 24, 20, 6, 26, 0, 2, 7, 10, 12, 14, 16, 18, 22, 1, 14}

// PatternM1 represents the 4x4 trame patterns for Mode 1 ASCII art.
// This is equivalent to the C# trameM1 array.
type PatternM1 [16][4][4]byte

// SpritesHard represents CPC+ hardware sprites.
// 8 banks, 16 sprites per bank, 16x16 pixels per sprite.
type SpritesHard [8][16][16][16]byte

// PaletteSprite represents the sprite palette (16 colors).
type PaletteSprite [16]int

// GetCpcRgb returns the standard CPC 27-color palette as a slice.
func GetCpcRgb() []bitmap.RgbColor {
	result := make([]bitmap.RgbColor, 27)
	copy(result, CpcRgbPalette[:])
	return result
}

// GetColorFromCpcPlus converts a CPC+ 12-bit color value to RgbColor.
func GetColorFromCpcPlus(colorValue int) bitmap.RgbColor {
	// CPC+ 12-bit format: 0x0VBR where V=green, B=blue, R=red
	r := uint8((colorValue & 0x0F) * 17)            // Red: bits 3-0
	g := uint8(((colorValue & 0xF00) >> 8) * 17)    // Green: bits 11-8
	b := uint8(((colorValue & 0xF0) >> 4) * 17)     // Blue: bits 7-4
	return bitmap.NewRgbColor(r, g, b)
}