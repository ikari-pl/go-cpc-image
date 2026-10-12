// Package convert provides image conversion parameters and configuration.
package convert

import "github.com/ikari-pl/go-cpc-image/pkg/cpc"

// SizeMode defines how the source image should be sized for conversion.
type SizeMode int

const (
	Fit         SizeMode = iota // Fit image within CPC screen bounds
	KeepSmaller                 // Keep smaller dimension unchanged
	KeepLarger                  // Keep larger dimension unchanged
	UserSize                    // User-specified dimensions
	Origin                      // Keep original size
)

// Param contains all conversion parameters.
// This structure maintains compatibility with the C# Param class.
type Settings struct {
	// Size and positioning
	SMode        SizeMode `json:"sMode"`
	SizeX        int      `json:"sizeX"`
	SizeY        int      `json:"sizeY"`
	PosX         int      `json:"posX"`
	PosY         int      `json:"posY"`

	// Core conversion settings
	Method       string   `json:"method"`
	Pct          int      `json:"pct"`
	LockState    [16]int  `json:"lockState"`
	DisableState [16]int  `json:"disableState"`
	Palette      [16]int  `json:"palette"`

	// Color adjustment
	PctLumi     int `json:"pctLumi"`
	PctSat      int `json:"pctSat"`
	PctContrast int `json:"pctContrast"`
	PctRed      int `json:"pctRed"`
	PctGreen    int `json:"pctGreen"`
	PctBlue     int `json:"pctBlue"`

	// CPC settings
	CpcPlus     bool `json:"cpcPlus"`
	VirtualMode int  `json:"virtualMode"`
	YEgx        int  `json:"yEgx"`
	NumCols     int  `json:"numCols"`
	NumLines    int  `json:"numLines"`

	// Conversion options
	NewMethod       bool `json:"newMethod"`
	ReductPal1      bool `json:"reductPal1"`
	ReductPal2      bool `json:"reductPal2"`
	ReductPal3      bool `json:"reductPal3"`
	ReductPal4      bool `json:"reductPal4"`
	NewSortPal      byte `json:"newSortPal"`
	SetPalCpc       bool `json:"setPalCpc"`
	Smoothing       bool `json:"smoothing"`
	TrueColorDither bool `json:"trueColorDither"`
	NewReduc        bool `json:"newReduc"`
	DiffErr         bool `json:"diffErr"`
	Filter          bool `json:"filter"`

	// File I/O options
	WithCode    bool `json:"withCode"`
	WithPalette bool `json:"withPalette"`

	// Mode-specific settings
	TrackModeX int `json:"trackModeX"`

	// File paths
	LastReadPath string `json:"lastReadPath"`
	LastSavePath string `json:"lastSavePath"`

	// Color conversion coefficients for RGB to luminance
	CoefR int `json:"coefR"`
	CoefV int `json:"coefV"`
	CoefB int `json:"coefB"`

	// RGB level thresholds for 3-level quantization (27 colors)
	CstR1 int `json:"cstR1"`
	CstR2 int `json:"cstR2"`
	CstR3 int `json:"cstR3"`
	CstR4 int `json:"cstR4"`
	CstV1 int `json:"cstV1"`
	CstV2 int `json:"cstV2"`
	CstV3 int `json:"cstV3"`
	CstV4 int `json:"cstV4"`
	CstB1 int `json:"cstB1"`
	CstB2 int `json:"cstB2"`
	CstB3 int `json:"cstB3"`
	CstB4 int `json:"cstB4"`

	// Color depth and options
	BitsRGB int `json:"bitsRGB"`

	// K-means clustering options
	KMeansDist  int  `json:"kMeansDist"`
	KMeansColor int  `json:"kMeansColor"`
	KMeansPass  int  `json:"kMeansPass"`
	AutoRecalc  bool `json:"autoRecalc"`

	// Localization
	Language string `json:"language"`

	// Import/Draw mode
	ModeImpDraw bool `json:"modeImpDraw"`

	// Crop region (normalized 0-1 coordinates)
	CropEnabled    bool    `json:"cropEnabled,omitempty"`
	CropLockAspect bool    `json:"cropLockAspect,omitempty"`
	CropX1         float64 `json:"cropX1,omitempty"`
	CropY1         float64 `json:"cropY1,omitempty"`
	CropX2         float64 `json:"cropX2,omitempty"`
	CropY2         float64 `json:"cropY2,omitempty"`
}

// NewDefaultSettings creates a Param with default values matching the C# version.
func NewDefaultSettings() *Settings {
	p := &Settings{
		SMode:       Fit,
		Method:     "Floyd-Steinberg (2x2)",
		DiffErr:     true, // Floyd-Steinberg uses error diffusion (matching C# auto-check)
		Pct:         50,
		PctLumi:     100,
		PctSat:      100,
		PctContrast: 100,
		PctRed:      100,
		PctGreen:    100,
		PctBlue:     100,
		VirtualMode: 1, // Default to Mode 1
		NumCols:      80,
		NumLines:    200,
		CoefR:       9798,
		CoefV:       19235,
		CoefB:       3735,
		CstR1:       85,
		CstR2:       170,
		CstR3:       255,
		CstR4:       340,
		CstV1:       85,
		CstV2:       170,
		CstV3:       255,
		CstV4:       340,
		CstB1:       85,
		CstB2:       170,
		CstB3:       255,
		CstB4:       340,
		BitsRGB:     24,
		KMeansDist:  0,
		KMeansColor: 16,
		KMeansPass:  16,
		Language:      "EN",
	}

	// Initialize default palette (matching C# Cpc.Palette)
	p.Palette = [16]int{1, 24, 20, 6, 26, 0, 2, 7, 10, 12, 14, 16, 18, 22, 1, 14}

	return p
}

// GetThreshold returns the RGB quantization thresholds for the given component.
func (p *Settings) GetThresholds(component string) [4]int {
	switch component {
	case "R":
		return [4]int{p.CstR1, p.CstR2, p.CstR3, p.CstR4}
	case "V", "G":
		return [4]int{p.CstV1, p.CstV2, p.CstV3, p.CstV4}
	case "B":
		return [4]int{p.CstB1, p.CstB2, p.CstB3, p.CstB4}
	default:
		return [4]int{85, 170, 255, 340}
	}
}

// IsOverscan returns true if the current configuration uses overscan mode.
func (p *Settings) IsOverscan() bool {
	return p.NumCols > 80 || p.NumLines > 200
}

// bufferTimeline reports whether this conversion keeps the fixed 640-dot
// timeline instead of the screen's own grid. Only the EGX modes need it: their
// two line phases hold different pixel counts (a mode-1 line follows a mode-0
// one), so no single grid describes the picture and the timeline, with its
// per-line pixel width, is the only uniform representation of it.
func (p *Settings) bufferTimeline() bool {
	return p.VirtualMode == 3 || p.VirtualMode == 4
}

// rowsPerLine is how many buffer rows one CRTC line occupies: one in the native
// grid, two on the timeline.
func (p *Settings) rowsPerLine() int {
	if p.bufferTimeline() {
		return 2
	}
	return 1
}

// CanvasRow maps a buffer row to the row the CPC lookups are indexed by: the
// pen tables, the dither matrix and the CRTC line are all keyed at timeline
// scale, i.e. two rows per line.
func (p *Settings) CanvasRow(y int) int { return y * p.rowsPerLine() }

// LineIndex maps a buffer row to its CRTC line.
func (p *Settings) LineIndex(y int) int { return y / p.rowsPerLine() }

// LineStep is the number of buffer rows one CRTC line occupies, i.e. the step
// between the rows a line pair is read from.
func (p *Settings) LineStep() int { return p.rowsPerLine() }

// RowStep is the number of buffer rows one conversion step covers: one CRTC
// line, or two when true-colour dithering averages a line pair.
func (p *Settings) RowStep() int {
	if p.TrueColorDither {
		return p.rowsPerLine() * 2
	}
	return p.rowsPerLine()
}

// PixelStep is the number of buffer columns one picture pixel occupies: one in
// the native grid, the mode's pixel width (Tx) on the timeline.
func (p *Settings) PixelStep(y int) int {
	if p.bufferTimeline() {
		return cpc.PixelWidth(p.VirtualMode, y, p.YEgx)
	}
	return 1
}

// GetScreenWidth returns the width of the conversion buffer in pixels: the
// screen's own grid (NumCols*8 divided by the mode's pixel width), or the fixed
// timeline the EGX modes need.
func (p *Settings) GetScreenWidth() int {
	return p.NumCols * 8 / cpc.PixelWidth(p.VirtualMode, 0, p.YEgx)
}

// GetScreenHeight returns the height of the conversion buffer in rows: one row
// per CRTC line, or two on the EGX timeline.
func (p *Settings) GetScreenHeight() int {
	return p.NumLines * p.rowsPerLine()
}