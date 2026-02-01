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
	SMode        SizeMode // Size mode
	SizeX, SizeY int      // Target size in UserSize mode
	PosX, PosY   int      // Position offset in UserSize mode

	// Core conversion settings
	Method      string    // Dithering method name
	Pct          int       // Palette optimization percentage
	LockState    [16]int   // Palette pen lock states
	DisableState [16]int   // Palette pen disable states
	Palette      [16]int   // Current CPC palette (pen to color mapping)

	// Color adjustment
	PctLumi     int // Luminosity percentage
	PctSat      int // Saturation percentage
	PctContrast int // Contrast percentage
	PctRed      int // Red channel percentage
	PctGreen    int // Green channel percentage
	PctBlue     int // Blue channel percentage

	// CPC settings
	CpcPlus     bool // Use CPC+ mode (4096 colors vs 27)
	VirtualMode int  // Virtual screen mode (0-11)
	NumCols      int  // Number of columns
	NumLines    int  // Number of lines

	// Conversion options
	NewMethod  bool // Use new conversion method
	ReductPal1  bool // Palette reduction option 1
	ReductPal2  bool // Palette reduction option 2
	ReductPal3  bool // Palette reduction option 3
	ReductPal4  bool // Palette reduction option 4
	NewSortPal  byte // New palette sorting method
	SetPalCpc   bool // Set CPC palette
	Smoothing     bool // Smoothing
	TrueColorDither     bool // Trame true color
	NewReduc    bool // New color reduction
	DiffErr     bool // Error diffusion
	Filter      bool // Apply filter

	// File I/O options
	WithCode    bool // Include display code
	WithPalette bool // Include palette data

	// Mode-specific settings
	TrackModeX int // Mode X track setting

	// File paths
	LastReadPath string // Last read directory
	LastSavePath string // Last save directory

	// Color conversion coefficients for RGB to luminance
	CoefR int // Red coefficient (default: 9798)
	CoefV int // Green coefficient (default: 19235)
	CoefB int // Blue coefficient (default: 3735)

	// RGB level thresholds for 3-level quantization (27 colors)
	CstR1, CstR2, CstR3, CstR4 int // Red thresholds
	CstV1, CstV2, CstV3, CstV4 int // Green thresholds
	CstB1, CstB2, CstB3, CstB4 int // Blue thresholds

	// Color depth and options
	BitsRGB int // Color bit depth (default: 24)

	// K-means clustering options
	KMeansDist  int  // K-means distance metric
	KMeansColor int  // Number of K-means colors
	KMeansPass  int  // Number of K-means passes
	AutoRecalc  bool // Auto-recalculate palette

	// Localization
	Language string // Language ("FR" or "EN")

	// Import/Draw mode
	ModeImpDraw bool // Import draw mode
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