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

// AMSDOS header field offsets. The file type byte sits immediately before the
// two length bytes in both layouts; only its offset differs.
const (
	// AmsdosTypeOffset is the file type byte in the header layout this package
	// reads and writes (CpcAmsdos, CreeEntete).
	AmsdosTypeOffset = 0x12
	// AmsdosStockTypeOffset is the file type byte in the layout a stock AMSDOS
	// writer produces: its name field is the 11 bytes of the 8.3 name, so every
	// byte after it sits four bytes earlier than in this package's structure,
	// which keeps a 15-byte name field (see AmsdosNameLen).
	AmsdosStockTypeOffset = 0x0E
)

// AmsdosNameLen is the length of the 8.3 name and extension inside the 15-byte
// FileName field; the four bytes after it are padding, not name.
const AmsdosNameLen = 11

// readableAmsdosName reports whether the name field holds printable ASCII
// (spaces included), i.e. whether it could be a file name at all.
func readableAmsdosName(name [15]byte) bool {
	for _, b := range name[:AmsdosNameLen] {
		if b < 0x20 || b > 0x7E {
			return false
		}
	}
	return true
}

// anyNonSpace reports whether b holds a character that is not a space.
func anyNonSpace(b []byte) bool {
	for _, c := range b {
		if c != ' ' {
			return true
		}
	}
	return false
}

// amsdosTypeOffset returns the offset of the file type byte of the layout
// entete uses: this package's own offset when that byte holds a file type, the
// stock AMSDOS offset otherwise. ok is false when neither holds one, which
// means the prefix is not a header of either kind.
func amsdosTypeOffset(entete []byte) (offset int, ok bool) {
	if entete[AmsdosTypeOffset] <= 2 {
		return AmsdosTypeOffset, true
	}
	if entete[AmsdosStockTypeOffset] <= 2 {
		return AmsdosStockTypeOffset, true
	}
	return 0, false
}

// AmsdosNameReadable reports whether the header's name field holds printable
// ASCII (spaces allowed): the weaker test that tells a file name from binary
// noise, so a caller can decide whether a broken header is worth reporting.
func AmsdosNameReadable(entete []byte) bool {
	if len(entete) < 128 {
		return false
	}
	h, err := GetAmsdos(entete)
	if err != nil {
		return false
	}
	return readableAmsdosName(h.FileName)
}

// AmsdosFieldsPlausible reports whether the identifying fields of a header look
// like an AMSDOS one: a printable 8.3 name with at least one character that is
// not a space, and a file type byte in {0, 1, 2} in one of the two layouts a
// writer may use. The checksum is deliberately not part of this test — writers
// do get it wrong and their files are still read as headed — while the fields
// here are what separates a header from whatever bytes happen to start a file:
// 128 zero bytes satisfy a checksum comparison (0 sums to a stored 0) but have
// no name, which is exactly how a zeroed prefix used to be taken for a header.
func AmsdosFieldsPlausible(entete []byte) bool {
	if len(entete) < 128 {
		return false
	}
	h, err := GetAmsdos(entete)
	if err != nil {
		return false
	}
	if !readableAmsdosName(h.FileName) || !anyNonSpace(h.FileName[:AmsdosNameLen]) {
		return false
	}
	_, ok := amsdosTypeOffset(entete)
	return ok
}

// AmsdosIssues lists the problems that keep a 128-byte prefix from being a
// usable AMSDOS header, one entry per problem, and returns nil when there are
// none. payloadLen is the number of bytes that follow the header, or -1 when
// the caller does not know; a length field that disagrees is reported but is
// never decisive, because storing 0 there is a legitimate way of saying
// "unknown" and some writers do exactly that.
func AmsdosIssues(entete []byte, payloadLen int) []string {
	if len(entete) < 128 {
		return []string{fmt.Sprintf("prefix is %d bytes, not a 128-byte header", len(entete))}
	}

	h, err := GetAmsdos(entete)
	if err != nil {
		return []string{err.Error()}
	}

	var issues []string
	if !readableAmsdosName(h.FileName) {
		issues = append(issues, "file name is not printable 8.3 text")
	} else if !anyNonSpace(h.FileName[:AmsdosNameLen]) {
		issues = append(issues, "file name is blank")
	}

	typeOffset, ok := amsdosTypeOffset(entete)
	if !ok {
		issues = append(issues, fmt.Sprintf("file type is not 0, 1 or 2 (%d at 0x%02X, %d at 0x%02X)",
			entete[AmsdosTypeOffset], AmsdosTypeOffset, entete[AmsdosStockTypeOffset], AmsdosStockTypeOffset))
	}

	if sum := CalcCheckSum(entete); sum != int(h.CheckSum) {
		issues = append(issues, fmt.Sprintf("checksum is %d, the header stores %d", sum, h.CheckSum))
	}

	if ok && payloadLen >= 0 {
		// The length follows the type byte in both layouts.
		length := int(entete[typeOffset+1]) | int(entete[typeOffset+2])<<8
		if length != 0 && length != payloadLen {
			issues = append(issues, fmt.Sprintf("length field is %d, the payload is %d bytes", length, payloadLen))
		}
	}

	return issues
}

// CheckAmsdos verifies that a 128-byte prefix is a usable AMSDOS header: its
// checksum has to match AND its identifying fields have to be plausible. The
// checksum alone is not enough — 128 zero bytes sum to 0 and store 0, so a
// zeroed prefix would otherwise pass and silently lose its first 128 bytes as a
// "header" — and the fields alone are not enough either, because a checksum
// that disagrees means the header cannot be trusted to describe its file.
func CheckAmsdos(entete []byte) bool {
	if len(entete) < 128 {
		return false
	}

	if !AmsdosFieldsPlausible(entete) {
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