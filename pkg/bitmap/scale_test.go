package bitmap

import (
	"image"
	"image/color"
	"testing"
)

// TestScaleImageReplicatesWholePixels pins the one thing the scaler may do to a
// picture: repeat or drop whole pixels, so a pen colour survives exactly.
func TestScaleImageReplicatesWholePixels(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	colors := [...][4]uint8{
		{1, 2, 3, 255}, {4, 5, 6, 255},
		{7, 8, 9, 255}, {10, 11, 12, 255},
	}
	for i, c := range colors {
		src.Set(i%2, i/2, color.RGBA{R: c[0], G: c[1], B: c[2], A: c[3]})
	}

	got := ScaleImage(src, 6, 4) // 3x across, 2x down
	if got.Bounds().Dx() != 6 || got.Bounds().Dy() != 4 {
		t.Fatalf("size = %dx%d, want 6x4", got.Bounds().Dx(), got.Bounds().Dy())
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 6; x++ {
			want := src.RGBAAt(x/3, y/2)
			if got.RGBAAt(x, y) != want {
				t.Errorf("(%d,%d) = %v, want %v (the pixel at (%d,%d))",
					x, y, got.RGBAAt(x, y), want, x/3, y/2)
			}
		}
	}

	// A request for the size it already has is the picture itself, and a
	// reduction drops whole pixels instead of blending them.
	if ScaleImage(src, 2, 2) != src {
		t.Error("a same-size request returned a copy")
	}
	if small := ScaleImage(src, 1, 1); small.RGBAAt(0, 0) != src.RGBAAt(0, 0) {
		t.Errorf("reduced pixel = %v, want the source's own %v",
			small.RGBAAt(0, 0), src.RGBAAt(0, 0))
	}
	if ScaleImage(nil, 4, 4) != nil {
		t.Error("a nil source produced a picture")
	}
}
