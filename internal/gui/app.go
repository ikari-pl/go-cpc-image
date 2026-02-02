// Package gui provides the Fyne GUI implementation for Go CPC image.
// This file contains the main application structure and window layout.
package gui

import (
	"context"
	"fmt"
	"image"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
	"github.com/ikari-pl/go-cpc-image/pkg/convert"
	"github.com/ikari-pl/go-cpc-image/pkg/cpc"
	"github.com/ikari-pl/go-cpc-image/pkg/render"
)

// convRequest is sent to the conversion worker goroutine.
type convRequest struct {
	ctx context.Context
}

// Application represents the main GUI application state.
// This structure maintains all the UI components and conversion state.
type Application struct {
	app    fyne.App
	window fyne.Window

	// Conversion state — only accessed by the conversion worker goroutine
	params    *convert.Settings
	sourceImg *bitmap.DirectBitmap
	cpcImage  *convert.ImageCpc

	// Lock-free display for the Fyne render thread
	cpcDisplay atomic.Pointer[image.RGBA]

	// sourceMu protects sourceImg which is written from the UI thread
	// and read from the worker goroutine
	sourceMu sync.Mutex

	// UI Components
	preview   *PreviewWidget
	controls  *ControlsWidget
	palette   *PaletteWidget
	toolbar   *ToolbarWidget
	statusBar *widget.Label

	// Drawing editor
	editor *Editor

	// Crop region (in source image coordinates, 0.0-1.0 normalized)
	cropEnabled    bool
	cropX1, cropY1 float64 // top-left normalized 0-1
	cropX2, cropY2 float64 // bottom-right normalized 0-1
	cropLockAspect bool
	cropAspect     float64 // dynamic, based on current screen mode
	cropDragging   bool    // true while user is drawing a crop rectangle
	cropDragMode   cropDragMode
	cropDragStartX float64
	cropDragStartY float64
	cropOrigX1     float64
	cropOrigY1     float64
	cropOrigX2     float64
	cropOrigY2     float64

	// Animation state
	animFrames []*bitmap.DirectBitmap  // loaded GIF frames
	animDelays []time.Duration         // per-frame delays
	animIndex  int                     // current frame index
	animSlider *widget.Slider          // frame navigation slider
	animLabel  *widget.Label           // frame counter label
	animPanel  *fyne.Container         // animation controls container

	// Debouncing + cancel-and-restart
	debounceMu    sync.Mutex // lightweight lock for timer only
	debounceTimer *time.Timer
	conversionCh  chan convRequest
	cancelConv    context.CancelFunc // cancels the currently running conversion
}

// NewApplication creates and initializes a new GUI application.
func NewApplication(app fyne.App, window fyne.Window) (*Application, error) {
	guiApp := &Application{
		app:          app,
		window:       window,
		params:       convert.NewDefaultSettings(),
		conversionCh: make(chan convRequest, 1),
		statusBar:    widget.NewLabel("Ready"),
	}

	// Initialize conversion image structures
	guiApp.cpcImage = &convert.ImageCpc{
		BitmapCpc:  render.NewBitmapCpcWithParams(80, 200, false), // Standard screen
		DisplayBmp: bitmap.NewDirectBitmap(640, 400),              // Display bitmap for SetPixelCpc
		Width:      640,
		Height:     400,
		Mode:       1,
	}

	// Create drawing editor (before UI components so they can reference it)
	guiApp.editor = NewEditor(guiApp)

	// Create UI components
	var err error

	// Create preview widget (custom raster for live CPC display)
	guiApp.preview, err = NewPreviewWidget(guiApp)
	if err != nil {
		return nil, fmt.Errorf("failed to create preview widget: %w", err)
	}

	// Create controls panel
	guiApp.controls, err = NewControlsWidget(guiApp)
	if err != nil {
		return nil, fmt.Errorf("failed to create controls widget: %w", err)
	}

	// Create palette widget
	guiApp.palette, err = NewPaletteWidget(guiApp)
	if err != nil {
		return nil, fmt.Errorf("failed to create palette widget: %w", err)
	}

	// Create toolbar
	guiApp.toolbar, err = NewToolbarWidget(guiApp)
	if err != nil {
		return nil, fmt.Errorf("failed to create toolbar widget: %w", err)
	}

	// Initialize animation panel (hidden by default)
	guiApp.animLabel = widget.NewLabel("Frame: 0/0")
	guiApp.animSlider = widget.NewSlider(0, 0)
	guiApp.animSlider.Step = 1
	guiApp.animSlider.OnChanged = func(val float64) {
		guiApp.selectAnimFrame(int(val))
	}
	guiApp.animPanel = container.NewVBox(
		widget.NewLabel("Animation Frames"),
		guiApp.animSlider,
		guiApp.animLabel,
	)
	guiApp.animPanel.Hide()

	// Start debounced conversion goroutine
	go guiApp.conversionWorker()

	return guiApp, nil
}

// BuildLayout creates and returns the main window layout.
// Layout: Split panel with source/CPC images (left/right), controls below, toolbar top, status bottom.
func (app *Application) BuildLayout() fyne.CanvasObject {
	// Main split: image views (left) | palette + controls (right)
	imageContainer := container.NewHSplit(
		app.preview.GetSourceContainer(),
		app.preview.GetCpcContainer(),
	)
	imageContainer.SetOffset(0.5) // 50/50 split

	// Right panel: palette + controls + animation (scrollable)
	rightPanel := container.NewVScroll(container.NewVBox(
		app.palette.GetWidget(),
		widget.NewSeparator(),
		app.controls.GetWidget(),
		widget.NewSeparator(),
		app.animPanel,
	))

	// Main content: images (left) | right panel
	mainContent := container.NewHSplit(
		imageContainer,
		rightPanel,
	)
	mainContent.SetOffset(0.7) // 70% for images, 30% for controls

	// Complete layout: toolbar / main content / status bar
	return container.NewBorder(
		app.toolbar.GetWidget(), // top
		app.statusBar,           // bottom
		nil,                     // left
		nil,                     // right
		mainContent,             // center
	)
}

// TriggerConversion schedules a conversion with debouncing.
// Safe to call from the UI thread — never blocks.
func (app *Application) TriggerConversion() {
	app.debounceMu.Lock()
	defer app.debounceMu.Unlock()

	// Cancel any in-flight conversion so the worker abandons it quickly
	if app.cancelConv != nil {
		app.cancelConv()
	}

	if app.debounceTimer != nil {
		app.debounceTimer.Stop()
	}

	app.debounceTimer = time.AfterFunc(30*time.Millisecond, func() {
		ctx, cancel := context.WithCancel(context.Background())

		app.debounceMu.Lock()
		app.cancelConv = cancel
		app.debounceMu.Unlock()

		// Non-blocking send: if channel is full the worker will drain it
		select {
		case app.conversionCh <- convRequest{ctx: ctx}:
		default:
			// Channel full — worker will pick up the existing request;
			// drain and replace with our newer context
			select {
			case <-app.conversionCh:
			default:
			}
			app.conversionCh <- convRequest{ctx: ctx}
		}
	})
}

// conversionWorker runs the debounced conversion pipeline in a background goroutine.
// It owns all mutable conversion state (cpcImage, params during conversion).
// Before starting a conversion, it drains any newer request that arrived while
// the previous conversion was running, so we always process the latest state.
func (app *Application) conversionWorker() {
	for req := range app.conversionCh {
		// Drain: if a newer request arrived while we were busy, skip to it
		drained := req
		for {
			select {
			case newer := <-app.conversionCh:
				drained = newer
			default:
				goto run
			}
		}
	run:
		if drained.ctx.Err() == nil {
			app.performConversion(drained.ctx)
		}
	}
}

// performConversion runs the actual conversion pipeline and updates the display.
// It is only called from conversionWorker, so it owns all conversion state.
func (app *Application) performConversion(ctx context.Context) {
	app.sourceMu.Lock()
	srcImg := app.sourceImg
	app.sourceMu.Unlock()

	if srcImg == nil {
		app.cpcDisplay.Store(CreateDemoCpcImage(320, 200))
		fyne.Do(func() {
			app.statusBar.SetText("Demo mode - load an image to convert")
			app.preview.RefreshCpc()
		})
		return
	}

	fyne.Do(func() {
		app.statusBar.SetText("Converting...")
	})
	startTime := time.Now()

	var numColors int
	var convErr interface{}

	func() {
		defer func() {
			convErr = recover()
		}()

		// Recreate BitmapCpc if dimensions or CPC+ mode changed
		bmp := app.cpcImage.BitmapCpc
		if bmp == nil || bmp.NumCol != app.params.NumCols || bmp.NumLig != app.params.NumLines || bmp.CpcPlus != app.params.CpcPlus {
			cpcW := app.params.GetScreenWidth()
			cpcH := app.params.GetScreenHeight()
			app.cpcImage.BitmapCpc = render.NewBitmapCpcWithParams(app.params.NumCols, app.params.NumLines, app.params.CpcPlus)
			app.cpcImage.DisplayBmp = bitmap.NewDirectBitmap(cpcW, cpcH)
			app.cpcImage.Width = cpcW
			app.cpcImage.Height = cpcH
		}

		resizedImg := app.getResizeBitmap()

		// Check for cancellation before the expensive conversion
		if ctx.Err() != nil {
			return
		}

		numColors = convert.Convert(resizedImg, app.cpcImage, app.params, true)
		copy(app.params.Palette[:], app.cpcImage.BitmapCpc.Palette[:])

		// Check for cancellation before rendering
		if ctx.Err() != nil {
			return
		}

		app.cpcDisplay.Store(app.renderCpcToRGBA())
	}()

	// If cancelled, silently abandon — a newer conversion is coming
	if ctx.Err() != nil {
		return
	}

	if convErr != nil {
		app.cpcDisplay.Store(CreateDemoCpcImage(
			app.params.GetScreenWidth(),
			app.params.GetScreenHeight(),
		))
		fyne.Do(func() {
			app.statusBar.SetText(fmt.Sprintf("Demo mode - conversion error: %v", convErr))
		})
	} else {
		// Count how many pens are actually used in the output
		usedPens := app.countUsedPens()
		elapsed := time.Since(startTime)
		// Display actual CPC resolution (pixels per byte depends on mode)
		ppb := 2 // Mode 0: 2 pixels/byte
		switch app.params.VirtualMode {
		case 1:
			ppb = 4
		case 2:
			ppb = 8
		}
		cpcW := app.params.NumCols * ppb
		cpcH := app.params.NumLines
		statusText := fmt.Sprintf("Mode %d, %dx%d, %d colors found, %d pens used, %.1fms",
			app.params.VirtualMode,
			cpcW,
			cpcH,
			numColors,
			usedPens,
			float64(elapsed.Nanoseconds())/1e6,
		)
		fyne.Do(func() {
			app.statusBar.SetText(statusText)
		})
	}

	fyne.Do(func() {
		app.palette.Refresh()
		app.preview.RefreshCpc()
	})
}

// countUsedPens counts how many distinct pen indices appear in the CPC screen buffer.
func (app *Application) countUsedPens() int {
	if app.cpcImage == nil || app.cpcImage.BitmapCpc == nil {
		return 0
	}

	b := app.cpcImage.BitmapCpc
	used := [16]bool{}
	height := b.NumLig << 1

	for y := 0; y < height; y += 2 {
		mode := b.VirtualMode
		if mode >= 5 {
			mode = 1
		} else if mode >= 3 {
			if (y & 2) == 0 {
				mode = b.VirtualMode - 2
			} else {
				mode = b.VirtualMode - 3
			}
		}

		cpcAddr := cpc.CpcAddress(y, b.NumCol, b.NumLig)

		for x := 0; x < b.NumCol; x++ {
			addr := cpcAddr + x
			if addr >= len(b.ScreenData) {
				continue
			}
			pixByte := b.ScreenData[addr]

			switch mode {
			case 0:
				p0 := int(pixByte>>7) + int((pixByte&0x20)>>3) + int((pixByte&0x08)>>2) + int((pixByte&0x02)<<2)
				p1 := int((pixByte&0x40)>>6) + int((pixByte&0x10)>>2) + int((pixByte&0x04)>>1) + int((pixByte&0x01)<<3)
				if p0 < 16 {
					used[p0] = true
				}
				if p1 < 16 {
					used[p1] = true
				}
			case 1:
				p0 := int((pixByte>>7)&1) + int((pixByte>>2)&2)
				p1 := int((pixByte>>6)&1) + int((pixByte>>1)&2)
				p2 := int((pixByte>>5)&1) + int((pixByte>>0)&2)
				p3 := int((pixByte>>4)&1) + int((pixByte<<1)&2)
				if p0 < 16 {
					used[p0] = true
				}
				if p1 < 16 {
					used[p1] = true
				}
				if p2 < 16 {
					used[p2] = true
				}
				if p3 < 16 {
					used[p3] = true
				}
			case 2:
				for i := 7; i >= 0; i-- {
					p := int(pixByte>>uint(i)) & 1
					used[p] = true
				}
			}
		}
	}

	count := 0
	for _, u := range used {
		if u {
			count++
		}
	}
	return count
}

// getResizeBitmap resizes the source image to CPC target dimensions.
// This matches the C# GetResizeBitmap function behavior.
func (app *Application) getResizeBitmap() *bitmap.DirectBitmap {
	cpcWidth := app.params.GetScreenWidth()
	cpcHeight := app.params.GetScreenHeight()

	// Create new bitmap at CPC dimensions
	resized := bitmap.NewDirectBitmap(cpcWidth, cpcHeight)

	// Fill with background color (pen 0 from palette)
	bgColorIndex := app.params.Palette[0]
	var bgColor bitmap.RgbColor
	if app.params.CpcPlus {
		bgColor = cpc.GetColorFromCpcPlus(bgColorIndex)
	} else {
		if bgColorIndex >= 0 && bgColorIndex < 27 {
			bgColor = cpc.CpcRgbPalette[bgColorIndex]
		} else {
			bgColor = bitmap.NewRgbColor(0, 0, 0) // Black default
		}
	}

	for y := 0; y < cpcHeight; y++ {
		for x := 0; x < cpcWidth; x++ {
			resized.SetPixelColor(x, y, bgColor)
		}
	}

	// Use cropped source if crop is enabled
	srcBitmap := app.GetCroppedSource()

	// Calculate scaling based on size mode
	srcWidth := srcBitmap.Width()
	srcHeight := srcBitmap.Height()

	switch app.params.SMode {
	case convert.Fit:
		// Stretch to fill entire CPC screen
		app.scaleImage(srcBitmap, resized, 0, 0, cpcWidth, cpcHeight)

	case convert.KeepSmaller:
		// Keep aspect ratio, fit within bounds (letterbox/pillarbox)
		ratio := float64(srcWidth*cpcHeight) / float64(srcHeight*cpcWidth)
		if ratio < 1.0 {
			// Source is narrower - add pillarbox
			newW := int(float64(cpcWidth) * ratio)
			offsetX := (cpcWidth - newW) / 2
			app.scaleImage(srcBitmap, resized, offsetX, 0, newW, cpcHeight)
		} else {
			// Source is taller - add letterbox
			newH := int(float64(cpcHeight) / ratio)
			offsetY := (cpcHeight - newH) / 2
			app.scaleImage(srcBitmap, resized, 0, offsetY, cpcWidth, newH)
		}

	case convert.KeepLarger:
		// Keep aspect ratio, fill entire screen (crop edges)
		ratio := float64(srcWidth*cpcHeight) / float64(srcHeight*cpcWidth)
		if ratio < 1.0 {
			// Source is narrower - crop top/bottom
			newH := int(float64(cpcHeight) / ratio)
			offsetY := (cpcHeight - newH) / 2
			app.scaleImage(srcBitmap, resized, 0, offsetY, cpcWidth, newH)
		} else {
			// Source is wider - crop left/right
			newW := int(float64(cpcWidth) * ratio)
			offsetX := (cpcWidth - newW) / 2
			app.scaleImage(srcBitmap, resized, offsetX, 0, newW, cpcHeight)
		}

	case convert.UserSize:
		// User-specified size and position
		app.scaleImage(srcBitmap, resized, app.params.PosX, app.params.PosY, app.params.SizeX, app.params.SizeY)

	default:
		// Default to Fit mode
		app.scaleImage(srcBitmap, resized, 0, 0, cpcWidth, cpcHeight)
	}

	return resized
}

// scaleImage scales the source image into the destination bitmap using nearest-neighbor interpolation.
func (app *Application) scaleImage(src *bitmap.DirectBitmap, dest *bitmap.DirectBitmap, dstX, dstY, dstW, dstH int) {
	srcWidth := src.Width()
	srcHeight := src.Height()

	// Nearest-neighbor scaling
	for dy := 0; dy < dstH; dy++ {
		destY := dstY + dy
		if destY < 0 || destY >= dest.Height() {
			continue
		}

		srcY := dy * srcHeight / dstH
		if srcY >= srcHeight {
			srcY = srcHeight - 1
		}

		for dx := 0; dx < dstW; dx++ {
			destX := dstX + dx
			if destX < 0 || destX >= dest.Width() {
				continue
			}

			srcX := dx * srcWidth / dstW
			if srcX >= srcWidth {
				srcX = srcWidth - 1
			}

			color := src.GetPixelColor(srcX, srcY)
			dest.SetPixelColor(destX, destY, color)
		}
	}
}

// renderCpcToRGBA converts the CPC screen buffer to an RGBA image for display.
func (app *Application) renderCpcToRGBA() *image.RGBA {
	defer func() {
		if r := recover(); r != nil {
			// Render failed, return demo image
			app.cpcDisplay.Store(CreateDemoCpcImage(640, 400))
		}
	}()

	if app.cpcImage.BitmapCpc == nil {
		// Return demo image if no CPC data
		return CreateDemoCpcImage(
			app.params.GetScreenWidth(),
			app.params.GetScreenHeight(),
		)
	}

	// Use the render package to convert CPC screen data to RGBA
	return app.cpcImage.BitmapCpc.RenderToRGBA()
}

// GetParams returns the current conversion parameters.
func (app *Application) GetParams() *convert.Settings {
	return app.params
}

// SetSourceImage sets a new source image and triggers conversion.
func (app *Application) SetSourceImage(img *bitmap.DirectBitmap) {
	app.sourceMu.Lock()
	app.sourceImg = img
	app.sourceMu.Unlock()

	if img != nil {
		app.preview.SetSourceImage(img.ToRGBA())
		app.TriggerConversion()
	} else {
		app.preview.SetSourceImage(nil)
	}
}

// GetCroppedSource returns the cropped region of the source image if crop is enabled,
// otherwise returns the full source image.
func (app *Application) GetCroppedSource() *bitmap.DirectBitmap {
	if !app.cropEnabled || app.sourceImg == nil {
		return app.sourceImg
	}

	// Normalize coordinates so x1<x2, y1<y2
	x1, x2 := app.cropX1, app.cropX2
	y1, y2 := app.cropY1, app.cropY2
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}

	// Skip if crop region is essentially the full image or too small
	if (x2-x1) < 0.01 || (y2-y1) < 0.01 {
		return app.sourceImg
	}

	srcW := app.sourceImg.Width()
	srcH := app.sourceImg.Height()

	// Convert normalized coords to pixel coords
	px1 := int(x1 * float64(srcW))
	py1 := int(y1 * float64(srcH))
	px2 := int(x2 * float64(srcW))
	py2 := int(y2 * float64(srcH))

	// Clamp
	if px1 < 0 {
		px1 = 0
	}
	if py1 < 0 {
		py1 = 0
	}
	if px2 > srcW {
		px2 = srcW
	}
	if py2 > srcH {
		py2 = srcH
	}

	cropW := px2 - px1
	cropH := py2 - py1
	if cropW <= 0 || cropH <= 0 {
		return app.sourceImg
	}

	cropped := bitmap.NewDirectBitmap(cropW, cropH)
	for y := 0; y < cropH; y++ {
		for x := 0; x < cropW; x++ {
			c := app.sourceImg.GetPixelColor(px1+x, py1+y)
			cropped.SetPixelColor(x, y, c)
		}
	}
	return cropped
}

// GetCpcDisplay returns the current CPC display image for preview.
// Lock-free — safe to call from Fyne's render thread.
func (app *Application) GetCpcDisplay() *image.RGBA {
	return app.cpcDisplay.Load()
}

// UpdatePalette updates the palette display when colors change.
func (app *Application) UpdatePalette() {
	app.palette.Refresh()
	app.TriggerConversion()
}

// RefreshControls syncs all UI controls to match the current params.
func (app *Application) RefreshControls() {
	app.controls.RefreshFromParams()
}

// SetStatus updates the status bar text. Safe to call from any goroutine.
func (app *Application) SetStatus(status string) {
	fyne.Do(func() {
		app.statusBar.SetText(status)
	})
}

// ShowProgress shows a modal progress dialog that blocks UI interaction.
// Returns a function to update progress (0.0-1.0) and a function to dismiss.
func (app *Application) ShowProgress(title string) (update func(progress float64, status string), dismiss func()) {
	bar := widget.NewProgressBar()
	label := widget.NewLabel(title)
	content := container.NewVBox(label, bar)

	d := dialog.NewCustomWithoutButtons(title, content, app.window)

	fyne.Do(func() {
		d.Show()
		d.Resize(fyne.NewSize(400, 100))
	})

	update = func(progress float64, status string) {
		fyne.Do(func() {
			bar.SetValue(progress)
			label.SetText(status)
		})
	}

	dismiss = func() {
		fyne.Do(func() {
			d.Hide()
		})
	}

	return update, dismiss
}

// syncCropToParams copies the GUI crop state into params for project save.
func (app *Application) syncCropToParams() {
	app.params.CropEnabled = app.cropEnabled
	app.params.CropLockAspect = app.cropLockAspect
	app.params.CropX1 = app.cropX1
	app.params.CropY1 = app.cropY1
	app.params.CropX2 = app.cropX2
	app.params.CropY2 = app.cropY2
}

// syncCropFromParams restores the GUI crop state from loaded params.
func (app *Application) syncCropFromParams() {
	app.cropEnabled = app.params.CropEnabled
	app.cropLockAspect = app.params.CropLockAspect
	app.cropX1 = app.params.CropX1
	app.cropY1 = app.params.CropY1
	app.cropX2 = app.params.CropX2
	app.cropY2 = app.params.CropY2
	if app.cropLockAspect {
		app.cropAspect = float64(app.params.GetScreenWidth()) / float64(app.params.GetScreenHeight())
	}
}

// SetAnimationFrames stores loaded GIF frames and shows the animation panel.
func (app *Application) SetAnimationFrames(frames []*bitmap.DirectBitmap, delays []time.Duration) {
	app.animFrames = frames
	app.animDelays = delays
	app.animIndex = 0

	if len(frames) > 1 {
		app.animSlider.Max = float64(len(frames) - 1)
		app.animSlider.SetValue(0)
		app.animLabel.SetText(fmt.Sprintf("Frame: 1/%d", len(frames)))
		app.animPanel.Show()
	} else {
		app.animPanel.Hide()
	}

	// Show the first frame as the source image.
	if len(frames) > 0 {
		app.SetSourceImage(frames[0])
	}
}

// selectAnimFrame switches to the animation frame at the given index.
func (app *Application) selectAnimFrame(index int) {
	if index < 0 || index >= len(app.animFrames) {
		return
	}
	app.animIndex = index
	app.animLabel.SetText(fmt.Sprintf("Frame: %d/%d", index+1, len(app.animFrames)))
	app.SetSourceImage(app.animFrames[index])
}