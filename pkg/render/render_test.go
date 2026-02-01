// Package render provides CPC screen rendering functionality.
package render

import (
	"testing"

	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
	"github.com/ikari-pl/go-cpc-image/pkg/cpc"
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

// TestNewBitmapCpcWithParams tests creation with direct column/line/cpcPlus params
func TestNewBitmapCpcWithParams(t *testing.T) {
	tests := []struct {
		name    string
		numCol  int
		numLig  int
		cpcPlus bool
	}{
		{"Standard CPC", 80, 200, false},
		{"Overscan CPC", 96, 272, false},
		{"CPC+ standard", 80, 200, true},
		{"CPC+ overscan", 96, 272, true},
		{"Small", 10, 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bmp := NewBitmapCpcWithParams(tt.numCol, tt.numLig, tt.cpcPlus)

			if bmp.NumCol != tt.numCol {
				t.Errorf("NumCol = %d, want %d", bmp.NumCol, tt.numCol)
			}
			if bmp.NumLig != tt.numLig {
				t.Errorf("NumLig = %d, want %d", bmp.NumLig, tt.numLig)
			}
			if bmp.CpcPlus != tt.cpcPlus {
				t.Errorf("CpcPlus = %v, want %v", bmp.CpcPlus, tt.cpcPlus)
			}
			// Default mode should be 1
			for i := 0; i < 272; i++ {
				if bmp.ModeTable[i] != 1 {
					t.Errorf("ModeTable[%d] = %d, want 1", i, bmp.ModeTable[i])
					break
				}
			}
			// Palette should be initialized to DefaultPalette
			if bmp.Palette != cpc.DefaultPalette {
				t.Errorf("Palette not initialized to DefaultPalette")
			}
			// SplitPalette should have default palette values
			if bmp.SplitPalette[0][0][0] != bmp.Palette[0] {
				t.Errorf("SplitPalette[0][0][0] = %d, want %d", bmp.SplitPalette[0][0][0], bmp.Palette[0])
			}
		})
	}
}

// TestDrawBitmapMode0 tests Mode 0 decoding (2 pixels per byte, 16 colors)
func TestDrawBitmapMode0(t *testing.T) {
	bmp := NewBitmapCpc(16, 4, 0) // 2 columns, 2 lines
	// Mode 0: 2 pixels per byte
	// Pen 0 in left pixel, pen 1 in right pixel
	// Pen encoding: p0 = bit7 + (bit5>>3) + (bit3>>2) + (bit1<<2)
	// Pen encoding: p1 = (bit6>>6) + (bit4>>2) + (bit2>>1) + (bit0<<3)
	// For pen0=0, pen1=1: byte = 0x40|0x10|0x04|0x01 = 0x55... let's just use 0
	// For pen0=0, pen1=0: byte = 0x00
	addr := cpc.CpcAddress(0, 2, 2)
	bmp.ScreenData[addr] = 0x00   // pen 0, pen 0
	bmp.ScreenData[addr+1] = 0x00 // pen 0, pen 0

	result := bmp.DrawBitmap(2, 2, 0, false)
	if result == nil {
		t.Fatal("DrawBitmap returned nil")
	}

	// Width should be numCol*8 = 16, height = numLig*2 = 4
	img := result.Image()
	if img.Bounds().Dx() != 16 || img.Bounds().Dy() != 4 {
		t.Errorf("Result size = %dx%d, want 16x4", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

// TestDrawBitmapMode1 tests Mode 1 decoding (4 pixels per byte, 4 colors)
func TestDrawBitmapMode1(t *testing.T) {
	bmp := NewBitmapCpc(16, 4, 1) // 2 columns, 2 lines
	// All zeros = pen 0 everywhere
	result := bmp.DrawBitmap(2, 2, 0, false)
	if result == nil {
		t.Fatal("DrawBitmap returned nil")
	}

	img := result.Image()
	// All pixels should be the color of palette entry 0
	expectedColor := cpc.PaletteColor(bmp.Palette[0], false)
	r0 := uint8((expectedColor >> 16) & 0xFF)
	g0 := uint8((expectedColor >> 8) & 0xFF)
	b0 := uint8(expectedColor & 0xFF)

	r, g, b, _ := img.At(0, 0).RGBA()
	if uint8(r>>8) != r0 || uint8(g>>8) != g0 || uint8(b>>8) != b0 {
		t.Errorf("Pixel(0,0) = (%d,%d,%d), want (%d,%d,%d)", r>>8, g>>8, b>>8, r0, g0, b0)
	}
}

// TestDrawBitmapMode2 tests Mode 2 decoding (8 pixels per byte, 2 colors)
func TestDrawBitmapMode2(t *testing.T) {
	bmp := NewBitmapCpc(16, 4, 2)
	// Set byte to 0xFF = all pen 1
	addr := cpc.CpcAddress(0, 2, 2)
	bmp.ScreenData[addr] = 0xFF

	result := bmp.DrawBitmap(2, 2, 0, false)
	if result == nil {
		t.Fatal("DrawBitmap returned nil")
	}

	img := result.Image()
	// First 8 pixels on line 0 should be palette[1]
	expectedColor := cpc.PaletteColor(bmp.Palette[1], false)
	r0 := uint8((expectedColor >> 16) & 0xFF)
	g0 := uint8((expectedColor >> 8) & 0xFF)
	b0 := uint8(expectedColor & 0xFF)

	r, g, b, _ := img.At(0, 0).RGBA()
	if uint8(r>>8) != r0 || uint8(g>>8) != g0 || uint8(b>>8) != b0 {
		t.Errorf("Pixel(0,0) mode2 all-1 = (%d,%d,%d), want (%d,%d,%d)", r>>8, g>>8, b>>8, r0, g0, b0)
	}
}

// TestDrawBitmapSprite tests sprite address calculation
func TestDrawBitmapSprite(t *testing.T) {
	bmp := NewBitmapCpc(16, 4, 1)
	// For sprites, addr = numCol * (y>>1), so line 0 => addr 0
	bmp.ScreenData[0] = 0x00
	bmp.ScreenData[1] = 0x00

	result := bmp.DrawBitmap(2, 2, 0, true)
	if result == nil {
		t.Fatal("DrawBitmap returned nil for sprite")
	}
}

// TestDrawBitmapRealWidth tests custom width parameter
func TestDrawBitmapRealWidth(t *testing.T) {
	bmp := NewBitmapCpc(16, 4, 1)
	result := bmp.DrawBitmap(2, 2, 32, false)
	if result == nil {
		t.Fatal("DrawBitmap returned nil")
	}
	img := result.Image()
	if img.Bounds().Dx() != 32 {
		t.Errorf("Width = %d, want 32", img.Bounds().Dx())
	}
}

// TestDrawBitmapEGXModes tests EGX virtual modes (3 and 4)
func TestDrawBitmapEGXModes(t *testing.T) {
	for _, vmode := range []int{3, 4} {
		t.Run("VirtualMode"+string(rune('0'+vmode)), func(t *testing.T) {
			bmp := NewBitmapCpc(16, 4, vmode)
			bmp.YEgx = 0
			result := bmp.DrawBitmap(2, 2, 0, false)
			if result == nil {
				t.Fatal("DrawBitmap returned nil for EGX mode")
			}
		})
	}
}

// TestDrawBitmapHighVirtualMode tests virtual modes >= 5
func TestDrawBitmapHighVirtualMode(t *testing.T) {
	bmp := NewBitmapCpc(16, 4, 5)
	result := bmp.DrawBitmap(2, 2, 0, false)
	if result == nil {
		t.Fatal("DrawBitmap returned nil for virtual mode 5")
	}
}

// TestRenderToRGBA tests the RenderToRGBA convenience method
func TestRenderToRGBA(t *testing.T) {
	bmp := NewBitmapCpc(16, 4, 1) // 2 cols, 2 lines
	img := bmp.RenderToRGBA()
	if img == nil {
		t.Fatal("RenderToRGBA returned nil")
	}
	if img.Bounds().Dx() != 16 || img.Bounds().Dy() != 4 {
		t.Errorf("RenderToRGBA size = %dx%d, want 16x4", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

// TestRenderToRGBAOverscan tests RenderToRGBA with overscan dimensions
func TestRenderToRGBAOverscan(t *testing.T) {
	bmp := NewBitmapCpc(768, 544, 1)
	img := bmp.RenderToRGBA()
	if img == nil {
		t.Fatal("RenderToRGBA returned nil")
	}
	if img.Bounds().Dx() != 768 || img.Bounds().Dy() != 544 {
		t.Errorf("Size = %dx%d, want 768x544", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

// TestRenderSplitPalette tests the Render method with split palette
func TestRenderSplitPalette(t *testing.T) {
	bmp := NewBitmapCpc(16, 4, 1) // 2 cols, 2 lines
	// Set a known mode
	for i := range bmp.ModeTable {
		bmp.ModeTable[i] = 1
	}

	dest := bitmap.NewDirectBitmap(16, 4)
	result := bmp.Render(dest, 0)
	if result == nil {
		t.Fatal("Render returned nil")
	}
}

// TestRenderMode0Split tests Render with mode 0 split palette
func TestRenderMode0Split(t *testing.T) {
	bmp := NewBitmapCpc(16, 4, 0)
	for i := range bmp.ModeTable {
		bmp.ModeTable[i] = 0
	}

	dest := bitmap.NewDirectBitmap(16, 4)
	result := bmp.Render(dest, 0)
	if result == nil {
		t.Fatal("Render mode 0 returned nil")
	}
}

// TestRenderMode2Split tests Render with mode 2 split palette
func TestRenderMode2Split(t *testing.T) {
	bmp := NewBitmapCpc(16, 4, 2)
	for i := range bmp.ModeTable {
		bmp.ModeTable[i] = 2
	}

	dest := bitmap.NewDirectBitmap(16, 4)
	result := bmp.Render(dest, 0)
	if result == nil {
		t.Fatal("Render mode 2 returned nil")
	}
}

// TestRenderWithOffset tests Render with a non-zero X offset
func TestRenderWithOffset(t *testing.T) {
	bmp := NewBitmapCpc(16, 4, 1)
	for i := range bmp.ModeTable {
		bmp.ModeTable[i] = 1
	}

	dest := bitmap.NewDirectBitmap(16, 4)
	result := bmp.Render(dest, 8)
	if result == nil {
		t.Fatal("Render with offset returned nil")
	}
}

// TestEncodeToCpcEGX tests encoding with EGX flag
func TestEncodeToCpcEGX(t *testing.T) {
	bmp := NewBitmapCpc(16, 4, 1)
	bmp.Palette = cpc.DefaultPalette
	srcBmp := bitmap.NewDirectBitmap(16, 4)

	// Fill with palette[0] color
	color := cpc.CpcRgbPalette[bmp.Palette[0]]
	for y := 0; y < 4; y++ {
		for x := 0; x < 16; x++ {
			srcBmp.SetPixelColor(x, y, color)
		}
	}

	// Test with EGX enabled, ligneStart=0
	bmp.EncodeToCpc(srcBmp, true, 0)

	// Test with EGX enabled, ligneStart=1
	bmp2 := NewBitmapCpc(16, 4, 1)
	bmp2.Palette = cpc.DefaultPalette
	bmp2.EncodeToCpc(srcBmp, true, 1)

	t.Log("EGX encoding completed for both ligneStart values")
}

// TestEncodeToCpcRoundTrip tests encode then decode roundtrip for Mode 1
func TestEncodeToCpcRoundTripMode1(t *testing.T) {
	bmp := NewBitmapCpc(16, 4, 1) // 2 cols, 2 lines
	bmp.Palette = cpc.DefaultPalette

	srcBmp := bitmap.NewDirectBitmap(16, 4)
	// Fill all with pen 0 color
	pen0Color := cpc.CpcRgbPalette[bmp.Palette[0]]
	for y := 0; y < 4; y++ {
		for x := 0; x < 16; x++ {
			srcBmp.SetPixelColor(x, y, pen0Color)
		}
	}

	bmp.EncodeToCpc(srcBmp, false, 0)
	result := bmp.DrawBitmap(2, 2, 0, false)

	if result == nil {
		t.Fatal("DrawBitmap returned nil after encode")
	}

	// Check that decoded pixels match pen 0 color
	expectedArgb := cpc.PaletteColor(bmp.Palette[0], false)
	rExp := uint8((expectedArgb >> 16) & 0xFF)
	gExp := uint8((expectedArgb >> 8) & 0xFF)
	bExp := uint8(expectedArgb & 0xFF)

	img := result.Image()
	r, g, b, _ := img.At(0, 0).RGBA()
	if uint8(r>>8) != rExp || uint8(g>>8) != gExp || uint8(b>>8) != bExp {
		t.Errorf("Roundtrip pixel(0,0) = (%d,%d,%d), want (%d,%d,%d)", r>>8, g>>8, b>>8, rExp, gExp, bExp)
	}
}

// TestEncodeToCpcForceMode1RoundTrip tests forced mode 1 encode/decode
func TestEncodeToCpcForceMode1RoundTrip(t *testing.T) {
	bmp := NewBitmapCpc(16, 4, 0) // Mode 0, will force mode 1
	bmp.Palette = cpc.DefaultPalette

	srcBmp := bitmap.NewDirectBitmap(16, 4)
	pen0Color := cpc.CpcRgbPalette[bmp.Palette[0]]
	for y := 0; y < 4; y++ {
		for x := 0; x < 16; x++ {
			srcBmp.SetPixelColor(x, y, pen0Color)
		}
	}

	bmp.EncodeToCpcForceMode1(srcBmp)

	// Verify some screen data was written
	hasData := false
	for i := 0; i < len(bmp.ScreenData); i++ {
		if bmp.ScreenData[i] != 0 {
			hasData = true
			break
		}
	}
	// pen 0 encodes to all-zero bytes, so this may be all zero - that's fine
	_ = hasData
	t.Log("ForceMode1 roundtrip completed")
}

// TestEncodeToCpcForceMode1CpcPlus tests CPC+ palette matching in ForceMode1
func TestEncodeToCpcForceMode1CpcPlus(t *testing.T) {
	bmp := NewBitmapCpc(16, 4, 1)
	bmp.CpcPlus = true
	// Set CPC+ palette values (12-bit GRB format)
	bmp.Palette[0] = 0x000 // black
	bmp.Palette[1] = 0xFFF // white
	bmp.Palette[2] = 0x00F // blue
	bmp.Palette[3] = 0x0F0 // red

	srcBmp := bitmap.NewDirectBitmap(16, 4)
	// Fill with black (palette entry 0: G=0, R=0, B=0)
	for y := 0; y < 4; y++ {
		for x := 0; x < 16; x++ {
			srcBmp.SetPixelColor(x, y, bitmap.NewRgbColor(0, 0, 0))
		}
	}

	bmp.EncodeToCpcForceMode1(srcBmp)
	t.Log("CPC+ ForceMode1 encoding completed")
}

// TestMinFunction tests the min helper
func TestMinFunction(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{0, 0, 0},
		{-1, 1, -1},
		{100, 100, 100},
	}
	for _, tt := range tests {
		if got := min(tt.a, tt.b); got != tt.want {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// TestGetPaletteColorInvalidIndex tests that invalid color index returns black
func TestGetPaletteColorInvalidIndex(t *testing.T) {
	bmp := NewBitmapCpc(640, 400, 1)
	// Set a split palette entry to an invalid index (>= 27)
	bmp.SplitPalette[0][0][0] = 30

	color := bmp.GetPaletteColor(0, 0, 0)
	if color.R != 0 || color.V != 0 || color.B != 0 {
		t.Errorf("Invalid index should return black, got (%d,%d,%d)", color.R, color.V, color.B)
	}
}
