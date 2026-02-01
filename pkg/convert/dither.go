// Package convert provides image conversion functionality for go-cpc-image.
// This file contains the dithering system: matrix definitions, error diffusion, and ordered dithering.
package convert

import (
	"math"

	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
)

// Bitmap represents a minimal bitmap interface for dithering operations
type Bitmap interface {
	Width() int
	Height() int
	GetPixelColor(x, y int) bitmap.RgbColor
	SetPixel(x, y int, color bitmap.RgbColor)
}

// DitherParam represents the parameters needed for dithering operations
type DitherParam struct {
	CpcPlus  bool
	Pct      int
	Method  string // Dithering method name
	DiffErr  bool   // Error diffusion flag
}

// Dithering matrices used for ordered and error-diffusion dithering
var (
	// Floyd-Steinberg (2x2)
	floyd = [][]float64{
		{7, 3},
		{5, 1},
	}

	// Bayer matrices
	bayer1 = [][]float64{
		{1, 3},
		{4, 2},
	}

	bayer2 = [][]float64{
		{0, 12, 3, 15},
		{8, 4, 11, 7},
		{2, 14, 1, 13},
		{10, 6, 9, 5},
	}

	bayer3 = [][]float64{
		{1, 9, 3, 11},
		{13, 5, 15, 7},
		{4, 12, 2, 10},
		{16, 8, 14, 6},
	}

	// Ordered dithering matrices
	ord1 = [][]float64{
		{1, 3},
		{2, 4},
	}

	ord2 = [][]float64{
		{8, 3, 4},
		{6, 1, 2},
		{7, 5, 9},
	}

	ord3 = [][]float64{
		{0, 8, 2, 10},
		{12, 4, 14, 6},
		{3, 11, 1, 9},
		{15, 7, 13, 5},
	}

	ord4 = [][]float64{
		{0, 48, 12, 60, 3, 51, 15, 63},
		{32, 16, 44, 28, 35, 19, 47, 31},
		{8, 56, 4, 52, 11, 59, 7, 55},
		{40, 24, 36, 20, 43, 27, 39, 23},
		{2, 50, 14, 62, 1, 49, 13, 61},
		{34, 18, 46, 30, 33, 17, 45, 29},
		{10, 58, 6, 54, 9, 57, 5, 53},
		{42, 26, 38, 22, 41, 25, 37, 21},
	}

	// ZigZag patterns
	zigzag1 = [][]float64{
		{0, 4, 0},
		{3, 0, 1},
		{0, 2, 0},
	}

	zigzag2 = [][]float64{
		{0, 4, 2, 0},
		{6, 0, 5, 3},
		{0, 7, 1, 0},
	}

	zigzag3 = [][]float64{
		{0, 0, 0, 7, 0},
		{0, 2, 6, 9, 8},
		{3, 0, 1, 5, 0},
		{0, 4, 0, 0, 0},
	}

	// Test patterns
	test0 = [][]float64{
		{1.0, 16.0, 1.0, 16.0},
		{16.0, 1.0, 16.0, 1.0},
		{1, 16, 1, 16},
		{16, 1, 16, 1},
	}

	test1 = [][]float64{
		{1, 4, 1, 4},
		{4, 1, 4, 1},
		{1, 4, 1, 4},
		{4, 1, 4, 1},
	}

	test2 = [][]float64{
		{8, 1, 8, 1},
		{1, 8, 1, 8},
		{8, 1, 8, 1},
		{1, 8, 1, 8},
	}

	test3 = [][]float64{
		{8, 16, 16, 8},
		{16, 8, 8, 16},
	}

	test4 = [][]float64{
		{0, 3},
		{0, 5},
		{7, 1},
	}

	test5 = [][]float64{
		{0, 0, 7},
		{3, 5, 1},
	}

	test6 = [][]float64{
		{1, 9, 4, 7},
		{4, 7, 1, 9},
		{1, 9, 4, 7},
		{4, 7, 1, 9},
	}

	test7 = [][]float64{
		{12, 11, 0},
		{13, 10, 19},
		{11, 13, 0},
	}

	test8 = [][]float64{
		{3, 7, 6, 2},
		{5, 4, 1, 0},
	}

	test9 = [][]float64{
		{1, 5, 10, 14},
		{3, 7, 8, 12},
		{13, 9, 6, 2},
		{15, 11, 4, 0},
	}
)

// Available dithering methods:
//
// Floyd-Steinberg and Bayer patterns:
//   - Floyd-Steinberg (2x2): Error diffusion dithering
//   - Bayer 1 (2X2): Small Bayer pattern
//   - Bayer 2 (4x4): Medium Bayer pattern
//   - Bayer 3 (4X4): Alternative medium Bayer pattern
//
// Ordered dithering patterns:
//   - Ordered 1 (2x2): Simple ordered dithering
//   - Ordered 2 (3x3): Medium ordered dithering
//   - Ordered 3 (4x4): Complex ordered dithering
//
// ZigZag patterns:
//   - ZigZag1 (3x3): Small zigzag pattern
//   - ZigZag2 (4x3): Medium zigzag pattern
//   - ZigZag3 (5x4): Large zigzag pattern
//
// Test patterns (Test0-Test9):
//   - Various experimental dithering matrices for testing
var DitherMatrices = map[string][][]float64{
	"Floyd-Steinberg (2x2)": floyd,
	"Bayer 1 (2X2)":         bayer1,
	"Bayer 2 (4x4)":         bayer2,
	"Bayer 3 (4X4)":         bayer3,
	"Ordered 1 (2x2)":       ord1,
	"Ordered 2 (3x3)":       ord3,
	"Ordered 3 (4x4)":       ord4,
	"ZigZag1 (3x3)":         zigzag1,
	"ZigZag2 (4x3)":         zigzag2,
	"ZigZag3 (5x4)":         zigzag3,
	"Checker Hi (4x4)":        test0,
	"Checker Lo (4x4)":        test1,
	"Checker Mid (4x4)":       test2,
	"Diamond (2x4)":           test3,
	"Stucki Lite (3x2)":       test4,
	"Floyd-Steinberg 2 (2x3)": test5,
	"Dispersed (4x4)":         test6,
	"Weighted Left (3x3)":     test7,
	"Ramp (2x4)":              test8,
	"Gradient (4x4)":          test9,
}

// Current dithering matrix, set by SetMatDither
var matDither [][]float64

// minMaxByte clamps a float64 value to the valid byte range [0, 255]
func minMaxByte(value float64) uint8 {
	if value >= 0 {
		if value <= 255 {
			return uint8(value)
		}
		return 255
	}
	return 0
}

// SetMatDither sets up the dithering matrix based on parameters.
// Returns the effective percentage value used, or 0 if dithering is disabled.
// CPC+ uses 3-bit shift (8x) for finer color granularity; standard CPC uses 1-bit shift (2x).
func SetMatDither(prm DitherParam) int {
	var pct int
	if prm.CpcPlus {
		pct = prm.Pct << 3 // Multiply by 8 for CPC+
	} else {
		pct = prm.Pct << 1 // Multiply by 2 for regular CPC
	}

	matrix, exists := DitherMatrices[prm.Method]
	if pct > 0 && exists {
		// Create a copy of the matrix to avoid modifying the original
		height := len(matrix)
		width := len(matrix[0])
		matDither = make([][]float64, height)
		for i := range matDither {
			matDither[i] = make([]float64, width)
			copy(matDither[i], matrix[i])
		}

		// Calculate sum of matrix values
		var sum float64
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				sum += matDither[y][x]
			}
		}

		// Normalize matrix values based on percentage
		// Normalize: scale each cell by (pct / sum) so the matrix sums to pct
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				matDither[y][x] = (matDither[y][x] * float64(pct)) / sum
			}
		}
	} else {
		pct = 0
		matDither = nil
	}

	return pct
}

// DoDitherFull applies dithering to a bitmap region.
// For error diffusion (diffErr=true), spreads quantization error to neighboring source pixels.
// For ordered dithering (diffErr=false), applies a matrix offset to the current pixel.
// Returns the (potentially modified) pixel color — important for ordered dithering
// since Go passes structs by value.
func DoDitherFull(source Bitmap, xPix, yPix, tx int, p, chosen bitmap.RgbColor, diffErr bool) bitmap.RgbColor {
	if matDither == nil {
		return p
	}

	if diffErr {
		// Error diffusion: propagates quantization error (p - chosen) to
		// neighboring source pixels, weighted by matDither[x,y] / 256.
		matHeight := len(matDither)
		matWidth := len(matDither[0])

		for y := 0; y < matHeight; y++ {
			for x := 0; x < matWidth; x++ {
				pixelX := xPix + tx*x
				pixelY := yPix + (y << 1) // y * 2

				if pixelX < source.Width() && pixelY < source.Height() {
					pix := source.GetPixelColor(pixelX, pixelY)

					// Add weighted quantization error to each channel
					pix.R = minMaxByte(float64(pix.R) + float64(int(p.R)-int(chosen.R))*matDither[y][x]/256.0)
					pix.V = minMaxByte(float64(pix.V) + float64(int(p.V)-int(chosen.V))*matDither[y][x]/256.0)
					pix.B = minMaxByte(float64(pix.B) + float64(int(p.B)-int(chosen.B))*matDither[y][x]/256.0)

					source.SetPixel(pixelX, pixelY, pix)
				}
			}
		}
	} else {
		// Ordered dithering: add a position-dependent offset from the matrix.
		matHeight := len(matDither)
		matWidth := len(matDither[0])

		xm := (xPix / tx) % matWidth
		ym := (yPix >> 1) % matHeight

		// Apply the matrix value as an additive bias before quantization
		ditherValue := matDither[ym][xm]
		p.R = minMaxByte(float64(p.R) + ditherValue)
		p.V = minMaxByte(float64(p.V) + ditherValue)
		p.B = minMaxByte(float64(p.B) + ditherValue)
	}
	return p
}

// GetAvailableMethods returns a list of all available dithering method names
// in a fixed logical order (not random map iteration order).
func GetAvailableMethods() []string {
	return []string{
		"Floyd-Steinberg (2x2)",
		"Bayer 1 (2X2)",
		"Bayer 2 (4x4)",
		"Bayer 3 (4X4)",
		"Ordered 1 (2x2)",
		"Ordered 2 (3x3)",
		"Ordered 3 (4x4)",
		"ZigZag1 (3x3)",
		"ZigZag2 (4x3)",
		"ZigZag3 (5x4)",
		"Checker Lo (4x4)",
		"Checker Mid (4x4)",
		"Checker Hi (4x4)",
		"Diamond (2x4)",
		"Dispersed (4x4)",
		"Gradient (4x4)",
		"Ramp (2x4)",
		"Stucki Lite (3x2)",
		"Floyd-Steinberg 2 (2x3)",
		"Weighted Left (3x3)",
	}
}

// GetMatrixSize returns the dimensions (width, height) of a dithering matrix
func GetMatrixSize(method string) (width, height int, exists bool) {
	matrix, exists := DitherMatrices[method]
	if !exists {
		return 0, 0, false
	}
	return len(matrix[0]), len(matrix), true
}

// CalculateDistance calculates the Euclidean distance between two RGB colors
// This can be useful for color quantization algorithms that work with dithering
func CalculateDistance(c1, c2 bitmap.RgbColor) float64 {
	dr := float64(int(c1.R) - int(c2.R))
	dv := float64(int(c1.V) - int(c2.V))
	db := float64(int(c1.B) - int(c2.B))
	return math.Sqrt(dr*dr + dv*dv + db*db)
}