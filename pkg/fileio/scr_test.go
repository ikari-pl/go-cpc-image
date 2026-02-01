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