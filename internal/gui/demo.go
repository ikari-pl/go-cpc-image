// Package gui provides demo functionality for testing the GUI without full conversion.
package gui

import (
	"image"
	"image/color"

	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
)

// CreateDemoImage creates a simple test image for GUI demonstration.
func CreateDemoImage() *bitmap.DirectBitmap {
	width, height := 320, 200
	img := bitmap.NewDirectBitmap(width, height)

	// Create a simple test pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Create colorful stripes
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b := uint8(((x + y) * 255) / (width + height))

			color := bitmap.NewRgbColor(r, g, b)
			img.SetPixelColor(x, y, color)
		}
	}

	return img
}

// CreateDemoCpcImage creates a demo CPC-style image for preview.
func CreateDemoCpcImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Create CPC-style blocks to simulate the chunky pixel look
	blockSize := 4
	for y := 0; y < height; y += blockSize {
		for x := 0; x < width; x += blockSize {
			// Use CPC-style colors (limited palette)
			colorIndex := ((x / blockSize) + (y / blockSize)) % 4
			var c color.RGBA

			switch colorIndex {
			case 0:
				c = color.RGBA{0, 0, 0, 255}       // Black
			case 1:
				c = color.RGBA{0, 255, 255, 255}   // Cyan
			case 2:
				c = color.RGBA{255, 0, 255, 255}   // Magenta
			case 3:
				c = color.RGBA{255, 255, 255, 255} // White
			}

			// Fill the block
			for dy := 0; dy < blockSize && y+dy < height; dy++ {
				for dx := 0; dx < blockSize && x+dx < width; dx++ {
					img.Set(x+dx, y+dy, c)
				}
			}
		}
	}

	return img
}