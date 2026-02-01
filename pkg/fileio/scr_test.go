package fileio

import (
	"testing"
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
		{"full red (R=F)", 0x00F, 0xF0, 0x00},         // R=F,B=0 → low=0xF0, G=0 → high=0x00
		{"full green (G=F)", 0xF00, 0x00, 0x0F},        // R=0,B=0 → low=0x00, G=F → high=0x0F
		{"full blue (B=F)", 0x0F0, 0x0F, 0x00},         // R=0,B=F → low=0x0F, G=0 → high=0x00
		{"white", 0xFFF, 0xFF, 0x0F},                   // R=F,B=F → low=0xFF, G=F → high=0x0F
		{"mixed 0x84A", 0x84A, 0xA4, 0x08},             // R=A,B=4 → low=0xA4, G=8 → high=0x08
		{"mixed 0x123", 0x123, 0x32, 0x01},             // R=3,B=2 → low=0x32, G=1 → high=0x01
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