// Package gui provides the Fyne GUI implementation for ConvImgCpc.
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
	"fyne.io/fyne/v2/widget"

	"github.com/ikari/go-cpc-image/pkg/bitmap"
	"github.com/ikari/go-cpc-image/pkg/convert"
	"github.com/ikari/go-cpc-image/pkg/cpc"
	"github.com/ikari/go-cpc-image/pkg/render"
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
	params    *convert.Param
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
		params:       convert.NewDefaultParam(),
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

	// Right panel: palette + controls
	rightPanel := container.NewVBox(
		app.palette.GetWidget(),
		widget.NewSeparator(),
		app.controls.GetWidget(),
	)

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

	app.debounceTimer = time.AfterFunc(50*time.Millisecond, func() {
		app.debounceMu.Lock()
		ctx, cancel := context.WithCancel(context.Background())
		app.cancelConv = cancel
		app.debounceMu.Unlock()

		// Drain any stale request and send the new one
		select {
		case <-app.conversionCh:
		default:
		}
		app.conversionCh <- convRequest{ctx: ctx}
	})
}

// conversionWorker runs the debounced conversion pipeline in a background goroutine.
// It owns all mutable conversion state (cpcImage, params during conversion).
func (app *Application) conversionWorker() {
	for req := range app.conversionCh {
		app.performConversion(req.ctx)
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
		app.statusBar.SetText("Demo mode - load an image to convert")
		app.preview.RefreshCpc()
		return
	}

	app.statusBar.SetText("Converting...")
	startTime := time.Now()

	var numColors int
	var convErr interface{}

	func() {
		defer func() {
			convErr = recover()
		}()

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
		app.statusBar.SetText(fmt.Sprintf("Demo mode - conversion error: %v", convErr))
	} else {
		// Count how many pens are actually used in the output
		usedPens := app.countUsedPens()
		elapsed := time.Since(startTime)
		app.statusBar.SetText(fmt.Sprintf("Mode %d, %dx%d, %d colors found, %d pens used, %.1fms",
			app.params.VirtualMode,
			app.params.GetScreenWidth(),
			app.params.GetScreenHeight(),
			numColors,
			usedPens,
			float64(elapsed.Nanoseconds())/1e6,
		))
	}

	app.palette.Refresh()
	app.preview.RefreshCpc()
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

	// Calculate scaling based on size mode
	srcWidth := app.sourceImg.Width()
	srcHeight := app.sourceImg.Height()

	switch app.params.SMode {
	case convert.Fit:
		// Stretch to fill entire CPC screen
		app.scaleImage(resized, 0, 0, cpcWidth, cpcHeight)

	case convert.KeepSmaller:
		// Keep aspect ratio, fit within bounds (letterbox/pillarbox)
		ratio := float64(srcWidth*cpcHeight) / float64(srcHeight*cpcWidth)
		if ratio < 1.0 {
			// Source is narrower - add pillarbox
			newW := int(float64(cpcWidth) * ratio)
			offsetX := (cpcWidth - newW) / 2
			app.scaleImage(resized, offsetX, 0, newW, cpcHeight)
		} else {
			// Source is taller - add letterbox
			newH := int(float64(cpcHeight) / ratio)
			offsetY := (cpcHeight - newH) / 2
			app.scaleImage(resized, 0, offsetY, cpcWidth, newH)
		}

	case convert.KeepLarger:
		// Keep aspect ratio, fill entire screen (crop edges)
		ratio := float64(srcWidth*cpcHeight) / float64(srcHeight*cpcWidth)
		if ratio < 1.0 {
			// Source is narrower - crop top/bottom
			newH := int(float64(cpcHeight) / ratio)
			offsetY := (cpcHeight - newH) / 2
			app.scaleImage(resized, 0, offsetY, cpcWidth, newH)
		} else {
			// Source is wider - crop left/right
			newW := int(float64(cpcWidth) * ratio)
			offsetX := (cpcWidth - newW) / 2
			app.scaleImage(resized, offsetX, 0, newW, cpcHeight)
		}

	case convert.UserSize:
		// User-specified size and position
		app.scaleImage(resized, app.params.PosX, app.params.PosY, app.params.SizeX, app.params.SizeY)

	default:
		// Default to Fit mode
		app.scaleImage(resized, 0, 0, cpcWidth, cpcHeight)
	}

	return resized
}

// scaleImage scales the source image into the destination bitmap using nearest-neighbor interpolation.
func (app *Application) scaleImage(dest *bitmap.DirectBitmap, dstX, dstY, dstW, dstH int) {
	srcWidth := app.sourceImg.Width()
	srcHeight := app.sourceImg.Height()

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

			color := app.sourceImg.GetPixelColor(srcX, srcY)
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
func (app *Application) GetParams() *convert.Param {
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

// SetStatus updates the status bar text.
func (app *Application) SetStatus(status string) {
	app.statusBar.SetText(status)
}