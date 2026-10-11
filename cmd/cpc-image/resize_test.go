package main

import (
	"testing"

	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
	"github.com/ikari-pl/go-cpc-image/pkg/convert"
	"github.com/ikari-pl/go-cpc-image/pkg/cpc"
)

// resizeScreen returns the settings the resize tests map into. The mapping only
// needs a grid, so a standard mode-1 screen (320x200) does: what is under test is
// where the source pixels land, not the palette.
func resizeScreen() *convert.Settings {
	prm := convert.NewDefaultSettings()
	prm.VirtualMode = 1
	prm.NumCols = cpc.StandardCols
	prm.NumLines = cpc.StandardLines
	return prm
}

var (
	markerTopRow  = bitmap.NewRgbColor(255, 0, 0) // the source's first and last row
	markerLeftCol = bitmap.NewRgbColor(0, 255, 0) // the source's first and last column
)

// markerSource builds a w x h picture whose interior encodes its own coordinates
// — (x, y, 200) — and whose first and last rows and columns carry marker colours.
// A marker that is missing from the output can only have been cropped, and the
// interior lets a sample be traced back to the pixel it came from.
func markerSource(w, h int) *bitmap.DirectBitmap {
	src := bitmap.NewDirectBitmap(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			src.SetPixelColor(x, y, bitmap.NewRgbColor(uint8(x), uint8(y), 200))
		}
	}
	for x := 0; x < w; x++ {
		src.SetPixelColor(x, 0, markerTopRow)
		src.SetPixelColor(x, h-1, markerTopRow)
	}
	for y := 0; y < h; y++ {
		src.SetPixelColor(0, y, markerLeftCol)
		src.SetPixelColor(w-1, y, markerLeftCol)
	}
	return src
}

// hasColour reports whether the picture contains a colour, used to see at a
// glance whether an edge of the source survived the mapping.
func hasColour(img *bitmap.DirectBitmap, c bitmap.RgbColor) bool {
	for y := 0; y < img.Height(); y++ {
		for x := 0; x < img.Width(); x++ {
			if img.GetPixelColor(x, y) == c {
				return true
			}
		}
	}
	return false
}

// TestResizePolicyFitStretches pins the default policy, which is what the tool
// has always done: both axes fill the screen, so the corners of the source are
// the corners of the screen and nothing keeps the background colour.
func TestResizePolicyFitStretches(t *testing.T) {
	prm := resizeScreen()
	src := markerSource(40, 30)

	out := resizeToCanvas(src, prm, "fit")

	bg := cpc.GetColor(prm.Palette[0], prm.CpcPlus)
	if got, want := out.Width(), prm.GetScreenWidth(); got != want {
		t.Fatalf("output width = %d, want %d", got, want)
	}
	if got := out.GetPixelColor(0, 0); got != src.GetPixelColor(0, 0) {
		t.Errorf("top-left = %v, want the source's top-left %v", got, src.GetPixelColor(0, 0))
	}
	if got, want := out.GetPixelColor(out.Width()-1, out.Height()-1), src.GetPixelColor(39, 29); got != want {
		t.Errorf("bottom-right = %v, want the source's bottom-right %v", got, want)
	}
	if hasColour(out, bg) {
		t.Error("a stretch must cover the whole screen, but the background colour survived")
	}
	if !hasColour(out, markerTopRow) || !hasColour(out, markerLeftCol) {
		t.Error("a stretch keeps every edge of the source, markers are missing")
	}
}

// TestResizePolicyKeepSmallerLetterboxes pins the aspect-preserving fit: the
// picture keeps its shape, is centred, and the pen-0 colour fills the band it
// leaves free.
func TestResizePolicyKeepSmallerLetterboxes(t *testing.T) {
	prm := resizeScreen()
	src := markerSource(100, 100) // square on a 320x200 screen: pillarboxed

	out := resizeToCanvas(src, prm, "keepsmaller")

	bg := cpc.GetColor(prm.Palette[0], prm.CpcPlus)
	// A square that fits the 200-line height is 200 wide, so 60 columns of
	// background sit on each side.
	if got := out.GetPixelColor(0, 100); got != bg {
		t.Errorf("left band = %v, want the background %v", got, bg)
	}
	if got := out.GetPixelColor(59, 100); got != bg {
		t.Errorf("column 59 = %v, want the background %v", got, bg)
	}
	if got := out.GetPixelColor(60, 100); got == bg {
		t.Error("column 60 is background, want the first column of the picture")
	}
	if got := out.GetPixelColor(259, 100); got == bg {
		t.Error("column 259 is background, want the last column of the picture")
	}
	if got := out.GetPixelColor(260, 100); got != bg {
		t.Errorf("column 260 = %v, want the background %v", got, bg)
	}
	// Nothing is cropped: both markers survive.
	if !hasColour(out, markerTopRow) || !hasColour(out, markerLeftCol) {
		t.Error("keepsmaller crops nothing, markers are missing")
	}
}

// TestResizePolicyKeepLargerCrops pins the other aspect-preserving policy: the
// picture fills the screen and the overflow is cut off, so the marker rows of a
// square source are gone while nothing is left to the background.
func TestResizePolicyKeepLargerCrops(t *testing.T) {
	prm := resizeScreen()
	src := markerSource(100, 100)

	out := resizeToCanvas(src, prm, "keeplarger")

	bg := cpc.GetColor(prm.Palette[0], prm.CpcPlus)
	if hasColour(out, bg) {
		t.Error("keeplarger fills the screen, but the background colour survived")
	}
	if !hasColour(out, markerLeftCol) {
		t.Error("the source's full width is kept, the column marker is missing")
	}
	if hasColour(out, markerTopRow) {
		t.Error("the cropped rows still show, the row marker survived")
	}
}

// TestResizePolicyOriginClips pins the no-scaling policy: the picture is copied
// pixel for pixel from 0,0 and everything outside the screen is dropped.
func TestResizePolicyOriginClips(t *testing.T) {
	prm := resizeScreen()
	src := markerSource(40, 30)

	out := resizeToCanvas(src, prm, "origin")

	bg := cpc.GetColor(prm.Palette[0], prm.CpcPlus)
	if got, want := out.GetPixelColor(0, 0), src.GetPixelColor(0, 0); got != want {
		t.Errorf("top-left = %v, want the source's %v", got, want)
	}
	if got, want := out.GetPixelColor(39, 29), src.GetPixelColor(39, 29); got != want {
		t.Errorf("last source pixel = %v, want %v", got, want)
	}
	// One pixel further out is off the picture and keeps the background.
	if got := out.GetPixelColor(40, 29); got != bg {
		t.Errorf("column 40 = %v, want the background %v", got, bg)
	}
	if got := out.GetPixelColor(39, 30); got != bg {
		t.Errorf("row 30 = %v, want the background %v", got, bg)
	}
	// A picture wider than the screen is clipped at the screen's edge.
	wide := markerSource(prm.GetScreenWidth()+20, 30)
	if got, want := resizeToCanvas(wide, prm, "origin").Width(), prm.GetScreenWidth(); got != want {
		t.Errorf("clipped width = %d, want %d", got, want)
	}
}

func TestParseResizePolicy(t *testing.T) {
	cases := []struct {
		value   string
		want    string
		wantErr bool
	}{
		{"fit", "fit", false},
		{"KeepSmaller", "keepsmaller", false},
		{"KEEPLARGER", "keeplarger", false},
		{"origin", "origin", false},
		{" stretch ", "fit", true}, // stretch was never a policy name
		{"", "fit", true},
	}

	for _, tc := range cases {
		got, err := parseResizePolicy(tc.value)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseResizePolicy(%q) = %q, want an error", tc.value, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseResizePolicy(%q) failed: %v", tc.value, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseResizePolicy(%q) = %q, want %q", tc.value, got, tc.want)
		}
	}
}
