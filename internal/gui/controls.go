// Package gui provides the controls widget for conversion parameters.
// This file implements the control panel matching the original WinForms layout.
package gui

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"

	"github.com/ikari/go-cpc-image/pkg/convert"
)

// ControlsWidget manages all the conversion parameter controls.
type ControlsWidget struct {
	app *Application

	// Main container
	container *fyne.Container

	// CPC Settings
	modeSelect   *widget.Select
	colsEntry    *widget.Entry
	linesEntry   *widget.Entry
	overscanBtn  *widget.Button
	standardBtn  *widget.Button
	cpcPlusCheck *widget.Check

	// Dithering controls
	ditherSelect *widget.Select
	ditherSlider *widget.Slider
	ditherPctLabel *widget.Label

	// Conversion controls
	convertBtn  *widget.Button
	autoConvert *widget.Check

	// Data bindings
	ditherPctBinding binding.Float
}

// NewControlsWidget creates a new controls widget.
func NewControlsWidget(app *Application) (*ControlsWidget, error) {
	cw := &ControlsWidget{
		app: app,
	}

	// Create data bindings
	cw.ditherPctBinding = binding.NewFloat()
	cw.ditherPctBinding.Set(float64(app.params.Pct))

	// Set up controls
	cw.setupCpcControls()
	cw.setupDitherControls()
	cw.setupConversionControls()

	// Build layout
	cw.buildLayout()

	// Wire up bindings
	cw.wireBindings()

	return cw, nil
}

// setupCpcControls creates the CPC resolution and mode controls.
func (cw *ControlsWidget) setupCpcControls() {
	// Mode selector
	modes := []string{"Mode 0", "Mode 1", "Mode 2"}
	cw.modeSelect = widget.NewSelect(modes, cw.onModeChanged)
	cw.modeSelect.SetSelected("Mode 1") // Default

	// Columns and lines
	cw.colsEntry = widget.NewEntry()
	cw.colsEntry.SetText("80")
	cw.colsEntry.OnChanged = cw.onSizeChanged

	cw.linesEntry = widget.NewEntry()
	cw.linesEntry.SetText("200")
	cw.linesEntry.OnChanged = cw.onSizeChanged

	// Resolution buttons
	cw.standardBtn = widget.NewButton("Standard", func() {
		cw.colsEntry.SetText("80")
		cw.linesEntry.SetText("200")
		cw.onSizeChanged("")
	})

	cw.overscanBtn = widget.NewButton("Fullscreen", func() {
		cw.colsEntry.SetText("96")
		cw.linesEntry.SetText("272")
		cw.onSizeChanged("")
	})

	// CPC Plus checkbox
	cw.cpcPlusCheck = widget.NewCheck("CPC+", cw.onCpcPlusChanged)
}

// setupDitherControls creates the dithering method and percentage controls.
func (cw *ControlsWidget) setupDitherControls() {
	// Get available dithering methods
	methods := []string{"None"}
	methods = append(methods, convert.GetAvailableMethods()...)

	cw.ditherSelect = widget.NewSelect(methods, cw.onDitherMethodChanged)
	cw.ditherSelect.SetSelected("Floyd-Steinberg (2x2)")

	// Dithering percentage slider
	cw.ditherSlider = widget.NewSlider(0, 100)
	cw.ditherSlider.SetValue(float64(cw.app.params.Pct))
	cw.ditherSlider.OnChanged = cw.onDitherPctChanged

	// Percentage label
	cw.ditherPctLabel = widget.NewLabel("50%")
}

// setupConversionControls creates the conversion and auto-convert controls.
func (cw *ControlsWidget) setupConversionControls() {
	cw.convertBtn = widget.NewButton("Convert", func() {
		cw.app.TriggerConversion()
	})
	cw.convertBtn.Importance = widget.HighImportance

	cw.autoConvert = widget.NewCheck("Auto-convert", func(checked bool) {
		if checked {
			cw.app.TriggerConversion()
		}
	})
	cw.autoConvert.SetChecked(true) // Auto-convert enabled by default
}

// buildLayout creates the control panel layout.
func (cw *ControlsWidget) buildLayout() {
	// CPC Resolution group
	cpcGroup := widget.NewCard("CPC Resolution", "",
		container.NewVBox(
			container.NewGridWithColumns(2,
				widget.NewLabel("Columns:"), cw.colsEntry,
				widget.NewLabel("Lines:"), cw.linesEntry,
			),
			container.NewHBox(cw.standardBtn, cw.overscanBtn),
			container.NewGridWithColumns(2,
				widget.NewLabel("Mode:"), cw.modeSelect,
			),
			cw.cpcPlusCheck,
		),
	)

	// Dithering group
	ditherGroup := widget.NewCard("Dithering", "",
		container.NewVBox(
			container.NewGridWithColumns(2,
				widget.NewLabel("Type:"), cw.ditherSelect,
			),
			container.NewBorder(
				nil, nil,
				widget.NewLabel("0%"),
				container.NewHBox(cw.ditherPctLabel, widget.NewLabel(" ")),
				cw.ditherSlider,
			),
		),
	)

	// Conversion controls
	conversionGroup := widget.NewCard("Conversion", "",
		container.NewVBox(
			cw.convertBtn,
			cw.autoConvert,
		),
	)

	// Main layout: stacked groups
	cw.container = container.NewVBox(
		cpcGroup,
		ditherGroup,
		conversionGroup,
	)
}

// wireBindings connects the data bindings to update the UI.
func (cw *ControlsWidget) wireBindings() {
	// Listen to dithering percentage changes
	cw.ditherPctBinding.AddListener(binding.NewDataListener(func() {
		val, _ := cw.ditherPctBinding.Get()
		cw.ditherPctLabel.SetText(strconv.Itoa(int(val)) + "%")
		cw.app.params.Pct = int(val)

		// Trigger conversion if auto-convert is enabled
		if cw.autoConvert.Checked {
			cw.app.TriggerConversion()
		}
	}))
}

// Event handlers

// onModeChanged handles CPC mode changes.
func (cw *ControlsWidget) onModeChanged(mode string) {
	switch mode {
	case "Mode 0":
		cw.app.params.VirtualMode = 0
	case "Mode 1":
		cw.app.params.VirtualMode = 1
	case "Mode 2":
		cw.app.params.VirtualMode = 2
	}

	if cw.autoConvert != nil && cw.autoConvert.Checked {
		cw.app.TriggerConversion()
	}
}

// onSizeChanged handles column/line count changes.
func (cw *ControlsWidget) onSizeChanged(text string) {
	if cols, err := strconv.Atoi(cw.colsEntry.Text); err == nil && cols >= 1 && cols <= 128 {
		cw.app.params.NumCols = cols
	}

	if lines, err := strconv.Atoi(cw.linesEntry.Text); err == nil && lines >= 1 && lines <= 272 {
		cw.app.params.NumLines = lines
	}

	if cw.autoConvert != nil && cw.autoConvert.Checked {
		cw.app.TriggerConversion()
	}
}

// onCpcPlusChanged handles CPC+ mode toggle.
func (cw *ControlsWidget) onCpcPlusChanged(checked bool) {
	cw.app.params.CpcPlus = checked

	if cw.autoConvert != nil && cw.autoConvert.Checked {
		cw.app.TriggerConversion()
	}
}

// onDitherMethodChanged handles dithering method selection.
func (cw *ControlsWidget) onDitherMethodChanged(method string) {
	if method == "None" {
		cw.app.params.Method = ""
	} else {
		cw.app.params.Method = method
	}

	// Match C# behavior: Floyd-Steinberg automatically enables error diffusion
	cw.app.params.DiffErr = (method == "Floyd-Steinberg (2x2)")

	if cw.autoConvert != nil && cw.autoConvert.Checked {
		cw.app.TriggerConversion()
	}
}

// onDitherPctChanged handles dithering percentage changes.
func (cw *ControlsWidget) onDitherPctChanged(value float64) {
	cw.ditherPctBinding.Set(value)
}

// GetWidget returns the main controls container.
func (cw *ControlsWidget) GetWidget() *fyne.Container {
	return cw.container
}

// RefreshFromParams updates all controls to match the current parameters.
func (cw *ControlsWidget) RefreshFromParams() {
	params := cw.app.params

	// Update mode selection
	modeNames := []string{"Mode 0", "Mode 1", "Mode 2"}
	if params.VirtualMode >= 0 && params.VirtualMode < len(modeNames) {
		cw.modeSelect.SetSelected(modeNames[params.VirtualMode])
	}

	// Update size entries
	cw.colsEntry.SetText(strconv.Itoa(params.NumCols))
	cw.linesEntry.SetText(strconv.Itoa(params.NumLines))

	// Update CPC+ checkbox
	cw.cpcPlusCheck.SetChecked(params.CpcPlus)

	// Update dithering controls
	if params.Method == "" {
		cw.ditherSelect.SetSelected("None")
	} else {
		cw.ditherSelect.SetSelected(params.Method)
	}

	cw.ditherSlider.SetValue(float64(params.Pct))
	cw.ditherPctBinding.Set(float64(params.Pct))
}