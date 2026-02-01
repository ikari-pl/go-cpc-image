# Go CPC image!

Convert images to Amstrad CPC screen formats — standard modes, CPC+ 4096 colours, overscan, EGX, split-raster effects, animations, and more. Includes both a CLI and a Fyne-based GUI with live preview.

<img width="100%" alt="Go CPC Image! in action" src="https://github.com/user-attachments/assets/6144148c-9aaa-40fd-95d9-2bfa75af28c3" />

## Features

- **All CPC screen modes** — Mode 0 (16 colours), Mode 1 (4 colours), Mode 2 (2 colours)
- **CPC+ support** — full 4096-colour palette (12-bit RGB)
- **Overscan** — 96×272 column/line screens beyond the standard 80×200
- **EGX modes** — alternating Mode 0/1 or Mode 0/2 per scanline for extra colours
- **Mode X & split-raster** — per-line palette changes with cycle-exact timing
- **ASCII art mode** — pattern-based character conversion with trame presets
- **Dithering** — Floyd-Steinberg, Bayer, ordered, zigzag matrices at adjustable strength
- **Palette optimisation** — k-means quantisation, colour locking/disabling
- **Compression** — ZX0, ZX1, LZW, OCP, PKS
- **Z80 assembly generation** — self-displaying SCR files with embedded depack routines
- **DSK disk images** — format, add files, 40/80 track support
- **Animation** — GIF import, delta encoding, frame sequencing
- **Hardware sprites** — CPC+ 16×16 sprite encode/decode/transform
- **GUI** — Fyne-based editor with live preview, drawing tools, undo/redo, palette editor

## Quick start

```bash
# CLI — convert an image to CPC Mode 1
go build ./cmd/cpc-image/
./cpc-image convert -i photo.png -o screen.scr -m 1

# GUI — interactive editor with live preview
go build ./cmd/cpc-image-gui/
./cpc-image-gui
```

## CLI usage

```bash
# Basic conversion
cpc-image convert -i image.png -o screen.scr -m 0

# CPC+ with dithering
cpc-image convert -i photo.jpg -o screen.scr -m 1 --plus -d floyd-steinberg --dither-pct 40

# Overscan mode
cpc-image convert -i art.png -o screen.scr -m 0 --overscan

# Generate Z80 assembly
cpc-image convert -i sprite.png -o sprite.asm -f asm -m 1

# Compress a screen file
cpc-image pack -i screen.scr -o screen.zx0 --method zx0

# File info
cpc-image info screen.scr
```

## Building

Requires Go 1.21+. The CLI builds with no external dependencies (`CGO_ENABLED=0`). The GUI needs Fyne's system libraries:

```bash
# macOS — no extra deps needed
# Linux
sudo apt install libgl1-mesa-dev libx11-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev
# Windows — needs a C compiler (mingw-w64)
```

## Project structure

```
cmd/cpc-image/       CLI application
cmd/cpc-image-gui/   Fyne GUI application
pkg/bitmap/          Image handling (DirectBitmap, RgbColor)
pkg/cpc/             CPC hardware (palette, modes, memory, AMSDOS, sprites)
pkg/convert/         Conversion pipeline (dithering, quantisation, all modes)
pkg/render/          CPC screen rendering to RGBA
pkg/asmgen/          Z80 assembly code generation
pkg/fileio/          File I/O (SCR, DSK, SNA, GIF, PAL, KIT, project)
pkg/compress/        Compression (ZX0, ZX1, LZW, OCP, PKS)
pkg/animation/       Delta encoding and frame management
pkg/splitscreen/     Split-raster palette management
internal/gui/        GUI widgets (controls, palette editor, toolbar, preview)
```

## Test coverage

```
pkg/render       98.7%
pkg/splitscreen  97.3%
pkg/animation    94.2%
pkg/cpc          81.2%
pkg/bitmap       76.6%
pkg/convert      68.8%
pkg/compress     59.3%
pkg/fileio       48.8%
pkg/asmgen       42.4%
```

## Licence

MIT

---

### Thanks

:beer: Inspired by [ConvImgCpc](https://music.retroacpc.net/convimgcpc/) by Music — the original C# Amstrad CPC image converter that started it all.
