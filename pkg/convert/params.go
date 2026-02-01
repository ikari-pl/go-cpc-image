// Package convert provides image conversion parameters and configuration.
package convert

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
type Param struct {
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
}

// NewDefaultParam creates a Param with default values matching the C# version.
func NewDefaultParam() *Param {
	p := &Param{
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
func (p *Param) GetThresholds(component string) [4]int {
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
func (p *Param) IsOverscan() bool {
	return p.NumCols > 80 || p.NumLines > 200
}

// GetScreenWidth returns the screen width in pixels.
func (p *Param) GetScreenWidth() int {
	return p.NumCols * 8
}

// GetScreenHeight returns the screen height in pixels.
func (p *Param) GetScreenHeight() int {
	return p.NumLines * 2
}