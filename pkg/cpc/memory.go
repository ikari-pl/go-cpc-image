// Package cpc provides CPC memory layout and screen addressing.
package cpc

// CpcAddress calculates the CPC screen memory address for a given Y coordinate.
// CPC screen memory uses interleaved addressing:
// - Standard: 80 cols x 200 lines at #C000
// - Overscan: 96 cols x 272 lines at #0200
// Address = (y/16)*numCol + (y%16)*#400, with +#3800 offset for y>255 in overscan
func CpcAddress(y int, numCol int, numLig int) int {
	addr := (y>>4)*numCol + (y&14)*0x400
	if y > 255 && (numCol*numLig > 0x4000) {
		addr += 0x3800
	}
	return addr
}

// BitmapSize calculates the total size needed for CPC screen buffer.
func BitmapSize(numCol int, numLig int) int {
	height := numLig << 1 // Convert lines to pixels (each line = 2 pixels high)
	return numCol + CpcAddress((height&0x3F8)-2, numCol, numLig)
}

// Screen geometry constants and functions

// StandardScreen represents standard CPC screen dimensions
const (
	StandardCols  = 80  // 80 columns * 8 pixels = 640 pixels wide
	StandardLines = 200 // 200 lines * 2 pixels = 400 pixels high
)

// OverscanScreen represents overscan CPC screen dimensions
const (
	OverscanCols  = 96  // 96 columns * 8 pixels = 768 pixels wide
	OverscanLines = 272 // 272 lines * 2 pixels = 544 pixels high
)

// ModePalOffset is the offset of the embedded ModePal block in a standard SCR
// (mode byte + 16 ink values). Overscan dumps carry it at 0x600 instead.
const ModePalOffset = 0x17D0

// ScreenConfig holds the current screen configuration
type ScreenConfig struct {
	NumCol int // Number of columns (80 standard, 96 overscan)
	NumLig int // Number of lines (200 standard, 272 overscan)
	YEgx   int // EGX line offset (for EGX modes)
}

// NewStandardScreen returns a standard CPC screen configuration
func NewStandardScreen() ScreenConfig {
	return ScreenConfig{
		NumCol: StandardCols,
		NumLig: StandardLines,
		YEgx:   0,
	}
}

// NewOverscanScreen returns an overscan CPC screen configuration
func NewOverscanScreen() ScreenConfig {
	return ScreenConfig{
		NumCol: OverscanCols,
		NumLig: OverscanLines,
		YEgx:   0,
	}
}

// TailleX returns the screen width in pixels (columns * 8)
func (s ScreenConfig) TailleX() int {
	return s.NumCol << 3
}

// TailleY returns the screen height in pixels (lines * 2)
func (s ScreenConfig) TailleY() int {
	return s.NumLig << 1
}

// SetTailleX sets the screen width in pixels (auto-calculates columns)
func (s *ScreenConfig) SetTailleX(pixels int) {
	s.NumCol = pixels >> 3
}

// SetTailleY sets the screen height in pixels (auto-calculates lines)
func (s *ScreenConfig) SetTailleY(pixels int) {
	s.NumLig = pixels >> 1
}

// GetAdr calculates the CPC screen memory address for a given Y coordinate
func (s ScreenConfig) GetAdr(y int) int {
	return CpcAddress(y, s.NumCol, s.NumLig)
}

// GetBitmapSize returns the total buffer size needed for this screen configuration
func (s ScreenConfig) GetBitmapSize() int {
	return BitmapSize(s.NumCol, s.NumLig)
}

// PKSLPixelBytes is the pixel-only size of a standard screen: 80 columns ×
// 200 lines = 16000 bytes, excluding the CPC layout gaps where the display
// code and the ModePal live.
const PKSLPixelBytes = StandardCols * StandardLines

// ScreenToColumnMajor extracts the 16000 pixel bytes of a standard CPC screen
// dump in the column-major order used by the PKSL format:
//
//	for x := 0; x < 80; x++ {
//	  for y := 0; y < 200; y++ {
//	    out = append(out, bmp[x + CpcAddress(y*2, 80, 200)])
//	  }
//	}
//
// bmp must cover the pixel addresses; missing bytes are treated as zero.
func ScreenToColumnMajor(bmp []byte) []byte {
	out := make([]byte, PKSLPixelBytes)
	i := 0
	for x := 0; x < StandardCols; x++ {
		for y := 0; y < StandardLines; y++ {
			adr := x + CpcAddress(y<<1, StandardCols, StandardLines)
			if adr < len(bmp) {
				out[i] = bmp[adr]
			}
			i++
		}
	}
	return out
}

// ColumnMajorToScreen scatters a 16000-byte column-major PKSL payload back into
// a standard CPC screen dump (BitmapSize(StandardCols, StandardLines) bytes).
// Gaps between pixel regions (where the display code / ModePal live) are left
// zeroed.
func ColumnMajorToScreen(data []byte) []byte {
	size := BitmapSize(StandardCols, StandardLines)
	out := make([]byte, size)
	i := 0
	for x := 0; x < StandardCols; x++ {
		for y := 0; y < StandardLines; y++ {
			if i >= len(data) {
				return out
			}
			adr := x + CpcAddress(y<<1, StandardCols, StandardLines)
			if adr < len(out) {
				out[adr] = data[i]
			}
			i++
		}
	}
	return out
}

// ExtractModePal returns the 17-byte ModePal block (mode + 16 inks) from a
// standard CPC screen dump if present and plausible. Values are ink indices
// 0..26, or 0xFF for unused pens. Returns nil if the dump is too short or the
// block is not a plausible palette.
func ExtractModePal(bmp []byte) []byte {
	if len(bmp) < ModePalOffset+17 {
		return nil
	}
	pal := make([]byte, 17)
	copy(pal, bmp[ModePalOffset:ModePalOffset+17])
	for _, b := range pal {
		if b != 0xFF && b > 26 {
			return nil
		}
	}
	return pal
}

// EmbedModePal writes a 17-byte ModePal block (mode + 16 inks) into a standard
// CPC screen dump at ModePalOffset. bmp is grown if needed.
func EmbedModePal(bmp []byte, pal []byte) []byte {
	need := ModePalOffset + 17
	if len(bmp) < need {
		grown := make([]byte, need)
		copy(grown, bmp)
		bmp = grown
	}
	n := 17
	if len(pal) < n {
		n = len(pal)
	}
	copy(bmp[ModePalOffset:], pal[:n])
	return bmp
}
