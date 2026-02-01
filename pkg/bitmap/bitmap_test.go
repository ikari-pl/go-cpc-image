// Package bitmap provides bitmap handling compatible with CPC image conversion.
package bitmap

import (
	"testing"
)

// TestNewDirectBitmap tests bitmap creation
func TestNewDirectBitmap(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"Small bitmap", 10, 10},
		{"Standard CPC screen", 640, 400},
		{"Overscan CPC screen", 768, 544},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bmp := NewDirectBitmap(tt.width, tt.height)
			if bmp == nil {
				t.Fatal("NewDirectBitmap returned nil")
			}
			if bmp.Width() != tt.width {
				t.Errorf("Width() = %d, want %d", bmp.Width(), tt.width)
			}
			if bmp.Height() != tt.height {
				t.Errorf("Height() = %d, want %d", bmp.Height(), tt.height)
			}
			if bmp.Length() != tt.width*tt.height {
				t.Errorf("Length() = %d, want %d", bmp.Length(), tt.width*tt.height)
			}
		})
	}
}

// TestSetPixelGetPixel tests round-trip pixel setting and reading
func TestSetPixelGetPixel(t *testing.T) {
	bmp := NewDirectBitmap(100, 100)

	tests := []struct {
		name  string
		x     int
		y     int
		color int
	}{
		{"Black", 10, 10, 0x000000},
		{"White", 20, 20, 0xFFFFFF},
		{"Red", 30, 30, 0xFF0000},
		{"Green", 40, 40, 0x00FF00},
		{"Blue", 50, 50, 0x0000FF},
		{"Yellow", 60, 60, 0xFFFF00},
		{"Cyan", 70, 70, 0x00FFFF},
		{"Magenta", 80, 80, 0xFF00FF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bmp.SetPixel(tt.x, tt.y, tt.color)
			got := bmp.GetPixel(tt.x, tt.y)
			// Mask to RGB only (no alpha)
			got &= 0xFFFFFF
			if got != tt.color {
				t.Errorf("GetPixel(%d, %d) = 0x%06X, want 0x%06X", tt.x, tt.y, got, tt.color)
			}
		})
	}
}

// TestSetPixelGetPixelWithAlpha tests ARGB pixel handling
func TestSetPixelGetPixelWithAlpha(t *testing.T) {
	bmp := NewDirectBitmap(10, 10)

	// SetPixel with alpha channel (should be forced to 255 if 0)
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"ARGB with alpha=0 (forced to 255)", 0x00FF0000, 0xFF0000},
		{"ARGB with alpha=128", 0x80FF0000, 0xFF0000},
		{"ARGB with alpha=255", 0xFFFF0000, 0xFF0000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bmp.SetPixel(5, 5, tt.input)
			got := bmp.GetPixel(5, 5) & 0xFFFFFF // Mask to RGB only
			if got != tt.expected {
				t.Errorf("Round-trip color = 0x%06X, want 0x%06X", got, tt.expected)
			}
		})
	}
}

// TestSetPixelColor tests setting pixels with RgbColor
func TestSetPixelColor(t *testing.T) {
	bmp := NewDirectBitmap(10, 10)

	tests := []struct {
		name  string
		color RgbColor
	}{
		{"Black", NewRgbColor(0x00, 0x00, 0x00)},
		{"White", NewRgbColor(0xFF, 0xFF, 0xFF)},
		{"Red", NewRgbColor(0xFF, 0x00, 0x00)},
		{"Green", NewRgbColor(0x00, 0xFF, 0x00)},
		{"Blue", NewRgbColor(0x00, 0x00, 0xFF)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bmp.SetPixelColor(5, 5, tt.color)
			got := bmp.GetPixelColor(5, 5)
			if got.R != tt.color.R || got.V != tt.color.V || got.B != tt.color.B {
				t.Errorf("GetPixelColor() = (%d,%d,%d), want (%d,%d,%d)",
					got.R, got.V, got.B, tt.color.R, tt.color.V, tt.color.B)
			}
		})
	}
}

// TestGetPixelOutOfBounds tests boundary checking
func TestGetPixelOutOfBounds(t *testing.T) {
	bmp := NewDirectBitmap(10, 10)

	tests := []struct {
		name string
		x    int
		y    int
	}{
		{"Negative x", -1, 5},
		{"Negative y", 5, -1},
		{"X too large", 10, 5},
		{"Y too large", 5, 10},
		{"Both too large", 20, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bmp.GetPixel(tt.x, tt.y)
			if got != 0 {
				t.Errorf("Out of bounds GetPixel(%d, %d) = 0x%06X, want 0", tt.x, tt.y, got)
			}

			gotColor := bmp.GetPixelColor(tt.x, tt.y)
			if gotColor.R != 0 || gotColor.V != 0 || gotColor.B != 0 {
				t.Errorf("Out of bounds GetPixelColor(%d, %d) = (%d,%d,%d), want (0,0,0)",
					tt.x, tt.y, gotColor.R, gotColor.V, gotColor.B)
			}
		})
	}
}

// TestSetPixelOutOfBounds tests that setting out of bounds doesn't crash
func TestSetPixelOutOfBounds(t *testing.T) {
	bmp := NewDirectBitmap(10, 10)

	// These should not crash
	bmp.SetPixel(-1, 5, 0xFFFFFF)
	bmp.SetPixel(5, -1, 0xFFFFFF)
	bmp.SetPixel(10, 5, 0xFFFFFF)
	bmp.SetPixel(5, 10, 0xFFFFFF)
	bmp.SetPixel(20, 20, 0xFFFFFF)

	bmp.SetPixelColor(-1, 5, NewRgbColor(0xFF, 0xFF, 0xFF))
	bmp.SetPixelColor(5, -1, NewRgbColor(0xFF, 0xFF, 0xFF))
	bmp.SetPixelColor(10, 5, NewRgbColor(0xFF, 0xFF, 0xFF))
	bmp.SetPixelColor(5, 10, NewRgbColor(0xFF, 0xFF, 0xFF))
	bmp.SetPixelColor(20, 20, NewRgbColor(0xFF, 0xFF, 0xFF))
}

// TestSetHorizontalLineDouble tests double-height horizontal line drawing
func TestSetHorizontalLineDouble(t *testing.T) {
	bmp := NewDirectBitmap(100, 100)

	// Draw a horizontal line at y=10
	bmp.SetHorizontalLineDouble(10, 10, 20, 0xFF0000) // Red line, 20 pixels wide

	// Verify the line is drawn on both y=10 and y=11
	for x := 10; x < 30; x++ {
		color10 := bmp.GetPixel(x, 10) & 0xFFFFFF
		color11 := bmp.GetPixel(x, 11) & 0xFFFFFF
		if color10 != 0xFF0000 {
			t.Errorf("Pixel at (%d, 10) = 0x%06X, want 0xFF0000", x, color10)
		}
		if color11 != 0xFF0000 {
			t.Errorf("Pixel at (%d, 11) = 0x%06X, want 0xFF0000", x, color11)
		}
	}

	// Verify pixels above and below are not set
	for x := 10; x < 30; x++ {
		color9 := bmp.GetPixel(x, 9) & 0xFFFFFF
		color12 := bmp.GetPixel(x, 12) & 0xFFFFFF
		if color9 != 0x000000 {
			t.Errorf("Pixel at (%d, 9) should be black, got 0x%06X", x, color9)
		}
		if color12 != 0x000000 {
			t.Errorf("Pixel at (%d, 12) should be black, got 0x%06X", x, color12)
		}
	}

	// Verify pixels before and after are not set
	color9_10 := bmp.GetPixel(9, 10) & 0xFFFFFF
	color30_10 := bmp.GetPixel(30, 10) & 0xFFFFFF
	if color9_10 != 0x000000 {
		t.Errorf("Pixel before line at (9, 10) should be black, got 0x%06X", color9_10)
	}
	if color30_10 != 0x000000 {
		t.Errorf("Pixel after line at (30, 10) should be black, got 0x%06X", color30_10)
	}
}

// TestSetHorizontalLineDoubleBoundary tests boundary conditions
func TestSetHorizontalLineDoubleBoundary(t *testing.T) {
	bmp := NewDirectBitmap(50, 50)

	// Draw line that extends beyond width
	bmp.SetHorizontalLineDouble(40, 10, 20, 0x00FF00)

	// Verify only pixels within bounds are set
	for x := 40; x < 50; x++ {
		color := bmp.GetPixel(x, 10) & 0xFFFFFF
		if color != 0x00FF00 {
			t.Errorf("Pixel at (%d, 10) = 0x%06X, want 0x00FF00", x, color)
		}
	}

	// Draw line at bottom edge
	bmp.SetHorizontalLineDouble(10, 49, 10, 0x0000FF)

	// y=49 should be set
	for x := 10; x < 20; x++ {
		color := bmp.GetPixel(x, 49) & 0xFFFFFF
		if color != 0x0000FF {
			t.Errorf("Pixel at (%d, 49) = 0x%06X, want 0x0000FF", x, color)
		}
	}

	// y=50 is out of bounds, should not crash
}

// TestCopyBits tests copying pixel data between bitmaps
func TestCopyBits(t *testing.T) {
	source := NewDirectBitmap(20, 20)
	dest := NewDirectBitmap(20, 20)

	// Fill source with a pattern
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			color := ((x * 16) << 16) | ((y * 16) << 8)
			source.SetPixel(x, y, color)
		}
	}

	// Copy to destination
	dest.CopyBits(source)

	// Verify all pixels match
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			srcColor := source.GetPixel(x, y) & 0xFFFFFF
			dstColor := dest.GetPixel(x, y) & 0xFFFFFF
			if srcColor != dstColor {
				t.Errorf("Pixel at (%d, %d): source=0x%06X, dest=0x%06X", x, y, srcColor, dstColor)
			}
		}
	}
}

// TestCopyBitsDifferentSizes tests copying between different sized bitmaps
func TestCopyBitsDifferentSizes(t *testing.T) {
	source := NewDirectBitmap(10, 10)
	dest := NewDirectBitmap(20, 20)

	// Fill source
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			source.SetPixel(x, y, 0xFF0000)
		}
	}

	// Copy to larger destination
	dest.CopyBits(source)

	// Verify copied area
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			color := dest.GetPixel(x, y) & 0xFFFFFF
			if color != 0xFF0000 {
				t.Errorf("Copied pixel at (%d, %d) = 0x%06X, want 0xFF0000", x, y, color)
			}
		}
	}

	// Verify area outside copied region is still black
	color := dest.GetPixel(15, 15) & 0xFFFFFF
	if color != 0x000000 {
		t.Errorf("Pixel outside copied area at (15, 15) = 0x%06X, want 0x000000", color)
	}
}

// TestCopyBitsNilSource tests that copying from nil doesn't crash
func TestCopyBitsNilSource(t *testing.T) {
	dest := NewDirectBitmap(10, 10)
	dest.CopyBits(nil) // Should not crash
}

// TestImage tests access to underlying image.RGBA
func TestImage(t *testing.T) {
	bmp := NewDirectBitmap(10, 10)
	img := bmp.Image()

	if img == nil {
		t.Error("Image() returned nil")
	}

	bounds := img.Bounds()
	if bounds.Dx() != 10 || bounds.Dy() != 10 {
		t.Errorf("Image bounds = %dx%d, want 10x10", bounds.Dx(), bounds.Dy())
	}
}
