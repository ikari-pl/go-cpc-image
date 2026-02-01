// Package cpc provides AMSDOS header handling for CPC files.
package cpc

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestCreeEntete tests AMSDOS header creation with known values
func TestCreeEntete(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		start    uint16
		length   uint16
		entry    uint16
	}{
		{
			name:     "Standard screen file",
			filename: "SCREEN.SCR",
			start:    0xC000,
			length:   0x4000,
			entry:    0xC000,
		},
		{
			name:     "Binary file",
			filename: "code.bin",
			start:    0x1000,
			length:   0x2000,
			entry:    0x1000,
		},
		{
			name:     "Long filename gets truncated",
			filename: "verylongfilename.dat",
			start:    0x4000,
			length:   0x1000,
			entry:    0x4000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := CreeEntete(tt.filename, tt.start, tt.length, tt.entry)

			// Verify basic fields
			if header.Address != tt.start {
				t.Errorf("Address = 0x%04X, want 0x%04X", header.Address, tt.start)
			}
			if header.Length != tt.length {
				t.Errorf("Length = 0x%04X, want 0x%04X", header.Length, tt.length)
			}
			if header.RealLength != tt.length {
				t.Errorf("RealLength = 0x%04X, want 0x%04X", header.RealLength, tt.length)
			}
			if header.LogicalLength != tt.length {
				t.Errorf("LogicalLength = 0x%04X, want 0x%04X", header.LogicalLength, tt.length)
			}
			if header.EntryAddress != tt.entry {
				t.Errorf("EntryAddress = 0x%04X, want 0x%04X", header.EntryAddress, tt.entry)
			}
			if header.FileType != 2 {
				t.Errorf("FileType = %d, want 2 (binary)", header.FileType)
			}

			// Verify checksum is calculated
			if header.CheckSum == 0 {
				t.Error("CheckSum should not be 0")
			}

			// Verify checksum is correct
			checksum := CalcCheckSumStruct(header)
			if int(header.CheckSum) != checksum {
				t.Errorf("CheckSum = %d, calculated = %d", header.CheckSum, checksum)
			}
		})
	}
}

// TestCreeEnteteFilename tests filename formatting in AMSDOS header
func TestCreeEnteteFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "Simple filename",
			filename: "TEST.SCR",
			expected: "TEST    SCR",
		},
		{
			name:     "Short name",
			filename: "AB.C",
			expected: "AB      C  ",
		},
		{
			name:     "No extension",
			filename: "NOEXT",
			expected: "NOEXT      ",
		},
		{
			name:     "Lowercase converted to uppercase",
			filename: "screen.scr",
			expected: "SCREEN  SCR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := CreeEntete(tt.filename, 0xC000, 0x4000, 0xC000)
			got := string(header.FileName[:11])
			if got != tt.expected {
				t.Errorf("Filename = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestCalcCheckSum tests AMSDOS checksum calculation
func TestCalcCheckSum(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected int
	}{
		{
			name:     "All zeros",
			data:     make([]byte, 67),
			expected: 0,
		},
		{
			name:     "All ones",
			data:     bytes.Repeat([]byte{1}, 67),
			expected: 67,
		},
		{
			name:     "Sequential values",
			data:     []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			expected: 55,
		},
		{
			name:     "Handles short arrays",
			data:     []byte{10, 20, 30},
			expected: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcCheckSum(tt.data)
			if got != tt.expected {
				t.Errorf("CalcCheckSum() = %d, want %d", got, tt.expected)
			}
		})
	}
}

// TestCalcCheckSumStruct tests checksum calculation for AMSDOS struct
func TestCalcCheckSumStruct(t *testing.T) {
	// Create a header with known values
	header := CpcAmsdos{
		UserNumber:    0,
		FileType:      2,
		Length:        0x4000,
		Address:       0xC000,
		LogicalLength: 0x4000,
		EntryAddress:  0xC000,
		RealLength:    0x4000,
		BigLength:     0,
	}
	copy(header.FileName[:], "TEST    SCR")

	checksum := CalcCheckSumStruct(header)

	// Verify checksum is non-zero for non-zero data
	if checksum == 0 {
		t.Error("Checksum should not be 0 for non-zero header")
	}

	// Verify checksum is reproducible
	checksum2 := CalcCheckSumStruct(header)
	if checksum != checksum2 {
		t.Errorf("Checksum not reproducible: %d != %d", checksum, checksum2)
	}
}

// TestGetAmsdos tests AMSDOS header parsing from byte array
func TestGetAmsdos(t *testing.T) {
	// Create a known header
	original := CreeEntete("TEST.SCR", 0xC000, 0x4000, 0xC000)

	// Convert to bytes
	headerBytes, err := AmsdosToByte(original)
	if err != nil {
		t.Fatalf("Failed to convert header to bytes: %v", err)
	}

	// Parse it back
	parsed, err := GetAmsdos(headerBytes)
	if err != nil {
		t.Fatalf("Failed to parse header: %v", err)
	}

	// Verify key fields match
	if parsed.Address != original.Address {
		t.Errorf("Address = 0x%04X, want 0x%04X", parsed.Address, original.Address)
	}
	if parsed.Length != original.Length {
		t.Errorf("Length = 0x%04X, want 0x%04X", parsed.Length, original.Length)
	}
	if parsed.EntryAddress != original.EntryAddress {
		t.Errorf("EntryAddress = 0x%04X, want 0x%04X", parsed.EntryAddress, original.EntryAddress)
	}
	if parsed.FileType != original.FileType {
		t.Errorf("FileType = %d, want %d", parsed.FileType, original.FileType)
	}
	if parsed.CheckSum != original.CheckSum {
		t.Errorf("CheckSum = 0x%04X, want 0x%04X", parsed.CheckSum, original.CheckSum)
	}
}

// TestGetAmsdosShortArray tests error handling for short arrays
func TestGetAmsdosShortArray(t *testing.T) {
	shortArray := make([]byte, 100) // Less than 128 bytes
	_, err := GetAmsdos(shortArray)
	if err == nil {
		t.Error("Expected error for array shorter than 128 bytes, got nil")
	}
}

// TestCheckAmsdos tests AMSDOS header checksum verification
func TestCheckAmsdos(t *testing.T) {
	// Create a valid header
	header := CreeEntete("TEST.SCR", 0xC000, 0x4000, 0xC000)
	headerBytes, err := AmsdosToByte(header)
	if err != nil {
		t.Fatalf("Failed to convert header to bytes: %v", err)
	}

	// Valid header should pass
	if !CheckAmsdos(headerBytes) {
		t.Error("Valid header failed checksum verification")
	}

	// Corrupt the header
	corruptBytes := make([]byte, len(headerBytes))
	copy(corruptBytes, headerBytes)
	corruptBytes[10] ^= 0xFF // Flip some bits

	// Corrupted header should fail
	if CheckAmsdos(corruptBytes) {
		t.Error("Corrupted header passed checksum verification")
	}

	// Short array should fail
	if CheckAmsdos(headerBytes[:100]) {
		t.Error("Short array passed checksum verification")
	}
}

// TestAmsdosToByte tests AMSDOS struct to byte array conversion
func TestAmsdosToByte(t *testing.T) {
	header := CreeEntete("TEST.SCR", 0xC000, 0x4000, 0xC000)

	bytes, err := AmsdosToByte(header)
	if err != nil {
		t.Fatalf("AmsdosToByte() error = %v", err)
	}

	if len(bytes) != 128 {
		t.Errorf("AmsdosToByte() returned %d bytes, want 128", len(bytes))
	}

	// Verify we can parse it back
	parsed, err := GetAmsdos(bytes)
	if err != nil {
		t.Fatalf("Failed to parse converted bytes: %v", err)
	}

	if parsed.Address != header.Address {
		t.Errorf("Round-trip Address mismatch: got 0x%04X, want 0x%04X", parsed.Address, header.Address)
	}
}

// TestGetFilename tests filename extraction from AMSDOS header
func TestGetFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "File with extension",
			input:    "TEST.SCR",
			expected: "TEST.SCR",
		},
		{
			name:     "File without extension",
			input:    "NOEXT",
			expected: "NOEXT",
		},
		{
			name:     "Short name with extension",
			input:    "AB.C",
			expected: "AB.C",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := CreeEntete(tt.input, 0xC000, 0x4000, 0xC000)
			got := header.GetFilename()
			if got != tt.expected {
				t.Errorf("GetFilename() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestAmsdosHeaderSize tests that AMSDOS header is exactly 128 bytes
func TestAmsdosHeaderSize(t *testing.T) {
	var header CpcAmsdos
	size := binary.Size(header)
	if size != 128 {
		t.Errorf("CpcAmsdos struct size = %d bytes, want 128", size)
	}
}

// TestAmsdosRoundTrip tests complete round-trip conversion
func TestAmsdosRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		start uint16
		len   uint16
		entry uint16
	}{
		{"Standard screen", 0xC000, 0x4000, 0xC000},
		{"Small binary", 0x8000, 0x0100, 0x8000},
		{"Large binary", 0x4000, 0x8000, 0x4000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create header
			original := CreeEntete("TEST.BIN", tt.start, tt.len, tt.entry)

			// Convert to bytes
			bytes, err := AmsdosToByte(original)
			if err != nil {
				t.Fatalf("AmsdosToByte error: %v", err)
			}

			// Verify checksum
			if !CheckAmsdos(bytes) {
				t.Error("Checksum verification failed")
			}

			// Parse back
			parsed, err := GetAmsdos(bytes)
			if err != nil {
				t.Fatalf("GetAmsdos error: %v", err)
			}

			// Verify all fields match
			if parsed.Address != original.Address {
				t.Errorf("Address mismatch: got 0x%04X, want 0x%04X", parsed.Address, original.Address)
			}
			if parsed.Length != original.Length {
				t.Errorf("Length mismatch: got 0x%04X, want 0x%04X", parsed.Length, original.Length)
			}
			if parsed.EntryAddress != original.EntryAddress {
				t.Errorf("EntryAddress mismatch: got 0x%04X, want 0x%04X", parsed.EntryAddress, original.EntryAddress)
			}
			if parsed.CheckSum != original.CheckSum {
				t.Errorf("CheckSum mismatch: got 0x%04X, want 0x%04X", parsed.CheckSum, original.CheckSum)
			}
		})
	}
}
