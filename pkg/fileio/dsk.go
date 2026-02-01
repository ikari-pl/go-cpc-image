// Package fileio provides file I/O operations for CPC disk images.
package fileio

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	MaxDSK       = 2       // Number of DSK files managed
	MaxTracks    = 99      // Maximum number of tracks per DSK
	SectSize     = 512     // Standard sector size
	UserDeleted  = 0xE5    // Byte for deleted file
	SeekBack     = 0x1000  // For compression algorithms
	MaxString    = 256     // Maximum string length for compression
	MaxSects     = 29      // Maximum sectors per track
	TrackDataSize = 0x1800 // Track data size
)

// DskError represents errors that can occur during DSK operations
type DskError int

const (
	ErrNoErr DskError = iota
	ErrNoDirEntry
	ErrNoBlock
	ErrFileExist
)

func (e DskError) Error() string {
	switch e {
	case ErrNoErr:
		return "no error"
	case ErrNoDirEntry:
		return "no directory entry available"
	case ErrNoBlock:
		return "no block available"
	case ErrFileExist:
		return "file already exists"
	default:
		return "unknown error"
	}
}

// CPCEMUEnt represents the DSK file header
type CPCEMUEnt struct {
	ID             [48]byte // "EXTENDED CPC DSK File\r\nDisk-Info\r\nConvImgCpc    "
	NbTracks       byte
	NbHeads        byte
	TrackSize      uint16
	TrackSizeTable [204]byte
}

// CPCEMUSect represents a sector in a track
type CPCEMUSect struct {
	C        byte   // track
	H        byte   // head
	R        byte   // sector
	N        byte   // size
	ST1      byte   // Status 1
	ST2      byte   // Status 2
	SectSize uint16 // Sector size in bytes
}

// CPCEMUTrack represents a track
type CPCEMUTrack struct {
	ID      [16]byte                // "Track-Info\r\n    "
	Track   byte
	Head    byte
	Unused  uint16
	SectSize byte
	NbSect  byte
	Gap3    byte
	OctRemp byte
	Sect    [MaxSects]CPCEMUSect
}

// StDirEntry represents a directory entry
type StDirEntry struct {
	User    byte
	Nom     [8]byte  // filename
	Ext     [3]byte  // extension
	NumPage byte
	Unused  uint16
	NbPages byte
	Blocks  [16]byte
}

// GestDSK manages DSK disk images
type GestDSK struct {
	NomFic    string
	Infos     CPCEMUEnt
	Tracks    [MaxTracks][2]*CPCEMUTrack
	FlagWrite byte
	Data      [MaxTracks][2][]byte
	Bitmap    [256]byte
}

// NewGestDSK creates a new DSK manager
func NewGestDSK() *GestDSK {
	g := &GestDSK{}
	g.FormatDsk()
	return g
}

// FormatDsk formats the DSK with default values
func (g *GestDSK) FormatDsk() {
	// Set the ID string
	id := "EXTENDED CPC DSK File\r\nDisk-Info\r\nConvImgCpc    "
	copy(g.Infos.ID[:], []byte(id))

	g.Infos.NbTracks = 40
	g.Infos.NbHeads = 1

	// Initialize tracks and data
	for t := 0; t < MaxTracks; t++ {
		// Initialize tracks
		g.Tracks[t][0] = &CPCEMUTrack{}
		g.Tracks[t][1] = &CPCEMUTrack{}

		// Initialize data
		g.Data[t][0] = make([]byte, TrackDataSize)
		g.Data[t][1] = make([]byte, TrackDataSize)

		// Format tracks with default parameters
		g.formatTrack(t, 0, 0xC1, 9) // Default format = data (face 0)
		g.formatTrack(t, 1, 0xC1, 9) // Default format = data (face 1)

		g.Infos.TrackSizeTable[t] = 0x13
	}
}

// getPosData returns the position of a sector in the DSK file
func (g *GestDSK) getPosData(track, sect int) int {
	pos := 0
	tr := g.Tracks[track][0]

	for s := 0; s < int(tr.NbSect); s++ {
		if tr.Sect[s].R == byte(sect) {
			break
		}

		if tr.Sect[s].SectSize != 0 {
			pos += int(tr.Sect[s].SectSize)
		} else {
			pos += 128 << tr.Sect[s].N
		}
	}
	return pos
}

// getIndexSecteur returns the index (in the sector table) of a sector
func (g *GestDSK) getIndexSecteur(track, sect int) int {
	tr := g.Tracks[track][0]
	for s := 0; s < int(tr.NbSect); s++ {
		if tr.Sect[s].R == byte(sect) {
			return s
		}
	}
	return -1
}

// getMinSect finds the smallest sector of a track
func (g *GestDSK) getMinSect(forceStd bool) int {
	sect := 0x100
	tr := g.Tracks[0][0]

	for s := 0; s < int(tr.NbSect); s++ {
		curSect := int(tr.Sect[s].R)
		if forceStd {
			if curSect == 0x01 || curSect == 0x41 || curSect == 0xC1 {
				return curSect
			}
		}

		if sect > curSect {
			sect = curSect
		}
	}
	return sect
}

// getInfoDirEntry returns a directory entry
func (g *GestDSK) getInfoDirEntry(numDir int) StDirEntry {
	var dir StDirEntry
	minSect := g.getMinSect(true)
	s := (numDir >> 4) + minSect

	var t int
	switch minSect {
	case 0x41:
		t = 2
	case 1:
		t = 1
	default:
		t = 0
	}

	pos := g.getPosData(t, s) + ((numDir & 15) << 5)

	// Read directory entry from data
	data := g.Data[t][0][pos:pos+32] // Directory entry is 32 bytes

	dir.User = data[0]
	copy(dir.Nom[:], data[1:9])
	copy(dir.Ext[:], data[9:12])
	dir.NumPage = data[12]
	dir.Unused = binary.LittleEndian.Uint16(data[13:15])
	dir.NbPages = data[15]
	copy(dir.Blocks[:], data[16:32])

	return dir
}

// fillBitmap fills the bitmap of used blocks
func (g *GestDSK) fillBitmap() int {
	nbKo := 0

	// Clear bitmap
	for i := range g.Bitmap {
		g.Bitmap[i] = 0
	}
	g.Bitmap[0] = 1

	for i := 0; i < 64; i++ {
		dir := g.getInfoDirEntry(i)
		if dir.User != UserDeleted {
			for j := 0; j < 16; j++ {
				b := int(dir.Blocks[j])
				if b > 1 && g.Bitmap[b] == 0 {
					g.Bitmap[b] = 1
					nbKo++
				}
			}
		}
	}
	return nbKo
}

// compareNomsAmsdos compares AMSDOS names (without special codes)
func (g *GestDSK) compareNomsAmsdos(n1, n2 string) int {
	for i := 0; i < 11 && i < len(n1) && i < len(n2); i++ {
		if (n1[i] & 0x7F) != (n2[i] & 0x7F) {
			return 1
		}
	}
	return 0
}

// fileExist checks if a file exists, returns the file index if it exists, -1 otherwise
func (g *GestDSK) fileExist(nom string, user int) int {
	for i := 0; i < 64; i++ {
		dir := g.getInfoDirEntry(i)
		fullName := string(dir.Nom[:]) + string(dir.Ext[:])
		if int(dir.User) == user && g.compareNomsAmsdos(nom, fullName) == 0 {
			return i
		}
	}
	return -1
}

// getNomDir converts a filename to directory entry format
func (g *GestDSK) getNomDir(nomFic string) StDirEntry {
	var dirLoc StDirEntry

	var nom, ext string
	if p := strings.Index(nomFic, "."); p > 0 {
		nom = nomFic[:p]
		ext = nomFic[p+1:]
	} else {
		if len(nomFic) > 8 {
			nom = strings.ToUpper(nomFic[:8])
			ext = strings.ToUpper(nomFic[8:])
		} else {
			nom = strings.ToUpper(nomFic)
			ext = "BIN"
		}
	}

	// Pad to required lengths
	for len(nom) < 8 {
		nom += " "
	}
	for len(ext) < 3 {
		ext += " "
	}

	copy(dirLoc.Nom[:], []byte(nom))
	copy(dirLoc.Ext[:], []byte(ext))

	return dirLoc
}

// findFreeDir searches for a free directory entry.
func (g *GestDSK) findFreeDir() int {
	for i := 0; i < 64; i++ {
		dir := g.getInfoDirEntry(i)
		if dir.User == UserDeleted {
			return i
		}
	}
	return -1
}

// findFreeBlock searches for a free block and marks it as used.
func (g *GestDSK) findFreeBlock(maxBloc int) int {
	for i := 2; i < maxBloc; i++ {
		if g.Bitmap[i] == 0 {
			g.Bitmap[i] = 1
			return i
		}
	}
	return 0
}

// formatTrack formats a track
func (g *GestDSK) formatTrack(t, h, minSect, nbSect int) {
	tr := g.Tracks[t][h]

	// Fill with deleted user marker
	for i := 0; i < TrackDataSize; i++ {
		g.Data[t][h][i] = UserDeleted
	}

	// Set track info
	copy(tr.ID[:], []byte("Track-Info\r\n    "))
	tr.Track = byte(t)
	tr.Head = byte(h)
	tr.SectSize = 2
	tr.NbSect = byte(nbSect)
	tr.Gap3 = 0x4E
	tr.OctRemp = UserDeleted

	readSect := 0

	// Handle sector interleaving
	for s := 0; s < nbSect; {
		tr.Sect[s].C = byte(t)
		tr.Sect[s].H = byte(h)
		tr.Sect[s].R = byte(readSect + minSect)
		tr.Sect[s].N = 2
		tr.Sect[s].SectSize = SectSize
		readSect++

		s++
		if s < nbSect {
			tr.Sect[s].C = byte(t)
			tr.Sect[s].H = byte(h)
			tr.Sect[s].R = byte(readSect + minSect + 4)
			tr.Sect[s].N = 2
			tr.Sect[s].SectSize = SectSize
			s++
		}
	}
}

// writeBloc writes an AMSDOS block (1 block = 2 sectors)
func (g *GestDSK) writeBloc(bloc int, bufBloc []byte, offset int) {
	track := (bloc << 1) / 9
	sect := (bloc << 1) % 9
	minSect := g.getMinSect(true)

	switch minSect {
	case 0x41:
		track += 2
	case 0x01:
		track++
	}

	// Adjust number of tracks if capacity exceeded
	if track > int(g.Infos.NbTracks)-1 {
		g.Infos.NbTracks = byte(track + 1)
		g.formatTrack(track, 0, minSect, 9)
	}

	pos := g.getPosData(track, sect+minSect)
	l := min(SectSize, len(bufBloc)-offset)
	if l > 0 {
		copy(g.Data[track][0][pos:pos+l], bufBloc[offset:offset+l])
	}

	sect++
	if sect > 8 {
		track++
		sect = 0
	}

	pos = g.getPosData(track, sect+minSect)
	l = min(SectSize, len(bufBloc)-offset-SectSize)
	if l > 0 {
		copy(g.Data[track][0][pos:pos+l], bufBloc[offset+SectSize:offset+SectSize+l])
	}
}

// setInfoDirEntry sets an entry in the directory
func (g *GestDSK) setInfoDirEntry(numDir int, dir StDirEntry) {
	minSect := g.getMinSect(true)
	s := (numDir >> 4) + minSect

	var t int
	switch minSect {
	case 0x41:
		t = 2
	case 1:
		t = 1
	default:
		t = 0
	}

	pos := g.getPosData(t, s) + ((numDir & 15) << 5)

	// Write directory entry to data
	data := g.Data[t][0][pos:pos+32]
	data[0] = dir.User
	copy(data[1:9], dir.Nom[:])
	copy(data[9:12], dir.Ext[:])
	data[12] = dir.NumPage
	binary.LittleEndian.PutUint16(data[13:15], dir.Unused)
	data[15] = dir.NbPages
	copy(data[16:32], dir.Blocks[:])
}

// CopieFichier copies a file to the DSK
func (g *GestDSK) CopieFichier(bufFile []byte, nomFic string, tailleFic, maxBloc, user int) DskError {
	g.fillBitmap()
	dirLoc := g.getNomDir(nomFic)
	fullName := string(dirLoc.Nom[:]) + string(dirLoc.Ext[:])

	if g.fileExist(fullName, user) > -1 {
		return ErrFileExist
	}

	nbPages := 0
	posFile := 0

	for posFile < tailleFic {
		posDir := g.findFreeDir()
		if posDir == -1 {
			return ErrNoDirEntry
		}

		dirLoc.User = byte(user)
		dirLoc.NumPage = byte(nbPages)
		nbPages++

		taillePage := min(128, ((tailleFic-posFile)+127)>>7)
		dirLoc.NbPages = byte(taillePage)
		l := (int(dirLoc.NbPages) + 7) >> 3

		for j := 0; j < l; j++ {
			bloc := g.findFreeBlock(maxBloc)
			if bloc == 0 {
				return ErrNoBlock
			}

			dirLoc.Blocks[j] = byte(bloc)
			g.writeBloc(bloc, bufFile, posFile)
			posFile += 1024
		}

		g.setInfoDirEntry(posDir, dirLoc)
	}

	return ErrNoErr
}

// Save saves the DSK to a file
func (g *GestDSK) Save(fileName string) error {
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write header
	if err := binary.Write(file, binary.LittleEndian, g.Infos); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write tracks
	for t := 0; t < int(g.Infos.NbTracks); t++ {
		for h := 0; h < int(g.Infos.NbHeads); h++ {
			tr := g.Tracks[t][h]

			// Write track header
			if err := binary.Write(file, binary.LittleEndian, tr); err != nil {
				return fmt.Errorf("failed to write track %d/%d: %w", t, h, err)
			}

			// Calculate data size
			tailleData := 0
			for s := 0; s < int(tr.NbSect); s++ {
				sect := &tr.Sect[s]
				n := int(sect.SectSize)

				if (n & 0xFF) > 0 {
					n = (n + 0xFF) & 0x1F00
				} else if n == 0 {
					n = int(sect.N)
					if n < 6 {
						n = 128 << n
					} else {
						n = TrackDataSize
					}
				}

				if n > 0x100 {
					tailleData += n
				} else {
					tailleData += 0x100
				}
			}

			// Write track data
			if _, err := file.Write(g.Data[t][h][:tailleData]); err != nil {
				return fmt.Errorf("failed to write track data %d/%d: %w", t, h, err)
			}
		}
	}

	return nil
}

// Load loads a DSK from a file
func (g *GestDSK) Load(fileName string) error {
	g.NomFic = fileName

	file, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read header
	if err := binary.Read(file, binary.LittleEndian, &g.Infos); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	// Read tracks
	for t := 0; t < int(g.Infos.NbTracks); t++ {
		for h := 0; h < int(g.Infos.NbHeads); h++ {
			tr := g.Tracks[t][h]

			// Read track header
			if err := binary.Read(file, binary.LittleEndian, tr); err != nil {
				return fmt.Errorf("failed to read track %d/%d: %w", t, h, err)
			}

			// Check track ID
			idStr := string(tr.ID[:])
			if len(idStr) < 10 || !strings.HasPrefix(idStr, "Track-Info") {
				break
			}

			// If single face, copy face 0 to face 1
			if tr.Head == 0 && g.Infos.NbHeads == 1 {
				g.Tracks[t][1] = g.Tracks[t][0]
				g.Data[t][1] = g.Data[t][0]
			}

			// Calculate data size
			tailleData := 0
			for s := 0; s < int(tr.NbSect); s++ {
				sect := &tr.Sect[s]
				n := int(sect.SectSize)

				if (n & 0xFF) > 0 {
					n = (n + 0xFF) & 0x1F00
				} else if n == 0 {
					n = int(sect.N)
					if n < 6 {
						n = 128 << n
					} else {
						n = TrackDataSize
					}
				}

				if n > 0x100 {
					tailleData += n
				} else {
					tailleData += 0x100
				}
			}

			// Read track data
			g.Data[t][h] = make([]byte, tailleData)
			if _, err := io.ReadFull(file, g.Data[t][h]); err != nil {
				return fmt.Errorf("failed to read track data %d/%d: %w", t, h, err)
			}
		}
	}

	return nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}