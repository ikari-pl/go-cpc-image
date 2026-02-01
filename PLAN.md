# go-cpc-image: Full Feature Parity Plan

## Current State

The Go port covers ~60% of core conversion functionality and ~20% of advanced features. The app name should be **go-cpc-image** throughout (not the old C# name).

## Phase 1: Fix Current Conversion Quality

### 1.1 Dithering Investigation (DONE)
- Pass 1 dithers against 27 CPC colors, Pass 2 picks nearest from 16-color palette
- This matches C# architecture exactly — ConvertStd.cs has no dithering
- Boundary-only dithering is expected when dither % is low or source colors map cleanly to CPC palette

### 1.2 Remaining Bugs
- [ ] Verify dithering with higher pct values (80-100%) produces visible whole-image dithering
- [ ] Test with real photographs (not synthetic gradients) for better color diversity

## Phase 2: Missing Conversion Modes

### 2.1 EGX Modes
- **EGX1**: Alternating Mode 0 (even lines) / Mode 1 (odd lines) — 16+4 colors
- **EGX2**: Alternating Mode 0 (even lines) / Mode 2 (odd lines) — 16+2 colors
- Implementation: ConvertStd already handles VirtualMode 3/4, but the full EGX conversion with proper per-line mode switching needs the `yEgx` parameter (0 or 2) wired through
- C# source: `ConvertBase.cs` lines handling `modeVirtuel >= 3`
- Files: `pkg/convert/standard.go` (add EGX-specific Pass2 logic)

### 2.2 ASC-UT Mode (Pre-calculated Patterns)
- Uses 4×4 trame matrices from `TramesAscUt.cs` (8 preset pattern sets, 16 patterns each)
- `ConvertAscUt.cs`: matches 8×8 source blocks against all 16 patterns
- Already partially ported in `pkg/convert/ascii.go` (`ConvertAscUt`), but:
  - [ ] Port the 8 preset pattern sets from `TramesAscUt.cs`
  - [ ] Port automatic pattern generation (`CnvTrame` function)
  - [ ] Wire up pattern set selection in UI

### 2.3 Sprite Capture Mode
- Extracts hardware sprites from screen areas
- C# source: `CaptureSprites.cs`
- Low priority — niche feature

## Phase 3: Missing Dithering Methods

### 3.1 Additional Dither Matrices
Already have: Floyd-Steinberg (2×2), Bayer 1/2/3, Ordered 1/2/3
Missing:
- [ ] ZigZag1 (3×3), ZigZag2 (4×3), ZigZag3 (5×4)
- [ ] Test patterns (test0-test9) — used for debugging, low priority
- C# source: `Dither.cs` — the matrix data is defined inline

### 3.2 Custom Dither Matrix Support
- Allow user-defined dither matrices
- Low priority

## Phase 4: Missing File Format Support

### 4.1 Import Formats
- [ ] **SNA** (Z80 snapshot) import — extract screen memory from snapshot files
- [ ] **OCP** compression (MJH header) — used by OCP Art Studio
- [ ] **PKS variants** (PKSL, PKS3, PKSP, PKVL, PKVP) — proprietary packed formats
- [ ] **Kit** palette format
- C# source: `SauveImage.cs`, `BitmapCpc.cs`

### 4.2 Compression
Already have: ZX0, ZX1, LZW
Missing:
- [ ] **OCP** (MJH) compression/decompression
- [ ] **ZX0Ovs** (overscan-optimized variant)
- [ ] **ZX0_V2** variant
- C# source: `PackModule.cs`

### 4.3 Export Enhancements
- [ ] Auto-numbered DSK files (CNVIMG00.SCR pattern)
- [ ] Better AMSDOS directory management

## Phase 5: CPC+ Hardware Features

### 5.1 Hardware Sprites
- 16×16 pixels, 16 colors per sprite
- 8 banks × 16 sprites = 128 sprites total
- Sprite magnification (×1, ×2, ×4)
- C# source: `EditSprites.cs`, `PosSpriteHard.cs`
- Files to create: `pkg/cpc/sprites.go`

### 5.2 ASIC Features
- [ ] ASIC unlock sequence generation
- [ ] Raster interrupt tables (`RasterTablePlus.cs`)
- [ ] Split-screen with CPC+ palette changes

## Phase 6: Assembly Code Generation

### 6.1 Already Implemented
- Basic palette init, display code, depack routines

### 6.2 Missing ASM Generators
- [ ] **EGX mode code** — scanline mode switching with timing-critical raster interrupts
- [ ] **Mode X VBL sync** — per-line palette updates synchronized to vertical blank
- [ ] **128K banking code** — memory bank switching for large screens/animations
- [ ] **Animation loop code** — frame sequencing with speed control
- [ ] **DrawBlock mode** — block-based screen updates
- [ ] **DrawAscii modes** — Full, Optimized, Delta character-based rendering
- [ ] **Full split-screen timing** — pixel-perfect NOP delay generation for raster splits
- C# source: `SaveAsm.cs`, `GenSplitAsm.cs`

## Phase 7: Image Processing Enhancements

### 7.1 K-Means Pre-filtering
- Already ported in `pkg/convert/palette.go` (`QuantizePalette`)
- [ ] Verify it matches C# behavior
- [ ] Wire up `KMeansColor` and `KMeansPass` parameters in GUI

### 7.2 Palette Algorithms
- [ ] **Alternative reduction** (`newReduc` flag) — find most-different among most-used colors
  - Already ported but needs testing
- [ ] **Palette sorting** (`newSortPal` parameter) — sort by brightness
  - Already ported but needs testing
- [ ] **Color locking** — lock specific palette entries during conversion
  - Partially implemented, needs full UI support
- [ ] **Color disabling** — exclude specific pens from use
  - Partially implemented

### 7.3 Resize Modes
C# supports 5 modes: Fit, KeepSmaller, KeepLarger, UserSize, Origin
- [ ] Port all resize modes (currently only basic fit)

## Phase 8: GUI / Editor Features

### 8.1 Current State
- Fyne-based GUI with image loading, mode selection, dithering slider, palette display
- Live preview with conversion

### 8.2 Missing Editor Features (Priority Order)
1. [ ] **Undo/Redo** — essential for any editor
2. [ ] **Drawing tools** — pencil, line, rectangle, fill, spray
3. [ ] **Palette editor** — click to change colors, lock/disable pens
4. [ ] **Zoom** — multiple zoom levels
5. [ ] **Grid overlay** — show CPC pixel boundaries
6. [ ] **Color picker** — pick color from image
7. [ ] **Split-screen editor** — visual split positioning, per-line palette
8. [ ] **ASCII pattern editor** — edit 4×4 trame patterns
9. [ ] **Sprite editor** — 16×16 sprite editing (CPC+)

### 8.3 UI Improvements
- [ ] Cancel-and-restart conversion (prevent UI freezing during rapid slider changes)
- [ ] Compact palette layout
- [ ] Status bar with pen usage count
- [ ] File browser with recent files
- [ ] Keyboard shortcuts

## Phase 9: Animation Support

### 9.1 Current State
- Basic delta compression and frame management ported

### 9.2 Missing
- [ ] **GIF import** — load animated GIFs as frame sequences
- [ ] **Frame editor** — insert, delete, reorder frames
- [ ] **Per-frame timing** — variable speed per frame
- [ ] **128K bank management** — spread frames across memory banks
- [ ] **Keyframe insertion** — periodic full frames among deltas
- [ ] **Animation ASM export** — complete playback code with bank switching

## Phase 10: Advanced Features

### 10.1 Batch Processing
- [ ] Convert multiple images with consistent palette
- [ ] Create frame sequences from numbered files

### 10.2 DSK Management
- [ ] List files on DSK
- [ ] Add/remove individual files
- [ ] 40/80 track support
- [ ] Extended DSK format

### 10.3 Emulator Integration
- [ ] SNA export for direct emulator testing
- [ ] Screen capture from emulator memory dumps

---

## Priority Ranking

**Critical (needed for basic usability):**
1. Verify dithering quality with real images
2. EGX modes
3. UI responsiveness (cancel-and-restart)

**High (common user workflows):**
4. Missing dither matrices (ZigZag)
5. SNA import
6. Drawing tools (pencil, line at minimum)
7. Undo/redo
8. Palette editor with lock/disable

**Medium (advanced users):**
9. ASC-UT patterns with presets
10. OCP/PKS compression formats
11. Full ASM generation (EGX, Mode X, Split)
12. Animation GIF import
13. Hardware sprites (CPC+)

**Low (niche features):**
14. Sprite capture mode
15. Custom dither matrices
16. Batch processing
17. Emulator integration
18. 128K banking
