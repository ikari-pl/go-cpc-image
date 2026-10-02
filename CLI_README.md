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
- `--palette` - Lock the palette from a file: `.pal` (classic) or `.kit` (CPC Plus)

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

# Lock the pens to a .pal file (pen i = entry i), instead of letting the
# converter choose them by pixel frequency
cpc-image convert -i pic.png -o screen.scr -m 1 --palette screen.pal

# SCR -> PNG honouring an external palette instead of the one in the screen
cpc-image convert -i screen.scr -o screen.png --palette screen.pal
```

**Source palette handling:** `convert` honours a source palette, in this order:
1. `--palette <file>` — the file wins, in BOTH directions (PNG→SCR and SCR→PNG).
   Each of its 16 pens is locked, so the SCR/PKS embeds exactly those colours in
   that order. `.pal` holds classic ink indices, `.kit` holds CPC Plus 12-bit
   colours; the container is picked from the extension.
2. Otherwise, an **indexed PNG** (a `PLTE`-carrying image, such as the one
   `convert` itself writes for SCR→PNG) is honoured when it holds no more
   entries than the selected mode supports (mode 0 → 16, 1 → 4, 2 → 2): pen i
   takes entry i, so the pen order survives the round trip. An entry that is not
   exactly representable is snapped to the nearest CPC colour (a classic ink, or
   the nearest 12-bit `0x0VBR` under `--plus`) and the count is warned about.
3. Otherwise the truecolor path is used, with the pens assigned by frequency.

A PNG whose palette holds MORE entries than the mode supports cannot be
represented: a warning is printed and the image is converted exactly like a
truecolor one (its palette is ignored). When the input is an SCR and `--palette`
is given, the screen's own ModePal is ignored — `-v` reports
`using palette from <file> (SCR palette ignored)`.

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
- `--palette` - Palette file (`.pal`) to embed in the PKSL header

**Available Compression Methods:**
- `zx0` - ZX0 optimal compression
- `zx0v2` - ZX0 version 2
- `zx1` - ZX1 optimal compression
- `lzw` - LZW compression
- `pks` - PKS packed screen, variant auto-detected from the input screen
- `pksl` - PKS 320x200 standard (classic palette)
- `pks3` - PKS 320x200 mode 3
- `pksp` - PKS 320x200 CPC Plus
- `pkvl` - PKS overscan standard
- `pkvp` - PKS overscan CPC Plus
- `pkul` - PKS underscan standard
- `pku3` - PKS underscan mode 3
- `pkup` - PKS underscan CPC Plus

**Examples:**
```bash
# Compress screen file with ZX0
cpc-image pack -i screen.scr -o screen.zx0 --method zx0

# Compress with ZX1
cpc-image pack -i data.bin -o data.zx1 --method zx1

# Pack a screen as PKSL (320x200 standard); the 17-byte ModePal is taken from
# the .SCR itself and embedded in the PKS header
cpc-image pack -i screen.scr -o screen.pks --method pksl

# ...or embed the palette from an external .pal file instead
cpc-image pack -i screen.scr -o screen.pks --method pksl --palette screen.pal

# Let pack pick the variant from the screen itself
cpc-image pack -i screen.scr -o screen.pks --method pks
```

**PKS variant auto-detection (`--method pks`):** the variant comes from the input
screen: an overscan dump (AMSDOS load address `&0200`, or a payload of a full 16K
bank) selects `PKVL`/`PKVP`, an underscan dump (a `&3E00` payload, i.e. the
64x192 screen) selects `PKUL`/`PKU3`/`PKUP`, the `0x80` bit of the mode byte at
`&17D0` (`&600` for overscan) marks a CPC Plus screen (`PKSP`/`PKVP`/`PKUP`), mode
3 combined with a valid ModePal selects `PKS3`/`PKU3`, and anything else stays
`PKSL`. `-v` prints the signals that were read, and a warning is printed when
none were found at all. Naming a variant explicitly (`--method pksl`, …) skips
the detection.

**PKS palettes:** only the `L` variants (`PKSL`, `PKUL`) embed a 17-byte ModePal
(mode + 16 inks): their payload is column-major pixel data, so the palette has
nowhere else to live. By default `pack` takes it from the input screen's own
ModePal at `&17D0`; `--palette` overrides it. The other variants pack the screen
dump as it is, palette block included — a CPC Plus palette is 33 bytes and
travels inside that dump, at `&17D0` exactly as in the raw saves, so `pkup` needs
no palette field and no header extension beyond its 4-byte signature. Those
variants (`pks3`, `pksp`, `pkvl`, `pkvp`, `pku3`, `pkup`) carry no palette field,
so the flag is ignored with a warning. When the screen has no usable (non-empty)
ModePal and no `--palette` is given, the header keeps an empty palette and a
warning is printed.

**Compressing the screen, not the header:** every method — the whole PKS family
included — strips a leading AMSDOS header first, so the packed stream holds
exactly the payload a CPC decompressor writes back to memory (128 bytes less, and
no shift in the picture). The header is recognised by its checksum or — when that
checksum is not one this tool validates — by its fields: binary file type, a
length that matches the file, a screen load address (`&C000` or `&0200`) and a
payload of a screen size. A file that is not screen-shaped is compressed as it
is, header included. The reported ratio is measured against the compressed
payload, and `-v` says when a header was stripped.

**Rejecting already-compressed input:** `pack` refuses to re-compress data that
is already compressed and exits with a **non-zero code** so scripts can react.
A PKS `PK` signature, an OCP `MJH` signature, or the compressed file
extensions `.zx0` / `.zx1` / `.lzw` / `.pks` / `.cmp` (LZW/ZX0/ZX1/PKS/CMP have
no magic signature) all cause an error. Such extensions are rejected
unconditionally — even when the file happens to carry a valid AMSDOS screen
header. For any other extension, a valid AMSDOS header with a screen load
 address is treated as a legit raw SCR.

### unpack - Decompress binary files

Decompress files compressed with LZW, OCP or the PKS family. When `--method` is
omitted, the format is **auto-detected** from the file header: a `PK` payload
selects PKS, an `MJH` payload selects OCP, and everything else is assumed to be
LZW.

**Basic Usage:**
```bash
# Auto-detected decompression
cpc-image unpack -i data.lzw -o data.bin

# Explicit method
cpc-image unpack -i data.ocp -o data.bin --method ocp

# PKS screen (variant read from the PKxx signature)
cpc-image unpack -i screen.pks -o screen.scr
```

**PKS family (`PK` payloads):** decompressed transparently, whichever variant
the signature names. For `PKSL` and `PKUL` the payload is column-major pixel
data: it is scattered back into the CPC screen layout and the embedded ModePal is
re-written at `&17D0`, so the restored `.SCR` is self-describing, like the
original screens — `PKUL` screens end at `&3E00` bytes (64x192) instead of the
standard `&3FD0`. `PKVL` / `PKVP` (overscan) and the other variants emit the
payload as it is.

**Rejecting raw (uncompressed) SCR input:** in auto-detect mode, `unpack` refuses
to operate on a raw (already-uncompressed) CPC screen dump: a valid AMSDOS header
with a screen load address (`&C000` or `&0200`), or a payload whose size matches
the standard (16336 bytes) or overscan (31936 bytes) bitmap, fails with a
non-zero exit. This prevents silently emitting garbage or echoing the input back.
With an explicit `--method`, the CLI keeps trying but still rejects a trivial
LZW round-trip (input echoed byte-for-byte).

**Rejecting unrecognized data:** LZW decompression of an unknown/non-LZW file can
succeed with a bogus output. To prevent emitting a corrupt file, `unpack` only
accepts LZW output whose size matches a recognized CPC screen payload
(16336 / 31936 / 16000 bytes); anything else fails with a non-zero exit.

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