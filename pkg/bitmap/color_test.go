// Package bitmap provides color and bitmap abstractions for CPC image handling.
package bitmap

import "testing"

// TestNewRgbColor tests RgbColor creation
func TestNewRgbColor(t *testing.T) {
	tests := []struct {
		name string
		r, v, b uint8
	}{
		{"Black", 0x00, 0x00, 0x00},
		{"White", 0xFF, 0xFF, 0xFF},
		{"Red", 0xFF, 0x00, 0x00},
		{"Green", 0x00, 0xFF, 0x00},
		{"Blue", 0x00, 0x00, 0xFF},
		{"Mid gray", 0x80, 0x80, 0x80},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewRgbColor(tt.r, tt.v, tt.b)
			if c.R != tt.r {
				t.Errorf("R = %d, want %d", c.R, tt.r)
			}
			if c.V != tt.v {
				t.Errorf("V = %d, want %d", c.V, tt.v)
			}
			if c.B != tt.b {
				t.Errorf("B = %d, want %d", c.B, tt.b)
			}
		})
	}
}

// TestNewRgbColorFromInt tests creating RgbColor from packed ARGB integer
func TestNewRgbColorFromInt(t *testing.T) {
	tests := []struct {
		name  string
		value int
		r, v, b uint8
	}{
		{"Black", 0x00000000, 0x00, 0x00, 0x00},
		{"White", 0xFFFFFFFF, 0xFF, 0xFF, 0xFF},
		{"Red", 0xFFFF0000, 0xFF, 0x00, 0x00},
		{"Green", 0xFF00FF00, 0x00, 0xFF, 0x00},
		{"Blue", 0xFF0000FF, 0x00, 0x00, 0xFF},
		{"Yellow", 0xFFFFFF00, 0xFF, 0xFF, 0x00},
		{"Cyan", 0xFF00FFFF, 0x00, 0xFF, 0xFF},
		{"Magenta", 0xFFFF00FF, 0xFF, 0x00, 0xFF},
		{"Alpha ignored", 0x80FF00FF, 0xFF, 0x00, 0xFF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewRgbColorFromInt(tt.value)
			if c.R != tt.r {
				t.Errorf("R = %d, want %d", c.R, tt.r)
			}
			if c.V != tt.v {
				t.Errorf("V = %d, want %d", c.V, tt.v)
			}
			if c.B != tt.b {
				t.Errorf("B = %d, want %d", c.B, tt.b)
			}
		})
	}
}

// TestGetColor_PacksRGBWithoutAlpha verifies that GetColor packs the red,
// green and blue channels into a single uint32 while discarding alpha —
// the inverse of SetColorArgb's full ARGB packing.
func TestGetColor_PacksRGBWithoutAlpha(t *testing.T) {
	tests := []struct {
		name     string
		r, v, b  uint8
		expected int
	}{
		{"Black", 0x00, 0x00, 0x00, 0x000000},
		{"White", 0xFF, 0xFF, 0xFF, 0xFFFFFF},
		{"Red", 0xFF, 0x00, 0x00, 0xFF0000},
		{"Green", 0x00, 0xFF, 0x00, 0x00FF00},
		{"Blue", 0x00, 0x00, 0xFF, 0x0000FF},
		{"Yellow", 0xFF, 0xFF, 0x00, 0xFFFF00},
		{"Cyan", 0x00, 0xFF, 0xFF, 0x00FFFF},
		{"Magenta", 0xFF, 0x00, 0xFF, 0xFF00FF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewRgbColor(tt.r, tt.v, tt.b)
			got := c.GetColor()
			if got != tt.expected {
				t.Errorf("GetColor() = 0x%06X, want 0x%06X", got, tt.expected)
			}
		})
	}
}

// TestGetColorArgb tests ARGB packing (alpha=255)
func TestGetColorArgb(t *testing.T) {
	tests := []struct {
		name     string
		r, v, b  uint8
		expected int
	}{
		{"Black with alpha", 0x00, 0x00, 0x00, 0xFF000000},
		{"White with alpha", 0xFF, 0xFF, 0xFF, 0xFFFFFFFF},
		{"Red with alpha", 0xFF, 0x00, 0x00, 0xFFFF0000},
		{"Green with alpha", 0x00, 0xFF, 0x00, 0xFF00FF00},
		{"Blue with alpha", 0x00, 0x00, 0xFF, 0xFF0000FF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewRgbColor(tt.r, tt.v, tt.b)
			got := c.GetColorArgb()
			if got != tt.expected {
				t.Errorf("GetColorArgb() = 0x%08X, want 0x%08X", got, tt.expected)
			}
		})
	}
}

// TestSetColorArgb tests setting color from ARGB integer
func TestSetColorArgb(t *testing.T) {
	tests := []struct {
		name  string
		value int
		r, v, b uint8
	}{
		{"Set black", 0xFF000000, 0x00, 0x00, 0x00},
		{"Set white", 0xFFFFFFFF, 0xFF, 0xFF, 0xFF},
		{"Set red", 0xFFFF0000, 0xFF, 0x00, 0x00},
		{"Set green", 0xFF00FF00, 0x00, 0xFF, 0x00},
		{"Set blue", 0xFF0000FF, 0x00, 0x00, 0xFF},
		{"Alpha ignored when setting", 0x80FF0000, 0xFF, 0x00, 0x00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c RgbColor
			c.SetColorArgb(tt.value)
			if c.R != tt.r {
				t.Errorf("R = %d, want %d", c.R, tt.r)
			}
			if c.V != tt.v {
				t.Errorf("V = %d, want %d", c.V, tt.v)
			}
			if c.B != tt.b {
				t.Errorf("B = %d, want %d", c.B, tt.b)
			}
		})
	}
}

// TestRgbColorRoundTrip tests round-trip conversion
func TestRgbColorRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		r, v, b uint8
	}{
		{"Black", 0x00, 0x00, 0x00},
		{"White", 0xFF, 0xFF, 0xFF},
		{"Red", 0xFF, 0x00, 0x00},
		{"Green", 0x00, 0xFF, 0x00},
		{"Blue", 0x00, 0x00, 0xFF},
		{"Random color", 0x12, 0x34, 0x56},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create color
			c1 := NewRgbColor(tt.r, tt.v, tt.b)

			// Convert to int and back
			argb := c1.GetColorArgb()
			c2 := NewRgbColorFromInt(argb)

			if c2.R != c1.R || c2.V != c1.V || c2.B != c1.B {
				t.Errorf("Round-trip failed: original (%d,%d,%d), got (%d,%d,%d)",
					c1.R, c1.V, c1.B, c2.R, c2.V, c2.B)
			}
		})
	}
}

// TestRgbColorRoundTripWithSet tests round-trip using SetColorArgb
func TestRgbColorRoundTripWithSet(t *testing.T) {
	tests := []struct {
		name string
		r, v, b uint8
	}{
		{"Black", 0x00, 0x00, 0x00},
		{"White", 0xFF, 0xFF, 0xFF},
		{"Red", 0xFF, 0x00, 0x00},
		{"Green", 0x00, 0xFF, 0x00},
		{"Blue", 0x00, 0x00, 0xFF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create color
			c1 := NewRgbColor(tt.r, tt.v, tt.b)

			// Convert to int
			argb := c1.GetColorArgb()

			// Set into new color
			var c2 RgbColor
			c2.SetColorArgb(argb)

			if c2.R != c1.R || c2.V != c1.V || c2.B != c1.B {
				t.Errorf("Round-trip failed: original (%d,%d,%d), got (%d,%d,%d)",
					c1.R, c1.V, c1.B, c2.R, c2.V, c2.B)
			}
		})
	}
}

// TestRgbColorPacking tests bit packing order
func TestRgbColorPacking(t *testing.T) {
	// Test that packing order is correct: B + (V << 8) + (R << 16)
	c := NewRgbColor(0x12, 0x34, 0x56)

	packed := c.GetColor()
	expectedPacked := 0x123456
	if packed != expectedPacked {
		t.Errorf("GetColor() = 0x%06X, want 0x%06X", packed, expectedPacked)
	}

	packedArgb := c.GetColorArgb()
	expectedArgb := 0xFF123456
	if packedArgb != expectedArgb {
		t.Errorf("GetColorArgb() = 0x%08X, want 0x%08X", packedArgb, expectedArgb)
	}
}

// TestRgbColorUnpacking tests bit unpacking order
func TestRgbColorUnpacking(t *testing.T) {
	// Test that unpacking order is correct
	c := NewRgbColorFromInt(0xFF123456)

	if c.R != 0x12 {
		t.Errorf("R = 0x%02X, want 0x12", c.R)
	}
	if c.V != 0x34 {
		t.Errorf("V = 0x%02X, want 0x34", c.V)
	}
	if c.B != 0x56 {
		t.Errorf("B = 0x%02X, want 0x56", c.B)
	}
}

// TestRgbColorZeroValue tests zero-initialized RgbColor
func TestRgbColorZeroValue(t *testing.T) {
	var c RgbColor

	if c.R != 0 || c.V != 0 || c.B != 0 {
		t.Errorf("Zero value RgbColor = (%d,%d,%d), want (0,0,0)", c.R, c.V, c.B)
	}

	packed := c.GetColor()
	if packed != 0 {
		t.Errorf("Zero value GetColor() = 0x%06X, want 0x000000", packed)
	}

	packedArgb := c.GetColorArgb()
	if packedArgb != 0xFF000000 {
		t.Errorf("Zero value GetColorArgb() = 0x%08X, want 0xFF000000", packedArgb)
	}
}
