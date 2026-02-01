// Package cpc provides CPC memory layout and screen addressing.
package cpc

import "testing"

// TestCpcAddress tests CPC screen memory address calculation with known values
func TestCpcAddress(t *testing.T) {
	// Test standard screen addressing (80 cols x 200 lines at #C000)
	// Formula: adr = (y/16)*numCol + (y%16)*#400
	tests := []struct {
		name   string
		y      int
		numCol int
		numLig int
		expect int
	}{
		// Standard screen (80 cols x 200 lines)
		{"Standard line 0", 0, 80, 200, 0x0000},
		{"Standard line 2", 2, 80, 200, 0x0800},
		{"Standard line 16", 16, 80, 200, 0x0050},
		{"Standard line 18", 18, 80, 200, 0x0850},
		{"Standard line 32", 32, 80, 200, 0x00A0},
		{"Standard line 64", 64, 80, 200, 0x0140},
		{"Standard line 128", 128, 80, 200, 0x0280},
		{"Standard line 160", 160, 80, 200, 0x0320},

		// Overscan screen (96 cols x 272 lines at #0200)
		{"Overscan line 0", 0, 96, 272, 0x0000},
		{"Overscan line 2", 2, 96, 272, 0x0800},
		{"Overscan line 16", 16, 96, 272, 0x0060},
		{"Overscan line 256", 256, 96, 272, 0x3E00},
		{"Overscan line 258", 258, 96, 272, 0x4600}, // y>255: adds 0x3800
		{"Overscan line 270", 270, 96, 272, 0x7600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CpcAddress(tt.y, tt.numCol, tt.numLig)
			if got != tt.expect {
				t.Errorf("CpcAddress(%d, %d, %d) = 0x%04X, want 0x%04X",
					tt.y, tt.numCol, tt.numLig, got, tt.expect)
			}
		})
	}
}

// TestCpcAddressInterleaving tests the interleaved addressing pattern
func TestCpcAddressInterleaving(t *testing.T) {
	// CPC screen memory is interleaved: lines 0,8,16,24... are consecutive,
	// then lines 2,10,18,26..., then 4,12,20,28..., etc.
	// This tests that the interleaving formula is correct.

	numCol := 80
	numLig := 200

	// Test that consecutive even lines are 0x800 apart (within same group)
	for y := 0; y < 14; y += 2 {
		addr0 := CpcAddress(y, numCol, numLig)
		addr1 := CpcAddress(y+2, numCol, numLig)
		diff := addr1 - addr0
		if diff != 0x800 {
			t.Errorf("Address difference between line %d and %d = 0x%04X, want 0x800", y, y+2, diff)
		}
	}

	// Test that line 16 follows line 0 with +numCol offset
	addr0 := CpcAddress(0, numCol, numLig)
	addr16 := CpcAddress(16, numCol, numLig)
	diff := addr16 - addr0
	if diff != numCol {
		t.Errorf("Address difference between line 0 and 16 = 0x%04X, want 0x%04X", diff, numCol)
	}
}

// TestCpcAddressOverscanOffset tests the 0x3800 offset for y>255 in overscan
func TestCpcAddressOverscanOffset(t *testing.T) {
	numCol := 96
	numLig := 272

	// Lines 254 and 256 should have a larger gap due to the 0x3800 offset
	addr254 := CpcAddress(254, numCol, numLig)
	addr256 := CpcAddress(256, numCol, numLig)

	// Without the offset, the difference would be (256-254)/16*numCol + ((256&14)-(254&14))*0x400
	// With offset: addr256 should have +0x3800
	expectedDiff := (256>>4)*numCol + ((256&14)*0x400) + 0x3800 - addr254

	diff := addr256 - addr254
	if diff != expectedDiff {
		t.Logf("addr254=0x%04X, addr256=0x%04X", addr254, addr256)
		t.Errorf("Address jump from line 254 to 256 = 0x%04X, want 0x%04X", diff, expectedDiff)
	}

	// Verify that the offset is exactly 0x3800 more than it would be without the offset
	addrWithoutOffset := (256>>4)*numCol + ((256&14) * 0x400)
	addrWithOffset := addr256
	offsetApplied := addrWithOffset - (addrWithoutOffset - addr254 + addr254)
	if offsetApplied != 0x3800 {
		t.Errorf("Overscan offset applied = 0x%04X, want 0x3800", offsetApplied)
	}
}

// TestBitmapSize tests the total size calculation for CPC screen buffer
func TestBitmapSize(t *testing.T) {
	tests := []struct {
		name   string
		numCol int
		numLig int
		expect int
	}{
		{"Standard screen", 80, 200, 16336},   // Actual size = 0x3FD0
		{"Overscan screen", 96, 272, 31936},   // Actual size = 0x7CC0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BitmapSize(tt.numCol, tt.numLig)
			if got != tt.expect {
				t.Errorf("BitmapSize(%d, %d) = %d (0x%04X), want %d (0x%04X)",
					tt.numCol, tt.numLig, got, got, tt.expect, tt.expect)
			}
		})
	}
}

// TestNewStandardScreen tests standard screen configuration
func TestNewStandardScreen(t *testing.T) {
	cfg := NewStandardScreen()

	if cfg.NumCol != StandardCols {
		t.Errorf("NumCol = %d, want %d", cfg.NumCol, StandardCols)
	}
	if cfg.NumLig != StandardLines {
		t.Errorf("NumLig = %d, want %d", cfg.NumLig, StandardLines)
	}
	if cfg.YEgx != 0 {
		t.Errorf("YEgx = %d, want 0", cfg.YEgx)
	}
}

// TestNewOverscanScreen tests overscan screen configuration
func TestNewOverscanScreen(t *testing.T) {
	cfg := NewOverscanScreen()

	if cfg.NumCol != OverscanCols {
		t.Errorf("NumCol = %d, want %d", cfg.NumCol, OverscanCols)
	}
	if cfg.NumLig != OverscanLines {
		t.Errorf("NumLig = %d, want %d", cfg.NumLig, OverscanLines)
	}
	if cfg.YEgx != 0 {
		t.Errorf("YEgx = %d, want 0", cfg.YEgx)
	}
}

// TestScreenConfigTailleX tests screen width calculation
func TestScreenConfigTailleX(t *testing.T) {
	tests := []struct {
		name   string
		numCol int
		expect int
	}{
		{"Standard 80 cols = 640 pixels", 80, 640},
		{"Overscan 96 cols = 768 pixels", 96, 768},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ScreenConfig{NumCol: tt.numCol}
			got := cfg.TailleX()
			if got != tt.expect {
				t.Errorf("TailleX() = %d, want %d", got, tt.expect)
			}
		})
	}
}

// TestScreenConfigTailleY tests screen height calculation
func TestScreenConfigTailleY(t *testing.T) {
	tests := []struct {
		name   string
		numLig int
		expect int
	}{
		{"Standard 200 lines = 400 pixels", 200, 400},
		{"Overscan 272 lines = 544 pixels", 272, 544},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ScreenConfig{NumLig: tt.numLig}
			got := cfg.TailleY()
			if got != tt.expect {
				t.Errorf("TailleY() = %d, want %d", got, tt.expect)
			}
		})
	}
}

// TestScreenConfigSetTailleX tests setting screen width from pixels
func TestScreenConfigSetTailleX(t *testing.T) {
	cfg := ScreenConfig{}
	cfg.SetTailleX(640)
	if cfg.NumCol != 80 {
		t.Errorf("After SetTailleX(640), NumCol = %d, want 80", cfg.NumCol)
	}

	cfg.SetTailleX(768)
	if cfg.NumCol != 96 {
		t.Errorf("After SetTailleX(768), NumCol = %d, want 96", cfg.NumCol)
	}
}

// TestScreenConfigSetTailleY tests setting screen height from pixels
func TestScreenConfigSetTailleY(t *testing.T) {
	cfg := ScreenConfig{}
	cfg.SetTailleY(400)
	if cfg.NumLig != 200 {
		t.Errorf("After SetTailleY(400), NumLig = %d, want 200", cfg.NumLig)
	}

	cfg.SetTailleY(544)
	if cfg.NumLig != 272 {
		t.Errorf("After SetTailleY(544), NumLig = %d, want 272", cfg.NumLig)
	}
}

// TestScreenConfigGetAdr tests address calculation via ScreenConfig
func TestScreenConfigGetAdr(t *testing.T) {
	cfg := NewStandardScreen()
	addr := cfg.GetAdr(16)
	expected := CpcAddress(16, StandardCols, StandardLines)
	if addr != expected {
		t.Errorf("GetAdr(16) = 0x%04X, want 0x%04X", addr, expected)
	}
}

// TestScreenConfigGetBitmapSize tests bitmap size calculation via ScreenConfig
func TestScreenConfigGetBitmapSize(t *testing.T) {
	tests := []struct {
		name   string
		cfg    ScreenConfig
		minSize int
	}{
		{"Standard screen", NewStandardScreen(), 16336},
		{"Overscan screen", NewOverscanScreen(), 31936},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.GetBitmapSize()
			if got != tt.minSize {
				t.Errorf("GetBitmapSize() = %d, want %d", got, tt.minSize)
			}
		})
	}
}
