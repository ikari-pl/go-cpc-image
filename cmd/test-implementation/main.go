// Package main provides test functionality for the CPC conversion implementations.
package main

import (
	"fmt"
	"os"

	"github.com/ikari-pl/go-cpc-image/pkg/compress"
	"github.com/ikari-pl/go-cpc-image/pkg/fileio"
)

func main() {
	fmt.Println("ConvImgCPC Go Implementation Test")
	fmt.Println("================================")

	// Test DSK functionality
	fmt.Println("\n1. Testing DSK functionality...")
	testDSK()

	// Test compression functionality
	fmt.Println("\n2. Testing compression functionality...")
	testCompression()
}

func testDSK() {
	// Create a new DSK
	dsk := fileio.NewGestDSK()
	fmt.Printf("   ✓ Created new DSK with %d tracks\n", dsk.Infos.NbTracks)

	// Test adding a file
	testData := []byte("Hello, World! This is test data for the DSK implementation.")
	err := dsk.CopieFichier(testData, "TEST.BIN", len(testData), 180, 0)
	if err == fileio.ErrNoErr {
		fmt.Printf("   ✓ Successfully added test file (%d bytes)\n", len(testData))
	} else {
		fmt.Printf("   ✗ Failed to add test file: %v\n", err)
	}

	// Test saving DSK
	testDskPath := "/tmp/test.dsk"
	err2 := dsk.Save(testDskPath)
	if err2 == nil {
		fmt.Printf("   ✓ Successfully saved DSK to %s\n", testDskPath)

		// Clean up
		os.Remove(testDskPath)
		fmt.Printf("   ✓ Cleaned up test file\n")
	} else {
		fmt.Printf("   ✗ Failed to save DSK: %v\n", err2)
	}
}

func testCompression() {
	compressor := compress.NewCompressor()

	// Test data
	testData := []byte("AAABBBCCCDDDEEEFFFGGGHHHIIIJJJKKKLLLMMMNNNOOOPPPQQQRRRSSSTTTUUUVVVWWWXXXYYYZZZ")
	inputSize := len(testData)
	outputBuffer := make([]byte, inputSize*2) // Ensure enough space

	fmt.Printf("   Original data size: %d bytes\n", inputSize)

	// Test Standard LZW compression
	compressedSize, err := compressor.Pack(testData, inputSize, outputBuffer, 0, compress.Standard)
	if err == nil {
		fmt.Printf("   ✓ Standard LZW compression: %d bytes (%.1f%% ratio)\n",
			compressedSize, float64(compressedSize)/float64(inputSize)*100)

		// Test decompression
		decompressedBuffer := make([]byte, inputSize*2)
		decompressedSize, err := compressor.Depack(outputBuffer, 0, decompressedBuffer, compress.Standard)
		if err == nil && decompressedSize > 0 {
			fmt.Printf("   ✓ Standard LZW decompression: %d bytes\n", decompressedSize)
		} else {
			fmt.Printf("   ✗ Standard LZW decompression failed: %v\n", err)
		}
	} else {
		fmt.Printf("   ✗ Standard LZW compression failed: %v\n", err)
	}

	// Test ZX0 compression
	zx0CompressedSize, err := compressor.Pack(testData, inputSize, outputBuffer, 0, compress.MethodZX0)
	if err == nil {
		fmt.Printf("   ✓ ZX0 compression: %d bytes (%.1f%% ratio)\n",
			zx0CompressedSize, float64(zx0CompressedSize)/float64(inputSize)*100)
	} else {
		fmt.Printf("   ✗ ZX0 compression failed: %v\n", err)
	}

	// Test ZX0 v2 compression
	zx0v2CompressedSize, err := compressor.Pack(testData, inputSize, outputBuffer, 0, compress.MethodZX0V2)
	if err == nil {
		fmt.Printf("   ✓ ZX0 v2 compression: %d bytes (%.1f%% ratio)\n",
			zx0v2CompressedSize, float64(zx0v2CompressedSize)/float64(inputSize)*100)
	} else {
		fmt.Printf("   ✗ ZX0 v2 compression failed: %v\n", err)
	}

	// Test ZX1 compression
	zx1CompressedSize, err := compressor.Pack(testData, inputSize, outputBuffer, 0, compress.MethodZX1)
	if err == nil {
		fmt.Printf("   ✓ ZX1 compression: %d bytes (%.1f%% ratio)\n",
			zx1CompressedSize, float64(zx1CompressedSize)/float64(inputSize)*100)
	} else {
		fmt.Printf("   ✗ ZX1 compression failed: %v\n", err)
	}

	fmt.Println("\n   All compression algorithms tested!")
}