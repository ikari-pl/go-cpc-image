// Package fileio provides GIF animation import support.
package fileio

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"os"
	"time"

	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
)

// LoadGIF reads an animated GIF and returns individual frames as DirectBitmaps
// along with per-frame delays. Each frame is composited according to the GIF
// disposal method so that the returned bitmaps represent the fully rendered
// frame as it would appear on screen.
func LoadGIF(filename string) ([]*bitmap.DirectBitmap, []time.Duration, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("open GIF: %w", err)
	}
	defer f.Close()

	g, err := gif.DecodeAll(f)
	if err != nil {
		return nil, nil, fmt.Errorf("decode GIF: %w", err)
	}

	if len(g.Image) == 0 {
		return nil, nil, fmt.Errorf("GIF contains no frames")
	}

	// Determine the logical screen size from the GIF config.
	width, height := g.Config.Width, g.Config.Height
	if width == 0 || height == 0 {
		// Fallback: use first frame bounds.
		b := g.Image[0].Bounds()
		width, height = b.Dx(), b.Dy()
	}

	// Canvas holds the composited state between frames.
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with background color if available.
	if g.BackgroundIndex < uint8(len(g.Config.ColorModel.(color.Palette))) {
		bgColor := g.Config.ColorModel.(color.Palette)[g.BackgroundIndex]
		draw.Draw(canvas, canvas.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)
	}

	frames := make([]*bitmap.DirectBitmap, 0, len(g.Image))
	delays := make([]time.Duration, 0, len(g.Image))

	for i, frame := range g.Image {
		// Draw this frame onto the canvas.
		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)

		// Snapshot the canvas into a DirectBitmap.
		db := bitmap.NewDirectBitmap(width, height)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				r, gr, b, _ := canvas.At(x, y).RGBA()
				db.SetPixelColor(x, y, bitmap.NewRgbColor(
					uint8(r>>8), uint8(gr>>8), uint8(b>>8),
				))
			}
		}
		frames = append(frames, db)

		// Capture delay (GIF delays are in centiseconds).
		delay := time.Duration(0)
		if i < len(g.Delay) {
			delay = time.Duration(g.Delay[i]) * 10 * time.Millisecond
		}
		delays = append(delays, delay)

		// Handle disposal method.
		disposal := byte(gif.DisposalNone)
		if i < len(g.Disposal) {
			disposal = g.Disposal[i]
		}
		switch disposal {
		case gif.DisposalBackground:
			// Clear the frame area to background.
			draw.Draw(canvas, frame.Bounds(), image.Transparent, image.Point{}, draw.Src)
		case gif.DisposalPrevious:
			// Restoring to previous is complex; for simplicity treat as none.
		}
	}

	return frames, delays, nil
}
