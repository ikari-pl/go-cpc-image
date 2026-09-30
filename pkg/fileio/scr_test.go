package fileio

import (
	"os"
	"testing"

	"github.com/ikari-pl/go-cpc-image/pkg/cpc"
)

func TestPoke16(t *testing.T) {
	data := make([]byte, 4)
	Poke16(data, 0, 0x1234)

	if data[0] != 0x34 || data[1] != 0x12 {
		t.Errorf("Poke16 failed: expected [0x34, 0x12], got [0x%02X, 0x%02X]", data[0], data[1])
	}
}

func TestZ80CodeArrays(t *testing.T) {
	// Test that code arrays are not empty
	if len(CodeStd) == 0 {
		t.Error("CodeStd should not be empty")
	}
	if len(CodeP0) == 0 {
		t.Error("CodeP0 should not be empty")
	}
	if len(codeDepack) == 0 {
		t.Error("codeDepack should not be empty")
	}
	if len(codeDZX0) == 0 {
		t.Error("codeDZX0 should not be empty")
	}
}

func TestCpcVGALookup(t *testing.T) {
	// Test that CpcVGA string has expected length
	if len(CpcVGA) != 27 {
		t.Errorf("CpcVGA should have 27 characters, got %d", len(CpcVGA))
	}
}

// TestCpcPlusPaletteEncoding verifies the SCR palette encoding converts
// internal 0x0GBR format to ASIC 0x0GRB little-endian byte order.
func TestCpcPlusPaletteEncoding(t *testing.T) {
	// Simulate what SaveSCR does for CPC+ palette encoding
	tests := []struct {
		name     string
		palette  uint16 // Internal format: 0x0GBR
		wantLow  byte   // First byte: 0xRB (ASIC low byte)
		wantHigh byte   // Second byte: 0x0G (ASIC high byte)
	}{
		{"black", 0x000, 0x00, 0x00},
		{"full red (R=F)", 0x00F, 0xF0, 0x00},   // R=F,B=0 → low=0xF0, G=0 → high=0x00
		{"full green (G=F)", 0xF00, 0x00, 0x0F}, // R=0,B=0 → low=0x00, G=F → high=0x0F
		{"full blue (B=F)", 0x0F0, 0x0F, 0x00},  // R=0,B=F → low=0x0F, G=0 → high=0x00
		{"white", 0xFFF, 0xFF, 0x0F},            // R=F,B=F → low=0xFF, G=F → high=0x0F
		{"mixed 0x84A", 0x84A, 0xA4, 0x08},      // R=A,B=4 → low=0xA4, G=8 → high=0x08
		{"mixed 0x123", 0x123, 0x32, 0x01},      // R=3,B=2 → low=0x32, G=1 → high=0x01
	}

	for _, tt := range tests {
		// Replicate the encoding from scr.go lines 585-588
		low := byte(((tt.palette >> 4) & 0x0F) | (tt.palette << 4))
		high := byte(tt.palette >> 8)

		if low != tt.wantLow || high != tt.wantHigh {
			t.Errorf("%s (0x%03X): got [0x%02X,0x%02X], want [0x%02X,0x%02X]",
				tt.name, tt.palette, low, high, tt.wantLow, tt.wantHigh)
		}

		// Verify roundtrip: ASIC bytes (0x0GRB little-endian) back to internal 0x0GBR
		// ASIC 16-bit value = high<<8 | low = 0x0GRB
		asicVal := uint16(high)<<8 | uint16(low)
		// Extract from 0x0GRB: G=bits 11-8, R=bits 7-4, B=bits 3-0
		asicG := (asicVal >> 8) & 0x0F
		asicR := (asicVal >> 4) & 0x0F
		asicB := asicVal & 0x0F
		// Convert back to internal 0x0GBR
		roundtrip := uint16(asicG<<8 | asicB<<4 | asicR)
		if roundtrip != tt.palette {
			t.Errorf("%s: roundtrip 0x%03X != original 0x%03X (ASIC=0x%04X)",
				tt.name, roundtrip, tt.palette, asicVal)
		}
	}
}

// TestSaveSCROverscanThreshold (A4) verifies that a standard-mode screen
// (16336 bytes = BitmapSize(80, 200)) is NOT misclassified as overscan.
// With the old threshold (> 0x3F00 = 16128) a 16336-byte standard screen
// was classified as overscan, yielding a 0x0200 load address and palette at
// offset 0x600 instead of 0x17D0. The correct threshold (> 0x4000 = 16384)
// classifies 16336 as standard.
func TestSaveSCROverscanThreshold(t *testing.T) {
	tempDir := t.TempDir()
	scrPath := tempDir + "/test.scr"

	bitmapSize := cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines) // 16336
	raw := make([]byte, bitmapSize)
	// Put a distinctive pattern
	for i := range raw {
		raw[i] = byte(i % 256)
	}

	params := SCRParams{
		WithPalette: true,
		WithCode:    true,
		CPCPlus:     false,
		VirtualMode: 1,
	}
	pal := []uint16{1, 24, 20, 6, 26, 0, 2, 7, 10, 12, 14, 16, 18, 22, 1, 14}

	if _, err := SaveSCR(scrPath, raw, bitmapSize,
		PackNone, OutputBinary, params, pal, nil); err != nil {
		t.Fatalf("SaveSCR failed: %v", err)
	}

	data, err := os.ReadFile(scrPath)
	if err != nil {
		t.Fatalf("read SCR failed: %v", err)
	}

	// Must have AMSDOS header
	if len(data) < 128 || !cpc.CheckAmsdos(data) {
		t.Fatal("SCR missing valid AMSDOS header")
	}

	// Standard mode screen must have load address 0xC000, NOT 0x0200 (overscan).
	if ent, e := cpc.GetAmsdos(data); e == nil {
		if ent.Address != 0xC000 {
			t.Errorf("standard SCR load address = &%04X, want &C000 (overscan misclassification)", ent.Address)
		}
	} else {
		t.Fatalf("GetAmsdos failed: %v", e)
	}

	// Standard screen writes its palette at offset 0x17D0, not 0x600.
	if len(data) >= 128+0x17D0 {
		// modePal[0] should be the mode byte (1), not zero
		if data[128+0x17D0] != byte(params.VirtualMode) {
			t.Errorf("palette mode byte at 0x17D0 = %d, want %d", data[128+0x17D0], params.VirtualMode)
		}
	}
}
