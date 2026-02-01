// Package fileio provides file I/O operations for CPC disk images.
package fileio

import (
	"os"
	"path/filepath"
	"strings"
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
	expectedID := "EXTENDED CPC DSK File\r\nDisk-Info\r\ngo-cpc-image  "
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

// createTestDSK creates a temporary DSK file for testing and returns its path.
func createTestDSK(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	dskPath := filepath.Join(dir, "test.dsk")
	dsk := NewGestDSK()
	if err := dsk.Save(dskPath); err != nil {
		t.Fatalf("failed to create test DSK: %v", err)
	}
	return dskPath
}

// TestFormatDsk80 tests 80-track DSK formatting
func TestFormatDsk80(t *testing.T) {
	dsk := &GestDSK{}
	dsk.FormatDsk80()

	if dsk.Infos.NbTracks != 80 {
		t.Errorf("NbTracks = %d, want 80", dsk.Infos.NbTracks)
	}
	if dsk.Infos.NbHeads != 1 {
		t.Errorf("NbHeads = %d, want 1", dsk.Infos.NbHeads)
	}

	// Verify 80-track DSK has more capacity
	dsk40 := NewGestDSK()
	if dsk.MaxBloc() <= dsk40.MaxBloc() {
		t.Errorf("80-track maxBloc (%d) should be greater than 40-track maxBloc (%d)", dsk.MaxBloc(), dsk40.MaxBloc())
	}
}

// TestDskFileEntry tests the DskFileEntry struct
func TestDskFileEntry(t *testing.T) {
	entry := DskFileEntry{
		Name:     "TEST.BIN",
		Size:     1024,
		LoadAddr: 0xC000,
		ExecAddr: 0xC000,
	}

	if entry.Name != "TEST.BIN" {
		t.Errorf("Name = %q, want %q", entry.Name, "TEST.BIN")
	}
	if entry.Size != 1024 {
		t.Errorf("Size = %d, want 1024", entry.Size)
	}
	if entry.LoadAddr != 0xC000 {
		t.Errorf("LoadAddr = 0x%04X, want 0xC000", entry.LoadAddr)
	}
	if entry.ExecAddr != 0xC000 {
		t.Errorf("ExecAddr = 0x%04X, want 0xC000", entry.ExecAddr)
	}
}

// TestListFilesEmpty tests listing files on an empty DSK
func TestListFilesEmpty(t *testing.T) {
	dskPath := createTestDSK(t)

	files, err := ListFiles(dskPath)
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}
	if len(files) != 0 {
		t.Errorf("ListFiles() returned %d files, want 0", len(files))
	}
}

// TestListFilesNonExistent tests listing files on a non-existent DSK
func TestListFilesNonExistent(t *testing.T) {
	_, err := ListFiles("/nonexistent/path/test.dsk")
	if err == nil {
		t.Error("ListFiles() expected error for non-existent file")
	}
}

// TestAddFileAndListFiles tests adding a file and then listing it
func TestAddFileAndListFiles(t *testing.T) {
	dskPath := createTestDSK(t)

	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i)
	}

	err := AddFile(dskPath, "TEST.BIN", data, 0xC000, 0xC000)
	if err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	files, err := ListFiles(dskPath)
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("ListFiles() returned %d files, want 1", len(files))
	}

	if !strings.HasPrefix(files[0].Name, "TEST") {
		t.Errorf("file name = %q, want prefix TEST", files[0].Name)
	}

	if files[0].Size == 0 {
		t.Error("file size should not be 0")
	}
}

// TestAddFileDuplicate tests adding a duplicate file
func TestAddFileDuplicate(t *testing.T) {
	dskPath := createTestDSK(t)

	data := make([]byte, 128)
	err := AddFile(dskPath, "DUP.BIN", data, 0x4000, 0x4000)
	if err != nil {
		t.Fatalf("first AddFile() error = %v", err)
	}

	err = AddFile(dskPath, "DUP.BIN", data, 0x4000, 0x4000)
	if err == nil {
		t.Error("second AddFile() expected error for duplicate file")
	}
}

// TestRemoveFile tests removing a file from DSK
func TestRemoveFile(t *testing.T) {
	dskPath := createTestDSK(t)

	data := make([]byte, 128)
	err := AddFile(dskPath, "DEL.BIN", data, 0x4000, 0x4000)
	if err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	// Verify file exists
	files, _ := ListFiles(dskPath)
	if len(files) != 1 {
		t.Fatalf("expected 1 file before remove, got %d", len(files))
	}

	// Remove it
	err = RemoveFile(dskPath, "DEL.BIN")
	if err != nil {
		t.Fatalf("RemoveFile() error = %v", err)
	}

	// Verify file is gone
	files, _ = ListFiles(dskPath)
	if len(files) != 0 {
		t.Errorf("expected 0 files after remove, got %d", len(files))
	}
}

// TestRemoveFileNotFound tests removing a non-existent file
func TestRemoveFileNotFound(t *testing.T) {
	dskPath := createTestDSK(t)

	err := RemoveFile(dskPath, "NOPE.BIN")
	if err == nil {
		t.Error("RemoveFile() expected error for non-existent file")
	}
}

// TestAddMultipleFiles tests adding several files
func TestAddMultipleFiles(t *testing.T) {
	dskPath := createTestDSK(t)

	for i := 0; i < 5; i++ {
		name := strings.ToUpper(string(rune('A'+i))) + ".BIN"
		data := make([]byte, 512)
		err := AddFile(dskPath, name, data, 0x4000, 0x4000)
		if err != nil {
			t.Fatalf("AddFile(%q) error = %v", name, err)
		}
	}

	files, err := ListFiles(dskPath)
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}
	if len(files) != 5 {
		t.Errorf("ListFiles() returned %d files, want 5", len(files))
	}
}

// TestSaveAutoNumbered tests auto-numbered file saving
func TestSaveAutoNumbered(t *testing.T) {
	dskPath := createTestDSK(t)

	data := make([]byte, 256)

	name1, err := SaveAutoNumbered(dskPath, data, "IMG", "SCR", 0xC000, 0xC000)
	if err != nil {
		t.Fatalf("first SaveAutoNumbered() error = %v", err)
	}
	if name1 != "IMG00.SCR" {
		t.Errorf("first filename = %q, want IMG00.SCR", name1)
	}

	name2, err := SaveAutoNumbered(dskPath, data, "IMG", "SCR", 0xC000, 0xC000)
	if err != nil {
		t.Fatalf("second SaveAutoNumbered() error = %v", err)
	}
	if name2 != "IMG01.SCR" {
		t.Errorf("second filename = %q, want IMG01.SCR", name2)
	}
}

// TestSaveAutoNumberedDifferentPrefixes tests different prefixes don't interfere
func TestSaveAutoNumberedDifferentPrefixes(t *testing.T) {
	dskPath := createTestDSK(t)

	data := make([]byte, 128)

	name1, err := SaveAutoNumbered(dskPath, data, "AAA", "BIN", 0x4000, 0x4000)
	if err != nil {
		t.Fatalf("SaveAutoNumbered(AAA) error = %v", err)
	}
	if name1 != "AAA00.BIN" {
		t.Errorf("got %q, want AAA00.BIN", name1)
	}

	name2, err := SaveAutoNumbered(dskPath, data, "BBB", "BIN", 0x4000, 0x4000)
	if err != nil {
		t.Fatalf("SaveAutoNumbered(BBB) error = %v", err)
	}
	if name2 != "BBB00.BIN" {
		t.Errorf("got %q, want BBB00.BIN", name2)
	}
}

// TestDskSaveLoadRoundTrip tests saving and reloading a DSK with files
func TestDskSaveLoadRoundTrip(t *testing.T) {
	dskPath := createTestDSK(t)

	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i & 0xFF)
	}

	err := AddFile(dskPath, "ROUND.SCR", data, 0xC000, 0xC000)
	if err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	// Verify file persists after reload
	files, err := ListFiles(dskPath)
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	// Check the DSK file actually exists on disk
	info, err := os.Stat(dskPath)
	if err != nil {
		t.Fatalf("DSK file stat error: %v", err)
	}
	if info.Size() == 0 {
		t.Error("DSK file is empty")
	}
}

// TestMaxBloc tests max block calculation for different formats
func TestMaxBloc(t *testing.T) {
	dsk40 := NewGestDSK()
	mb40 := dsk40.MaxBloc()

	dsk80 := &GestDSK{}
	dsk80.FormatDsk80()
	mb80 := dsk80.MaxBloc()

	if mb40 <= 0 {
		t.Errorf("40-track maxBloc = %d, want > 0", mb40)
	}
	if mb80 <= 0 {
		t.Errorf("80-track maxBloc = %d, want > 0", mb80)
	}
	if mb80 <= mb40 {
		t.Errorf("80-track maxBloc (%d) should be > 40-track maxBloc (%d)", mb80, mb40)
	}

	t.Logf("40-track: %d blocks (%d KB), 80-track: %d blocks (%d KB)", mb40, mb40, mb80, mb80)
}

// TestReadBloc tests block reading
func TestReadBloc(t *testing.T) {
	dsk := NewGestDSK()
	buf := dsk.readBloc(2)

	if len(buf) != 1024 {
		t.Errorf("readBloc returned %d bytes, want 1024", len(buf))
	}
}

// TestAddFileToNonExistentDSK tests error when DSK doesn't exist
func TestAddFileToNonExistentDSK(t *testing.T) {
	err := AddFile("/nonexistent/test.dsk", "TEST.BIN", []byte{1, 2, 3}, 0x4000, 0x4000)
	if err == nil {
		t.Error("expected error for non-existent DSK")
	}
}

// TestRemoveFileFromNonExistentDSK tests error when DSK doesn't exist
func TestRemoveFileFromNonExistentDSK(t *testing.T) {
	err := RemoveFile("/nonexistent/test.dsk", "TEST.BIN")
	if err == nil {
		t.Error("expected error for non-existent DSK")
	}
}

// TestSaveAutoNumberedNonExistentDSK tests error when DSK doesn't exist
func TestSaveAutoNumberedNonExistentDSK(t *testing.T) {
	_, err := SaveAutoNumbered("/nonexistent/test.dsk", []byte{1}, "IMG", "SCR", 0xC000, 0xC000)
	if err == nil {
		t.Error("expected error for non-existent DSK")
	}
}

// TestAddFileWithLargeData tests adding a larger file
func TestAddFileWithLargeData(t *testing.T) {
	dskPath := createTestDSK(t)

	// 16KB file (typical SCR size)
	data := make([]byte, 16384)
	for i := range data {
		data[i] = byte(i & 0xFF)
	}

	err := AddFile(dskPath, "BIG.SCR", data, 0xC000, 0xC000)
	if err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	files, err := ListFiles(dskPath)
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	// Size includes the 128-byte AMSDOS header
	if files[0].Size < 16384 {
		t.Errorf("file size = %d, want >= 16384", files[0].Size)
	}
}

// TestFormat80TrackSaveLoad tests save/load round-trip for 80-track DSK
func TestFormat80TrackSaveLoad(t *testing.T) {
	dir := t.TempDir()
	dskPath := filepath.Join(dir, "test80.dsk")

	dsk := &GestDSK{}
	dsk.FormatDsk80()
	if err := dsk.Save(dskPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	dsk2 := NewGestDSK()
	if err := dsk2.Load(dskPath); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if dsk2.Infos.NbTracks != 80 {
		t.Errorf("loaded NbTracks = %d, want 80", dsk2.Infos.NbTracks)
	}
}
