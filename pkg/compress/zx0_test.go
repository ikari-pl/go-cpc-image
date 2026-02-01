// Package compress provides ZX0 compression algorithm implementation.
package compress

import (
	"bytes"
	"testing"
)

// TestZX0CompressDecompress tests compress/decompress round-trip with known data
func TestZX0CompressDecompress(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Simple repeating pattern",
			data: []byte{0xAA, 0xAA, 0xAA, 0xAA, 0x55, 0x55, 0x55, 0x55},
		},
		{
			name: "Sequential data",
			data: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		},
		{
			name: "Mixed pattern",
			data: []byte{1, 2, 3, 1, 2, 3, 1, 2, 3, 4, 5, 6, 4, 5, 6, 4, 5, 6},
		},
		{
			name: "Single byte repeated",
			data: bytes.Repeat([]byte{0xFF}, 100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zx0 := NewZX0()

			// Compress
			compressed := make([]byte, len(tt.data)*2) // Allow space for expansion
			compSize, err := zx0.PackZX0(tt.data, len(tt.data), compressed, false)
			if err != nil {
				t.Fatalf("Compress failed: %v", err)
			}

			if compSize <= 0 {
				t.Fatalf("Compressed size = %d, expected > 0", compSize)
			}

			// Note: Decompression is typically done by CPC Z80 code, not in Go.
			// We test that compression completes without error and produces output.
			t.Logf("Original size: %d, compressed size: %d, ratio: %.2f%%",
				len(tt.data), compSize, float64(compSize)*100.0/float64(len(tt.data)))
		})
	}
}

// TestZX0CompressEmpty tests compression of empty data
func TestZX0CompressEmpty(t *testing.T) {
	zx0 := NewZX0()
	data := []byte{}
	compressed := make([]byte, 100)

	_, err := zx0.PackZX0(data, 0, compressed, false)
	if err == nil {
		t.Error("Expected error for empty input, got nil")
	}
}

// TestZX0CompressSingleByte tests compression of single byte
func TestZX0CompressSingleByte(t *testing.T) {
	zx0 := NewZX0()
	data := []byte{0x42}
	compressed := make([]byte, 100)

	compSize, err := zx0.PackZX0(data, len(data), compressed, false)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if compSize <= 0 {
		t.Errorf("Compressed size = %d, expected > 0", compSize)
	}
}

// TestZX0AllocateFreeBlock tests block allocation and reuse
func TestZX0AllocateFreeBlock(t *testing.T) {
	zx0 := NewZX0()

	// Allocate a block
	block1 := zx0.allocate(10, 100, 50, nil)
	if block1 == nil {
		t.Fatal("allocate returned nil")
	}
	if block1.Bits != 10 {
		t.Errorf("Bits = %d, want 10", block1.Bits)
	}
	if block1.Index != 100 {
		t.Errorf("Index = %d, want 100", block1.Index)
	}
	if block1.Offset != 50 {
		t.Errorf("Offset = %d, want 50", block1.Offset)
	}

	// Allocate another block with chain
	block2 := zx0.allocate(20, 200, 75, block1)
	if block2 == nil {
		t.Fatal("allocate returned nil")
	}
	if block2.Chain != block1 {
		t.Error("Chain not set correctly")
	}
	if block1.References != 1 {
		t.Errorf("block1 References = %d, want 1", block1.References)
	}
}

// TestZX0AssignBlock tests block assignment with reference counting
func TestZX0AssignBlock(t *testing.T) {
	zx0 := NewZX0()

	block1 := zx0.allocate(10, 100, 50, nil)
	var ptr *Block

	// Assign block
	zx0.assign(&ptr, block1)
	if ptr != block1 {
		t.Error("Block not assigned correctly")
	}
	if block1.References != 1 {
		t.Errorf("References = %d, want 1", block1.References)
	}

	// Reassign to new block
	block2 := zx0.allocate(20, 200, 75, nil)
	zx0.assign(&ptr, block2)
	if ptr != block2 {
		t.Error("Block not reassigned correctly")
	}
	if block2.References != 1 {
		t.Errorf("block2 References = %d, want 1", block2.References)
	}
	// block1 should be freed (references = 0) and added to ghost chain
	if block1.References != 0 {
		t.Errorf("block1 References = %d, want 0", block1.References)
	}
}

// TestZX0EliasGammaBits tests Elias gamma encoding bit count
func TestZX0EliasGammaBits(t *testing.T) {
	zx0 := NewZX0()

	tests := []struct {
		value    int
		expected int
	}{
		{1, 1},   // 1 -> 1 bit
		{2, 3},   // 10 -> 3 bits (0,1,0)
		{3, 3},   // 11 -> 3 bits (0,1,1)
		{4, 5},   // 100 -> 5 bits (00,1,00)
		{5, 5},   // 101 -> 5 bits (00,1,01)
		{7, 5},   // 111 -> 5 bits (00,1,11)
		{8, 7},   // 1000 -> 7 bits (000,1,000)
		{15, 7},  // 1111 -> 7 bits (000,1,111)
		{16, 9},  // 10000 -> 9 bits (0000,1,0000)
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := zx0.eliasGammaBits(tt.value)
			if got != tt.expected {
				t.Errorf("eliasGammaBits(%d) = %d, want %d", tt.value, got, tt.expected)
			}
		})
	}
}

// TestZX0CompressLargeData tests compression of larger data sets
func TestZX0CompressLargeData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large data test in short mode")
	}

	// Create a 1KB test pattern
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	zx0 := NewZX0()
	compressed := make([]byte, len(data)*2)

	compSize, err := zx0.PackZX0(data, len(data), compressed, false)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if compSize <= 0 {
		t.Errorf("Compressed size = %d, expected > 0", compSize)
	}

	t.Logf("Compressed 1KB: %d bytes (%.2f%% ratio)",
		compSize, float64(compSize)*100.0/float64(len(data)))
}

// TestZX0CompressHighlyCompressible tests highly compressible data
func TestZX0CompressHighlyCompressible(t *testing.T) {
	// Data with lots of repetition should compress well
	data := bytes.Repeat([]byte{0x00}, 256)

	zx0 := NewZX0()
	compressed := make([]byte, len(data))

	compSize, err := zx0.PackZX0(data, len(data), compressed, false)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// Compressed size should be significantly smaller
	if compSize >= len(data) {
		t.Errorf("Highly compressible data not compressed: original=%d, compressed=%d",
			len(data), compSize)
	}

	t.Logf("Highly compressible: %d -> %d bytes (%.2f%% ratio)",
		len(data), compSize, float64(compSize)*100.0/float64(len(data)))
}

// TestZX0BlockChaining tests that block chaining works correctly
func TestZX0BlockChaining(t *testing.T) {
	zx0 := NewZX0()

	// Create a chain: block3 -> block2 -> block1
	block1 := zx0.allocate(10, 100, 0, nil)
	block2 := zx0.allocate(20, 200, 0, block1)
	block3 := zx0.allocate(30, 300, 0, block2)

	// Verify chain
	if block3.Chain != block2 {
		t.Error("block3.Chain should be block2")
	}
	if block2.Chain != block1 {
		t.Error("block2.Chain should be block1")
	}
	if block1.Chain != nil {
		t.Error("block1.Chain should be nil")
	}

	// Verify reference counts
	if block1.References != 1 {
		t.Errorf("block1.References = %d, want 1", block1.References)
	}
	if block2.References != 1 {
		t.Errorf("block2.References = %d, want 1", block2.References)
	}
}
