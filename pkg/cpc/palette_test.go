// Package cpc provides CPC hardware constants and functions.
package cpc

import (
	"testing"

	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
)

// TestCpcRgbPalette validates the 27-color CPC palette values
func TestCpcRgbPalette(t *testing.T) {
	tests := []struct {
		name     string
		index    int
		expected bitmap.RgbColor
	}{
		{"Black", 0, bitmap.NewRgbColor(0x00, 0x00, 0x00)},
		{"Blue", 1, bitmap.NewRgbColor(0x00, 0x00, 0x66)},
		{"Bright Blue", 2, bitmap.NewRgbColor(0x00, 0x00, 0xFF)},
		{"Red", 3, bitmap.NewRgbColor(0x66, 0x00, 0x00)},
		{"Magenta", 4, bitmap.NewRgbColor(0x66, 0x00, 0x66)},
		{"White", 13, bitmap.NewRgbColor(0x66, 0x66, 0x66)},
		{"Bright Yellow", 24, bitmap.NewRgbColor(0xFF, 0xFF, 0x00)},
		{"Pastel Yellow", 25, bitmap.NewRgbColor(0xFF, 0xFF, 0x66)},
		{"Bright White", 26, bitmap.NewRgbColor(0xFF, 0xFF, 0xFF)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CpcRgbPalette[tt.index]
			if got.R != tt.expected.R || got.V != tt.expected.V || got.B != tt.expected.B {
				t.Errorf("CpcRgbPalette[%d] = (%d,%d,%d), want (%d,%d,%d)",
					tt.index, got.R, got.V, got.B, tt.expected.R, tt.expected.V, tt.expected.B)
			}
		})
	}
}

// TestPaletteColor tests standard CPC color palette retrieval
func TestPaletteColor(t *testing.T) {
	tests := []struct {
		name     string
		index    int
		cpcPlus  bool
		expected int
	}{
		{"Black standard", 0, false, 0x000000},
		{"Blue standard", 1, false, 0x000066},
		{"Bright White standard", 26, false, 0xFFFFFF},
		{"Out of range negative", -1, false, 0x000000}, // Should default to 0
		{"Out of range high", 30, false, 0x000000},     // Should default to 0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PaletteColor(tt.index, tt.cpcPlus)
			if got != tt.expected {
				t.Errorf("PaletteColor(%d, %v) = 0x%06X, want 0x%06X",
					tt.index, tt.cpcPlus, got, tt.expected)
			}
		})
	}
}

// TestPaletteColorPlus tests CPC+ 12-bit color conversion (VBR to RGB)
func TestPaletteColorPlus(t *testing.T) {
	tests := []struct {
		name     string
		grbValue int
		expected int
	}{
		{
			name:     "Black CPC+",
			grbValue: 0x000, // G=0, R=0, B=0
			expected: 0x000000,
		},
		{
			name:     "White CPC+",
			grbValue: 0xFFF, // G=15, R=15, B=15
			expected: 0xFFFFFF,
		},
		{
			name:     "Red CPC+",
			grbValue: 0x00F, // V=0, B=0, R=15
			expected: 0xFF0000,
		},
		{
			name:     "Green CPC+",
			grbValue: 0xF00, // V=15, B=0, R=0
			expected: 0x00FF00,
		},
		{
			name:     "Blue CPC+",
			grbValue: 0x0F0, // V=0, B=15, R=0
			expected: 0x0000FF,
		},
		{
			name:     "Mid gray CPC+",
			grbValue: 0x888, // G=8, R=8, B=8
			expected: 0x888888,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PaletteColor(tt.grbValue, true)
			if got != tt.expected {
				t.Errorf("PaletteColor(0x%03X, true) = 0x%06X, want 0x%06X",
					tt.grbValue, got, tt.expected)
			}
		})
	}
}

// TestGetColor_StandardAndCpcPlus verifies that GetColor returns the
// correct RGB triplet for both standard CPC palette indices (27 hardware
// colours) and CPC+ 12-bit palette entries (4096 possible colours).
func TestGetColor_StandardAndCpcPlus(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		cpcPlus  bool
		expected bitmap.RgbColor
	}{
		{
			name:     "Standard CPC black",
			value:    0,
			cpcPlus:  false,
			expected: bitmap.NewRgbColor(0x00, 0x00, 0x00),
		},
		{
			name:     "Standard CPC white",
			value:    13,
			cpcPlus:  false,
			expected: bitmap.NewRgbColor(0x66, 0x66, 0x66),
		},
		{
			name:     "CPC+ red (VBR=0x00F)",
			value:    0x00F,
			cpcPlus:  true,
			expected: bitmap.NewRgbColor(0xFF, 0x00, 0x00),
		},
		{
			name:     "CPC+ green (VBR=0xF00)",
			value:    0xF00,
			cpcPlus:  true,
			expected: bitmap.NewRgbColor(0x00, 0xFF, 0x00),
		},
		{
			name:     "CPC+ blue (VBR=0x0F0)",
			value:    0x0F0,
			cpcPlus:  true,
			expected: bitmap.NewRgbColor(0x00, 0x00, 0xFF),
		},
		{
			name:     "Invalid value (>= 0xFFFF)",
			value:    0xFFFF,
			cpcPlus:  false,
			expected: bitmap.RgbColor{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetColor(tt.value, tt.cpcPlus)
			if got.R != tt.expected.R || got.V != tt.expected.V || got.B != tt.expected.B {
				t.Errorf("GetColor(%d, %v) = (%d,%d,%d), want (%d,%d,%d)",
					tt.value, tt.cpcPlus, got.R, got.V, got.B,
					tt.expected.R, tt.expected.V, tt.expected.B)
			}
		})
	}
}

// TestGetPenColor tests pen color lookup in bitmap
func TestGetPenColor(t *testing.T) {
	// Create a test bitmap with known colors
	bmp := bitmap.NewDirectBitmap(10, 10)

	tests := []struct {
		name        string
		setupPixel  bitmap.RgbColor
		palette     [16]int
		cpcPlus     bool
		expectedPen int
	}{
		{
			name:       "Standard CPC exact match pen 0",
			setupPixel: CpcRgbPalette[1], // Blue
			palette: [16]int{1, 24, 20, 6, 26, 0, 2, 7, 10, 12, 14, 16, 18, 22, 1, 14},
			cpcPlus:    false,
			expectedPen: 0,
		},
		{
			name:       "Standard CPC exact match pen 1",
			setupPixel: CpcRgbPalette[24], // Bright Yellow
			palette: [16]int{1, 24, 20, 6, 26, 0, 2, 7, 10, 12, 14, 16, 18, 22, 1, 14},
			cpcPlus:    false,
			expectedPen: 1,
		},
		{
			name:       "No match defaults to pen 0",
			setupPixel: bitmap.NewRgbColor(0x33, 0x33, 0x33),
			palette: [16]int{1, 24, 20, 6, 26, 0, 2, 7, 10, 12, 14, 16, 18, 22, 1, 14},
			cpcPlus:    false,
			expectedPen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bmp.SetPixelColor(5, 5, tt.setupPixel)
			got := GetPenColor(bmp, 5, 5, tt.palette, tt.cpcPlus)
			if got != tt.expectedPen {
				t.Errorf("GetPenColor() = %d, want %d", got, tt.expectedPen)
			}
		})
	}
}

// TestGetPenColorCPCPlus tests CPC+ 4-bit precision matching
func TestGetPenColorCPCPlus(t *testing.T) {
	bmp := bitmap.NewDirectBitmap(10, 10)

	// Set up a CPC+ palette with known VBR values
	palette := [16]int{
		0x000, // Black
		0xF00, // Green
		0x00F, // Red
		0x0F0, // Blue
		0xFFF, // White
		0xFFFF, 0xFFFF, 0xFFFF, 0xFFFF, 0xFFFF, 0xFFFF, 0xFFFF, 0xFFFF, 0xFFFF, 0xFFFF, 0xFFFF,
	}

	tests := []struct {
		name        string
		setupPixel  bitmap.RgbColor
		expectedPen int
	}{
		{
			name:        "CPC+ Red match (4-bit precision)",
			setupPixel:  bitmap.NewRgbColor(0xFF, 0x00, 0x00),
			expectedPen: 2,
		},
		{
			name:        "CPC+ Green match (4-bit precision)",
			setupPixel:  bitmap.NewRgbColor(0x00, 0xFF, 0x00),
			expectedPen: 1,
		},
		{
			name:        "CPC+ Blue match (4-bit precision)",
			setupPixel:  bitmap.NewRgbColor(0x00, 0x00, 0xFF),
			expectedPen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bmp.SetPixelColor(5, 5, tt.setupPixel)
			got := GetPenColor(bmp, 5, 5, palette, true)
			if got != tt.expectedPen {
				t.Errorf("GetPenColor() = %d, want %d", got, tt.expectedPen)
			}
		})
	}
}

// TestDefaultPalette verifies the default palette values
func TestDefaultPalette(t *testing.T) {
	expected := [16]int{1, 24, 20, 6, 26, 0, 2, 7, 10, 12, 14, 16, 18, 22, 1, 14}

	if len(DefaultPalette) != 16 {
		t.Fatalf("DefaultPalette length = %d, want 16", len(DefaultPalette))
	}

	for i := 0; i < 16; i++ {
		if DefaultPalette[i] != expected[i] {
			t.Errorf("DefaultPalette[%d] = %d, want %d", i, DefaultPalette[i], expected[i])
		}
	}
}
