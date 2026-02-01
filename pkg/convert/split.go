// Package convert provides Split-raster mode conversion functionality.
// Split mode uses 3 fixed colors + 6 per-line colors with hardware split-raster effects.
package convert

import (
	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
)

// SplitConverter handles Split-raster mode conversion.
// Split mode: 3 fixed colors globally + 6 variable colors per line = 9 total colors per line.
type SplitConverter struct {
	params       *Param
	colorFrequency  [4096][272]int // Color frequency table [color][line]
	cpcPlus      bool
}

// NewSplitConverter creates a new Split-raster converter with the given parameters.
func NewSplitConverter(params *Param, cpcPlus bool) *SplitConverter {
	return &SplitConverter{
		params:  params,
		cpcPlus: cpcPlus,
	}
}

// FindBestColorsModeSplit finds optimal colors for Split-raster mode conversion.
// Split mode: 3 fixed colors + 6 variable colors per line = 9 colors total.
// Returns the number of unique colors found.
func (conv *SplitConverter) FindBestColorsModeSplit(colMode5 *[272][16]int, lockState [16]int, yMax int) int {
	findMax := 27
	if conv.cpcPlus {
		findMax = 4096
	}

	// The first three colors are "fixed"
	for c := 0; c < 3; c++ {
		if lockState[c] > 0 {
			// Color is locked - use the current palette color
			for y := 0; y < 272; y++ {
				conv.colorFrequency[conv.params.Palette[c]][y] = 0
				colMode5[y][c] = conv.params.Palette[c]
			}
		} else {
			// Find the most frequent color across all lines for this fixed position
			maxFreq := 0
			bestColor := 0
			for i := 0; i < findMax; i++ {
				valFound := 0
				for y := 0; y < yMax>>1; y++ {
					valFound += conv.colorFrequency[i][y]
				}
				if maxFreq < valFound {
					maxFreq = valFound
					bestColor = i
				}
			}
			conv.params.Palette[c] = bestColor

			// Clear usage for this color and assign to all lines
			for y := 0; y < 272; y++ {
				conv.colorFrequency[conv.params.Palette[c]][y] = 0
				colMode5[y][c] = conv.params.Palette[c]
			}
		}
	}

	// Search colors per line
	// Colors 3-8 are variable per line (6 split colors)
	for c := 3; c < 9; c++ {
		for y := 0; y < yMax>>1; y++ {
			maxFreq := 0
			bestColor := 0

			// Search within tracking window around current line
			for i := 0; i < findMax; i++ {
				for deltaY := -(conv.params.TrackModeX << 1); deltaY <= (conv.params.TrackModeX << 1); deltaY++ {
					lineY := y + deltaY
					if lineY >= 0 && lineY < (yMax>>1) && maxFreq < conv.colorFrequency[i][lineY] {
						maxFreq = conv.colorFrequency[i][lineY]
						bestColor = i
					}
				}
			}
			colMode5[y][c] = bestColor
			conv.colorFrequency[colMode5[y][c]][y] = 0
		}
	}

	// Count unique colors used
	cFound := [4096]bool{}
	numColors := 0
	for i := 0; i < 16; i++ {
		for y := 0; y < 272; y++ {
			if !cFound[colMode5[y][i]] {
				cFound[colMode5[y][i]] = true
				numColors++
			}
		}
	}
	return numColors
}

// ConvertSplit performs Split-raster mode conversion on the source image.
// Split mode simulates hardware raster splits with color changes within scanlines.
func (conv *SplitConverter) ConvertSplit(source *bitmap.DirectBitmap, dest *ImageCpc, colorTable *[9][272]bitmap.RgbColor) {
	width := dest.Width
	height := dest.Height

	for y := 0; y < height; y += 2 {
		tailleSplit := 0   // Current split length in pixels
		splitColor := -1     // Current split color (-1 = none active)

		for x := 0; x < width; x += 2 {
			oldDist := 0x7FFFFFFF
			pix := source.GetPixelColor(x, y)
			chosen := 0
			memoPen := 0

			// Test all 9 available colors for this line
			for i := 0; i < 9; i++ {
				memoPen = i

				// If we have an active split and haven't reached 32 pixels yet,
				// prefer the current split color for colors > 2 (split colors)
				if splitColor != -1 && tailleSplit < 32 && i > 2 {
					memoPen = splitColor
				}

				c := colorTable[memoPen][y>>1]

				// Calculate squared Euclidean distance with coefficients
				rDiff := int(c.R) - int(pix.R)
				vDiff := int(c.V) - int(pix.V)
				bDiff := int(c.B) - int(pix.B)
				dist := rDiff*rDiff*conv.params.CoefR +
				       vDiff*vDiff*conv.params.CoefV +
				       bDiff*bDiff*conv.params.CoefB

				if dist < oldDist {
					chosen = memoPen
					oldDist = dist
					if dist == 0 || memoPen == splitColor {
						break // Exact match or continuing current split
					}
				}
			}

			// Manage split state
			if chosen > 2 { // Split colors (3-8)
				if splitColor != chosen {
					// Starting new split or changing split color
					splitColor = chosen
					tailleSplit = 0
				}
			}
			// Always increment split length (even for fixed colors 0-2)
			tailleSplit++

			dest.SetPixelCpc(x, y, chosen, 2)
		}
	}
}

// UpdateCoulTrouvee updates the color frequency table used for optimization.
// This should be called during the first pass to count color usage per line.
func (conv *SplitConverter) UpdateCoulTrouvee(color, line, count int) {
	if color >= 0 && color < 4096 && line >= 0 && line < 272 {
		conv.colorFrequency[color][line] = count
	}
}

// GetCoulTrouvee returns the color frequency at the specified color and line.
func (conv *SplitConverter) GetCoulTrouvee(color, line int) int {
	if color >= 0 && color < 4096 && line >= 0 && line < 272 {
		return conv.colorFrequency[color][line]
	}
	return 0
}

// ClearCoulTrouvee resets the color frequency table.
func (conv *SplitConverter) ClearCoulTrouvee() {
	for i := range conv.colorFrequency {
		for j := range conv.colorFrequency[i] {
			conv.colorFrequency[i][j] = 0
		}
	}
}