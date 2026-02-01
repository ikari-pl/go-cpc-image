// Package gui provides the preview widget for displaying source and CPC images.
// This file implements the live preview functionality with zoom controls.
package gui

import (
	"image"
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type cropDragMode int

const (
	cropDragNew cropDragMode = iota // drawing a new rectangle
	cropDragMove                    // moving the entire box
	cropDragN                       // resize north edge
	cropDragS                       // resize south edge
	cropDragE                       // resize east edge
	cropDragW                       // resize west edge
	cropDragNE                      // resize NE corner
	cropDragNW                      // resize NW corner
	cropDragSE                      // resize SE corner
	cropDragSW                      // resize SW corner
)

// drawableRaster is a custom widget that wraps a Raster and handles mouse events for drawing.
type drawableRaster struct {
	widget.BaseWidget
	raster      *canvas.Raster
	onMouseDown func(fyne.Position, fyne.Size)
	onMouseDrag func(fyne.Position, fyne.Size)
	onMouseUp   func(fyne.Position, fyne.Size)
	dragStarted bool
}

func newDrawableRaster(raster *canvas.Raster) *drawableRaster {
	d := &drawableRaster{raster: raster}
	d.ExtendBaseWidget(d)
	return d
}

func (d *drawableRaster) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(d.raster)
}

func (d *drawableRaster) MinSize() fyne.Size {
	return d.raster.MinSize()
}

func (d *drawableRaster) Tapped(ev *fyne.PointEvent) {
	// Tapped = click without drag. Fire down+up so a click sets+resets crop anchor.
	if d.onMouseDown != nil {
		d.onMouseDown(ev.Position, d.Size())
	}
	if d.onMouseUp != nil {
		d.onMouseUp(ev.Position, d.Size())
	}
}

func (d *drawableRaster) Dragged(ev *fyne.DragEvent) {
	// Fyne calls Dragged without calling Tapped first, so we fire onMouseDown
	// on the first Dragged event to initialise the drag anchor.
	if !d.dragStarted {
		d.dragStarted = true
		if d.onMouseDown != nil {
			d.onMouseDown(ev.Position, d.Size())
		}
	}
	if d.onMouseDrag != nil {
		d.onMouseDrag(ev.Position, d.Size())
	}
}

func (d *drawableRaster) DragEnd() {
	d.dragStarted = false
	if d.onMouseUp != nil {
		d.onMouseUp(fyne.Position{X: -1, Y: -1}, d.Size())
	}
}

// PreviewWidget manages the split preview display showing source and CPC images.
type PreviewWidget struct {
	app *Application

	// Source image display
	sourceContainer *fyne.Container
	sourceRaster    *canvas.Raster
	sourceDrawable  *drawableRaster
	sourceImage     *image.RGBA

	// CPC image display
	cpcContainer *fyne.Container
	cpcRaster    *canvas.Raster
	cpcDrawable  *drawableRaster

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

	// Wrap source raster in drawable for crop mouse events
	pw.sourceDrawable = newDrawableRaster(pw.sourceRaster)
	pw.sourceDrawable.onMouseDown = func(pos fyne.Position, size fyne.Size) {
		if !app.cropEnabled {
			return
		}
		nx, ny := pw.screenToNormalized(pos, size)
		if nx < 0 || ny < 0 {
			return
		}

		const edgeThreshold = 0.02

		// Normalize existing crop coords
		cx1, cx2 := app.cropX1, app.cropX2
		cy1, cy2 := app.cropY1, app.cropY2
		if cx1 > cx2 {
			cx1, cx2 = cx2, cx1
		}
		if cy1 > cy2 {
			cy1, cy2 = cy2, cy1
		}

		nearLeft := math.Abs(nx-cx1) < edgeThreshold
		nearRight := math.Abs(nx-cx2) < edgeThreshold
		nearTop := math.Abs(ny-cy1) < edgeThreshold
		nearBottom := math.Abs(ny-cy2) < edgeThreshold
		insideX := nx > cx1+edgeThreshold && nx < cx2-edgeThreshold
		insideY := ny > cy1+edgeThreshold && ny < cy2-edgeThreshold

		var mode cropDragMode
		if nearTop && nearLeft {
			mode = cropDragNW
		} else if nearTop && nearRight {
			mode = cropDragNE
		} else if nearBottom && nearLeft {
			mode = cropDragSW
		} else if nearBottom && nearRight {
			mode = cropDragSE
		} else if nearTop && insideX {
			mode = cropDragN
		} else if nearBottom && insideX {
			mode = cropDragS
		} else if nearLeft && insideY {
			mode = cropDragW
		} else if nearRight && insideY {
			mode = cropDragE
		} else if insideX && insideY {
			mode = cropDragMove
		} else {
			mode = cropDragNew
		}

		app.cropDragMode = mode
		app.cropDragStartX = nx
		app.cropDragStartY = ny
		app.cropOrigX1 = cx1
		app.cropOrigY1 = cy1
		app.cropOrigX2 = cx2
		app.cropOrigY2 = cy2

		switch mode {
		case cropDragNew:
			app.cropX1 = nx
			app.cropY1 = ny
			app.cropX2 = nx
			app.cropY2 = ny
		case cropDragNW:
			// Anchor at opposite corner (SE)
			app.cropX1 = cx2
			app.cropY1 = cy2
			app.cropX2 = nx
			app.cropY2 = ny
		case cropDragNE:
			app.cropX1 = cx1
			app.cropY1 = cy2
			app.cropX2 = nx
			app.cropY2 = ny
		case cropDragSW:
			app.cropX1 = cx2
			app.cropY1 = cy1
			app.cropX2 = nx
			app.cropY2 = ny
		case cropDragSE:
			app.cropX1 = cx1
			app.cropY1 = cy1
			app.cropX2 = nx
			app.cropY2 = ny
		}

		app.cropDragging = true
		pw.sourceRaster.Refresh()
	}
	pw.sourceDrawable.onMouseDrag = func(pos fyne.Position, size fyne.Size) {
		if !app.cropEnabled || !app.cropDragging {
			return
		}
		nx, ny := pw.screenToNormalized(pos, size)
		// Clamp to 0-1
		if nx < 0 {
			nx = 0
		}
		if nx > 1 {
			nx = 1
		}
		if ny < 0 {
			ny = 0
		}
		if ny > 1 {
			ny = 1
		}

		switch app.cropDragMode {
		case cropDragMove:
			dx := nx - app.cropDragStartX
			dy := ny - app.cropDragStartY
			newX1 := app.cropOrigX1 + dx
			newY1 := app.cropOrigY1 + dy
			newX2 := app.cropOrigX2 + dx
			newY2 := app.cropOrigY2 + dy
			if newX1 < 0 {
				newX2 -= newX1
				newX1 = 0
			}
			if newY1 < 0 {
				newY2 -= newY1
				newY1 = 0
			}
			if newX2 > 1 {
				newX1 -= (newX2 - 1)
				newX2 = 1
			}
			if newY2 > 1 {
				newY1 -= (newY2 - 1)
				newY2 = 1
			}
			app.cropX1 = newX1
			app.cropY1 = newY1
			app.cropX2 = newX2
			app.cropY2 = newY2
			pw.sourceRaster.Refresh()
			return

		case cropDragN:
			app.cropY1 = app.cropOrigY1 + (ny - app.cropDragStartY)
			if app.cropY1 < 0 {
				app.cropY1 = 0
			}
			if app.cropY1 > app.cropY2 {
				app.cropY1 = app.cropY2
			}
			if app.cropLockAspect && app.cropAspect > 0 {
				pw.enforceAspectFromEdge(app)
			}
			pw.sourceRaster.Refresh()
			return

		case cropDragS:
			app.cropY2 = app.cropOrigY2 + (ny - app.cropDragStartY)
			if app.cropY2 > 1 {
				app.cropY2 = 1
			}
			if app.cropY2 < app.cropY1 {
				app.cropY2 = app.cropY1
			}
			if app.cropLockAspect && app.cropAspect > 0 {
				pw.enforceAspectFromEdge(app)
			}
			pw.sourceRaster.Refresh()
			return

		case cropDragW:
			app.cropX1 = app.cropOrigX1 + (nx - app.cropDragStartX)
			if app.cropX1 < 0 {
				app.cropX1 = 0
			}
			if app.cropX1 > app.cropX2 {
				app.cropX1 = app.cropX2
			}
			if app.cropLockAspect && app.cropAspect > 0 {
				pw.enforceAspectFromEdge(app)
			}
			pw.sourceRaster.Refresh()
			return

		case cropDragE:
			app.cropX2 = app.cropOrigX2 + (nx - app.cropDragStartX)
			if app.cropX2 > 1 {
				app.cropX2 = 1
			}
			if app.cropX2 < app.cropX1 {
				app.cropX2 = app.cropX1
			}
			if app.cropLockAspect && app.cropAspect > 0 {
				pw.enforceAspectFromEdge(app)
			}
			pw.sourceRaster.Refresh()
			return
		}

		// cropDragNew, cropDragNW, cropDragNE, cropDragSW, cropDragSE:
		// All use anchor at cropX1,cropY1 and set cropX2,cropY2
		if app.cropLockAspect && app.cropAspect > 0 {
			// Enforce aspect ratio from the anchor point (cropX1, cropY1)
			dx := nx - app.cropX1
			dy := ny - app.cropY1
			// Use source image aspect to convert normalized coords to pixel space
			imgAspect := 1.0
			if pw.sourceImage != nil {
				srcB := pw.sourceImage.Bounds()
				if srcB.Dy() > 0 {
					imgAspect = float64(srcB.Dx()) / float64(srcB.Dy())
				}
			}
			targetRatio := app.cropAspect / imgAspect
			if dy != 0 {
				absDx := dx
				if absDx < 0 {
					absDx = -absDx
				}
				absDy := dy
				if absDy < 0 {
					absDy = -absDy
				}
				if absDx/absDy > targetRatio {
					sign := 1.0
					if dx < 0 {
						sign = -1.0
					}
					dx = sign * absDy * targetRatio
				} else {
					sign := 1.0
					if dy < 0 {
						sign = -1.0
					}
					dy = sign * absDx / targetRatio
				}
			}
			nx = app.cropX1 + dx
			ny = app.cropY1 + dy
			if nx < 0 {
				nx = 0
			}
			if nx > 1 {
				nx = 1
			}
			if ny < 0 {
				ny = 0
			}
			if ny > 1 {
				ny = 1
			}
		}

		app.cropX2 = nx
		app.cropY2 = ny
		pw.sourceRaster.Refresh()
	}
	pw.sourceDrawable.onMouseUp = func(pos fyne.Position, size fyne.Size) {
		if !app.cropEnabled {
			return
		}
		app.cropDragging = false
		// Trigger conversion with the new crop
		if app.controls != nil && app.controls.autoConvert != nil && app.controls.autoConvert.Checked {
			app.TriggerConversion()
		}
		pw.sourceRaster.Refresh()
	}

	// Create CPC image raster
	pw.cpcRaster = canvas.NewRaster(pw.drawCpcImage)
	pw.cpcRaster.SetMinSize(fyne.NewSize(320, 200))

	// Wrap in drawable for mouse events
	pw.cpcDrawable = newDrawableRaster(pw.cpcRaster)
	pw.cpcDrawable.onMouseDown = func(pos fyne.Position, size fyne.Size) {
		if app.editor == nil {
			return
		}
		cx, cy := app.editor.ScreenToCpc(pos.X, pos.Y, size.Width, size.Height)
		app.editor.OnMouseDown(cx, cy)
	}
	pw.cpcDrawable.onMouseDrag = func(pos fyne.Position, size fyne.Size) {
		if app.editor == nil {
			return
		}
		cx, cy := app.editor.ScreenToCpc(pos.X, pos.Y, size.Width, size.Height)
		app.editor.OnMouseDrag(cx, cy)
	}
	pw.cpcDrawable.onMouseUp = func(pos fyne.Position, size fyne.Size) {
		if app.editor == nil {
			return
		}
		cx, cy := app.editor.ScreenToCpc(pos.X, pos.Y, size.Width, size.Height)
		app.editor.OnMouseUp(cx, cy)
	}

	// Create zoom controls
	pw.setupZoomControls()

	// Create containers
	pw.sourceContainer = container.NewBorder(
		widget.NewLabel("Source Image"),
		nil, nil, nil,
		pw.sourceDrawable,
	)

	pw.cpcContainer = container.NewBorder(
		container.NewHBox(
			widget.NewLabel("CPC Preview"),
			widget.NewSeparator(),
			pw.zoomContainer,
		),
		nil, nil, nil,
		pw.cpcDrawable,
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
	img := pw.scaleImage(pw.sourceImage, w, h)

	// Draw crop overlay if enabled
	if pw.app.cropEnabled {
		rgbaImg, ok := img.(*image.RGBA)
		if !ok {
			return img
		}

		srcBounds := pw.sourceImage.Bounds()
		srcW := float64(srcBounds.Dx())
		srcH := float64(srcBounds.Dy())

		// Recalculate image placement (same as scaleImage)
		srcAspect := srcW / srcH
		dstAspect := float64(w) / float64(h)

		var scaledW, scaledH float64
		var offsetX, offsetY float64

		if srcAspect > dstAspect {
			scaledW = float64(w)
			scaledH = float64(w) / srcAspect
			offsetY = (float64(h) - scaledH) / 2
		} else {
			scaledW = float64(h) * srcAspect
			scaledH = float64(h)
			offsetX = (float64(w) - scaledW) / 2
		}

		// Normalize crop coords (ensure x1<x2, y1<y2)
		cx1, cx2 := pw.app.cropX1, pw.app.cropX2
		cy1, cy2 := pw.app.cropY1, pw.app.cropY2
		if cx1 > cx2 {
			cx1, cx2 = cx2, cx1
		}
		if cy1 > cy2 {
			cy1, cy2 = cy2, cy1
		}

		// Convert normalized crop to pixel coords in the rendered image
		cropPx1 := int(offsetX + cx1*scaledW)
		cropPy1 := int(offsetY + cy1*scaledH)
		cropPx2 := int(offsetX + cx2*scaledW)
		cropPy2 := int(offsetY + cy2*scaledH)

		// Draw semi-transparent dark overlay on non-selected areas
		overlay := color.RGBA{0, 0, 0, 128}
		imgX0 := int(offsetX)
		imgY0 := int(offsetY)
		imgX1 := int(offsetX + scaledW)
		imgY1 := int(offsetY + scaledH)

		for y := imgY0; y < imgY1; y++ {
			for x := imgX0; x < imgX1; x++ {
				if x >= cropPx1 && x < cropPx2 && y >= cropPy1 && y < cropPy2 {
					continue // inside crop region, leave as-is
				}
				// Blend overlay
				if x >= 0 && x < w && y >= 0 && y < h {
					orig := rgbaImg.RGBAAt(x, y)
					orig.R = uint8((int(orig.R)*128 + int(overlay.R)*128) / 256)
					orig.G = uint8((int(orig.G)*128 + int(overlay.G)*128) / 256)
					orig.B = uint8((int(orig.B)*128 + int(overlay.B)*128) / 256)
					rgbaImg.SetRGBA(x, y, orig)
				}
			}
		}

		// Draw crop rectangle border (white dashed-style)
		borderColor := color.RGBA{255, 255, 255, 255}
		for x := cropPx1; x < cropPx2; x++ {
			if x >= 0 && x < w {
				if cropPy1 >= 0 && cropPy1 < h {
					rgbaImg.Set(x, cropPy1, borderColor)
				}
				if cropPy2-1 >= 0 && cropPy2-1 < h {
					rgbaImg.Set(x, cropPy2-1, borderColor)
				}
			}
		}
		for y := cropPy1; y < cropPy2; y++ {
			if y >= 0 && y < h {
				if cropPx1 >= 0 && cropPx1 < w {
					rgbaImg.Set(cropPx1, y, borderColor)
				}
				if cropPx2-1 >= 0 && cropPx2-1 < w {
					rgbaImg.Set(cropPx2-1, y, borderColor)
				}
			}
		}

		// Draw handles at corners and edge midpoints
		handleSize := 4
		handles := [][2]int{
			{cropPx1, cropPy1},                         // NW
			{cropPx2 - 1, cropPy1},                     // NE
			{cropPx1, cropPy2 - 1},                     // SW
			{cropPx2 - 1, cropPy2 - 1},                 // SE
			{(cropPx1 + cropPx2) / 2, cropPy1},         // N
			{(cropPx1 + cropPx2) / 2, cropPy2 - 1},     // S
			{cropPx1, (cropPy1 + cropPy2) / 2},         // W
			{cropPx2 - 1, (cropPy1 + cropPy2) / 2},     // E
		}
		for _, hp := range handles {
			for dy := -handleSize; dy <= handleSize; dy++ {
				for dx := -handleSize; dx <= handleSize; dx++ {
					px, py := hp[0]+dx, hp[1]+dy
					if px >= 0 && px < w && py >= 0 && py < h {
						rgbaImg.Set(px, py, borderColor)
					}
				}
			}
		}

		return rgbaImg
	}

	return img
}

// drawCpcImage renders the CPC image to the canvas with zoom.
// Always displays at 4:3 aspect ratio (matching a real CPC monitor).
func (pw *PreviewWidget) drawCpcImage(w, h int) image.Image {
	cpcImg := pw.app.GetCpcDisplay()
	if cpcImg == nil {
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		black := color.RGBA{0, 0, 0, 255}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				img.Set(x, y, black)
			}
		}
		return img
	}

	// Apply zoom scaling with forced 4:3 aspect ratio
	if pw.zoomLevel > 1 {
		return pw.scaleImageWithAspect(cpcImg, w, h, 4.0/3.0, true)
	}
	return pw.scaleImageWithAspect(cpcImg, w, h, 4.0/3.0, false)
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

// scaleImageWithAspect scales src into w×h, forcing a specific display aspect ratio.
// The source pixels are mapped to a virtual rectangle with the given aspect ratio,
// letterboxed/pillarboxed into the destination. If nearest is true, uses nearest-neighbor.
func (pw *PreviewWidget) scaleImageWithAspect(src *image.RGBA, w, h int, aspect float64, nearest bool) image.Image {
	if src == nil {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}

	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	if srcW == 0 || srcH == 0 {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}

	// Fit a 4:3 rectangle into w×h
	dstAspect := float64(w) / float64(h)
	var scaledW, scaledH int
	if aspect > dstAspect {
		scaledW = w
		scaledH = int(float64(w) / aspect)
	} else {
		scaledH = h
		scaledW = int(float64(h) * aspect)
	}

	offsetX := (w - scaledW) / 2
	offsetY := (h - scaledH) / 2

	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	xRatio := float64(srcW) / float64(scaledW)
	yRatio := float64(srcH) / float64(scaledH)

	for y := 0; y < scaledH; y++ {
		srcY := int(float64(y) * yRatio)
		if srcY >= srcH {
			srcY = srcH - 1
		}
		for x := 0; x < scaledW; x++ {
			srcX := int(float64(x) * xRatio)
			if srcX >= srcW {
				srcX = srcW - 1
			}
			dst.Set(offsetX+x, offsetY+y, src.At(srcX, srcY))
		}
	}

	return dst
}

// enforceAspectFromEdge adjusts the perpendicular dimension symmetrically
// around the center when an edge is dragged with aspect lock on.
func (pw *PreviewWidget) enforceAspectFromEdge(app *Application) {
	imgAspect := 1.0
	if pw.sourceImage != nil {
		srcB := pw.sourceImage.Bounds()
		if srcB.Dy() > 0 {
			imgAspect = float64(srcB.Dx()) / float64(srcB.Dy())
		}
	}
	targetRatio := app.cropAspect / imgAspect

	cx1, cx2 := app.cropX1, app.cropX2
	cy1, cy2 := app.cropY1, app.cropY2
	if cx1 > cx2 {
		cx1, cx2 = cx2, cx1
	}
	if cy1 > cy2 {
		cy1, cy2 = cy2, cy1
	}

	w := cx2 - cx1
	h := cy2 - cy1
	centerX := (cx1 + cx2) / 2
	centerY := (cy1 + cy2) / 2

	currentRatio := 0.0
	if h > 0 {
		currentRatio = w / h
	}

	if currentRatio > targetRatio {
		// Too wide, expand height
		newH := w / targetRatio
		cy1 = centerY - newH/2
		cy2 = centerY + newH/2
	} else {
		// Too tall, expand width
		newW := h * targetRatio
		cx1 = centerX - newW/2
		cx2 = centerX + newW/2
	}

	// Clamp
	if cx1 < 0 {
		cx1 = 0
	}
	if cy1 < 0 {
		cy1 = 0
	}
	if cx2 > 1 {
		cx2 = 1
	}
	if cy2 > 1 {
		cy2 = 1
	}

	app.cropX1 = cx1
	app.cropY1 = cy1
	app.cropX2 = cx2
	app.cropY2 = cy2
}

// screenToNormalized converts a screen position within the source raster widget
// to normalized (0-1) coordinates relative to the actual image area (accounting for letterbox).
func (pw *PreviewWidget) screenToNormalized(pos fyne.Position, widgetSize fyne.Size) (float64, float64) {
	if pw.sourceImage == nil {
		return -1, -1
	}

	w := float64(widgetSize.Width)
	h := float64(widgetSize.Height)
	if w <= 0 || h <= 0 {
		return -1, -1
	}

	srcBounds := pw.sourceImage.Bounds()
	srcW := float64(srcBounds.Dx())
	srcH := float64(srcBounds.Dy())
	if srcW <= 0 || srcH <= 0 {
		return -1, -1
	}

	// Calculate where the image is drawn (same logic as scaleImage)
	srcAspect := srcW / srcH
	dstAspect := w / h

	var scaledW, scaledH float64
	var offsetX, offsetY float64

	if srcAspect > dstAspect {
		scaledW = w
		scaledH = w / srcAspect
		offsetX = 0
		offsetY = (h - scaledH) / 2
	} else {
		scaledW = h * srcAspect
		scaledH = h
		offsetX = (w - scaledW) / 2
		offsetY = 0
	}

	// Convert to normalized coords within the image area
	nx := (float64(pos.X) - offsetX) / scaledW
	ny := (float64(pos.Y) - offsetY) / scaledH

	return nx, ny
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