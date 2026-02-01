// Package render provides CPC screen rendering functionality.
// This file contains DrawBitmap and EncodeToCpc functions for converting between
// CPC screen memory and displayable bitmap formats.
package render

import (
	"image"

	"github.com/ikari/go-cpc-image/pkg/bitmap"
	"github.com/ikari/go-cpc-image/pkg/cpc"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// BitmapCpc represents CPC screen memory and rendering state
type BitmapCpc struct {
	ScreenData      [0x10000]byte     // 64KB CPC screen buffer
	Palette     [16]int           // Current palette (16 pens)
	ModeTable     [272]int          // Mode per scanline for split screens
	SplitPalette [96][272][17]int // Per-column palette for split screens
	CpcPlus     bool              // CPC+ mode flag
	NumCol       int               // Number of columns (80 or 96)
	NumLig       int               // Number of lines (200 or 272)
	VirtualMode int               // Virtual screen mode
	IsComputed      bool              // Whether screen data is calculated
}

// NewBitmapCpc creates a new CPC bitmap with specified dimensions and mode
func NewBitmapCpc(tx, ty, mode int) *BitmapCpc {
	b := &BitmapCpc{
		NumCol:       tx >> 3, // Convert pixels to columns (8 pixels per column)
		NumLig:       ty >> 1, // Convert pixels to lines (2 pixels per line)
		VirtualMode: mode,
	}

	// Initialize default palette
	b.Palette = cpc.DefaultPalette

	// Initialize mode table
	for y := 0; y < 272; y++ {
		b.ModeTable[y] = mode
	}

	// Initialize split palette with default palette
	for y := 0; y < 272; y++ {
		for x := 0; x < 96; x++ {
			for i := 0; i < 16; i++ {
				b.SplitPalette[x][y][i] = b.Palette[i]
			}
		}
	}

	return b
}

// NewBitmapCpc creates a new CPC bitmap with columns, lines and CPC+ flag
func NewBitmapCpcWithParams(numCol, numLig int, cpcPlus bool) *BitmapCpc {
	b := &BitmapCpc{
		NumCol:   numCol,
		NumLig:   numLig,
		CpcPlus: cpcPlus,
	}

	// Initialize default palette
	b.Palette = cpc.DefaultPalette

	// Initialize mode table
	for y := 0; y < 272; y++ {
		b.ModeTable[y] = 1 // Default to mode 1
	}

	// Initialize split palette with default palette
	for y := 0; y < 272; y++ {
		for x := 0; x < 96; x++ {
			for i := 0; i < 16; i++ {
				b.SplitPalette[x][y][i] = b.Palette[i]
			}
		}
	}

	return b
}

// GetColorPal returns the RGB color for a palette entry
func (b *BitmapCpc) GetColorPal(palEntry int) bitmap.RgbColor {
	return cpc.GetColor(b.Palette[palEntry], b.CpcPlus)
}

// GetPaletteColor returns the RGB color for a split screen position
func (b *BitmapCpc) GetPaletteColor(x, y, col int) bitmap.RgbColor {
	colorIndex := b.SplitPalette[x][y][col]
	if colorIndex < 27 {
		return cpc.CpcRgbPalette[colorIndex]
	}
	return bitmap.RgbColor{} // Black for invalid indices
}

// EncodeToCpc converts a bitmap to CPC screen memory format.
// This encodes pixels into the CPC's packed byte format where multiple pixels
// are stored per byte depending on the screen mode.
func (b *BitmapCpc) EncodeToCpc(bmpLock *bitmap.DirectBitmap, egx bool, ligneStart int) {
	// Clear screen buffer
	for i := range b.ScreenData {
		b.ScreenData[i] = 0
	}

	width := b.NumCol << 3  // Convert columns to pixels
	height := b.NumLig << 1  // Convert lines to pixels

	// Process each scanline pair
	for y := 0; y < height; y += 2 {
		cpcAddr := cpc.CpcAddress(y, b.NumCol, b.NumLig)
		tx := cpc.PixelWidth(b.VirtualMode, y, 0)

		// Process each byte (8 pixels)
		for x := 0; x < width; x += 8 {
			var pixByte byte = 0
			decal := 0

			// Skip EGX lines if not in the right phase
			if !egx || ((y>>1)&1) == ligneStart {
				// Pack pixels into byte according to mode
				for p := 0; p < 8; p += tx {
					pen := cpc.GetPenColor(bmpLock, x+p, y, b.Palette, b.CpcPlus)
					pixByte |= byte(cpc.TabOctetMode[pen] >> uint(decal))
					decal++
				}
				// Bounds check for ScreenData array
				addr := cpcAddr + (x >> 3)
				if addr < len(b.ScreenData) {
					b.ScreenData[addr] = pixByte
				}
			}
		}
	}
}

// EncodeToCpcForceMode1 converts bitmap to CPC format forcing Mode 1 encoding.
// This is used for specific conversion scenarios where Mode 1 pixel packing is required.
func (b *BitmapCpc) EncodeToCpcForceMode1(bmpLock *bitmap.DirectBitmap) {
	// Clear screen buffer
	for i := range b.ScreenData {
		b.ScreenData[i] = 0
	}

	width := b.NumCol << 3
	height := b.NumLig << 1

	for y := 0; y < height; y += 2 {
		cpcAddr := cpc.CpcAddress(y, b.NumCol, b.NumLig)
		tx := cpc.PixelWidth(b.VirtualMode, y, 0)
		maxPen := cpc.MaxPen(b.VirtualMode, y, 0)

		for x := 0; x < width; x += 8 {
			var pen byte
			var pixByte byte = 0
			decal := 0

			for p := 0; p < 8; p += tx {
				col := bmpLock.GetPixelColor(x+p, y)

				// Find matching pen in palette
				for pen = 0; int(pen) < maxPen; pen++ {
					if b.CpcPlus {
						// CPC+ mode: match 4-bit precision
						if (col.V>>4) == byte((b.Palette[pen]>>8)&0x0F) &&
						   (col.R>>4) == byte((b.Palette[pen]>>4)&0x0F) &&
						   (col.B>>4) == byte(b.Palette[pen]&0x0F) {
							break
						}
					} else {
						// Standard CPC: exact match
						fixedCol := cpc.CpcRgbPalette[b.Palette[pen]]
						if fixedCol.R == col.R && fixedCol.B == col.B && fixedCol.V == col.V {
							break
						}
					}
				}

				// Limit to 4 pens for Mode 1 encoding
				if pen > 3 {
					pen = 3
				}

				pixByte |= byte(cpc.TabOctetMode[pen] >> uint(decal))
				decal++
			}
			b.ScreenData[cpcAddr+(x>>3)] = pixByte
		}
	}
}

// DrawBitmap renders CPC screen memory to a displayable bitmap.
// This decodes the packed CPC format back into individual pixels.
func (b *BitmapCpc) DrawBitmap(numCol, numLig int, realWidth int, isSprite bool) *bitmap.DirectBitmap {
	width := realWidth
	if realWidth == 0 {
		width = numCol << 3
	}

	loc := bitmap.NewDirectBitmap(width, numLig<<1)

	for y := 0; y < numLig<<1; y += 2 {
		// Determine mode for this scanline
		mode := b.VirtualMode
		if b.VirtualMode >= 5 {
			mode = 1 // Mode X and Split use Mode 1 encoding
		} else if b.VirtualMode >= 3 {
			// EGX modes alternate between modes
			if (y&2) == 0 { // yEgx would be stored in context
				mode = b.VirtualMode - 2
			} else {
				mode = b.VirtualMode - 3
			}
		}

		var cpcAddr int
		if isSprite {
			cpcAddr = numCol * (y >> 1)
		} else {
			cpcAddr = cpc.CpcAddress(y, numCol, numLig)
		}

		xBitmap := 0

		// Decode each byte
		for x := 0; x < numCol; x++ {
			// Bounds check for ScreenData array
			addr := cpcAddr + x
			var pixByte byte = 0
			if addr < len(b.ScreenData) {
				pixByte = b.ScreenData[addr]
			}

			switch mode {
			case 0: // Mode 0: 2 pixels per byte, 16 colors
				p0 := int(pixByte>>7) + int((pixByte&0x20)>>3) + int((pixByte&0x08)>>2) + int((pixByte&0x02)<<2)
				p1 := int((pixByte&0x40)>>6) + int((pixByte&0x10)>>2) + int((pixByte&0x04)>>1) + int((pixByte&0x01)<<3)

				loc.SetHorizontalLineDouble(xBitmap, y, 4, cpc.PaletteColor(b.Palette[p0], b.CpcPlus))
				loc.SetHorizontalLineDouble(xBitmap+4, y, 4, cpc.PaletteColor(b.Palette[p1], b.CpcPlus))
				xBitmap += 8

			case 1: // Mode 1: 4 pixels per byte, 4 colors
				p0 := int((pixByte>>7)&1) + int((pixByte>>2)&2)
				p1 := int((pixByte>>6)&1) + int((pixByte>>1)&2)
				p2 := int((pixByte>>5)&1) + int((pixByte>>0)&2)
				p3 := int((pixByte>>4)&1) + int((pixByte<<1)&2)

				loc.SetHorizontalLineDouble(xBitmap, y, 2, cpc.PaletteColor(b.Palette[p0], b.CpcPlus))
				loc.SetHorizontalLineDouble(xBitmap+2, y, 2, cpc.PaletteColor(b.Palette[p1], b.CpcPlus))
				loc.SetHorizontalLineDouble(xBitmap+4, y, 2, cpc.PaletteColor(b.Palette[p2], b.CpcPlus))
				loc.SetHorizontalLineDouble(xBitmap+6, y, 2, cpc.PaletteColor(b.Palette[p3], b.CpcPlus))
				xBitmap += 8

			case 2: // Mode 2: 8 pixels per byte, 2 colors
				for i := 8; i > 0; i-- {
					p0 := int(pixByte>>uint(i-1)) & 1
					color := cpc.PaletteColor(b.Palette[p0], b.CpcPlus)
					loc.SetPixel(xBitmap, y, color)
					loc.SetPixel(xBitmap, y+1, color)
					xBitmap++
				}
			}
		}
	}

	return loc
}

// Render renders the CPC screen using split screen palette information.
// This handles complex palette changes per column and scanline.
func (b *BitmapCpc) Render(bmp *bitmap.DirectBitmap, offsetX int) *bitmap.DirectBitmap {
	for y := 0; y < b.NumLig<<1; y += 2 {
		cpcAddr := (y>>4)*b.NumCol + (y&14)*0x400

		// Handle overscan offset for large screens
		if y > 255 && (b.NumCol*b.NumLig > 0x3FFF) {
			cpcAddr += 0x3800
		}

		cpcAddr += offsetX >> 3
		xBitmap := 0

		for x := 0; x < b.NumCol; x++ {
			pixByte := b.ScreenData[cpcAddr+x]
			mode := b.ModeTable[y>>1]

			switch mode {
			case 0: // Mode 0 with split palette
				p0 := int(pixByte>>7) + int((pixByte&0x20)>>3) + int((pixByte&0x08)>>2) + int((pixByte&0x02)<<2)
				p1 := int((pixByte&0x40)>>6) + int((pixByte&0x10)>>2) + int((pixByte&0x04)>>1) + int((pixByte&0x01)<<3)

				color0 := cpc.PaletteColor(b.SplitPalette[x][y>>1][p0], b.CpcPlus)
				color1 := cpc.PaletteColor(b.SplitPalette[x][y>>1][p1], b.CpcPlus)

				bmp.SetHorizontalLineDouble(xBitmap, y, 4, color0)
				bmp.SetHorizontalLineDouble(xBitmap+4, y, 4, color1)
				xBitmap += 8

			case 1, 3: // Mode 1 or EGX Mode 1 with split palette
				p0 := int((pixByte>>7)&1) + int((pixByte>>2)&2)
				p1 := int((pixByte>>6)&1) + int((pixByte>>1)&2)
				p2 := int((pixByte>>5)&1) + int((pixByte>>0)&2)
				p3 := int((pixByte>>4)&1) + int((pixByte<<1)&2)

				bmp.SetHorizontalLineDouble(xBitmap, y, 2, cpc.PaletteColor(b.SplitPalette[x][y>>1][p0], b.CpcPlus))
				bmp.SetHorizontalLineDouble(xBitmap+2, y, 2, cpc.PaletteColor(b.SplitPalette[x][y>>1][p1], b.CpcPlus))
				bmp.SetHorizontalLineDouble(xBitmap+4, y, 2, cpc.PaletteColor(b.SplitPalette[x][y>>1][p2], b.CpcPlus))
				bmp.SetHorizontalLineDouble(xBitmap+6, y, 2, cpc.PaletteColor(b.SplitPalette[x][y>>1][p3], b.CpcPlus))
				xBitmap += 8

			case 2: // Mode 2 with split palette
				for i := 8; i > 0; i-- {
					p0 := int(pixByte>>uint(i-1)) & 1
					color := cpc.PaletteColor(b.SplitPalette[x][y>>1][p0], b.CpcPlus)
					bmp.SetPixel(xBitmap, y, color)
					bmp.SetPixel(xBitmap, y+1, color)
					xBitmap++
				}
			}
		}
	}

	return bmp
}

// RenderToRGBA creates an RGBA image from the CPC screen data for display in GUI
func (b *BitmapCpc) RenderToRGBA() *image.RGBA {
	bmp := b.DrawBitmap(b.NumCol, b.NumLig, 0, false)
	return bmp.Image()
}