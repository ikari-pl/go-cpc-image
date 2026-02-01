# ConvImgCpc Fyne GUI Application

This directory contains the Fyne-based GUI application for ConvImgCpc, providing a graphical interface that matches the original WinForms layout.

## Features

- **Split panel layout**: Source image (left) and CPC preview (right) with controls below
- **Live preview**: Real-time CPC conversion with 50ms debouncing for responsive UI
- **Mode controls**: CPC mode selector (0/1/2), overscan/standard resolution toggles
- **Dithering controls**: Method selector with all matrices from dither.go, percentage slider
- **Palette editor**: Interactive 16-pen palette with CPC 27-color picker or CPC+ 4096-color picker
- **File operations**: Open images (PNG/JPG/GIF/BMP/SCR), save SCR/ASM/DSK/PNG
- **Zoom controls**: 1x/2x/4x zoom on CPC preview with pixel-perfect scaling
- **Status bar**: Current mode, dimensions, color count, conversion time

## Architecture

The GUI follows the requirements in Phase 4 of GO_REWRITE_PLAN.md:

### Components

- **`main.go`**: Application entry point, creates Fyne app and main window
- **`app.go`**: Main application state, layout management, conversion orchestration
- **`preview.go`**: Split image preview with source and CPC display using `canvas.Raster`
- **`controls.go`**: Conversion parameter controls (mode, dithering, size, CPC+)
- **`palette.go`**: 16-pen palette grid with CPC color picker dialogs
- **`toolbar.go`**: File operations toolbar (open, save, export)

### Live Preview Architecture

```
Parameter change → Fyne data binding → 50ms debounce →
Goroutine conversion → Update CPC screen buffer →
canvas.Raster.Refresh() → UI update
```

### Thread Safety

- `sync.Mutex` protects shared CPC screen buffer between conversion goroutine and render callback
- Debounced conversion channel prevents UI blocking during rapid parameter changes
- All UI updates happen on the main thread via Fyne's binding system

## Building

1. **Add Fyne dependency**:
   ```bash
   go get fyne.io/fyne/v2
   go mod tidy
   ```

2. **Build the GUI**:
   ```bash
   go build ./cmd/convimgcpc-gui/
   ```

3. **Run the application**:
   ```bash
   ./convimgcpc-gui
   ```

## Package Dependencies

The GUI application depends on these existing packages:

- `pkg/convert`: Conversion pipeline, parameters, dithering
- `pkg/bitmap`: Image handling with DirectBitmap
- `pkg/cpc`: CPC hardware constants, palette, memory layout
- `pkg/render`: CPC screen rendering to RGB images
- `pkg/fileio`: SCR/DSK/ASM file I/O

## Implementation Notes

### English Only
All UI labels, comments, and strings are in English only, following the coding standards.

### Layout Matching
The layout closely matches the original WinForms design:
- CPC Resolution group (columns, lines, mode, Standard/Fullscreen buttons)
- Dithering group (method dropdown, percentage slider)
- Palette display (4×4 grid of pen slots with color previews)
- File toolbar (Open, Save, Export menu)

### Data Bindings
Fyne data bindings connect UI controls to the `convert.Param` structure:
- Mode selector → `params.ModeVirtuel`
- Dither method → `params.Methode`
- Dither percentage → `params.Pct`
- CPC+ checkbox → `params.CpcPlus`
- Size entries → `params.NbCols`, `params.NbLignes`

### Conversion Pipeline Integration
The GUI integrates with the existing conversion pipeline:
1. User loads image → `bitmap.NewDirectBitmapFromFile()`
2. Parameters change → Trigger debounced conversion
3. Conversion → `convert.Convert(source, dest, params, noInfo)`
4. Render → `render.BitmapCpc.Render()` to get RGBA display image
5. Display → `canvas.Raster` refresh with new image data