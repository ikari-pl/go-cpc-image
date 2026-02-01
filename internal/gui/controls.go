// Package gui provides the controls widget for conversion parameters.
// This file implements the control panel matching the original WinForms layout.
package gui

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"

	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ikari-pl/go-cpc-image/pkg/convert"
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
	resLabel     *widget.Label // actual pixel resolution
	overscanBtn  *widget.Button
	standardBtn  *widget.Button
	cpcPlusCheck *widget.Check

	// Dithering controls
	ditherSelect   *widget.Select
	ditherSlider   *widget.Slider
	ditherPctLabel *widget.Label

	// Color adjustment sliders
	lumiSlider     *widget.Slider
	lumiLabel      *widget.Label
	satSlider      *widget.Slider
	satLabel       *widget.Label
	contrastSlider *widget.Slider
	contrastLabel  *widget.Label
	redSlider      *widget.Slider
	redLabel       *widget.Label
	greenSlider    *widget.Slider
	greenLabel     *widget.Label
	blueSlider     *widget.Slider
	blueLabel      *widget.Label

	// Options checkboxes
	smoothingCheck       *ttwidget.Check
	trueColorDitherCheck *ttwidget.Check
	newMethodCheck       *ttwidget.Check
	newReducCheck        *ttwidget.Check
	reductPal1Check      *ttwidget.Check
	reductPal2Check      *ttwidget.Check
	reductPal3Check      *ttwidget.Check
	reductPal4Check      *ttwidget.Check
	kmeansCheck          *ttwidget.Check

	// Bit depth radio
	bitsRGBRadio *widget.RadioGroup

	// Size mode
	sizeModeRadio  *widget.RadioGroup
	sizeXEntry     *widget.Entry
	sizeYEntry     *widget.Entry
	posXEntry      *widget.Entry
	posYEntry      *widget.Entry
	userSizeFields *fyne.Container

	// Crop controls
	cropEnableCheck *widget.Check
	cropLockCheck   *widget.Check
	cropResetBtn    *widget.Button

	// Conversion controls
	convertBtn  *widget.Button
	autoConvert *widget.Check

	// Reset button
	resetBtn *widget.Button

	// Data bindings
	ditherPctBinding binding.Float

	// Guard flag to suppress callbacks during RefreshFromParams
	refreshing bool
}

// newLabeledSlider creates a slider with a value label, returning (row container, slider, label).
func (cw *ControlsWidget) newLabeledSlider(name string, min, max, value float64, param *int) (*fyne.Container, *widget.Slider, *widget.Label) {
	lbl := widget.NewLabel(strconv.Itoa(int(value)) + "%")
	sl := widget.NewSlider(min, max)
	sl.SetValue(value)
	sl.OnChanged = func(v float64) {
		*param = int(v)
		lbl.SetText(strconv.Itoa(int(v)) + "%")
		if cw.autoConvert != nil && cw.autoConvert.Checked {
			cw.app.TriggerConversion()
		}
	}
	row := container.NewGridWithColumns(2,
		widget.NewLabel(name+":"), container.NewBorder(nil, nil, nil, lbl, sl),
	)
	return row, sl, lbl
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
	cw.setupColorControls()
	cw.setupOptionsControls()
	cw.setupSizeModeControls()
	cw.setupCropControls()
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
	modes := []string{"Mode 0", "Mode 1", "Mode 2", "EGX1", "EGX2", "Mode X", "Split", "ASC-UT", "ASC0", "ASC1", "ASC2"}
	// Pixel resolution label (must be created before modeSelect triggers onModeChanged)
	cw.resLabel = widget.NewLabel("")

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

// setupColorControls creates the color adjustment sliders.
func (cw *ControlsWidget) setupColorControls() {
	p := cw.app.params
	_, cw.lumiSlider, cw.lumiLabel = cw.newLabeledSlider("Luminosity", 0, 200, float64(p.PctLumi), &cw.app.params.PctLumi)
	_, cw.satSlider, cw.satLabel = cw.newLabeledSlider("Saturation", 0, 200, float64(p.PctSat), &cw.app.params.PctSat)
	_, cw.contrastSlider, cw.contrastLabel = cw.newLabeledSlider("Contrast", 0, 200, float64(p.PctContrast), &cw.app.params.PctContrast)
	_, cw.redSlider, cw.redLabel = cw.newLabeledSlider("Red %", 0, 200, float64(p.PctRed), &cw.app.params.PctRed)
	_, cw.greenSlider, cw.greenLabel = cw.newLabeledSlider("Green %", 0, 200, float64(p.PctGreen), &cw.app.params.PctGreen)
	_, cw.blueSlider, cw.blueLabel = cw.newLabeledSlider("Blue %", 0, 200, float64(p.PctBlue), &cw.app.params.PctBlue)
}

// setupOptionsControls creates the options checkboxes and bit depth radio.
func (cw *ControlsWidget) setupOptionsControls() {
	makeCheck := func(label string, param *bool, tip string) *ttwidget.Check {
		c := ttwidget.NewCheck(label, func(checked bool) {
			*param = checked
			if cw.autoConvert != nil && cw.autoConvert.Checked {
				cw.app.TriggerConversion()
			}
		})
		c.SetChecked(*param)
		if tip != "" {
			c.SetToolTip(tip)
		}
		return c
	}

	cw.smoothingCheck = makeCheck("Smoothing", &cw.app.params.Smoothing,
		"Box-blur each CPC pixel block before\ncolour matching, reducing aliasing when\ndownsampling to CPC resolution")
	cw.trueColorDitherCheck = makeCheck("Scanline interlace", &cw.app.params.TrueColorDither,
		"Average pairs of scanlines and assign\ntwo different palette colours in a\ncheckerboard to simulate intermediate colours")
	cw.newMethodCheck = makeCheck("Nearest colour", &cw.app.params.NewMethod,
		"Use weighted Euclidean distance to find\nthe nearest CPC colour instead of the old\nthreshold-based channel quantisation")
	cw.newReducCheck = makeCheck("Diverse palette", &cw.app.params.NewReduc,
		"Alternate between most-frequent and\nmost-contrasting colours when building\nthe palette, producing a more diverse result")
	cw.reductPal1Check = makeCheck("Brighten darks", &cw.app.params.ReductPal1,
		"OR each RGB byte with 0x11 — pushes\ndark values brighter, reducing the\ndarkest tonal distinctions")
	cw.reductPal2Check = makeCheck("Snap even", &cw.app.params.ReductPal2,
		"AND each RGB byte with 0xEE — zeroes\nodd intensity levels, snapping colours\nto even values")
	cw.reductPal3Check = makeCheck("Raise mids", &cw.app.params.ReductPal3,
		"OR each RGB byte with 0x22 — raises\nthe floor of mid-tones, further\nreducing tonal range")
	cw.reductPal4Check = makeCheck("Coarsen", &cw.app.params.ReductPal4,
		"AND each RGB byte with 0xDD — zeroes\nanother set of intensity levels for\ncoarser quantisation")
	cw.kmeansCheck = makeCheck("K-means", &cw.app.params.Filter,
		"Pre-process with k-means clustering:\ngroup pixels into N colour clusters\nand replace each with its centroid\nbefore CPC conversion")

	// Bit depth radio
	cw.bitsRGBRadio = widget.NewRadioGroup([]string{"24 bit", "12 bit", "9 bit", "6 bit"}, func(s string) {
		switch s {
		case "24 bit":
			cw.app.params.BitsRGB = 24
		case "12 bit":
			cw.app.params.BitsRGB = 12
		case "9 bit":
			cw.app.params.BitsRGB = 9
		case "6 bit":
			cw.app.params.BitsRGB = 6
		}
		if cw.autoConvert != nil && cw.autoConvert.Checked {
			cw.app.TriggerConversion()
		}
	})
	cw.bitsRGBRadio.Horizontal = true
	cw.bitsRGBRadio.SetSelected("24 bit")
}

// setupSizeModeControls creates the size mode radio and user size entries.
func (cw *ControlsWidget) setupSizeModeControls() {
	cw.sizeXEntry = widget.NewEntry()
	cw.sizeXEntry.SetText(strconv.Itoa(cw.app.params.SizeX))
	cw.sizeYEntry = widget.NewEntry()
	cw.sizeYEntry.SetText(strconv.Itoa(cw.app.params.SizeY))
	cw.posXEntry = widget.NewEntry()
	cw.posXEntry.SetText(strconv.Itoa(cw.app.params.PosX))
	cw.posYEntry = widget.NewEntry()
	cw.posYEntry.SetText(strconv.Itoa(cw.app.params.PosY))

	onUserSizeChanged := func(_ string) {
		if v, err := strconv.Atoi(cw.sizeXEntry.Text); err == nil {
			cw.app.params.SizeX = v
		}
		if v, err := strconv.Atoi(cw.sizeYEntry.Text); err == nil {
			cw.app.params.SizeY = v
		}
		if v, err := strconv.Atoi(cw.posXEntry.Text); err == nil {
			cw.app.params.PosX = v
		}
		if v, err := strconv.Atoi(cw.posYEntry.Text); err == nil {
			cw.app.params.PosY = v
		}
		if cw.autoConvert != nil && cw.autoConvert.Checked {
			cw.app.TriggerConversion()
		}
	}
	cw.sizeXEntry.OnChanged = onUserSizeChanged
	cw.sizeYEntry.OnChanged = onUserSizeChanged
	cw.posXEntry.OnChanged = onUserSizeChanged
	cw.posYEntry.OnChanged = onUserSizeChanged

	cw.userSizeFields = container.NewGridWithColumns(4,
		widget.NewLabel("W:"), cw.sizeXEntry,
		widget.NewLabel("H:"), cw.sizeYEntry,
		widget.NewLabel("X:"), cw.posXEntry,
		widget.NewLabel("Y:"), cw.posYEntry,
	)
	cw.userSizeFields.Hide()

	smodeNames := []string{"Fit", "Keep Smaller", "Keep Larger", "User Size", "Origin"}
	cw.sizeModeRadio = widget.NewRadioGroup(smodeNames, func(s string) {
		switch s {
		case "Fit":
			cw.app.params.SMode = convert.Fit
		case "Keep Smaller":
			cw.app.params.SMode = convert.KeepSmaller
		case "Keep Larger":
			cw.app.params.SMode = convert.KeepLarger
		case "User Size":
			cw.app.params.SMode = convert.UserSize
		case "Origin":
			cw.app.params.SMode = convert.Origin
		}
		if s == "User Size" {
			cw.userSizeFields.Show()
		} else {
			cw.userSizeFields.Hide()
		}
		if cw.autoConvert != nil && cw.autoConvert.Checked {
			cw.app.TriggerConversion()
		}
	})
	cw.sizeModeRadio.SetSelected("Fit")
}

// setupCropControls creates the crop region controls.
func (cw *ControlsWidget) setupCropControls() {
	cw.cropEnableCheck = widget.NewCheck("Enable Crop", func(checked bool) {
		cw.app.cropEnabled = checked
		if !checked {
			// Reset crop region when disabled
			cw.app.cropX1 = 0
			cw.app.cropY1 = 0
			cw.app.cropX2 = 1
			cw.app.cropY2 = 1
		}
		if cw.app.preview != nil {
			cw.app.preview.sourceRaster.Refresh()
		}
		if cw.autoConvert != nil && cw.autoConvert.Checked {
			cw.app.TriggerConversion()
		}
	})

	cw.cropLockCheck = widget.NewCheck("Lock aspect", func(checked bool) {
		cw.app.cropLockAspect = checked
		cw.updateCropAspect()
	})

	cw.cropResetBtn = widget.NewButton("Reset Crop", func() {
		cw.app.cropX1 = 0
		cw.app.cropY1 = 0
		cw.app.cropX2 = 1
		cw.app.cropY2 = 1
		if cw.app.preview != nil {
			cw.app.preview.sourceRaster.Refresh()
		}
		if cw.autoConvert != nil && cw.autoConvert.Checked {
			cw.app.TriggerConversion()
		}
	})
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

	cw.resetBtn = widget.NewButton("Reset All", func() {
		defaults := convert.NewDefaultSettings()
		p := cw.app.params
		p.PctLumi = defaults.PctLumi
		p.PctSat = defaults.PctSat
		p.PctContrast = defaults.PctContrast
		p.PctRed = defaults.PctRed
		p.PctGreen = defaults.PctGreen
		p.PctBlue = defaults.PctBlue
		p.Smoothing = defaults.Smoothing
		p.TrueColorDither = defaults.TrueColorDither
		p.NewMethod = defaults.NewMethod
		p.NewReduc = defaults.NewReduc
		p.ReductPal1 = defaults.ReductPal1
		p.ReductPal2 = defaults.ReductPal2
		p.ReductPal3 = defaults.ReductPal3
		p.ReductPal4 = defaults.ReductPal4
		p.AutoRecalc = defaults.AutoRecalc
		p.BitsRGB = defaults.BitsRGB
		p.SMode = defaults.SMode
		p.SizeX = defaults.SizeX
		p.SizeY = defaults.SizeY
		p.PosX = defaults.PosX
		p.PosY = defaults.PosY
		p.VirtualMode = defaults.VirtualMode
		p.NumCols = defaults.NumCols
		p.NumLines = defaults.NumLines
		p.CpcPlus = defaults.CpcPlus
		p.Method = defaults.Method
		p.DiffErr = defaults.DiffErr
		p.Pct = defaults.Pct
		cw.RefreshFromParams()
		if cw.autoConvert != nil && cw.autoConvert.Checked {
			cw.app.TriggerConversion()
		}
	})
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
			cw.resLabel,
			cw.cpcPlusCheck,
		),
	)

	// Color Adjustments group
	colorGroup := widget.NewCard("Color Adjustments", "",
		container.NewVBox(
			container.NewGridWithColumns(2,
				widget.NewLabel("Luminosity:"), container.NewBorder(nil, nil, nil, cw.lumiLabel, cw.lumiSlider),
				widget.NewLabel("Saturation:"), container.NewBorder(nil, nil, nil, cw.satLabel, cw.satSlider),
				widget.NewLabel("Contrast:"), container.NewBorder(nil, nil, nil, cw.contrastLabel, cw.contrastSlider),
				widget.NewLabel("Red %:"), container.NewBorder(nil, nil, nil, cw.redLabel, cw.redSlider),
				widget.NewLabel("Green %:"), container.NewBorder(nil, nil, nil, cw.greenLabel, cw.greenSlider),
				widget.NewLabel("Blue %:"), container.NewBorder(nil, nil, nil, cw.blueLabel, cw.blueSlider),
			),
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

	// Options group (3-column grid for compactness)
	optionsGroup := widget.NewCard("Options", "",
		container.NewVBox(
			container.NewGridWithColumns(3,
				cw.smoothingCheck, cw.trueColorDitherCheck, cw.newMethodCheck,
				cw.newReducCheck, cw.reductPal1Check, cw.reductPal2Check,
				cw.reductPal3Check, cw.reductPal4Check, cw.kmeansCheck,
			),
			cw.bitsRGBRadio,
		),
	)

	// Size Mode + Crop merged group
	sizeCropGroup := widget.NewCard("Size Mode / Crop", "",
		container.NewVBox(
			cw.sizeModeRadio,
			cw.userSizeFields,
			widget.NewSeparator(),
			container.NewHBox(cw.cropEnableCheck, cw.cropLockCheck, cw.cropResetBtn),
		),
	)

	// Conversion controls inline (Convert + Reset + Auto-convert)
	conversionRow := container.NewHBox(
		cw.convertBtn, cw.resetBtn, cw.autoConvert,
	)

	// Main layout
	cw.container = container.NewVBox(
		cpcGroup,
		colorGroup,
		ditherGroup,
		optionsGroup,
		sizeCropGroup,
		conversionRow,
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

// updateResLabel refreshes the pixel resolution label based on current mode and size.
func (cw *ControlsWidget) updateResLabel() {
	ppb := 2
	switch cw.app.params.VirtualMode {
	case 1:
		ppb = 4
	case 2:
		ppb = 8
	case 3: // EGX1: Mode 0 (2ppb) / Mode 1 (4ppb)
		ppb = 2
	case 4: // EGX2: Mode 1 (4ppb) / Mode 2 (8ppb)
		ppb = 4
	}
	w := cw.app.params.NumCols * ppb
	h := cw.app.params.NumLines
	cw.resLabel.SetText(fmt.Sprintf("%dx%d pixels", w, h))
}

// onModeChanged handles CPC mode changes.
func (cw *ControlsWidget) onModeChanged(mode string) {
	if cw.refreshing {
		return
	}
	switch mode {
	case "Mode 0":
		cw.app.params.VirtualMode = 0
	case "Mode 1":
		cw.app.params.VirtualMode = 1
	case "Mode 2":
		cw.app.params.VirtualMode = 2
	case "EGX1":
		cw.app.params.VirtualMode = 3
	case "EGX2":
		cw.app.params.VirtualMode = 4
	case "Mode X":
		cw.app.params.VirtualMode = 5
	case "Split":
		cw.app.params.VirtualMode = 6
	case "ASC-UT":
		cw.app.params.VirtualMode = 7
	case "ASC0":
		cw.app.params.VirtualMode = 8
	case "ASC1":
		cw.app.params.VirtualMode = 9
	case "ASC2":
		cw.app.params.VirtualMode = 10
	}

	cw.updateResLabel()

	if cw.autoConvert != nil && cw.autoConvert.Checked {
		cw.app.TriggerConversion()
	}
}

// updateCropAspect recomputes the crop aspect ratio from the current screen mode parameters.
func (cw *ControlsWidget) updateCropAspect() {
	if cw.app.cropLockAspect {
		cw.app.cropAspect = float64(cw.app.params.GetScreenWidth()) / float64(cw.app.params.GetScreenHeight())
	}
}

// onSizeChanged handles column/line count changes.
func (cw *ControlsWidget) onSizeChanged(text string) {
	if cw.refreshing {
		return
	}
	if cols, err := strconv.Atoi(cw.colsEntry.Text); err == nil && cols >= 1 && cols <= 128 {
		cw.app.params.NumCols = cols
	}

	if lines, err := strconv.Atoi(cw.linesEntry.Text); err == nil && lines >= 1 && lines <= 272 {
		cw.app.params.NumLines = lines
	}

	cw.updateResLabel()
	cw.updateCropAspect()

	if cw.autoConvert != nil && cw.autoConvert.Checked {
		cw.app.TriggerConversion()
	}
}

// onCpcPlusChanged handles CPC+ mode toggle.
func (cw *ControlsWidget) onCpcPlusChanged(checked bool) {
	if cw.refreshing {
		return
	}
	cw.app.params.CpcPlus = checked

	if cw.autoConvert != nil && cw.autoConvert.Checked {
		cw.app.TriggerConversion()
	}
}

// onDitherMethodChanged handles dithering method selection.
func (cw *ControlsWidget) onDitherMethodChanged(method string) {
	if cw.refreshing {
		return
	}
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
	if cw.refreshing {
		return
	}
	cw.ditherPctBinding.Set(value)
}

// GetWidget returns the main controls container.
func (cw *ControlsWidget) GetWidget() *fyne.Container {
	return cw.container
}

// RefreshFromParams updates all controls to match the current parameters.
func (cw *ControlsWidget) RefreshFromParams() {
	cw.refreshing = true
	defer func() { cw.refreshing = false }()

	params := cw.app.params

	// Update mode selection
	modeMap := map[int]string{
		0: "Mode 0", 1: "Mode 1", 2: "Mode 2",
		3: "EGX1", 4: "EGX2",
		5: "Mode X", 6: "Split",
		7: "ASC-UT", 8: "ASC0", 9: "ASC1", 10: "ASC2",
	}
	if name, ok := modeMap[params.VirtualMode]; ok {
		cw.modeSelect.SetSelected(name)
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

	// Update color adjustment sliders
	cw.lumiSlider.SetValue(float64(params.PctLumi))
	cw.lumiLabel.SetText(strconv.Itoa(params.PctLumi) + "%")
	cw.satSlider.SetValue(float64(params.PctSat))
	cw.satLabel.SetText(strconv.Itoa(params.PctSat) + "%")
	cw.contrastSlider.SetValue(float64(params.PctContrast))
	cw.contrastLabel.SetText(strconv.Itoa(params.PctContrast) + "%")
	cw.redSlider.SetValue(float64(params.PctRed))
	cw.redLabel.SetText(strconv.Itoa(params.PctRed) + "%")
	cw.greenSlider.SetValue(float64(params.PctGreen))
	cw.greenLabel.SetText(strconv.Itoa(params.PctGreen) + "%")
	cw.blueSlider.SetValue(float64(params.PctBlue))
	cw.blueLabel.SetText(strconv.Itoa(params.PctBlue) + "%")

	// Update options checkboxes
	cw.smoothingCheck.SetChecked(params.Smoothing)
	cw.trueColorDitherCheck.SetChecked(params.TrueColorDither)
	cw.newMethodCheck.SetChecked(params.NewMethod)
	cw.newReducCheck.SetChecked(params.NewReduc)
	cw.reductPal1Check.SetChecked(params.ReductPal1)
	cw.reductPal2Check.SetChecked(params.ReductPal2)
	cw.reductPal3Check.SetChecked(params.ReductPal3)
	cw.reductPal4Check.SetChecked(params.ReductPal4)
	cw.kmeansCheck.SetChecked(params.Filter)

	// Update bit depth radio
	switch params.BitsRGB {
	case 24:
		cw.bitsRGBRadio.SetSelected("24 bit")
	case 12:
		cw.bitsRGBRadio.SetSelected("12 bit")
	case 9:
		cw.bitsRGBRadio.SetSelected("9 bit")
	case 6:
		cw.bitsRGBRadio.SetSelected("6 bit")
	}

	// Update size mode radio
	smodeMap := map[convert.SizeMode]string{
		convert.Fit:         "Fit",
		convert.KeepSmaller: "Keep Smaller",
		convert.KeepLarger:  "Keep Larger",
		convert.UserSize:    "User Size",
		convert.Origin:      "Origin",
	}
	if name, ok := smodeMap[params.SMode]; ok {
		cw.sizeModeRadio.SetSelected(name)
	}
	cw.sizeXEntry.SetText(strconv.Itoa(params.SizeX))
	cw.sizeYEntry.SetText(strconv.Itoa(params.SizeY))
	cw.posXEntry.SetText(strconv.Itoa(params.PosX))
	cw.posYEntry.SetText(strconv.Itoa(params.PosY))

	// Update crop checkboxes
	cw.cropEnableCheck.SetChecked(cw.app.cropEnabled)
	cw.cropLockCheck.SetChecked(cw.app.cropLockAspect)

	cw.updateResLabel()
}
