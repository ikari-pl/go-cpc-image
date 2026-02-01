# CPC File I/O Package

This package provides comprehensive file I/O operations for Amstrad CPC image and palette formats, including SCR files with embedded Z80 code and AMSDOS headers.

## Features

### SCR Files (`scr.go`)
- **SCR file creation** with AMSDOS headers
- **Embedded Z80 display code** for self-displaying images
- **Multiple compression methods**: None, LZW, ZX0, ZX0v2, ZX1
- **CPC and CPC Plus support** with different code routines
- **Standard and overscan modes**
- **EGX mode support** (modes 3 and 4)

### Palette Files (`palette_io.go`)
- **PAL format** - Standard CPC palette files
- **KIT format** - CPC Plus palette files
- **Automatic format detection**
- **CPC and CPC Plus palette conversion**

## Z80 Code Arrays

The module includes exact byte-for-byte copies of Z80 machine code from the original C# version:

### Standard CPC Display Routines
- `CodeStd` - Standard CPC display initialization
- `CodeOv` - Overscan display routine
- `CodeP0`, `CodeP1`, `CodeP3` - CPC Plus display routines with ASIC unlock

### EGX Mode Routines
- `codeEgx0`, `codeEgx1` - Enhanced Graphics eXtension mode display

### Decompression Routines
- `codeDepack` - Standard LZW decompression
- `codeDZX0` - ZX0 decompression
- `codeDZX0_V2` - ZX0 version 2 decompression
- `codeDZX1` - ZX1 decompression

## Usage Examples

### Creating an SCR File

```go
// Create bitmap data
bitmap := make([]byte, 0x4000) // 16KB standard screen
// ... fill with image data

// Set parameters
params := fileio.SCRParams{
    WithPalette: true,
    WithCode:    true,
    CPCPlus:     false,
    ModeVirtuel: 0,
}

// Create palette
palette := make([]uint16, 16)
for i := range palette {
    palette[i] = uint16(i)
}

// Save SCR file
size, err := fileio.SaveSCR("image.scr", bitmap, len(bitmap),
                            fileio.PackZX0, fileio.OutputBinary,
                            params, palette, nil)
```

### Working with Palettes

```go
// Save palette in PAL format
err := fileio.SavePalette("image.pal", palette, params)

// Load palette
loadedPalette := make([]uint16, 16)
err = fileio.LoadPalette("image.pal", loadedPalette, &params)

// CPC Plus KIT format
err = fileio.SavePaletteKit("image.kit", palette)
err = fileio.LoadPaletteKit("image.kit", palette)
```

## Compression Methods

- `PackNone` - No compression
- `PackStandard` - Standard LZW compression
- `PackZX0` - ZX0 compression (recommended)
- `PackZX0V2` - ZX0 version 2
- `PackZX1` - ZX1 compression
- `PackZX0Ovs` - ZX0 with overscan post-processing

## Output Formats

- `OutputBinary` - Binary file with AMSDOS header
- `OutputAssembler` - Assembly source code
- `OutputDSK` - DSK disk image format

## Technical Details

### AMSDOS Integration
All files include proper AMSDOS headers using the existing `cpc.CreeEntete()` function from the CPC package.

### Address Patching
The `Poke16()` function handles little-endian 16-bit address patching in Z80 machine code, maintaining exact compatibility with the C# version.

### Memory Layout
- Standard mode: Screen at #C000, code at #C7D0
- Overscan mode: Screen at #0200, code varies by mode
- Compressed data: Positioned to decompress to correct addresses

### Error Handling
All functions return descriptive errors for debugging and user feedback.

## Dependencies

- `github.com/ikari/go-cpc-image/pkg/cpc` - AMSDOS header handling
- `github.com/ikari/go-cpc-image/pkg/compress` - Compression algorithms

## Compatibility

This package maintains byte-exact compatibility with the original C# tool, ensuring that generated files work identically on real CPC hardware and emulators.