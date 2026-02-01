// Package gui provides the preview widget for displaying source and CPC images.
// This file implements the live preview functionality with zoom controls.
package gui

import (
	"image"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// PreviewWidget manages the split preview display showing source and CPC images.
type PreviewWidget struct {
	app *Application

	// Source image display
	sourceContainer *fyne.Container
	sourceRaster    *canvas.Raster
	sourceImage     *image.RGBA

	// CPC image display
	cpcContainer *fyne.Container
	cpcRaster    *canvas.Raster

	// Zoom controls
	zoomLevel     int // 1x, 2x, 4x
	zoomContainer *fyne.Container
}

// NewPreviewWidget creates a new preview widget for the application.
func NewPreviewWidget(app *Application) (*PreviewWidget, error) {
	pw := &PreviewWidget{
		app:       app,
		zoomLevel: 1,
	}

	// Create source image raster
	pw.sourceRaster = canvas.NewRaster(pw.drawSourceImage)
	pw.sourceRaster.SetMinSize(fyne.NewSize(320, 200))

	// Create CPC image raster
	pw.cpcRaster = canvas.NewRaster(pw.drawCpcImage)
	pw.cpcRaster.SetMinSize(fyne.NewSize(320, 200))

	// Create zoom controls
	pw.setupZoomControls()

	// Create containers
	pw.sourceContainer = container.NewBorder(
		widget.NewLabel("Source Image"),
		nil, nil, nil,
		pw.sourceRaster,
	)

	pw.cpcContainer = container.NewBorder(
		container.NewHBox(
			widget.NewLabel("CPC Preview"),
			widget.NewSeparator(),
			pw.zoomContainer,
		),
		nil, nil, nil,
		pw.cpcRaster,
	)

	return pw, nil
}

// setupZoomControls creates the zoom control buttons.
func (pw *PreviewWidget) setupZoomControls() {
	zoom1x := widget.NewButton("1x", func() { pw.setZoom(1) })
	zoom2x := widget.NewButton("2x", func() { pw.setZoom(2) })
	zoom4x := widget.NewButton("4x", func() { pw.setZoom(4) })

	// Set initial button state
	zoom1x.Importance = widget.HighImportance

	pw.zoomContainer = container.NewHBox(
		widget.NewLabel("Zoom:"),
		zoom1x, zoom2x, zoom4x,
	)
}

// setZoom changes the zoom level and updates the display.
func (pw *PreviewWidget) setZoom(level int) {
	pw.zoomLevel = level

	// Update raster size based on zoom
	baseWidth := 320
	baseHeight := 200

	if pw.app.params.IsOverscan() {
		baseWidth = 384  // 96 cols * 4
		baseHeight = 272 // overscan height
	}

	newSize := fyne.NewSize(float32(baseWidth*level), float32(baseHeight*level))
	pw.cpcRaster.SetMinSize(newSize)
	pw.cpcRaster.Refresh()

	// Update button states
	for i, obj := range pw.zoomContainer.Objects {
		if i < 1 { // Skip the label
			continue
		}
		if btn, ok := obj.(*widget.Button); ok {
			if (i == 1 && level == 1) || (i == 2 && level == 2) || (i == 3 && level == 4) {
				btn.Importance = widget.HighImportance
			} else {
				btn.Importance = widget.MediumImportance
			}
			btn.Refresh()
		}
	}
}

// drawSourceImage renders the source image to the canvas.
func (pw *PreviewWidget) drawSourceImage(w, h int) image.Image {
	if pw.sourceImage == nil {
		// Return empty gray image
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		gray := color.RGBA{128, 128, 128, 255}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				img.Set(x, y, gray)
			}
		}
		return img
	}

	// Scale source image to fit the canvas
	return pw.scaleImage(pw.sourceImage, w, h)
}

// drawCpcImage renders the CPC image to the canvas with zoom.
func (pw *PreviewWidget) drawCpcImage(w, h int) image.Image {
	cpcImg := pw.app.GetCpcDisplay()
	if cpcImg == nil {
		// Return empty black image
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		black := color.RGBA{0, 0, 0, 255}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				img.Set(x, y, black)
			}
		}
		return img
	}

	// Apply zoom scaling
	if pw.zoomLevel > 1 {
		return pw.scaleImageNearest(cpcImg, w, h)
	}

	return pw.scaleImage(cpcImg, w, h)
}

// scaleImage scales an image to fit the given dimensions using bilinear interpolation.
// Preserves aspect ratio and centers the image with black bars if needed.
func (pw *PreviewWidget) scaleImage(src *image.RGBA, w, h int) image.Image {
	if src == nil {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}

	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	if srcW == 0 || srcH == 0 {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}

	// Calculate aspect-preserving dimensions
	srcAspect := float64(srcW) / float64(srcH)
	dstAspect := float64(w) / float64(h)

	var scaledW, scaledH int
	var offsetX, offsetY int

	if srcAspect > dstAspect {
		// Source is wider - fit to width
		scaledW = w
		scaledH = int(float64(w) / srcAspect)
		offsetX = 0
		offsetY = (h - scaledH) / 2
	} else {
		// Source is taller - fit to height
		scaledW = int(float64(h) * srcAspect)
		scaledH = h
		offsetX = (w - scaledW) / 2
		offsetY = 0
	}

	// Create destination with black background
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	black := color.RGBA{0, 0, 0, 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(x, y, black)
		}
	}

	// Scale and draw the image centered
	xRatio := float64(srcW) / float64(scaledW)
	yRatio := float64(srcH) / float64(scaledH)

	for y := 0; y < scaledH; y++ {
		for x := 0; x < scaledW; x++ {
			srcX := int(float64(x) * xRatio)
			srcY := int(float64(y) * yRatio)

			if srcX >= srcW {
				srcX = srcW - 1
			}
			if srcY >= srcH {
				srcY = srcH - 1
			}

			dst.Set(offsetX+x, offsetY+y, src.At(srcX, srcY))
		}
	}

	return dst
}

// scaleImageNearest scales an image using nearest-neighbor interpolation (for pixel-perfect zoom).
func (pw *PreviewWidget) scaleImageNearest(src *image.RGBA, w, h int) image.Image {
	if src == nil {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}

	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	if srcW == 0 || srcH == 0 {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	// For zoomed display, each source pixel becomes zoomLevel x zoomLevel dest pixels
	pixelW := w / srcW
	pixelH := h / srcH

	if pixelW == 0 { pixelW = 1 }
	if pixelH == 0 { pixelH = 1 }

	for srcY := 0; srcY < srcH; srcY++ {
		for srcX := 0; srcX < srcW; srcX++ {
			c := src.At(srcX, srcY)

			// Draw pixelW x pixelH block
			for dy := 0; dy < pixelH && srcY*pixelH+dy < h; dy++ {
				for dx := 0; dx < pixelW && srcX*pixelW+dx < w; dx++ {
					dst.Set(srcX*pixelW+dx, srcY*pixelH+dy, c)
				}
			}
		}
	}

	return dst
}

// SetSourceImage updates the displayed source image.
func (pw *PreviewWidget) SetSourceImage(img *image.RGBA) {
	pw.sourceImage = img
	pw.sourceRaster.Refresh()
}

// RefreshCpc updates the CPC preview display.
func (pw *PreviewWidget) RefreshCpc() {
	pw.cpcRaster.Refresh()
}

// GetSourceContainer returns the source image container.
func (pw *PreviewWidget) GetSourceContainer() *fyne.Container {
	return pw.sourceContainer
}

// GetCpcContainer returns the CPC image container.
func (pw *PreviewWidget) GetCpcContainer() *fyne.Container {
	return pw.cpcContainer
}