// Package cpc provides CPC memory layout and screen addressing.
package cpc

import (
	"bytes"
	"testing"
)

// TestCpcAddress tests CPC screen memory address calculation with known values.
// The argument is a CRTC line: eight lines make a character row, and each line
// of that row starts a further 0x800 bytes into the dump.
func TestCpcAddress(t *testing.T) {
	tests := []struct {
		name   string
		line   int
		numCol int
		numLig int
		expect int
	}{
		// Standard screen (80 cols x 200 lines)
		{"Standard line 0", 0, 80, 200, 0x0000},
		{"Standard line 1", 1, 80, 200, 0x0800},
		{"Standard line 8", 8, 80, 200, 0x0050},
		{"Standard line 9", 9, 80, 200, 0x0850},
		{"Standard line 16", 16, 80, 200, 0x00A0},
		{"Standard line 32", 32, 80, 200, 0x0140},
		{"Standard line 64", 64, 80, 200, 0x0280},
		{"Standard line 80", 80, 80, 200, 0x0320},

		// Overscan screen (96 cols x 272 lines at #0200); its dump passes 16 KB,
		// so line 128 and later take the #3800 wrap.
		{"Overscan line 0", 0, 96, 272, 0x0000},
		{"Overscan line 1", 1, 96, 272, 0x0800},
		{"Overscan line 8", 8, 96, 272, 0x0060},
		{"Overscan line 128", 128, 96, 272, 0x3E00},
		{"Overscan line 129", 129, 96, 272, 0x4600},
		{"Overscan line 135", 135, 96, 272, 0x7600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CpcAddress(tt.line, tt.numCol, tt.numLig)
			if got != tt.expect {
				t.Errorf("CpcAddress(%d, %d, %d) = 0x%04X, want 0x%04X",
					tt.line, tt.numCol, tt.numLig, got, tt.expect)
			}
		})
	}
}

// TestCpcAddressTakesCrtcLines pins the contract of CpcAddress: its argument is
// a CRTC line, not the pixel row of the doubled canvas the pipeline still
// stores. The two agree wherever the canvas samples a line — even pixel rows,
// which is where every caller reads it — so this was a renaming of the
// convention, not a move of any address.
func TestCpcAddressTakesCrtcLines(t *testing.T) {
	for _, s := range []struct{ numCol, numLig int }{{80, 200}, {96, 272}, {64, 192}} {
		for line := 0; line < s.numLig; line++ {
			row := line << 1 // the canvas stores each line as two pixel rows
			canvas := (row>>4)*s.numCol + (row&14)*0x400
			if row > 255 && s.numCol*s.numLig > 0x4000 {
				canvas += 0x3800
			}
			if got := CpcAddress(line, s.numCol, s.numLig); got != canvas {
				t.Fatalf("CpcAddress(%d, %d, %d) = 0x%04X, want the canvas value 0x%04X",
					line, s.numCol, s.numLig, got, canvas)
			}
		}
	}
}

// TestCpcAddressInterleaving pins the two strides of the layout: consecutive
// lines of a character row are 0x800 apart, and the next character row starts
// numCol bytes after the first line of the previous one.
func TestCpcAddressInterleaving(t *testing.T) {
	numCol := 80
	numLig := 200

	for line := 0; line < 7; line++ {
		addr0 := CpcAddress(line, numCol, numLig)
		addr1 := CpcAddress(line+1, numCol, numLig)
		if diff := addr1 - addr0; diff != 0x800 {
			t.Errorf("Address difference between line %d and %d = 0x%04X, want 0x800", line, line+1, diff)
		}
	}

	addr0 := CpcAddress(0, numCol, numLig)
	addr8 := CpcAddress(8, numCol, numLig)
	if diff := addr8 - addr0; diff != numCol {
		t.Errorf("Address difference between line 0 and 8 = 0x%04X, want 0x%04X", diff, numCol)
	}
}

// TestCpcAddressOverscanOffset pins the #3800 offset an overscan dump takes
// past line 127, and that nothing before it — and no smaller screen — ever does.
func TestCpcAddressOverscanOffset(t *testing.T) {
	numCol := 96
	numLig := 272

	if got, want := CpcAddress(128, numCol, numLig), 16*numCol+0x3800; got != want {
		t.Errorf("CpcAddress(128, 96, 272) = 0x%04X, want 0x%04X", got, want)
	}

	for line := 0; line < 128; line++ {
		plain := (line>>3)*numCol + (line&7)*0x800
		if got := CpcAddress(line, numCol, numLig); got != plain {
			t.Errorf("CpcAddress(%d, 96, 272) = 0x%04X, want 0x%04X (no wrap before line 128)", line, got, plain)
		}
	}

	// An underscan dump is 12288 bytes: its lines pass 127 without wrapping.
	if got, want := CpcAddress(128, 64, 192), 16*64; got != want {
		t.Errorf("CpcAddress(128, 64, 192) = 0x%04X, want 0x%04X (a 12 KB dump does not wrap)", got, want)
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
		{"Standard screen", 80, 200, 16336}, // Actual size = 0x3FD0
		{"Overscan screen", 96, 272, 31936}, // Actual size = 0x7CC0
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
		name    string
		cfg     ScreenConfig
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

// TestUnderscanGeometry pins the underscan screen dimensions added for the
// underscan PKS variants, and guards the pre-existing standard/overscan values.
func TestUnderscanGeometry(t *testing.T) {
	if got := BitmapSize(UnderscanCols, UnderscanLines); got != 15872 {
		t.Errorf("BitmapSize(64, 192) = %d, want 15872", got)
	}
	if UnderscanPixelBytes != 12288 {
		t.Errorf("UnderscanPixelBytes = %d, want 12288", UnderscanPixelBytes)
	}
	cfg := NewUnderscanScreen()
	if cfg.NumCol != 64 || cfg.NumLig != 192 {
		t.Errorf("NewUnderscanScreen() = %dx%d, want 64x192", cfg.NumCol, cfg.NumLig)
	}
	if cfg.TailleX() != 512 || cfg.TailleY() != 384 {
		t.Errorf("underscan code space = %dx%d, want 512x384", cfg.TailleX(), cfg.TailleY())
	}
	if cfg.GetBitmapSize() != 15872 {
		t.Errorf("underscan GetBitmapSize() = %d, want 15872", cfg.GetBitmapSize())
	}

	// The standard and overscan values must not move.
	for _, tc := range []struct {
		name      string
		got, want int
	}{
		{"standard bitmap size", BitmapSize(StandardCols, StandardLines), 16336},
		{"overscan bitmap size", BitmapSize(OverscanCols, OverscanLines), 31936},
		{"PKSL pixel bytes", PKSLPixelBytes, 16000},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %d, want %d", tc.name, tc.got, tc.want)
		}
	}
}

// TestColumnMajorGeometryRoundTrip verifies the generalised column-major helpers
// for both geometries: the payload size follows the geometry, scattering it back
// restores every pixel region byte, and only the layout gaps are left zeroed.
func TestColumnMajorGeometryRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name      string
		numCol    int
		numLig    int
		pixelSize int
	}{
		{"standard 80x200", StandardCols, StandardLines, PKSLPixelBytes},
		{"underscan 64x192", UnderscanCols, UnderscanLines, UnderscanPixelBytes},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bmp := make([]byte, BitmapSize(tc.numCol, tc.numLig))
			for i := range bmp {
				bmp[i] = byte(i*7 + 3)
			}

			payload := ScreenToColumnMajor(bmp, tc.numCol, tc.numLig)
			if len(payload) != tc.pixelSize {
				t.Fatalf("payload size = %d, want %d", len(payload), tc.pixelSize)
			}

			scattered := ColumnMajorToScreen(payload, tc.numCol, tc.numLig)
			if len(scattered) != len(bmp) {
				t.Fatalf("scattered size = %d, want %d", len(scattered), len(bmp))
			}
			for x := 0; x < tc.numCol; x++ {
				for y := 0; y < tc.numLig; y++ {
					adr := x + CpcAddress(y, tc.numCol, tc.numLig)
					if scattered[adr] != bmp[adr] {
						t.Fatalf("pixel (%d,%d) at %d = %d, want %d", x, y, adr, scattered[adr], bmp[adr])
					}
				}
			}
			if !bytes.Equal(ScreenToColumnMajor(scattered, tc.numCol, tc.numLig), payload) {
				t.Error("column-major payload does not survive a screen round-trip")
			}
		})
	}
}

// TestModePalBlockLen verifies the block width follows the mode byte: 33 bytes
// for a CPC Plus screen (bit 7 set, 16 low/high colour pairs) and 17 otherwise.
func TestModePalBlockLen(t *testing.T) {
	tests := []struct {
		modeByte byte
		want     int
	}{
		{0x00, 17}, {0x01, 17}, {0x02, 17}, {0x03, 17}, {0x04, 17},
		{0x80, 33}, {0x8C, 33}, {0x8D, 33}, {0x8E, 33}, {0x8F, 33},
	}
	for _, tt := range tests {
		if got := ModePalBlockLen(tt.modeByte); got != tt.want {
			t.Errorf("ModePalBlockLen(0x%02X) = %d, want %d", tt.modeByte, got, tt.want)
		}
	}
}

// plusBlock builds a 33-byte CPC Plus ModePal block from 12-bit 0x0VBR values,
// the way fileio.SaveSCR writes it: mode byte with bit 7 set, then 16 low/high
// pairs.
func plusBlock(mode int, vals [16]int) []byte {
	block := make([]byte, 33)
	block[0] = byte(mode | 0x8C)
	for i, v := range vals {
		block[1+i*2] = byte(((v >> 4) & 0x0F) | ((v << 4) & 0xF0))
		block[2+i*2] = byte((v >> 8) & 0x0F)
	}
	return block
}

// palDump returns a standard screen dump with block embedded at ModePalOffset.
func palDump(block []byte) []byte {
	return EmbedModePal(make([]byte, BitmapSize(StandardCols, StandardLines)), block)
}

// TestExtractModePalClassicAndPlus verifies that both block widths are read back
// and that the classic-only reader rejects a CPC Plus block, which the mode-3
// certainty check and the 17-byte PKS palette field rely on.
func TestExtractModePalClassicAndPlus(t *testing.T) {
	classic := []byte{1, 0, 6, 24, 26, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	var vals [16]int
	for i := range vals {
		vals[i] = (i << 8) | ((15 - i) << 4) | i
	}
	plus := plusBlock(1, vals)

	if got := ExtractModePal(palDump(classic)); !bytes.Equal(got, classic) {
		t.Errorf("classic ExtractModePal = % x, want % x", got, classic)
	}
	if got := ExtractModePal(palDump(plus)); !bytes.Equal(got, plus) {
		t.Errorf("Plus ExtractModePal = % x, want % x", got, plus)
	}
	if got := ExtractClassicModePal(palDump(classic)); !bytes.Equal(got, classic) {
		t.Errorf("classic ExtractClassicModePal = % x, want % x", got, classic)
	}
	if got := ExtractClassicModePal(palDump(plus)); got != nil {
		t.Errorf("ExtractClassicModePal of a Plus block = % x, want nil", got)
	}

	// A classic block with an out-of-range ink is not a palette.
	bad := implausibleClassicBlock()
	if got := ExtractModePal(palDump(bad)); got != nil {
		t.Errorf("implausible classic block = % x, want nil", got)
	}

	// Too short to hold a block at all.
	if got := ExtractModePal(make([]byte, ModePalOffset)); got != nil {
		t.Errorf("short dump = % x, want nil", got)
	}
}

// implausibleClassicBlock returns a 17-byte block holding an ink index above 26,
// i.e. a plausible-looking mode byte followed by a value no real palette uses.
func implausibleClassicBlock() []byte {
	return []byte{1, 0, 6, 40, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
}

// TestEmbedModePalAt verifies the offset-aware writer: it grows a short dump, and
// the block width follows the payload (a 33-byte slice writes the Plus block).
func TestEmbedModePalAt(t *testing.T) {
	// Growing: an empty dump gets extended to hold the block.
	classic := []byte{1, 0, 6}
	got := EmbedModePalAt(nil, OverscanModePalOffset, classic)
	if len(got) != OverscanModePalOffset+3 {
		t.Fatalf("grown dump length = %d, want %d", len(got), OverscanModePalOffset+3)
	}
	if !bytes.Equal(got[OverscanModePalOffset:], classic) {
		t.Errorf("overscan block = % x, want % x", got[OverscanModePalOffset:], classic)
	}

	// A 33-byte payload writes a 33-byte block, leaving the dump grown to fit.
	var vals [16]int
	vals[0] = 0x4F2
	block := plusBlock(1, vals)
	dump := EmbedModePalAt(make([]byte, BitmapSize(StandardCols, StandardLines)), ModePalOffset, block)
	if !bytes.Equal(dump[ModePalOffset:ModePalOffset+33], block) {
		t.Errorf("Plus block = % x, want % x", dump[ModePalOffset:ModePalOffset+33], block)
	}
	// The unrelated bytes right after the block are untouched.
	if dump[ModePalOffset+33] != 0 {
		t.Errorf("byte after the block = 0x%02X, want 0", dump[ModePalOffset+33])
	}
}
