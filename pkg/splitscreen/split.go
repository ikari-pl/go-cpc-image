// Package splitscreen provides CPC split-screen raster effect management.
// This handles the SplitLine structures and split palette calculations.
package splitscreen

const (
	// DelayMin is the minimum delay value for split timing.
	DelayMin = -4 // Equivalent to C# retardMin

	// MaxSplits is the maximum number of splits per line.
	MaxSplits = 7

	// MaxLines is the maximum number of CPC screen lines.
	MaxLines = 272

	// MaxCols is the maximum number of CPC screen columns.
	MaxCols = 96
)

// Split represents a single raster split within a scanline.
// This is equivalent to the C# Split class.
type Split struct {
	Length int  // Length of the split in pixels (default: 32)
	Color  int  // Color index for this split
	Enable   bool // Whether this split is enabled
}

// NewSplit creates a new Split with default values.
func NewSplit() *Split {
	return &Split{
		Length: 32,
		Color:  0,
		Enable:   false,
	}
}

// SplitLine represents the split configuration for a single scanline.
// This is equivalent to the C# SplitLine class.
type SplitLine struct {
	SplitList  []*Split // List of splits for this line
	NumPen      int      // Pen number for mode changes
	Delay      int      // Timing delay for the splits
	NewMode     int      // New screen mode for this line
	ChangeMode  bool     // Whether to change screen mode
}

// NewSplitLine creates a new SplitLine with default values.
func NewSplitLine() *SplitLine {
	ligne := &SplitLine{
		SplitList: make([]*Split, 0, MaxSplits),
		NumPen:     0,
		Delay:     DelayMin,
		NewMode:    0,
		ChangeMode: false,
	}

	// Initialize with default splits
	for j := 0; j < MaxSplits; j++ {
		ligne.SplitList = append(ligne.SplitList, NewSplit())
	}

	return ligne
}

// GetSplit returns the split at the specified index, creating it if necessary.
// This replicates the C# GetSplit method behavior.
func (ls *SplitLine) GetSplit(num int) *Split {
	// Extend list if necessary
	for len(ls.SplitList) <= num {
		ls.SplitList = append(ls.SplitList, NewSplit())
	}
	return ls.SplitList[num]
}

// AddSplit adds a new split to this line.
func (ls *SplitLine) AddSplit(length, color int, enable bool) {
	split := &Split{
		Length: length,
		Color:  color,
		Enable:   enable,
	}
	ls.SplitList = append(ls.SplitList, split)
}

// SplitScreen represents the complete split-screen configuration for an image.
// This is equivalent to the C# SplitScreen class.
type SplitScreen struct {
	SplitLines []*SplitLine // Array of split configurations per line
}

// NewSplitScreen creates a new SplitScreen with initialized lines.
func NewSplitScreen() *SplitScreen {
	return &SplitScreen{
		SplitLines: make([]*SplitLine, 0, MaxLines),
	}
}

// GetLine returns the split configuration for the specified line number.
func (se *SplitScreen) GetLine(num int) *SplitLine {
	if num >= 0 && num < len(se.SplitLines) {
		return se.SplitLines[num]
	}
	return nil
}

// InitializeLines initializes all lines with default split configurations.
func (se *SplitScreen) InitializeLines() {
	se.SplitLines = make([]*SplitLine, MaxLines)
	for y := 0; y < MaxLines; y++ {
		se.SplitLines[y] = NewSplitLine()
	}
}

// BitmapCpc represents a CPC bitmap with split-screen support.
// This is equivalent to the relevant parts of C# BitmapCpc class.
type BitmapCpc struct {
	ScreenData        []byte               // CPC image data buffer
	ImgAscii      []byte               // ASCII image data (smaller buffer)
	ModeTable       []int                // Screen mode per line
	SplitScreen    *SplitScreen          // Split-screen configuration
	SplitPalette  [MaxCols][MaxLines][17]int // Split palette data [x][y][pen]
	IsComputed        bool                 // Whether splits have been calculated
	Width         int                  // Image width
	Height        int                  // Image height
}

// NewBitmapCpc creates a new CPC bitmap with split-screen support.
func NewBitmapCpc(width, height, mode int) *BitmapCpc {
	bmp := &BitmapCpc{
		ScreenData:       make([]byte, 0x10000), // 64KB CPC memory
		ImgAscii:     make([]byte, 0x1000),  // 4KB ASCII data
		ModeTable:      make([]int, MaxLines),
		SplitScreen:   NewSplitScreen(),
		IsComputed:       false,
		Width:        width,
		Height:       height,
	}

	bmp.SplitScreen.InitializeLines()

	// Initialize mode table and split palette
	for y := 0; y < MaxLines; y++ {
		bmp.ModeTable[y] = mode
		for x := 0; x < MaxCols; x++ {
			for i := 0; i < 17; i++ {
				// Initialize with default palette values
				if i < 16 {
					bmp.SplitPalette[x][y][i] = i // Default pen mapping
				} else {
					bmp.SplitPalette[x][y][i] = 0 // Extra slot
				}
			}
		}
	}

	return bmp
}

// AppliquePalette applies palette colors to a range of X positions for the given Y line.
// This replicates the C# AppliquePalette method.
func (bmp *BitmapCpc) AppliquePalette(x, y, xpos int, curPal []int) int {
	for ; x < xpos && x < MaxCols; x++ {
		for i := 0; i < 16 && i < len(curPal); i++ {
			bmp.SplitPalette[x][y][i] = curPal[i]
		}
	}
	return x
}

// CalcPaletteSplit calculates the split palette for the entire screen.
// This replicates the C# CalcPaletteSplit method behavior.
func (bmp *BitmapCpc) CalcPaletteSplit() {
	// Initialize current palette with default values
	curPal := make([]int, 16)
	for i := 0; i < 16; i++ {
		curPal[i] = bmp.SplitPalette[0][0][i]
	}

	// Process each scanline
	for y := 0; y < MaxLines && y < bmp.Height>>1; y++ {
		ligne := bmp.SplitScreen.GetLine(y)
		if ligne == nil {
			continue
		}

		x := 0
		oldX := 0

		// Apply initial palette for the start of the line
		x = bmp.AppliquePalette(x, y, bmp.Width>>3, curPal)

		// Process each split in the line
		for _, split := range ligne.SplitList {
			if split == nil || !split.Enable {
				continue
			}

			// Update palette color
			if split.Color >= 0 && split.Color < 16 {
				curPal[split.Color] = split.Color
			}

			// Calculate position for this split
			newX := oldX + (split.Length >> 3) // Convert pixels to columns
			if newX > MaxCols {
				newX = MaxCols
			}

			// Apply palette from current position to split position
			x = bmp.AppliquePalette(x, y, newX, curPal)
			oldX = newX
		}

		// Fill remaining columns with current palette
		if x < MaxCols {
			bmp.AppliquePalette(x, y, MaxCols, curPal)
		}
	}

	bmp.IsComputed = true
}

// GetSplitPalette returns the palette color for the specified position and pen.
func (bmp *BitmapCpc) GetSplitPalette(x, y, pen int) int {
	if x >= 0 && x < MaxCols && y >= 0 && y < MaxLines && pen >= 0 && pen < 17 {
		return bmp.SplitPalette[x][y][pen]
	}
	return 0
}

// SetSplitPalette sets the palette color for the specified position and pen.
func (bmp *BitmapCpc) SetSplitPalette(x, y, pen, color int) {
	if x >= 0 && x < MaxCols && y >= 0 && y < MaxLines && pen >= 0 && pen < 17 {
		bmp.SplitPalette[x][y][pen] = color
	}
}

// OptimizeSplits optimizes the split configuration to reduce hardware requirements.
func (bmp *BitmapCpc) OptimizeSplits() {
	for y := 0; y < len(bmp.SplitScreen.SplitLines); y++ {
		ligne := bmp.SplitScreen.SplitLines[y]
		if ligne == nil {
			continue
		}

		// Remove disabled or redundant splits
		activeSplits := []*Split{}
		for _, split := range ligne.SplitList {
			if split != nil && split.Enable && split.Length > 0 {
				activeSplits = append(activeSplits, split)
			}
		}

		// Merge consecutive splits with the same color
		mergedSplits := []*Split{}
		for i, split := range activeSplits {
			if i > 0 && mergedSplits[len(mergedSplits)-1].Color == split.Color {
				// Merge with previous split
				mergedSplits[len(mergedSplits)-1].Length += split.Length
			} else {
				mergedSplits = append(mergedSplits, split)
			}
		}

		ligne.SplitList = mergedSplits
	}
}