// Package convert provides the main image conversion pipeline for go-cpc-image.
// This file contains the main Convert function that orchestrates Pass1 and Pass2.
package convert

import (
	"github.com/ikari/go-cpc-image/pkg/bitmap"
	"github.com/ikari/go-cpc-image/pkg/cpc"
	"github.com/ikari/go-cpc-image/pkg/render"
)

// Distance represents different color distance calculation methods
type Distance int

const (
	DISTANCE_SUP       Distance = 0 // Max difference (supremum distance)
	DISTANCE_EUCLIDE   Distance = 1 // Euclidean distance
	DISTANCE_MANHATTAN Distance = 2 // Manhattan distance
)

// FrequencyTable stores color frequency counts for palette optimization
// First dimension: color index (4096 for CPC+, 27 for standard CPC)
// Second dimension: scanline (272 max for overscan)
var colorFrequency [4096][272]int

// ContrastTable lookup table for contrast adjustment
var contrastTable [256]byte

// ImageCpc represents a CPC image with screen memory and palette
type ImageCpc struct {
	BitmapCpc   *render.BitmapCpc   // CPC screen memory
	DisplayBmp  *bitmap.DirectBitmap // Display bitmap for intermediate conversion
	SplitModeColors    [136][16]int        // Mode X/Split palette data (136 = 272/2)
	Width       int                 // Image width in pixels (for compatibility)
	Height      int                 // Image height in pixels (for compatibility)
	Mode        int                 // CPC mode (for compatibility)
}

// MinMaxByte clamps a value to byte range [0, 255]
func MinMaxByte(value float64) byte {
	if value >= 0 {
		if value <= 255 {
			return byte(value)
		}
		return 255
	}
	return 0
}

// Convert performs the complete conversion from source image to CPC format.
// This is the main entry point replicating the C# Convert function.
// Returns the number of unique colors found in the source image.
func Convert(source *bitmap.DirectBitmap, dest *ImageCpc, prm *Param, noInfo bool) int {
	// Clear frequency table
	for i := range colorFrequency {
		for j := range colorFrequency[i] {
			colorFrequency[i][j] = 0
		}
	}

	// Build contrast lookup table
	c := float64(prm.PctContrast) / 100.0
	for i := 0; i < 256; i++ {
		contrastTable[i] = MinMaxByte(((float64(i)/255.0-0.5)*c+0.5)*255.0)
	}

	// Sync BitmapCpc mode and dimensions with params so EncodeToCpc/DrawBitmap
	// use the same pixel stride as the converter (SetPixelCpc).
	if dest.BitmapCpc != nil {
		dest.BitmapCpc.VirtualMode = prm.VirtualMode
		dest.BitmapCpc.YEgx = prm.YEgx
		dest.BitmapCpc.NumCol = prm.NumCols
		dest.BitmapCpc.NumLig = prm.NumLines
	}

	// Apply k-means palette reduction if requested
	if prm.Filter {
		QuantizePalette(source, prm)
	}

	// Pass 1: Color reduction and frequency counting
	ConvertPass1(source, prm)

	// Calculate number of unique colors in image
	numColors := 0
	for i := 0; i < len(colorFrequency); i++ {
		found := false
		for y := 0; y < 272; y++ {
			if !found && colorFrequency[i][y] > 0 {
				numColors++
				found = true
			}
		}
	}

	// Pass 2: Final conversion to CPC format
	splitCount := 0
	Pass2(source, dest, prm, &splitCount)

	// After Pass2, encode the display bitmap to CPC screen memory
	if dest.DisplayBmp != nil && dest.BitmapCpc != nil {
		dest.BitmapCpc.EncodeToCpc(dest.DisplayBmp, false, 0)
	}

	// Set info if not suppressed
	if !noInfo {
		// dest.main.SetInfo would be here in C#, but we don't have UI
		if splitCount > 0 {
			// Additional split color info would go here
		}
	}

	dest.BitmapCpc.IsComputed = true
	return numColors
}

// GetFrequencyTable returns the color frequency table for diagnostic purposes.
// Returns a map of color index -> total pixel count across all lines.
func GetFrequencyTable(maxColors int) map[int]int {
	result := make(map[int]int)
	for i := 0; i < maxColors; i++ {
		total := 0
		for y := 0; y < 272; y++ {
			total += colorFrequency[i][y]
		}
		if total > 0 {
			result[i] = total
		}
	}
	return result
}

// SetPixelCpc sets a pixel in the display bitmap using the palette color.
// This matches the C# ImageCpc.SetPixelCpc behavior exactly:
//   BmpLock.SetHorizontalLineDouble(xPos, yPos, tx, Cpc.PaletteColor(...))
func (dest *ImageCpc) SetPixelCpc(x, y, pen, tx int) {
	if dest.DisplayBmp == nil || dest.BitmapCpc == nil {
		return
	}

	var colorIdx int
	if dest.BitmapCpc.VirtualMode == 5 || dest.BitmapCpc.VirtualMode == 6 {
		// Mode X/Split: use SplitModeColors palette
		yIdx := y >> 1
		if yIdx < len(dest.SplitModeColors) && pen < len(dest.SplitModeColors[yIdx]) {
			colorIdx = dest.SplitModeColors[yIdx][pen]
		}
	} else {
		// Standard modes: use main palette
		if pen < len(dest.BitmapCpc.Palette) {
			colorIdx = dest.BitmapCpc.Palette[pen]
		}
	}

	// Match C# PaletteColor: invalid indices clamp to 0
	rgbColor := cpc.PaletteColor(colorIdx, dest.BitmapCpc.CpcPlus)
	dest.DisplayBmp.SetHorizontalLineDouble(x, y, tx, rgbColor)
}