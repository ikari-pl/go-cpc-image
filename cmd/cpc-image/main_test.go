// Package main provides tests for the cpc-image command-line tool
// (cmd/cpc-image/main.go).
package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ikari-pl/go-cpc-image/pkg/compress"
	"github.com/ikari-pl/go-cpc-image/pkg/cpc"
	"github.com/ikari-pl/go-cpc-image/pkg/fileio"
)

// mkRawSCR creates a synthetic CPC screen dump with a distinctive pixel
// pattern in the pixel regions of the given bitmap size.
func mkRawSCR(size int) []byte {
	raw := make([]byte, size)
	for x := 0; x < cpc.StandardCols; x++ {
		for y := 0; y < cpc.StandardLines; y++ {
			adr := x + cpc.CpcAddress(y<<1, cpc.StandardCols, cpc.StandardLines)
			if adr < size {
				raw[adr] = byte((x*13 + y*7) & 0xFF)
			}
		}
	}
	return raw
}

// writeSCR writes a standard-mode CPC screen dump (with AMSDOS header) to path
// via fileio.SaveSCR, the same way the real convert flow writes .scr files.
func writeSCR(path string, size int) error {
	params := fileio.SCRParams{WithPalette: true, WithCode: true, CPCPlus: false, VirtualMode: 1}
	_, err := fileio.SaveSCR(path, mkRawSCR(size), size,
		fileio.PackNone, fileio.OutputBinary, params,
		[]uint16{1, 24, 20, 6, 26, 0, 2, 7, 10, 12, 14, 16, 18, 22, 1, 14}, nil)
	return err
}

// TestCLIPackRejectsCompressedInput verifies the
// `pack` command refuses to re-compress already-compressed data:
//   - content signatures — PKS payload ("PK…") and OCP payload ("MJH"), with
//     and without an AMSDOS wrapper;
//   - compressed-looking extensions (.zx0/.zx1/.lzw/.pks/.cmp) are rejected
//     unconditionally, even with a valid AMSDOS screen header.
func TestCLIPackRejectsCompressedInput(t *testing.T) {
	tempDir := t.TempDir()

	// --- PKS signature via a real PackPKS payload (neutral extension). ---
	raw := mkRawSCR(cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines))
	pksBuf := make([]byte, len(raw)*2+1024)
	packedSize, perr := compress.NewCompressor().PKS().PackPKS(raw, len(raw), pksBuf, compress.PKSL, nil)
	if perr != nil {
		t.Fatalf("PackPKS failed: %v", perr)
	}

	// Raw PKS payload (no AMSDOS header).
	pksRaw := filepath.Join(tempDir, "screen.bin")
	if err := os.WriteFile(pksRaw, pksBuf[:packedSize], 0644); err != nil {
		t.Fatalf("write PKS fixture failed: %v", err)
	}
	rootCmd.SetArgs([]string{"pack", "-i", pksRaw, "-o", filepath.Join(tempDir, "out.bin"), "--method", "zx0"})
	if err := rootCmd.Execute(); err == nil {
		t.Error("pack on a PKS payload (no AMSDOS) should fail by signature")
	}

	// PKS payload wrapped in a valid AMSDOS header (the real on-disk format:
	// PKS files also carry AMSDOS 0xC000 — signature checks must run first).
	entete := cpc.CreeEntete("screen.pks", 0xC000, uint16(packedSize), 0xC7D0)
	entete.Length = 0
	entete.CheckSum = uint16(cpc.CalcCheckSumStruct(entete))
	headerBytes, hdrErr := cpc.AmsdosToByte(entete)
	if hdrErr != nil {
		t.Fatalf("build AMSDOS header failed: %v", hdrErr)
	}
	pksAms := make([]byte, len(headerBytes)+packedSize)
	copy(pksAms, headerBytes)
	copy(pksAms[len(headerBytes):], pksBuf[:packedSize])
	pksAmsPath := filepath.Join(tempDir, "screen2.bin")
	if err := os.WriteFile(pksAmsPath, pksAms, 0644); err != nil {
		t.Fatalf("write PKS+AMSDOS fixture failed: %v", err)
	}
	rootCmd.SetArgs([]string{"pack", "-i", pksAmsPath, "-o", filepath.Join(tempDir, "out.bin"), "--method", "zx0"})
	if err := rootCmd.Execute(); err == nil {
		t.Error("pack on a PKS payload with a valid AMSDOS header should fail by signature")
	}

	// --- OCP signature "MJH" (neutral extension). ---
	ocpPath := filepath.Join(tempDir, "raw.bin")
	if err := os.WriteFile(ocpPath, []byte{'M', 'J', 'H', 0, 1, 2, 3}, 0644); err != nil {
		t.Fatalf("write OCP fixture failed: %v", err)
	}
	rootCmd.SetArgs([]string{"pack", "-i", ocpPath, "-o", filepath.Join(tempDir, "out.bin"), "--method", "zx0"})
	if err := rootCmd.Execute(); err == nil {
		t.Error("pack on an OCP payload should fail by signature")
	}

	// --- Extension blocklist (LZW/ZX0/ZX1/PKS/CMP have no reliable magic). ---
	for _, ext := range []string{".zx0", ".zx1", ".lzw", ".pks", ".cmp"} {
		fake := filepath.Join(tempDir, "x"+ext)
		if werr := os.WriteFile(fake, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 0644); werr != nil {
			t.Fatalf("write fake %s failed: %v", ext, werr)
		}
		rootCmd.SetArgs([]string{"pack", "-i", fake, "-o", filepath.Join(tempDir, "out.bin"), "--method", "zx0"})
		if err := rootCmd.Execute(); err == nil {
			t.Errorf("pack on %s should fail via extension heuristic", ext)
		}
	}

	// A compressed-looking extension must be rejected even when the file
	// carries a valid AMSDOS screen header (e.g. a .CMP repacked with a header).
	screenSize := cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines)
	amdosPath := filepath.Join(tempDir, "withamdos.cmp")
	if werr := writeSCR(amdosPath, screenSize); werr != nil {
		t.Fatalf("SaveSCR failed: %v", werr)
	}
	rootCmd.SetArgs([]string{"pack", "-i", amdosPath, "-o", filepath.Join(tempDir, "out.bin"), "--method", "zx0"})
	if err := rootCmd.Execute(); err == nil {
		t.Error("pack on .cmp with a valid AMSDOS screen header should still fail by extension")
	}
}

// TestCLIPackAcceptsLegitSCR verifies the AMSDOS-screen carve-out: a real SCR
// (valid AMSDOS screen header + non-blocklisted extension) still packs fine.
func TestCLIPackAcceptsLegitSCR(t *testing.T) {
	tempDir := t.TempDir()
	screenSize := cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines)
	scrPath := filepath.Join(tempDir, "screen.scr")
	if werr := writeSCR(scrPath, screenSize); werr != nil {
		t.Fatalf("SaveSCR failed: %v", werr)
	}
	outPath := filepath.Join(tempDir, "screen.zx0")
	rootCmd.SetArgs([]string{"pack", "-i", scrPath, "-o", outPath, "--method", "zx0"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pack on a legit SCR should succeed: %v", err)
	}
	if _, oerr := os.Stat(outPath); oerr != nil {
		t.Errorf("pack output file was not created: %v", oerr)
	}
}