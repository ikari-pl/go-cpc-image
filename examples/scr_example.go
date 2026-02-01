// Example demonstrating how to use the fileio and asmgen packages
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ikari/go-cpc-image/pkg/asmgen"
	"github.com/ikari/go-cpc-image/pkg/fileio"
)

func main() {
	// Example 1: Creating an SCR file with embedded Z80 code
	fmt.Println("=== SCR File Creation Example ===")

	// Create a simple bitmap (just a test pattern)
	bitmap := make([]byte, 0x4000) // 16KB standard screen
	for i := range bitmap {
		bitmap[i] = byte(i & 0xFF) // Simple pattern
	}

	// Set up parameters
	params := fileio.SCRParams{
		WithPalette: true,
		WithCode:    true,
		CPCPlus:     false,
		VirtualMode: 0,
	}

	// Create a test palette
	palette := make([]uint16, 16)
	for i := range palette {
		palette[i] = uint16(i) // Simple palette
	}

	// Save as SCR file
	filename := "/tmp/test.scr"
	size, err := fileio.SaveSCR(filename, bitmap, len(bitmap), fileio.PackNone, fileio.OutputBinary, params, palette, nil)
	if err != nil {
		fmt.Printf("Error creating SCR file: %v\n", err)
	} else {
		fmt.Printf("Created SCR file: %s (%d bytes)\n", filename, size)
	}

	// Example 2: Assembly generation
	fmt.Println("\n=== Assembly Generation Example ===")

	var asmOutput strings.Builder
	aw := asmgen.NewASMWriter(&asmOutput)

	// Generate assembly header
	aw.WriteHeader("v1.0 Go", "Mode 0 - 160x200, 16 colors", 160, 200)
	aw.WriteBlankLine()

	// Generate some Z80 code
	aw.WriteOrg(0x8000)
	aw.WriteLabel("Start")
	aw.WriteInstruction("DI", "", "Disable interrupts")
	aw.WriteInstruction("LD", "HL,#C000", "Screen address")
	aw.WriteInstruction("LD", "BC,#4000", "Screen size")
	aw.WriteInstruction("LD", "A,#FF", "Fill value")
	aw.WriteBlankLine()

	aw.WriteLabel("FillLoop")
	aw.WriteInstruction("LD", "(HL),A", "Fill screen")
	aw.WriteInstruction("INC", "HL", "")
	aw.WriteInstruction("DEC", "BC", "")
	aw.WriteInstruction("LD", "A,B", "")
	aw.WriteInstruction("OR", "C", "")
	aw.WriteInstruction("JR", "NZ,FillLoop", "Continue if not done")
	aw.WriteBlankLine()

	// Generate data
	testData := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	aw.WriteDB(testData, 4, "TestData", true)
	aw.WriteBlankLine()

	// Generate palette
	paletteParams := asmgen.PaletteParams{
		CPCPlus:     false,
		VirtualMode: 0,
	}
	aw.GeneratePalette(palette, paletteParams, false, true, "Palette")
	aw.WriteBlankLine()

	// Generate decompression routine
	aw.WriteComment("ZX0 Decompression routine")
	aw.GenerateZX0Depack("DisplayImage")

	fmt.Printf("Generated Assembly:\n%s\n", asmOutput.String())

	// Example 3: Palette file operations
	fmt.Println("\n=== Palette File Example ===")

	// Save palette
	palFilename := "/tmp/test.pal"
	err = fileio.SavePalette(palFilename, palette, params)
	if err != nil {
		fmt.Printf("Error saving palette: %v\n", err)
	} else {
		fmt.Printf("Saved palette file: %s\n", palFilename)
	}

	// Load palette back
	loadedPalette := make([]uint16, 16)
	err = fileio.LoadPalette(palFilename, loadedPalette, &params)
	if err != nil {
		fmt.Printf("Error loading palette: %v\n", err)
	} else {
		fmt.Printf("Loaded palette: %v\n", loadedPalette[:4]) // Show first 4 colors
	}

	// Clean up temp files
	os.Remove(filename)
	os.Remove(palFilename)

	fmt.Println("\n=== Example Complete ===")
}