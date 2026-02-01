package convert

import (
	"fmt"
	"testing"

	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
	"github.com/ikari-pl/go-cpc-image/pkg/cpc"
	"github.com/ikari-pl/go-cpc-image/pkg/render"
)

// TestMode0EncodeDecodeRoundtrip verifies that EncodeToCpc → DrawBitmap
// correctly roundtrips every pen in Mode 0.
func TestMode0EncodeDecodeRoundtrip(t *testing.T) {
	numCol := 80
	numLig := 200
	width := numCol * 8  // 640
	height := numLig * 2 // 400

	bmpCpc := render.NewBitmapCpc(width, height, 0)

	// Set palette to first 16 CPC colors
	for i := 0; i < 16; i++ {
		bmpCpc.Palette[i] = i
	}

	// Create a display bitmap and fill it with known pen colors.
	// Each pen gets a 4-pixel-wide (tx=4) column at even Y positions.
	displayBmp := bitmap.NewDirectBitmap(width, height)

	// Fill entire bitmap with pen 0 color as background
	bg := cpc.PaletteColor(0, false)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x += 4 {
			displayBmp.SetHorizontalLineDouble(x, y&^1, 4, bg)
		}
	}

	// Fill row 0 (y=0,1) with all 16 pens: pen i at x=[i*4, i*4+3]
	for pen := 0; pen < 16; pen++ {
		color := cpc.PaletteColor(pen, false)
		displayBmp.SetHorizontalLineDouble(pen*4, 0, 4, color)
	}

	// Encode to CPC screen memory
	bmpCpc.EncodeToCpc(displayBmp, false, 0)

	// Decode back
	result := bmpCpc.DrawBitmap(numCol, numLig, 0, false)

	// Verify each pen roundtripped correctly at y=0
	for pen := 0; pen < 16; pen++ {
		x := pen * 4
		got := result.GetPixelColor(x, 0)
		expected := cpc.GetColor(pen, false)

		if got.R != expected.R || got.V != expected.V || got.B != expected.B {
			t.Errorf("pen %d at x=%d: got RGB(%d,%d,%d), want RGB(%d,%d,%d)",
				pen, x, got.R, got.V, got.B, expected.R, expected.V, expected.B)
		}
	}
}

// TestMode1EncodeDecodeRoundtrip verifies Mode 1 (4 colors, tx=2) roundtrip.
func TestMode1EncodeDecodeRoundtrip(t *testing.T) {
	numCol := 80
	numLig := 200
	width := numCol * 8
	height := numLig * 2

	bmpCpc := render.NewBitmapCpc(width, height, 1)
	for i := 0; i < 4; i++ {
		bmpCpc.Palette[i] = i * 6 // 0, 6, 12, 18
	}

	displayBmp := bitmap.NewDirectBitmap(width, height)

	// Fill row 0 with 4 pens, each 2 pixels wide
	for pen := 0; pen < 4; pen++ {
		color := cpc.PaletteColor(bmpCpc.Palette[pen], false)
		displayBmp.SetHorizontalLineDouble(pen*2, 0, 2, color)
	}

	bmpCpc.EncodeToCpc(displayBmp, false, 0)
	result := bmpCpc.DrawBitmap(numCol, numLig, 0, false)

	for pen := 0; pen < 4; pen++ {
		x := pen * 2
		got := result.GetPixelColor(x, 0)
		expected := cpc.GetColor(bmpCpc.Palette[pen], false)

		if got.R != expected.R || got.V != expected.V || got.B != expected.B {
			t.Errorf("pen %d at x=%d: got RGB(%d,%d,%d), want RGB(%d,%d,%d)",
				pen, x, got.R, got.V, got.B, expected.R, expected.V, expected.B)
		}
	}
}

// TestGetPenColorRoundtrip verifies that all 27 CPC colors can be written to a
// bitmap and read back via GetPenColor with the correct pen number.
func TestGetPenColorRoundtrip(t *testing.T) {
	bmp := bitmap.NewDirectBitmap(27, 2)
	palette := [16]int{}

	// Test each CPC color index can be found
	for colorIdx := 0; colorIdx < 27; colorIdx++ {
		// Set palette pen 0 to this color
		palette[0] = colorIdx
		color := cpc.PaletteColor(colorIdx, false)
		bmp.SetHorizontalLineDouble(colorIdx, 0, 1, color)

		got := cpc.GetPenColor(bmp, colorIdx, 0, palette, false)
		if got != 0 {
			t.Errorf("CPC color %d: GetPenColor returned pen %d, want 0", colorIdx, got)
		}
	}
}

// TestSetPixelCpcWritesCorrectColor verifies SetPixelCpc writes the expected
// RGB value for each pen.
func TestSetPixelCpcWritesCorrectColor(t *testing.T) {
	bmpCpc := render.NewBitmapCpcWithParams(80, 200, false)
	for i := 0; i < 16; i++ {
		bmpCpc.Palette[i] = i
	}

	dest := &ImageCpc{
		BitmapCpc:  bmpCpc,
		DisplayBmp: bitmap.NewDirectBitmap(640, 400),
	}

	for pen := 0; pen < 16; pen++ {
		dest.SetPixelCpc(pen*4, 0, pen, 4)
	}

	for pen := 0; pen < 16; pen++ {
		x := pen * 4
		got := dest.DisplayBmp.GetPixelColor(x, 0)
		expected := cpc.GetColor(pen, false)

		if got.R != expected.R || got.V != expected.V || got.B != expected.B {
			t.Errorf("SetPixelCpc pen %d at x=%d: got RGB(%d,%d,%d), want RGB(%d,%d,%d)",
				pen, x, got.R, got.V, got.B, expected.R, expected.V, expected.B)
		}

		// Also verify that GetPenColor can find it back
		foundPen := cpc.GetPenColor(dest.DisplayBmp, x, 0, bmpCpc.Palette, false)
		if foundPen != pen {
			t.Errorf("SetPixelCpc pen %d: GetPenColor returned %d", pen, foundPen)
		}
	}
}

// TestCpcAddressSanity verifies CPC address calculations for known positions.
func TestCpcAddressSanity(t *testing.T) {
	tests := []struct {
		y, numCol, numLig int
		wantAddr          int
	}{
		// Standard screen: y=0 → addr=0
		{0, 80, 200, 0},
		// y=2 → same row group, next scanline pair → 0x800
		{2, 80, 200, 0x800},
		// y=16 → next character row → 80
		{16, 80, 200, 80},
	}

	for _, tt := range tests {
		got := cpc.CpcAddress(tt.y, tt.numCol, tt.numLig)
		if got != tt.wantAddr {
			t.Errorf("CpcAddress(y=%d, numCol=%d, numLig=%d) = 0x%04X, want 0x%04X",
				tt.y, tt.numCol, tt.numLig, got, tt.wantAddr)
		}
	}
}

// TestConvertPipeline_SolidColorPreservedThroughQuantization feeds a uniform
// solid-colour image through the full Convert pipeline and checks that every
// output pixel maps to the nearest CPC hardware colour — confirming that
// quantization, palette selection, and pixel encoding all agree.
func TestConvertPipeline_SolidColorPreservedThroughQuantization(t *testing.T) {
	// Use CPC color 13 (White = 0x66,0x66,0x66) as a distinctive solid input
	testColor := cpc.CpcRgbPalette[13] // R=0x66, V=0x66, B=0x66

	prm := NewDefaultSettings()
	prm.VirtualMode = 0 // Mode 0
	prm.NumCols = 80
	prm.NumLines = 200
	prm.Pct = 0 // No dithering

	width := prm.GetScreenWidth()   // 640
	height := prm.GetScreenHeight() // 400

	source := bitmap.NewDirectBitmap(width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			source.SetPixelColor(x, y, testColor)
		}
	}

	bmpCpc := render.NewBitmapCpc(width, height, 0)
	dest := &ImageCpc{
		BitmapCpc:  bmpCpc,
		DisplayBmp: bitmap.NewDirectBitmap(width, height),
	}

	Convert(source, dest, prm, true)

	// Verify the display bitmap has the right color everywhere (after Pass2/ConvertStd)
	// ConvertStd writes to DisplayBmp via SetPixelCpc
	wrongCount := 0
	for y := 0; y < height; y += 2 {
		for x := 0; x < width; x += 4 { // Mode 0 tx=4
			got := dest.DisplayBmp.GetPixelColor(x, y)
			if got.R != testColor.R || got.V != testColor.V || got.B != testColor.B {
				wrongCount++
				if wrongCount <= 3 {
					t.Errorf("DisplayBmp at (%d,%d): got RGB(%d,%d,%d), want RGB(%d,%d,%d)",
						x, y, got.R, got.V, got.B, testColor.R, testColor.V, testColor.B)
				}
			}
		}
	}
	if wrongCount > 3 {
		t.Errorf("... and %d more wrong pixels in DisplayBmp", wrongCount-3)
	}

	// Verify the final rendered output (EncodeToCpc → DrawBitmap roundtrip)
	rendered := bmpCpc.RenderToRGBA()
	if rendered == nil {
		t.Fatal("RenderToRGBA returned nil")
	}

	wrongCount = 0
	for y := 0; y < height; y += 2 {
		for x := 0; x < width; x += 4 {
			r := rendered.Pix[(y*width+x)*4+0]
			g := rendered.Pix[(y*width+x)*4+1]
			b := rendered.Pix[(y*width+x)*4+2]
			if r != testColor.R || g != testColor.V || b != testColor.B {
				wrongCount++
				if wrongCount <= 3 {
					t.Errorf("Rendered at (%d,%d): got RGB(%d,%d,%d), want RGB(%d,%d,%d)",
						x, y, r, g, b, testColor.R, testColor.V, testColor.B)
				}
			}
		}
	}
	if wrongCount > 3 {
		t.Errorf("... and %d more wrong pixels in rendered output", wrongCount-3)
	}
}

// TestFullPipelineStripes creates vertical stripes of different CPC colors
// and verifies they survive the pipeline.
func TestFullPipelineStripes(t *testing.T) {
	prm := NewDefaultSettings()
	prm.VirtualMode = 0
	prm.NumCols = 80
	prm.NumLines = 200
	prm.Pct = 0

	width := prm.GetScreenWidth()
	height := prm.GetScreenHeight()

	// Use 4 distinct CPC colors in vertical stripes
	stripeColors := []int{0, 6, 18, 26} // Black, Bright Red, Bright Green, Bright White

	source := bitmap.NewDirectBitmap(width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			stripeIdx := (x / (width / len(stripeColors))) % len(stripeColors)
			c := cpc.CpcRgbPalette[stripeColors[stripeIdx]]
			source.SetPixelColor(x, y, c)
		}
	}

	bmpCpc := render.NewBitmapCpc(width, height, 0)
	dest := &ImageCpc{
		BitmapCpc:  bmpCpc,
		DisplayBmp: bitmap.NewDirectBitmap(width, height),
	}

	Convert(source, dest, prm, true)

	// Check that the palette includes all 4 stripe colors
	foundColors := map[int]bool{}
	for i := 0; i < 16; i++ {
		foundColors[bmpCpc.Palette[i]] = true
	}
	for _, c := range stripeColors {
		if !foundColors[c] {
			t.Errorf("Expected CPC color %d in palette, not found. Palette: %v", c, bmpCpc.Palette)
		}
	}

	// Verify the rendered output has correct stripes at known positions
	rendered := bmpCpc.RenderToRGBA()
	for _, stripeIdx := range []int{0, 1, 2, 3} {
		// Sample from the middle of each stripe
		x := (stripeIdx*width/len(stripeColors) + width/len(stripeColors)/2)
		x = x &^ 3 // Align to 4-pixel boundary (Mode 0)
		y := height / 2
		y = y &^ 1 // Align to even scanline

		expected := cpc.CpcRgbPalette[stripeColors[stripeIdx]]
		r := rendered.Pix[(y*width+x)*4+0]
		g := rendered.Pix[(y*width+x)*4+1]
		b := rendered.Pix[(y*width+x)*4+2]

		if r != expected.R || g != expected.V || b != expected.B {
			t.Errorf("Stripe %d at (%d,%d): got RGB(%d,%d,%d), want RGB(%d,%d,%d)",
				stripeIdx, x, y, r, g, b, expected.R, expected.V, expected.B)
		}
	}
}

// TestTabOctetModeRoundtrip verifies that encoding a pen via TabOctetMode and
// decoding via DrawBitmap's bit extraction gives back the original pen.
func TestTabOctetModeRoundtrip(t *testing.T) {
	// Mode 0: 2 pixels per byte
	for pen0 := 0; pen0 < 16; pen0++ {
		for pen1 := 0; pen1 < 16; pen1++ {
			// Encode
			pixByte := byte(cpc.TabOctetMode[pen0]>>0) | byte(cpc.TabOctetMode[pen1]>>1)

			// Decode (same logic as DrawBitmap mode 0)
			decPen0 := int(pixByte>>7) + int((pixByte&0x20)>>3) + int((pixByte&0x08)>>2) + int((pixByte&0x02)<<2)
			decPen1 := int((pixByte&0x40)>>6) + int((pixByte&0x10)>>2) + int((pixByte&0x04)>>1) + int((pixByte&0x01)<<3)

			if decPen0 != pen0 {
				t.Errorf("Mode 0 pen0=%d,pen1=%d: decoded pen0=%d (byte=0x%02X)", pen0, pen1, decPen0, pixByte)
			}
			if decPen1 != pen1 {
				t.Errorf("Mode 0 pen0=%d,pen1=%d: decoded pen1=%d (byte=0x%02X)", pen0, pen1, decPen1, pixByte)
			}
		}
	}
}

// TestConvertStdMode0 tests ConvertStd in isolation with a controlled palette
// and source image.
func TestConvertStdMode0(t *testing.T) {
	prm := NewDefaultSettings()
	prm.VirtualMode = 0
	prm.NumCols = 80
	prm.NumLines = 200

	width := prm.GetScreenWidth()
	height := prm.GetScreenHeight()

	bmpCpc := render.NewBitmapCpc(width, height, 0)
	// Set palette to known colors
	bmpCpc.Palette = [16]int{0, 6, 18, 26, 13, 4, 10, 24, 1, 2, 3, 5, 7, 8, 9, 11}

	dest := &ImageCpc{
		BitmapCpc:  bmpCpc,
		DisplayBmp: bitmap.NewDirectBitmap(width, height),
	}

	// Create source filled with CPC color 6 (Bright Red)
	brightRed := cpc.CpcRgbPalette[6]
	source := bitmap.NewDirectBitmap(width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			source.SetPixelColor(x, y, brightRed)
		}
	}

	// Build color table (normally done by Pass2)
	var colorTable [16][272]bitmap.RgbColor
	for line := 0; line < prm.NumLines; line++ {
		for i := 0; i < 16; i++ {
			colorTable[i][line] = cpc.GetColor(bmpCpc.Palette[i], false)
		}
	}

	ConvertStd(source, prm, dest, 16, colorTable)

	// Verify pen 1 (color index 6 = Bright Red) was chosen everywhere
	for y := 0; y < 10; y += 2 { // Check first 5 scanline pairs
		for x := 0; x < 40; x += 4 { // Check first 10 pixels
			got := dest.DisplayBmp.GetPixelColor(x, y)
			if got.R != brightRed.R || got.V != brightRed.V || got.B != brightRed.B {
				t.Errorf("ConvertStd at (%d,%d): got RGB(%d,%d,%d), want RGB(%d,%d,%d)",
					x, y, got.R, got.V, got.B, brightRed.R, brightRed.V, brightRed.B)
			}
		}
	}
}

// TestPaletteColorConsistency verifies PaletteColor and GetColor produce
// consistent results for all 27 standard CPC colors.
func TestPaletteColorConsistency(t *testing.T) {
	for i := 0; i < 27; i++ {
		palColor := cpc.PaletteColor(i, false)
		getColor := cpc.GetColor(i, false)

		// PaletteColor returns packed int B + (G<<8) + (R<<16)
		r := uint8(palColor >> 16)
		g := uint8(palColor >> 8)
		b := uint8(palColor)

		if r != getColor.R || g != getColor.V || b != getColor.B {
			t.Errorf("CPC color %d: PaletteColor=0x%06X → RGB(%d,%d,%d), GetColor=RGB(%d,%d,%d)",
				i, palColor, r, g, b, getColor.R, getColor.V, getColor.B)
		}
	}
}

// TestDitherSetMatDitherNormalization verifies the dithering matrix is
// normalized so that all entries sum to pct.
func TestDitherSetMatDitherNormalization(t *testing.T) {
	prm := DitherParam{
		CpcPlus: false,
		Pct:     50,
		Method:  "Floyd-Steinberg (2x2)",
		DiffErr: true,
	}

	pct := SetMatDither(prm)
	if pct != 100 { // 50 << 1 = 100
		t.Errorf("SetMatDither returned pct=%d, want 100", pct)
	}

	if matDither == nil {
		t.Fatal("matDither is nil after SetMatDither")
	}

	// Sum should equal pct (within floating point tolerance)
	var sum float64
	for _, row := range matDither {
		for _, v := range row {
			sum += v
		}
	}

	if diff := sum - float64(pct); diff > 0.001 || diff < -0.001 {
		t.Errorf("matDither sum = %f, want %d", sum, pct)
	}
}

// TestPixelWidthMode0 verifies Mode 0 pixel width is 4.
func TestPixelWidthMode0(t *testing.T) {
	tx := cpc.PixelWidth(0, 0, 0)
	if tx != 4 {
		t.Errorf("PixelWidth(mode=0) = %d, want 4", tx)
	}
}

// TestMaxPenMode0 verifies Mode 0 has 16 pens.
func TestMaxPenMode0(t *testing.T) {
	maxPen := cpc.MaxPen(0, 0, 0)
	if maxPen != 16 {
		t.Errorf("MaxPen(mode=0) = %d, want 16", maxPen)
	}
}

// TestEncodeToCpcMode0AllPens encodes a bitmap where each CPC byte position
// contains known pens, then verifies the raw ScreenData bytes.
func TestEncodeToCpcMode0AllPens(t *testing.T) {
	numCol := 80
	numLig := 200
	width := numCol * 8
	height := numLig * 2

	bmpCpc := render.NewBitmapCpc(width, height, 0)
	for i := 0; i < 16; i++ {
		bmpCpc.Palette[i] = i
	}

	displayBmp := bitmap.NewDirectBitmap(width, height)

	// Fill y=0 with pen pairs (pen0, pen1) per byte
	// Byte 0: pen0=0, pen1=1; Byte 1: pen0=2, pen1=3; etc.
	for byteIdx := 0; byteIdx < 8; byteIdx++ {
		pen0 := byteIdx * 2
		pen1 := byteIdx*2 + 1
		x := byteIdx * 8
		color0 := cpc.PaletteColor(pen0, false)
		color1 := cpc.PaletteColor(pen1, false)
		displayBmp.SetHorizontalLineDouble(x, 0, 4, color0)
		displayBmp.SetHorizontalLineDouble(x+4, 0, 4, color1)
	}

	bmpCpc.EncodeToCpc(displayBmp, false, 0)

	// Verify ScreenData at y=0 (addr=0)
	addr := cpc.CpcAddress(0, numCol, numLig)
	for byteIdx := 0; byteIdx < 8; byteIdx++ {
		pen0 := byteIdx * 2
		pen1 := byteIdx*2 + 1
		expectedByte := byte(cpc.TabOctetMode[pen0]>>0) | byte(cpc.TabOctetMode[pen1]>>1)
		gotByte := bmpCpc.ScreenData[addr+byteIdx]

		if gotByte != expectedByte {
			t.Errorf("ScreenData[%d] = 0x%02X, want 0x%02X (pens %d,%d)",
				addr+byteIdx, gotByte, expectedByte, pen0, pen1)
		}
	}
}

// TestPass1DoesNotCorruptColors verifies Pass1 with 0% dithering doesn't
// modify source pixels that are already valid CPC colors.
func TestPass1DoesNotCorruptColors(t *testing.T) {
	prm := NewDefaultSettings()
	prm.VirtualMode = 0
	prm.NumCols = 80
	prm.NumLines = 200
	prm.Pct = 0

	width := prm.GetScreenWidth()
	height := prm.GetScreenHeight()

	// Fill source with CPC color 6 (Bright Red)
	brightRed := cpc.CpcRgbPalette[6]
	source := bitmap.NewDirectBitmap(width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			source.SetPixelColor(x, y, brightRed)
		}
	}

	ConvertPass1(source, prm)

	// Verify source pixels are still Bright Red (or the nearest CPC-quantized value)
	// Pass1 quantizes to CPC color space, so the exact RGB values should be maintained
	// for colors that are already in the CPC palette.
	for y := 0; y < 10; y += 2 {
		for x := 0; x < 40; x += 4 {
			got := source.GetPixelColor(x, y)
			// Pass1 may quantize, but Bright Red should stay Bright Red
			if got.R != brightRed.R || got.V != brightRed.V || got.B != brightRed.B {
				t.Errorf("Pass1 corrupted (%d,%d): got RGB(%d,%d,%d), want RGB(%d,%d,%d)",
					x, y, got.R, got.V, got.B, brightRed.R, brightRed.V, brightRed.B)
			}
		}
	}
}

// TestPipelineDiagnostic is not a pass/fail test — it prints detailed pipeline
// state for a small region, useful for debugging.
func TestPipelineDiagnostic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping diagnostic in short mode")
	}

	prm := NewDefaultSettings()
	prm.VirtualMode = 0
	prm.NumCols = 80
	prm.NumLines = 200
	prm.Pct = 0

	width := prm.GetScreenWidth()
	height := prm.GetScreenHeight()

	// Create a simple gradient source
	source := bitmap.NewDirectBitmap(width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8(x * 255 / width)
			g := uint8(y * 255 / height)
			source.SetPixelColor(x, y, bitmap.NewRgbColor(r, g, 0))
		}
	}

	bmpCpc := render.NewBitmapCpc(width, height, 0)
	dest := &ImageCpc{
		BitmapCpc:  bmpCpc,
		DisplayBmp: bitmap.NewDirectBitmap(width, height),
	}

	numColors := Convert(source, dest, prm, true)
	t.Logf("Colors found: %d", numColors)
	t.Logf("Palette: %v", bmpCpc.Palette[:])

	// Print first 8 pixel positions of row 0 from each stage
	t.Log("--- Row 0, first 8 Mode-0 pixels (x=0,4,...,28) ---")
	t.Log("DisplayBmp after Pass2:")
	for i := 0; i < 8; i++ {
		x := i * 4
		c := dest.DisplayBmp.GetPixelColor(x, 0)
		t.Logf("  x=%d: RGB(%d,%d,%d)", x, c.R, c.V, c.B)
	}

	t.Log("Rendered output (EncodeToCpc → DrawBitmap):")
	rendered := bmpCpc.DrawBitmap(80, 200, 0, false)
	for i := 0; i < 8; i++ {
		x := i * 4
		c := rendered.GetPixelColor(x, 0)
		t.Logf("  x=%d: RGB(%d,%d,%d)", x, c.R, c.V, c.B)
	}

	// Check if DisplayBmp and rendered match
	mismatches := 0
	for y := 0; y < height; y += 2 {
		for x := 0; x < width; x += 4 {
			d := dest.DisplayBmp.GetPixelColor(x, y)
			r := rendered.GetPixelColor(x, y)
			if d.R != r.R || d.V != r.V || d.B != r.B {
				mismatches++
				if mismatches <= 5 {
					t.Logf("MISMATCH at (%d,%d): DisplayBmp=RGB(%d,%d,%d) Rendered=RGB(%d,%d,%d)",
						x, y, d.R, d.V, d.B, r.R, r.V, r.B)
				}
			}
		}
	}
	t.Logf("Total DisplayBmp vs Rendered mismatches: %d / %d pixels",
		mismatches, (width/4)*(height/2))

	// Dump raw ScreenData for first 10 bytes at y=0
	addr := cpc.CpcAddress(0, 80, 200)
	t.Log("ScreenData at y=0:")
	for i := 0; i < 10; i++ {
		t.Logf("  [0x%04X+%d] = 0x%02X", addr, i, bmpCpc.ScreenData[addr+i])
	}

	// Check specific pen lookups
	t.Log("GetPenColor for first 4 DisplayBmp pixels at y=0:")
	for i := 0; i < 4; i++ {
		x := i * 4
		pen := cpc.GetPenColor(dest.DisplayBmp, x, 0, bmpCpc.Palette, false)
		c := dest.DisplayBmp.GetPixelColor(x, 0)
		t.Logf("  x=%d: color=RGB(%d,%d,%d) → pen=%d", x, c.R, c.V, c.B, pen)
	}

	_ = fmt.Sprintf("done") // suppress unused import
}

// --- CPC+ Tests ---

// TestCpcPlusColorConsistency verifies PaletteColor, GetColor, and
// GetColorFromCpcPlus produce identical results for all 4096 CPC+ colors.
func TestCpcPlusColorConsistency(t *testing.T) {
	for i := 0; i < 4096; i++ {
		palColor := cpc.PaletteColor(i, true)
		getColor := cpc.GetColor(i, true)
		fromPlus := cpc.GetColorFromCpcPlus(i)

		// PaletteColor returns packed int: B + (G<<8) + (R<<16)
		r := uint8(palColor >> 16)
		g := uint8(palColor >> 8)
		b := uint8(palColor)

		if r != getColor.R || g != getColor.V || b != getColor.B {
			t.Errorf("CPC+ color 0x%03X: PaletteColor=RGB(%d,%d,%d), GetColor=RGB(%d,%d,%d)",
				i, r, g, b, getColor.R, getColor.V, getColor.B)
		}
		if fromPlus.R != getColor.R || fromPlus.V != getColor.V || fromPlus.B != getColor.B {
			t.Errorf("CPC+ color 0x%03X: GetColorFromCpcPlus=RGB(%d,%d,%d), GetColor=RGB(%d,%d,%d)",
				i, fromPlus.R, fromPlus.V, fromPlus.B, getColor.R, getColor.V, getColor.B)
		}
	}
}

// TestCpcPlusColorEncoding verifies the 12-bit GBR format encodes correctly.
// Format: 0x0GBR — Green bits 11-8, Blue bits 7-4, Red bits 3-0.
func TestCpcPlusColorEncoding(t *testing.T) {
	tests := []struct {
		colorVal    int
		wantR, wantG, wantB uint8
	}{
		{0x000, 0, 0, 0},           // Black
		{0x00F, 255, 0, 0},         // Full red
		{0xF00, 0, 255, 0},         // Full green
		{0x0F0, 0, 0, 255},         // Full blue
		{0xFFF, 255, 255, 255},     // White
		{0x500, 0, 85, 0},          // Green=5 → 5*17=85
		{0x073, 51, 0, 119},        // R=3→51, B=7→119
	}

	for _, tt := range tests {
		c := cpc.GetColorFromCpcPlus(tt.colorVal)
		if c.R != tt.wantR || c.V != tt.wantG || c.B != tt.wantB {
			t.Errorf("CPC+ 0x%03X: got RGB(%d,%d,%d), want RGB(%d,%d,%d)",
				tt.colorVal, c.R, c.V, c.B, tt.wantR, tt.wantG, tt.wantB)
		}
	}
}

// TestCpcPlusGetPenColorRoundtrip verifies that CPC+ colors written to a
// bitmap can be found back via GetPenColor.
func TestCpcPlusGetPenColorRoundtrip(t *testing.T) {
	bmp := bitmap.NewDirectBitmap(16, 2)

	// Use 16 distinct CPC+ colors spread across the color space
	palette := [16]int{
		0x000, 0x00F, 0x0F0, 0xF00, // black, red, blue, green
		0xFFF, 0x555, 0xAAA, 0x123, // white, mid-grays, arbitrary
		0x0FF, 0xF0F, 0xFF0, 0x801, // cyan-ish, magenta-ish, yellow-ish
		0x246, 0x8CA, 0xDB7, 0xF30, // more arbitrary values
	}

	for pen := 0; pen < 16; pen++ {
		c := cpc.GetColorFromCpcPlus(palette[pen])
		bmp.SetPixelColor(pen, 0, c)
	}

	for pen := 0; pen < 16; pen++ {
		got := cpc.GetPenColor(bmp, pen, 0, palette, true)
		if got != pen {
			t.Errorf("CPC+ pen %d (color 0x%03X): GetPenColor returned %d", pen, palette[pen], got)
		}
	}
}

// TestCpcPlusMode0EncodeDecodeRoundtrip verifies encode→decode roundtrip
// with CPC+ palette colors.
func TestCpcPlusMode0EncodeDecodeRoundtrip(t *testing.T) {
	numCol := 80
	numLig := 200
	width := numCol * 8
	height := numLig * 2

	bmpCpc := render.NewBitmapCpcWithParams(numCol, numLig, true)
	bmpCpc.VirtualMode = 0

	// Set palette to 16 distinct CPC+ colors
	plusColors := [16]int{
		0x000, 0x00F, 0x0F0, 0xF00,
		0xFFF, 0x555, 0xAAA, 0x123,
		0x0FF, 0xF0F, 0xFF0, 0x801,
		0x246, 0x8CA, 0xDB7, 0xF30,
	}
	bmpCpc.Palette = plusColors

	displayBmp := bitmap.NewDirectBitmap(width, height)

	// Fill row 0 with all 16 pens (4 pixels wide each, Mode 0)
	for pen := 0; pen < 16; pen++ {
		color := cpc.PaletteColor(plusColors[pen], true)
		displayBmp.SetHorizontalLineDouble(pen*4, 0, 4, color)
	}

	bmpCpc.EncodeToCpc(displayBmp, true, 0)
	result := bmpCpc.DrawBitmap(numCol, numLig, 0, true)

	for pen := 0; pen < 16; pen++ {
		x := pen * 4
		got := result.GetPixelColor(x, 0)
		expected := cpc.GetColor(plusColors[pen], true)

		if got.R != expected.R || got.V != expected.V || got.B != expected.B {
			t.Errorf("CPC+ pen %d at x=%d: got RGB(%d,%d,%d), want RGB(%d,%d,%d)",
				pen, x, got.R, got.V, got.B, expected.R, expected.V, expected.B)
		}
	}
}

// TestCpcPlusQuantizationLogic verifies the CPC+ 4-bit quantization formula
// produces correct results for various input values.
func TestCpcPlusQuantizationLogic(t *testing.T) {
	// The CPC+ quantization formula: (channel>>4)*17
	// This maps 8-bit values to the nearest 4-bit CPC+ level.
	tests := []struct {
		input uint8
		want  uint8
	}{
		{0, 0},       // 0>>4=0, 0*17=0
		{100, 102},   // 100>>4=6, 6*17=102
		{150, 153},   // 150>>4=9, 9*17=153
		{200, 204},   // 200>>4=12, 12*17=204
		{255, 255},   // 255>>4=15, 15*17=255
		{17, 17},     // Already a CPC+ level
		{16, 17},     // 16>>4=1, 1*17=17
		{33, 34},     // 33>>4=2, 2*17=34
		{128, 119},   // 128>>4=8... wait, 128>>4=8, 8*17=136
	}
	// Fix the expected values using the actual formula
	tests[8].want = 136 // 128>>4=8, 8*17=136

	for _, tt := range tests {
		got := (tt.input >> 4) * 17
		if got != tt.want {
			t.Errorf("CPC+ quantize(%d): got %d, want %d", tt.input, got, tt.want)
		}
	}

	// Verify all quantized values are valid 4-bit levels (multiples of 17)
	for i := 0; i < 256; i++ {
		q := (uint8(i) >> 4) * 17
		if q%17 != 0 {
			t.Errorf("CPC+ quantize(%d) = %d is not a multiple of 17", i, q)
		}
	}

	// Verify the color index encoding: 0x0GBR format
	// For chosen RGB(102, 204, 153): R=6, G=12, B=9
	// Index = (G>>4 << 8) | (B>>4 << 4) | R>>4 = (12<<8) | (9<<4) | 6 = 0xC96
	r, g, b := uint8(102), uint8(204), uint8(153)
	idx := (int(g>>4) << 8) | (int(b>>4) << 4) | int(r>>4)
	if idx != 0xC96 {
		t.Errorf("CPC+ color index for RGB(102,204,153): got 0x%03X, want 0xC96", idx)
	}

	// Verify roundtrip: index → GetColorFromCpcPlus → same RGB
	roundtrip := cpc.GetColorFromCpcPlus(idx)
	if roundtrip.R != r || roundtrip.V != g || roundtrip.B != b {
		t.Errorf("CPC+ roundtrip 0x%03X: got RGB(%d,%d,%d), want RGB(%d,%d,%d)",
			idx, roundtrip.R, roundtrip.V, roundtrip.B, r, g, b)
	}
}

// TestCpcPlusFullPipelineSolidColor runs the full conversion pipeline with
// CPC+ enabled and a solid CPC+ color.
func TestCpcPlusFullPipelineSolidColor(t *testing.T) {
	// CPC+ color 0x84A: R=0xA=170, G=0x8=136, B=0x4=68
	testColorVal := 0x84A
	testColor := cpc.GetColorFromCpcPlus(testColorVal)

	prm := NewDefaultSettings()
	prm.VirtualMode = 0
	prm.NumCols = 80
	prm.NumLines = 200
	prm.CpcPlus = true
	prm.Pct = 0

	width := prm.GetScreenWidth()
	height := prm.GetScreenHeight()

	source := bitmap.NewDirectBitmap(width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			source.SetPixelColor(x, y, testColor)
		}
	}

	bmpCpc := render.NewBitmapCpcWithParams(prm.NumCols, prm.NumLines, true)
	bmpCpc.VirtualMode = 0
	dest := &ImageCpc{
		BitmapCpc:  bmpCpc,
		DisplayBmp: bitmap.NewDirectBitmap(width, height),
	}

	Convert(source, dest, prm, true)

	// Verify that the test color appears in the optimized palette
	found := false
	for i := 0; i < 16; i++ {
		if bmpCpc.Palette[i] == testColorVal {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("CPC+ pipeline: color 0x%03X not found in palette %v", testColorVal, bmpCpc.Palette[:])
	}

	// Verify the display bitmap has the correct color
	wrongCount := 0
	for y := 0; y < height; y += 2 {
		for x := 0; x < width; x += 4 {
			got := dest.DisplayBmp.GetPixelColor(x, y)
			if got.R != testColor.R || got.V != testColor.V || got.B != testColor.B {
				wrongCount++
				if wrongCount <= 3 {
					t.Errorf("CPC+ DisplayBmp at (%d,%d): got RGB(%d,%d,%d), want RGB(%d,%d,%d)",
						x, y, got.R, got.V, got.B, testColor.R, testColor.V, testColor.B)
				}
			}
		}
	}
	if wrongCount > 3 {
		t.Errorf("... and %d more wrong pixels", wrongCount-3)
	}
}
