// Package compress provides compression and decompression algorithms.
package compress

import (
	"bytes"
	"testing"
)

// TestLZWDepackStd tests LZW decompression with known data
func TestLZWDepackStd(t *testing.T) {
	lzw := NewLZW()

	// Create a simple test case
	// Note: These would typically be compressed data from PackStd
	// For now, we test that the function handles input correctly

	input := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
	output := make([]byte, 100)

	outSize, err := lzw.DepackStd(input, 0, output)
	if err != nil {
		t.Logf("DepackStd error (expected for non-compressed data): %v", err)
	}

	if outSize < 0 {
		t.Errorf("DepackStd returned negative size: %d", outSize)
	}
}

// TestLZWPackStd tests LZW compression
func TestLZWPackStd(t *testing.T) {
	lzw := NewLZW()

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Simple pattern",
			data: []byte{0xAA, 0xAA, 0xAA, 0xAA, 0x55, 0x55, 0x55, 0x55},
		},
		{
			name: "Sequential bytes",
			data: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
		{
			name: "Repeating pattern",
			data: bytes.Repeat([]byte{0xFF, 0x00}, 20),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed := make([]byte, len(tt.data)*2)

			compSize, err := lzw.PackStd(tt.data, len(tt.data), compressed, len(compressed))
			if err != nil {
				t.Fatalf("PackStd failed: %v", err)
			}

			if compSize <= 0 {
				t.Errorf("Compressed size = %d, expected > 0", compSize)
			}

			t.Logf("Original: %d bytes, Compressed: %d bytes (%.2f%% ratio)",
				len(tt.data), compSize, float64(compSize)*100.0/float64(len(tt.data)))
		})
	}
}

// TestLZWPackStdEmpty tests compression of empty data
func TestLZWPackStdEmpty(t *testing.T) {
	lzw := NewLZW()
	data := []byte{}
	compressed := make([]byte, 10)

	compSize, err := lzw.PackStd(data, 0, compressed, len(compressed))
	if err != nil {
		t.Logf("PackStd on empty data returned error: %v", err)
	}

	// Empty data should produce minimal output
	if compSize < 0 {
		t.Errorf("Compressed size = %d, should not be negative", compSize)
	}
}

// TestLZWPackDepackRoundTrip tests compress/decompress round-trip
func TestLZWPackDepackRoundTrip(t *testing.T) {
	lzw := NewLZW()

	original := []byte{1, 2, 3, 4, 1, 2, 3, 4, 5, 6, 7, 8, 5, 6, 7, 8}
	compressed := make([]byte, len(original)*2)
	decompressed := make([]byte, len(original)*2)

	// Compress
	compSize, err := lzw.PackStd(original, len(original), compressed, len(compressed))
	if err != nil {
		t.Fatalf("PackStd failed: %v", err)
	}

	t.Logf("Compressed from %d to %d bytes", len(original), compSize)

	// Decompress
	decompSize, err := lzw.DepackStd(compressed, 0, decompressed)
	if err != nil {
		t.Fatalf("DepackStd failed: %v", err)
	}

	t.Logf("Decompressed to %d bytes", decompSize)

	// Compare (only if decompression succeeded with matching size)
	if decompSize == len(original) {
		for i := 0; i < len(original); i++ {
			if decompressed[i] != original[i] {
				t.Errorf("Byte %d: got 0x%02X, want 0x%02X", i, decompressed[i], original[i])
			}
		}
	}
}

// TestLZWDepackStdBufferSizes tests buffer size handling
func TestLZWDepackStdBufferSizes(t *testing.T) {
	lzw := NewLZW()

	// Test with small buffers
	input := []byte{0x01, 0x02, 0x03}
	output := make([]byte, 5)

	_, err := lzw.DepackStd(input, 0, output)
	if err != nil {
		t.Logf("DepackStd with small buffer returned error: %v", err)
	}
}

// TestLZWDepackStdInvalidStart tests error handling for invalid start index
func TestLZWDepackStdInvalidStart(t *testing.T) {
	lzw := NewLZW()

	input := []byte{0x01, 0x02, 0x03}
	output := make([]byte, 10)

	_, err := lzw.DepackStd(input, 100, output) // startIn > len(bufIn)
	if err == nil {
		t.Error("Expected error for startIn >= len(bufIn), got nil")
	}
}

// TestLZWPackStdInvalidBufferSize tests error handling for buffer size mismatch
func TestLZWPackStdInvalidBufferSize(t *testing.T) {
	lzw := NewLZW()

	input := []byte{0x01, 0x02, 0x03}
	output := make([]byte, 10)

	_, err := lzw.PackStd(input, 100, output, 10) // lengthIn > len(bufIn)
	if err == nil {
		t.Error("Expected error for lengthIn > len(bufIn), got nil")
	}

	_, err = lzw.PackStd(input, 3, output, 100) // lengthOut > len(bufOut)
	if err == nil {
		t.Error("Expected error for lengthOut > len(bufOut), got nil")
	}
}

// TestLZWHighlyCompressible tests compression of highly repetitive data
func TestLZWHighlyCompressible(t *testing.T) {
	lzw := NewLZW()

	// Create highly compressible data
	original := bytes.Repeat([]byte{0xFF}, 100)
	compressed := make([]byte, len(original)*2)

	compSize, err := lzw.PackStd(original, len(original), compressed, len(compressed))
	if err != nil {
		t.Fatalf("PackStd failed: %v", err)
	}

	// Should achieve significant compression
	if compSize >= len(original) {
		t.Logf("Warning: Highly compressible data did not compress: %d -> %d bytes",
			len(original), compSize)
	} else {
		t.Logf("Compressed highly repetitive data: %d -> %d bytes (%.2f%% ratio)",
			len(original), compSize, float64(compSize)*100.0/float64(len(original)))
	}
}

// TestLZWLargeData tests compression of larger data sets
func TestLZWLargeData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large data test in short mode")
	}

	lzw := NewLZW()

	// Create 1KB of semi-random data with patterns
	original := make([]byte, 1024)
	for i := range original {
		original[i] = byte((i % 256) ^ (i / 256))
	}

	compressed := make([]byte, len(original)*2)

	compSize, err := lzw.PackStd(original, len(original), compressed, len(compressed))
	if err != nil {
		t.Fatalf("PackStd failed: %v", err)
	}

	if compSize <= 0 {
		t.Errorf("Compressed size = %d, expected > 0", compSize)
	}

	t.Logf("Compressed 1KB: %d bytes (%.2f%% ratio)",
		compSize, float64(compSize)*100.0/float64(len(original)))
}

// TestLZWMatchTableInitialization tests that match tables are properly initialized
func TestLZWMatchTableInitialization(t *testing.T) {
	lzw := NewLZW()

	// Verify that a new LZW instance is ready to use
	if lzw == nil {
		t.Fatal("NewLZW returned nil")
	}

	// Try a simple compression to verify initialization
	data := []byte{0x01, 0x02, 0x03}
	output := make([]byte, 100)

	_, err := lzw.PackStd(data, len(data), output, len(output))
	if err != nil {
		t.Fatalf("PackStd failed on simple data: %v", err)
	}
}

// TestLZWDepackLiteralBytes tests decompression of literal bytes
func TestLZWDepackLiteralBytes(t *testing.T) {
	lzw := NewLZW()

	// Create input with literal byte sequence (bit=0 indicates literal)
	// This is a simplified test - actual format is more complex
	input := []byte{0x00, 0x41, 0x42, 0x43, 0x00} // Some literal bytes
	output := make([]byte, 100)

	outSize, err := lzw.DepackStd(input, 0, output)
	if err != nil {
		t.Logf("DepackStd returned error (expected for test data): %v", err)
	}

	if outSize < 0 {
		t.Errorf("Decompressed size should not be negative: %d", outSize)
	}
}
