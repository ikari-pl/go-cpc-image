// Package convert provides standard mode conversion for ConvImgCpc.
// This file contains the ConvertStd function for modes 0, 1, 2, and EGX 3/4.
package convert

import (
	"github.com/ikari/go-cpc-image/pkg/bitmap"
	"github.com/ikari/go-cpc-image/pkg/cpc"
)

// Converter interface defines the contract for mode-specific converters
type Converter interface {
	Convert(source *bitmap.DirectBitmap, prm *Param, dest *ImageCpc, maxPen int, colorTable [16][272]bitmap.RgbColor)
}

// StandardConverter implements conversion for standard CPC modes (0, 1, 2) and EGX modes (3, 4)
type StandardConverter struct{}

// NewStandardConverter creates a new standard mode converter
func NewStandardConverter() *StandardConverter {
	return &StandardConverter{}
}

// Convert implements standard CPC conversion for modes 0, 1, 2, and EGX modes 3, 4.
// This replicates the C# ConvertStd function exactly.
func (sc *StandardConverter) Convert(source *bitmap.DirectBitmap, prm *Param, dest *ImageCpc, maxPen int, colorTable [16][272]bitmap.RgbColor) {
	// Calculate screen dimensions
	width := prm.GetScreenWidth()
	height := prm.GetScreenHeight()

	// Offset for import draw mode
	offsetY := 0
	if prm.ModeImpDraw && height == 544 {
		offsetY = 2
	}

	// Process each scanline pair
	for y := 0; y < height; y += 2 {
		// Calculate pixel step and max pens for this scanline
		Tx := cpc.PixelWidth(prm.VirtualMode, y, 0) // Using yEgx=0 for standard processing
		maxPen = cpc.MaxPen(prm.VirtualMode, y, 0)

		// Process each pixel group
		for x := 0; x < width; x += Tx {
			oldDist := 0x7FFFFFFF
			pix := source.GetPixelColor(x, y)
			chosen := 0

			// Find closest color in palette
			for i := 0; i < maxPen; i++ {
				c := colorTable[i][y>>1] // Use scanline-specific palette
				if prm.DisableState[i] == 0 {
					// Calculate weighted Euclidean distance
					dist := int(c.R-pix.R)*int(c.R-pix.R)*prm.CoefR +
						int(c.V-pix.V)*int(c.V-pix.V)*prm.CoefV +
						int(c.B-pix.B)*int(c.B-pix.B)*prm.CoefB

					if dist < oldDist {
						chosen = i
						oldDist = dist
						if dist == 0 {
							break // Perfect match found
						}
					}
				}
			}

			// Set pixel in CPC format
			dest.SetPixelCpc(x, y+offsetY, chosen, Tx)
		}
	}
}

// ConvertStd is the main function called by Pass2 for standard mode conversion.
// It creates a StandardConverter and calls its Convert method.
func ConvertStd(source *bitmap.DirectBitmap, prm *Param, dest *ImageCpc, maxPen int, colorTable [16][272]bitmap.RgbColor) {
	converter := NewStandardConverter()
	converter.Convert(source, prm, dest, maxPen, colorTable)
}

// EGXConverter handles EGX mode conversion with alternating scanline modes
type EGXConverter struct{}

// NewEGXConverter creates a new EGX mode converter
func NewEGXConverter() *EGXConverter {
	return &EGXConverter{}
}

// Convert implements EGX mode conversion.
// EGX modes alternate between two different modes on even/odd scanlines.
// Mode 3 (EGX1): alternates between mode 0 and mode 1
// Mode 4 (EGX2): alternates between mode 1 and mode 2
func (ec *EGXConverter) Convert(source *bitmap.DirectBitmap, prm *Param, dest *ImageCpc, maxPen int, colorTable [16][272]bitmap.RgbColor, yEgx int) {
	width := prm.GetScreenWidth()
	height := prm.GetScreenHeight()

	for y := 0; y < height; y += 2 {
		// Determine mode for this scanline based on EGX pattern
		Tx := cpc.PixelWidth(prm.VirtualMode, y, yEgx)
		maxPen = cpc.MaxPen(prm.VirtualMode, y, yEgx)

		for x := 0; x < width; x += Tx {
			oldDist := 0x7FFFFFFF
			pix := source.GetPixelColor(x, y)
			chosen := 0

			// Find closest color in scanline-specific palette
			for i := 0; i < maxPen; i++ {
				c := colorTable[i][y>>1]
				if prm.DisableState[i] == 0 {
					dist := int(c.R-pix.R)*int(c.R-pix.R)*prm.CoefR +
						int(c.V-pix.V)*int(c.V-pix.V)*prm.CoefV +
						int(c.B-pix.B)*int(c.B-pix.B)*prm.CoefB

					if dist < oldDist {
						chosen = i
						oldDist = dist
						if dist == 0 {
							break
						}
					}
				}
			}

			dest.SetPixelCpc(x, y, chosen, Tx)
		}
	}
}

// Additional converter types can be implemented here for other modes:
// - ModeXConverter for Mode X (mode 5)
// - SplitConverter for Split mode (mode 6)
// - ASCIIConverter for ASCII modes (modes 7-10)
// - SpriteConverter for sprite capture (mode 11)

// Pass2 performs the second conversion pass: reduces image to final CPC format.
// This is called by the main Convert function.
func Pass2(source *bitmap.DirectBitmap, dest *ImageCpc, prm *Param, splitCount *int) {
	var colorTable [16][272]bitmap.RgbColor
	var MemoLockState [16]int

	// Ensure display bitmap is the right size
	if dest.DisplayBmp == nil || dest.DisplayBmp.Width() != prm.GetScreenWidth() || dest.DisplayBmp.Height() != prm.GetScreenHeight() {
		dest.DisplayBmp = bitmap.NewDirectBitmap(prm.GetScreenWidth(), prm.GetScreenHeight())
	}

	// Clear the display bitmap to black (the C# version does not pre-fill;
	// ConvertStd overwrites all visible pixels via SetPixelCpc/SetHorizontalLineDouble)
	if dest.DisplayBmp != nil {
		bg := bitmap.NewRgbColor(0, 0, 0)
		for y := 0; y < dest.DisplayBmp.Height(); y++ {
			for x := 0; x < dest.DisplayBmp.Width(); x++ {
				dest.DisplayBmp.SetPixelColor(x, y, bg)
			}
		}
	}

	// Clear the CPC screen buffer before encoding
	if dest.BitmapCpc != nil {
		for i := range dest.BitmapCpc.ScreenData {
			dest.BitmapCpc.ScreenData[i] = 0
		}
	}

	// Initialize with current lock states
	copy(MemoLockState[:], prm.LockState[:])

	maxPen := cpc.MaxPen(prm.VirtualMode, 0, 0) // Max pens for actual mode

	// Handle EGX modes (3, 4)
	if prm.VirtualMode == 3 || prm.VirtualMode == 4 {
		// For EGX modes, find palette for the specific EGX line pattern
		newMax := cpc.MaxPen(prm.VirtualMode, 0, 0) // yEgx would be passed from context
		FindBestColors(newMax, MemoLockState, prm, &dest.BitmapCpc.Palette)
		for i := 0; i < newMax; i++ {
			MemoLockState[i] = 1
		}
	}

	// Handle Mode X (5) and Split mode (6)
	if prm.VirtualMode == 5 || prm.VirtualMode == 6 {
		// These would use FindBestColorsModeX and FindBestColorsModeSplit
		// For now, using standard palette search
		FindBestColors(maxPen, MemoLockState, prm, &dest.BitmapCpc.Palette)

		if prm.VirtualMode == 6 {
			maxPen = 9 // Split mode uses 9 pens
		}

		// Fill color table for Mode X/Split
		for y := 0; y < (prm.NumLines >> 1); y++ {
			for i := 0; i < maxPen; i++ {
				colorTable[i][y] = cpc.GetColor(dest.SplitModeColors[y][i], prm.CpcPlus)
			}
		}
	} else {
		// Standard CPC modes or EGX modes
		FindBestColors(maxPen, MemoLockState, prm, &dest.BitmapCpc.Palette)

		// Fill color table from bitmap palette
		// NumLines is CPC lines (200 standard), screen height is NumLines*2 pixels.
		// The converter reads colorTable[i][y>>1] where y ranges over pixel height (0..399),
		// so we must fill colorTable for all CPC line indices 0..NumLines-1.
		for line := 0; line < prm.NumLines; line++ {
			yPixel := line * 2 // pixel Y for this CPC line
			maxPen = cpc.MaxPen(prm.VirtualMode, yPixel, 0)
			for i := 0; i < maxPen; i++ {
				colorTable[i][line] = cpc.GetColor(dest.BitmapCpc.Palette[i], prm.CpcPlus)
			}
		}
	}

	// Dispatch to appropriate converter based on virtual mode
	switch prm.VirtualMode {
	case 5:
		// ConvertModeX(source, prm, dest, colorTable) - would be implemented
		// For now, fall back to standard
		ConvertStd(source, prm, dest, maxPen, colorTable)

	case 6:
		// ConvertSplit(source, prm, dest, colorTable) - would be implemented
		// For now, fall back to standard
		ConvertStd(source, prm, dest, maxPen, colorTable)

	case 7:
		// ConvertAscUt(source, prm, dest, colorTable) - would be implemented
		// For now, fall back to standard
		ConvertStd(source, prm, dest, maxPen, colorTable)

	case 8, 9, 10:
		// ConvertAscii(source, prm, dest, maxPen, colorTable) - would be implemented
		// For now, fall back to standard
		ConvertStd(source, prm, dest, maxPen, colorTable)

	default:
		// Standard modes 0, 1, 2 and EGX modes 3, 4
		ConvertStd(source, prm, dest, maxPen, colorTable)
	}
}