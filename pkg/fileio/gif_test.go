package fileio

import (
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGIF_TwoFrames(t *testing.T) {
	// Create a synthetic 2-frame GIF in a temp file.
	dir := t.TempDir()
	path := filepath.Join(dir, "test.gif")

	g := &gif.GIF{
		Image: []*image.Paletted{
			createFrame(4, 4, color.RGBA{255, 0, 0, 255}),
			createFrame(4, 4, color.RGBA{0, 0, 255, 255}),
		},
		Delay:    []int{10, 20}, // 100ms, 200ms
		Disposal: []byte{gif.DisposalNone, gif.DisposalNone},
		Config: image.Config{
			Width:  4,
			Height: 4,
			ColorModel: color.Palette{
				color.RGBA{0, 0, 0, 255},
				color.RGBA{255, 0, 0, 255},
				color.RGBA{0, 0, 255, 255},
			},
		},
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := gif.EncodeAll(f, g); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	// Load the GIF back.
	frames, delays, err := LoadGIF(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(frames) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(frames))
	}
	if len(delays) != 2 {
		t.Fatalf("expected 2 delays, got %d", len(delays))
	}

	// Check dimensions.
	for i, fr := range frames {
		if fr.Width() != 4 || fr.Height() != 4 {
			t.Errorf("frame %d: expected 4x4, got %dx%d", i, fr.Width(), fr.Height())
		}
	}

	// Check delays.
	if delays[0].Milliseconds() != 100 {
		t.Errorf("frame 0 delay: expected 100ms, got %dms", delays[0].Milliseconds())
	}
	if delays[1].Milliseconds() != 200 {
		t.Errorf("frame 1 delay: expected 200ms, got %dms", delays[1].Milliseconds())
	}

	// Check that frame 0 has red pixels (top-left).
	c := frames[0].GetPixelColor(0, 0)
	if c.R < 200 || c.V > 50 || c.B > 50 {
		t.Errorf("frame 0 pixel: expected red, got R=%d V=%d B=%d", c.R, c.V, c.B)
	}
}

func createFrame(w, h int, fill color.RGBA) *image.Paletted {
	palette := color.Palette{color.RGBA{0, 0, 0, 255}, fill}
	img := image.NewPaletted(image.Rect(0, 0, w, h), palette)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetColorIndex(x, y, 1) // fill color
		}
	}
	return img
}
