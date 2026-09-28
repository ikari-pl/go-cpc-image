// Package main provides the command-line interface for ConvImgCpc image conversion utility.
// This CLI application allows users to convert images to Amstrad CPC formats, compress files,
// extract information from CPC files, and manage palettes.
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"  // Register GIF decoder
	_ "image/jpeg" // Register JPEG decoder
	"image/png"    // PNG decoder and encoder
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
	"github.com/ikari-pl/go-cpc-image/pkg/compress"
	"github.com/ikari-pl/go-cpc-image/pkg/convert"
	"github.com/ikari-pl/go-cpc-image/pkg/cpc"
	"github.com/ikari-pl/go-cpc-image/pkg/fileio"
	"github.com/ikari-pl/go-cpc-image/pkg/render"
)

var (
	// Global flags
	verbose bool

	// Common conversion flags
	inputFile    string
	outputFile   string
	mode         int
	overscan     bool
	plus         bool
	ditherMethod string
	ditherPct    int
	format       string
	paletteFile  string

	// Compression flags
	compressionMethod string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "convimgcpc",
	Short: "Amstrad CPC image conversion utility",
	Long: `ConvImgCpc is a tool for converting images to Amstrad CPC formats.
It supports various CPC modes, dithering algorithms, compression methods,
and can generate .scr, .asm, .dsk, and .png files.

Examples:
  convimgcpc convert -i photo.png -o screen.scr -m 1
  convimgcpc pack -i data.bin -o data.zx0 --method zx0
  convimgcpc info screen.scr`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verbose {
			fmt.Println("ConvImgCpc Go version - verbose output enabled")
		}
	},
}

// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert images to CPC formats",
	Long: `Convert bitmap images (PNG, JPEG, BMP, GIF) to Amstrad CPC screen formats.
Supports various CPC modes (0, 1, 2), dithering algorithms, and output formats.

Examples:
  convimgcpc convert -i photo.png -o screen.scr -m 1
  convimgcpc convert -i image.jpg -o screen.scr -m 0 --overscan --plus
  convimgcpc convert -i pic.bmp -o screen.asm -f asm -d floyd-steinberg --dither-pct 75`,
	RunE: runConvert,
}

// packCmd represents the pack command
var packCmd = &cobra.Command{
	Use:   "pack",
	Short: "Compress binary files",
	Long: `Compress binary files using various compression algorithms.
Supported methods: zx0, zx0v2, zx1, lzw

Examples:
  convimgcpc pack -i data.bin -o data.zx0 --method zx0
  convimgcpc pack -i screen.scr -o screen.zx1 --method zx1`,
	RunE: runPack,
}

// unpackCmd represents the unpack command
var unpackCmd = &cobra.Command{
	Use:   "unpack",
	Short: "Decompress binary files",
	Long: `Decompress binary files compressed with various algorithms.
Supported methods: pks, lzw, ocp (auto-detected from header if omitted)

Examples:
  convimgcpc unpack -i data.pks -o data.bin
  convimgcpc unpack -i data.pks -o data.bin --method pks
  convimgcpc unpack -i data.lzw -o data.bin --method lzw`,
	RunE: runUnpack,
}

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info [file]",
	Short: "Show information about CPC files",
	Long: `Display metadata about CPC files including AMSDOS headers,
palette information, and file dimensions.

Supported file types: .scr, .dsk, .pal

Examples:
  convimgcpc info screen.scr
  convimgcpc info disk.dsk`,
	Args: cobra.ExactArgs(1),
	RunE: runInfo,
}

// paletteCmd represents the palette command
var paletteCmd = &cobra.Command{
	Use:   "palette",
	Short: "Extract or convert palettes",
	Long: `Extract palette from SCR files or convert between palette formats.

Examples:
  convimgcpc palette --extract screen.scr
  convimgcpc palette --convert input.pal output.kit`,
	RunE: runPalette,
}

// Available dithering methods
var availableDitherMethods = []string{
	"floyd-steinberg", "bayer1", "bayer2", "bayer3",
	"ordered1", "ordered2", "ordered3",
	"zigzag1", "zigzag2", "zigzag3", "none",
}

// Available output formats
var availableFormats = []string{"scr", "asm", "dsk", "png"}

// Available compression methods
var availableCompressionMethods = []string{"zx0", "zx0v2", "zx1", "lzw"}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Errors are surfaced once by main() as a single "Error: ..." line on
	// stderr, so silence cobra's own duplicate error/usage printing for a
	// clean one-line error output.
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Convert command flags
	convertCmd.Flags().StringVarP(&inputFile, "input", "i", "", "input image file (required)")
	convertCmd.Flags().StringVarP(&outputFile, "output", "o", "gfx.scr", "output file")
	convertCmd.Flags().IntVarP(&mode, "mode", "m", 1, "CPC mode: 0, 1, or 2")
	convertCmd.Flags().BoolVar(&overscan, "overscan", false, "enable overscan mode (96x272)")
	convertCmd.Flags().BoolVar(&plus, "plus", false, "CPC Plus mode (4096 colors)")
	convertCmd.Flags().StringVarP(&ditherMethod, "dither", "d", "floyd-steinberg",
		fmt.Sprintf("dithering method (%s)", strings.Join(availableDitherMethods, ", ")))
	convertCmd.Flags().IntVar(&ditherPct, "dither-pct", 50, "dithering percentage (0-100)")
	convertCmd.Flags().StringVarP(&format, "format", "f", "scr",
		fmt.Sprintf("output format (%s)", strings.Join(availableFormats, ", ")))
	convertCmd.Flags().StringVar(&paletteFile, "palette", "", "lock palette from file (.pal)")

	convertCmd.MarkFlagRequired("input")

	// Pack command flags
	packCmd.Flags().StringVarP(&inputFile, "input", "i", "", "input binary file (required)")
	packCmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file (required)")
	packCmd.Flags().StringVar(&compressionMethod, "method", "zx0",
		fmt.Sprintf("compression method (%s)", strings.Join(availableCompressionMethods, ", ")))

	packCmd.MarkFlagRequired("input")
	packCmd.MarkFlagRequired("output")

	// Unpack command flags
	unpackCmd.Flags().StringVarP(&inputFile, "input", "i", "", "input compressed file (required)")
	unpackCmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file (required)")
	unpackCmd.Flags().StringVar(&compressionMethod, "method", "", "decompression method (auto-detected if omitted: pks, lzw, ocp)")
	unpackCmd.MarkFlagRequired("input")
	unpackCmd.MarkFlagRequired("output")

	// Palette command flags
	paletteCmd.Flags().StringVar(&inputFile, "extract", "", "extract palette from SCR file")
	paletteCmd.Flags().StringVar(&outputFile, "convert", "", "convert between palette formats (input output)")

	// Add commands to root
	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(packCmd)
	rootCmd.AddCommand(unpackCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(paletteCmd)
}

// runConvert handles the convert command
func runConvert(cmd *cobra.Command, args []string) error {
	// Validate input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", inputFile)
	}

	// If the file is an .SCR input is a CPC screen file, not a bitmap image, route it
	// to the SCR -> PNG converter instead of image.Decode (which only handles
	// PNG/JPEG/GIF).
	if strings.ToLower(filepath.Ext(inputFile)) == ".scr" {
		return convertSCRToPNG(inputFile, outputFile)
	}

	// Validate mode
	if mode < 0 || mode > 2 {
		return fmt.Errorf("invalid mode: %d (must be 0, 1, or 2)", mode)
	}

	// Validate dither percentage
	if ditherPct < 0 || ditherPct > 100 {
		return fmt.Errorf("invalid dither percentage: %d (must be 0-100)", ditherPct)
	}

	// Validate dither method
	validDither := false
	for _, method := range availableDitherMethods {
		if method == ditherMethod {
			validDither = true
			break
		}
	}
	if !validDither {
		return fmt.Errorf("invalid dither method: %s (available: %s)",
			ditherMethod, strings.Join(availableDitherMethods, ", "))
	}

	// Validate format
	validFormat := false
	for _, fmt := range availableFormats {
		if fmt == format {
			validFormat = true
			break
		}
	}
	if !validFormat {
		return fmt.Errorf("invalid format: %s (available: %s)",
			format, strings.Join(availableFormats, ", "))
	}

	if verbose {
		fmt.Printf("Converting %s to %s\n", inputFile, outputFile)
		fmt.Printf("Mode: %d, Dither: %s (%d%%), Format: %s\n",
			mode, ditherMethod, ditherPct, format)
		if overscan {
			fmt.Println("Overscan mode enabled")
		}
		if plus {
			fmt.Println("CPC Plus mode enabled")
		}
	}

	// Load source image
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		// Not a bitmap: the file may still be a CPC SCR that lacks the .scr
		// extension. Try the SCR -> PNG route before giving up.
		if cvtErr := convertSCRToPNG(inputFile, outputFile); cvtErr == nil {
			return nil
		}
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Convert to DirectBitmap
	directBitmap := bitmap.NewFromImage(img)

	// Set up conversion parameters
	params := convert.NewDefaultSettings()
	params.VirtualMode = mode
	params.Method = mapDitherMethod(ditherMethod)
	params.Pct = ditherPct
	params.CpcPlus = plus

	if overscan {
		params.NumCols = 96
		params.NumLines = 272
	} else {
		params.NumCols = 80
		params.NumLines = 200
	}

	// Scale the source image to the CPC canvas size
	// so that small images fill the full screen instead of only 1/4 of it
	// (out-of-grid reads -> black). Nearest-neighbour stretch; pen-0 fills gaps.
	directBitmap = resizeToCanvas(directBitmap, params)

	// Load palette if specified
	if paletteFile != "" {
		if verbose {
			fmt.Printf("Loading palette from %s\n", paletteFile)
		}
		// TODO: Implement palette loading from fileio package
		// palette, err := fileio.LoadPalette(paletteFile)
		// if err != nil {
		//     return fmt.Errorf("failed to load palette: %w", err)
		// }
		// params.Palette = palette
	}

	// Create destination image
	bitmapCpc := render.NewBitmapCpcWithParams(params.NumCols, params.NumLines, params.CpcPlus)
	dest := &convert.ImageCpc{
		BitmapCpc: bitmapCpc,
		Width:     directBitmap.Width(),
		Height:    directBitmap.Height(),
		Mode:      params.VirtualMode,
	}

	// Perform conversion
	if verbose {
		fmt.Println("Starting conversion...")
	}

	startTime := getTimeMs()
	numColors := convert.Convert(directBitmap, dest, params, false)
	endTime := getTimeMs()

	// The converter computes the real reduced palette in
	// dest.BitmapCpc.Palette (FindBestColors). Sync params.Palette so the SCR
	// / PKS embed the actually-used colors (same pattern the GUI applies).
	copy(params.Palette[:], dest.BitmapCpc.Palette[:])

	if verbose {
		fmt.Printf("Conversion completed in %dms\n", endTime-startTime)
		fmt.Printf("Colors used: %d\n", numColors)
	}

	// Save output based on format
	switch format {
	case "scr":
		err = saveSCR(outputFile, dest, params)
	case "asm":
		err = saveASM(outputFile, dest, params)
	case "dsk":
		err = saveDSK(outputFile, dest, params)
	case "png":
		err = savePNG(outputFile, dest, params)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to save output: %w", err)
	}

	// Print summary
	fileInfo, _ := os.Stat(outputFile)
	var fileSize int64
	if fileInfo != nil {
		fileSize = fileInfo.Size()
	}

	fmt.Printf("Conversion successful!\n")
	fmt.Printf("  Input: %s (%dx%d)\n", inputFile, directBitmap.Width(), directBitmap.Height())
	fmt.Printf("  Output: %s (%d bytes)\n", outputFile, fileSize)
	fmt.Printf("  Mode: %d (%dx%d pixels)\n", mode, params.GetScreenWidth(), params.GetScreenHeight())
	fmt.Printf("  Colors used: %d\n", numColors)
	fmt.Printf("  Dithering: %s at %d%%\n", ditherMethod, ditherPct)

	return nil
}

// runPack handles the pack command
func runPack(cmd *cobra.Command, args []string) error {
	// Validate input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", inputFile)
	}

	// Validate compression method
	validMethod := false
	for _, method := range availableCompressionMethods {
		if method == compressionMethod {
			validMethod = true
			break
		}
	}
	if !validMethod {
		return fmt.Errorf("invalid compression method: %s (available: %s)",
			compressionMethod, strings.Join(availableCompressionMethods, ", "))
	}

	if verbose {
		fmt.Printf("Compressing %s to %s using %s\n", inputFile, outputFile, compressionMethod)
	}

	// Read input file
	inputData, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	inputSize := len(inputData)
	if verbose {
		fmt.Printf("Input size: %d bytes\n", inputSize)
	}

	// ---- Refuse to re-compress already-compressed data.
	// Packing an already-compressed file produces corrupt output, so fail with
	// a non-zero exit code instead (shell-script friendly).

	// Content-signature checks first: PKS ("PK…") and OCP ("MJH") have reliable
	// magics. PKS files also carry a valid AMSDOS header, so it is stripped
	// before inspecting the payload.
	payloadCheck := inputData
	if len(inputData) >= 128 && cpc.CheckAmsdos(inputData) {
		payloadCheck = inputData[128:]
	}
	if _, perr := compress.ParsePKSHeader(payloadCheck); perr == nil {
		return fmt.Errorf("pack: input file is already PKS-compressed: %s", inputFile)
	}
	if len(payloadCheck) >= 3 && payloadCheck[0] == 'M' && payloadCheck[1] == 'J' && payloadCheck[2] == 'H' {
		return fmt.Errorf("pack: input file is already OCP-compressed: %s", inputFile)
	}

	// LZW / ZX0 / ZX1 / PKS / CMP have no reliable magic signature, so the file
	// extension decides the format. Compressed-looking extensions are
	// rejected unconditionally — even when the file carries a valid AMSDOS
	// screen header (e.g. a .CMP repacked with a header).
	ext := strings.ToLower(filepath.Ext(inputFile))
	switch ext {
	case ".zx0", ".zx1", ".lzw", ".pks", ".cmp":
		return fmt.Errorf("pack: input file appears to be already compressed (extension %s): %s", ext, inputFile)
	}

	// Compress data
	compressor := compress.NewCompressor()
	outputBuffer := make([]byte, inputSize*2) // Allocate extra space

	var packMethod compress.PackMethod
	switch compressionMethod {
	case "zx0":
		packMethod = compress.MethodZX0
	case "zx0v2":
		packMethod = compress.MethodZX0V2
	case "zx1":
		packMethod = compress.MethodZX1
	case "lzw":
		packMethod = compress.Standard
	}

	outputSize, err := compressor.Pack(inputData, inputSize, outputBuffer, 0, packMethod)
	if err != nil {
		return fmt.Errorf("compression failed: %w", err)
	}

	// Write output file
	err = os.WriteFile(outputFile, outputBuffer[:outputSize], 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	// Print summary
	compressionRatio := float64(outputSize) / float64(inputSize) * 100

	fmt.Printf("Compression successful!\n")
	fmt.Printf("  Input: %s (%d bytes)\n", inputFile, inputSize)
	fmt.Printf("  Output: %s (%d bytes)\n", outputFile, outputSize)
	fmt.Printf("  Method: %s\n", compressionMethod)
	fmt.Printf("  Compression ratio: %.1f%%\n", compressionRatio)
	fmt.Printf("  Space saved: %d bytes (%.1f%%)\n",
		inputSize-outputSize, 100.0-compressionRatio)

	return nil
}

// runUnpack handles the unpack command
func runUnpack(cmd *cobra.Command, args []string) error {
	// Validate input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", inputFile)
	}

	inputData, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	inputSize := len(inputData)
	if inputSize == 0 {
		return fmt.Errorf("input file is empty")
	}

	compressor := compress.NewCompressor()
	outputBuffer := make([]byte, 0x20000) // 128KB max buffer for decompressed CPC data

	var method string
	if cmd.Flags().Changed("method") && compressionMethod != "" {
		method = strings.ToLower(compressionMethod)
	} else {
		// Auto-detection: strip a valid AMSDOS header before inspecting the
		// payload signatures.
		payload := inputData
		if len(inputData) >= 128 && cpc.CheckAmsdos(inputData) {
			payload = inputData[128:]
		}
		// PKS signature ("PK…") → auto-detected. Not yet supported: the
		// column-major decompression is not implemented in this command.
		// Reject now to avoid silently producing a corrupt output file.
		if len(payload) >= 4 && payload[0] == 'P' && payload[1] == 'K' {
			return fmt.Errorf("unpack: PKS files are not yet supported (use --method pks, once implemented): %s", inputFile)
		}
		if len(payload) >= 4 && payload[0] == 'M' && payload[1] == 'J' && payload[2] == 'H' {
			method = "ocp"
		} else if looksLikeRawSCR(inputData) {
			// A raw (uncompressed) SCR submitted to the auto-detect path is
			// rejected up-front: unpacking it would produce garbage.
			return fmt.Errorf("unpack: input file appears to be uncompressed SCR (raw): %s", inputFile)
		} else {
			method = "lzw"
		}
	}

	if verbose {
		fmt.Printf("Decompressing %s to %s using %s\n", inputFile, outputFile, method)
	}

	var outputSize int
	switch method {
	case "lzw":
		outputSize, err = compressor.Depack(inputData, 0, outputBuffer, compress.Standard)
		if err != nil {
			return fmt.Errorf("LZW decompression failed: %w", err)
		}
		// An explicit LZW unpack of a raw (uncompressed) input "succeeds" with
		// the input echoed back unchanged — a trivial round-trip that would
		// silently produce garbage. Reject it.
		if bytes.Equal(outputBuffer[:outputSize], inputData) {
			return fmt.Errorf("unpack: input file does not appear to be compressed (trivial round-trip): %s", inputFile)
		}
		// Guard against feeding an unsupported/non-LZW file into the LZW
		// decompressor and silently emitting corrupt data: the decompressed
		// output of CPC-compressed data is always a screen-sized payload
		// (standard 16336 / overscan 31936) or the PKSL pixel buffer
		// (16000). An output that matches none of those sizes is treated as
		// corrupt.
		if !isKnownScreenSize(outputSize) {
			return fmt.Errorf("unpack: LZW output (%d bytes) is not a recognized CPC screen size; input is not a supported compressed file: %s", outputSize, inputFile)
		}

	case "ocp":
		payload := inputData
		if len(inputData) >= 128 && cpc.CheckAmsdos(inputData) {
			payload = inputData[128:]
		}
		outputSize, err = compressor.Depack(payload, 0, outputBuffer, compress.MethodOCP)
		if err != nil {
			return fmt.Errorf("OCP decompression failed: %w", err)
		}

	default:
		return fmt.Errorf("unsupported decompression method: %s", method)
	}

	err = os.WriteFile(outputFile, outputBuffer[:outputSize], 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("Decompression successful!\n")
	fmt.Printf("  Input: %s (%d bytes)\n", inputFile, inputSize)
	fmt.Printf("  Output: %s (%d bytes)\n", outputFile, outputSize)
	fmt.Printf("  Method: %s\n", method)

	return nil
}

// isKnownScreenSize reports whether n is a plausible decompressed CPC screen
// payload size (standard or overscan bitmap).
func isKnownScreenSize(n int) bool {
	screenSizes := map[int]bool{
		cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines): true,
		cpc.BitmapSize(cpc.OverscanCols, cpc.OverscanLines): true,
		cpc.StandardCols * cpc.StandardLines:                true, // PKSL pixel payload
	}
	return screenSizes[n]
}

// looksLikeRawSCR reports whether the input looks like an uncompressed CPC
// screen dump: either a valid AMSDOS header with a screen load address, or a
// payload whose size matches the standard/overscan bitmap sizes.
func looksLikeRawSCR(data []byte) bool {
	payload := data
	if len(data) >= 128 && cpc.CheckAmsdos(data) {
		if ent, e := cpc.GetAmsdos(data); e == nil {
			if ent.Address == 0xC000 || ent.Address == 0x0200 {
				return true
			}
		}
		payload = data[128:]
	}
	// Size heuristic with a small tolerance for slightly different writers.
	for _, target := range []int{16336, 31936} {
		d := len(payload) - target
		if d < 0 {
			d = -d
		}
		if d <= 256 {
			return true
		}
	}
	return false
}

// runInfo handles the info command
func runInfo(cmd *cobra.Command, args []string) error {
	filename := args[0]

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filename)
	}

	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".scr":
		return showSCRInfo(filename)
	case ".dsk":
		return showDSKInfo(filename)
	case ".pal":
		return showPaletteInfo(filename)
	default:
		return fmt.Errorf("unsupported file type: %s (supported: .scr, .dsk, .pal)", ext)
	}
}

// runPalette handles the palette command
func runPalette(cmd *cobra.Command, args []string) error {
	if inputFile != "" {
		// Extract palette from SCR file
		return extractPalette(inputFile)
	} else if outputFile != "" && len(args) >= 2 {
		// Convert palette format
		return convertPalette(args[0], args[1])
	} else {
		return fmt.Errorf("specify either --extract <scrfile> or --convert <input> <output>")
	}
}

// Helper functions

// mapDitherMethod converts CLI dither method names to internal method names
func mapDitherMethod(method string) string {
	switch method {
	case "floyd-steinberg":
		return "Floyd-Steinberg (2x2)"
	case "bayer1":
		return "Bayer 1 (2X2)"
	case "bayer2":
		return "Bayer 2 (4x4)"
	case "bayer3":
		return "Bayer 3 (4X4)"
	case "ordered1":
		return "Ordered 1 (2x2)"
	case "ordered2":
		return "Ordered 2 (3x3)"
	case "ordered3":
		return "Ordered 3 (4x4)"
	case "zigzag1":
		return "ZigZag1 (3x3)"
	case "zigzag2":
		return "ZigZag2 (4x3)"
	case "zigzag3":
		return "ZigZag3 (5x4)"
	case "none":
		return "None"
	default:
		return "Floyd-Steinberg (2x2)"
	}
}

// resizeToCanvas scales the source bitmap to the CPC display canvas so that
// a small source image fills the full screen (640x400 standard, 768x544
// overscan) instead of being rendered at 1:1 and reading out-of-grid pixels.
// Uses nearest-neighbour stretch; the background is filled with pen 0 color
// (mirrors the GUI getResizeBitmap). Known limitation: native-resolution
// fidelity is only approximate; TODO: non-integer resize and overscan
func resizeToCanvas(source *bitmap.DirectBitmap, prm *convert.Settings) *bitmap.DirectBitmap {
	cpcW := prm.GetScreenWidth()
	cpcH := prm.GetScreenHeight()

	// Background = pen 0 color.
	bg := cpc.GetColor(prm.Palette[0], prm.CpcPlus)
	resized := bitmap.NewDirectBitmap(cpcW, cpcH)
	for y := 0; y < cpcH; y++ {
		for x := 0; x < cpcW; x++ {
			resized.SetPixelColor(x, y, bg)
		}
	}

	srcW := source.Width()
	srcH := source.Height()
	if srcW == 0 || srcH == 0 {
		return resized
	}

	for dy := 0; dy < cpcH; dy++ {
		srcY := dy * srcH / cpcH
		if srcY >= srcH {
			srcY = srcH - 1
		}
		for dx := 0; dx < cpcW; dx++ {
			srcX := dx * srcW / cpcW
			if srcX >= srcW {
				srcX = srcW - 1
			}
			resized.SetPixelColor(dx, dy, source.GetPixelColor(srcX, srcY))
		}
	}
	return resized
}

// applySCRPalette parses the embedded ModePal block from the screen data and
// populates bmp's palette. Standard SCR mode: 17-byte block at ModePalOffset
// (mode byte + 16 ink values). Overscan: 33-byte block at 0x600. Classic mode
// uses 27-color ink indices (0..26, 0xFF = unused → pen 0). CPC Plus uses
// 12-bit 0x0VBR pairs (low = B|R<<4, high = V).
// Falls back silently to the default palette if the block is absent or
// implausible.
func applySCRPalette(bmp *render.BitmapCpc, screenData []byte, modeOffset int) {
	if modeOffset+17 > len(screenData) {
		return
	}
	modeByte := screenData[modeOffset]

	if (modeByte & 0x80) != 0 {
		// CPC Plus: 33-byte block; each pen is a low/high byte pair encoding
		// 0x0VBR (as written by fileio.SaveSCR: low = B | R<<4, high = V).
		if modeOffset+33 > len(screenData) {
			return
		}
		bmp.CpcPlus = true
		for i := 0; i < 16; i++ {
			lo := screenData[modeOffset+1+i*2]
			hi := screenData[modeOffset+2+i*2]
			if lo == 0xFF && hi == 0xFF {
				bmp.Palette[i] = 0 // unused pen → black
				continue
			}
			bmp.Palette[i] = int(hi&0x0F)<<8 | int(lo&0x0F)<<4 | int((lo>>4)&0x0F)
		}
		bmp.VirtualMode = int(modeByte) & 0x03
		return
	}

	// Classic: 17-byte block; inks 0..26 or 0xFF (unused pen). Only accept a
	// plausible ModePal (mode 0..4, all inks valid) to avoid misreading pixel
	// data as a palette.
	if int(modeByte) > 4 {
		return
	}
	for i := 0; i < 16; i++ {
		b := screenData[modeOffset+1+i]
		if b != 0xFF && b > 26 {
			return
		}
	}
	bmp.VirtualMode = int(modeByte)
	for i := 0; i < 16; i++ {
		ink := int(screenData[modeOffset+1+i])
		if ink == 0xFF {
			ink = 0 // unused pen → black
		}
		bmp.Palette[i] = ink
	}
}

// rgbaToIndexed converts a rendered RGBA frame into an indexed-color image
// using 16 palette entries ordered exactly as the SCR pens.
// Rendered pixels come from cpc.PaletteColor, so they match the palette
// entries exactly; a nearest-color fallback guards rounding edge cases.
func rgbaToIndexed(img *image.RGBA, bmp *render.BitmapCpc) *image.Paletted {
	// 16 pens in the exact order of the SCR palette.
	palette16 := make(color.Palette, 16)
	lookup := make(map[int]uint8, 16)
	palRGB := make([]int, 16)
	for i := 0; i < 16; i++ {
		rgb := cpc.PaletteColor(bmp.Palette[i], bmp.CpcPlus)
		palette16[i] = color.RGBA{R: uint8(rgb >> 16), G: uint8(rgb >> 8), B: uint8(rgb), A: 255}
		palRGB[i] = rgb
		if _, ok := lookup[rgb]; !ok {
			lookup[rgb] = uint8(i)
		}
	}

	bounds := img.Bounds()
	out := image.NewPaletted(bounds, palette16)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
			rgb := int(c.B) | int(c.G)<<8 | int(c.R)<<16
			idx, ok := lookup[rgb]
			if !ok {
				idx = nearestPalIndex(palRGB, rgb)
			}
			out.SetColorIndex(x, y, idx)
		}
	}
	return out
}

// nearestPalIndex returns the palette entry closest (Euclidean) to rgb.
func nearestPalIndex(palRGB []int, rgb int) uint8 {
	best := 0
	bestDist := int64(1) << 62
	for i, p := range palRGB {
		dr := int64((p>>16)&0xFF) - int64((rgb>>16)&0xFF)
		dg := int64((p>>8)&0xFF) - int64((rgb>>8)&0xFF)
		db := int64(p&0xFF) - int64(rgb&0xFF)
		d := dr*dr + dg*dg + db*db
		if d < bestDist {
			bestDist = d
			best = i
		}
	}
	return uint8(best)
}

// convertSCRToPNG converts an Amstrad CPC SCR screen into a PNG rendering.
// This is the reverse direction of the normal convert command: input is .SCR,
// output is .PNG.
func convertSCRToPNG(inputFile, outputFile string) error {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	payload := data
	if len(data) >= 128 && cpc.CheckAmsdos(data) {
		payload = data[128:]
	}

	// PKS-compressed screens carry a "PK" signature instead of raw screen
	// data. Rendering the compressed bytes as if they were a screen would
	// silently produce a corrupt PNG, so reject every variant explicitly.
	if len(payload) >= 4 && payload[0] == 'P' && payload[1] == 'K' {
		return fmt.Errorf("input file is PKS-compressed, which convert does not support: %s", inputFile)
	}

	// LoadSCR needs at least 16384 bytes; a standard SCR payload is 16336.
	if len(payload) < 16384 {
		pad := make([]byte, 16384)
		copy(pad, payload)
		payload = pad
	}

	screenData, _, lerr := fileio.LoadSCR(payload)
	if lerr != nil {
		return fmt.Errorf("failed to load SCR data: %w", lerr)
	}

	bmp := render.NewBitmapCpcWithParams(cpc.StandardCols, cpc.StandardLines, false)
	copy(bmp.ScreenData[:], screenData)

	// ModePal offset: standard SCR layout places the mode + palette at
	// ModePalOffset; overscan uses a different location (0x600). The overscan
	// signal is derived from the AMSDOS load address or the file size.
	modeOffset := cpc.ModePalOffset
	addr := uint16(0)
	if len(data) >= 128 && cpc.CheckAmsdos(data) {
		if ent, e := cpc.GetAmsdos(data); e == nil {
			addr = ent.Address
		}
	}
	if (addr == 0x0200) || (addr == 0 && len(data) >= 0x4000) {
		modeOffset = 0x600
	}

	// Use the palette embedded in the SCR (ModePal block) instead of the
	// fixed default palette returned by LoadSCR.
	applySCRPalette(bmp, screenData, modeOffset)

	img := bmp.RenderToRGBA()
	if img == nil {
		return fmt.Errorf("failed to render SCR screen data")
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()
	// Indexed-color PNG whose palette follows the SCR pen order.
	if err := png.Encode(file, rgbaToIndexed(img, bmp)); err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}

	fileInfo, _ := os.Stat(outputFile)
	fmt.Printf("Conversion successful!\n")
	fmt.Printf("  Input: %s (%d bytes)\n", inputFile, len(data))
	if fileInfo != nil {
		fmt.Printf("  Output: %s (%d bytes)\n", outputFile, fileInfo.Size())
	} else {
		fmt.Printf("  Output: %s\n", outputFile)
	}
	fmt.Printf("  Mode: %d (%dx%d pixels)\n", bmp.VirtualMode, bmp.NumCol*8, bmp.NumLig*2)

	return nil
}

// saveSCR saves the converted image as an SCR file
func saveSCR(filename string, dest *convert.ImageCpc, params *convert.Settings) error {
	if verbose {
		fmt.Printf("Saving SCR file: %s\n", filename)
	}

	// Create SCR parameters
	scrParams := fileio.SCRParams{
		WithPalette: true,
		WithCode:    true,
		CPCPlus:     params.CpcPlus,
		VirtualMode: params.VirtualMode,
	}

	// Get bitmap data — slice to the real CPC bitmap size (Bug 3). ScreenData
	// is a [0x10000]byte buffer; writing the whole 64KB would produce a file
	// that is 128+65536 bytes instead of 128+BitmapSize(80|96, 200|272).
	bitmapSize := cpc.BitmapSize(params.NumCols, params.NumLines)
	bitmapData := dest.BitmapCpc.ScreenData[:bitmapSize]

	// Extract palette (simplified - should use real palette from conversion)
	palette := make([]uint16, 16)
	for i := 0; i < 16; i++ {
		if params.CpcPlus {
			palette[i] = uint16(params.Palette[i])
		} else {
			palette[i] = uint16(params.Palette[i])
		}
	}

	_, err := fileio.SaveSCR(filename, bitmapData, bitmapSize,
		fileio.PackNone, fileio.OutputBinary, scrParams, palette, nil)
	return err
}

// saveASM saves the converted image as assembly source
func saveASM(filename string, dest *convert.ImageCpc, params *convert.Settings) error {
	if verbose {
		fmt.Printf("Saving ASM file: %s\n", filename)
	}
	// TODO: Implement ASM generation using asmgen package
	return fmt.Errorf("ASM output not implemented yet")
}

// saveDSK saves the converted image to a DSK file
func saveDSK(filename string, dest *convert.ImageCpc, params *convert.Settings) error {
	if verbose {
		fmt.Printf("Saving DSK file: %s\n", filename)
	}
	// TODO: Implement DSK generation using fileio package
	return fmt.Errorf("DSK output not implemented yet")
}

// savePNG saves a PNG preview of the CPC image
func savePNG(filename string, dest *convert.ImageCpc, params *convert.Settings) error {
	if verbose {
		fmt.Printf("Saving PNG file: %s\n", filename)
	}
	// TODO: Implement PNG rendering using render package
	return fmt.Errorf("PNG output not implemented yet")
}

// showSCRInfo displays information about an SCR file
func showSCRInfo(filename string) error {
	fmt.Printf("SCR File Information: %s\n", filename)

	// TODO: Implement SCR info extraction using fileio package
	// This would read the AMSDOS header and display:
	// - File size, load address, execution address
	// - CPC mode, palette information
	// - Whether it's compressed, overscan mode, etc.

	fileInfo, err := os.Stat(filename)
	if err != nil {
		return err
	}

	fmt.Printf("  File size: %d bytes\n", fileInfo.Size())
	fmt.Printf("  [Additional SCR metadata would be shown here]\n")

	return fmt.Errorf("SCR info extraction not implemented yet")
}

// showDSKInfo displays information about a DSK file
func showDSKInfo(filename string) error {
	fmt.Printf("DSK File Information: %s\n", filename)

	// TODO: Implement DSK info extraction using fileio package
	fmt.Printf("  [DSK metadata would be shown here]\n")

	return fmt.Errorf("DSK info extraction not implemented yet")
}

// showPaletteInfo displays information about a palette file
func showPaletteInfo(filename string) error {
	fmt.Printf("Palette File Information: %s\n", filename)

	// TODO: Implement palette info extraction using fileio package
	fmt.Printf("  [Palette metadata would be shown here]\n")

	return fmt.Errorf("Palette info extraction not implemented yet")
}

// extractPalette extracts palette from an SCR file
func extractPalette(filename string) error {
	fmt.Printf("Extracting palette from: %s\n", filename)

	// TODO: Implement palette extraction using fileio package
	return fmt.Errorf("Palette extraction not implemented yet")
}

// convertPalette converts between palette formats
func convertPalette(inputFile, outputFile string) error {
	fmt.Printf("Converting palette from %s to %s\n", inputFile, outputFile)

	// TODO: Implement palette conversion using fileio package
	return fmt.Errorf("Palette conversion not implemented yet")
}

// getTimeMs returns current time in milliseconds (simplified)
func getTimeMs() int64 {
	// This is a placeholder - in real implementation would use time.Now().UnixMilli()
	return 0
}
