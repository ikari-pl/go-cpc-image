// Package gui provides the palette widget for CPC color editing.
// This file implements the 16-pen palette editor with CPC color selection.
package gui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/ikari/go-cpc-image/pkg/bitmap"
	"github.com/ikari/go-cpc-image/pkg/cpc"
)

// PaletteWidget manages the CPC palette display and editing.
type PaletteWidget struct {
	app *Application

	// Main container
	container *widget.Card

	// Palette display - color swatches with labels below
	penSwatches   [16]*canvas.Rectangle
	penLabels     [16]*widget.Label
	penContainers [16]*fyne.Container

	// Controls
	autoOptimizeBtn *widget.Button
	lockAllBtn      *widget.Button
	unlockAllBtn    *widget.Button
}

// NewPaletteWidget creates a new palette widget.
func NewPaletteWidget(app *Application) (*PaletteWidget, error) {
	pw := &PaletteWidget{
		app: app,
	}

	pw.setupPaletteDisplay()
	pw.setupControls()
	pw.buildLayout()
	pw.refreshPalette()

	return pw, nil
}

// luminance returns perceived brightness (0-255) of a color.
func luminance(r, g, b uint8) float64 {
	return 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
}

// setupPaletteDisplay creates the 16 palette pen swatches with labels below.
func (pw *PaletteWidget) setupPaletteDisplay() {
	for i := 0; i < 16; i++ {
		penIndex := i // Capture for closure

		// Color swatch
		pw.penSwatches[i] = canvas.NewRectangle(color.RGBA{0, 0, 0, 255})
		pw.penSwatches[i].SetMinSize(fyne.NewSize(28, 22))
		pw.penSwatches[i].CornerRadius = 3

		// Small label below showing pen number
		pw.penLabels[i] = widget.NewLabel(fmt.Sprintf("%d", i))
		pw.penLabels[i].Alignment = fyne.TextAlignCenter
		pw.penLabels[i].TextStyle = fyne.TextStyle{Monospace: true}

		// Invisible button overlaying the swatch for click handling
		clickBtn := widget.NewButton("", func() {
			pw.editPenColor(penIndex)
		})
		clickBtn.Importance = widget.LowImportance

		// Stack: swatch with invisible button on top
		swatchWithClick := container.NewStack(
			pw.penSwatches[i],
			clickBtn,
		)

		// Border layout: label on bottom, swatch fills center — no gap
		pw.penContainers[i] = container.NewBorder(
			nil, pw.penLabels[i], nil, nil,
			swatchWithClick,
		)
	}
}

// setupControls creates the palette control buttons.
func (pw *PaletteWidget) setupControls() {
	pw.autoOptimizeBtn = widget.NewButton("Auto-Optimize", func() {
		pw.app.SetStatus("Auto-optimizing palette...")
		pw.app.TriggerConversion()
	})

	pw.lockAllBtn = widget.NewButton("Lock All", func() {
		for i := 0; i < 16; i++ {
			pw.app.params.LockState[i] = 1
		}
		pw.refreshPalette()
		pw.app.UpdatePalette()
	})

	pw.unlockAllBtn = widget.NewButton("Unlock All", func() {
		for i := 0; i < 16; i++ {
			pw.app.params.LockState[i] = 0
		}
		pw.refreshPalette()
		pw.app.UpdatePalette()
	})
}

// buildLayout creates the palette widget layout.
func (pw *PaletteWidget) buildLayout() {
	// Create grid of pen swatches: 8 columns x 2 rows
	paletteGrid := container.NewGridWithColumns(8)
	for i := 0; i < 16; i++ {
		paletteGrid.Add(pw.penContainers[i])
	}

	// Control buttons
	controls := container.NewHBox(
		pw.autoOptimizeBtn,
		widget.NewSeparator(),
		pw.lockAllBtn,
		pw.unlockAllBtn,
	)

	// Main palette container
	pw.container = widget.NewCard("Palette", "",
		container.NewVBox(
			paletteGrid,
			controls,
		),
	)
}

// editPenColor opens the color picker dialog for a specific pen.
func (pw *PaletteWidget) editPenColor(penIndex int) {
	colorPicker := pw.createCpcColorPicker(penIndex)

	dialog := dialog.NewCustom(
		fmt.Sprintf("Edit Pen %d Color", penIndex),
		"Close",
		colorPicker,
		pw.app.window,
	)

	dialog.Show()
}

// createCpcColorPicker creates a color picker showing the 27 CPC colors or 4096 CPC+ colors.
func (pw *PaletteWidget) createCpcColorPicker(penIndex int) *fyne.Container {
	if pw.app.params.CpcPlus {
		return pw.createCpcPlusColorPicker(penIndex)
	}
	return pw.createStandardCpcColorPicker(penIndex)
}

// createStandardCpcColorPicker creates a picker for the 27 standard CPC colors.
func (pw *PaletteWidget) createStandardCpcColorPicker(penIndex int) *fyne.Container {
	colorGrid := container.NewGridWithColumns(9)

	cpcPalette := cpc.GetCpcRgb()

	for colorIndex := 0; colorIndex < 27; colorIndex++ {
		colorValue := colorIndex
		rgbColor := cpcPalette[colorIndex]

		fyneColor := color.RGBA{
			R: uint8(rgbColor.R),
			G: uint8(rgbColor.V),
			B: uint8(rgbColor.B),
			A: 255,
		}

		colorRect := canvas.NewRectangle(fyneColor)
		colorRect.SetMinSize(fyne.NewSize(30, 30))

		colorBtn := widget.NewButton("", func() {
			pw.setPenColor(penIndex, colorValue)
		})

		colorContainer := container.NewStack(colorRect, colorBtn)
		colorGrid.Add(colorContainer)
	}

	return container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Select color for pen %d:", penIndex)),
		colorGrid,
	)
}

// createCpcPlusColorPicker creates a picker for CPC+ 4096 colors (simplified version).
func (pw *PaletteWidget) createCpcPlusColorPicker(penIndex int) *fyne.Container {
	colorGrid := container.NewGridWithColumns(16)

	for g := 0; g < 16; g++ {
		for r := 0; r < 16; r++ {
			if len(colorGrid.Objects) >= 256 {
				break
			}

			colorValue := (g << 8) | (r << 4) | 8

			cpcColor := cpc.GetColorFromCpcPlus(colorValue)
			fyneColor := color.RGBA{
				R: uint8(cpcColor.R),
				G: uint8(cpcColor.V),
				B: uint8(cpcColor.B),
				A: 255,
			}

			colorRect := canvas.NewRectangle(fyneColor)
			colorRect.SetMinSize(fyne.NewSize(20, 20))

			colorBtn := widget.NewButton("", func() {
				pw.setPenColor(penIndex, colorValue)
			})

			colorContainer := container.NewStack(colorRect, colorBtn)
			colorGrid.Add(colorContainer)
		}
	}

	return container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Select CPC+ color for pen %d:", penIndex)),
		widget.NewLabel("(Simplified picker - 256 of 4096 colors shown)"),
		container.NewScroll(colorGrid),
	)
}

// setPenColor sets a specific pen to a CPC color value.
func (pw *PaletteWidget) setPenColor(penIndex, colorValue int) {
	pw.app.params.Palette[penIndex] = colorValue
	pw.refreshPalette()
	pw.app.UpdatePalette()
}

// refreshPalette updates the palette display to match current parameters.
func (pw *PaletteWidget) refreshPalette() {
	var cpcPalette []bitmap.RgbColor

	if pw.app.params.CpcPlus {
		cpcPalette = make([]bitmap.RgbColor, 4096)
		for i := 0; i < 4096; i++ {
			cpcPalette[i] = cpc.GetColorFromCpcPlus(i)
		}
	} else {
		cpcPalette = cpc.GetCpcRgb()
	}

	for i := 0; i < 16; i++ {
		colorIndex := pw.app.params.Palette[i]

		var r, g, b uint8
		if colorIndex >= 0 && colorIndex < len(cpcPalette) {
			cpcColor := cpcPalette[colorIndex]
			r, g, b = uint8(cpcColor.R), uint8(cpcColor.V), uint8(cpcColor.B)
		}

		fyneColor := color.RGBA{R: r, G: g, B: b, A: 255}
		pw.penSwatches[i].FillColor = fyneColor

		// Label with lock indicator
		label := fmt.Sprintf("%d", i)
		if pw.app.params.LockState[i] != 0 {
			label = fmt.Sprintf("%d*", i)
		}
		pw.penLabels[i].SetText(label)

		pw.penSwatches[i].Refresh()
	}
}

// GetWidget returns the main palette container.
func (pw *PaletteWidget) GetWidget() *widget.Card {
	return pw.container
}

// Refresh updates the palette display.
func (pw *PaletteWidget) Refresh() {
	pw.refreshPalette()
}
