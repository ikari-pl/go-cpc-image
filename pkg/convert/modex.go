// Package convert provides Mode X conversion functionality.
// Mode X uses per-line palette changes with 4 colors: 2 fixed + 2 variable per line.
package convert

import (
	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
	"github.com/ikari-pl/go-cpc-image/pkg/cpc"
)

// SimpleCpcImage represents a CPC image with conversion context for Mode X.
// This is equivalent to the C# ImageCpc class used in conversion.
type SimpleCpcImage struct {
	Width     int
	Height    int
	Mode      int
	Palette   [16]int
	PixelData [][]int // [y][x] -> pen number
}

// NewSimpleCpcImage creates a new CPC image for conversion.
func NewSimpleCpcImage(width, height, mode int) *SimpleCpcImage {
	img := &SimpleCpcImage{
		Width:     width,
		Height:    height,
		Mode:      mode,
		Palette:   cpc.DefaultPalette,
		PixelData: make([][]int, height),
	}
	for y := 0; y < height; y++ {
		img.PixelData[y] = make([]int, width)
	}
	return img
}

// SetPixelCpc sets a pixel in CPC format with the given pen and pixel width.
// This replicates the C# SetPixelCpc method behavior.
func (img *SimpleCpcImage) SetPixelCpc(x, y, pen, tx int) {
	if y >= img.Height || x >= img.Width {
		return
	}

	// Set pixels with the given width (tx)
	for i := 0; i < tx && x+i < img.Width; i++ {
		img.PixelData[y][x+i] = pen
	}
}

// ModeXConverter handles Mode X conversion with per-line palette optimization.
type ModeXConverter struct {
	params       *Settings
	colorFrequency  [4096][272]int // Color frequency table [color][line]
	cpcPlus      bool
}

// NewModeXConverter creates a new Mode X converter with the given parameters.
func NewModeXConverter(params *Settings, cpcPlus bool) *ModeXConverter {
	return &ModeXConverter{
		params:  params,
		cpcPlus: cpcPlus,
	}
}

// FindBestColorsModeX finds optimal colors for Mode X conversion.
// Mode X: 2 fixed colors + 2 variable colors per line.
// Returns the number of unique colors found.
func (conv *ModeXConverter) FindBestColorsModeX(colMode5 *[272][16]int, lockState [16]int, yMax int) int {
	findMax := 27
	if conv.cpcPlus {
		findMax = 4096
	}

	// The first two colors are "fixed"
	for c := 0; c < 2; c++ {
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
	// Colors 2 and 3 are variable per line
	for c := 2; c < 4; c++ {
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

// ConvertModeX performs Mode X conversion on the source image.
// Mode X uses 4 colors per line: 2 fixed globally + 2 variable per line.
func (conv *ModeXConverter) ConvertModeX(source *bitmap.DirectBitmap, dest *ImageCpc, colorTable *[4][272]bitmap.RgbColor) {
	width := dest.Width
	height := dest.Height

	for y := 0; y < height; y += 2 {
		for x := 0; x < width; x += 2 {
			oldDist := 0x7FFFFFFF
			pix := source.GetPixelColor(x, y)
			chosen := 0

			// Find the best matching color among the 4 available for this line
			for i := 0; i < 4; i++ {
				c := colorTable[i][y>>1]
				if c.R == 0 && c.V == 0 && c.B == 0 && i > 0 {
					continue // Skip invalid colors
				}

				// Calculate squared Euclidean distance with coefficients
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
						break // Exact match found
					}
				}
			}

			// Apply Y offset for import draw mode if needed
			offsetY := 0
			if conv.params.ModeImpDraw && height == 544 {
				offsetY = 2
			}

			dest.SetPixelCpc(x, y+offsetY, chosen, 2)
		}
	}
}

// UpdateCoulTrouvee updates the color frequency table used for optimization.
// This should be called during the first pass to count color usage per line.
func (conv *ModeXConverter) UpdateCoulTrouvee(color, line, count int) {
	if color >= 0 && color < 4096 && line >= 0 && line < 272 {
		conv.colorFrequency[color][line] = count
	}
}

// GetCoulTrouvee returns the color frequency at the specified color and line.
func (conv *ModeXConverter) GetCoulTrouvee(color, line int) int {
	if color >= 0 && color < 4096 && line >= 0 && line < 272 {
		return conv.colorFrequency[color][line]
	}
	return 0
}

// ClearCoulTrouvee resets the color frequency table.
func (conv *ModeXConverter) ClearCoulTrouvee() {
	for i := range conv.colorFrequency {
		for j := range conv.colorFrequency[i] {
			conv.colorFrequency[i][j] = 0
		}
	}
}