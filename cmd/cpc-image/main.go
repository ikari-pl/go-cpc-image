// Package main provides the command-line interface for ConvImgCpc image conversion utility.
// This CLI application allows users to convert images to Amstrad CPC formats, compress files,
// extract information from CPC files, and manage palettes.
package main

import (
	"fmt"
	"image"
	_ "image/gif"  // Register GIF decoder
	_ "image/jpeg" // Register JPEG decoder
	_ "image/png"  // Register PNG decoder
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
	"github.com/ikari-pl/go-cpc-image/pkg/compress"
	"github.com/ikari-pl/go-cpc-image/pkg/convert"
	"github.com/ikari-pl/go-cpc-image/pkg/fileio"
	"github.com/ikari-pl/go-cpc-image/pkg/render"
)

var (
	// Global flags
	verbose bool

	// Common conversion flags
	inputFile      string
	outputFile     string
	mode           int
	overscan       bool
	plus           bool
	ditherMethod   string
	ditherPct      int
	format         string
	paletteFile    string

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

func init() {
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

	// Palette command flags
	paletteCmd.Flags().StringVar(&inputFile, "extract", "", "extract palette from SCR file")
	paletteCmd.Flags().StringVar(&outputFile, "convert", "", "convert between palette formats (input output)")

	// Add commands to root
	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(packCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(paletteCmd)
}

// runConvert handles the convert command
func runConvert(cmd *cobra.Command, args []string) error {
	// Validate input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", inputFile)
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

	// Get bitmap data
	bitmapData := dest.BitmapCpc.ScreenData[:]
	bitmapSize := len(bitmapData)

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

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}