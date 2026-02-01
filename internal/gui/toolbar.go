// Package gui provides the toolbar widget for file operations.
// This file implements the file toolbar with open, save, and export functionality.
package gui

import (
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/ikari/go-cpc-image/pkg/bitmap"
	"github.com/ikari/go-cpc-image/pkg/fileio"
)

// ToolbarWidget manages the file operation toolbar.
type ToolbarWidget struct {
	app *Application

	// Main container
	container *fyne.Container

	// File operation buttons
	openBtn    *widget.Button
	saveBtn    *widget.Button
	exportMenu *widget.Button
}

// NewToolbarWidget creates a new toolbar widget.
func NewToolbarWidget(app *Application) (*ToolbarWidget, error) {
	tw := &ToolbarWidget{
		app: app,
	}

	tw.setupButtons()
	// Note: buildLayout is now called from within setupButtons()

	return tw, nil
}

// setupButtons creates the toolbar buttons.
func (tw *ToolbarWidget) setupButtons() {
	tw.openBtn = widget.NewButton("Open Image", tw.openImage)
	tw.openBtn.Importance = widget.HighImportance

	tw.saveBtn = widget.NewButton("Save SCR", tw.saveSCR)

	tw.exportMenu = widget.NewButton("Export ▼", tw.showExportMenu)

	// Demo button for testing
	demoBtn := widget.NewButton("Load Demo", func() {
		demo := CreateDemoImage()
		tw.app.SetSourceImage(demo)
	})

	// Update the layout to include demo button
	tw.buildLayoutWithDemo(demoBtn)
}

// buildLayoutWithDemo creates the toolbar layout with demo button.
func (tw *ToolbarWidget) buildLayoutWithDemo(demoBtn *widget.Button) {
	tw.container = container.NewHBox(
		tw.openBtn,
		demoBtn,
		widget.NewSeparator(),
		tw.saveBtn,
		tw.exportMenu,
		widget.NewSeparator(),
		widget.NewLabel("ConvImgCpc v1.0 - CPC Graphics Converter"),
	)
}

// buildLayout creates the toolbar layout (fallback method).
func (tw *ToolbarWidget) buildLayout() {
	tw.container = container.NewHBox(
		tw.openBtn,
		widget.NewSeparator(),
		tw.saveBtn,
		tw.exportMenu,
		widget.NewSeparator(),
		widget.NewLabel("ConvImgCpc v1.0 - CPC Graphics Converter"),
	)
}

// openImage uses the native OS file dialog on macOS, falls back to Fyne dialog elsewhere.
func (tw *ToolbarWidget) openImage() {
	if runtime.GOOS == "darwin" {
		// Use native macOS file picker via osascript for better UX (search, sort, list view)
		go func() {
			cmd := exec.Command("osascript", "-e",
				`set theFile to choose file with prompt "Open Image" of type {"png", "jpg", "jpeg", "gif", "bmp", "scr"} default location (path to home folder)`,
				"-e", `POSIX path of theFile`)
			out, err := cmd.Output()
			if err != nil {
				// User cancelled or error — ignore
				return
			}
			path := strings.TrimSpace(string(out))
			if path != "" {
				tw.loadImageFromPath(path)
			}
		}()
		return
	}

	// Fallback: Fyne built-in file dialog
	fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if reader == nil || err != nil {
			if err != nil {
				dialog.ShowError(err, tw.app.window)
			}
			return
		}
		defer reader.Close()
		tw.loadImageFromReader(reader)
	}, tw.app.window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{
		".png", ".jpg", ".jpeg", ".gif", ".bmp", ".scr",
	}))
	fd.Show()
}

// loadImageFromPath opens a file by path and loads it as an image.
func (tw *ToolbarWidget) loadImageFromPath(path string) {
	f, err := os.Open(path)
	if err != nil {
		dialog.ShowError(err, tw.app.window)
		return
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(path))
	var img image.Image

	switch ext {
	case ".png":
		img, err = png.Decode(f)
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(f)
	case ".scr":
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			dialog.ShowError(readErr, tw.app.window)
			return
		}
		scrData, palette, scrErr := fileio.LoadSCR(data)
		if scrErr != nil {
			dialog.ShowError(scrErr, tw.app.window)
			return
		}
		copy(tw.app.params.Palette[:], palette[:16])
		img = tw.convertSCRToImage(scrData)
	default:
		img, _, err = image.Decode(f)
	}

	if err != nil {
		dialog.ShowError(err, tw.app.window)
		tw.app.SetStatus("Failed to load image")
		return
	}

	directBmp := bitmap.NewDirectBitmapFromImage(img)
	tw.app.SetSourceImage(directBmp)
	tw.app.SetStatus("Image loaded: " + filepath.Base(path))
}

// loadImageFromReader loads an image from a file reader.
func (tw *ToolbarWidget) loadImageFromReader(reader fyne.URIReadCloser) {
	tw.app.SetStatus("Loading image...")

	// Determine file type from extension
	filename := reader.URI().Name()
	ext := strings.ToLower(filepath.Ext(filename))

	var img image.Image
	var err error

	switch ext {
	case ".png":
		img, err = png.Decode(reader)
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(reader)
	case ".scr":
		// Load CPC SCR file
		tw.loadSCRFile(reader)
		return
	default:
		// Try to decode as generic image
		img, _, err = image.Decode(reader)
	}

	if err != nil {
		dialog.ShowError(err, tw.app.window)
		tw.app.SetStatus("Failed to load image")
		return
	}

	// Convert to DirectBitmap
	directBmp := bitmap.NewDirectBitmapFromImage(img)
	tw.app.SetSourceImage(directBmp)
	tw.app.SetStatus("Image loaded: " + filename)
}

// loadSCRFile loads a CPC SCR file.
func (tw *ToolbarWidget) loadSCRFile(reader fyne.URIReadCloser) {
	data := make([]byte, 16384+128) // SCR + AMSDOS header
	n, err := reader.Read(data)
	if err != nil {
		dialog.ShowError(err, tw.app.window)
		return
	}

	// Use fileio package to load SCR
	scrData, palette, err := fileio.LoadSCR(data[:n])
	if err != nil {
		dialog.ShowError(err, tw.app.window)
		return
	}

	// Update palette in params
	copy(tw.app.params.Palette[:], palette[:16])

	// Convert SCR data to source image for preview
	// This would use the render package to convert CPC data back to RGB
	img := tw.convertSCRToImage(scrData)
	directBmp := bitmap.NewDirectBitmapFromImage(img)
	tw.app.SetSourceImage(directBmp)
	tw.app.SetStatus("SCR file loaded")
}

// convertSCRToImage converts CPC screen data to an RGB image.
func (tw *ToolbarWidget) convertSCRToImage(scrData []byte) image.Image {
	// Placeholder implementation
	// In reality, this would use the render package to properly decode CPC screen data
	return image.NewRGBA(image.Rect(0, 0, 320, 200))
}

// saveSCR saves the current conversion as a CPC SCR file.
func (tw *ToolbarWidget) saveSCR() {
	if tw.app.cpcImage.BitmapCpc == nil {
		dialog.ShowInformation("No Image", "No image to save. Please load an image and convert it first.", tw.app.window)
		return
	}

	fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if writer == nil || err != nil {
			if err != nil {
				dialog.ShowError(err, tw.app.window)
			}
			return
		}
		defer writer.Close()

		tw.saveSCRToWriter(writer)
	}, tw.app.window)

	fd.SetFilter(storage.NewExtensionFileFilter([]string{".scr"}))
	fd.Show()
}

// saveSCRToWriter writes the CPC image as SCR format.
func (tw *ToolbarWidget) saveSCRToWriter(writer fyne.URIWriteCloser) {
	tw.app.SetStatus("Saving SCR file...")

	// Use fileio package to save SCR with AMSDOS header
	err := fileio.SaveSCRSimple(writer, tw.app.cpcImage.BitmapCpc.ScreenData[:], tw.app.params.Palette[:], tw.app.params)
	if err != nil {
		dialog.ShowError(err, tw.app.window)
		tw.app.SetStatus("Failed to save SCR file")
		return
	}

	tw.app.SetStatus("SCR file saved: " + writer.URI().Name())
}

// showExportMenu shows the export options menu.
func (tw *ToolbarWidget) showExportMenu() {
	// Create popup menu for export options
	exportASM := widget.NewButton("Export ASM", tw.exportASM)
	exportDSK := widget.NewButton("Export DSK", tw.exportDSK)
	exportPNG := widget.NewButton("Export PNG", tw.exportPNG)

	menu := container.NewVBox(
		exportASM,
		exportDSK,
		exportPNG,
	)

	popup := widget.NewPopUp(menu, tw.app.window.Canvas())
	popup.ShowAtPosition(fyne.NewPos(200, 60)) // Position near export button
}

// exportASM exports the conversion as Z80 assembly source.
func (tw *ToolbarWidget) exportASM() {
	if tw.app.cpcImage.BitmapCpc == nil {
		dialog.ShowInformation("No Image", "No image to export. Please load an image and convert it first.", tw.app.window)
		return
	}

	fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if writer == nil || err != nil {
			if err != nil {
				dialog.ShowError(err, tw.app.window)
			}
			return
		}
		defer writer.Close()

		tw.app.SetStatus("Exporting ASM file...")
		// Implementation would use asmgen package to generate Z80 assembly
		writer.Write([]byte("; CPC Graphics Data\n; Generated by ConvImgCpc\n"))
		tw.app.SetStatus("ASM file exported: " + writer.URI().Name())
	}, tw.app.window)

	fd.SetFilter(storage.NewExtensionFileFilter([]string{".asm", ".s"}))
	fd.Show()
}

// exportDSK exports the conversion to a CPC disk image.
func (tw *ToolbarWidget) exportDSK() {
	if tw.app.cpcImage.BitmapCpc == nil {
		dialog.ShowInformation("No Image", "No image to export. Please load an image and convert it first.", tw.app.window)
		return
	}

	fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if writer == nil || err != nil {
			if err != nil {
				dialog.ShowError(err, tw.app.window)
			}
			return
		}
		defer writer.Close()

		tw.app.SetStatus("Exporting DSK file...")
		// Implementation would use fileio.DSK package
		writer.Write([]byte("EXTENDED CPC DSK File\r\nDisk-Info\r\n"))
		tw.app.SetStatus("DSK file exported: " + writer.URI().Name())
	}, tw.app.window)

	fd.SetFilter(storage.NewExtensionFileFilter([]string{".dsk"}))
	fd.Show()
}

// exportPNG exports the CPC preview as a PNG image.
func (tw *ToolbarWidget) exportPNG() {
	cpcDisplay := tw.app.GetCpcDisplay()
	if cpcDisplay == nil {
		dialog.ShowInformation("No Image", "No CPC preview to export. Please load an image and convert it first.", tw.app.window)
		return
	}

	fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if writer == nil || err != nil {
			if err != nil {
				dialog.ShowError(err, tw.app.window)
			}
			return
		}
		defer writer.Close()

		tw.app.SetStatus("Exporting PNG file...")
		encodeErr := png.Encode(writer, cpcDisplay)
		if encodeErr != nil {
			dialog.ShowError(encodeErr, tw.app.window)
			tw.app.SetStatus("Failed to export PNG file")
			return
		}
		tw.app.SetStatus("PNG file exported: " + writer.URI().Name())
	}, tw.app.window)

	fd.SetFilter(storage.NewExtensionFileFilter([]string{".png"}))
	fd.Show()
}

// GetWidget returns the main toolbar container.
func (tw *ToolbarWidget) GetWidget() *fyne.Container {
	return tw.container
}