// Package cpc provides CPC memory layout and screen addressing.
package cpc

// CpcAddress calculates the CPC screen memory address of a CRTC line. The
// hardware walks the memory itself: eight lines make a character row, and each
// line of that row starts a further 0x800 bytes into the dump.
// - Standard: 80 cols x 200 lines at #C000
// - Overscan: 96 cols x 272 lines at #0200
// Address = (line/8)*numCol + (line%8)*#800, with a #3800 offset past line 127
// of an overscan screen, whose dump is more than 16 KB and therefore wraps.
func CpcAddress(line int, numCol int, numLig int) int {
	addr := (line>>3)*numCol + (line&7)*0x800
	if line > 127 && (numCol*numLig > 0x4000) {
		addr += 0x3800
	}
	return addr
}

// BitmapSize calculates the total size needed for CPC screen buffer: the bytes
// of the last line, plus the columns that precede it in its character row.
func BitmapSize(numCol int, numLig int) int {
	return numCol + CpcAddress(numLig-1, numCol, numLig)
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

// Underscan screen represents underscan CPC screen dimensions: 64 bytes per
// line (512 pixels wide in mode 0, 256 in mode 1, 512 in mode 2) at half the
// height of a standard screen.
const (
	UnderscanCols  = 64  // 64 columns * 8 pixels = 512 pixels wide
	UnderscanLines = 192 // 192 lines * 2 pixels = 384 pixels high
)

// ModePalOffset is the offset of the embedded ModePal block in a standard SCR
// (mode byte + 16 ink values). Overscan dumps carry it at
// OverscanModePalOffset instead; underscan dumps keep it here, exactly like
// standard ones.
const ModePalOffset = 0x17D0

// OverscanModePalOffset is where an overscan SCR keeps its ModePal block
// instead of ModePalOffset. fileio.SaveSCR writes the palette there and
// applySCRPalette reads it back from there, for the classic and the 33-byte
// CPC Plus blocks alike.
const OverscanModePalOffset = 0x600

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

// NewUnderscanScreen returns an underscan CPC screen configuration
func NewUnderscanScreen() ScreenConfig {
	return ScreenConfig{
		NumCol: UnderscanCols,
		NumLig: UnderscanLines,
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

// GetAdr returns the memory address of a CRTC line in this screen's layout
func (s ScreenConfig) GetAdr(line int) int {
	return CpcAddress(line, s.NumCol, s.NumLig)
}

// GetBitmapSize returns the total buffer size needed for this screen configuration
func (s ScreenConfig) GetBitmapSize() int {
	return BitmapSize(s.NumCol, s.NumLig)
}

// PKSLPixelBytes is the pixel-only size of a standard screen: 80 columns ×
// 200 lines = 16000 bytes, excluding the CPC layout gaps where the display
// code and the ModePal live.
const PKSLPixelBytes = StandardCols * StandardLines

// UnderscanPixelBytes is the pixel-only size of an underscan screen: 64 columns
// × 192 lines = 12288 bytes, excluding the same layout gaps.
const UnderscanPixelBytes = UnderscanCols * UnderscanLines

// ScreenToColumnMajor extracts the pixel bytes of a CPC screen dump in the
// column-major order used by the PKSL/PKUL formats (numCol×numLig bytes: 16000
// for a standard screen, 12288 for an underscan one):
//
//	for x := 0; x < numCol; x++ {
//	  for y := 0; y < numLig; y++ {
//	    out = append(out, bmp[x + CpcAddress(y, numCol, numLig)])
//	  }
//	}
//
// bmp must cover the pixel addresses; missing bytes are treated as zero.
func ScreenToColumnMajor(bmp []byte, numCol, numLig int) []byte {
	out := make([]byte, numCol*numLig)
	i := 0
	for x := 0; x < numCol; x++ {
		for y := 0; y < numLig; y++ {
			adr := x + CpcAddress(y, numCol, numLig)
			if adr < len(bmp) {
				out[i] = bmp[adr]
			}
			i++
		}
	}
	return out
}

// ColumnMajorToScreen scatters a column-major pixel payload back into a CPC
// screen dump of BitmapSize(numCol, numLig) bytes.
// Gaps between pixel regions (where the display code / ModePal live) are left
// zeroed.
func ColumnMajorToScreen(data []byte, numCol, numLig int) []byte {
	size := BitmapSize(numCol, numLig)
	out := make([]byte, size)
	i := 0
	for x := 0; x < numCol; x++ {
		for y := 0; y < numLig; y++ {
			if i >= len(data) {
				return out
			}
			adr := x + CpcAddress(y, numCol, numLig)
			if adr < len(out) {
				out[adr] = data[i]
			}
			i++
		}
	}
	return out
}

// ModePalBlockLen returns the length of the ModePal block a screen carries: 33
// bytes for a CPC Plus screen (the mode byte has bit 7 set and is followed by 16
// low/high 12-bit colour pairs) and 17 bytes otherwise (mode + 16 classic ink
// indices).
func ModePalBlockLen(modeByte byte) int {
	if modeByte&0x80 != 0 {
		return 33
	}
	return 17
}

// ExtractModePal returns the ModePal block from a standard CPC screen dump at
// ModePalOffset: the 17-byte classic block (mode + 16 ink indices, 0xFF for an
// unused pen) or the 33-byte CPC Plus block when the mode byte has bit 7 set.
// Returns nil if the dump is too short or a classic block is not a plausible
// palette (an implausible CPC Plus block cannot be told apart from pixel data,
// so it is accepted as it is and left to the caller's non-empty check).
func ExtractModePal(bmp []byte) []byte {
	if len(bmp) <= ModePalOffset {
		return nil
	}
	n := ModePalBlockLen(bmp[ModePalOffset])
	if len(bmp) < ModePalOffset+n {
		return nil
	}
	pal := make([]byte, n)
	copy(pal, bmp[ModePalOffset:ModePalOffset+n])
	if n == 33 {
		return pal
	}
	for _, b := range pal {
		if b != 0xFF && b > 26 {
			return nil
		}
	}
	return pal
}

// ExtractClassicModePal returns the 17-byte classic ModePal block, or nil when
// the screen carries a CPC Plus block instead (bit 7 set) or no plausible
// palette at all. Use it wherever only classic ink indices are usable: the
// 17-byte PKS palette field of the "L" variants, and the mode-3 certainty check
// of the variant detection.
func ExtractClassicModePal(bmp []byte) []byte {
	pal := ExtractModePal(bmp)
	if len(pal) != 17 {
		return nil
	}
	return pal
}

// EmbedModePal writes a ModePal block into a standard CPC screen dump at
// ModePalOffset: the 17-byte classic block, or a 33-byte CPC Plus block when pal
// is that long. bmp is grown if needed.
func EmbedModePal(bmp []byte, pal []byte) []byte {
	return EmbedModePalAt(bmp, ModePalOffset, pal)
}

// EmbedModePalAt writes a ModePal block at an arbitrary offset — an overscan dump
// keeps its palette at OverscanModePalOffset instead. The block width follows
// the payload, the same rule ExtractModePal reads it back with. bmp is grown if
// needed.
func EmbedModePalAt(bmp []byte, offset int, pal []byte) []byte {
	n := 17
	if len(pal) > 17 {
		n = 33
	}
	if len(pal) < n {
		n = len(pal)
	}
	need := offset + n
	if len(bmp) < need {
		grown := make([]byte, need)
		copy(grown, bmp)
		bmp = grown
	}
	copy(bmp[offset:], pal[:n])
	return bmp
}
