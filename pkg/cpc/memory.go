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
	StandardCols  = 80   // 80 columns * 8 pixels = 640 pixels wide
	StandardLines = 200  // 200 lines * 2 pixels = 400 pixels high
)

// OverscanScreen represents overscan CPC screen dimensions
const (
	OverscanCols  = 96   // 96 columns * 8 pixels = 768 pixels wide
	OverscanLines = 272  // 272 lines * 2 pixels = 544 pixels high
)

// ScreenConfig holds the current screen configuration
type ScreenConfig struct {
	NumCol int // Number of columns (80 standard, 96 overscan)
	NumLig int // Number of lines (200 standard, 272 overscan)
	YEgx  int // EGX line offset (for EGX modes)
}

// NewStandardScreen returns a standard CPC screen configuration
func NewStandardScreen() ScreenConfig {
	return ScreenConfig{
		NumCol: StandardCols,
		NumLig: StandardLines,
		YEgx:  0,
	}
}

// NewOverscanScreen returns an overscan CPC screen configuration
func NewOverscanScreen() ScreenConfig {
	return ScreenConfig{
		NumCol: OverscanCols,
		NumLig: OverscanLines,
		YEgx:  0,
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