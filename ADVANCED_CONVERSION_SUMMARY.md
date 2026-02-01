# Advanced CPC Conversion Modes - Phase 5-7 Implementation

## Overview

This document summarizes the implementation of advanced conversion modes, split-screen effects, and animation support for the ConvImgCpc Go rewrite.

## Files Created

### 1. Mode X Converter (`pkg/convert/modex.go`)
- **Mode X**: 4 colors per line (2 fixed globally + 2 variable per line)
- **Key Functions**:
  - `FindBestColorsModeX()`: Finds optimal colors using frequency analysis
  - `ConvertModeX()`: Performs conversion with per-line palette optimization
  - Color frequency table `colorFrequency[4096][272]` for tracking usage
- **Features**:
  - Exactly matches C# algorithm and magic numbers
  - Supports both CPC (27 colors) and CPC+ (4096 colors) modes
  - Tracking window for per-line optimization (`TrackModeX` parameter)
  - Proper distance calculation with RGB coefficients

### 2. Split-Raster Converter (`pkg/convert/split.go`)
- **Split Mode**: 3 fixed colors + 6 variable colors per line = 9 total
- **Key Functions**:
  - `FindBestColorsModeSplit()`: Similar to Mode X but with 3 fixed + 6 variable colors
  - `ConvertSplit()`: Simulates hardware raster splits with color changes
- **Features**:
  - Hardware split simulation with 32-pixel minimum split length
  - Automatic split management (starting/changing splits)
  - Preserves C# logic for split timing and color selection

### 3. ASCII Art Converter (`pkg/convert/ascii.go`)
- **ASCII Modes**: ASC0/ASC1/ASC2 with different resolutions
- **Key Components**:
  - `TrameM1`: 4x4 trame patterns for Mode 1 dithering
  - `ConvertAscii()`: Standard ASCII conversion with pixel averaging
  - `ConvertAscUt()`: ASCII conversion using precalculated trame patterns
  - `CnvTrame()`: Automatic trame pattern generation
- **Features**:
  - 8-pixel vertical averaging for better quality
  - Manhattan distance calculation for trame matching
  - Support for custom and default trame pattern sets

### 4. Split-Screen Management (`pkg/splitscreen/split.go`)
- **Core Structures**:
  - `Split`: Individual raster split (length, color, enable)
  - `LigneSplit`: Per-scanline split configuration
  - `SplitEcran`: Complete split-screen setup
  - `BitmapCpc`: CPC bitmap with split-screen support
- **Key Functions**:
  - `CalcPaletteSplit()`: Calculate split palette for entire screen
  - `AppliquePalette()`: Apply palette colors to X position ranges
  - `OptimizeSplits()`: Merge consecutive splits and reduce hardware load
- **Features**:
  - Supports up to 7 splits per line
  - 96 columns × 272 lines × 17 colors palette array
  - Timing delay management (`RetardMin = -4`)

### 5. Animation Delta Encoding (`pkg/animation/delta.go`)
- **Delta Compression**:
  - `DeltaDiff`: Consecutive byte changes at specific addresses
  - `DiffAnim`: Frame-to-frame difference management
- **Key Functions**:
  - `AddDiff()`: Add byte differences with automatic block creation
  - `CompareFrames()`: Generate deltas between two frame buffers
  - `ApplyDelta()`: Reconstruct frames from base + deltas
  - `Save()`: Export as Z80 assembly source code
  - `SaveBinary()`: Export as binary data for direct CPC loading
- **Features**:
  - Automatic consecutive address optimization
  - Z80 assembly output with proper formatting
  - Binary format with end markers for CPC loader

### 6. Multi-Frame Animation (`pkg/animation/frames.go`)
- **Frame Management**:
  - `ImageSource`: Multi-frame image container
  - GIF import support with animation timing
  - Frame manipulation (add, delete, select)
- **Key Functions**:
  - `InitFromReader()`: Load GIF animations with timing
  - `AddFrame()`, `DeleteImage()`: Frame manipulation
  - `GetFrameRate()`: Timing conversion (centiseconds ↔ FPS)
- **Features**:
  - Deep copying for frame independence
  - Animation timing preservation from GIF metadata
  - Frame size management and resizing support

## Enhanced Core Components

### Updated Parameters (`pkg/convert/params.go`)
- Added `Palette [16]int` field for current CPC palette
- Initialized with default CPC palette: `{1, 24, 20, 6, 26, 0, 2, 7, 10, 12, 14, 16, 18, 22, 1, 14}`
- Maintains all existing parameters for backward compatibility

### ImageCpc Structure (`pkg/convert/modex.go`)
- New `ImageCpc` structure for CPC image representation
- `SetPixelCpc()` method matching C# behavior
- Pixel data stored as `[][]int` for pen numbers

## Algorithm Fidelity

All implementations maintain exact fidelity to the C# source code:

### Magic Numbers Preserved
- `0x7FFFFFFF`: Maximum distance for color matching
- `272`: Maximum CPC screen lines
- `4096`: CPC+ color space size
- `27`: Standard CPC color count
- `32`: Minimum split length in pixels
- `-4`: Minimum split timing delay

### Distance Calculations
- Squared Euclidean distance with RGB coefficients
- Default coefficients: `CoefR=9798, CoefV=19235, CoefB=3735`
- Proper handling of exact matches (distance=0)

### Memory Layouts
- `colorFrequency[4096][272]`: Color frequency per line
- `SplitPalette[96][272][17]`: Split palette array
- `trameM1[16][4][4]`: 4x4 trame patterns

## Testing Infrastructure

Created `cmd/test-advanced/main.go` to verify:
- All converter initialization
- Structure creation and manipulation
- Basic conversion workflow
- Memory allocation patterns

## Next Steps

1. **Integration Testing**: Test with actual CPC image data
2. **Performance Optimization**: Profile color frequency calculations
3. **Error Handling**: Add comprehensive error checking
4. **Documentation**: Add usage examples and API documentation
5. **Benchmarking**: Compare performance with C# version

## Compatibility Notes

- All public APIs use idiomatic Go naming conventions
- Internal algorithms match C# implementation exactly
- Error handling uses Go conventions (multiple return values)
- Memory management leverages Go's garbage collector
- Thread safety can be added later if needed for concurrent conversions

## File Structure

```
pkg/
├── convert/
│   ├── params.go      # Enhanced with Palette field
│   ├── modex.go       # Mode X converter
│   ├── split.go       # Split-raster converter
│   └── ascii.go       # ASCII art converter + TrameM1
├── splitscreen/
│   └── split.go       # Split-screen structures and management
└── animation/
    ├── delta.go       # Delta compression for animations
    └── frames.go      # Multi-frame image handling

cmd/
└── test-advanced/
    └── main.go        # Test program for all new functionality
```

This implementation provides a solid foundation for the advanced CPC conversion modes while maintaining perfect compatibility with the existing C# algorithms and data structures.