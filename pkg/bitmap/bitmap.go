// Package bitmap provides bitmap handling compatible with CPC image conversion.
package bitmap

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

// DirectBitmap provides a direct pixel buffer for efficient image manipulation.
// This is equivalent to the C# DirectBitmap class, using image.RGBA internally.
type DirectBitmap struct {
	img    *image.RGBA
	width  int
	height int
	// Tps field from C# - not used in current implementation but kept for compatibility
	Tps int
}

// NewDirectBitmap creates a new DirectBitmap with the specified dimensions.
func NewDirectBitmap(width, height int) *DirectBitmap {
	return &DirectBitmap{
		img:    image.NewRGBA(image.Rect(0, 0, width, height)),
		width:  width,
		height: height,
	}
}

// NewDirectBitmapFromFile creates a DirectBitmap by loading an image from a file.
func NewDirectBitmapFromFile(filename string) (*DirectBitmap, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file %s: %w", filename, err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image %s: %w", filename, err)
	}

	return NewFromImage(img), nil
}

// NewFromImage creates a DirectBitmap from an image.Image.
func NewFromImage(img image.Image) *DirectBitmap {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	db := &DirectBitmap{
		img:    image.NewRGBA(bounds),
		width:  width,
		height: height,
	}

	// Copy pixels from decoded image to RGBA buffer
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
			db.img.Set(x, y, c)
		}
	}

	return db
}

// Width returns the bitmap width.
func (db *DirectBitmap) Width() int {
	return db.width
}

// Height returns the bitmap height.
func (db *DirectBitmap) Height() int {
	return db.height
}

// Length returns the total number of pixels (width * height).
func (db *DirectBitmap) Length() int {
	return db.width * db.height
}

// Image returns the underlying image.RGBA for direct access.
func (db *DirectBitmap) Image() *image.RGBA {
	return db.img
}

// SetPixel sets a pixel at (x, y) to the specified ARGB color.
func (db *DirectBitmap) SetPixel(x, y int, argbColor int) {
	if y >= db.height || x >= db.width {
		return
	}

	// Convert ARGB to RGBA
	a := uint8(argbColor >> 24)
	r := uint8(argbColor >> 16)
	g := uint8(argbColor >> 8)
	b := uint8(argbColor)

	// Force alpha to 255 as in C# version
	if a == 0 {
		a = 255
	}

	db.img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: a})
}

// SetPixelColor sets a pixel at (x, y) to the specified RgbColor.
func (db *DirectBitmap) SetPixelColor(x, y int, c RgbColor) {
	if y >= db.height || x >= db.width {
		return
	}

	db.img.Set(x, y, color.RGBA{R: c.R, G: c.V, B: c.B, A: 255})
}

// GetPixel returns the RGB color at (x, y) as a packed integer (no alpha).
func (db *DirectBitmap) GetPixel(x, y int) int {
	if x >= db.width || y >= db.height {
		return 0
	}

	c := color.RGBAModel.Convert(db.img.At(x, y)).(color.RGBA)
	return int(c.B) + (int(c.G) << 8) + (int(c.R) << 16)
}

// GetPixelColor returns the color at (x, y) as an RgbColor.
func (db *DirectBitmap) GetPixelColor(x, y int) RgbColor {
	if x >= db.width || y >= db.height {
		return RgbColor{}
	}

	c := color.RGBAModel.Convert(db.img.At(x, y)).(color.RGBA)
	return RgbColor{R: c.R, V: c.G, B: c.B}
}

// SetHorizontalLineDouble draws a horizontal line of pixels with double height.
// This replicates the C# SetHorizontalLineDouble method behavior.
func (db *DirectBitmap) SetHorizontalLineDouble(pixelX, pixelY, lineLength int, argbColor int) {
	// Convert ARGB to RGBA components
	a := uint8(argbColor >> 24)
	r := uint8(argbColor >> 16)
	g := uint8(argbColor >> 8)
	b := uint8(argbColor)

	// Force alpha to 255 as in C# version
	if a == 0 {
		a = 255
	}

	c := color.RGBA{R: r, G: g, B: b, A: a}

	for i := 0; i < lineLength; i++ {
		x := pixelX + i
		if x >= db.width {
			break
		}

		// Set current line
		if pixelY < db.height {
			db.img.Set(x, pixelY, c)
		}

		// Set line below (double height)
		if pixelY+1 < db.height {
			db.img.Set(x, pixelY+1, c)
		}
	}
}

// CopyBits copies pixel data from another DirectBitmap.
func (db *DirectBitmap) CopyBits(source *DirectBitmap) {
	if source == nil {
		return
	}

	// Copy the smaller of the two dimensions
	width := db.width
	if source.width < width {
		width = source.width
	}

	height := db.height
	if source.height < height {
		height = source.height
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := source.img.At(x, y)
			db.img.Set(x, y, c)
		}
	}
}

// NewDirectBitmapFromImage creates a DirectBitmap from any image.Image.
func NewDirectBitmapFromImage(img image.Image) *DirectBitmap {
	return NewFromImage(img)
}

// ToRGBA converts the DirectBitmap to an *image.RGBA.
func (db *DirectBitmap) ToRGBA() *image.RGBA {
	return db.img
}