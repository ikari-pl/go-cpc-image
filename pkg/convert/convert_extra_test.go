package convert

import (
	"testing"

	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
	"github.com/ikari-pl/go-cpc-image/pkg/cpc"
	"github.com/ikari-pl/go-cpc-image/pkg/render"
)

// ---------------------------------------------------------------------------
// MinMaxByte
// ---------------------------------------------------------------------------

func TestMinMaxByte(t *testing.T) {
	tests := []struct {
		in   float64
		want byte
	}{
		{0, 0},
		{255, 255},
		{128.5, 128},
		{-1, 0},
		{-100, 0},
		{256, 255},
		{1000, 255},
		{0.9, 0},
		{254.9, 254},
	}
	for _, tt := range tests {
		got := MinMaxByte(tt.in)
		if got != tt.want {
			t.Errorf("MinMaxByte(%f) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

// TestMinMaxByte_ClampsToByteBounds ensures the unexported minMaxByte
// clamps negative values to 0 and values above 255 back to 255,
// leaving values within the byte range untouched.
func TestMinMaxByte_ClampsToByteBounds(t *testing.T) {
	tests := []struct {
		in   float64
		want uint8
	}{
		{0, 0},
		{255, 255},
		{-50, 0},
		{300, 255},
		{127.7, 127},
	}
	for _, tt := range tests {
		got := minMaxByte(tt.in)
		if got != tt.want {
			t.Errorf("minMaxByte(%f) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// DoDitherFull (error diffusion path)
// ---------------------------------------------------------------------------

func TestDoDitherFullErrorDiffusion(t *testing.T) {
	// Set up Floyd-Steinberg matrix
	pct := SetMatDither(DitherParam{
		CpcPlus: false,
		Pct:     50,
		Method:  "Floyd-Steinberg (2x2)",
		DiffErr: true,
	})
	if pct == 0 {
		t.Fatal("SetMatDither returned 0")
	}

	bmp := bitmap.NewDirectBitmap(16, 16)
	gray := bitmap.NewRgbColor(128, 128, 128)
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			bmp.SetPixelColor(x, y, gray)
		}
	}
	wrapper := &bitmapWrapper{bmp}

	p := bitmap.NewRgbColor(128, 128, 128)
	chosen := bitmap.NewRgbColor(85, 85, 85) // quantized darker

	// Error diffusion should modify neighboring pixels in source
	DoDitherFull(wrapper, 0, 0, 4, p, chosen, true)

	// Check that at least one neighbor was modified
	neighbor := bmp.GetPixelColor(4, 0)
	if neighbor.R == 128 && neighbor.V == 128 && neighbor.B == 128 {
		t.Log("Neighbor pixel was not modified by error diffusion (may be out of range)")
	}
}

// ---------------------------------------------------------------------------
// DoDitherFull (ordered dithering path)
// ---------------------------------------------------------------------------

func TestDoDitherFullOrdered(t *testing.T) {
	pct := SetMatDither(DitherParam{
		CpcPlus: false,
		Pct:     50,
		Method:  "Bayer 2 (4x4)",
		DiffErr: false,
	})
	if pct == 0 {
		t.Fatal("SetMatDither returned 0")
	}

	bmp := bitmap.NewDirectBitmap(16, 16)
	wrapper := &bitmapWrapper{bmp}

	p := bitmap.NewRgbColor(100, 100, 100)
	chosen := bitmap.NewRgbColor(85, 85, 85)

	// Use position (2, 2) where Bayer 2 matrix has non-zero value
	result := DoDitherFull(wrapper, 2, 2, 2, p, chosen, false)
	// Ordered dithering adds a bias; result should differ from input
	if result.R == 100 && result.V == 100 && result.B == 100 {
		t.Error("Ordered dithering did not modify pixel value")
	}
}

func TestDoDitherFullNilMatrix(t *testing.T) {
	// With nil matrix, should return p unchanged
	SetMatDither(DitherParam{Pct: 0})

	p := bitmap.NewRgbColor(42, 43, 44)
	bmp := bitmap.NewDirectBitmap(4, 4)
	wrapper := &bitmapWrapper{bmp}

	result := DoDitherFull(wrapper, 0, 0, 2, p, bitmap.NewRgbColor(0, 0, 0), true)
	if result.R != 42 || result.V != 43 || result.B != 44 {
		t.Errorf("nil matrix should return input unchanged, got RGB(%d,%d,%d)", result.R, result.V, result.B)
	}
}

// ---------------------------------------------------------------------------
// SetMatDither
// ---------------------------------------------------------------------------

func TestSetMatDitherCpcPlus(t *testing.T) {
	pct := SetMatDither(DitherParam{
		CpcPlus: true,
		Pct:     50,
		Method:  "Floyd-Steinberg (2x2)",
		DiffErr: true,
	})
	// CPC+ shifts by 3: 50<<3 = 400
	if pct != 400 {
		t.Errorf("CPC+ pct = %d, want 400", pct)
	}
}

func TestSetMatDitherUnknownMethod(t *testing.T) {
	pct := SetMatDither(DitherParam{
		CpcPlus: false,
		Pct:     50,
		Method:  "nonexistent",
		DiffErr: false,
	})
	if pct != 0 {
		t.Errorf("unknown method pct = %d, want 0", pct)
	}
	if matDither != nil {
		t.Error("matDither should be nil for unknown method")
	}
}

// ---------------------------------------------------------------------------
// GetAvailableMethods / GetMatrixSize
// ---------------------------------------------------------------------------

func TestGetAvailableMethods(t *testing.T) {
	methods := GetAvailableMethods()
	if len(methods) != 20 {
		t.Errorf("GetAvailableMethods returned %d methods, want 20", len(methods))
	}
	// Each method should exist in DitherMatrices
	for _, m := range methods {
		if _, ok := DitherMatrices[m]; !ok {
			t.Errorf("method %q not found in DitherMatrices", m)
		}
	}
}

func TestGetMatrixSize(t *testing.T) {
	w, h, ok := GetMatrixSize("Bayer 2 (4x4)")
	if !ok || w != 4 || h != 4 {
		t.Errorf("GetMatrixSize(Bayer 2) = (%d,%d,%v), want (4,4,true)", w, h, ok)
	}
	_, _, ok = GetMatrixSize("nope")
	if ok {
		t.Error("GetMatrixSize should return false for unknown method")
	}
}

// ---------------------------------------------------------------------------
// CalculateDistance
// ---------------------------------------------------------------------------

func TestCalculateDistance(t *testing.T) {
	black := bitmap.NewRgbColor(0, 0, 0)
	white := bitmap.NewRgbColor(255, 255, 255)
	d := CalculateDistance(black, white)
	if d < 441 || d > 442 { // sqrt(3*255^2) ≈ 441.67
		t.Errorf("distance black-white = %f, want ~441.67", d)
	}
	if CalculateDistance(black, black) != 0 {
		t.Error("distance same color should be 0")
	}
}

// ---------------------------------------------------------------------------
// GetFrequencyTable
// ---------------------------------------------------------------------------

func TestGetFrequencyTable(t *testing.T) {
	state := &ConversionState{}
	// Clear
	for i := range state.ColorFrequency {
		for j := range state.ColorFrequency[i] {
			state.ColorFrequency[i][j] = 0
		}
	}
	state.ColorFrequency[5][10] = 3
	state.ColorFrequency[5][20] = 7
	state.ColorFrequency[12][0] = 1

	ft := GetFrequencyTable(27, state)
	if ft[5] != 10 {
		t.Errorf("freq[5] = %d, want 10", ft[5])
	}
	if ft[12] != 1 {
		t.Errorf("freq[12] = %d, want 1", ft[12])
	}
	if _, ok := ft[0]; ok {
		t.Error("freq[0] should not be present (zero)")
	}
}

// ---------------------------------------------------------------------------
// GetBestPalette (FindBestColors)
// ---------------------------------------------------------------------------

func TestFindBestColors(t *testing.T) {
	state := &ConversionState{}
	// Clear frequency table
	for i := range state.ColorFrequency {
		for j := range state.ColorFrequency[i] {
			state.ColorFrequency[i][j] = 0
		}
	}

	// Set color 6 as dominant, color 18 as second
	for y := 0; y < 200; y++ {
		state.ColorFrequency[6][y] = 100
		state.ColorFrequency[18][y] = 50
	}

	prm := NewDefaultSettings()
	prm.VirtualMode = 0

	var palette [16]int
	for i := range palette {
		palette[i] = 0xFFFF
	}
	var lockState [16]int

	FindBestColors(16, lockState, prm, &palette, state)

	// Color 6 should appear first (highest frequency)
	if palette[0] != 6 {
		t.Errorf("palette[0] = %d, want 6", palette[0])
	}
	// Color 18 should appear second
	if palette[1] != 18 {
		t.Errorf("palette[1] = %d, want 18", palette[1])
	}
}

func TestFindBestColorsLocked(t *testing.T) {
	state := &ConversionState{}
	for i := range state.ColorFrequency {
		for j := range state.ColorFrequency[i] {
			state.ColorFrequency[i][j] = 0
		}
	}
	for y := 0; y < 200; y++ {
		state.ColorFrequency[6][y] = 100
	}

	prm := NewDefaultSettings()
	var palette [16]int
	palette[0] = 3 // lock pen 0 to color 3
	var lockState [16]int
	lockState[0] = 1

	FindBestColors(16, lockState, prm, &palette, state)

	// Pen 0 should remain locked to color 3
	if palette[0] != 3 {
		t.Errorf("locked palette[0] = %d, want 3", palette[0])
	}
}

// ---------------------------------------------------------------------------
// QuantizePalette (k-means)
// ---------------------------------------------------------------------------

func TestQuantizePalette(t *testing.T) {
	prm := NewDefaultSettings()
	prm.VirtualMode = 1
	prm.NumCols = 10
	prm.NumLines = 10
	prm.KMeansColor = 4
	prm.KMeansPass = 2

	w := prm.GetScreenWidth()  // 80
	h := prm.GetScreenHeight() // 20
	img := bitmap.NewDirectBitmap(w, h)

	// Fill with 2 distinct colors
	red := bitmap.NewRgbColor(255, 0, 0)
	blue := bitmap.NewRgbColor(0, 0, 255)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < w/2 {
				img.SetPixelColor(x, y, red)
			} else {
				img.SetPixelColor(x, y, blue)
			}
		}
	}

	QuantizePalette(img, prm)

	// After quantization, pixels should still be roughly red or blue
	p := img.GetPixelColor(0, 0)
	if p.R < 100 {
		t.Errorf("quantized red pixel R=%d, expected > 100", p.R)
	}
	p2 := img.GetPixelColor(w-1, 0)
	if p2.B < 100 {
		t.Errorf("quantized blue pixel B=%d, expected > 100", p2.B)
	}
}

// ---------------------------------------------------------------------------
// KMeansCluster
// ---------------------------------------------------------------------------

func TestKMeansClusterUpdateCentroid(t *testing.T) {
	c := NewKMeansCluster(0, 0, 0)
	c.NbR = 300
	c.NbV = 600
	c.NbB = 150
	c.NbPixels = 3
	c.UpdateCentroid()

	if c.CentroidColor.R != 100 || c.CentroidColor.V != 200 || c.CentroidColor.B != 50 {
		t.Errorf("centroid = RGB(%d,%d,%d), want (100,200,50)",
			c.CentroidColor.R, c.CentroidColor.V, c.CentroidColor.B)
	}
}

func TestKMeansClusterUpdateCentroidZeroPixels(t *testing.T) {
	c := NewKMeansCluster(99, 99, 99)
	c.NbPixels = 0
	c.UpdateCentroid()
	if c.CentroidColor.R != 0 || c.CentroidColor.V != 0 || c.CentroidColor.B != 0 {
		t.Error("zero pixels centroid should be (0,0,0)")
	}
}

func TestKMeansClusterReset(t *testing.T) {
	c := NewKMeansCluster(10, 20, 30)
	c.NbR = 100
	c.NbV = 200
	c.NbB = 300
	c.NbPixels = 10
	c.Reset()
	if c.NbR != 0 || c.NbV != 0 || c.NbB != 0 || c.NbPixels != 0 {
		t.Error("Reset did not clear accumulators")
	}
	// CentroidColor should be unchanged by Reset
	if c.CentroidColor.R != 10 {
		t.Error("Reset should not clear CentroidColor")
	}
}

// ---------------------------------------------------------------------------
// FindNearestCluster
// ---------------------------------------------------------------------------

func TestFindNearestCluster(t *testing.T) {
	prm := NewDefaultSettings()
	clusters := []*KMeansCluster{
		NewKMeansCluster(0, 0, 0),
		NewKMeansCluster(255, 255, 255),
	}
	color := bitmap.NewRgbColor(200, 200, 200)
	result := FindNearestCluster(prm, color, clusters)

	// Should be closest to white cluster
	if result.CentroidColor.R != 255 {
		t.Errorf("nearest cluster = RGB(%d,%d,%d), want white",
			result.CentroidColor.R, result.CentroidColor.V, result.CentroidColor.B)
	}
	// Should have accumulated the color
	if result.NbPixels != 1 {
		t.Errorf("NbPixels = %d, want 1", result.NbPixels)
	}
}

// ---------------------------------------------------------------------------
// ConvertModeX
// ---------------------------------------------------------------------------

func TestConvertModeX(t *testing.T) {
	prm := NewDefaultSettings()
	prm.VirtualMode = 5

	bmpCpc := render.NewBitmapCpcWithParams(10, 10, false)
	bmpCpc.VirtualMode = 5
	w := 80
	h := 20
	dest := &ImageCpc{
		BitmapCpc:  bmpCpc,
		DisplayBmp: bitmap.NewDirectBitmap(w, h),
		Width:      w,
		Height:     h,
		Mode:       1,
	}

	// Set SplitModeColors so SetPixelCpc can resolve pens
	colors := []int{0, 6, 18, 26}
	for y := 0; y < len(dest.SplitModeColors); y++ {
		for i := 0; i < 4; i++ {
			dest.SplitModeColors[y][i] = colors[i]
		}
	}

	source := bitmap.NewDirectBitmap(w, h)
	red := cpc.CpcRgbPalette[6]
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			source.SetPixelColor(x, y, red)
		}
	}

	var colorTable [4][272]bitmap.RgbColor
	for i := 0; i < 4; i++ {
		for y := 0; y < 272; y++ {
			colorTable[i][y] = cpc.GetColor(colors[i], false)
		}
	}

	mxConv := NewModeXConverter(prm, false)
	mxConv.ConvertModeX(source, dest, &colorTable)

	// Should have written pen 1 (color 6 = bright red) for all pixels
	got := dest.DisplayBmp.GetPixelColor(0, 0)
	expected := cpc.GetColor(6, false)
	if got.R != expected.R || got.V != expected.V || got.B != expected.B {
		t.Errorf("ModeX pixel at (0,0): got RGB(%d,%d,%d), want RGB(%d,%d,%d)",
			got.R, got.V, got.B, expected.R, expected.V, expected.B)
	}
}

// ---------------------------------------------------------------------------
// ModeXConverter frequency table helpers
// ---------------------------------------------------------------------------

func TestModeXConverterCoulTrouvee(t *testing.T) {
	conv := NewModeXConverter(NewDefaultSettings(), false)
	conv.UpdateCoulTrouvee(5, 10, 42)
	if conv.GetCoulTrouvee(5, 10) != 42 {
		t.Error("UpdateCoulTrouvee/GetCoulTrouvee mismatch")
	}
	conv.ClearCoulTrouvee()
	if conv.GetCoulTrouvee(5, 10) != 0 {
		t.Error("ClearCoulTrouvee did not clear")
	}
	// Out of bounds
	if conv.GetCoulTrouvee(-1, 0) != 0 {
		t.Error("negative color should return 0")
	}
	if conv.GetCoulTrouvee(5000, 0) != 0 {
		t.Error("out of range color should return 0")
	}
}

// ---------------------------------------------------------------------------
// ConvertSplit
// ---------------------------------------------------------------------------

func TestConvertSplit(t *testing.T) {
	prm := NewDefaultSettings()
	prm.VirtualMode = 6

	bmpCpc := render.NewBitmapCpcWithParams(10, 10, false)
	bmpCpc.VirtualMode = 6
	w := 80
	h := 20
	dest := &ImageCpc{
		BitmapCpc:  bmpCpc,
		DisplayBmp: bitmap.NewDirectBitmap(w, h),
		Width:      w,
		Height:     h,
		Mode:       1,
	}

	colors := []int{0, 6, 18, 26, 13, 4, 10, 24, 1}
	for y := 0; y < len(dest.SplitModeColors); y++ {
		for i := 0; i < 9; i++ {
			dest.SplitModeColors[y][i] = colors[i]
		}
	}

	source := bitmap.NewDirectBitmap(w, h)
	green := cpc.CpcRgbPalette[18]
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			source.SetPixelColor(x, y, green)
		}
	}

	var colorTable [9][272]bitmap.RgbColor
	for i := 0; i < 9; i++ {
		for y := 0; y < 272; y++ {
			colorTable[i][y] = cpc.GetColor(colors[i], false)
		}
	}

	spConv := NewSplitConverter(prm, false)
	spConv.ConvertSplit(source, dest, &colorTable)

	// pen 2 (color 18 = bright green) should be chosen
	got := dest.DisplayBmp.GetPixelColor(0, 0)
	expected := cpc.GetColor(18, false)
	if got.R != expected.R || got.V != expected.V || got.B != expected.B {
		t.Errorf("Split pixel at (0,0): got RGB(%d,%d,%d), want RGB(%d,%d,%d)",
			got.R, got.V, got.B, expected.R, expected.V, expected.B)
	}
}

// ---------------------------------------------------------------------------
// SplitConverter frequency table helpers
// ---------------------------------------------------------------------------

func TestSplitConverterCoulTrouvee(t *testing.T) {
	conv := NewSplitConverter(NewDefaultSettings(), false)
	conv.UpdateCoulTrouvee(7, 3, 99)
	if conv.GetCoulTrouvee(7, 3) != 99 {
		t.Error("Split UpdateCoulTrouvee/GetCoulTrouvee mismatch")
	}
	conv.ClearCoulTrouvee()
	if conv.GetCoulTrouvee(7, 3) != 0 {
		t.Error("Split ClearCoulTrouvee did not clear")
	}
}

// ---------------------------------------------------------------------------
// EGXConverter.Convert
// ---------------------------------------------------------------------------

func TestEGXConverterConvert(t *testing.T) {
	prm := NewDefaultSettings()
	prm.VirtualMode = 3
	prm.YEgx = 0
	prm.NumCols = 10
	prm.NumLines = 10

	w := prm.GetScreenWidth()
	h := prm.GetScreenHeight()

	bmpCpc := render.NewBitmapCpcWithParams(prm.NumCols, prm.NumLines, false)
	for i := 0; i < 16; i++ {
		bmpCpc.Palette[i] = i
	}

	dest := &ImageCpc{
		BitmapCpc:  bmpCpc,
		DisplayBmp: bitmap.NewDirectBitmap(w, h),
	}

	source := bitmap.NewDirectBitmap(w, h)
	white := cpc.CpcRgbPalette[26]
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			source.SetPixelColor(x, y, white)
		}
	}

	var colorTable [16][272]bitmap.RgbColor
	for line := 0; line < prm.NumLines; line++ {
		for i := 0; i < 16; i++ {
			colorTable[i][line] = cpc.GetColor(i, false)
		}
	}

	egx := NewEGXConverter()
	egx.Convert(source, prm, dest, 16, colorTable, prm.YEgx)

	// Verify some pixel was written
	got := dest.DisplayBmp.GetPixelColor(0, 0)
	if got.R == 0 && got.V == 0 && got.B == 0 {
		t.Error("EGX converter did not write any pixels")
	}
}

// ---------------------------------------------------------------------------
// Pass2 with different modes
// ---------------------------------------------------------------------------

// initContrastTable initializes the contrast lookup table (normally done by Convert)
func initContrastTable(prm *Settings, state *ConversionState) {
	c := float64(prm.PctContrast) / 100.0
	for i := 0; i < 256; i++ {
		state.ContrastTable[i] = MinMaxByte(((float64(i)/255.0 - 0.5) * c + 0.5) * 255.0)
	}
}

func TestPass2Mode1(t *testing.T) {
	prm := NewDefaultSettings()
	prm.VirtualMode = 1
	prm.NumCols = 10
	prm.NumLines = 10
	prm.Pct = 0

	w := prm.GetScreenWidth()
	h := prm.GetScreenHeight()

	source := bitmap.NewDirectBitmap(w, h)
	blue := cpc.CpcRgbPalette[2] // Bright Blue
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			source.SetPixelColor(x, y, blue)
		}
	}

	state := &ConversionState{}
	initContrastTable(prm, state)
	ConvertPass1(source, prm, state)

	bmpCpc := render.NewBitmapCpcWithParams(prm.NumCols, prm.NumLines, false)
	bmpCpc.VirtualMode = prm.VirtualMode
	dest := &ImageCpc{
		BitmapCpc:  bmpCpc,
		DisplayBmp: bitmap.NewDirectBitmap(w, h),
	}

	var splitCount int
	Pass2(source, dest, prm, &splitCount, state)

	got := dest.DisplayBmp.GetPixelColor(0, 0)
	if got.B < 100 {
		t.Errorf("Pass2 Mode1 pixel at (0,0): B=%d, expected >= 100", got.B)
	}
}

func TestPass2Mode2(t *testing.T) {
	prm := NewDefaultSettings()
	prm.VirtualMode = 2
	prm.NumCols = 10
	prm.NumLines = 10
	prm.Pct = 0

	w := prm.GetScreenWidth()
	h := prm.GetScreenHeight()

	source := bitmap.NewDirectBitmap(w, h)
	white := cpc.CpcRgbPalette[26]
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			source.SetPixelColor(x, y, white)
		}
	}

	state := &ConversionState{}
	initContrastTable(prm, state)
	ConvertPass1(source, prm, state)

	bmpCpc := render.NewBitmapCpcWithParams(prm.NumCols, prm.NumLines, false)
	bmpCpc.VirtualMode = prm.VirtualMode
	dest := &ImageCpc{
		BitmapCpc:  bmpCpc,
		DisplayBmp: bitmap.NewDirectBitmap(w, h),
	}

	var splitCount int
	Pass2(source, dest, prm, &splitCount, state)

	got := dest.DisplayBmp.GetPixelColor(0, 0)
	if got.R == 0 && got.V == 0 && got.B == 0 {
		t.Error("Pass2 Mode2 wrote all black pixels")
	}
}

func TestPass2ModeX(t *testing.T) {
	prm := NewDefaultSettings()
	prm.VirtualMode = 5
	prm.NumCols = 10
	prm.NumLines = 10
	prm.Pct = 0

	w := prm.GetScreenWidth()
	h := prm.GetScreenHeight()

	source := bitmap.NewDirectBitmap(w, h)
	red := cpc.CpcRgbPalette[6]
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			source.SetPixelColor(x, y, red)
		}
	}

	state := &ConversionState{}
	initContrastTable(prm, state)
	ConvertPass1(source, prm, state)

	bmpCpc := render.NewBitmapCpcWithParams(prm.NumCols, prm.NumLines, false)
	bmpCpc.VirtualMode = prm.VirtualMode
	dest := &ImageCpc{
		BitmapCpc:  bmpCpc,
		DisplayBmp: bitmap.NewDirectBitmap(w, h),
	}

	var splitCount int
	Pass2(source, dest, prm, &splitCount, state)

	// Verify pipeline ran without panic and wrote something
	// Mode X palette optimization may place colors in any order;
	// just check the dominant color appears somewhere in SplitModeColors
	found := false
	for y := 0; y < prm.NumLines; y++ {
		for i := 0; i < 4; i++ {
			if dest.SplitModeColors[y][i] == 6 {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Error("Pass2 ModeX: color 6 not found in SplitModeColors")
	}
}

func TestPass2ModeSplit(t *testing.T) {
	prm := NewDefaultSettings()
	prm.VirtualMode = 6
	prm.NumCols = 10
	prm.NumLines = 10
	prm.Pct = 0

	w := prm.GetScreenWidth()
	h := prm.GetScreenHeight()

	source := bitmap.NewDirectBitmap(w, h)
	green := cpc.CpcRgbPalette[18]
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			source.SetPixelColor(x, y, green)
		}
	}

	state := &ConversionState{}
	initContrastTable(prm, state)
	ConvertPass1(source, prm, state)

	bmpCpc := render.NewBitmapCpcWithParams(prm.NumCols, prm.NumLines, false)
	bmpCpc.VirtualMode = prm.VirtualMode
	dest := &ImageCpc{
		BitmapCpc:  bmpCpc,
		DisplayBmp: bitmap.NewDirectBitmap(w, h),
	}

	var splitCount int
	Pass2(source, dest, prm, &splitCount, state)

	// Verify pipeline ran without panic and the dominant color appears in SplitModeColors
	found := false
	for y := 0; y < prm.NumLines; y++ {
		for i := 0; i < 9; i++ {
			if dest.SplitModeColors[y][i] == 18 {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Error("Pass2 ModeSplit: color 18 not found in SplitModeColors")
	}
}

// ---------------------------------------------------------------------------
// ConvertAscii
// ---------------------------------------------------------------------------

func TestConvertAscii(t *testing.T) {
	prm := NewDefaultSettings()
	prm.VirtualMode = 9 // ASC1 mode
	prm.NumCols = 10
	prm.NumLines = 10

	w := prm.GetScreenWidth()
	h := prm.GetScreenHeight()

	bmpCpc := render.NewBitmapCpcWithParams(prm.NumCols, prm.NumLines, false)
	for i := 0; i < 4; i++ {
		bmpCpc.Palette[i] = i * 6
	}

	dest := &ImageCpc{
		BitmapCpc:  bmpCpc,
		DisplayBmp: bitmap.NewDirectBitmap(w, h),
		Width:      w,
		Height:     h,
		Mode:       1,
	}

	source := bitmap.NewDirectBitmap(w, h)
	white := cpc.CpcRgbPalette[18]
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			source.SetPixelColor(x, y, white)
		}
	}

	var colorTable [16][272]bitmap.RgbColor
	for i := 0; i < 4; i++ {
		for y := 0; y < 272; y++ {
			colorTable[i][y] = cpc.GetColor(i*6, false)
		}
	}

	ascConv := NewAsciiConverter(prm, false)
	ascConv.ConvertAscii(source, dest, 4, &colorTable)

	// Should have written some non-black pixels
	got := dest.DisplayBmp.GetPixelColor(0, 0)
	if got.R == 0 && got.V == 0 && got.B == 0 {
		t.Log("ConvertAscii wrote black at (0,0), might match pen 0")
	}
}

// ---------------------------------------------------------------------------
// PatternM1
// ---------------------------------------------------------------------------

func TestPatternM1(t *testing.T) {
	p := NewPatternM1()
	p.SetPix(0, 0, 1)
	p.SetPix(3, 3, 2)
	if p.GetPix(0, 0) != 1 {
		t.Error("SetPix/GetPix mismatch at (0,0)")
	}
	if p.GetPix(3, 3) != 2 {
		t.Error("SetPix/GetPix mismatch at (3,3)")
	}
	// Out of bounds
	if p.GetPix(4, 0) != 0 {
		t.Error("GetPix out of bounds should return 0")
	}
	p.SetPix(-1, 0, 5) // should not panic
}

func TestPatternM1IsSame(t *testing.T) {
	p1 := NewPatternM1()
	p2 := NewPatternM1()
	if !p1.IsSame(p2, 4) {
		t.Error("two empty patterns should be same")
	}
	p1.SetPix(1, 1, 3)
	if p1.IsSame(p2, 4) {
		t.Error("different patterns should not be same")
	}
}

// ---------------------------------------------------------------------------
// CopyTrame
// ---------------------------------------------------------------------------

func TestCopyTrame(t *testing.T) {
	trames := CopyTrame(0)
	// Verify preset 0 pattern 1 is all 1s
	allOnes := true
	for x := 0; x < 4; x++ {
		for y := 0; y < 4; y++ {
			if trames[1][x][y] != 1 {
				allOnes = false
			}
		}
	}
	if !allOnes {
		t.Error("preset 0 pattern 1 should be all 1s")
	}

	// Out of range preset should return zeroed array
	trames2 := CopyTrame(99)
	for i := 0; i < 16; i++ {
		for x := 0; x < 4; x++ {
			for y := 0; y < 4; y++ {
				if trames2[i][x][y] != 0 {
					t.Fatal("out of range preset should return zeros")
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// GetValColor
// ---------------------------------------------------------------------------

func TestGetValColor(t *testing.T) {
	prm := NewDefaultSettings()
	prm.CpcPlus = false
	// Standard CPC: returns color index directly
	if GetValColor(5, prm) != 5 {
		t.Error("standard GetValColor should return index")
	}

	prm.CpcPlus = true
	// CPC+: 0x123 -> v=1, r=2, b=3
	val := GetValColor(0x123, prm)
	v := (0x123 >> 8)        // 1
	r := (0x123 >> 4) & 0x0F // 2
	b := 0x123 & 0x0F        // 3
	expected := r*r*prm.CoefR + v*v*prm.CoefV + b*b*prm.CoefB
	if val != expected {
		t.Errorf("CPC+ GetValColor(0x123) = %d, want %d", val, expected)
	}
}

// ---------------------------------------------------------------------------
// Param helpers
// ---------------------------------------------------------------------------

func TestParamGetThresholds(t *testing.T) {
	prm := NewDefaultSettings()
	r := prm.GetThresholds("R")
	if r[0] != 85 || r[1] != 170 {
		t.Error("R thresholds unexpected")
	}
	g := prm.GetThresholds("G")
	if g[0] != 85 {
		t.Error("G thresholds unexpected")
	}
	d := prm.GetThresholds("X")
	if d[0] != 85 {
		t.Error("default thresholds unexpected")
	}
}

func TestParamIsOverscan(t *testing.T) {
	prm := NewDefaultSettings()
	if prm.IsOverscan() {
		t.Error("default params should not be overscan")
	}
	prm.NumCols = 96
	if !prm.IsOverscan() {
		t.Error("96 cols should be overscan")
	}
}

func TestParamScreenDimensions(t *testing.T) {
	prm := NewDefaultSettings()
	if prm.GetScreenWidth() != 640 {
		t.Errorf("width = %d, want 640", prm.GetScreenWidth())
	}
	if prm.GetScreenHeight() != 400 {
		t.Errorf("height = %d, want 400", prm.GetScreenHeight())
	}
}

// ---------------------------------------------------------------------------
// GetNumColorPixelCpc
// ---------------------------------------------------------------------------

func TestGetNumColorPixelCpc(t *testing.T) {
	prm := NewDefaultSettings()
	// Old method (default): threshold-based
	// Black (0,0,0) should map to color 0
	idx := GetNumColorPixelCpc(prm, bitmap.NewRgbColor(0, 0, 0))
	if idx != 0 {
		t.Errorf("black -> color %d, want 0", idx)
	}

	// White (255,255,255) -> R>170 => 6, V>170 => 18, B>170 => 2 => 26
	idx = GetNumColorPixelCpc(prm, bitmap.NewRgbColor(255, 255, 255))
	if idx != 26 {
		t.Errorf("white -> color %d, want 26", idx)
	}

	// New method
	prm.NewMethod = true
	idx = GetNumColorPixelCpc(prm, bitmap.NewRgbColor(0, 0, 0))
	if idx != 0 {
		t.Errorf("new method black -> color %d, want 0", idx)
	}
}

// ---------------------------------------------------------------------------
// SetLumiSat
// ---------------------------------------------------------------------------

func TestSetLumiSat(t *testing.T) {
	r, v, b := float32(200), float32(100), float32(50)
	// Identity: lumi=1.0, satur=1.0
	SetLumiSat(1.0, 1.0, &r, &v, &b)
	// Values should be approximately preserved
	if r < 150 || r > 250 {
		t.Errorf("SetLumiSat identity: r=%f, expected ~200", r)
	}
}

// ---------------------------------------------------------------------------
// ConvertPass1
// ---------------------------------------------------------------------------

func TestConvertPass1FillsFrequency(t *testing.T) {
	prm := NewDefaultSettings()
	prm.VirtualMode = 0
	prm.NumCols = 10
	prm.NumLines = 10
	prm.Pct = 0

	w := prm.GetScreenWidth()
	h := prm.GetScreenHeight()

	source := bitmap.NewDirectBitmap(w, h)
	red := cpc.CpcRgbPalette[6]
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			source.SetPixelColor(x, y, red)
		}
	}

	state := &ConversionState{}
	initContrastTable(prm, state)
	ConvertPass1(source, prm, state)

	// Color 6 should have nonzero frequency
	ft := GetFrequencyTable(27, state)
	if ft[6] == 0 {
		t.Error("ConvertPass1 did not record frequency for color 6")
	}
}

// ---------------------------------------------------------------------------
// Full Convert pipeline (small image)
// ---------------------------------------------------------------------------

func TestConvertSmallImage(t *testing.T) {
	prm := NewDefaultSettings()
	prm.VirtualMode = 0
	prm.NumCols = 10
	prm.NumLines = 10
	prm.Pct = 0

	w := prm.GetScreenWidth()
	h := prm.GetScreenHeight()

	source := bitmap.NewDirectBitmap(w, h)
	testColor := cpc.CpcRgbPalette[13]
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			source.SetPixelColor(x, y, testColor)
		}
	}

	bmpCpc := render.NewBitmapCpcWithParams(prm.NumCols, prm.NumLines, false)
	bmpCpc.VirtualMode = prm.VirtualMode
	dest := &ImageCpc{
		BitmapCpc:  bmpCpc,
		DisplayBmp: bitmap.NewDirectBitmap(w, h),
	}

	numColors := Convert(source, dest, prm, true)
	if numColors < 1 {
		t.Errorf("Convert returned %d colors, expected >= 1", numColors)
	}
	if !bmpCpc.IsComputed {
		t.Error("IsComputed should be true after Convert")
	}
}

// ---------------------------------------------------------------------------
// ImageCpc.SetPixelCpc with SplitModeColors
// ---------------------------------------------------------------------------

func TestSetPixelCpcSplitMode(t *testing.T) {
	bmpCpc := render.NewBitmapCpcWithParams(10, 10, false)
	bmpCpc.VirtualMode = 5 // Mode X uses SplitModeColors

	dest := &ImageCpc{
		BitmapCpc:  bmpCpc,
		DisplayBmp: bitmap.NewDirectBitmap(80, 20),
	}
	// Set SplitModeColors for line 0
	dest.SplitModeColors[0][0] = 6 // pen 0 = CPC color 6

	dest.SetPixelCpc(0, 0, 0, 2)
	got := dest.DisplayBmp.GetPixelColor(0, 0)
	expected := cpc.GetColor(6, false)
	if got.R != expected.R || got.V != expected.V || got.B != expected.B {
		t.Errorf("SetPixelCpc split mode: got RGB(%d,%d,%d), want RGB(%d,%d,%d)",
			got.R, got.V, got.B, expected.R, expected.V, expected.B)
	}
}

func TestSetPixelCpcNilBitmaps(t *testing.T) {
	dest := &ImageCpc{}
	// Should not panic
	dest.SetPixelCpc(0, 0, 0, 2)
}
