// Package gui provides the toolbar widget for file operations.
// This file implements the file toolbar with open, save, and export functionality.
package gui

import (
	"fmt"
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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/ikari/go-cpc-image/pkg/bitmap"
	"github.com/ikari/go-cpc-image/pkg/cpc"
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

	// Drawing tool buttons
	toolBtns [4]*widget.Button
	undoBtn  *widget.Button
	redoBtn  *widget.Button
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

	// Drawing tool buttons
	toolNames := [4]string{"Pencil", "Line", "Rect", "Fill"}
	for i := 0; i < 4; i++ {
		idx := i
		tw.toolBtns[i] = widget.NewButton(toolNames[i], func() {
			tw.selectTool(DrawTool(idx))
		})
	}
	tw.toolBtns[0].Importance = widget.HighImportance // pencil selected by default

	// Undo / Redo
	tw.undoBtn = widget.NewButton("Undo", func() {
		if tw.app.editor != nil {
			tw.app.editor.Undo()
		}
	})
	tw.redoBtn = widget.NewButton("Redo", func() {
		if tw.app.editor != nil {
			tw.app.editor.Redo()
		}
	})

	loadAnimBtn := widget.NewButton("Load Animation", tw.loadAnimation)

	saveProjectBtn := widget.NewButton("Save Project", tw.saveProject)
	loadProjectBtn := widget.NewButton("Open Project", tw.loadProject)

	// Update the layout to include demo and animation buttons
	tw.buildLayoutWithDemo(demoBtn, loadAnimBtn, saveProjectBtn, loadProjectBtn)
}

// selectTool sets the active drawing tool and updates button highlights.
func (tw *ToolbarWidget) selectTool(t DrawTool) {
	if tw.app.editor != nil {
		tw.app.editor.Tool = t
	}
	for i, btn := range tw.toolBtns {
		if DrawTool(i) == t {
			btn.Importance = widget.HighImportance
		} else {
			btn.Importance = widget.MediumImportance
		}
		btn.Refresh()
	}
}

// buildLayoutWithDemo creates the toolbar layout with demo and animation buttons.
func (tw *ToolbarWidget) buildLayoutWithDemo(demoBtn, loadAnimBtn, saveProjectBtn, loadProjectBtn *widget.Button) {
	tw.container = container.NewHBox(
		tw.openBtn,
		loadProjectBtn,
		demoBtn,
		loadAnimBtn,
		layout.NewSpacer(),
		widget.NewSeparator(),
		tw.saveBtn,
		saveProjectBtn,
		tw.exportMenu,
		layout.NewSpacer(),
		widget.NewSeparator(),
		tw.toolBtns[0], tw.toolBtns[1], tw.toolBtns[2], tw.toolBtns[3],
		layout.NewSpacer(),
		widget.NewSeparator(),
		tw.undoBtn, tw.redoBtn,
		layout.NewSpacer(),
		widget.NewLabel("Go CPC image! v1.0.0"),
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
		widget.NewLabel("Go CPC image! v1.0.0"),
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
		tw.saveSCRToWriter(writer)
	}, tw.app.window)

	fd.SetFilter(storage.NewExtensionFileFilter([]string{".scr"}))
	fd.Show()
}

// saveSCRToWriter writes the CPC image as SCR format.
func (tw *ToolbarWidget) saveSCRToWriter(writer fyne.URIWriteCloser) {
	tw.app.SetStatus("Saving SCR file...")

	// Get the file path and close the writer so SaveSCR can create the file itself.
	filePath := writer.URI().Path()
	writer.Close()

	// Calculate bitmap size: standard 16KB or overscan full buffer.
	bitmapSize := cpc.BitmapSize(tw.app.params.NumCols, tw.app.params.NumLines)
	if bitmapSize > 0x10000 {
		bitmapSize = 0x10000
	}

	// Build uint16 palette slice for SaveSCR.
	palette := make([]uint16, 16)
	for i := 0; i < 16; i++ {
		palette[i] = uint16(tw.app.params.Palette[i])
	}

	scrParams := fileio.SCRParams{
		WithPalette: tw.app.params.WithPalette,
		WithCode:    tw.app.params.WithCode,
		CPCPlus:     tw.app.params.CpcPlus,
		VirtualMode: tw.app.params.VirtualMode,
	}

	_, err := fileio.SaveSCR(filePath, tw.app.cpcImage.BitmapCpc.ScreenData[:], bitmapSize,
		fileio.PackNone, fileio.OutputBinary, scrParams, palette, nil)
	if err != nil {
		dialog.ShowError(err, tw.app.window)
		tw.app.SetStatus("Failed to save SCR file")
		return
	}

	tw.app.SetStatus("SCR file saved: " + filepath.Base(filePath))
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

// exportViaSaveSCR is a shared helper for ASM and DSK exports.
// It routes through fileio.SaveSCR which properly embeds display code
// (including overscan CRTC setup, palette, and mode switching routines).
func (tw *ToolbarWidget) exportViaSaveSCR(filePath string, format fileio.OutputFormat) error {
	bitmapSize := cpc.BitmapSize(tw.app.params.NumCols, tw.app.params.NumLines)
	if bitmapSize > 0x10000 {
		bitmapSize = 0x10000
	}

	palette := make([]uint16, 16)
	for i := 0; i < 16; i++ {
		palette[i] = uint16(tw.app.params.Palette[i])
	}

	scrParams := fileio.SCRParams{
		WithPalette: true,
		WithCode:    true,
		CPCPlus:     tw.app.params.CpcPlus,
		VirtualMode: tw.app.params.VirtualMode,
	}

	_, err := fileio.SaveSCR(filePath, tw.app.cpcImage.BitmapCpc.ScreenData[:], bitmapSize,
		fileio.PackNone, format, scrParams, palette, nil)
	return err
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
		asmPath := writer.URI().Path()
		writer.Close()

		tw.app.SetStatus("Exporting ASM file...")
		if exportErr := tw.exportViaSaveSCR(asmPath, fileio.OutputAssembler); exportErr != nil {
			dialog.ShowError(exportErr, tw.app.window)
			tw.app.SetStatus("Failed to export ASM file")
			return
		}
		tw.app.SetStatus("ASM file exported: " + filepath.Base(asmPath))
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
		dskPath := writer.URI().Path()
		writer.Close()

		tw.app.SetStatus("Exporting DSK file...")
		if exportErr := tw.exportViaSaveSCR(dskPath, fileio.OutputDSK); exportErr != nil {
			dialog.ShowError(exportErr, tw.app.window)
			tw.app.SetStatus("Failed to export DSK file")
			return
		}
		tw.app.SetStatus("DSK file exported: " + filepath.Base(dskPath))
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

// loadAnimation opens a file dialog filtered for .gif and loads all frames.
func (tw *ToolbarWidget) loadAnimation() {
	if runtime.GOOS == "darwin" {
		go func() {
			cmd := exec.Command("osascript", "-e",
				`set theFile to choose file with prompt "Load Animation" of type {"gif"} default location (path to home folder)`,
				"-e", `POSIX path of theFile`)
			out, err := cmd.Output()
			if err != nil {
				return
			}
			path := strings.TrimSpace(string(out))
			if path != "" {
				tw.loadGIFFromPath(path)
			}
		}()
		return
	}

	fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if reader == nil || err != nil {
			if err != nil {
				dialog.ShowError(err, tw.app.window)
			}
			return
		}
		defer reader.Close()
		// Get the file path from the URI
		path := reader.URI().Path()
		tw.loadGIFFromPath(path)
	}, tw.app.window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".gif"}))
	fd.Show()
}

// loadGIFFromPath loads an animated GIF and stores frames in the application.
func (tw *ToolbarWidget) loadGIFFromPath(path string) {
	tw.app.SetStatus("Loading animation...")

	frames, delays, err := fileio.LoadGIF(path)
	if err != nil {
		dialog.ShowError(err, tw.app.window)
		tw.app.SetStatus("Failed to load animation")
		return
	}

	tw.app.SetAnimationFrames(frames, delays)
	tw.app.SetStatus(fmt.Sprintf("Animation loaded: %s (%d frames)", filepath.Base(path), len(frames)))
}

// saveProject saves the current project state to a .gocpcimg file.
func (tw *ToolbarWidget) saveProject() {
	tw.app.sourceMu.Lock()
	src := tw.app.sourceImg
	tw.app.sourceMu.Unlock()

	if src == nil {
		dialog.ShowInformation("No Image", "No source image to save. Please load an image first.", tw.app.window)
		return
	}

	if runtime.GOOS == "darwin" {
		go func() {
			cmd := exec.Command("osascript", "-e",
				`set theFile to choose file name with prompt "Save Project" default name "project.gocpcimg" default location (path to home folder)`,
				"-e", `POSIX path of theFile`)
			out, err := cmd.Output()
			if err != nil {
				return
			}
			path := strings.TrimSpace(string(out))
			if path == "" {
				return
			}
			if !strings.HasSuffix(strings.ToLower(path), ".gocpcimg") {
				path += ".gocpcimg"
			}
			tw.app.sourceMu.Lock()
			srcImg := tw.app.sourceImg
			tw.app.sourceMu.Unlock()
			if srcImg == nil {
				return
			}
			if err := fileio.SaveProject(path, tw.app.params, srcImg.ToRGBA()); err != nil {
				dialog.ShowError(err, tw.app.window)
				tw.app.SetStatus("Failed to save project")
				return
			}
			tw.app.SetStatus("Project saved: " + filepath.Base(path))
		}()
		return
	}

	fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if writer == nil || err != nil {
			if err != nil {
				dialog.ShowError(err, tw.app.window)
			}
			return
		}
		path := writer.URI().Path()
		writer.Close()

		tw.app.sourceMu.Lock()
		srcImg := tw.app.sourceImg
		tw.app.sourceMu.Unlock()
		if srcImg == nil {
			return
		}
		if saveErr := fileio.SaveProject(path, tw.app.params, srcImg.ToRGBA()); saveErr != nil {
			dialog.ShowError(saveErr, tw.app.window)
			tw.app.SetStatus("Failed to save project")
			return
		}
		tw.app.SetStatus("Project saved: " + filepath.Base(path))
	}, tw.app.window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".gocpcimg"}))
	fd.Show()
}

// loadProject loads a project state from a .gocpcimg file.
func (tw *ToolbarWidget) loadProject() {
	if runtime.GOOS == "darwin" {
		go func() {
			cmd := exec.Command("osascript", "-e",
				`set theFile to choose file with prompt "Open Project" of type {"gocpcimg"} default location (path to home folder)`,
				"-e", `POSIX path of theFile`)
			out, err := cmd.Output()
			if err != nil {
				return
			}
			path := strings.TrimSpace(string(out))
			if path != "" {
				tw.loadProjectFromPath(path)
			}
		}()
		return
	}

	fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if reader == nil || err != nil {
			if err != nil {
				dialog.ShowError(err, tw.app.window)
			}
			return
		}
		path := reader.URI().Path()
		reader.Close()
		tw.loadProjectFromPath(path)
	}, tw.app.window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".gocpcimg"}))
	fd.Show()
}

// loadProjectFromPath loads a project from a file path.
func (tw *ToolbarWidget) loadProjectFromPath(path string) {
	tw.app.SetStatus("Loading project...")

	params, sourceRGBA, err := fileio.LoadProject(path)
	if err != nil {
		dialog.ShowError(err, tw.app.window)
		tw.app.SetStatus("Failed to load project")
		return
	}

	// Apply loaded params
	*tw.app.params = *params

	// Convert *image.RGBA to DirectBitmap and set as source
	directBmp := bitmap.NewDirectBitmapFromImage(sourceRGBA)
	tw.app.SetSourceImage(directBmp)

	// Refresh UI and trigger conversion
	tw.app.RefreshControls()
	tw.app.TriggerConversion()

	tw.app.SetStatus("Project loaded: " + filepath.Base(path))
}

// GetWidget returns the main toolbar container.
func (tw *ToolbarWidget) GetWidget() *fyne.Container {
	return tw.container
}