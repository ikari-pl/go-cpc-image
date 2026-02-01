# File I/O and Z80 Assembly Generation Implementation

This document summarizes the implementation of Phases 2 and 5 of go-cpc-image, covering file I/O operations and Z80 assembly generation.

## Completed Modules

### 1. File I/O Package (`pkg/fileio/`)

#### `scr.go` - SCR File Operations
- **Complete SCR file creation** with AMSDOS headers
- **All Z80 embedded code arrays** copied exactly from C# SauveImage.cs:
  - `CodeStd` - Standard CPC display routine (39 bytes)
  - `CodeP0`, `CodeP1`, `CodeP3` - CPC Plus display routines
  - `CodeOv`, `CodeOvP` - Overscan display routines
  - `codeEgx0`, `codeEgx1` - EGX mode display routines
  - `codeDepack` - Standard LZW decompression (171 bytes)
  - `codeDZX0`, `codeDZX0_V2`, `codeDZX1` - ZX0 family decompressors
- **Poke16 address patching** for runtime code modification
- **Multiple compression support**: None, LZW, ZX0, ZX0v2, ZX1, ZX0Ovs
- **Multiple output formats**: Binary, Assembler, DSK
- **Standard and overscan mode support**
- **CPC and CPC Plus compatibility**

#### `palette_io.go` - Palette File Operations
- **PAL format support** (CPC standard palette files)
- **KIT format support** (CPC Plus palette files)
- **CpcVGA lookup table** for palette conversion
- **Automatic format detection** and validation
- **AMSDOS header creation** for palette files
- **Round-trip compatibility** with original C# implementation

### 2. Assembly Generation Package (`pkg/asmgen/`)

#### `util.go` - Assembly Generation Utilities
- **ASMWriter structure** for structured Z80 assembly output
- **DB/DW data generation** with automatic DS optimization
- **Label and section management**
- **Header generation** with version and mode information
- **Automatic line formatting** and commenting
- **Sectioned data output** with auto-labeling

#### `display.go` - Display Routine Generation
- **Standard CPC display code** generation
- **EGX mode support** with raster effects
- **Mode X display** for color cycling effects
- **CPC Plus ASIC initialization** with unlock sequences
- **Screen format generation** for custom resolutions
- **Keyboard input routines** (space key detection)
- **VBI synchronization** code
- **Overscan and standard screen handling**

#### `depack.go` - Decompression Code Generation
- **LZW decompression routine** (standard CPC compression)
- **ZX0 decompression routine** (modern optimal compression)
- **ZX0v2 decompression routine** (enhanced version)
- **ZX1 decompression routine** (alternative format)
- **Proper label management** and jump target customization
- **Elias gamma decoding** implementation
- **Memory-efficient decompression** code

#### `palette.go` - Palette Data Generation
- **CPC standard palette** generation
- **CPC Plus palette** generation with hardware format
- **Animation palette sequences**
- **Palette transition effects** with interpolation
- **OCP+ format support**
- **KIT format support**
- **Split-raster palette data** for advanced effects
- **Disabled color handling** for special effects

#### `split.go` - Split-Raster Assembly Generation
- **Cycle-exact timing** for horizontal color splits
- **NOP padding generation** with optimal instruction selection
- **Multi-color scanlines** (up to 7 colors per line)
- **Automatic delay calculation** using various instruction types:
  - Large delays: BC loops (7 cycles per iteration)
  - Medium delays: DJNZ loops (4 cycles per iteration)
  - Small delays: EX (SP),HL, PUSH/POP IX, ADD HL,BC
  - Single cycles: NOP instructions
- **Register usage optimization** to minimize timing overhead
- **Complete split-raster framework** with initialization and main loop
- **ZX0 decompression integration** for compressed background images

## Key Features Implemented

### 1. Exact C# Compatibility
- **All Z80 machine code arrays** copied byte-for-byte from C# source
- **Identical AMSDOS header generation** using existing CPC package
- **Same memory layout** and address calculations
- **Identical file format outputs**

### 2. Advanced Assembly Generation
- **Structured ASM output** with proper formatting and comments
- **Automatic optimization** (DS for repeated data, register reuse)
- **Flexible label management** and section organization
- **Multiple compression method support**

### 3. Split-Raster Implementation
- **Cycle-exact timing calculations** for CPC hardware
- **Automatic instruction selection** for optimal delays
- **Up to 7 colors per scanline** with proper timing
- **Complete framework** from initialization to display loop

### 4. Comprehensive Testing
- **Unit tests** for core functionality
- **Integration examples** showing real-world usage
- **Format validation** for generated files
- **Cross-compatibility testing** with original C# tool

## File Structure

```
pkg/
├── fileio/
│   ├── scr.go              # SCR file operations with embedded Z80 code
│   ├── palette_io.go       # PAL and KIT palette file formats
│   ├── scr_test.go         # Unit tests
│   └── README.md           # Package documentation
└── asmgen/
    ├── util.go             # Assembly generation utilities
    ├── display.go          # Display routine generation
    ├── depack.go           # Decompression routine generation
    ├── palette.go          # Palette data generation
    ├── split.go            # Split-raster code generation
    ├── util_test.go        # Unit tests
    └── README.md           # Package documentation

examples/
└── scr_example.go          # Usage examples
```

## Technical Achievements

### 1. Z80 Machine Code Preservation
All embedded Z80 routines maintain exact byte compatibility:
- Display initialization routines
- Decompression algorithms
- ASIC unlock sequences
- Keyboard reading routines
- VBI synchronization code

### 2. Advanced Timing System
The split-raster system implements cycle-exact timing:
- 1 NOP = 8 pixels horizontal timing
- Automatic delay calculation with optimal instruction selection
- Register allocation awareness for minimal overhead
- Support for up to 7 color changes per scanline

### 3. Comprehensive Format Support
- SCR files with AMSDOS headers and embedded code
- PAL files for standard CPC palette storage
- KIT files for CPC Plus palette storage
- Multiple compression formats (LZW, ZX0 family)
- Assembly source code generation

### 4. Integration Ready
- Seamless integration with existing CPC package for AMSDOS headers
- Compatible with compress package for compression algorithms
- Ready for integration with bitmap and convert packages
- Maintains exact API compatibility with C# version

## Usage Examples

The implementation includes comprehensive examples showing:
- SCR file creation with embedded display code
- Assembly generation for custom display routines
- Palette file operations
- Split-raster effect creation
- Integration with existing Go modules

## Next Steps

This implementation provides the foundation for:
1. **Complete image conversion pipeline** integration
2. **Advanced effect generation** (animations, transitions)
3. **Custom compression algorithm** integration
4. **Real-time preview** and debugging tools
5. **Cross-platform file format** support

The modules are now ready for integration with the main go-cpc-image application and provide a solid foundation for the complete Go rewrite.