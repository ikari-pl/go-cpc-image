// Diagnostic tool to test conversion pipeline with real images.
// Tests mode 0 (16 colors), analyses per-line pen usage, detects anomalies.
package main

import (
	"fmt"
	"image/png"
	"os"

	"github.com/ikari/go-cpc-image/pkg/bitmap"
	"github.com/ikari/go-cpc-image/pkg/convert"
	"github.com/ikari/go-cpc-image/pkg/cpc"
	"github.com/ikari/go-cpc-image/pkg/render"
)

const outDir = "/tmp/convimgcpc-debug"

func main() {
	os.MkdirAll(outDir, 0755)

	// Use a real photo if available, else generate a gradient
	testImages := []string{
		"/Users/ikari/Desktop/worek/Pixel Heaven 2022/2-20220611_122038.jpg",
		"/Users/ikari/Sync/wp12203476.jpg",
		"/Users/ikari/src/cpc/gem-knight/art/2026-02-01-01-36-splash-gemknight-notext.png",
	}

	var src *bitmap.DirectBitmap
	var srcName string
	for _, path := range testImages {
		if _, err := os.Stat(path); err == nil {
			img, err := bitmap.NewDirectBitmapFromFile(path)
			if err == nil {
				src = img
				srcName = path
				break
			}
		}
	}

	if src == nil {
		fmt.Println("No real photo found, generating gradient test image")
		srcName = "gradient"
		src = generateGradient(320, 200)
	}

	fmt.Printf("Source: %s (%dx%d)\n\n", srcName, src.Width(), src.Height())

	// Test mode 0 with real photo
	fmt.Println("========== REAL PHOTO MODE 0 ==========")
	testMode(src, 0, "photo")
	fmt.Println()

	// Load gem-knight image if available
	gemPath := "/Users/ikari/src/cpc/gem-knight/art/2026-02-01-01-36-splash-gemknight-notext.png"
	if gem, err := bitmap.NewDirectBitmapFromFile(gemPath); err == nil {
		fmt.Println("========== GEM KNIGHT MODE 0 ==========")
		testMode(gem, 0, "gemknight")
		fmt.Println()
	}

	// Test mode 0 with rainbow (guarantees all 27 CPC colors)
	fmt.Println("========== RAINBOW MODE 0 ==========")
	rainbow := generateRainbow(640, 400)
	testMode(rainbow, 0, "rainbow")
	fmt.Println()

	// Test mode 1 with real photo
	fmt.Println("========== REAL PHOTO MODE 1 ==========")
	testMode(src, 1, "photo")
	fmt.Println()
}

func testMode(src *bitmap.DirectBitmap, mode int, tag string) {
	params := convert.NewDefaultParam()
	params.VirtualMode = mode
	params.NumCols = 80
	params.NumLines = 200
	params.Pct = 50
	params.Method = "Floyd-Steinberg (2x2)"
	params.NewMethod = true

	cpcW := params.GetScreenWidth()
	cpcH := params.GetScreenHeight()

	// Resize source to CPC dimensions
	resized := bitmap.NewDirectBitmap(cpcW, cpcH)
	for dy := 0; dy < cpcH; dy++ {
		srcY := dy * src.Height() / cpcH
		for dx := 0; dx < cpcW; dx++ {
			srcX := dx * src.Width() / cpcW
			resized.SetPixelColor(dx, dy, src.GetPixelColor(srcX, srcY))
		}
	}

	// Create CPC image
	bmpCpc := render.NewBitmapCpcWithParams(80, 200, false)
	imgCpc := &convert.ImageCpc{
		BitmapCpc:  bmpCpc,
		DisplayBmp: bitmap.NewDirectBitmap(cpcW, cpcH),
		Width:      cpcW,
		Height:     cpcH,
		Mode:       mode,
	}

	// Run Pass1 separately to inspect frequency table before FindBestColors clears it
	convert.ConvertPass1(resized, params)
	freqTable := convert.GetFrequencyTable(27)
	fmt.Printf("  After Pass1 — frequency table (%d distinct CPC colors):\n", len(freqTable))
	for idx := 0; idx < 27; idx++ {
		if count, ok := freqTable[idx]; ok {
			rgb := cpc.CpcRgbPalette[idx]
			fmt.Printf("    CPC#%2d RGB(%3d,%3d,%3d): %d pixels\n", idx, rgb.R, rgb.V, rgb.B, count)
		}
	}

	// Now run full conversion (Pass1 runs again inside Convert, that's fine)
	// Re-create source since Pass1 modifies it
	resized2 := bitmap.NewDirectBitmap(cpcW, cpcH)
	for dy := 0; dy < cpcH; dy++ {
		srcY := dy * src.Height() / cpcH
		for dx := 0; dx < cpcW; dx++ {
			srcX := dx * src.Width() / cpcW
			resized2.SetPixelColor(dx, dy, src.GetPixelColor(srcX, srcY))
		}
	}

	numColors := convert.Convert(resized2, imgCpc, params, true)
	fmt.Printf("  Convert returned: %d unique colors\n", numColors)

	// Print palette
	maxPen := 1 << (4 >> mode) // mode0=16, mode1=4, mode2=2
	fmt.Printf("  Max pens for mode %d: %d\n", mode, maxPen)
	fmt.Printf("  Palette:\n")
	for i := 0; i < maxPen; i++ {
		palIdx := bmpCpc.Palette[i]
		if palIdx >= 27 {
			fmt.Printf("    Pen %2d: INVALID (0x%04X)\n", i, palIdx)
		} else {
			rgb := cpc.CpcRgbPalette[palIdx]
			fmt.Printf("    Pen %2d: CPC#%2d -> RGB(%3d,%3d,%3d)\n", i, palIdx, rgb.R, rgb.V, rgb.B)
		}
	}

	// Decode CPC buffer and count pen usage per line
	usedGlobal := [16]int{} // count of pixels per pen
	lineAnomaly := []string{}

	for y := 0; y < bmpCpc.NumLig<<1; y += 2 {
		usedLine := [16]bool{}
		cpcAddr := cpc.CpcAddress(y, bmpCpc.NumCol, bmpCpc.NumLig)

		for x := 0; x < bmpCpc.NumCol; x++ {
			addr := cpcAddr + x
			if addr >= len(bmpCpc.ScreenData) {
				continue
			}
			pixByte := bmpCpc.ScreenData[addr]

			var pens []int
			switch mode {
			case 0:
				p0 := int(pixByte>>7) + int((pixByte&0x20)>>3) + int((pixByte&0x08)>>2) + int((pixByte&0x02)<<2)
				p1 := int((pixByte&0x40)>>6) + int((pixByte&0x10)>>2) + int((pixByte&0x04)>>1) + int((pixByte&0x01)<<3)
				pens = []int{p0, p1}
			case 1:
				p0 := int((pixByte>>7)&1) + int((pixByte>>2)&2)
				p1 := int((pixByte>>6)&1) + int((pixByte>>1)&2)
				p2 := int((pixByte>>5)&1) + int((pixByte>>0)&2)
				p3 := int((pixByte>>4)&1) + int((pixByte<<1)&2)
				pens = []int{p0, p1, p2, p3}
			case 2:
				for i := 7; i >= 0; i-- {
					pens = append(pens, int(pixByte>>uint(i))&1)
				}
			}

			for _, p := range pens {
				if p < 16 {
					usedLine[p] = true
					usedGlobal[p]++
				}
			}
		}

		// Check if line 0 or early lines only use pen 0 (the "fade to black" issue)
		lineNum := y >> 1
		lineUsedCount := 0
		for _, u := range usedLine {
			if u {
				lineUsedCount++
			}
		}
		if lineNum < 5 && lineUsedCount <= 1 {
			lineAnomaly = append(lineAnomaly, fmt.Sprintf("    LINE %d: only %d pen(s) used (possible fade-to-black)", lineNum, lineUsedCount))
		}
	}

	// Report
	totalPensUsed := 0
	for i := 0; i < maxPen; i++ {
		if usedGlobal[i] > 0 {
			totalPensUsed++
		}
	}

	fmt.Printf("  Pens actually used in output: %d / %d\n", totalPensUsed, maxPen)
	for i := 0; i < maxPen; i++ {
		marker := " "
		if usedGlobal[i] == 0 {
			marker = "!"
		}
		fmt.Printf("   %s Pen %2d: %6d pixels\n", marker, i, usedGlobal[i])
	}

	if len(lineAnomaly) > 0 {
		fmt.Println("  ANOMALIES:")
		for _, a := range lineAnomaly {
			fmt.Println(a)
		}
	} else {
		fmt.Println("  No line anomalies detected in first 5 lines.")
	}

	// Check for all-zero first line in CPC buffer
	baseAddr := cpc.CpcAddress(0, bmpCpc.NumCol, bmpCpc.NumLig)
	allZero := true
	for x := 0; x < bmpCpc.NumCol; x++ {
		if bmpCpc.ScreenData[baseAddr+x] != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		fmt.Println("  WARNING: First CPC line is all-zero bytes (pen 0 everywhere)")
		// Dump what the source resized image had at y=0
		fmt.Printf("  Source pixels at y=0 (first 10): ")
		for x := 0; x < 10 && x < resized.Width(); x++ {
			p := resized.GetPixelColor(x, 0)
			fmt.Printf("(%d,%d,%d) ", p.R, p.V, p.B)
		}
		fmt.Println()
		// Dump what DisplayBmp has at y=0
		if imgCpc.DisplayBmp != nil {
			fmt.Printf("  DisplayBmp at y=0 (first 10): ")
			for x := 0; x < 10 && x < imgCpc.DisplayBmp.Width(); x++ {
				p := imgCpc.DisplayBmp.GetPixelColor(x, 0)
				fmt.Printf("(%d,%d,%d) ", p.R, p.V, p.B)
			}
			fmt.Println()
		}
		// Check if source was modified by dithering (Pass1 mutates source)
		fmt.Printf("  Post-Pass1 source at y=0 (first 10): ")
		for x := 0; x < 10 && x < resized.Width(); x++ {
			p := resized.GetPixelColor(x, 0)
			fmt.Printf("(%d,%d,%d) ", p.R, p.V, p.B)
		}
		fmt.Println()
	}

	// Save output
	result := bmpCpc.RenderToRGBA()
	fname := fmt.Sprintf("%s/%s_mode%d_output.png", outDir, tag, mode)
	f, _ := os.Create(fname)
	png.Encode(f, result)
	f.Close()
	fmt.Printf("  Saved: %s\n", fname)

	// Also save the DisplayBmp (intermediate before CPC encode)
	if imgCpc.DisplayBmp != nil {
		fname2 := fmt.Sprintf("%s/%s_mode%d_displaybmp.png", outDir, tag, mode)
		f2, _ := os.Create(fname2)
		png.Encode(f2, imgCpc.DisplayBmp.Image())
		f2.Close()
		fmt.Printf("  Saved: %s\n", fname2)
	}

	// Verify: compare DisplayBmp vs RenderToRGBA for first few lines
	if imgCpc.DisplayBmp != nil {
		mismatches := 0
		for y := 0; y < 10 && y < imgCpc.DisplayBmp.Height(); y += 2 {
			for x := 0; x < imgCpc.DisplayBmp.Width(); x++ {
				dpx := imgCpc.DisplayBmp.GetPixelColor(x, y)
				// Get the same pixel from RenderToRGBA result
				r, g, b, _ := result.At(x, y).RGBA()
				rr, gg, bb := uint8(r>>8), uint8(g>>8), uint8(b>>8)
				if dpx.R != rr || dpx.V != gg || dpx.B != bb {
					if mismatches < 5 {
						fmt.Printf("  MISMATCH at (%d,%d): DisplayBmp=(%d,%d,%d) Render=(%d,%d,%d)\n",
							x, y, dpx.R, dpx.V, dpx.B, rr, gg, bb)
					}
					mismatches++
				}
			}
		}
		if mismatches > 0 {
			fmt.Printf("  Total mismatches in first 5 lines: %d\n", mismatches)
		} else {
			fmt.Println("  DisplayBmp and RenderToRGBA match for first 5 lines.")
		}
	}
}

func generateGradient(w, h int) *bitmap.DirectBitmap {
	src := bitmap.NewDirectBitmap(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r := uint8((x * 255) / w)
			g := uint8((y * 255) / h)
			b := uint8(((x + y) * 255) / (w + h))
			src.SetPixelColor(x, y, bitmap.NewRgbColor(r, g, b))
		}
	}
	return src
}

// generateRainbow creates a test image with all 27 CPC colors represented as large blocks,
// plus smooth gradients. This guarantees maximum color diversity for mode 0 testing.
func generateRainbow(w, h int) *bitmap.DirectBitmap {
	src := bitmap.NewDirectBitmap(w, h)
	rgbCpc := cpc.GetCpcRgb()

	// Top half: 27 CPC color blocks in a 9x3 grid
	blockW := w / 9
	blockH := (h / 2) / 3
	for i := 0; i < 27 && i < len(rgbCpc); i++ {
		col := i % 9
		row := i / 9
		c := rgbCpc[i]
		for y := row * blockH; y < (row+1)*blockH; y++ {
			for x := col * blockW; x < (col+1)*blockW; x++ {
				src.SetPixelColor(x, y, c)
			}
		}
	}

	// Bottom half: RGB gradients for smooth transitions
	for y := h / 2; y < h; y++ {
		for x := 0; x < w; x++ {
			section := (y - h/2) * 3 / (h / 2)
			t := uint8(x * 255 / w)
			switch section {
			case 0: // Red gradient
				src.SetPixelColor(x, y, bitmap.NewRgbColor(t, 0, uint8(255)-t))
			case 1: // Green gradient
				src.SetPixelColor(x, y, bitmap.NewRgbColor(0, t, uint8(255)-t))
			default: // Blue gradient
				src.SetPixelColor(x, y, bitmap.NewRgbColor(uint8(255)-t, t/2, t))
			}
		}
	}

	return src
}

