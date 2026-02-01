// Package cpc provides CPC screen mode handling and pixel encoding.
package cpc

import "testing"

// TestTabOctetMode verifies the pixel bit patterns for CPC screen modes
func TestTabOctetMode(t *testing.T) {
	expected := [16]int{
		0x00, 0x80, 0x08, 0x88, 0x20, 0xA0, 0x28, 0xA8,
		0x02, 0x82, 0x0A, 0x8A, 0x22, 0xA2, 0x2A, 0xAA,
	}

	if len(TabOctetMode) != 16 {
		t.Fatalf("TabOctetMode length = %d, want 16", len(TabOctetMode))
	}

	for i := 0; i < 16; i++ {
		if TabOctetMode[i] != expected[i] {
			t.Errorf("TabOctetMode[%d] = 0x%02X, want 0x%02X", i, TabOctetMode[i], expected[i])
		}
	}
}

// TestPixelWidth tests horizontal pixel step calculation for each mode
func TestPixelWidth(t *testing.T) {
	tests := []struct {
		name        string
		modeVirtuel int
		y           int
		yEgx        int
		expected    int
	}{
		// Standard modes 0/1/2
		{"Mode 0 (4 pixels/byte)", 0, 0, 0, 4},
		{"Mode 1 (2 pixels/byte)", 1, 0, 0, 2},
		{"Mode 2 (1 pixel/byte)", 2, 0, 0, 1},

		// EGX modes 3/4 - alternating line modes
		{"EGX1 even line (2 pixels)", 3, 0, 0, 2},
		{"EGX1 odd line (4 pixels)", 3, 2, 0, 4},
		{"EGX2 even line (1 pixel)", 4, 0, 0, 1},
		{"EGX2 odd line (2 pixels)", 4, 2, 0, 2},

		// Mode X and Split modes
		{"Mode X", 5, 0, 0, 2},
		{"Mode Split", 6, 0, 0, 2},

		// ASCII modes
		{"Mode ASC-UT", 7, 0, 0, 2},
		{"Mode ASC0 (8 pixels/byte)", 8, 0, 0, 8},
		{"Mode ASC1 (4 pixels/byte)", 9, 0, 0, 4},
		{"Mode ASC2 (2 pixels/byte)", 10, 0, 0, 2},

		// Capture Sprites mode
		{"Capture Sprites", 11, 0, 0, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PixelWidth(tt.modeVirtuel, tt.y, tt.yEgx)
			if got != tt.expected {
				t.Errorf("PixelWidth(%d, %d, %d) = %d, want %d",
					tt.modeVirtuel, tt.y, tt.yEgx, got, tt.expected)
			}
		})
	}
}

// TestMaxPen tests maximum pen count for each mode
func TestMaxPen(t *testing.T) {
	tests := []struct {
		name        string
		modeVirtuel int
		y           int
		yEgx        int
		expected    int
	}{
		// Standard modes
		{"Mode 0 (16 colors)", 0, 0, 0, 16},
		{"Mode 1 (4 colors)", 1, 0, 0, 4},
		{"Mode 2 (2 colors)", 2, 0, 0, 2},

		// EGX modes - alternating between mode 1 and mode 2
		{"EGX1 even line (4 colors)", 3, 0, 0, 4},
		{"EGX1 odd line (16 colors)", 3, 2, 0, 16},
		{"EGX2 even line (2 colors)", 4, 0, 0, 2},
		{"EGX2 odd line (4 colors)", 4, 2, 0, 4},

		// Special modes
		{"Mode X (4 colors)", 5, 0, 0, 4},
		{"Mode Split (16 colors)", 6, 0, 0, 16},
		{"Mode ASC-UT (4 colors)", 7, 0, 0, 4},
		{"Mode ASC0 (16 colors)", 8, 0, 0, 16},
		{"Mode ASC1 (4 colors)", 9, 0, 0, 4},
		{"Mode ASC2 (2 colors)", 10, 0, 0, 2},

		// Default case
		{"Unknown mode defaults to 16", 99, 0, 0, 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaxPen(tt.modeVirtuel, tt.y, tt.yEgx)
			if got != tt.expected {
				t.Errorf("MaxPen(%d, %d, %d) = %d, want %d",
					tt.modeVirtuel, tt.y, tt.yEgx, got, tt.expected)
			}
		})
	}
}

// TestModesVirtuels verifies the mode names array
func TestModesVirtuels(t *testing.T) {
	expected := [12]string{
		"Mode 0",
		"Mode 1",
		"Mode 2",
		"Mode EGX1",
		"Mode EGX2",
		"Mode X",
		"Mode <Split>",
		"Mode ASC-UT",
		"Mode ASC0",
		"Mode ASC1",
		"Mode ASC2",
		"Capture Sprites",
	}

	if len(ModesVirtuels) != 12 {
		t.Fatalf("ModesVirtuels length = %d, want 12", len(ModesVirtuels))
	}

	for i := 0; i < 12; i++ {
		if ModesVirtuels[i] != expected[i] {
			t.Errorf("ModesVirtuels[%d] = %q, want %q", i, ModesVirtuels[i], expected[i])
		}
	}
}

// TestPixelWidthEGXPhasing tests EGX mode line phasing
func TestPixelWidthEGXPhasing(t *testing.T) {
	// Test that EGX modes alternate correctly based on y&2 == yEgx
	tests := []struct {
		name     string
		mode     int
		yValues  []int
		yEgx     int
		expected []int
	}{
		{
			name:     "EGX1 with yEgx=0",
			mode:     3,
			yValues:  []int{0, 2, 4, 6, 8},
			yEgx:     0,
			expected: []int{2, 4, 2, 4, 2}, // Alternating 2,4,2,4,2
		},
		{
			name:     "EGX1 with yEgx=2",
			mode:     3,
			yValues:  []int{0, 2, 4, 6, 8},
			yEgx:     2,
			expected: []int{4, 2, 4, 2, 4}, // Alternating 4,2,4,2,4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, y := range tt.yValues {
				got := PixelWidth(tt.mode, y, tt.yEgx)
				if got != tt.expected[i] {
					t.Errorf("PixelWidth(%d, %d, %d) = %d, want %d",
						tt.mode, y, tt.yEgx, got, tt.expected[i])
				}
			}
		})
	}
}

// TestMaxPenEGXPhasing tests EGX mode pen count phasing
func TestMaxPenEGXPhasing(t *testing.T) {
	// Test that EGX modes alternate correctly based on y&2 == yEgx
	tests := []struct {
		name     string
		mode     int
		yValues  []int
		yEgx     int
		expected []int
	}{
		{
			name:     "EGX1 pen count with yEgx=0",
			mode:     3,
			yValues:  []int{0, 2, 4, 6, 8},
			yEgx:     0,
			expected: []int{4, 16, 4, 16, 4}, // Alternating 4,16,4,16,4
		},
		{
			name:     "EGX2 pen count with yEgx=0",
			mode:     4,
			yValues:  []int{0, 2, 4, 6, 8},
			yEgx:     0,
			expected: []int{2, 4, 2, 4, 2}, // Alternating 2,4,2,4,2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, y := range tt.yValues {
				got := MaxPen(tt.mode, y, tt.yEgx)
				if got != tt.expected[i] {
					t.Errorf("MaxPen(%d, %d, %d) = %d, want %d",
						tt.mode, y, tt.yEgx, got, tt.expected[i])
				}
			}
		})
	}
}
