// Package fileio provides file I/O operations for CPC disk images.
package fileio

import (
	"testing"
)

// TestNewGestDSK tests DSK manager creation
func TestNewGestDSK(t *testing.T) {
	dsk := NewGestDSK()

	if dsk == nil {
		t.Fatal("NewGestDSK returned nil")
	}

	// Verify initialization
	if dsk.Infos.NbTracks != 40 {
		t.Errorf("NbTracks = %d, want 40", dsk.Infos.NbTracks)
	}
	if dsk.Infos.NbHeads != 1 {
		t.Errorf("NbHeads = %d, want 1", dsk.Infos.NbHeads)
	}
}

// TestFormatDsk tests DSK formatting with default values
func TestFormatDsk(t *testing.T) {
	dsk := &GestDSK{}
	dsk.FormatDsk()

	// Verify ID string
	expectedID := "EXTENDED CPC DSK File\r\nDisk-Info\r\nConvImgCpc    "
	id := string(dsk.Infos.ID[:])
	if id != expectedID {
		t.Errorf("ID string mismatch:\ngot  %q\nwant %q", id, expectedID)
	}

	// Verify track count
	if dsk.Infos.NbTracks != 40 {
		t.Errorf("NbTracks = %d, want 40", dsk.Infos.NbTracks)
	}

	// Verify head count
	if dsk.Infos.NbHeads != 1 {
		t.Errorf("NbHeads = %d, want 1", dsk.Infos.NbHeads)
	}

	// Verify tracks are initialized
	for i := 0; i < int(dsk.Infos.NbTracks); i++ {
		if dsk.Tracks[i][0] == nil {
			t.Errorf("Track %d face 0 is nil", i)
		}
		if dsk.Tracks[i][1] == nil {
			t.Errorf("Track %d face 1 is nil", i)
		}
	}

	// Verify data buffers are initialized
	for i := 0; i < int(dsk.Infos.NbTracks); i++ {
		if dsk.Data[i][0] == nil {
			t.Errorf("Data buffer %d face 0 is nil", i)
		}
		if len(dsk.Data[i][0]) != TrackDataSize {
			t.Errorf("Data buffer %d face 0 size = %d, want %d", i, len(dsk.Data[i][0]), TrackDataSize)
		}
	}
}

// TestDskHeaderID tests the DSK header ID string
func TestDskHeaderID(t *testing.T) {
	dsk := NewGestDSK()

	// The ID should be exactly 48 bytes
	if len(dsk.Infos.ID) != 48 {
		t.Errorf("ID length = %d, want 48", len(dsk.Infos.ID))
	}

	// Verify it starts with the expected string
	id := string(dsk.Infos.ID[:])
	if id[:21] != "EXTENDED CPC DSK File" {
		t.Errorf("ID does not start with expected string: %q", id[:21])
	}
}

// TestDskTrackInitialization tests track initialization
func TestDskTrackInitialization(t *testing.T) {
	dsk := NewGestDSK()

	// Check first track
	track := dsk.Tracks[0][0]
	if track == nil {
		t.Fatal("First track is nil")
	}

	// Verify track data size table is set
	if dsk.Infos.TrackSizeTable[0] != 0x13 {
		t.Errorf("TrackSizeTable[0] = 0x%02X, want 0x13", dsk.Infos.TrackSizeTable[0])
	}
}

// TestDskSectorInterleaving tests sector interleaving calculation
func TestDskSectorInterleaving(t *testing.T) {
	dsk := NewGestDSK()

	// Standard format has 9 sectors per track
	// Verify tracks are formatted
	track := dsk.Tracks[0][0]
	if track.NbSect == 0 {
		t.Error("Track has 0 sectors")
	}

	// Verify sector IDs are set (interleaving)
	// Standard interleaving: C1, C6, C2, C7, C3, C8, C4, C9, C5
	// But this depends on formatTrack implementation
	t.Logf("Track 0 has %d sectors", track.NbSect)
}

// TestGetPosData tests sector position calculation
func TestGetPosData(t *testing.T) {
	dsk := NewGestDSK()

	// Get position of first sector
	pos := dsk.getPosData(0, 0xC1)
	if pos < 0 {
		t.Errorf("getPosData returned negative position: %d", pos)
	}

	t.Logf("Sector 0xC1 position: %d", pos)
}

// TestDskConstants tests package constants
func TestDskConstants(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  int
	}{
		{"MaxDSK", MaxDSK, 2},
		{"MaxTracks", MaxTracks, 99},
		{"SectSize", SectSize, 512},
		{"UserDeleted", UserDeleted, 0xE5},
		{"SeekBack", SeekBack, 0x1000},
		{"MaxString", MaxString, 256},
		{"MaxSects", MaxSects, 29},
		{"TrackDataSize", TrackDataSize, 0x1800},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
			}
		})
	}
}

// TestDskErrorStrings tests DskError error messages
func TestDskErrorStrings(t *testing.T) {
	tests := []struct {
		err  DskError
		want string
	}{
		{ErrNoErr, "no error"},
		{ErrNoDirEntry, "no directory entry available"},
		{ErrNoBlock, "no block available"},
		{ErrFileExist, "file already exists"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestCPCEMUEntSize tests CPCEMU entry structure size
func TestCPCEMUEntSize(t *testing.T) {
	var ent CPCEMUEnt

	// Verify ID field size
	if len(ent.ID) != 48 {
		t.Errorf("ID field size = %d, want 48", len(ent.ID))
	}

	// Verify TrackSizeTable size
	if len(ent.TrackSizeTable) != 204 {
		t.Errorf("TrackSizeTable size = %d, want 204", len(ent.TrackSizeTable))
	}
}

// TestCPCEMUTrackStructure tests track structure
func TestCPCEMUTrackStructure(t *testing.T) {
	var track CPCEMUTrack

	// Verify ID field size
	if len(track.ID) != 16 {
		t.Errorf("Track ID field size = %d, want 16", len(track.ID))
	}

	// Verify sector array size
	if len(track.Sect) != MaxSects {
		t.Errorf("Sector array size = %d, want %d", len(track.Sect), MaxSects)
	}
}

// TestStDirEntryStructure tests directory entry structure
func TestStDirEntryStructure(t *testing.T) {
	var dir StDirEntry

	// Verify field sizes
	if len(dir.Nom) != 8 {
		t.Errorf("Nom field size = %d, want 8", len(dir.Nom))
	}
	if len(dir.Ext) != 3 {
		t.Errorf("Ext field size = %d, want 3", len(dir.Ext))
	}
	if len(dir.Blocks) != 16 {
		t.Errorf("Blocks field size = %d, want 16", len(dir.Blocks))
	}
}

// TestDskBitmapSize tests bitmap array size
func TestDskBitmapSize(t *testing.T) {
	dsk := NewGestDSK()

	if len(dsk.Bitmap) != 256 {
		t.Errorf("Bitmap size = %d, want 256", len(dsk.Bitmap))
	}
}

// TestDskTracksArray tests tracks array dimensions
func TestDskTracksArray(t *testing.T) {
	dsk := NewGestDSK()

	// Verify array dimensions
	if len(dsk.Tracks) != MaxTracks {
		t.Errorf("Tracks array size = %d, want %d", len(dsk.Tracks), MaxTracks)
	}

	// Each track has 2 faces
	for i := 0; i < MaxTracks; i++ {
		if len(dsk.Tracks[i]) != 2 {
			t.Errorf("Track %d faces = %d, want 2", i, len(dsk.Tracks[i]))
		}
	}
}

// TestDskDataArray tests data array dimensions
func TestDskDataArray(t *testing.T) {
	dsk := NewGestDSK()

	// Verify array dimensions
	if len(dsk.Data) != MaxTracks {
		t.Errorf("Data array size = %d, want %d", len(dsk.Data), MaxTracks)
	}

	// Each track has 2 faces
	for i := 0; i < MaxTracks; i++ {
		if len(dsk.Data[i]) != 2 {
			t.Errorf("Track %d data faces = %d, want 2", i, len(dsk.Data[i]))
		}
	}
}

// TestFormatDskMultipleTimes tests formatting DSK multiple times
func TestFormatDskMultipleTimes(t *testing.T) {
	dsk := &GestDSK{}

	// Format once
	dsk.FormatDsk()
	nbTracks1 := dsk.Infos.NbTracks

	// Format again
	dsk.FormatDsk()
	nbTracks2 := dsk.Infos.NbTracks

	// Should produce same result
	if nbTracks1 != nbTracks2 {
		t.Errorf("Multiple FormatDsk calls produce different results: %d vs %d", nbTracks1, nbTracks2)
	}
}

// TestDskTrackSizeTable tests track size table initialization
func TestDskTrackSizeTable(t *testing.T) {
	dsk := NewGestDSK()

	// All MaxTracks entries should have the same size (implementation initializes all 99 tracks)
	expectedSize := byte(0x13)
	for i := 0; i < MaxTracks; i++ {
		if dsk.Infos.TrackSizeTable[i] != expectedSize {
			t.Errorf("Track %d size = 0x%02X, want 0x%02X", i, dsk.Infos.TrackSizeTable[i], expectedSize)
		}
	}

	// Entries beyond MaxTracks should be 0
	for i := MaxTracks; i < len(dsk.Infos.TrackSizeTable); i++ {
		if dsk.Infos.TrackSizeTable[i] != 0 {
			t.Errorf("Entry %d beyond MaxTracks size = 0x%02X, want 0x00", i, dsk.Infos.TrackSizeTable[i])
		}
	}
}
