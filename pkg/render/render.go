// Package render provides CPC screen rendering functionality.
// This file contains DrawBitmap and EncodeToCpc functions for converting between
// CPC screen memory and displayable bitmap formats.
package render

import (
	"image"

	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
	"github.com/ikari-pl/go-cpc-image/pkg/cpc"
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
	YEgx        int               // EGX line offset (0 or 2)
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

	// The bitmap holds one pixel per picture pixel, and its size says which model
	// wrote it: one buffer row per CRTC line (the screen's own grid) or two (the
	// fixed timeline the EGX modes need). The picture pixels a byte holds are then
	// bmpWidth/NumCol apart in the buffer — 1 in the native grid, the mode's
	// pixel width on the timeline — which is how convert lays it out.
	bmpW, bmpH := bmpLock.Width(), bmpLock.Height()
	if bmpW <= 0 || bmpH <= 0 || b.NumCol <= 0 || b.NumLig <= 0 {
		return
	}
	rowsPerLine := bmpH / b.NumLig
	if rowsPerLine < 1 {
		rowsPerLine = 1
	}
	perByte := 8 / cpc.PixelWidth(b.VirtualMode, 0, 0)
	step := bmpW / (b.NumCol * perByte)
	if step < 1 {
		step = 1
	}

	for line := 0; line < b.NumLig; line++ {
		cpcAddr := cpc.CpcAddress(line, b.NumCol, b.NumLig)
		y := line * rowsPerLine

		for col := 0; col < b.NumCol; col++ {
			if egx && (line&1) != ligneStart {
				continue // the other line phase owns this row
			}
			var pixByte byte = 0
			decal := 0
			for p := 0; p < perByte; p++ {
				x := (col*perByte + p) * step
				pen := cpc.GetPenColor(bmpLock, x, y, b.Palette, b.CpcPlus)
				pixByte |= byte(cpc.TabOctetMode[pen] >> uint(decal))
				decal++
			}
			addr := cpcAddr + col
			if addr < len(b.ScreenData) {
				b.ScreenData[addr] = pixByte
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
		cpcAddr := cpc.CpcAddress(y>>1, b.NumCol, b.NumLig)
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

// DrawBitmap renders CPC screen memory to a bitmap.
//
// The standard modes 0/1/2 give every line the same pixel width, so their picture
// is drawn on the screen's own grid: one bitmap pixel per picture pixel, one row
// per CRTC line (drawNative). The modes whose lines do not share one pixel width —
// EGX 3/4, which alternate two of them — and the modes carrying a per-line or
// per-column table (Mode X 5, Split 6, the ASCII modes, sprites) keep the fixed
// display canvas of numCol*8 x numLig*2 dot positions, the only grid that can hold
// them (drawCanvas).
func (b *BitmapCpc) DrawBitmap(numCol, numLig int, realWidth int, isSprite bool) *bitmap.DirectBitmap {
	if !isSprite && b.nativeGrid() {
		return b.drawNative(numCol, numLig, realWidth)
	}
	return b.drawCanvas(numCol, numLig, realWidth, isSprite)
}

// nativeGrid reports whether the screen's lines all hold the same number of
// picture pixels, so the screen's own grid describes the picture exactly.
func (b *BitmapCpc) nativeGrid() bool {
	return b.VirtualMode >= 0 && b.VirtualMode <= 2
}

// drawNative draws the screen on its own grid: numCol*8/tx pixels across, tx being
// the mode's pixel width, and one row per CRTC line. A realWidth larger than the
// grid asks for a wider picture, and then a whole number of bitmap pixels is spent
// on each picture pixel. Magnifying the result by tx across and two rows down
// rebuilds the display canvas pixel for pixel.
func (b *BitmapCpc) drawNative(numCol, numLig, realWidth int) *bitmap.DirectBitmap {
	tx := cpc.PixelWidth(b.VirtualMode, 0, 0)
	if tx < 1 {
		tx = 1
	}
	nativeW := numCol * 8 / tx

	width := nativeW
	if realWidth > nativeW {
		width = realWidth
	}
	step := 1
	if nativeW > 0 && width > nativeW {
		step = width / nativeW
	}

	loc := bitmap.NewDirectBitmap(width, numLig)

	for line := 0; line < numLig; line++ {
		cpcAddr := cpc.CpcAddress(line, numCol, numLig)
		x := 0

		for col := 0; col < numCol; col++ {
			addr := cpcAddr + col
			var pixByte byte = 0
			if addr < len(b.ScreenData) {
				pixByte = b.ScreenData[addr]
			}

			put := func(pen int) {
				c := cpc.PaletteColor(b.Palette[pen], b.CpcPlus)
				for i := 0; i < step; i++ {
					loc.SetPixel(x+i, line, c)
				}
				x += step
			}

			switch b.VirtualMode {
			case 0: // Mode 0: 2 pixels per byte, 16 colors
				put(int(pixByte>>7) + int((pixByte&0x20)>>3) + int((pixByte&0x08)>>2) + int((pixByte&0x02)<<2))
				put(int((pixByte&0x40)>>6) + int((pixByte&0x10)>>2) + int((pixByte&0x04)>>1) + int((pixByte&0x01)<<3))

			case 1: // Mode 1: 4 pixels per byte, 4 colors
				put(int((pixByte>>7)&1) + int((pixByte>>2)&2))
				put(int((pixByte>>6)&1) + int((pixByte>>1)&2))
				put(int((pixByte>>5)&1) + int((pixByte>>0)&2))
				put(int((pixByte>>4)&1) + int((pixByte<<1)&2))

			case 2: // Mode 2: 8 pixels per byte, 2 colors
				for i := 8; i > 0; i-- {
					put(int(pixByte>>uint(i-1)) & 1)
				}
			}
		}
	}

	return loc
}

// drawCanvas draws the screen on the fixed display canvas the renderer has always
// used: numCol*8 dot positions across and two rows per CRTC line, so a mode-0 pixel
// spans 4 of those dots, a mode-1 pixel 2 and a mode-2 pixel 1. It is what a screen
// is magnified onto for display, and the only grid that can hold a screen whose
// lines have different pixel widths.
func (b *BitmapCpc) drawCanvas(numCol, numLig int, realWidth int, isSprite bool) *bitmap.DirectBitmap {
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
			if (y&2) == b.YEgx {
				mode = b.VirtualMode - 2
			} else {
				mode = b.VirtualMode - 3
			}
		}

		var cpcAddr int
		if isSprite {
			cpcAddr = numCol * (y >> 1)
		} else {
			cpcAddr = cpc.CpcAddress(y>>1, numCol, numLig)
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
		if y > 255 && (b.NumCol*b.NumLig > 0x4000) {
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

// RenderToRGBA creates an RGBA image from the CPC screen data: the grid the
// screen's own pixels define (see DrawBitmap), one pixel per picture pixel and one
// row per CRTC line. The modes that have no such grid — EGX 3/4 and the ones with
// per-line tables — come back on the display canvas.
//
// This is the picture the output stage starts from: reaching the grid a file is
// written at is that stage's job, done by duplicating whole pixels.
func (b *BitmapCpc) RenderToRGBA() *image.RGBA {
	bmp := b.DrawBitmap(b.NumCol, b.NumLig, 0, false)
	return bmp.Image()
}

// RenderToRGBADisplay creates the RGBA image the GUI shows: the display canvas of
// NumCol*8 x NumLig*2 dot positions, i.e. the screen's own grid magnified by whole
// pixels — the mode's pixel width across, two rows per line — so it is the same
// picture as RenderToRGBA, only bigger. Nothing but a display goes through here.
func (b *BitmapCpc) RenderToRGBADisplay() *image.RGBA {
	if !b.nativeGrid() {
		return b.DrawBitmap(b.NumCol, b.NumLig, 0, false).Image()
	}
	return bitmap.ScaleImage(b.RenderToRGBA(), b.NumCol<<3, b.NumLig<<1)
}
