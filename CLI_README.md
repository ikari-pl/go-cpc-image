# go-cpc-image CLI Documentation

The go-cpc-image command-line interface provides powerful image conversion capabilities for Amstrad CPC formats.

## Installation

Build the CLI from source:

```bash
go build ./cmd/cpc-image/
```

This creates the `cpc-image` executable in the project root.

## Commands

### convert - Convert images to CPC formats

Convert bitmap images (PNG, JPEG, BMP, GIF) to Amstrad CPC screen formats.

**Basic Usage:**
```bash
cpc-image convert -i photo.png -o screen.scr -m 1
```

**Flags:**
- `-i, --input` - Input image file (required)
- `-o, --output` - Output file (default: `gfx.scr`)
- `-m, --mode` - CPC mode: 0, 1, or 2 (default: 1)
- `--overscan` - Enable overscan mode (96x272 pixels)
- `--plus` - Use CPC Plus mode (4096 colors instead of 27)
- `-d, --dither` - Dithering method (default: `floyd-steinberg`)
- `--dither-pct` - Dithering percentage 0-100 (default: 50)
- `-f, --format` - Output format: scr, asm, dsk, png (default: `scr`)
- `--palette` - Lock palette from .pal file

**Available Dithering Methods:**
- `floyd-steinberg` - Floyd-Steinberg error diffusion
- `bayer1`, `bayer2`, `bayer3` - Bayer matrix dithering (2x2, 4x4, 4x4)
- `ordered1`, `ordered2`, `ordered3` - Ordered dithering (2x2, 3x3, 4x4)
- `zigzag1`, `zigzag2`, `zigzag3` - ZigZag patterns (3x3, 4x3, 5x4)
- `none` - No dithering

**Examples:**
```bash
# Convert to Mode 1 with Floyd-Steinberg dithering
cpc-image convert -i image.png -o screen.scr -m 1 -d floyd-steinberg

# Convert to Mode 0 with overscan and CPC Plus
cpc-image convert -i photo.jpg -o screen.scr -m 0 --overscan --plus

# Generate assembly source code
cpc-image convert -i image.bmp -o screen.asm -f asm -m 1

# Use Bayer dithering at 75%
cpc-image convert -i pic.png -o screen.scr -d bayer2 --dither-pct 75
```

### pack - Compress binary files

Compress binary files using various compression algorithms.

**Basic Usage:**
```bash
cpc-image pack -i data.bin -o data.zx0 --method zx0
```

**Flags:**
- `-i, --input` - Input binary file (required)
- `-o, --output` - Output file (required)
- `--method` - Compression method (default: `zx0`)

**Available Compression Methods:**
- `zx0` - ZX0 optimal compression
- `zx0v2` - ZX0 version 2
- `zx1` - ZX1 optimal compression
- `lzw` - LZW compression

**Examples:**
```bash
# Compress screen file with ZX0
cpc-image pack -i screen.scr -o screen.zx0 --method zx0

# Compress with ZX1
cpc-image pack -i data.bin -o data.zx1 --method zx1
```

### info - Display file information

Show metadata about CPC files including AMSDOS headers, palette information, and file dimensions.

**Basic Usage:**
```bash
cpc-image info screen.scr
```

**Supported File Types:**
- `.scr` - CPC screen files
- `.dsk` - Disk image files
- `.pal` - Palette files

**Examples:**
```bash
# Show SCR file information
cpc-image info myscreen.scr

# Display disk information
cpc-image info mydisk.dsk
```

### palette - Extract or convert palettes

Extract palette from SCR files or convert between palette formats.

**Basic Usage:**
```bash
cpc-image palette --extract screen.scr
cpc-image palette --convert input.pal output.kit
```

**Flags:**
- `--extract` - Extract palette from SCR file
- `--convert` - Convert between palette formats (requires input and output arguments)

**Examples:**
```bash
# Extract palette from screen
cpc-image palette --extract screen.scr

# Convert palette format
cpc-image palette --convert mypalette.pal mypalette.kit
```

## Global Flags

- `-v, --verbose` - Enable verbose output showing detailed conversion information

## CPC Mode Reference

**Mode 0:** 160x200 pixels, 16 colors from palette, 2 pixels per byte
**Mode 1:** 320x200 pixels, 4 colors from palette, 4 pixels per byte (most common)
**Mode 2:** 640x200 pixels, 2 colors from palette, 8 pixels per byte

**Overscan:** Increases resolution to 96x272 columns/lines (768x544 pixels in Mode 1)
**CPC Plus:** Uses 4096 color palette instead of standard 27-color palette

## Examples by Use Case

### Basic Image Conversion
```bash
# Convert a photo to CPC Mode 1 format
cpc-image convert -i holiday.jpg -o holiday.scr -m 1
```

### High Quality Conversion
```bash
# Use CPC Plus with minimal dithering for best quality
cpc-image convert -i artwork.png -o artwork.scr -m 1 --plus -d floyd-steinberg --dither-pct 25
```

### Overscan Graphics
```bash
# Create overscan screen with Mode 0 for maximum colors
cpc-image convert -i background.png -o background.scr -m 0 --overscan --plus
```

### Assembly Development
```bash
# Generate assembly source for inclusion in Z80 projects
cpc-image convert -i sprite.png -o sprite.asm -f asm -m 1
```

### File Compression
```bash
# Convert and compress for minimum file size
cpc-image convert -i image.png -o screen.scr -m 1
cpc-image pack -i screen.scr -o screen.zx0 --method zx0
```

## Output Summary

After successful conversion, the CLI displays:
- Input file information (dimensions, format)
- Output file information (size, format)
- Conversion parameters (mode, dithering)
- Number of colors used
- Processing time (with `--verbose`)

This provides immediate feedback on the conversion quality and file characteristics.