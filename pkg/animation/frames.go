// Package animation provides multi-frame image handling and GIF import functionality.
// This handles frame management for CPC animation sequences.
package animation

import (
	"fmt"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"

	"github.com/ikari/go-cpc-image/pkg/bitmap"
)

// ImageSource manages multiple frames for animation sequences.
// This is equivalent to the C# ImageSource class.
type ImageSource struct {
	tabImage  []*bitmap.DirectBitmap // Array of frame images
	imageSel  int                     // Currently selected frame index
	tpsFrame  int                     // Frame timing in centiseconds
}

// NewImageSource creates a new empty image source.
func NewImageSource() *ImageSource {
	return &ImageSource{
		tabImage: make([]*bitmap.DirectBitmap, 0),
		imageSel: 0,
		tpsFrame: 0,
	}
}

// Init initializes the image source with a single empty frame.
func (is *ImageSource) Init() {
	is.ClearAll()
	is.tabImage = append(is.tabImage, bitmap.NewDirectBitmap(768, 544))
}

// ClearAll disposes of all frame images and clears the list.
func (is *ImageSource) ClearAll() {
	// In Go, we don't need explicit disposal, but we clear the slice
	is.tabImage = make([]*bitmap.DirectBitmap, 0)
	is.imageSel = 0
	is.tpsFrame = 0
}

// GetImage returns the currently selected frame image.
func (is *ImageSource) GetImage() *bitmap.DirectBitmap {
	if len(is.tabImage) == 0 {
		return nil
	}

	index := is.imageSel
	if index >= len(is.tabImage) {
		index = len(is.tabImage) - 1
	}

	return is.tabImage[index]
}

// NbImg returns the number of frame images.
func (is *ImageSource) NbImg() int {
	return len(is.tabImage)
}

// GetFrameTime returns the frame timing in centiseconds.
func (is *ImageSource) GetFrameTime() int {
	return is.tpsFrame
}

// SetFrameTime sets the frame timing in centiseconds.
func (is *ImageSource) SetFrameTime(time int) {
	is.tpsFrame = time
}

// InitBitmap initializes with a single bitmap frame.
func (is *ImageSource) InitBitmap(bmp *bitmap.DirectBitmap) {
	is.ClearAll()

	// Create a copy of the bitmap
	newBmp := bitmap.NewDirectBitmap(bmp.Width(), bmp.Height())
	newBmp.CopyBits(bmp)
	is.tabImage = append(is.tabImage, newBmp)
}

// InitFromReader initializes from an image reader (supports GIF animations).
func (is *ImageSource) InitFromReader(r io.Reader) error {
	is.ClearAll()

	// Try to decode as GIF first (for animation support)
	if gifImg, err := gif.DecodeAll(r); err == nil {
		return is.initFromGif(gifImg)
	}

	// Reset reader position is not possible with io.Reader
	// For single images, we would need to re-open the source
	// This is a limitation of the current API design
	return fmt.Errorf("failed to decode image - only GIF format supported for animations")
}

// initFromGif initializes from a decoded GIF image with animation support.
func (is *ImageSource) initFromGif(gifImg *gif.GIF) error {
	nbImg := len(gifImg.Image)
	if nbImg == 0 {
		return fmt.Errorf("no frames found in GIF")
	}

	// Set frame timing from GIF delay (convert from centiseconds)
	if nbImg > 1 && len(gifImg.Delay) > 0 {
		// GIF delay is in centiseconds, we store in centiseconds too
		is.tpsFrame = gifImg.Delay[0] * 10
	} else {
		is.tpsFrame = 0
	}

	// Process each frame
	for n := 0; n < nbImg; n++ {
		frame := gifImg.Image[n]
		if frame == nil {
			continue
		}

		bounds := frame.Bounds()
		width, height := bounds.Dx(), bounds.Dy()

		// Create DirectBitmap for this frame
		frameBmp := bitmap.NewDirectBitmap(width, height)

		// Copy pixel data from GIF frame to DirectBitmap
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				color := frame.At(x, y)
				// Convert to RgbColor
				r, g, b, _ := color.RGBA()
				rvbColor := bitmap.NewRgbColor(
					uint8(r>>8),
					uint8(g>>8),
					uint8(b>>8),
				)
				frameBmp.SetPixelColor(x, y, rvbColor)
			}
		}

		is.tabImage = append(is.tabImage, frameBmp)
	}

	return nil
}

// InitFrameCount initializes with a specified number of empty frames.
func (is *ImageSource) InitFrameCount(nbImg int) {
	is.ClearAll()

	for n := 0; n < nbImg; n++ {
		is.tabImage = append(is.tabImage, bitmap.NewDirectBitmap(768, 544))
	}
}

// ImportBitmap imports a bitmap to replace the frame at the specified index.
func (is *ImageSource) ImportBitmap(bmp *bitmap.DirectBitmap, numImage int) error {
	if numImage < 0 || numImage >= len(is.tabImage) {
		return fmt.Errorf("frame index %d out of range [0, %d)", numImage, len(is.tabImage))
	}

	// Create a copy of the bitmap
	newBmp := bitmap.NewDirectBitmap(bmp.Width(), bmp.Height())
	newBmp.CopyBits(bmp)
	is.tabImage[numImage] = newBmp

	return nil
}

// GetBitmap returns the bitmap for the specified frame index.
func (is *ImageSource) GetBitmap(numImage int) *bitmap.DirectBitmap {
	if numImage < 0 || numImage >= len(is.tabImage) {
		// Return the last frame if index is out of range
		if len(is.tabImage) > 0 {
			return is.tabImage[len(is.tabImage)-1]
		}
		return nil
	}

	return is.tabImage[numImage]
}

// AddFrame adds a new frame to the animation sequence.
func (is *ImageSource) AddFrame(bmp *bitmap.DirectBitmap) {
	// Create a copy of the bitmap
	newBmp := bitmap.NewDirectBitmap(bmp.Width(), bmp.Height())
	newBmp.CopyBits(bmp)
	is.tabImage = append(is.tabImage, newBmp)
}

// DeleteImage removes the frame at the specified index.
func (is *ImageSource) DeleteImage(numImage int) error {
	if numImage < 0 || numImage >= len(is.tabImage) {
		return fmt.Errorf("frame index %d out of range [0, %d)", numImage, len(is.tabImage))
	}

	// Remove the frame at the specified index
	is.tabImage = append(is.tabImage[:numImage], is.tabImage[numImage+1:]...)

	// Adjust selected frame index if necessary
	if is.imageSel >= len(is.tabImage) {
		is.imageSel = len(is.tabImage) - 1
	}
	if is.imageSel < 0 {
		is.imageSel = 0
	}

	return nil
}

// SelectBitmap sets the currently selected frame index.
func (is *ImageSource) SelectBitmap(num int) {
	if num >= 0 && num < len(is.tabImage) {
		is.imageSel = num
	}
}

// GetSelectedIndex returns the currently selected frame index.
func (is *ImageSource) GetSelectedIndex() int {
	return is.imageSel
}

// GetFrameRate returns the frame rate in frames per second.
func (is *ImageSource) GetFrameRate() float64 {
	if is.tpsFrame <= 0 {
		return 0.0
	}
	return 100.0 / float64(is.tpsFrame) // Convert centiseconds to FPS
}

// SetFrameRate sets the frame rate in frames per second.
func (is *ImageSource) SetFrameRate(fps float64) {
	if fps > 0 {
		is.tpsFrame = int(100.0 / fps) // Convert FPS to centiseconds
	} else {
		is.tpsFrame = 0
	}
}

// GetFrameDelay returns the delay for a specific frame in centiseconds.
// For now, all frames use the same delay.
func (is *ImageSource) GetFrameDelay(frameIndex int) int {
	return is.tpsFrame
}

// Clone creates a deep copy of the image source.
func (is *ImageSource) Clone() *ImageSource {
	clone := &ImageSource{
		tabImage: make([]*bitmap.DirectBitmap, len(is.tabImage)),
		imageSel: is.imageSel,
		tpsFrame: is.tpsFrame,
	}

	// Copy each frame
	for i, frame := range is.tabImage {
		if frame != nil {
			newFrame := bitmap.NewDirectBitmap(frame.Width(), frame.Height())
			newFrame.CopyBits(frame)
			clone.tabImage[i] = newFrame
		}
	}

	return clone
}

// GetFrameSize returns the dimensions of the first frame.
func (is *ImageSource) GetFrameSize() (int, int) {
	if len(is.tabImage) == 0 || is.tabImage[0] == nil {
		return 0, 0
	}

	frame := is.tabImage[0]
	return frame.Width(), frame.Height()
}

// ResizeAllFrames resizes all frames to the specified dimensions.
// Note: This is a simplified resize that doesn't do proper image scaling.
func (is *ImageSource) ResizeAllFrames(newWidth, newHeight int) {
	for i, frame := range is.tabImage {
		if frame == nil {
			continue
		}

		// Create new frame with target size
		newFrame := bitmap.NewDirectBitmap(newWidth, newHeight)

		// Simple copy (crop or extend as needed)
		oldWidth, oldHeight := frame.Width(), frame.Height()
		copyWidth := newWidth
		if oldWidth < copyWidth {
			copyWidth = oldWidth
		}
		copyHeight := newHeight
		if oldHeight < copyHeight {
			copyHeight = oldHeight
		}

		// Copy pixels from old to new frame
		for y := 0; y < copyHeight; y++ {
			for x := 0; x < copyWidth; x++ {
				color := frame.GetPixelColor(x, y)
				newFrame.SetPixelColor(x, y, color)
			}
		}

		is.tabImage[i] = newFrame
	}
}

// SaveAsGIF saves the animation as a GIF file (placeholder for future implementation).
// This would require a GIF encoder implementation or external library.
func (is *ImageSource) SaveAsGIF(w io.Writer) error {
	if len(is.tabImage) == 0 {
		return fmt.Errorf("no frames to save")
	}

	// This would require implementing GIF encoding
	// For now, return an error indicating it's not implemented
	return fmt.Errorf("GIF encoding not implemented yet")
}

// GetAnimationInfo returns information about the animation.
func (is *ImageSource) GetAnimationInfo() (frameCount int, frameRate float64, duration float64) {
	frameCount = len(is.tabImage)
	frameRate = is.GetFrameRate()

	if frameRate > 0 {
		duration = float64(frameCount) / frameRate
	} else {
		duration = 0
	}

	return
}