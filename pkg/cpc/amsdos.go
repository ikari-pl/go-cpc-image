// Package cpc provides AMSDOS header handling for CPC files.
package cpc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"path"
	"strings"
)

// CpcAmsdos represents the AMSDOS 128-byte header structure.
// This structure must maintain exact binary compatibility with the C# version.
type CpcAmsdos struct {
	UserNumber      uint8     // User number
	FileName        [15]byte  // Filename (8.3 format)
	BlockNum        uint8     // First block number (disk)
	LastBlock       uint8     // Last block number (disk)
	FileType        uint8     // File type (2 for binary)
	Length          uint16    // File length
	Address         uint16    // Load address
	FirstBlock      uint8     // First file block (disk)
	LogicalLength   uint16    // Logical length
	EntryAddress    uint16    // Entry point address
	Unused          [36]byte  // Unused area
	RealLength      uint16    // Real length
	BigLength       uint8     // Real length (3rd byte)
	CheckSum        uint16    // AMSDOS checksum
	Unused2         [59]byte  // Unused area 2
}

// CreeEntete creates an AMSDOS header for the given filename and parameters.
// This replicates the C# CreeEntete function exactly.
func CreeEntete(nomFic string, start uint16, length uint16, entry uint16) CpcAmsdos {
	var entete CpcAmsdos

	// Extract filename and extension
	nom := strings.ToUpper(path.Base(nomFic))
	ext := strings.ToUpper(path.Ext(nomFic))

	// Remove extension from name
	if ext != "" {
		nom = nom[:len(nom)-len(ext)]
		ext = ext[1:] // Remove the dot
	}

	// Convert to AMSDOS 8.3 format
	var result strings.Builder
	for i := 0; i < 8; i++ {
		if i < len(nom) {
			result.WriteByte(nom[i])
		} else {
			result.WriteByte(' ')
		}
	}

	for i := 0; i < 3; i++ {
		if i < len(ext) {
			result.WriteByte(ext[i])
		} else {
			result.WriteByte(' ')
		}
	}

	// Copy filename to header
	copy(entete.FileName[:], result.String())

	// Set header fields
	entete.Address = start
	entete.Length = length
	entete.RealLength = length
	entete.LogicalLength = length
	entete.FileType = 2
	entete.EntryAddress = entry

	// Calculate checksum
	entete.CheckSum = uint16(CalcCheckSumStruct(entete))

	return entete
}

// CalcCheckSum calculates the checksum of the first 67 bytes of a byte array.
func CalcCheckSum(arr []byte) int {
	checkSum := 0
	limit := 67
	if len(arr) < limit {
		limit = len(arr)
	}
	for i := 0; i < limit; i++ {
		checkSum += int(arr[i])
	}
	return checkSum
}

// CalcCheckSumStruct calculates the checksum for a CpcAmsdos struct.
func CalcCheckSumStruct(str CpcAmsdos) int {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.LittleEndian, str)
	if err != nil {
		return 0
	}
	return CalcCheckSum(buf.Bytes())
}

// GetAmsdos parses an AMSDOS header from a byte array.
func GetAmsdos(arr []byte) (CpcAmsdos, error) {
	var str CpcAmsdos
	if len(arr) < 128 {
		return str, fmt.Errorf("AMSDOS header too short: %d bytes, need 128", len(arr))
	}

	buf := bytes.NewReader(arr[:128])
	err := binary.Read(buf, binary.LittleEndian, &str)
	if err != nil {
		return str, fmt.Errorf("failed to parse AMSDOS header: %w", err)
	}

	return str, nil
}

// CheckAmsdos verifies the checksum of an AMSDOS header.
func CheckAmsdos(entete []byte) bool {
	if len(entete) < 128 {
		return false
	}

	enteteAms, err := GetAmsdos(entete)
	if err != nil {
		return false
	}

	return CalcCheckSum(entete) == int(enteteAms.CheckSum)
}

// AmsdosToByte converts a CpcAmsdos struct to a byte array.
func AmsdosToByte(obj CpcAmsdos) ([]byte, error) {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.LittleEndian, obj)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize AMSDOS header: %w", err)
	}
	return buf.Bytes(), nil
}

// GetFilename extracts the filename from an AMSDOS header.
func (a CpcAmsdos) GetFilename() string {
	filename := string(a.FileName[:])
	// Find the end of the name part (first 8 chars)
	name := strings.TrimRight(filename[:8], " ")
	ext := strings.TrimRight(filename[8:11], " ")

	if ext != "" {
		return name + "." + ext
	}
	return name
}