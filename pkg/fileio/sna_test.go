package fileio

import (
	"os"
	"path/filepath"
	"testing"
)

func createTestSNA(t *testing.T, mode byte, numCol byte, numLig byte) string {
	t.Helper()

	// Build a minimal valid SNA file: 256-byte header + 64KB memory
	data := make([]byte, 256+65536)

	// Signature
	copy(data[0:8], "MV - SNA")

	// CRTC register 1 (NumCol)
	data[snaCRTCReg1] = numCol

	// CRTC register 6 (NumLig)
	data[snaCRTCReg6] = numLig

	// GA config / mode
	data[snaGAConfig] = mode

	// Palette: use hardware values from CpcVGA for firmware indices 0-15
	for i := 0; i < 16 && i < len(CpcVGA); i++ {
		data[snaPalette+i] = CpcVGA[i]
	}

	// Write some recognizable pattern to screen memory at offset 0xC000 in the dump
	screenOffset := 256 + 0xC000
	for i := 0; i < 16384; i++ {
		data[screenOffset+i] = byte(i & 0xFF)
	}

	path := filepath.Join(t.TempDir(), "test.sna")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadSNA_Basic(t *testing.T) {
	path := createTestSNA(t, 1, 40, 25)

	bmp, err := LoadSNA(path)
	if err != nil {
		t.Fatalf("LoadSNA failed: %v", err)
	}

	if bmp.VirtualMode != 1 {
		t.Errorf("expected mode 1, got %d", bmp.VirtualMode)
	}
	if bmp.NumCol != 40 {
		t.Errorf("expected NumCol=40, got %d", bmp.NumCol)
	}
	if bmp.NumLig != 25 {
		t.Errorf("expected NumLig=25, got %d", bmp.NumLig)
	}

	// Verify palette mapping: firmware index i should map back to i
	for i := 0; i < 16; i++ {
		if bmp.Palette[i] != i {
			t.Errorf("palette[%d]: expected %d, got %d", i, i, bmp.Palette[i])
		}
	}

	// Verify screen data was copied
	if bmp.ScreenData[0] != 0x00 || bmp.ScreenData[1] != 0x01 || bmp.ScreenData[255] != 0xFF {
		t.Error("screen data not copied correctly")
	}
}

func TestLoadSNA_InvalidSignature(t *testing.T) {
	data := make([]byte, 256+65536)
	copy(data[0:8], "INVALID!")

	path := filepath.Join(t.TempDir(), "bad.sna")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadSNA(path)
	if err == nil {
		t.Error("expected error for invalid signature")
	}
}

func TestLoadSNA_FileTooSmall(t *testing.T) {
	data := make([]byte, 100)
	path := filepath.Join(t.TempDir(), "small.sna")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadSNA(path)
	if err == nil {
		t.Error("expected error for too-small file")
	}
}

func TestLoadSNA_Mode0(t *testing.T) {
	path := createTestSNA(t, 0, 20, 25)

	bmp, err := LoadSNA(path)
	if err != nil {
		t.Fatalf("LoadSNA failed: %v", err)
	}

	if bmp.VirtualMode != 0 {
		t.Errorf("expected mode 0, got %d", bmp.VirtualMode)
	}
}

func TestBuildHardwareToFirmware(t *testing.T) {
	m := buildHardwareToFirmware()
	if len(m) != 27 {
		t.Errorf("expected 27 entries, got %d", len(m))
	}
	// Round-trip: firmware -> hardware -> firmware
	for i := 0; i < 27; i++ {
		hw := CpcVGA[i]
		if m[hw] != i {
			t.Errorf("round-trip failed for firmware index %d: hw=0x%02X -> fw=%d", i, hw, m[hw])
		}
	}
}
