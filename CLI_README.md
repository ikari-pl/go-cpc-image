# go-cpc-image CLI Documentation

The go-cpc-image command-line interface provides powerful image conversion capabilities for Amstrad CPC formats.

## Installation

Build the CLI from source:

```bash
go build ./cmd/cpc-image/
```

This creates the `cpc-image` executable in the project root.

## Global flags

- `--version` - print the release version (`convimgcpc version 1.3.0`) and exit
- `-v, --verbose` - print a banner naming the version and the details of every step

## Commands

### convert - Convert images to CPC formats

Convert bitmap images (PNG, JPEG, BMP, GIF) to Amstrad CPC screen formats.

An input file with a `.scr` extension is treated as an Amstrad CPC screen and
converted the other way, to an **indexed-color PNG**: the palette follows the
exact pen order stored in the screen (its ModePal block), so colours are
preserved instead of being taken from a fixed default. A **PKS-compressed**
screen is unpacked first — the PKS family is what `pack` writes — so
`samples/**/pk_compressed/*.SCR` can be rendered directly; only a variant the
engine does not implement is refused.

A **`.cmp` input** — the original tool's "écran compacté": an AMSDOS header plus
the screen compressed with its *Standard* method, the same LZW the PKS core uses —
is decompressed first and rendered exactly like the raw screen it holds. The
method is not recorded in the file, so a save made with the ZX0 or ZX1 methods
cannot be told apart from a corrupt stream: it is refused with a message saying so
rather than rendered as garbage.

The geometry comes from the payload size and the whole payload is rendered: a
16336-byte dump is a standard 80x200 screen, a 15872-byte one an underscan
64x192 screen, and a 31936-byte one an **overscan** 96x272 screen (mode byte and
palette at `&600`) whose lines past 255 live in the second half of the file. A
leading AMSDOS header is stripped with the same rule `pack` uses — its fields
first, its checksum for confirmation — and a header with a problem is reported on
stderr before the file is read as it is. A payload of none of those sizes is
refused with a **non-zero exit** and nothing written, rather than guessed as a
standard screen: that is also what a file of zeros gets, whose first 128 bytes
used to pass for a header (0 sums to a stored 0) and which used to render as a
320x200 or 256x192 screen.

The PNG holds the screen's own picture — one pixel per CPC pixel, one row per CRTC
line — magnified by whole pixels so the mode's pixels come out square: 320x200 /
256x192 / 384x272 for modes 0 and 1 and 640x400 / 512x384 / 768x544 for mode 2 (a
mode-0 picture gains a column per pixel, a mode-2 one a row). Nothing is resampled
or blended, so no pen colour is invented or lost.

Every screen PNG also **declares the density of the grid it holds**, in the file's
`pHYs` chunk (unit 1, pixels per metre), so a viewer that honours the chunk has the
picture's real shape. The pair is the density of ONE PIXEL OF THE FILE, built from
the density of one CPC pixel — a mode-1 (square) pixel is taken as ~36 DPI, and the
modes divide the same pixel clock, so a mode-0 pixel is half that across and a mode-2
one twice it — multiplied by however many file pixels that CPC pixel was written as.
Names aside, the rule is: a mode-1 grid declares 1416 x 1416; a mode-0 default grid
declares the same, because writing its pixels twice across squares the 708 it starts
from; a mode-2 default grid declares 2832 x 2832, since it doubles every CPC line
and so puts one file pixel on half the height of a CPC one; a `--native` grid declares the
mode's own pair (708 x 1416, 1416 x 1416, 2832 x 1416), and `--scale N` multiplies
both values, because magnifying the file does not magnify the screen. Nothing else is
claimed — a density, never a size — and most viewers ignore the chunk, which is why
the PAR grid is still the shape that travels. `-v` reports what was declared.

**Basic Usage:**
```bash
cpc-image convert -i photo.png -o screen.scr -m 1
```

**Flags:**
- `-i, --input` - Input image file (required)
- `-o, --output` - Output file (default: `gfx.scr`)
- `-m, --mode` - CPC mode: 0, 1, or 2 (default: 1)
- `--geometry` - Screen geometry: `auto` (default), `standard` (80x200),
  `underscan` (64x192) or `overscan` (96x272), case-insensitive. `--overscan` is
  kept as an alias of `--geometry=overscan`; passing it with a different
  `--geometry` is an error
- `--resize` - How the source is mapped onto the screen: `fit` (stretch to fill,
  default), `keepsmaller` (keep aspect, centre, and fill the rest with pen 0),
  `keeplarger` (keep aspect, centre, and crop) or `origin` (no scaling: draw at
  0,0 and clip), case-insensitive
- `--native` - Write a screen PNG at the screen's own resolution instead of the
  PAR-pre-compensated grid (raw pixels: the grid itself carries no aspect correction,
  so a viewer that ignores the declared pixel density shows it stretched)
- `--scale` - Zoom a screen PNG by a whole factor (default: 1), applied after the
  PAR step
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

# Underscan (64x192): a dump this tool can also write now, and pack as PKUL
cpc-image convert -i pic.png -o screen.scr -m 1 --geometry underscan

# Let the source's size choose the scan area: 256x192 reads back as mode-1 underscan
cpc-image convert -i screen_underscan.png -o screen.scr -m 1 --geometry auto

# Letterbox a square picture with pen 0 instead of stretching it
cpc-image convert -i logo.png -o screen.scr -m 1 --resize keepsmaller

# Write the screen PNG as raw pixels, twice as large
cpc-image convert -i screen.scr -o screen.png --native --scale 2
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

**Screen geometry:** `--geometry auto` (the default) reads the scan area from the
input PNG's pixel size, because that is the only signal a bitmap carries. For the
`-m/--mode` given, a size matches

1. the size the renderer shows a screen at (its display canvas) — 640x400 standard,
   512x384 underscan, 768x544 overscan (these do not depend on the mode);
2. the grid this tool emits a screen PNG with — 320x200, 256x192 and 384x272 for
   modes 0 and 1, the canvases above for mode 2;
3. the screen's own native grid — 160x200, 128x192 and 192x272 for mode 0;
   320x200, 256x192 and 384x272 for mode 1; 640x200, 512x192 and 768x272 for mode 2.

The three families never name two scan areas in one mode, so the choice is exact
rather than a guess, and re-converting a PNG this tool produced returns the
geometry it came from. A size that matches nothing falls back to a standard 80x200
screen, which `-v` reports. `-m/--mode` stays explicit: a 640x400 canvas is a
standard screen in every mode, so the mode cannot come from the size.

`--resize` chooses how the source is mapped onto the grid the geometry selected
(see the flag list). `--native` and `--scale` apply to the PNG a screen is
rendered to — not to a `.scr` output, which has one size.

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
- `--palette` - Palette file to embed: a `.pal` (classic) or a `.kit` (CPC Plus)

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

**PKS palettes:** `--palette` accepts a `.pal` (classic ink indices) or a `.kit`
(CPC Plus 12-bit colours); the container decides how the file is decoded, and
either one can drive either kind of screen.

- The classic `L` variants (`PKSL`, `PKUL`) embed a 17-byte ModePal field
  (mode + 16 inks): their payload is column-major pixel data, so the palette has
  nowhere else to live. By default `pack` takes it from the input screen's own
  ModePal at `&17D0`; `--palette` overrides it.
- The CPC Plus variants (`PKSP`, `PKVP`, `PKUP`) have no palette field at all:
  their 33-byte block travels inside the packed screen dump, at `&17D0` exactly
  as a raw save keeps it (`&600` for overscan). `--palette <file.kit>` writes that
  block into the dump before compressing, so unpacking restores it byte for byte —
  the `0xFFFF` "unused pen" markers included. Without the flag the screen's own
  block is carried through untouched; a screen that has none gets a warning.
- The remaining variants (`PKS3`, `PKU3`, `PKVL`) have no palette field and pack
  the dump as it is, so `--palette` is ignored with a warning.

A 158-byte `.kit` (15 words, pens 1..15) is accepted next to the usual 160-byte
one, and a `.kit` is also recognised by its size, so a misnamed file still loads.
When a variant needs a palette and neither the screen nor `--palette` provides a
usable one, a warning is printed and the packed file keeps an empty palette.

**PKS payload layouts:** the `L` variants (`PKSL`, `PKUL`) compress the picture
alone, in column-major order and with the screen's own geometry — 16000 bytes for
an 80x200 screen, 12288 for an underscan 64x192 one. Every other variant (`PKS3`,
`PKSP`, `PKVL`, `PKVP`, `PKU3`, `PKUP`) compresses the screen dump as it is, its
layout gaps and palette block included.

**Compressing the screen, not the header:** every method — the whole PKS family
included — strips a leading AMSDOS header first, so the packed stream holds
exactly the payload a CPC decompressor writes back to memory (128 bytes less, and
no shift in the picture). A header is recognised by its **fields**: a printable
8.3 name, a file type of 0, 1 or 2, and a checksum that matches. The name is what
tells a header from whatever bytes happen to start a file — 128 zero bytes
satisfy a checksum comparison (0 sums to a stored 0) but have no name, which is
how a zeroed prefix used to lose its first 128 bytes to a "header" and render a
screen sized around them. For a **screen**, a header whose checksum a writer got
wrong is still accepted on its fields (binary file type, a length that matches the
file, a screen load address `&C000`/`&0200`, a payload of a screen size), because
refusing those would compress the header as picture data. A file that is not
screen-shaped is compressed as it is, header included. Either way the reading is
made visible: a zeroed prefix or a header with a field problem is reported on
stderr and the file is read as it is, while arbitrary data — a prefix with no
name-shaped text — stays silent. The reported ratio is measured against the
compressed payload, and `-v` says when a header was stripped.

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
selects PKS, an `MJH` payload selects OCP, a `.cmp` extension selects LZW (a
compressed screen is the only thing that extension can name), and everything else
is assumed to be LZW. A `.cmp` file — the original tool's "écran compacté", an
AMSDOS header plus a *Standard* (LZW) compressed screen — is read behind its
header, like the PKS payloads are; a save made with the ZX0 or ZX1 methods is
refused with an explanation, since the method is not recorded in the file.

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
the signature names. For `PKSL` and `PKUL` the payload is column-major pixel data
extracted with the screen's own geometry — 16000 bytes for a standard 80x200
screen, 12288 for an underscan 64x192 one — so it is scattered back with that
geometry and the embedded ModePal is re-written at `&17D0`, making the restored
`.SCR` self-describing (`&3FD0` bytes for a standard screen, `&3E00` for an
underscan one). `PKVL` / `PKVP` (overscan) and the other variants emit the payload
as it is. A `PKUL` payload of 16000 bytes — an underscan image padded inside the
standard frame, the layout this tool wrote before 1.3.0 — is **refused** with a
message that names it; re-pack the screen from its raw `.SCR` to get the 64x192
payload.

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

**Rejecting output names it cannot honour:** `unpack` writes the decompressed
bytes as they are — a screen dump for the PKS family, the plain stream for LZW and
OCP — so it never renders an image or assembles a file. An output name ending in
`.png`, `.jpg`, `.jpeg`, `.gif`, `.bmp`, `.asm` or `.dsk` is therefore refused
with a non-zero exit instead of being filled with raw bytes: a `.png` holding
16336 raw bytes opens in no viewer, and the mistake would only surface later, in
whatever read it. Render a restored screen with
`convert -i out.scr -o out.png`.

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

**Overscan:** Raises the resolution to 96x272 columns/lines (a 31936-byte dump)
**Underscan:** Lowers it to 64x192 columns/lines (a 15872-byte dump)
**CPC Plus:** Uses 4096 color palette instead of standard 27-color palette

SCR→PNG writes the screen's own picture with whole pixels duplicated where the
mode's pixels need it: 320x200 / 256x192 / 384x272 for modes 0 and 1, 640x400 /
512x384 / 768x544 for mode 2.

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