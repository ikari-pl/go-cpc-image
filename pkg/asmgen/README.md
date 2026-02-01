# CPC Assembly Generation Package

This package provides comprehensive Z80 assembly code generation for Amstrad CPC display routines, decompression code, palette management, and advanced effects like split-raster timing.

## Features

### Assembly Utilities (`util.go`)
- **Structured Z80 assembly output** with proper formatting
- **DB/DW data generation** with automatic optimization
- **Label and comment management**
- **Sectioned data output** with automatic labeling

### Display Routines (`display.go`)
- **Standard CPC display code** generation
- **EGX mode support** with cycle-exact raster effects
- **Mode X display** for special color cycling effects
- **CPC Plus ASIC initialization**
- **Overscan and standard screen formats**

### Decompression Code (`depack.go`)
- **LZW decompression** routine generation
- **ZX0 family decompressors** (ZX0, ZX0v2, ZX1)
- **Optimized assembly output** with proper labels
- **Jump target customization**

### Palette Generation (`palette.go`)
- **CPC and CPC Plus palette data**
- **Animation palette sequences**
- **Split-raster palette effects**
- **OCP+ and KIT format support**
- **Palette transitions and effects**

### Split-Raster Effects (`split.go`)
- **Cycle-exact timing** for horizontal color splits
- **NOP padding generation** for precise delays
- **Multi-color scanlines** (up to 7 colors per line)
- **Loop optimization** for large delays
- **Hardware register management**

## Core Components

### ASMWriter

The `ASMWriter` struct provides the foundation for all assembly generation:

```go
type ASMWriter struct {
    w io.Writer
}

func NewASMWriter(w io.Writer) *ASMWriter
```

### Basic Operations

```go
// Headers and structure
aw.WriteHeader("v1.0", "Mode 0", 80, 200)
aw.WriteOrg(0x8000)
aw.WriteLabel("StartLabel")

// Instructions
aw.WriteInstruction("LD", "A,#42", "Load constant")
aw.WriteInstruction("OUT", "(C),A", "Output to port")

// Data generation
data := []byte{0x01, 0x02, 0x03, 0x04}
aw.WriteDB(data, 4, "MyData", true)

words := []uint16{0x1234, 0x5678}
aw.WriteDW(words, 2, "MyWords", true)
```

## Display Code Generation

### Standard Display

```go
// Generate complete display routine
err := aw.GenerateStandardDisplay(overscan, packMethod,
                                  "ImageData", "Palette", cpcPlus)
```

### EGX Mode

```go
// Enhanced Graphics eXtension mode
err := aw.GenerateEGXDisplay(palette, overscan, packMethod,
                             "ImageData", "Palette")
```

### Mode X (Color Cycling)

```go
// Special color cycling mode
err := aw.GenerateModeXDisplay(colMode5, overscan, packMethod,
                               "ImageData", nbLines)
```

## Decompression Routines

```go
// Generate decompression code
err := aw.GenerateDepack(PackZX0, "DisplayImage")

// Specific decompressors
err := aw.GenerateLZWDepack("MainProgram")
err := aw.GenerateZX0Depack("ShowImage")
err := aw.GenerateZX1Depack("LoadData")
```

## Palette Generation

### Basic Palette

```go
params := asmgen.PaletteParams{
    CPCPlus:     false,
    ModeVirtuel: 0,
    DisableState: nil, // All colors enabled
}

err := aw.GeneratePalette(palette, params, true, true, "MainPalette")
```

### Advanced Palette Effects

```go
// Animation sequences
err := aw.GenerateAnimationPalette(palette, frameCount, "AnimPalette")

// Palette transitions
err := aw.GeneratePaletteTransition(startPal, endPal, steps, "FadePalette")

// Format-specific palettes
err := aw.GenerateOCPPalette(palette, "OCPData")
err := aw.GenerateKitPalette(palette, "KitData")
```

## Split-Raster Effects

Split-raster allows multiple colors per scanline through precise timing:

```go
// Define split data structure
type SplitEntry struct {
    Enable   bool
    Color    int
    Length   int    // Duration in pixels
    Position int    // Horizontal position
}

type LigneSplit struct {
    ListeSplit []SplitEntry  // Up to 7 color splits per line
    NumPen     int           // Pen register to use
    Retard     int           // Timing delay in pixels
}

type SplitScreen struct {
    LignesSplit []LigneSplit  // One per scanline (272 max)
}

// Generate complete split-raster effect
err := aw.GenerateSplitRasterASM(splitScreen, compressedData, 0x8000)
```

### Timing System

The split-raster system uses cycle-exact timing:

- **1 NOP = 8 pixels** horizontally
- **64 cycles per scanline** total
- **Automatic delay calculation** with optimal instruction selection
- **Register usage optimization** to minimize timing overhead

### Delay Generation

```go
// Generate precise delays
hlModified, err := aw.generateDelay(pixels, crashBC, canUseHL)
```

The delay generator automatically selects optimal instruction sequences:
- Large delays: `LD BC,n / DEC BC / JR NZ` loops
- Medium delays: `DJNZ` loops
- Small delays: `EX (SP),HL`, `PUSH/POP IX`, `ADD HL,BC`
- Single cycles: `NOP`

## Data Formatting

### Automatic Optimization

```go
// Detects repeated values and uses DS directive
data := []byte{0xFF, 0xFF, 0xFF, 0xFF}
aw.WriteDB(data, 4, "FillData", false)
// Generates: DS 4,#FF

// Regular data uses DB directives
data = []byte{0x01, 0x02, 0x03}
aw.WriteDB(data, 3, "VarData", false)
// Generates: DB #01,#02,#03
```

### Sectioned Output

```go
// Generate data with section labels
aw.WriteDBWithLabels(largeData, 16, 10, "DataSection", true)
// Creates: DataSection00, DataSection01, etc.
```

## Helper Functions

### Screen Format Generation

Automatically handles different screen configurations:

```go
// Generates CRTC register setup for custom screen sizes
err := aw.GenerateScreenFormat()
```

### Keyboard Input

```go
// Generates standard CPC keyboard reading routine
err := aw.GenerateWaitSpace(wait, withoutRet)
```

### CPC Plus Initialization

```go
// ASIC unlock and palette setup
err := aw.GenerateInitPlus("PaletteName")
err := aw.GenerateInitOld("PaletteName")
```

## Output Examples

### Generated Assembly Structure

```assembly
; Generated by ConvImgCpc v1.0 Go - 01/02/2024 (15 04 05)
; Screen mode - Mode 0 - 160x200, 16 colors
; Size (nbColsxNbRows) 160x200

	ORG	#8000
	RUN	$

Start:
	DI			; Disable interrupts
	LD	HL,ImageData	; Load image address
	LD	DE,#C000	; Screen destination
	CALL	Depack		; Decompress image

; Palette data
Palette:
	DB	#54,#5C,#06,#1E,#0C,#16,#1C,#5E,#4E,#0E,#4C,#0F,#5F,#1F,#0E,#07,#8C

; Decompression routine
Depack:
	ld	bc,#ffff		; preserve default offset 1
	push	bc
	; ... (complete ZX0 decompressor)

; Image data
ImageData:
	DB	#01,#02,#03,#04,#05,#06,#07,#08
	; Total size 8 bytes
```

## Integration

This package integrates seamlessly with:

- `pkg/fileio` - For embedding generated code in SCR files
- `pkg/compress` - For handling compressed data
- `pkg/cpc` - For CPC-specific constants and utilities

## Performance

- **Optimal instruction selection** for timing-critical code
- **Register allocation awareness** for split-raster effects
- **Automatic loop optimization** for large delays
- **Memory-efficient data structures** for large images

## Compatibility

Generates assembly compatible with:
- **Maxam assembler** (CPC native)
- **RASM** (modern cross-assembler)
- **WinAPE** assembler
- **Standard Z80 syntax** with CPC-specific addressing