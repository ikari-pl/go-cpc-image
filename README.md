# Go CPC image!

A Go application for converting images to Amstrad CPC formats.

## Phase 2 Completion Status ✅

This represents the completion of Phase 2 of the Go rewrite plan, focusing on DSK disk image handling and compression algorithms.

## Phase 1 Completion Status ✅

Phase 1 focused on core CPC hardware constants and types.

### Completed Components

1. **Go Module** - `/go.mod`
   - Initialized with `github.com/ikari/go-cpc-image`
   - Go 1.21+ compatibility

2. **Color Handling** - `pkg/bitmap/color.go`
   - `RvbColor` struct (ported from `RvbColor.cs`)
   - RGB/ARGB color conversion functions
   - French naming convention preserved ('V' = vert/green)

3. **Bitmap Handling** - `pkg/bitmap/bitmap.go`
   - `DirectBitmap` struct using `image.RGBA` internally
   - Pixel manipulation methods compatible with C# version
   - File loading support for PNG/JPEG/GIF

4. **CPC Hardware Constants** - `pkg/cpc/`

   **Palette** - `palette.go`:
   - 27-color CPC palette (`RgbCPC[27]`)
   - CPC+ 4096-color support
   - VGA mapping string (`CpcVGA`)
   - Color conversion functions (`GetPalCPC`, `GetColor`)
   - Pen color detection (`GetPenColor`)

   **Screen Modes** - `modes.go`:
   - Mode pixel encoding tables (`TabOctetMode`)
   - Virtual mode names (`ModesVirtuels`)
   - Pixel calculation functions (`CalcTx`, `MaxPen`)
   - Support for all CPC modes (0/1/2, EGX, Mode X, Split, ASCII)

   **Memory Layout** - `memory.go`:
   - CPC screen addressing (`GetAdrCpc`)
   - Standard vs Overscan screen configurations
   - Screen geometry calculations
   - Memory size calculations

   **AMSDOS Headers** - `amsdos.go`:
   - `CpcAmsdos` struct with exact binary layout
   - Header creation (`CreeEntete`)
   - Checksum calculation and verification
   - Binary serialization using `encoding/binary`

5. **Conversion Parameters** - `pkg/convert/params.go`
   - `Param` struct (ported from `Param.cs`)
   - All conversion settings and options
   - Default parameter initialization
   - Screen configuration helpers

### Key Design Decisions

- **No static mutable state** - All CPC constants are in immutable structures
- **Idiomatic Go** - Exported types, proper package structure
- **Binary compatibility** - AMSDOS headers maintain exact byte layout
- **Error handling** - Go errors instead of exceptions
- **Standard library** - Uses `image`, `encoding/binary`, no external dependencies

### File Structure
```
go-cpc-image/
├── go.mod
├── cmd/
│   └── test/main.go          # Test compilation
├── pkg/
│   ├── bitmap/               # Image handling
│   │   ├── color.go          # RvbColor
│   │   └── bitmap.go         # DirectBitmap
│   ├── cpc/                  # CPC hardware
│   │   ├── palette.go        # 27-color palette + CPC+
│   │   ├── modes.go          # Screen modes & pixel encoding
│   │   ├── memory.go         # Screen memory layout
│   │   └── amsdos.go         # AMSDOS header format
│   ├── convert/
│   │   └── params.go         # Conversion parameters
│   ├── fileio/               # ✅ Phase 2
│   │   └── dsk.go            # DSK disk image handler
│   ├── compress/             # ✅ Phase 2
│   │   ├── compress.go       # Unified compression interface
│   │   ├── lzw.go            # Standard LZW compression
│   │   ├── zx0.go            # ZX0 optimal parser
│   │   └── zx1.go            # ZX1 optimal parser
│   └── test_implementation.go # Phase 2 testing
└── README.md
```

## Phase 2 Completed Components ✅

6. **DSK Disk Image Handler** - `pkg/fileio/dsk.go`
   - Complete port of `GestDSK.cs`
   - Full EDSK format support with proper header management
   - `FormatDsk()` - Initialize empty DSK with default track structure
   - `Load()/Save()` - Read/write DSK files with binary serialization
   - `CopieFichier()` - Copy files to DSK with AMSDOS directory management
   - Sector interleaving (0, 4, 1, 5, 2, 6, 3, 7, 8) as per CPC standard
   - Directory management with 64 entries and AMSDOS filename handling
   - Block allocation with bitmap-based tracking
   - Multi-head support for single/double-sided disks
   - Byte-compatible with C# version using `binary.LittleEndian`

7. **Compression Algorithms** - `pkg/compress/`

   **LZW Standard** - `lzw.go`:
   - `PackStd()/DepackStd()` - Standard LZW compression/decompression
   - Sliding window match table management (4KB)
   - Multiple encoding formats (8-bit to 16-bit offsets)

   **ZX0 Optimal Parser** - `zx0.go`:
   - `PackZX0()` - Optimal ZX0 compression with v1/v2 support
   - Reference counting memory management
   - Elias gamma encoding for lengths and offsets
   - Interlaced bit encoding, max offset 32640 bytes

   **ZX1 Optimal Parser** - `zx1.go`:
   - `PackZX1()` - Optimal ZX1 compression
   - Different offset encoding (8-bit vs 16-bit)
   - Modified Elias gamma encoding, max offset 32512 bytes

   **Unified Interface** - `compress.go`:
   - `Compressor` struct with unified access to all methods
   - Support for Standard, ZX0, ZX0v2, ZX1, ZX0Ovs
   - ZX0Ovs: Special screen overlay handling (15 sections)

### Phase 2 Key Features

- **Binary Compatibility**: All magic numbers and byte-level logic exactly as in C#
- **Proper Error Handling**: Go's explicit error returns with custom error types
- **Memory Safety**: Bounds checking and proper slice allocation
- **LittleEndian Operations**: Consistent use of `encoding/binary.LittleEndian`
- **Reference Counting**: Optimal parser memory management for compression

### Next Steps (Phase 3)
- Core conversion pipeline (`pkg/convert/`)
- Dithering algorithms
- Standard mode converters
- CLI application

### Testing

To verify Phase 1 + Phase 2:
```bash
cd /Users/ikari/src/cpc/go-cpc-image
go mod tidy
go build ./...
go run cmd/test/main.go         # Phase 1 testing
go run pkg/test_implementation.go  # Phase 2 testing
```

**Phase 2 Test Coverage:**
- DSK creation, file addition, and saving
- All compression algorithms (Standard LZW, ZX0, ZX0v2, ZX1, ZX0Ovs)
- Round-trip compression/decompression validation
- Error handling and edge cases

### Usage Example (Phase 2)

```go
package main

import (
    "github.com/ikari/go-cpc-image/pkg/fileio"
    "github.com/ikari/go-cpc-image/pkg/compress"
)

func main() {
    // Create and format a new DSK
    dsk := fileio.NewGestDSK()

    // Add a file to the DSK
    data := []byte("Hello, CPC World!")
    err := dsk.CopieFichier(data, "HELLO.BIN", len(data), 180, 0)
    if err == fileio.ErrNoErr {
        dsk.Save("output.dsk")
    }

    // Compress data using ZX0
    compressor := compress.NewCompressor()
    output := make([]byte, len(data)*2)
    size, err := compressor.Pack(data, len(data), output, 0, compress.MethodZX0)
}
```

All magic numbers, lookup tables, and byte arrays have been ported exactly from the C# source to maintain compatibility. The DSK handler and compression algorithms are now fully functional and byte-compatible with the original C# implementation.