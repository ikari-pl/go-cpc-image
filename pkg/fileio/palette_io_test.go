// Package fileio provides CPC file format I/O operations.
package fileio

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/ikari-pl/go-cpc-image/pkg/cpc"
)

// TestLoadSavePaletteKitRoundTrip verifies that the KIT writer is the exact
// inverse of the reader across the whole 12-bit colour space. SavePaletteKit used
// to rearrange the wrong nibbles, so a palette read back from a .kit could never
// be written out again — 0x042F came out as 0x1300 — silently corrupting every
// saved CPC Plus palette.
func TestLoadSavePaletteKitRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "roundtrip.kit")

	for base := 0; base <= 0xFF0; base += 16 {
		var want, got [16]uint16
		for i := 0; i < 16; i++ {
			want[i] = uint16((base + i) & 0xFFF)
		}
		if err := SavePaletteKit(path, want[:]); err != nil {
			t.Fatalf("SavePaletteKit(base %#03X) failed: %v", base, err)
		}
		if err := LoadPaletteKit(path, got[:]); err != nil {
			t.Fatalf("LoadPaletteKit(base %#03X) failed: %v", base, err)
		}
		if got != want {
			t.Fatalf("KIT round trip at base %#03X: got %v, want %v", base, got, want)
		}
	}
}

// TestSavePaletteKitWordOrder pins the on-disk word layout the loader reads and
// the reference screens carry: bits 8-11 keep the green level, and the two
// nibbles of the low byte are swapped so blue sits in bits 4-7 and red in bits
// 0-3. 0x04F2 (V=4, B=F, R=2) must therefore be stored as 0x042F.
func TestSavePaletteKitWordOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "order.kit")
	pal := make([]uint16, 16)
	pal[0] = 0x04F2
	if err := SavePaletteKit(path, pal); err != nil {
		t.Fatalf("SavePaletteKit failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s failed: %v", path, err)
	}
	if len(data) != 160 {
		t.Fatalf("kit size = %d, want 160", len(data))
	}
	if !cpc.CheckAmsdos(data) {
		t.Fatal("kit has no valid AMSDOS header")
	}
	if got := binary.LittleEndian.Uint16(data[128:]); got != 0x042F {
		t.Errorf("first KIT word = 0x%04X, want 0x042F", got)
	}

	// ...and the loader must turn it back into the 12-bit colour.
	var back [16]uint16
	if err := LoadPaletteKit(path, back[:]); err != nil {
		t.Fatalf("LoadPaletteKit failed: %v", err)
	}
	if back[0] != 0x04F2 {
		t.Errorf("reloaded colour = 0x%03X, want 0x04F2", back[0])
	}
}

// TestLoadPaletteKitRejectsBadSize verifies an invalid file is reported rather
// than silently producing a palette.
func TestLoadPaletteKitRejectsBadSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.kit")
	if err := os.WriteFile(path, make([]byte, 200), 0644); err != nil {
		t.Fatalf("write %s failed: %v", path, err)
	}
	var pal [16]uint16
	if err := LoadPaletteKit(path, pal[:]); err == nil {
		t.Error("LoadPaletteKit accepted a file of the wrong size")
	}
}
