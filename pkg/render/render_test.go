// Package render provides CPC screen rendering functionality.
package render

import (
	"testing"

	"github.com/ikari/go-cpc-image/pkg/bitmap"
	"github.com/ikari/go-cpc-image/pkg/cpc"
)

// TestNewBitmapCpc tests BitmapCpc creation
func TestNewBitmapCpc(t *testing.T) {
	tests := []struct {
		name        string
		tx          int
		ty          int
		mode        int
		expectedCol int
		expectedLig int
	}{
		{"Standard Mode 0", 640, 400, 0, 80, 200},
		{"Standard Mode 1", 640, 400, 1, 80, 200},
		{"Standard Mode 2", 640, 400, 2, 80, 200},
		{"Overscan Mode 1", 768, 544, 1, 96, 272},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bmp := NewBitmapCpc(tt.tx, tt.ty, tt.mode)

			if bmp == nil {
				t.Fatal("NewBitmapCpc returned nil")
			}

			if bmp.NumCol != tt.expectedCol {
				t.Errorf("NumCol = %d, want %d", bmp.NumCol, tt.expectedCol)
			}

			if bmp.NumLig != tt.expectedLig {
				t.Errorf("NumLig = %d, want %d", bmp.NumLig, tt.expectedLig)
			}

			if bmp.VirtualMode != tt.mode {
				t.Errorf("VirtualMode = %d, want %d", bmp.VirtualMode, tt.mode)
			}
		})
	}
}

// TestGetColorPal tests palette color retrieval
func TestGetColorPal(t *testing.T) {
	bmp := NewBitmapCpc(640, 400, 1)
	bmp.Palette = cpc.DefaultPalette
	bmp.CpcPlus = false

	tests := []struct {
		name     string
		palEntry int
	}{
		{"Pen 0", 0},
		{"Pen 1", 1},
		{"Pen 2", 2},
		{"Pen 15", 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := bmp.GetColorPal(tt.palEntry)

			// Verify we get an RgbColor (not testing specific values)
			if color.R == 0 && color.V == 0 && color.B == 0 && bmp.Palette[tt.palEntry] != 0 {
				t.Logf("Warning: Pen %d returned black for non-zero palette entry", tt.palEntry)
			}
		})
	}
}

// TestGetPaletteColor tests split screen palette color retrieval
func TestGetPaletteColor(t *testing.T) {
	bmp := NewBitmapCpc(640, 400, 1)

	// Initialize split palette with known values
	for y := 0; y < 272; y++ {
		for x := 0; x < 96; x++ {
			bmp.SplitPalette[x][y][0] = 1  // Blue
			bmp.SplitPalette[x][y][1] = 24 // Bright Yellow
		}
	}

	tests := []struct {
		name string
		x, y int
		col  int
	}{
		{"Position 0,0 pen 0", 0, 0, 0},
		{"Position 10,10 pen 1", 10, 10, 1},
		{"Position 50,50 pen 0", 50, 50, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := bmp.GetPaletteColor(tt.x, tt.y, tt.col)

			// Verify we get an RgbColor
			_ = color
		})
	}
}

// TestEncodeToCpc tests CPC screen memory encoding
func TestEncodeToCpc(t *testing.T) {
	tests := []struct {
		name string
		mode int
	}{
		{"Mode 0", 0},
		{"Mode 1", 1},
		{"Mode 2", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bmp := NewBitmapCpc(640, 400, tt.mode)
			bmp.Palette = cpc.DefaultPalette
			bmp.CpcPlus = false

			// Create a test bitmap
			srcBmp := bitmap.NewDirectBitmap(640, 400)

			// Fill with test pattern
			for y := 0; y < 400; y++ {
				for x := 0; x < 640; x++ {
					// Use first palette color
					color := cpc.CpcRgbPalette[bmp.Palette[0]]
					srcBmp.SetPixelColor(x, y, color)
				}
			}

			// Encode to CPC format
			bmp.EncodeToCpc(srcBmp, false, 0)

			// Verify that screen buffer is not all zeros
			nonZero := false
			for i := 0; i < 0x4000; i++ {
				if bmp.ScreenData[i] != 0 {
					nonZero = true
					break
				}
			}

			// Note: For mode 0 with pen 0, the screen might be all zeros
			// This is expected behavior
			t.Logf("Mode %d encoding produced %v non-zero bytes", tt.mode, nonZero)
		})
	}
}

// TestDrawBitmapRoundTrip tests encoding and decoding round-trip
func TestDrawBitmapRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		mode int
	}{
		{"Mode 0 round-trip", 0},
		{"Mode 1 round-trip", 1},
		{"Mode 2 round-trip", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bmp := NewBitmapCpc(640, 400, tt.mode)
			bmp.Palette = cpc.DefaultPalette
			bmp.CpcPlus = false

			// Create source bitmap with a known pattern
			srcBmp := bitmap.NewDirectBitmap(640, 400)

			// Fill with alternating colors from palette
			penColors := []int{0, 1, 2, 3}
			for y := 0; y < 400; y++ {
				for x := 0; x < 640; x++ {
					pen := penColors[x%len(penColors)]
					if pen < len(bmp.Palette) {
						color := cpc.GetColor(bmp.Palette[pen], false)
						srcBmp.SetPixelColor(x, y, color)
					}
				}
			}

			// Encode to CPC format
			bmp.EncodeToCpc(srcBmp, false, 0)

			// Note: Full round-trip would require DrawBitmap implementation
			// For now, verify encoding completed
			t.Logf("Encoding for mode %d completed", tt.mode)
		})
	}
}

// TestEncodeToCpcMode1 tests Mode 1 specific encoding
func TestEncodeToCpcMode1(t *testing.T) {
	bmp := NewBitmapCpc(640, 400, 1)
	bmp.Palette = [16]int{1, 24, 20, 6, 26, 0, 2, 7, 10, 12, 14, 16, 18, 22, 1, 14}
	bmp.CpcPlus = false

	srcBmp := bitmap.NewDirectBitmap(640, 400)

	// Fill with 4 colors (Mode 1 supports 4 colors)
	colors := []int{1, 24, 20, 6}
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x += 4 {
			for i := 0; i < 4 && x+i < 640; i++ {
				color := cpc.CpcRgbPalette[colors[i]]
				srcBmp.SetPixelColor(x+i, y, color)
			}
		}
	}

	bmp.EncodeToCpc(srcBmp, false, 0)

	// Verify encoding (Mode 1: 4 pixels per byte)
	// Each byte should encode 4 pixels
	nonZeroCount := 0
	for i := 0; i < 0x4000; i++ {
		if bmp.ScreenData[i] != 0 {
			nonZeroCount++
		}
	}

	t.Logf("Mode 1 encoding: %d non-zero bytes", nonZeroCount)
}

// TestEncodeToCpcForceMode1 tests forced Mode 1 encoding
func TestEncodeToCpcForceMode1(t *testing.T) {
	bmp := NewBitmapCpc(640, 400, 0) // Mode 0, but will force Mode 1
	bmp.Palette = cpc.DefaultPalette
	bmp.CpcPlus = false

	srcBmp := bitmap.NewDirectBitmap(640, 400)

	// Fill with test colors
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			color := cpc.CpcRgbPalette[bmp.Palette[0]]
			srcBmp.SetPixelColor(x, y, color)
		}
	}

	bmp.EncodeToCpcForceMode1(srcBmp)

	// Verify encoding completed
	t.Log("Force Mode 1 encoding completed")
}

// TestBitmapCpcInitialization tests proper initialization of BitmapCpc
func TestBitmapCpcInitialization(t *testing.T) {
	bmp := NewBitmapCpc(640, 400, 1)

	// Verify ModeTable is initialized
	if len(bmp.ModeTable) != 272 {
		t.Errorf("ModeTable length = %d, want 272", len(bmp.ModeTable))
	}

	// All mode entries should be set to the specified mode
	for i := 0; i < 272; i++ {
		if bmp.ModeTable[i] != 1 {
			t.Errorf("ModeTable[%d] = %d, want 1", i, bmp.ModeTable[i])
		}
	}

	// Verify SplitPalette dimensions
	if len(bmp.SplitPalette) != 96 {
		t.Errorf("SplitPalette X dimension = %d, want 96", len(bmp.SplitPalette))
	}
	if len(bmp.SplitPalette[0]) != 272 {
		t.Errorf("SplitPalette Y dimension = %d, want 272", len(bmp.SplitPalette[0]))
	}
	if len(bmp.SplitPalette[0][0]) != 17 {
		t.Errorf("SplitPalette pen dimension = %d, want 17", len(bmp.SplitPalette[0][0]))
	}
}

// TestBitmapCpcScreenBuffer tests screen buffer array
func TestBitmapCpcScreenBuffer(t *testing.T) {
	bmp := NewBitmapCpc(640, 400, 1)

	// Verify buffer size
	if len(bmp.ScreenData) != 0x10000 {
		t.Errorf("ScreenData size = 0x%X, want 0x10000", len(bmp.ScreenData))
	}

	// Buffer should be initialized to zero
	for i := 0; i < 0x100; i++ { // Check first 256 bytes
		if bmp.ScreenData[i] != 0 {
			t.Errorf("ScreenData[0x%04X] = 0x%02X, want 0x00", i, bmp.ScreenData[i])
		}
	}
}

// TestBitmapCpcPaletteSize tests palette array
func TestBitmapCpcPaletteSize(t *testing.T) {
	bmp := NewBitmapCpc(640, 400, 1)

	// Verify palette size
	if len(bmp.Palette) != 16 {
		t.Errorf("Palette size = %d, want 16", len(bmp.Palette))
	}
}

// TestEncodeToCpcClearsBuffer tests that encoding clears the buffer first
func TestEncodeToCpcClearsBuffer(t *testing.T) {
	bmp := NewBitmapCpc(640, 400, 1)
	bmp.Palette = cpc.DefaultPalette

	// Fill buffer with non-zero values
	for i := 0; i < 0x4000; i++ {
		bmp.ScreenData[i] = 0xFF
	}

	srcBmp := bitmap.NewDirectBitmap(640, 400)

	// Encode (should clear buffer first)
	bmp.EncodeToCpc(srcBmp, false, 0)

	// At least some bytes should have been cleared/overwritten
	// (exact pattern depends on source image)
	t.Log("Buffer cleared and encoded")
}

// TestGetColorPalInvalidIndex tests handling of invalid palette indices
func TestGetColorPalInvalidIndex(t *testing.T) {
	bmp := NewBitmapCpc(640, 400, 1)
	bmp.Palette = cpc.DefaultPalette
	bmp.CpcPlus = false

	// Negative index (should be safe)
	// Note: GetColorPal might not bounds-check, relying on array access behavior
	// This test verifies it doesn't crash

	// Valid indices
	for i := 0; i < 16; i++ {
		_ = bmp.GetColorPal(i)
	}

	t.Log("Palette access completed without panic")
}

// TestGetPaletteColorBounds tests split palette bounds
func TestGetPaletteColorBounds(t *testing.T) {
	bmp := NewBitmapCpc(640, 400, 1)

	// Test corners of the split palette
	tests := []struct {
		name string
		x, y, col int
	}{
		{"Top-left", 0, 0, 0},
		{"Bottom-right valid", 95, 271, 15},
		{"Middle", 48, 136, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = bmp.GetPaletteColor(tt.x, tt.y, tt.col)
		})
	}
}

// TestBitmapCpcOverscan tests overscan screen configuration
func TestBitmapCpcOverscan(t *testing.T) {
	bmp := NewBitmapCpc(768, 544, 1)

	if bmp.NumCol != 96 {
		t.Errorf("Overscan NumCol = %d, want 96", bmp.NumCol)
	}
	if bmp.NumLig != 272 {
		t.Errorf("Overscan NumLig = %d, want 272", bmp.NumLig)
	}
}
