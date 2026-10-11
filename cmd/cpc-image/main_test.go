// Package main provides tests for the cpc-image command-line tool
// (cmd/cpc-image/main.go).
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ikari-pl/go-cpc-image/internal/version"
	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
	"github.com/ikari-pl/go-cpc-image/pkg/compress"
	"github.com/ikari-pl/go-cpc-image/pkg/convert"
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

// syntheticSCR builds a CPC screen dump of the given geometry and mode: every
// pixel byte carries a pattern and a visible ModePal block sits in the slot the
// geometry uses (&17D0, or &600 for overscan). The addressing follows the
// geometry itself, so an overscan dump spreads its lines past 255 over the
// second half of the payload.
func syntheticSCR(numCol, numLig, mode int) []byte {
	raw := make([]byte, cpc.BitmapSize(numCol, numLig))
	for x := 0; x < numCol; x++ {
		for y := 0; y < numLig; y++ {
			adr := x + cpc.CpcAddress(y<<1, numCol, numLig)
			if adr < len(raw) {
				raw[adr] = byte((x*13 + y*7) & 0xFF)
			}
		}
	}

	offset := cpc.ModePalOffset
	if numCol == cpc.OverscanCols {
		offset = cpc.OverscanModePalOffset
	}
	pal := make([]byte, 17)
	pal[0] = byte(mode)
	for i := 1; i < 17; i++ {
		pal[i] = byte(i) // 16 distinct inks: every pen the mode uses is visible
	}
	return cpc.EmbedModePalAt(raw, offset, pal)
}

// amsdosWrap prepends an AMSDOS header loading at addr, the way a saved screen
// carries one.
func amsdosWrap(t *testing.T, payload []byte, addr uint16) []byte {
	t.Helper()
	hdr, err := cpc.AmsdosToByte(cpc.CreeEntete("screen.scr", addr, uint16(len(payload)), 0xC7D0))
	if err != nil {
		t.Fatalf("AmsdosToByte failed: %v", err)
	}
	return append(hdr, payload...)
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

// TestCLIPackStripsAmsdosHeaderForEveryMethod verifies that every pack method
// compresses the screen payload only: for a screen file with an AMSDOS header
// the output must be byte-for-byte the stream of the bare payload, so that a
// decompressor writing back at the load address gets the picture and not 128
// header bytes followed by a shifted image. lzw is also unpacked again to check
// the CLI round-trip.
func TestCLIPackStripsAmsdosHeaderForEveryMethod(t *testing.T) {
	tempDir := t.TempDir()
	payload := mkRawSCR(cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines))
	inPath := filepath.Join(tempDir, "headed.scr")
	header, herr := cpc.AmsdosToByte(cpc.CreeEntete("screen.scr", 0xC000, uint16(len(payload)), 0xC7D0))
	if herr != nil {
		t.Fatalf("AmsdosToByte failed: %v", herr)
	}
	if werr := os.WriteFile(inPath, append(append([]byte{}, header...), payload...), 0644); werr != nil {
		t.Fatalf("write fixture failed: %v", werr)
	}

	for _, tc := range []struct {
		method string
		pm     compress.PackMethod
	}{
		{"zx0", compress.MethodZX0},
		{"zx0v2", compress.MethodZX0V2},
		{"zx1", compress.MethodZX1},
		{"lzw", compress.Standard},
	} {
		t.Run(tc.method, func(t *testing.T) {
			outPath := filepath.Join(tempDir, "headed_"+tc.method+".bin")
			rootCmd.SetArgs([]string{"pack", "-i", inPath, "-o", outPath, "--method", tc.method})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("pack failed: %v", err)
			}

			// Reference: the same compressor fed with the bare payload.
			want := make([]byte, len(payload)*2+1024)
			wn, werr := compress.NewCompressor().Pack(payload, len(payload), want, 0, tc.pm)
			if werr != nil {
				t.Fatalf("reference Pack failed: %v", werr)
			}
			got := mustRead(t, outPath)
			if !bytes.Equal(got, want[:wn]) {
				t.Errorf("pack output is %d bytes, want the %d-byte stream of the bare payload (the AMSDOS header must not be compressed)",
					len(got), wn)
			}
		})
	}

	// lzw is the method unpack understands, so its round-trip can be checked end
	// to end: the restored stream must be the payload, never header + payload.
	packedPath := filepath.Join(tempDir, "headed_lzw.bin")
	rootCmd.SetArgs([]string{"pack", "-i", inPath, "-o", packedPath, "--method", "lzw"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pack (lzw) failed: %v", err)
	}
	restoredPath := filepath.Join(tempDir, "headed_restored.bin")
	rootCmd.SetArgs([]string{"unpack", "-i", packedPath, "-o", restoredPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unpack failed: %v", err)
	}
	if restored := mustRead(t, restoredPath); !bytes.Equal(restored, payload) {
		t.Errorf("unpacked stream is %d bytes, want the %d-byte payload without the header",
			len(restored), len(payload))
	}
}

// TestCLIPackAmsdosLeniency pins the two edges of the header detection: a screen
// whose AMSDOS checksum does not validate is still recognised from its fields
// (compressing that header would shift the whole picture), while a file that is
// neither checksum-valid nor screen shaped is compressed as it is, header
// included.
func TestCLIPackAmsdosLeniency(t *testing.T) {
	tempDir := t.TempDir()
	payload := mkRawSCR(cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines))

	headerBytes, herr := cpc.AmsdosToByte(cpc.CreeEntete("screen.scr", 0xC000, uint16(len(payload)), 0xC7D0))
	if herr != nil {
		t.Fatalf("AmsdosToByte failed: %v", herr)
	}
	badHeader := append([]byte{}, headerBytes...)
	badHeader[0x43] ^= 0xFF // corrupt the stored checksum only
	if cpc.CheckAmsdos(badHeader) {
		t.Fatal("fixture corruption did not invalidate the checksum")
	}

	// (1) Corrupted checksum, screen-shaped fields: still stripped.
	badScreen := filepath.Join(tempDir, "badsum.scr")
	if werr := os.WriteFile(badScreen, append(append([]byte{}, badHeader...), payload...), 0644); werr != nil {
		t.Fatalf("write fixture failed: %v", werr)
	}
	wantStripped := make([]byte, len(payload)*2+1024)
	wn, werr := compress.NewCompressor().Pack(payload, len(payload), wantStripped, 0, compress.MethodZX0)
	if werr != nil {
		t.Fatalf("reference Pack failed: %v", werr)
	}
	badOut := filepath.Join(tempDir, "badsum.bin")
	rootCmd.SetArgs([]string{"pack", "-i", badScreen, "-o", badOut, "--method", "zx0"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pack failed: %v", err)
	}
	if got := mustRead(t, badOut); !bytes.Equal(got, wantStripped[:wn]) {
		t.Error("a screen with a non-standard checksum was not recognised as a screen")
	}

	// (2) Corrupted checksum and no screen shape (a 4096-byte payload): the
	// header stays part of the compressed data.
	binPayload := make([]byte, 4096)
	binHeader, berr := cpc.AmsdosToByte(cpc.CreeEntete("program.bin", 0xC000, uint16(len(binPayload)), 0xC000))
	if berr != nil {
		t.Fatalf("AmsdosToByte failed: %v", berr)
	}
	binHeader[0x43] ^= 0xFF
	if cpc.CheckAmsdos(binHeader) {
		t.Fatal("fixture corruption did not invalidate the checksum")
	}
	binary := append(append([]byte{}, binHeader...), binPayload...)
	binPath := filepath.Join(tempDir, "program.bin")
	if werr := os.WriteFile(binPath, binary, 0644); werr != nil {
		t.Fatalf("write fixture failed: %v", werr)
	}
	wantWhole := make([]byte, len(binary)*2+1024)
	wwn, wwerr := compress.NewCompressor().Pack(binary, len(binary), wantWhole, 0, compress.MethodZX0)
	if wwerr != nil {
		t.Fatalf("reference Pack (binary) failed: %v", wwerr)
	}
	binOut := filepath.Join(tempDir, "program.zx0")
	rootCmd.SetArgs([]string{"pack", "-i", binPath, "-o", binOut, "--method", "zx0"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pack (binary) failed: %v", err)
	}
	if got := mustRead(t, binOut); !bytes.Equal(got, wantWhole[:wwn]) {
		t.Error("a file that is not a screen must keep its AMSDOS header inside the compressed data")
	}
}

// writeTestPNG creates a small 160x200 PNG with a white rectangle on a black
// background — a copyright-safe fixture generated at runtime.
func writeTestPNG(t *testing.T, dir string, name string) string {
	img := image.NewRGBA(image.Rect(0, 0, 160, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 160; x++ {
			if x >= 40 && x < 120 && y >= 40 && y < 160 {
				img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
			} else {
				img.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 255})
			}
		}
	}
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("Failed to encode PNG: %v", err)
	}
	f.Close()
	return path
}

// TestCLIUnpackRejectsRawSCR verifies `unpack` refuses to decompress a raw
// (already-uncompressed) SCR file in the auto-detect path: it fails with a
// non-zero exit and never writes output garbage.
func TestCLIUnpackRejectsRawSCR(t *testing.T) {
	tempDir := t.TempDir()
	pngPath := writeTestPNG(t, tempDir, "src.png")

	scrPath := filepath.Join(tempDir, "raw.scr")
	rootCmd.SetArgs([]string{"convert", "-i", pngPath, "-o", scrPath, "-f", "scr", "-m", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert PNG -> SCR failed: %v", err)
	}

	// Auto-detected unpack of a raw SCR must fail (non-zero exit).
	rootCmd.SetArgs([]string{"unpack", "-i", scrPath, "-o", filepath.Join(tempDir, "out.bin")})
	if err := rootCmd.Execute(); err == nil {
		t.Error("unpack on a raw SCR should fail with error")
	}

	// A compressed-looking extension must also be rejected in auto mode even
	// if, by coincidence, the bytes look SCR-like.
	scrRenamed := filepath.Join(tempDir, "raw.lzw")
	if err := os.WriteFile(scrRenamed, mustRead(t, scrPath), 0644); err != nil {
		t.Fatalf("write renamed SCR failed: %v", err)
	}
	rootCmd.SetArgs([]string{"unpack", "-i", scrRenamed, "-o", filepath.Join(tempDir, "out2.bin")})
	if err := rootCmd.Execute(); err == nil {
		t.Error("unpack on a renamed (compressed-looking extension) SCR should fail")
	}

	// NOTE: with an explicit --method lzw/zx0/zx1 the CLI deliberately "keeps
	// trying": it only refuses when the decompressor echoes the input back
	// byte-for-byte (trivial round-trip), which is a rare best-effort guard.
	// The strong raw-input rejection lives in the auto-detect path above.
}

// TestCLIPackAndUnpackPKS verifies the PKS family round-trip through the CLI.
// Every variant must pack into AMSDOS + "PKxx" (auto-detected again on unpack)
// and restore the original screen: for PKSL the pixel data is compared in
// column-major order, because the restored dump has its layout gaps zeroed and
// only the ModePal re-embedded; the other variants carry the screen dump as-is
// and must match byte-for-byte.
func TestCLIPackAndUnpackPKS(t *testing.T) {
	tempDir := t.TempDir()

	// Synthetic standard screen dump with a valid ModePal at 0x17D0.
	testData := mkRawSCR(cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines))
	modePal := []byte{1, 0, 1, 2, 3, 6, 9, 10, 11, 12, 15, 18, 19, 20, 24, 25, 26}
	testData = cpc.EmbedModePal(testData, modePal)

	inputPath := filepath.Join(tempDir, "screen.bin")
	if err := os.WriteFile(inputPath, testData, 0644); err != nil {
		t.Fatalf("write input failed: %v", err)
	}

	methods := []struct {
		name    string
		method  string
		variant compress.PKSVariant
		isPKSL  bool
	}{
		{"PKS_Default", "pks", compress.PKSL, true},
		{"PKSL_Standard", "pksl", compress.PKSL, true},
		{"PKS3_Mode3", "pks3", compress.PKS3, false},
		{"PKSP_Plus", "pksp", compress.PKSP, false},
		{"PKVL_Overscan", "pkvl", compress.PKVL, false},
		{"PKVP_OverscanPlus", "pkvp", compress.PKVP, false},
	}

	for _, tc := range methods {
		t.Run(tc.name, func(t *testing.T) {
			packedPath := filepath.Join(tempDir, tc.name+".pks")
			unpackedPath := filepath.Join(tempDir, tc.name+"_restored.bin")

			rootCmd.SetArgs([]string{"pack", "-i", inputPath, "-o", packedPath, "--method", tc.method})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("pack failed for method %s: %v", tc.method, err)
			}

			packedData := mustRead(t, packedPath)
			if !cpc.CheckAmsdos(packedData) {
				t.Fatal("packed file is missing a valid AMSDOS header")
			}
			if len(packedData) < 132 {
				t.Fatalf("packed file too short: %d bytes", len(packedData))
			}
			header, err := compress.ParsePKSHeader(packedData[128:])
			if err != nil {
				t.Fatalf("ParsePKSHeader failed: %v", err)
			}
			if header.Variant != tc.variant {
				t.Errorf("header variant = %v, want %v", header.Variant, tc.variant)
			}
			if tc.isPKSL {
				if header.DataOffset != 21 {
					t.Errorf("PKSL payload offset = %d, want 21", header.DataOffset)
				}
				if header.Palette[0] != modePal[0] {
					t.Errorf("PKSL mode = %d, want %d", header.Palette[0], modePal[0])
				}
				if !bytes.Equal(header.Palette[1:17], modePal[1:17]) {
					t.Errorf("PKSL inks mismatch: got %v want %v", header.Palette[1:17], modePal[1:17])
				}
			}

			// Auto-detect mode: no --method, the "PK" signature must suffice.
			rootCmd.SetArgs([]string{"unpack", "-i", packedPath, "-o", unpackedPath})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("unpack failed for %s: %v", packedPath, err)
			}

			restored := mustRead(t, unpackedPath)
			if tc.isPKSL {
				if !bytes.Equal(cpc.ScreenToColumnMajor(restored, cpc.StandardCols, cpc.StandardLines),
					cpc.ScreenToColumnMajor(testData, cpc.StandardCols, cpc.StandardLines)) {
					t.Error("PKSL pixel data mismatch after round-trip")
				}
			} else if !bytes.Equal(restored, testData) {
				t.Errorf("restored data differs from the original (got %d bytes, want %d)",
					len(restored), len(testData))
			}
		})
	}
}

// TestCLIPackPaletteFlag verifies the `pack --palette <file.pal>` support: the
// external palette overrides the ModePal embedded in the input screen, is stored
// in the PKSL header, survives the unpack round-trip, and is ignored (with a
// warning) by the variants that carry no palette.
func TestCLIPackPaletteFlag(t *testing.T) {
	tempDir := t.TempDir()

	// Input screen carrying its own ModePal (mode 1 + inks).
	embedded := []byte{1, 0, 1, 2, 3, 6, 9, 10, 11, 12, 15, 18, 19, 20, 24, 25, 26}
	testData := cpc.EmbedModePal(mkRawSCR(cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines)), embedded)
	inputPath := filepath.Join(tempDir, "screen.bin")
	if err := os.WriteFile(inputPath, testData, 0644); err != nil {
		t.Fatalf("write input failed: %v", err)
	}

	// External palette file, shaped like a real .pal (payload = mode + 16 inks).
	extPal := []uint16{0, 24, 20, 6, 26, 0, 2, 7, 10, 12, 14, 16, 18, 22, 1, 14}
	palPath := filepath.Join(tempDir, "external.pal")
	if err := fileio.SavePalette(palPath, extPal, fileio.SCRParams{VirtualMode: 1}); err != nil {
		t.Fatalf("SavePalette failed: %v", err)
	}
	want := make([]byte, 17)
	want[0] = 1 // mode read back from the .pal payload
	for i, ink := range extPal {
		want[1+i] = byte(ink)
	}

	packHeader := func(t *testing.T, name string, args ...string) *compress.PKSHeader {
		t.Helper()
		out := filepath.Join(tempDir, name+".pks")
		cmdArgs := append([]string{"pack", "-i", inputPath, "-o", out, "--method", "pksl"}, args...)
		rootCmd.SetArgs(cmdArgs)
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("pack %s failed: %v", name, err)
		}
		packed := mustRead(t, out)
		if !cpc.CheckAmsdos(packed) {
			t.Fatalf("packed file %s has no AMSDOS header", name)
		}
		header, err := compress.ParsePKSHeader(packed[128:])
		if err != nil {
			t.Fatalf("ParsePKSHeader failed: %v", err)
		}
		return header
	}

	// Warnings are written to the command's stderr, and `--palette` binds a
	// process-global variable, so clear it before runs that must not see the
	// value left behind by an earlier Execute() in this test.
	var stderr bytes.Buffer
	rootCmd.SetErr(&stderr)
	defer rootCmd.SetErr(os.Stderr)
	resetPalette := func(t *testing.T) {
		t.Helper()
		if err := packCmd.Flags().Set("palette", ""); err != nil {
			t.Fatalf("reset --palette failed: %v", err)
		}
	}
	resetPalette(t)

	// Without --palette the ModePal embedded in the screen is used.
	if got := packHeader(t, "embedded").Palette; !bytes.Equal(got[:], embedded) {
		t.Errorf("embedded ModePal = % x, want % x", got, embedded)
	}

	// With --palette the external file wins.
	if got := packHeader(t, "external", "--palette", palPath).Palette; !bytes.Equal(got[:], want) {
		t.Errorf("--palette ModePal = % x, want % x", got, want)
	}

	// ...and it is re-embedded when the packed screen is unpacked again.
	restoredPath := filepath.Join(tempDir, "external_restored.scr")
	rootCmd.SetArgs([]string{"unpack", "-i", filepath.Join(tempDir, "external.pks"), "-o", restoredPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unpack failed: %v", err)
	}
	restored := mustRead(t, restoredPath)
	if !bytes.Equal(restored[0x17D0:0x17D0+17], want) {
		t.Errorf("restored ModePal = % x, want % x", restored[0x17D0:0x17D0+17], want)
	}

	// Variants without a palette accept the flag but ignore it (with a warning).
	stderr.Reset()
	pkvlPath := filepath.Join(tempDir, "ignored.pks")
	rootCmd.SetArgs([]string{"pack", "-i", inputPath, "-o", pkvlPath, "--method", "pkvl", "--palette", palPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pack pkvl with --palette should still succeed: %v", err)
	}
	if !strings.Contains(stderr.String(), "carries no palette") {
		t.Errorf("expected a '--palette ignored' warning, got: %q", stderr.String())
	}
	if h, err := compress.ParsePKSHeader(mustRead(t, pkvlPath)[128:]); err != nil {
		t.Fatalf("ParsePKSHeader failed: %v", err)
	} else if h.DataOffset != 4 {
		t.Errorf("PKVL payload offset = %d, want 4 (no palette)", h.DataOffset)
	}

	// A screen whose ModePal block is empty has no usable palette: warn and
	// leave the header palette empty instead of silently claiming black pens.
	noPalPath := filepath.Join(tempDir, "nopal.bin")
	if err := os.WriteFile(noPalPath, mkRawSCR(cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines)), 0644); err != nil {
		t.Fatalf("write palette-less input failed: %v", err)
	}
	stderr.Reset()
	resetPalette(t)
	noPalPKS := filepath.Join(tempDir, "nopal.pks")
	rootCmd.SetArgs([]string{"pack", "-i", noPalPath, "-o", noPalPKS, "--method", "pksl"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pack without a usable palette should still succeed: %v", err)
	}
	if !strings.Contains(stderr.String(), "no usable ModePal") {
		t.Errorf("expected a 'no usable ModePal' warning, got: %q", stderr.String())
	}
	h, err := compress.ParsePKSHeader(mustRead(t, noPalPKS)[128:])
	if err != nil {
		t.Fatalf("ParsePKSHeader failed: %v", err)
	}
	if anyNonZero(h.Palette[:]) {
		t.Errorf("header palette = % x, want an empty palette", h.Palette)
	}
}

// mustRead is a tiny helper for tests.
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return b
}

// TestCLIUnpackRejectsUnsupportedData verifies the auto-detect path does not
// fall silently through to LZW for an unknown file: a small blob that is not
// a PKS ("PK"), not OCP ("MJH"), not SCR-sized, and not LZW-valid fails with
// a non-zero exit instead of emitting a corrupt file.
func TestCLIUnpackRejectsUnsupportedData(t *testing.T) {
	tempDir := t.TempDir()

	// A tiny, clearly-not-CPC blob (not a screen size, not a signature).
	blob := filepath.Join(tempDir, "unknown.bin")
	if err := os.WriteFile(blob, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, 0644); err != nil {
		t.Fatalf("write blob failed: %v", err)
	}

	rootCmd.SetArgs([]string{"unpack", "-i", blob, "-o", filepath.Join(tempDir, "out.bin")})
	if err := rootCmd.Execute(); err == nil {
		t.Error("unpack on an unsupported/unrecognized file should fail (non-zero exit)")
	}
}

// TestCLIConvertPNGToSCRSize verifies convert PNG -> SCR
// writes 128 + BitmapSize(80,200) = 16464 bytes, NOT 128 + 65536.
func TestCLIConvertPNGToSCRSize(t *testing.T) {
	tempDir := t.TempDir()
	pngPath := writeTestPNG(t, tempDir, "src.png")

	scrPath := filepath.Join(tempDir, "out.scr")
	rootCmd.SetArgs([]string{"convert", "-i", pngPath, "-o", scrPath, "-f", "scr", "-m", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert PNG -> SCR failed: %v", err)
	}

	scrData, err := os.ReadFile(scrPath)
	if err != nil {
		t.Fatalf("read SCR failed: %v", err)
	}
	wantSize := 128 + cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines)
	if len(scrData) != wantSize {
		t.Errorf("SCR size = %d, want %d (must not be %d)", len(scrData), wantSize, 128+0x10000)
	}
	if !cpc.CheckAmsdos(scrData) {
		t.Error("SCR missing valid AMSDOS header")
	}

	// Reload via LoadSCR must yield non-trivial screen data.
	screenData, _, lerr := fileio.LoadSCR(scrData)
	if lerr != nil {
		t.Fatalf("LoadSCR failed: %v", lerr)
	}
	nonzero := false
	for i := 128; i < len(screenData); i++ {
		if screenData[i] != 0 {
			nonzero = true
			break
		}
	}
	if !nonzero {
		t.Error("SCR screen data is all zeros")
	}
}

// TestResizeToCanvasScalesSmallImage verifies that a small
// source bitmap is scaled to fill the entire CPC canvas (640x400 standard)
// instead of being used at 1:1 (which left 3/4 of the screen unread).
func TestResizeToCanvasScalesSmallImage(t *testing.T) {
	src := bitmap.NewDirectBitmap(160, 200)
	// Fill with bright red (pen 1 = 0x00F → RGB 255,0,0)
	red := cpc.GetColor(1, false)
	for y := 0; y < 200; y++ {
		for x := 0; x < 160; x++ {
			src.SetPixelColor(x, y, red)
		}
	}

	params := convert.NewDefaultSettings()
	params.NumCols = cpc.StandardCols
	params.NumLines = cpc.StandardLines
	// params.Palette[0] = 0 (black) — the default pen-0

	resized := resizeToCanvas(src, params)

	// Must be 640x400 (standard CPC canvas), not 160x200.
	if resized.Width() != 640 || resized.Height() != 400 {
		t.Fatalf("resized dims = %dx%d, want 640x400", resized.Width(), resized.Height())
	}

	// Nearest-neighbour stretch: source (160,200) → output block at (4,4)-(7,7).
	// Every pixel must match the source (all red) since the whole source is red.
	sample := resized.GetPixelColor(4, 4)
	if sample != red {
		t.Errorf("pixel at (4,4) = %v, want %v (nearest-neighbour stretch of red)",
			sample, red)
	}
	sample = resized.GetPixelColor(639, 399)
	if sample != red {
		t.Errorf("pixel at (639,399) = %v, want %v (bottom-right corner)",
			sample, red)
	}

	// Pen-0 background color check: with a 160x200→640x400 stretch (4x), the
	// whole canvas is covered. With a non-matching aspect (e.g., 200x100 →
	// 640x400 = 3.2x, 4x), the background pen-0 must fill the gaps.
	bg := cpc.GetColor(0, false) // black
	// Verify a small source that doesn't perfectly tile shows background.
	small := bitmap.NewDirectBitmap(100, 100)
	small.SetPixelColor(50, 50, cpc.GetColor(1, false)) // one red pixel
	rgParams := convert.NewDefaultSettings()
	rgParams.NumCols = cpc.StandardCols
	rgParams.NumLines = cpc.StandardLines
	smallResized := resizeToCanvas(small, rgParams)
	if smallResized.GetPixelColor(0, 0) != bg {
		t.Errorf("background pixel (0,0) = %v, want pen-0 (black) for non-matching aspect",
			smallResized.GetPixelColor(0, 0))
	}
	// The red pixel at (50,50) in 100x100 maps to a block in 640x400.
	// srcX = dx*100/640, so dx ~ 50*640/100 = 320 should map back to x=50.
	mappedX := 320 * 100 / 640
	if mappedX == 50 && smallResized.GetPixelColor(320, 200) != cpc.GetColor(1, false) {
		t.Errorf("scaled red pixel at (320,200) = %v, want red",
			smallResized.GetPixelColor(320, 200))
	}
}

// writeRGBA writes an RGBA image to a PNG file in dir and returns the path.
func writeRGBA(t *testing.T, dir, name string, img *image.RGBA) string {
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create %s: %v", name, err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("Failed to encode %s: %v", name, err)
	}
	f.Close()
	return path
}

// TestCLIConvertRoundTripPixelExact verifies that a 640×400 image made of
// exact CPC colors survives PNG → SCR → PNG as a pixel-perfect round trip
// (stretch is an identity at this size, the palette is embedded and re-read,
// and -d none disables dithering).
func TestCLIConvertRoundTripPixelExact(t *testing.T) {
	tempDir := t.TempDir()

	// 4 exact CPC colors, one per 160-px vertical stripe.
	cpcColors := []bitmap.RgbColor{
		cpc.CpcRgbPalette[0],  // black
		cpc.CpcRgbPalette[6],  // bright red
		cpc.CpcRgbPalette[24], // bright yellow
		cpc.CpcRgbPalette[26], // bright white
	}
	src := image.NewRGBA(image.Rect(0, 0, 640, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			c := cpcColors[x/160]
			src.Set(x, y, color.RGBA{R: c.R, G: c.V, B: c.B, A: 255})
		}
	}
	srcPath := writeRGBA(t, tempDir, "stripe.png", src)

	// PNG → SCR (uses resizeToCanvas = identity at 640×400, -d none).
	scrPath := filepath.Join(tempDir, "stripe.scr")
	rootCmd.SetArgs([]string{"convert", "-i", srcPath, "-o", scrPath, "-f", "scr", "-m", "1", "-d", "none"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert PNG -> SCR failed: %v", err)
	}

	// SCR → PNG (uses convertSCRToPNG + applySCRPalette + rgbaToIndexed).
	backPath := filepath.Join(tempDir, "stripe_back.png")
	rootCmd.SetArgs([]string{"convert", "-i", scrPath, "-o", backPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert SCR -> PNG failed: %v", err)
	}

	// Decode the round-tripped PNG and compare dimensions + pixels.
	f, err := os.Open(backPath)
	if err != nil {
		t.Fatalf("open back PNG failed: %v", err)
	}
	defer f.Close()
	back, _, derr := image.Decode(f)
	if derr != nil {
		t.Fatalf("back PNG failed to decode: %v", derr)
	}
	// SCR → PNG emits a mode-1 screen at its own resolution, so the round trip
	// comes back at half the canvas: every second source pixel, still exact.
	if back.Bounds().Dx() != 320 || back.Bounds().Dy() != 200 {
		t.Fatalf("back PNG dims = %dx%d, want 320x200 (the screen's own resolution)",
			back.Bounds().Dx(), back.Bounds().Dy())
	}

	mismatches := 0
	for y := 0; y < 200 && mismatches < 5; y++ {
		for x := 0; x < 320 && mismatches < 5; x++ {
			c1 := color.RGBAModel.Convert(src.At(2*x, 2*y)).(color.RGBA)
			c2 := color.RGBAModel.Convert(back.At(x, y)).(color.RGBA)
			if c1.R != c2.R || c1.G != c2.G || c1.B != c2.B {
				mismatches++
				t.Errorf("pixel mismatch at (%d,%d): got (%d,%d,%d) want (%d,%d,%d)",
					x, y, c2.R, c2.G, c2.B, c1.R, c1.G, c1.B)
			}
		}
	}
	if mismatches > 0 {
		t.Fatalf("round-trip had %d mismatching pixels (first 5 shown)", mismatches)
	}
}

// TestCLIConvertRendersPackedScreens verifies that convert transparently unpacks
// a PKS-compressed screen instead of refusing it: the restored dump is exactly
// what the raw .SCR holds, so a packed screen renders identically to the screen
// it was packed from, as an indexed PNG. An unsupported variant is still an
// error, and no output is written for it.
func TestCLIConvertRendersPackedScreens(t *testing.T) {
	tempDir := t.TempDir()

	// A standard screen and its packed (PKSL) form.
	rawPath := filepath.Join(tempDir, "screen.scr")
	if err := writeSCR(rawPath, cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines)); err != nil {
		t.Fatalf("writeSCR failed: %v", err)
	}
	pksPath := filepath.Join(tempDir, "screen.pks")
	rootCmd.SetArgs([]string{"pack", "-i", rawPath, "-o", pksPath, "--method", "pksl"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pack failed: %v", err)
	}

	// Converting the packed screen must give the same PNG as the raw screen.
	packedPNG := filepath.Join(tempDir, "packed.png")
	rootCmd.SetArgs([]string{"convert", "-i", pksPath, "-o", packedPNG})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert of a packed screen failed: %v", err)
	}
	rawPNG := filepath.Join(tempDir, "raw.png")
	rootCmd.SetArgs([]string{"convert", "-i", rawPath, "-o", rawPNG})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert of the raw screen failed: %v", err)
	}
	if a, b := mustRead(t, packedPNG), mustRead(t, rawPNG); !bytes.Equal(a, b) {
		t.Error("a packed screen must render exactly like the screen it was packed from")
	}
	if paletted := decodeIndexedPNG(t, packedPNG); len(paletted.Palette) != 16 {
		t.Errorf("palette entries = %d, want 16 (an indexed PNG)", len(paletted.Palette))
	}

	// A "PK" payload naming a variant the engine does not implement is still
	// rejected, with nothing written: only real packed screens are unpacked.
	unknownPath := filepath.Join(tempDir, "unknown.scr")
	if err := os.WriteFile(unknownPath, append([]byte("PKZZ"), make([]byte, 64)...), 0644); err != nil {
		t.Fatalf("write fixture failed: %v", err)
	}
	unknownPNG := filepath.Join(tempDir, "unknown.png")
	rootCmd.SetArgs([]string{"convert", "-i", unknownPath, "-o", unknownPNG})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("convert must fail on an unsupported PKS variant")
	}
	if !strings.Contains(err.Error(), "unpack PKS") {
		t.Errorf("error should report the failed unpack, got: %v", err)
	}
	if _, serr := os.Stat(unknownPNG); serr == nil {
		t.Error("convert must not write an output file for an unsupported variant")
	}
}

// TestCLIConvertPaletteLessScreenWarns verifies that a screen whose ModePal block
// is all zero — for example a screen shipped without an embedded palette, which
// keeps it in a separate .pal file, or one unpacked from a PKS packed without
// --palette — warns on stderr and renders with the default CPC palette instead
// of collapsing to a solid block (every pen was ink 0 before the fix). A screen
// carrying a usable embedded palette must stay silent.
func TestCLIConvertPaletteLessScreenWarns(t *testing.T) {
	cases := []struct {
		name     string
		embed    []byte // nil leaves the ModePal block all zero
		wantWarn bool
	}{
		{"stripped block", nil, true},
		{"embedded palette", []byte{1, 0, 1, 2, 3, 6, 9, 10, 11, 12, 15, 18, 19, 20, 24, 25, 26}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			screen := mkRawSCR(cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines))
			if tc.embed != nil {
				screen = cpc.EmbedModePal(screen, tc.embed)
			}
			inPath := filepath.Join(tempDir, "screen.scr")
			if err := os.WriteFile(inPath, screen, 0644); err != nil {
				t.Fatalf("write screen failed: %v", err)
			}

			var stderr bytes.Buffer
			rootCmd.SetErr(&stderr)
			defer rootCmd.SetErr(os.Stderr)

			outPath := filepath.Join(tempDir, "screen.png")
			rootCmd.SetArgs([]string{"convert", "-i", inPath, "-o", outPath})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("convert SCR -> PNG failed: %v", err)
			}

			if got := strings.Contains(stderr.String(), "no usable ModePal"); got != tc.wantWarn {
				t.Errorf("warning emitted = %v, want %v (stderr: %q)", got, tc.wantWarn, stderr.String())
			}

			f, err := os.Open(outPath)
			if err != nil {
				t.Fatalf("open PNG failed: %v", err)
			}
			defer f.Close()
			img, _, derr := image.Decode(f)
			if derr != nil {
				t.Fatalf("PNG failed to decode: %v", derr)
			}

			// The rendered image must not collapse to a single colour: the
			// default palette keeps the pens distinct.
			seen := make(map[color.RGBA]bool)
			for y := 0; y < img.Bounds().Dy(); y++ {
				for x := 0; x < img.Bounds().Dx(); x++ {
					seen[color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)] = true
				}
			}
			if len(seen) < 2 {
				t.Errorf("rendered image has %d distinct colour(s); a stripped block must not render as one flat colour", len(seen))
			}
		})
	}
}

// TestCLIPackAutoDetectPKSVariant verifies that `pack --method pks` derives the
// PK* variant from the input screen instead of always writing PKSL: an overscan
// dump (AMSDOS load address 0x0200, or a payload of a full 16K bank) and a CPC
// Plus screen (0x80 bit in the mode byte at 0x17D0 / 0x600) select PKVL, PKVP and
// PKSP, mode 3 selects PKS3 only when the whole ModePal block validates, and
// anything else stays PKSL.
func TestCLIPackAutoDetectPKSVariant(t *testing.T) {
	tempDir := t.TempDir()

	// --palette binds a process-global: clear whatever an earlier Execute() in
	// this package left behind.
	if err := packCmd.Flags().Set("palette", ""); err != nil {
		t.Fatalf("reset --palette failed: %v", err)
	}

	stdSize := cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines)
	ovsSize := cpc.BitmapSize(cpc.OverscanCols, cpc.OverscanLines)

	// modeByteAt returns a raw screen dump with modeByte written at off.
	modeByteAt := func(size, off int, modeByte byte) []byte {
		bmp := mkRawSCR(size)
		if off < len(bmp) {
			bmp[off] = modeByte
		}
		return bmp
	}
	// mode3Screen carries a full, valid ModePal (mode 3 + valid ink indices).
	mode3Screen := func(size int) []byte {
		pal := make([]byte, 17)
		pal[0] = 3
		for i := 1; i < 17; i++ {
			pal[i] = byte(i)
		}
		return cpc.EmbedModePal(mkRawSCR(size), pal)
	}
	underSize := cpc.BitmapSize(cpc.UnderscanCols, cpc.UnderscanLines)
	// amsdos wraps payload in an AMSDOS header that loads at addr.
	amsdos := func(t *testing.T, payload []byte, addr uint16) []byte {
		t.Helper()
		ent := cpc.CreeEntete("screen.scr", addr, uint16(len(payload)), 0xC7D0)
		hdr, err := cpc.AmsdosToByte(ent)
		if err != nil {
			t.Fatalf("AmsdosToByte failed: %v", err)
		}
		return append(hdr, payload...)
	}

	cases := []struct {
		name string
		data func(*testing.T) []byte
		want compress.PKSVariant
	}{
		{"std classic mode 0", func(*testing.T) []byte {
			return modeByteAt(stdSize, cpc.ModePalOffset, 0x00)
		}, compress.PKSL},
		{"std classic mode 1", func(*testing.T) []byte {
			return modeByteAt(stdSize, cpc.ModePalOffset, 0x01)
		}, compress.PKSL},
		{"std mode 3 byte without valid ModePal", func(*testing.T) []byte {
			bmp := modeByteAt(stdSize, cpc.ModePalOffset, 0x03)
			bmp[cpc.ModePalOffset+1] = 0x40 // ink index > 26: the block is not a ModePal
			return bmp
		}, compress.PKSL},
		{"std mode 3 with valid ModePal", func(*testing.T) []byte {
			return mode3Screen(stdSize)
		}, compress.PKS3},
		{"underscan classic", func(*testing.T) []byte {
			return modeByteAt(underSize, cpc.ModePalOffset, 0x01)
		}, compress.PKUL},
		{"underscan Plus", func(*testing.T) []byte {
			return modeByteAt(underSize, cpc.ModePalOffset, 0x8D)
		}, compress.PKUP},
		{"underscan mode 3 with valid ModePal", func(*testing.T) []byte {
			return mode3Screen(underSize)
		}, compress.PKU3},
		{"std Plus mode 1", func(*testing.T) []byte {
			return modeByteAt(stdSize, cpc.ModePalOffset, 0x81)
		}, compress.PKSP},
		{"std Plus mode 3", func(*testing.T) []byte {
			return modeByteAt(stdSize, cpc.ModePalOffset, 0x8F)
		}, compress.PKSP},
		{"ovs classic 16K payload", func(*testing.T) []byte {
			return modeByteAt(ovsSize, 0x600, 0x01)
		}, compress.PKVL},
		{"ovs Plus 16K payload", func(*testing.T) []byte {
			return modeByteAt(ovsSize, 0x600, 0x8D)
		}, compress.PKVP},
		{"std payload with AMSDOS 0x0200", func(t *testing.T) []byte {
			return amsdos(t, modeByteAt(stdSize, 0x600, 0x01), 0x0200)
		}, compress.PKVL},
		{"std payload with AMSDOS 0x0200 Plus", func(t *testing.T) []byte {
			return amsdos(t, modeByteAt(stdSize, 0x600, 0x8D), 0x0200)
		}, compress.PKVP},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			base := strings.ReplaceAll(tc.name, " ", "_")
			inPath := filepath.Join(tempDir, base+".bin")
			if err := os.WriteFile(inPath, tc.data(t), 0644); err != nil {
				t.Fatalf("write fixture failed: %v", err)
			}

			outPath := filepath.Join(tempDir, base+".pks")
			rootCmd.SetArgs([]string{"pack", "-i", inPath, "-o", outPath, "--method", "pks"})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("pack failed: %v", err)
			}

			header, err := compress.ParsePKSHeader(mustRead(t, outPath)[128:])
			if err != nil {
				t.Fatalf("ParsePKSHeader failed: %v", err)
			}
			if header.Variant != tc.want {
				t.Errorf("auto-detected variant = %s, want %s",
					pksVariantName(header.Variant), pksVariantName(tc.want))
			}
		})
	}

	// No signal at all: a dump too short to hold the mode byte falls back to
	// PKSL and says so (verbose warning, on stderr).
	t.Run("no signal falls back to PKSL", func(t *testing.T) {
		inPath := filepath.Join(tempDir, "tiny.bin")
		if err := os.WriteFile(inPath, make([]byte, 1024), 0644); err != nil {
			t.Fatalf("write fixture failed: %v", err)
		}

		var stderr bytes.Buffer
		rootCmd.SetErr(&stderr)
		defer rootCmd.SetErr(os.Stderr)

		outPath := filepath.Join(tempDir, "tiny.pks")
		rootCmd.SetArgs([]string{"pack", "-i", inPath, "-o", outPath, "--method", "pks", "-v"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("pack failed: %v", err)
		}

		header, err := compress.ParsePKSHeader(mustRead(t, outPath)[128:])
		if err != nil {
			t.Fatalf("ParsePKSHeader failed: %v", err)
		}
		if header.Variant != compress.PKSL {
			t.Errorf("fallback variant = %s, want PKSL", pksVariantName(header.Variant))
		}
		if !strings.Contains(stderr.String(), "falling back to PKSL") {
			t.Errorf("expected a fallback warning on stderr, got: %q", stderr.String())
		}
	})
}

// TestCLIPackUnderscanVariantsRoundTrip verifies the underscan variants through
// the CLI: pack emits PKUL/PKU3/PKUP for an underscan screen (it must not fall
// back to PKSL) and unpack restores it — column-major pixels plus the ModePal
// for PKUL, the dump as it is for the others.
func TestCLIPackUnderscanVariantsRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	underscanSize := cpc.BitmapSize(cpc.UnderscanCols, cpc.UnderscanLines)

	classicPal := []byte{1, 0, 1, 2, 3, 6, 9, 10, 11, 12, 15, 18, 19, 20, 24, 25, 26}
	screen := cpc.EmbedModePal(mkRawSCR(underscanSize), classicPal)
	inPath := filepath.Join(tempDir, "underscan.bin")
	if err := os.WriteFile(inPath, screen, 0644); err != nil {
		t.Fatalf("write fixture failed: %v", err)
	}

	for _, tc := range []struct {
		name    string
		method  string
		variant compress.PKSVariant
	}{
		{"PKUL", "pkul", compress.PKUL},
		{"PKU3", "pku3", compress.PKU3},
		{"PKUP", "pkup", compress.PKUP},
	} {
		t.Run(tc.name, func(t *testing.T) {
			packedPath := filepath.Join(tempDir, tc.name+".pks")
			unpackedPath := filepath.Join(tempDir, tc.name+"_restored.bin")

			rootCmd.SetArgs([]string{"pack", "-i", inPath, "-o", packedPath, "--method", tc.method})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("pack failed: %v", err)
			}
			header, err := compress.ParsePKSHeader(mustRead(t, packedPath)[128:])
			if err != nil {
				t.Fatalf("ParsePKSHeader failed: %v", err)
			}
			if header.Variant != tc.variant {
				t.Errorf("variant = %s, want %s", pksVariantName(header.Variant), pksVariantName(tc.variant))
			}

			rootCmd.SetArgs([]string{"unpack", "-i", packedPath, "-o", unpackedPath})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("unpack failed: %v", err)
			}
			restored := mustRead(t, unpackedPath)
			if len(restored) != underscanSize {
				t.Fatalf("restored size = %d, want %d", len(restored), underscanSize)
			}

			if tc.variant == compress.PKUL {
				// Pixel data only survives, compared in the payload layout.
				if !bytes.Equal(cpc.ScreenToColumnMajor(restored, cpc.StandardCols, cpc.StandardLines),
					cpc.ScreenToColumnMajor(screen, cpc.StandardCols, cpc.StandardLines)) {
					t.Error("PKUL pixel data mismatch after round-trip")
				}
				got := restored[cpc.ModePalOffset : cpc.ModePalOffset+17]
				if !bytes.Equal(got, classicPal) {
					t.Errorf("PKUL ModePal = % x, want % x", got, classicPal)
				}
			} else if !bytes.Equal(restored, screen) {
				t.Error("restored dump differs from the original screen")
			}
		})
	}
}

// TestCLIPackUnderscanPlusRoundTrip builds an underscan CPC Plus screen with
// fileio.SaveSCR (33-byte Plus block at 0x17D0), packs it with --method pks and
// checks the PKUP header, the round-trip and the rendered Plus palette.
func TestCLIPackUnderscanPlusRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	underscanSize := cpc.BitmapSize(cpc.UnderscanCols, cpc.UnderscanLines)

	palette16 := make([]uint16, 16)
	for i := range palette16 {
		palette16[i] = uint16(i * 0x111)
	}
	inPath := filepath.Join(tempDir, "plus.scr")
	params := fileio.SCRParams{WithPalette: true, WithCode: true, CPCPlus: true, VirtualMode: 1}
	if _, err := fileio.SaveSCR(inPath, mkRawSCR(underscanSize), underscanSize,
		fileio.PackNone, fileio.OutputBinary, params, palette16, nil); err != nil {
		t.Fatalf("SaveSCR failed: %v", err)
	}
	scrData := mustRead(t, inPath)
	if scrData[0x17D0]&0x80 == 0 {
		t.Fatalf("fixture has no Plus mode byte at 0x17D0: 0x%02X", scrData[0x17D0])
	}

	packedPath := filepath.Join(tempDir, "plus.pks")
	rootCmd.SetArgs([]string{"pack", "-i", inPath, "-o", packedPath, "--method", "pks"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pack failed: %v", err)
	}
	header, err := compress.ParsePKSHeader(mustRead(t, packedPath)[128:])
	if err != nil {
		t.Fatalf("ParsePKSHeader failed: %v", err)
	}
	if header.Variant != compress.PKUP {
		t.Fatalf("auto-detected variant = %s, want PKUP", pksVariantName(header.Variant))
	}
	if !header.CpcPlus {
		t.Error("PKUP should report CpcPlus")
	}
	if header.DataOffset != 4 {
		t.Errorf("PKUP data offset = %d, want 4 (no palette field)", header.DataOffset)
	}
	if header.Palette != ([17]byte{}) {
		t.Errorf("PKUP palette field = % x, want empty", header.Palette[:])
	}

	unpackedPath := filepath.Join(tempDir, "plus_restored.bin")
	rootCmd.SetArgs([]string{"unpack", "-i", packedPath, "-o", unpackedPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unpack failed: %v", err)
	}
	restored := mustRead(t, unpackedPath)
	if !bytes.Equal(restored, scrData[128:]) {
		t.Error("PKUP round-trip is not byte-for-byte")
	}
	// PKUP has no palette field, so the 33-byte Plus block must travel inside
	// the packed payload itself.
	if restored[0x17D0]&0x80 == 0 {
		t.Errorf("restored PKUP dump lost the Plus block at 0x17D0: 0x%02X", restored[0x17D0])
	}

	// The restored screen must render with the Plus palette at 0x17D0.
	backPath := filepath.Join(tempDir, "plus.png")
	rootCmd.SetArgs([]string{"convert", "-i", unpackedPath, "-o", backPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert failed: %v", err)
	}
	f, err := os.Open(backPath)
	if err != nil {
		t.Fatalf("open PNG failed: %v", err)
	}
	defer f.Close()
	img, _, derr := image.Decode(f)
	if derr != nil {
		t.Fatalf("PNG failed to decode: %v", derr)
	}
	if got := img.Bounds(); got.Dx() != 256 || got.Dy() != 192 {
		t.Errorf("underscan Plus PNG is %dx%d, want 256x192 (64x192 at the screen's own resolution)", got.Dx(), got.Dy())
	}
	paletted, ok := img.(*image.Paletted)
	if !ok {
		t.Fatalf("underscan Plus PNG is %T, want an indexed image", img)
	}
	for pen := 0; pen < 16; pen++ {
		want := cpc.PaletteColor(int(palette16[pen]), true)
		got := color.RGBAModel.Convert(paletted.Palette[pen]).(color.RGBA)
		if int(got.R) != (want>>16)&0xFF || int(got.G) != (want>>8)&0xFF || int(got.B) != want&0xFF {
			t.Errorf("pen %d colour = #%02X%02X%02X, want #%06X", pen, got.R, got.G, got.B, want)
		}
	}
}

// corpusSamplesEnv names the environment variable that points at a directory of
// read-only reference screens — the corpus this PKS support was validated
// against. Those screens are not part of this repository, so the tests that need
// them skip unless the variable is set, e.g.
//
//	CPC_SAMPLES_DIR=/path/to/samples go test ./...
const corpusSamplesEnv = "CPC_SAMPLES_DIR"

// corpusPath returns the path of a reference screen, skipping the test when the
// corpus directory is not configured or the sample is missing.
func corpusPath(t *testing.T, rel string) string {
	t.Helper()
	root := os.Getenv(corpusSamplesEnv)
	if root == "" {
		t.Skipf("set %s to run the corpus-driven tests", corpusSamplesEnv)
	}
	p := filepath.Join(root, filepath.FromSlash(rel))
	if _, err := os.Stat(p); err != nil {
		t.Skipf("sample not available: %s", p)
	}
	return p
}

// TestCLIConvertRealPackedScreens pins the reported command: a screen taken
// straight from the corpus' pk_compressed directory must render, not be refused.
// The palette travels in the PKS header of a PKSL payload, so the output must
// also carry the screen's own pens, not a default-palette guess.
func TestCLIConvertRealPackedScreens(t *testing.T) {
	resetConvertFlags(t)
	defer resetConvertFlags(t)

	inPath := corpusPath(t, "mode0/standard_160x200/pk_compressed/3DFIGHT.SCR")
	outPath := filepath.Join(t.TempDir(), "packed.png")

	rootCmd.SetArgs([]string{"convert", "-i", inPath, "-o", outPath, "-f", "png"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert of a packed screen failed: %v", err)
	}

	paletted := decodeIndexedPNG(t, outPath)
	if got := paletted.Bounds(); got.Dx() != 320 || got.Dy() != 200 {
		t.Errorf("PNG is %dx%d, want 320x200 (a standard screen at its own resolution)", got.Dx(), got.Dy())
	}
	if len(paletted.Palette) != 16 {
		t.Fatalf("palette entries = %d, want 16", len(paletted.Palette))
	}

	// Every pen must be the colour the PKS header names, in the same order.
	raw := mustRead(t, inPath)
	if !cpc.CheckAmsdos(raw) {
		t.Fatal("packed sample has no AMSDOS header")
	}
	header, err := compress.ParsePKSHeader(raw[128:])
	if err != nil {
		t.Fatalf("ParsePKSHeader failed: %v", err)
	}
	for pen := 0; pen < 16; pen++ {
		ink := int(header.Palette[1+pen])
		want := cpc.PaletteColor(ink, false)
		got := palettedRGB(t, paletted.Palette, pen)
		if int(got.R) != (want>>16)&0xFF || int(got.G) != (want>>8)&0xFF || int(got.B) != want&0xFF {
			t.Errorf("pen %d colour = #%02X%02X%02X, want #%06X (header ink %d)",
				pen, got.R, got.G, got.B, want, ink)
		}
	}
}

// TestCLIConvertUnderscanSCRToPNG verifies that a raw underscan screen is
// rendered with the underscan geometry and with the palette embedded at
// 0x17D0, instead of the standard 80x200 layout.
func TestCLIConvertUnderscanSCRToPNG(t *testing.T) {
	inPath := corpusPath(t, "mode1/underscan_256x192/raw/BATMAN1.SCR")
	outPath := filepath.Join(t.TempDir(), "underscan.png")

	rootCmd.SetArgs([]string{"convert", "-i", inPath, "-o", outPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert SCR -> PNG failed: %v", err)
	}

	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("open PNG failed: %v", err)
	}
	defer f.Close()
	img, _, derr := image.Decode(f)
	if derr != nil {
		t.Fatalf("PNG failed to decode: %v", derr)
	}
	if got := img.Bounds(); got.Dx() != 256 || got.Dy() != 192 {
		t.Errorf("underscan PNG is %dx%d, want 256x192 (64x192 code space)", got.Dx(), got.Dy())
	}

	// The palette must be the ModePal read at 0x17D0, in pen order.
	raw := mustRead(t, inPath)
	if !cpc.CheckAmsdos(raw) {
		t.Fatal("underscan sample has no AMSDOS header")
	}
	modePal := raw[128+cpc.ModePalOffset : 128+cpc.ModePalOffset+17]
	paletted, ok := img.(*image.Paletted)
	if !ok {
		t.Fatalf("underscan PNG is %T, want an indexed image", img)
	}
	if len(paletted.Palette) != 16 {
		t.Fatalf("palette entries = %d, want 16", len(paletted.Palette))
	}
	for pen := 0; pen < 16; pen++ {
		ink := int(modePal[1+pen])
		if ink == 0xFF {
			ink = 0
		}
		want := cpc.PaletteColor(ink, false)
		got := color.RGBAModel.Convert(paletted.Palette[pen]).(color.RGBA)
		if int(got.R) != (want>>16)&0xFF || int(got.G) != (want>>8)&0xFF || int(got.B) != want&0xFF {
			t.Errorf("pen %d colour = #%02X%02X%02X, want #%06X (ink %d)",
				pen, got.R, got.G, got.B, want, ink)
		}
	}
}

// TestCLIConvertSCRToPNGGeometry pins the size every screen shape is emitted at.
// The renderer draws on a 640-dot display canvas, so the PNG comes out at the
// screen's own resolution: the canvas halved for modes 0 and 1 (320x200
// standard, 256x192 underscan, 384x272 overscan) and whole for mode 2, whose
// pixels are one canvas dot wide (640x400 / 512x384 / 768x544).
func TestCLIConvertSCRToPNGGeometry(t *testing.T) {
	resetConvertFlags(t)
	defer resetConvertFlags(t)

	for _, tc := range []struct {
		name           string
		numCol, numLig int
		mode           int
		wantW, wantH   int
	}{
		{"standard 80x200 mode 0", cpc.StandardCols, cpc.StandardLines, 0, 320, 200},
		{"underscan 64x192 mode 0", cpc.UnderscanCols, cpc.UnderscanLines, 0, 256, 192},
		{"overscan 96x272 mode 0", cpc.OverscanCols, cpc.OverscanLines, 0, 384, 272},
		{"standard 80x200 mode 1", cpc.StandardCols, cpc.StandardLines, 1, 320, 200},
		{"underscan 64x192 mode 1", cpc.UnderscanCols, cpc.UnderscanLines, 1, 256, 192},
		{"overscan 96x272 mode 1", cpc.OverscanCols, cpc.OverscanLines, 1, 384, 272},
		{"standard 80x200 mode 2", cpc.StandardCols, cpc.StandardLines, 2, 640, 400},
		{"underscan 64x192 mode 2", cpc.UnderscanCols, cpc.UnderscanLines, 2, 512, 384},
		{"overscan 96x272 mode 2", cpc.OverscanCols, cpc.OverscanLines, 2, 768, 544},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			inPath := filepath.Join(dir, "screen.scr")
			// No AMSDOS header: the payload size is the only thing that says
			// which geometry this is.
			if err := os.WriteFile(inPath, syntheticSCR(tc.numCol, tc.numLig, tc.mode), 0644); err != nil {
				t.Fatalf("write screen failed: %v", err)
			}

			outPath := filepath.Join(dir, "screen.png")
			rootCmd.SetArgs([]string{"convert", "-i", inPath, "-o", outPath})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("convert SCR -> PNG failed: %v", err)
			}

			img := decodeIndexedPNG(t, outPath)
			if got := img.Bounds(); got.Dx() != tc.wantW || got.Dy() != tc.wantH {
				t.Errorf("PNG is %dx%d, want %dx%d", got.Dx(), got.Dy(), tc.wantW, tc.wantH)
			}
			if len(img.Palette) != 16 {
				t.Errorf("palette entries = %d, want 16 (an indexed PNG in pen order)", len(img.Palette))
			}
		})
	}
}

// TestCLIConvertOverscanSCRToPNG renders a synthetic overscan dump in full: the
// lines past 255 live past the 16 KB a loader that reads only the first half of
// the payload would copy, so a blank lower half is the truncation signature. The
// AMSDOS load address (&0200) is a second, independent overscan signal.
func TestCLIConvertOverscanSCRToPNG(t *testing.T) {
	resetConvertFlags(t)
	defer resetConvertFlags(t)

	// A marker byte in the second half of the payload — line 264 addresses past
	// the 16 KB a truncated load would read — with its leftmost pixel at pen 1
	// and the rest of the byte at pen 0: rendering it proves the whole payload
	// reached the renderer, and pins where the bits land.
	const markerLine, markerCol = 264, 5

	for _, tc := range []struct {
		name         string
		mode         int
		headered     bool
		wantW, wantH int
	}{
		{"mode 0", 0, false, 384, 272},
		{"mode 1", 1, false, 384, 272},
		{"mode 2", 2, false, 768, 544},
		{"mode 0 with AMSDOS header", 0, true, 384, 272},
		{"mode 2 with AMSDOS header", 2, true, 768, 544},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			payload := syntheticSCR(cpc.OverscanCols, cpc.OverscanLines, tc.mode)
			payload[markerCol+cpc.CpcAddress(markerLine*2, cpc.OverscanCols, cpc.OverscanLines)] = 0x80
			if tc.headered {
				payload = amsdosWrap(t, payload, 0x0200)
			}
			inPath := filepath.Join(dir, "screen.scr")
			if err := os.WriteFile(inPath, payload, 0644); err != nil {
				t.Fatalf("write screen failed: %v", err)
			}

			outPath := filepath.Join(dir, "screen.png")
			rootCmd.SetArgs([]string{"convert", "-i", inPath, "-o", outPath})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("convert SCR -> PNG failed: %v", err)
			}

			img := decodeIndexedPNG(t, outPath)
			if got := img.Bounds(); got.Dx() != tc.wantW || got.Dy() != tc.wantH {
				t.Fatalf("PNG is %dx%d, want %dx%d", got.Dx(), got.Dy(), tc.wantW, tc.wantH)
			}

			// Where the marker byte's first two pixels land in the emitted grid:
			// modes 0 and 1 emit one output row per line, mode 2 two.
			firstX, secondX := markerCol*4, markerCol*4+2 // mode 0: 2 pixels per byte
			rows := []int{markerLine}
			switch tc.mode {
			case 1:
				firstX, secondX = markerCol*4, markerCol*4+1 // 4 pixels per byte
			case 2:
				firstX, secondX = markerCol*8, markerCol*8+1 // 8 pixels per byte
				rows = []int{markerLine * 2, markerLine*2 + 1}
			}
			for _, row := range rows {
				if got := img.ColorIndexAt(firstX, row); got != 1 {
					t.Errorf("marker pixel (%d,%d) = pen %d, want pen 1 (the payload past line 255 must be rendered)",
						firstX, row, got)
				}
				if got := img.ColorIndexAt(secondX, row); got != 0 {
					t.Errorf("pixel after the marker (%d,%d) = pen %d, want pen 0", secondX, row, got)
				}
			}

			pens := map[uint8]bool{}
			for y := tc.wantH / 2; y < tc.wantH; y++ {
				for x := 0; x < tc.wantW; x++ {
					pens[img.ColorIndexAt(x, y)] = true
				}
			}
			if len(pens) < 2 {
				t.Errorf("lower half holds %d pens, want the picture data (a blank half means the dump was truncated)", len(pens))
			}
		})
	}
}

// TestCLIConvertRejectsUnknownSCRSize verifies that a payload of no screen size
// is refused with a non-zero exit and no output file, instead of being read as a
// standard 80x200 screen. The sizes are exact: guessing a geometry is what used
// to turn an overscan dump into a half-empty picture.
func TestCLIConvertRejectsUnknownSCRSize(t *testing.T) {
	resetConvertFlags(t)
	defer resetConvertFlags(t)

	stdSize := cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines)

	for _, tc := range []struct {
		name string
		data func(t *testing.T) []byte
	}{
		{"standard payload one byte short", func(*testing.T) []byte {
			return syntheticSCR(cpc.StandardCols, cpc.StandardLines, 1)[:stdSize-1]
		}},
		{"standard payload one byte long", func(*testing.T) []byte {
			return append(syntheticSCR(cpc.StandardCols, cpc.StandardLines, 1), 0)
		}},
		{"truncated payload wrapped in a header", func(t *testing.T) []byte {
			return amsdosWrap(t, syntheticSCR(cpc.StandardCols, cpc.StandardLines, 1)[:stdSize-1], 0xC000)
		}},
		{"PKSL pixel payload (not a screen dump)", func(*testing.T) []byte {
			// The column-major pixel buffer a PKSL header describes: 16000
			// bytes of picture data with no screen layout and no palette.
			return cpc.ScreenToColumnMajor(
				syntheticSCR(cpc.StandardCols, cpc.StandardLines, 1),
				cpc.StandardCols, cpc.StandardLines)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			inPath := filepath.Join(dir, "screen.scr")
			if err := os.WriteFile(inPath, tc.data(t), 0644); err != nil {
				t.Fatalf("write screen failed: %v", err)
			}

			outPath := filepath.Join(dir, "screen.png")
			rootCmd.SetArgs([]string{"convert", "-i", inPath, "-o", outPath})
			err := rootCmd.Execute()
			if err == nil {
				t.Fatal("convert must reject a payload that is not a screen size")
			}
			if !strings.Contains(err.Error(), "payload size") {
				t.Errorf("error should name the payload size, got: %v", err)
			}
			if _, serr := os.Stat(outPath); serr == nil {
				t.Error("convert must not write an output file when it rejects the input")
			}
		})
	}
}

// TestCLIUnpackRealPKULSamples unpacks every underscan PKUL sample that has a
// paired raw screen and checks the restored dump: the underscan size, the pixel
// data in the payload's column-major layout and the ModePal taken from the
// header. The layout gaps are not compared: the raw screens keep their display
// code there, while a restored dump has its gaps zeroed.
func TestCLIUnpackRealPKULSamples(t *testing.T) {
	root := os.Getenv(corpusSamplesEnv)
	if root == "" {
		t.Skipf("set %s to run the corpus-driven tests", corpusSamplesEnv)
	}
	pattern := filepath.Join(root, "mode*", "underscan_*", "pk_compressed", "*.SCR")
	packed, err := filepath.Glob(pattern)
	if err != nil || len(packed) == 0 {
		t.Skipf("no underscan samples available (%s)", pattern)
	}
	sep := string(filepath.Separator)

	checked := 0
	for _, pk := range packed {
		rawPath := strings.Replace(pk,
			sep+"pk_compressed"+sep, sep+"raw"+sep, 1)
		if _, err := os.Stat(rawPath); err != nil {
			continue
		}
		name := filepath.Base(pk)

		packedData := mustRead(t, pk)
		if string(packedData[128:132]) != "PKUL" {
			t.Fatalf("%s: signature = %q, want PKUL", name, packedData[128:132])
		}
		header, herr := compress.ParsePKSHeader(packedData[128:])
		if herr != nil {
			t.Fatalf("%s: ParsePKSHeader failed: %v", name, herr)
		}
		rawData := mustRead(t, rawPath)
		rawPayload := rawData
		if len(rawData) >= 128 && cpc.CheckAmsdos(rawData) {
			rawPayload = rawData[128:]
		}

		outPath := filepath.Join(t.TempDir(), name)
		rootCmd.SetArgs([]string{"unpack", "-i", pk, "-o", outPath})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("%s: unpack failed: %v", name, err)
		}
		restored := mustRead(t, outPath)
		if wantSize := cpc.BitmapSize(cpc.UnderscanCols, cpc.UnderscanLines); len(restored) != wantSize {
			t.Fatalf("%s: restored %d bytes, want %d", name, len(restored), wantSize)
		}
		if !bytes.Equal(
			cpc.ScreenToColumnMajor(restored, cpc.StandardCols, cpc.StandardLines),
			cpc.ScreenToColumnMajor(rawPayload, cpc.StandardCols, cpc.StandardLines)) {
			t.Errorf("%s: pixel data differs from the raw screen", name)
		}
		got := restored[cpc.ModePalOffset : cpc.ModePalOffset+17]
		if !bytes.Equal(got, header.Palette[:]) {
			t.Errorf("%s: ModePal = % x, want % x (from the PKUL header)", name, got, header.Palette[:])
		}
		checked++
	}
	if checked == 0 {
		t.Skip("no PKUL/raw pairs available")
	}
	t.Logf("checked %d real PKUL samples", checked)
}

// TestCLIConvertUnderscanPlusRealSamples renders the two real underscan CPC Plus
// screens: the geometry must be the underscan one, the palette must be decoded
// from the 33-byte Plus block at 0x17D0 (with FF FF marking an unused pen) and
// the PNG must be a 16-entry indexed image in pen order.
func TestCLIConvertUnderscanPlusRealSamples(t *testing.T) {
	for _, tc := range []struct {
		name     string
		rel      string
		modeByte byte
		wantW    int
		wantH    int
	}{
		{"SF2CHUNL", "mode1+/underscan_256x192/raw/SF2CHUNL.SCR", 0x8D, 256, 192},
		{"GEMPAINT", "mode2+/underscan_512x192/raw/GEMPAINT.SCR", 0x8E, 512, 384},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inPath := corpusPath(t, tc.rel)
			raw := mustRead(t, inPath)
			block := raw[128+cpc.ModePalOffset : 128+cpc.ModePalOffset+33]
			if block[0] != tc.modeByte {
				t.Fatalf("Plus mode byte = 0x%02X, want 0x%02X", block[0], tc.modeByte)
			}

			outPath := filepath.Join(t.TempDir(), "out.png")
			rootCmd.SetArgs([]string{"convert", "-i", inPath, "-o", outPath})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("convert failed: %v", err)
			}
			f, err := os.Open(outPath)
			if err != nil {
				t.Fatalf("open PNG failed: %v", err)
			}
			defer f.Close()
			img, _, derr := image.Decode(f)
			if derr != nil {
				t.Fatalf("PNG failed to decode: %v", derr)
			}
			if got := img.Bounds(); got.Dx() != tc.wantW || got.Dy() != tc.wantH {
				t.Errorf("PNG is %dx%d, want %dx%d (underscan at the screen's own resolution)",
					got.Dx(), got.Dy(), tc.wantW, tc.wantH)
			}
			paletted, ok := img.(*image.Paletted)
			if !ok {
				t.Fatalf("PNG is %T, want an indexed image", img)
			}
			if len(paletted.Palette) != 16 {
				t.Fatalf("palette entries = %d, want 16", len(paletted.Palette))
			}
			for pen := 0; pen < 16; pen++ {
				lo, hi := block[1+pen*2], block[2+pen*2]
				want := 0
				if lo != 0xFF || hi != 0xFF { // FF FF = unused pen, read as black
					want = int(hi&0x0F)<<8 | int(lo&0x0F)<<4 | int((lo>>4)&0x0F)
				}
				wantRGB := cpc.PaletteColor(want, true)
				got := color.RGBAModel.Convert(paletted.Palette[pen]).(color.RGBA)
				if int(got.R) != (wantRGB>>16)&0xFF || int(got.G) != (wantRGB>>8)&0xFF || int(got.B) != wantRGB&0xFF {
					t.Errorf("pen %d colour = #%02X%02X%02X, want #%06X (0x0VBR 0x%03X)",
						pen, got.R, got.G, got.B, wantRGB, want)
				}
			}
		})
	}
}

// TestCLIConvertPlusPaletteRealSamples pins the palette decoded from the real
// CPC Plus screens — standard and overscan, the 33-byte block at 0x17D0 or 0x600
// — against the values measured from those files, so a change in the shared
// decoder is caught immediately.
func TestCLIConvertPlusPaletteRealSamples(t *testing.T) {
	cases := []struct {
		name     string
		rel      string
		offset   int
		modeByte byte
		want     [16]uint16
	}{
		{"DEMO1", "mode0+/standard_160x200/raw/DEMO1.SCR", cpc.ModePalOffset, 0x8C,
			[16]uint16{0x000, 0x4F2, 0x8D4, 0x050, 0x999, 0x008, 0x444, 0x0B0, 0x50E, 0xEEE, 0x10D, 0x400, 0x700, 0x90E, 0xF0F, 0x000}},
		{"DEMO2", "mode0+/standard_160x200/raw/DEMO2.SCR", cpc.ModePalOffset, 0x8C,
			[16]uint16{0x020, 0x000, 0x050, 0x50E, 0x10D, 0x90E, 0x008, 0xEEE, 0x400, 0x700, 0xF0F, 0x0B0, 0x999, 0x4F2, 0x444, 0x8D4}},
		{"SPACE", "mode0+/standard_160x200/raw/SPACE.SCR", cpc.ModePalOffset, 0x8C,
			[16]uint16{0x000, 0x007, 0x242, 0x00A, 0x00D, 0x20E, 0x555, 0x40E, 0x575, 0x70E, 0x8A8, 0x999, 0xA0E, 0xBEB, 0xE5F, 0xFFF}},
		{"ODI", "mode1+/standard_320x200/raw/ODI.SCR", cpc.ModePalOffset, 0x8D,
			[16]uint16{0x000, 0x00F, 0x900, 0xDBF, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000}},
		{"SXYBEAST", "mode1+/standard_320x200/raw/SXYBEAST.SCR", cpc.ModePalOffset, 0x8D,
			[16]uint16{0x555, 0x55F, 0xFF5, 0xFFF, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000}},
		{"PREHIST2", "mode0+/overscan_192x272/raw/PREHIST2.SCR", 0x600, 0x8C,
			[16]uint16{0x000, 0x413, 0x455, 0x624, 0x633, 0x566, 0x676, 0x835, 0x779, 0x86A, 0x97B, 0xA5C, 0xC8D, 0xD99, 0xEBE, 0xFFF}},
		{"SNAKLADY", "mode1+/overscan_384x272/raw/SNAKLADY.SCR", 0x600, 0x8D,
			[16]uint16{0x024, 0x339, 0x96D, 0xFDF, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000}},
		{"CAST", "mode2+/overscan_768x272/raw/CAST.SCR", 0x600, 0x8E,
			[16]uint16{0x000, 0xEEA, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inPath := corpusPath(t, tc.rel)
			raw := mustRead(t, inPath)
			if got := raw[128+tc.offset]; got != tc.modeByte {
				t.Fatalf("mode byte = 0x%02X, want 0x%02X", got, tc.modeByte)
			}

			outPath := filepath.Join(t.TempDir(), "out.png")
			rootCmd.SetArgs([]string{"convert", "-i", inPath, "-o", outPath})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("convert failed: %v", err)
			}
			f, err := os.Open(outPath)
			if err != nil {
				t.Fatalf("open PNG failed: %v", err)
			}
			defer f.Close()
			img, _, derr := image.Decode(f)
			if derr != nil {
				t.Fatalf("PNG failed to decode: %v", derr)
			}
			paletted, ok := img.(*image.Paletted)
			if !ok {
				t.Fatalf("PNG is %T, want an indexed image", img)
			}
			if len(paletted.Palette) != 16 {
				t.Fatalf("palette entries = %d, want 16", len(paletted.Palette))
			}
			for pen := 0; pen < 16; pen++ {
				wantRGB := cpc.PaletteColor(int(tc.want[pen]), true)
				got := color.RGBAModel.Convert(paletted.Palette[pen]).(color.RGBA)
				if int(got.R) != (wantRGB>>16)&0xFF || int(got.G) != (wantRGB>>8)&0xFF || int(got.B) != wantRGB&0xFF {
					t.Errorf("pen %d colour = #%02X%02X%02X, want #%06X (0x0VBR 0x%03X)",
						pen, got.R, got.G, got.B, wantRGB, tc.want[pen])
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestCLIConvertOverscanRealSamples renders real overscan screens — a 31936-byte
// payload whose mode byte sits at &600 — and pins both the emitted size (384x272
// for modes 0 and 1, 768x544 for mode 2) and the picture reaching the lower
// half, which is where the lines past 255 live.
func TestCLIConvertOverscanRealSamples(t *testing.T) {
	for _, tc := range []struct {
		name     string
		rel      string
		modeByte byte
		wantW    int
		wantH    int
	}{
		{"CRZCARS2 mode 0", "mode0/overscan_192x272/raw/CRZCARS2.SCR", 0x00, 384, 272},
		{"PORSC959 mode 1", "mode1/overscan_384x272/raw/PORSC959.SCR", 0x01, 384, 272},
		{"NASAWALK mode 2", "mode2/overscan_768x272/raw/NASAWALK.SCR", 0x02, 768, 544},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inPath := corpusPath(t, tc.rel)
			raw := mustRead(t, inPath)
			if got := raw[128+cpc.OverscanModePalOffset]; got != tc.modeByte {
				t.Fatalf("mode byte at &600 = 0x%02X, want 0x%02X", got, tc.modeByte)
			}

			outPath := filepath.Join(t.TempDir(), "out.png")
			rootCmd.SetArgs([]string{"convert", "-i", inPath, "-o", outPath})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("convert failed: %v", err)
			}

			img := decodeIndexedPNG(t, outPath)
			if got := img.Bounds(); got.Dx() != tc.wantW || got.Dy() != tc.wantH {
				t.Errorf("PNG is %dx%d, want %dx%d (overscan at the screen's own resolution)",
					got.Dx(), got.Dy(), tc.wantW, tc.wantH)
			}

			pens := map[uint8]bool{}
			for y := tc.wantH / 2; y < tc.wantH; y++ {
				for x := 0; x < tc.wantW; x++ {
					pens[img.ColorIndexAt(x, y)] = true
				}
			}
			if len(pens) < 2 {
				t.Errorf("lower half holds %d pens, want the picture data (a blank half means the dump was truncated)", len(pens))
			}
		})
	}
}

// F5: honouring a source palette in `convert` (indexed PNG + --palette).

// resetConvertFlags clears the process-global convert flags (and the
// persistent --verbose) that an earlier Execute() in this package may have set,
// so each test starts from the defaults. Tests that set --plus/--overscan/-f/-v
// call it with defer.
func resetConvertFlags(t *testing.T) {
	t.Helper()
	if err := convertCmd.Flags().Set("palette", ""); err != nil {
		t.Fatalf("reset --palette failed: %v", err)
	}
	if err := convertCmd.Flags().Set("plus", "false"); err != nil {
		t.Fatalf("reset --plus failed: %v", err)
	}
	if err := convertCmd.Flags().Set("overscan", "false"); err != nil {
		t.Fatalf("reset --overscan failed: %v", err)
	}
	// `-f png` with a bitmap input goes through the (unimplemented) PNG writer,
	// so leaking it would break later PNG -> SCR tests.
	if err := convertCmd.Flags().Set("format", "scr"); err != nil {
		t.Fatalf("reset --format failed: %v", err)
	}
	if err := rootCmd.PersistentFlags().Set("verbose", "false"); err != nil {
		t.Fatalf("reset --verbose failed: %v", err)
	}
}

// modePalOf reads the 17-byte ModePal block from a saved SCR file (AMSDOS
// wrapped) and returns it.
func modePalOf(t *testing.T, path string) []byte {
	t.Helper()
	data := mustRead(t, path)
	if !cpc.CheckAmsdos(data) {
		t.Fatalf("%s has no valid AMSDOS header", path)
	}
	off := 128 + cpc.ModePalOffset
	if len(data) < off+17 {
		t.Fatalf("%s is too small to hold a ModePal block", path)
	}
	return data[off : off+17]
}

// decodeIndexedPNG opens an indexed PNG and returns it as an *image.Paletted.
func decodeIndexedPNG(t *testing.T, path string) *image.Paletted {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s failed: %v", path, err)
	}
	defer f.Close()
	img, _, derr := image.Decode(f)
	if derr != nil {
		t.Fatalf("decode %s failed: %v", path, derr)
	}
	paletted, ok := img.(*image.Paletted)
	if !ok {
		t.Fatalf("%s decoded as %T, want an indexed image", path, img)
	}
	return paletted
}

// palettedRGB returns the 8-bit RGB of palette entry i.
func palettedRGB(t *testing.T, p color.Palette, i int) color.RGBA {
	t.Helper()
	return color.RGBAModel.Convert(p[i]).(color.RGBA)
}

// writeIndexedPNG writes an 8-bit indexed PNG whose PLTE is exactly palette and
// whose pixels are given by at(x, y) (a palette index).
func writeIndexedPNG(t *testing.T, path string, palette color.Palette, w, h int, at func(x, y int) uint8) string {
	t.Helper()
	img := image.NewPaletted(image.Rect(0, 0, w, h), palette)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetColorIndex(x, y, at(x, y))
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s failed: %v", path, err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("encode %s failed: %v", path, err)
	}
	f.Close()
	return path
}

// exactInkPalette builds an opaque color.Palette from classic CPC ink indices.
func exactInkPalette(inks ...int) color.Palette {
	pal := make(color.Palette, len(inks))
	for i, ink := range inks {
		c := cpc.CpcRgbPalette[ink]
		pal[i] = color.RGBA{R: c.R, G: c.V, B: c.B, A: 255}
	}
	return pal
}

// embedPlusModePal writes a 33-byte CPC Plus ModePal block at ModePalOffset of a
// raw screen, growing it when needed (mirrors cpc.EmbedModePal for the Plus
// layout: mode byte with bit 7 set, then 16 low/high 0x0VBR pairs).
func embedPlusModePal(t *testing.T, screen []byte, block []byte) []byte {
	t.Helper()
	if len(block) != 33 {
		t.Fatalf("Plus ModePal block is %d bytes, want 33", len(block))
	}
	need := cpc.ModePalOffset + len(block)
	if len(screen) < need {
		grown := make([]byte, need)
		copy(grown, screen)
		screen = grown
	}
	copy(screen[cpc.ModePalOffset:], block)
	return screen
}

// plusModePalBlock encodes a 16-entry CPC Plus palette (0x0VBR values) into the
// 33-byte screen block the way fileio.SaveSCR writes it and applySCRPalette
// reads it back: mode byte (mode | 0x8C) then low/high pairs with the low byte
// B|R<<4 and the high byte V.
func plusModePalBlock(mode int, vals [16]int) []byte {
	block := make([]byte, 33)
	block[0] = byte(mode | 0x8C)
	k := 1
	for i := 0; i < 16; i++ {
		v := vals[i]
		block[k] = byte(((v >> 4) & 0x0F) | ((v << 4) & 0xFF))
		k++
		block[k] = byte((v >> 8) & 0x0F)
		k++
	}
	return block
}

// writeKIT writes a 160-byte CPC Plus .kit file (128-byte AMSDOS header + 16
// little-endian entries) that fileio.LoadPaletteKit decodes back to the given
// 0x0VBR values. It is written directly instead of via fileio.SavePaletteKit,
// whose KIT encoder is fixed by a later change.
func writeKIT(t *testing.T, path string, vals [16]int) string {
	t.Helper()
	entete := cpc.CreeEntete(filepath.Base(path), 0x6400, 32, 0)
	header, err := cpc.AmsdosToByte(entete)
	if err != nil {
		t.Fatalf("AmsdosToByte failed: %v", err)
	}
	data := make([]byte, 160)
	copy(data, header)
	for i := 0; i < 16; i++ {
		v := vals[i]
		kit := ((v>>8)&0x0F)<<8 | (v&0x0F)<<4 | (v>>4)&0x0F
		binary.LittleEndian.PutUint16(data[128+i*2:], uint16(kit))
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write KIT failed: %v", err)
	}
	return path
}

// TestCLIConvertPaletteFlagPNGToSCR verifies that `--palette` locks the pens of
// a PNG -> SCR conversion to the order stored in the .pal file, instead of the
// frequency order the truecolor path would pick. The source is dominated by
// white but the .pal asks for ink 1 (blue) on pen 0, so the two orders cannot be
// confused.
func TestCLIConvertPaletteFlagPNGToSCR(t *testing.T) {
	tempDir := t.TempDir()
	defer resetConvertFlags(t)

	// 4 exact CPC colours; white covers most of the canvas.
	white := cpc.CpcRgbPalette[26]
	blue := cpc.CpcRgbPalette[1]
	red := cpc.CpcRgbPalette[6]
	yellow := cpc.CpcRgbPalette[24]
	src := image.NewRGBA(image.Rect(0, 0, 640, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			c := white
			switch {
			case x >= 630:
				c = yellow
			case x >= 620:
				c = red
			case x >= 600:
				c = blue
			}
			src.Set(x, y, color.RGBA{R: c.R, G: c.V, B: c.B, A: 255})
		}
	}
	srcPath := writeRGBA(t, tempDir, "src.png", src)

	// The .pal starts with the rarest colour, so pen 0 differs from frequency.
	pal := [16]uint16{1, 24, 20, 6, 26, 0, 2, 7, 10, 12, 14, 16, 18, 22, 1, 14}
	palPath := filepath.Join(tempDir, "lock.pal")
	if err := fileio.SavePalette(palPath, pal[:], fileio.SCRParams{VirtualMode: 0}); err != nil {
		t.Fatalf("SavePalette failed: %v", err)
	}

	// Without --palette the pens follow frequency: pen 0 gets the most common
	// colour (white, ink 26).
	freqPath := filepath.Join(tempDir, "freq.scr")
	rootCmd.SetArgs([]string{"convert", "-i", srcPath, "-o", freqPath, "-m", "0", "-d", "none"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert (truecolor) failed: %v", err)
	}
	if got := modePalOf(t, freqPath); got[1] != 26 {
		t.Errorf("without --palette pen 0 ink = %d, want 26 (the most frequent colour)", got[1])
	}

	// With --palette the file's order wins, verbatim, over all 16 pens.
	lockedPath := filepath.Join(tempDir, "locked.scr")
	rootCmd.SetArgs([]string{"convert", "-i", srcPath, "-o", lockedPath, "-m", "0", "-d", "none", "--palette", palPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert --palette failed: %v", err)
	}
	want := make([]byte, 17)
	want[0] = 0 // mode 0, from -m
	for i, ink := range pal {
		want[1+i] = byte(ink)
	}
	if got := modePalOf(t, lockedPath); !bytes.Equal(got, want) {
		t.Errorf("locked ModePal = % x, want % x", got, want)
	}
}

// TestCLIConvertPaletteFlagSCRToPNG verifies that `--palette` overrides the
// ModePal embedded in an SCR when converting to PNG: the output PLTE must be the
// .pal colours, and the "SCR palette ignored" note must appear only with
// --verbose (it is not a quality loss).
func TestCLIConvertPaletteFlagSCRToPNG(t *testing.T) {
	tempDir := t.TempDir()
	defer resetConvertFlags(t)

	// The screen carries its own (different) ModePal: mode 1, all white pens.
	scrPal := []byte{1, 26, 26, 26, 26, 26, 26, 26, 26, 26, 26, 26, 26, 26, 26, 26, 26}
	screen := cpc.EmbedModePal(mkRawSCR(cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines)), scrPal)
	scrPath := filepath.Join(tempDir, "screen.scr")
	if err := os.WriteFile(scrPath, screen, 0644); err != nil {
		t.Fatalf("write screen failed: %v", err)
	}

	// The override palette has a distinctive, easily checked order.
	override := [16]uint16{0, 24, 20, 6, 26, 0, 2, 7, 10, 12, 14, 16, 18, 22, 1, 14}
	palPath := filepath.Join(tempDir, "override.pal")
	if err := fileio.SavePalette(palPath, override[:], fileio.SCRParams{VirtualMode: 1}); err != nil {
		t.Fatalf("SavePalette failed: %v", err)
	}

	var stderr bytes.Buffer
	rootCmd.SetErr(&stderr)
	defer rootCmd.SetErr(os.Stderr)

	// Default run: the note must be silent.
	defaultOut := filepath.Join(tempDir, "default.png")
	rootCmd.SetArgs([]string{"convert", "-i", scrPath, "-o", defaultOut, "--palette", palPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert SCR -> PNG failed: %v", err)
	}
	if strings.Contains(stderr.String(), "SCR palette ignored") {
		t.Errorf("override note printed without --verbose: %q", stderr.String())
	}

	paletted := decodeIndexedPNG(t, defaultOut)
	if len(paletted.Palette) != 16 {
		t.Fatalf("palette entries = %d, want 16", len(paletted.Palette))
	}
	for pen := 0; pen < 16; pen++ {
		want := cpc.PaletteColor(int(override[pen]), false)
		got := palettedRGB(t, paletted.Palette, pen)
		if int(got.R) != (want>>16)&0xFF || int(got.G) != (want>>8)&0xFF || int(got.B) != want&0xFF {
			t.Errorf("pen %d colour = #%02X%02X%02X, want #%06X (ink %d)",
				pen, got.R, got.G, got.B, want, override[pen])
		}
	}

	// With --verbose the informational note is printed.
	stderr.Reset()
	verboseOut := filepath.Join(tempDir, "verbose.png")
	rootCmd.SetArgs([]string{"convert", "-i", scrPath, "-o", verboseOut, "--palette", palPath, "-v"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert SCR -> PNG -v failed: %v", err)
	}
	if !strings.Contains(stderr.String(), "using palette from") || !strings.Contains(stderr.String(), "SCR palette ignored") {
		t.Errorf("missing verbose override note, got: %q", stderr.String())
	}
}

// TestCLIConvertPaletteFlagSCRToPNGPlus verifies the CPC Plus side of the SCR ->
// PNG override: a .kit file replaces the Plus ModePal of the screen, and the
// output PLTE holds the .kit's 12-bit colours.
func TestCLIConvertPaletteFlagSCRToPNGPlus(t *testing.T) {
	tempDir := t.TempDir()
	defer resetConvertFlags(t)

	// A CPC Plus screen (mode byte with bit 7 set) with its own Plus block.
	scrVals := [16]int{}
	for i := range scrVals {
		scrVals[i] = 0xFFF // all-white pens, clearly different from the .kit
	}
	screen := embedPlusModePal(t, mkRawSCR(cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines)),
		plusModePalBlock(1, scrVals))
	scrPath := filepath.Join(tempDir, "plus.scr")
	if err := os.WriteFile(scrPath, screen, 0644); err != nil {
		t.Fatalf("write screen failed: %v", err)
	}

	// 16 distinct 12-bit colours in the .kit.
	kitVals := [16]int{}
	for i := range kitVals {
		kitVals[i] = (i << 8) | ((15 - i) << 4) | i
	}
	kitPath := writeKIT(t, filepath.Join(tempDir, "override.kit"), kitVals)

	outPath := filepath.Join(tempDir, "out.png")
	rootCmd.SetArgs([]string{"convert", "-i", scrPath, "-o", outPath, "--palette", kitPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert Plus SCR -> PNG failed: %v", err)
	}

	paletted := decodeIndexedPNG(t, outPath)
	if len(paletted.Palette) != 16 {
		t.Fatalf("palette entries = %d, want 16", len(paletted.Palette))
	}
	for pen := 0; pen < 16; pen++ {
		want := cpc.PaletteColor(kitVals[pen], true)
		got := palettedRGB(t, paletted.Palette, pen)
		if int(got.R) != (want>>16)&0xFF || int(got.G) != (want>>8)&0xFF || int(got.B) != want&0xFF {
			t.Errorf("pen %d colour = #%02X%02X%02X, want #%06X (0x0VBR 0x%03X)",
				pen, got.R, got.G, got.B, want, kitVals[pen])
		}
	}
}

// TestCLIConvertIndexedPNGPaletteHonoured verifies that an indexed PNG whose
// entry count fits the mode budget is honoured: pen i takes PLTE entry i, so the
// pen order of the source survives into the SCR ModePal.
func TestCLIConvertIndexedPNGPaletteHonoured(t *testing.T) {
	tempDir := t.TempDir()
	defer resetConvertFlags(t)

	// Mode 1 supports 4 pens; give the PNG exactly 4 entries, in a non-obvious
	// order (blue, red, yellow, white).
	inks := []int{1, 6, 24, 26}
	pal := exactInkPalette(inks...)
	srcPath := writeIndexedPNG(t, filepath.Join(tempDir, "indexed.png"), pal, 640, 400,
		func(x, y int) uint8 { return uint8(x / 160) })

	scrPath := filepath.Join(tempDir, "out.scr")
	rootCmd.SetArgs([]string{"convert", "-i", srcPath, "-o", scrPath, "-m", "1", "-d", "none"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert indexed PNG -> SCR failed: %v", err)
	}

	got := modePalOf(t, scrPath)
	if got[0] != 1 {
		t.Errorf("ModePal mode byte = %d, want 1", got[0])
	}
	for pen, ink := range inks {
		if int(got[1+pen]) != ink {
			t.Errorf("pen %d ink = %d, want %d (PLTE order must be preserved)", pen, got[1+pen], ink)
		}
	}
	// Mode 1 only defines 4 pens: the rest stay marked unused.
	for pen := len(inks); pen < 16; pen++ {
		if got[1+pen] != 0xFF {
			t.Errorf("pen %d ink = %d, want 0xFF (unused in mode 1)", pen, got[1+pen])
		}
	}
}

// TestCLIConvertIndexedPNGSnapping verifies that a PNG palette entry which is not
// an exact CPC colour is snapped to the nearest one (classic ink, or 12-bit
// 0x0VBR under --plus) instead of falling back to truecolor, and that the
// quality loss is reported on stderr by default.
func TestCLIConvertIndexedPNGSnapping(t *testing.T) {
	t.Run("classic", func(t *testing.T) {
		tempDir := t.TempDir()
		defer resetConvertFlags(t)

		// Entry 0 is a dirty red (not a hardware colour) -> snaps to ink 6;
		// entry 1 is exact black. The order must survive.
		pal := color.Palette{
			color.RGBA{R: 0xFA, G: 0x05, B: 0x05, A: 255},
			color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 255},
		}
		srcPath := writeIndexedPNG(t, filepath.Join(tempDir, "snap.png"), pal, 640, 400,
			func(x, y int) uint8 { return uint8(x / 320) })

		var stderr bytes.Buffer
		rootCmd.SetErr(&stderr)
		defer rootCmd.SetErr(os.Stderr)

		scrPath := filepath.Join(tempDir, "out.scr")
		rootCmd.SetArgs([]string{"convert", "-i", srcPath, "-o", scrPath, "-m", "1", "-d", "none"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("convert indexed PNG -> SCR failed: %v", err)
		}
		if !strings.Contains(stderr.String(), "palette entries snapped") {
			t.Errorf("snapping must warn by default, got: %q", stderr.String())
		}

		got := modePalOf(t, scrPath)
		if got[1] != 6 || got[2] != 0 {
			t.Errorf("snapped pens = [%d %d], want [6 0] (nearest ink, PLTE order kept)", got[1], got[2])
		}
	})

	t.Run("plus", func(t *testing.T) {
		tempDir := t.TempDir()
		defer resetConvertFlags(t)

		// Under --plus the same dirty red snaps to the nearest 0x0VBR colour.
		pal := color.Palette{
			color.RGBA{R: 0xFA, G: 0x05, B: 0x05, A: 255},
			color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 255},
		}
		srcPath := writeIndexedPNG(t, filepath.Join(tempDir, "snap_plus.png"), pal, 640, 400,
			func(x, y int) uint8 { return uint8(x / 320) })

		var stderr bytes.Buffer
		rootCmd.SetErr(&stderr)
		defer rootCmd.SetErr(os.Stderr)

		scrPath := filepath.Join(tempDir, "out.scr")
		rootCmd.SetArgs([]string{"convert", "-i", srcPath, "-o", scrPath, "-m", "1", "--plus", "-d", "none"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("convert indexed PNG -> SCR --plus failed: %v", err)
		}
		if !strings.Contains(stderr.String(), "palette entries snapped") {
			t.Errorf("snapping must warn by default, got: %q", stderr.String())
		}

		snappedRed, _ := cpc.NearestPlus(0xFA, 0x05, 0x05)
		wantPens := [16]int{snappedRed, 0x000}
		want := plusModePalBlock(1, wantPens)
		got := modePalOfPlus(t, scrPath)
		if !bytes.Equal(got[:2], want[:2]) {
			t.Errorf("Plus ModePal mode byte = 0x%02X, want 0x%02X", got[0], want[0])
		}
		if !bytes.Equal(got[1:5], want[1:5]) {
			t.Errorf("snapped Plus pens = % x, want % x", got[1:5], want[1:5])
		}
	})
}

// modePalOfPlus reads the 33-byte CPC Plus ModePal block from a saved SCR file
// (AMSDOS wrapped) and returns it.
func modePalOfPlus(t *testing.T, path string) []byte {
	t.Helper()
	data := mustRead(t, path)
	if !cpc.CheckAmsdos(data) {
		t.Fatalf("%s has no valid AMSDOS header", path)
	}
	off := 128 + cpc.ModePalOffset
	if len(data) < off+33 {
		t.Fatalf("%s is too small to hold a Plus ModePal block", path)
	}
	return data[off : off+33]
}

// TestCLIConvertIndexedPNGOverBudget verifies rule 3: an indexed PNG with more
// entries than the selected mode supports cannot be represented, so it is
// treated exactly like a truecolor image (its palette is ignored) and the
// quality loss is warned about on stderr.
func TestCLIConvertIndexedPNGOverBudget(t *testing.T) {
	tempDir := t.TempDir()
	defer resetConvertFlags(t)

	// 8 distinct CPC colours, one per stripe.
	inks := []int{0, 1, 6, 13, 20, 24, 25, 26}
	pal := exactInkPalette(inks...)
	stripe := func(x, y int) uint8 { return uint8(x / 80) }

	// The same picture twice: once as an 8-entry indexed PNG (over the 4-pen
	// budget of mode 1), once as truecolor.
	indexedPath := writeIndexedPNG(t, filepath.Join(tempDir, "over.png"), pal, 640, 400, stripe)
	truecolor := image.NewRGBA(image.Rect(0, 0, 640, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			c := cpc.CpcRgbPalette[inks[x/80]]
			truecolor.Set(x, y, color.RGBA{R: c.R, G: c.V, B: c.B, A: 255})
		}
	}
	truecolorPath := writeRGBA(t, tempDir, "over_truecolor.png", truecolor)

	var stderr bytes.Buffer
	rootCmd.SetErr(&stderr)
	defer rootCmd.SetErr(os.Stderr)

	indexedSCR := filepath.Join(tempDir, "indexed.scr")
	rootCmd.SetArgs([]string{"convert", "-i", indexedPath, "-o", indexedSCR, "-m", "1", "-d", "none"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert over-budget indexed PNG failed: %v", err)
	}
	if !strings.Contains(stderr.String(), "exceeds the mode budget") {
		t.Errorf("over-budget palette must warn, got: %q", stderr.String())
	}

	stderr.Reset()
	truecolorSCR := filepath.Join(tempDir, "truecolor.scr")
	rootCmd.SetArgs([]string{"convert", "-i", truecolorPath, "-o", truecolorSCR, "-m", "1", "-d", "none"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert truecolor PNG failed: %v", err)
	}
	if strings.Contains(stderr.String(), "exceeds the mode budget") {
		t.Errorf("truecolor input must not warn about a palette budget, got: %q", stderr.String())
	}

	// Compare the payloads: the 128-byte AMSDOS header carries the output file
	// name, which differs between the two runs.
	a, b := mustRead(t, indexedSCR), mustRead(t, truecolorSCR)
	if !bytes.Equal(a[128:], b[128:]) {
		t.Error("an over-budget indexed PNG must produce exactly the truecolor result")
	}
}

// TestCLIConvertIndexedPNGEntryCount verifies that rule 3 counts palette ENTRIES
// and not the entries the pixels actually use: a 256-entry indexed PNG whose
// pixels reference only two of them must still be rejected as over budget.
func TestCLIConvertIndexedPNGEntryCount(t *testing.T) {
	tempDir := t.TempDir()
	defer resetConvertFlags(t)

	// 256 entries, only 0 (black) and 1 (white) ever referenced.
	pal := make(color.Palette, 256)
	for i := range pal {
		pal[i] = color.RGBA{R: uint8(i), G: uint8(255 - i), B: uint8(i), A: 255}
	}
	pal[0] = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	pal[1] = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	indexedPath := writeIndexedPNG(t, filepath.Join(tempDir, "many.png"), pal, 640, 400,
		func(x, y int) uint8 {
			if (x/8)%2 == 0 {
				return 0
			}
			return 1
		})

	// The equivalent truecolor picture (black/white stripes).
	truecolor := image.NewRGBA(image.Rect(0, 0, 640, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			v := uint8(0)
			if (x/8)%2 != 0 {
				v = 255
			}
			truecolor.Set(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	truecolorPath := writeRGBA(t, tempDir, "two_truecolor.png", truecolor)

	var stderr bytes.Buffer
	rootCmd.SetErr(&stderr)
	defer rootCmd.SetErr(os.Stderr)

	indexedSCR := filepath.Join(tempDir, "many.scr")
	rootCmd.SetArgs([]string{"convert", "-i", indexedPath, "-o", indexedSCR, "-m", "1", "-d", "none"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert 256-entry indexed PNG failed: %v", err)
	}
	if !strings.Contains(stderr.String(), "exceeds the mode budget") {
		t.Errorf("a 256-entry palette must warn even when 2 entries are used, got: %q", stderr.String())
	}

	stderr.Reset()
	truecolorSCR := filepath.Join(tempDir, "two_truecolor.scr")
	rootCmd.SetArgs([]string{"convert", "-i", truecolorPath, "-o", truecolorSCR, "-m", "1", "-d", "none"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert truecolor PNG failed: %v", err)
	}
	// Compare the payloads: the AMSDOS header carries the output file name.
	a, b := mustRead(t, indexedSCR), mustRead(t, truecolorSCR)
	if !bytes.Equal(a[128:], b[128:]) {
		t.Error("an over-budget indexed PNG must produce exactly the truecolor result")
	}
}

// TestCLIConvertPaletteRoundTripPenExact verifies the goal of F5: an SCR -> PNG
// -> SCR round trip keeps the pen order of the original when --palette pins it,
// while the same round trip without the flag does not (today's behaviour, which
// re-derives the pens by frequency).
func TestCLIConvertPaletteRoundTripPenExact(t *testing.T) {
	tempDir := t.TempDir()
	defer resetConvertFlags(t)

	// Original screen: mode 1, four distinct inks in a known order.
	original := []byte{1, 0, 6, 24, 26, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	screen := cpc.EmbedModePal(mkRawSCR(cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines)), original)
	scrPath := filepath.Join(tempDir, "original.scr")
	if err := os.WriteFile(scrPath, screen, 0644); err != nil {
		t.Fatalf("write screen failed: %v", err)
	}

	// The matching .pal (same inks, same order).
	palPath := filepath.Join(tempDir, "screen.pal")
	pal := make([]uint16, 16)
	for i := 0; i < 16; i++ {
		pal[i] = uint16(original[1+i])
	}
	if err := fileio.SavePalette(palPath, pal, fileio.SCRParams{VirtualMode: 1}); err != nil {
		t.Fatalf("SavePalette failed: %v", err)
	}

	// SCR -> PNG (the palette override drives the render and the PLTE).
	backPath := filepath.Join(tempDir, "back.png")
	rootCmd.SetArgs([]string{"convert", "-i", scrPath, "-o", backPath, "--palette", palPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert SCR -> PNG failed: %v", err)
	}

	var stderr bytes.Buffer
	rootCmd.SetErr(&stderr)
	defer rootCmd.SetErr(os.Stderr)

	// PNG -> SCR with --palette reproduces the original pen order exactly.
	lockedPath := filepath.Join(tempDir, "locked.scr")
	rootCmd.SetArgs([]string{"convert", "-i", backPath, "-o", lockedPath, "-m", "1", "-d", "none", "--palette", palPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert PNG -> SCR --palette failed: %v", err)
	}
	locked := modePalOf(t, lockedPath)
	// Mode 1 only defines 4 pens, and a classic .pal cannot express "unused"
	// (it stores ink indices 0..26), so compare the mode byte and those 4 pens.
	if !bytes.Equal(locked[:5], original[:5]) {
		t.Errorf("round trip with --palette = % x, want % x", locked[:5], original[:5])
	}

	// Without the flag the F1 indexed PNG (16 entries in mode 1) is over budget,
	// so the pens are re-derived by frequency: the order is not preserved.
	// (--palette is a process-global: clear it, otherwise the previous run's
	// value would still be in effect.)
	stderr.Reset()
	resetConvertFlags(t)
	plainPath := filepath.Join(tempDir, "plain.scr")
	rootCmd.SetArgs([]string{"convert", "-i", backPath, "-o", plainPath, "-m", "1", "-d", "none"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert PNG -> SCR failed: %v", err)
	}
	if !strings.Contains(stderr.String(), "exceeds the mode budget") {
		t.Errorf("without --palette the 16-entry PNG must be over budget, got: %q", stderr.String())
	}
}

// writeKITWords writes a 160-byte CPC Plus .kit file whose 16 words are given
// verbatim (little-endian), so a fixture can hold the 0xFFFF "unused pen" marker
// that a 12-bit value cannot express. It mirrors what fileio.SavePaletteKit
// writes and what fileio.LoadPaletteKit reads.
func writeKITWords(t *testing.T, path string, words [16]uint16) string {
	t.Helper()
	entete := cpc.CreeEntete(filepath.Base(path), 0x6400, 32, 0)
	header, err := cpc.AmsdosToByte(entete)
	if err != nil {
		t.Fatalf("AmsdosToByte failed: %v", err)
	}
	data := make([]byte, 160)
	copy(data, header)
	for i, w := range words {
		binary.LittleEndian.PutUint16(data[128+i*2:], w)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write KIT failed: %v", err)
	}
	return path
}

// TestCLIPackPlusPaletteKit verifies that a CPC Plus screen is packed with an
// external .kit palette: the 33-byte block the .kit describes is written into the
// packed dump — the Plus variants have no palette field in the header — so
// unpacking restores a dump whose &17D0 block is that block byte for byte, the
// unused-pen marker included.
func TestCLIPackPlusPaletteKit(t *testing.T) {
	tempDir := t.TempDir()
	defer resetConvertFlags(t)

	// A mode-1 CPC Plus screen carrying its own (different) Plus block.
	var screenVals [16]int
	for i := range screenVals {
		screenVals[i] = 0x0F0 + i
	}
	screen := embedPlusModePal(t, mkRawSCR(cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines)),
		plusModePalBlock(1, screenVals))
	scrPath := filepath.Join(tempDir, "screen.bin")
	if err := os.WriteFile(scrPath, screen, 0644); err != nil {
		t.Fatalf("write screen failed: %v", err)
	}

	// The .kit holds 16 raw words, deliberately including 0xFFFF (unused pen),
	// which a 12-bit decode cannot represent.
	words := [16]uint16{
		0x0000, 0x042F, 0x084D, 0xFFFF, 0x0FCF, 0x0FFF, 0x0123, 0x0ABC,
		0x0FED, 0x0100, 0x0200, 0x0300, 0xFFFF, 0x0AAA, 0x0BBB, 0x0CCC,
	}
	kitPath := writeKITWords(t, filepath.Join(tempDir, "override.kit"), words)

	pksPath := filepath.Join(tempDir, "out.pks")
	rootCmd.SetArgs([]string{"pack", "-i", scrPath, "-o", pksPath, "--method", "pksp", "--palette", kitPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pack --palette failed: %v", err)
	}

	// The Plus variants keep the 4-byte header: the block rides in the payload.
	packed := mustRead(t, pksPath)
	if !cpc.CheckAmsdos(packed) {
		t.Fatal("packed file has no AMSDOS header")
	}
	h, err := compress.ParsePKSHeader(packed[128:])
	if err != nil {
		t.Fatalf("ParsePKSHeader failed: %v", err)
	}
	if h.DataOffset != 4 {
		t.Errorf("PKSP payload offset = %d, want 4 (no palette field)", h.DataOffset)
	}

	backPath := filepath.Join(tempDir, "back.scr")
	rootCmd.SetArgs([]string{"unpack", "-i", pksPath, "-o", backPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unpack failed: %v", err)
	}

	back := mustRead(t, backPath)
	if len(back) != cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines) {
		t.Fatalf("restored dump is %d bytes, want %d", len(back), cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines))
	}
	want := make([]byte, 33)
	want[0] = 0x8C | 1 // mode 1, read from the screen's own Plus block
	for i, w := range words {
		binary.LittleEndian.PutUint16(want[1+i*2:], w)
	}
	if got := back[cpc.ModePalOffset : cpc.ModePalOffset+33]; !bytes.Equal(got, want) {
		t.Errorf("restored Plus block = % x, want % x", got, want)
	}
}

// TestCLIPackPlusPaletteWithoutFlag verifies that a CPC Plus screen keeps its own
// palette block when no --palette is given (silently, since there is nothing to
// warn about), while a screen whose block was stripped is reported.
func TestCLIPackPlusPaletteWithoutFlag(t *testing.T) {
	tempDir := t.TempDir()
	// --palette is a process-global bound to both commands: clear whatever an
	// earlier test left behind, and leave a clean state for the next one.
	resetConvertFlags(t)
	defer resetConvertFlags(t)

	var stderr bytes.Buffer
	rootCmd.SetErr(&stderr)
	defer rootCmd.SetErr(os.Stderr)

	size := cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines)

	t.Run("screen carries its own Plus block", func(t *testing.T) {
		vals := [16]int{0x00F, 0x0F0, 0xF00, 0xFFF}
		screen := embedPlusModePal(t, mkRawSCR(size), plusModePalBlock(1, vals))
		scrPath := filepath.Join(tempDir, "plus.scr")
		if err := os.WriteFile(scrPath, screen, 0644); err != nil {
			t.Fatalf("write screen failed: %v", err)
		}

		pksPath := filepath.Join(tempDir, "plus.pks")
		rootCmd.SetArgs([]string{"pack", "-i", scrPath, "-o", pksPath, "--method", "pksp"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("pack failed: %v", err)
		}
		if strings.Contains(stderr.String(), "no usable Plus palette") {
			t.Errorf("a screen with its own Plus block must not warn, got: %q", stderr.String())
		}

		backPath := filepath.Join(tempDir, "plus_back.scr")
		rootCmd.SetArgs([]string{"unpack", "-i", pksPath, "-o", backPath})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unpack failed: %v", err)
		}
		back := mustRead(t, backPath)
		if got := back[cpc.ModePalOffset : cpc.ModePalOffset+33]; !bytes.Equal(got, screen[cpc.ModePalOffset:cpc.ModePalOffset+33]) {
			t.Errorf("restored Plus block = % x, want % x", got, screen[cpc.ModePalOffset:cpc.ModePalOffset+33])
		}
	})

	t.Run("stripped Plus block", func(t *testing.T) {
		scrPath := filepath.Join(tempDir, "stripped.scr")
		if err := os.WriteFile(scrPath, mkRawSCR(size), 0644); err != nil {
			t.Fatalf("write screen failed: %v", err)
		}

		stderr.Reset()
		pksPath := filepath.Join(tempDir, "stripped.pks")
		rootCmd.SetArgs([]string{"pack", "-i", scrPath, "-o", pksPath, "--method", "pksp"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("pack failed: %v", err)
		}
		if !strings.Contains(stderr.String(), "no usable Plus palette") {
			t.Errorf("a stripped Plus block must warn, got: %q", stderr.String())
		}
	})
}

// plusBlockValue decodes one 0x0VBR colour of a 33-byte CPC Plus ModePal block
// the way applySCRPalette reads it back.
func plusBlockValue(block []byte, pen int) int {
	lo := block[1+pen*2]
	hi := block[2+pen*2]
	return int(hi&0x0F)<<8 | int(lo&0x0F)<<4 | int((lo>>4)&0x0F)
}

// TestCLIPackPlusPaletteFromClassicPal verifies a classic .pal can drive a CPC
// Plus variant: its ink indices are converted to the equivalent 12-bit colours —
// exact, since all 27 hardware colours are representable — and injected as a Plus
// block in the packed dump.
func TestCLIPackPlusPaletteFromClassicPal(t *testing.T) {
	tempDir := t.TempDir()
	defer resetConvertFlags(t)

	size := cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines)
	scrPath := filepath.Join(tempDir, "screen.scr")
	if err := os.WriteFile(scrPath, mkRawSCR(size), 0644); err != nil {
		t.Fatalf("write screen failed: %v", err)
	}

	inks := [16]uint16{0, 6, 24, 26, 1, 2, 13, 20, 0, 0, 0, 0, 0, 0, 0, 0}
	palPath := filepath.Join(tempDir, "lock.pal")
	if err := fileio.SavePalette(palPath, inks[:], fileio.SCRParams{VirtualMode: 1}); err != nil {
		t.Fatalf("SavePalette failed: %v", err)
	}

	pksPath := filepath.Join(tempDir, "out.pks")
	rootCmd.SetArgs([]string{"pack", "-i", scrPath, "-o", pksPath, "--method", "pksp", "--palette", palPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pack --palette failed: %v", err)
	}

	backPath := filepath.Join(tempDir, "back.scr")
	rootCmd.SetArgs([]string{"unpack", "-i", pksPath, "-o", backPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unpack failed: %v", err)
	}

	block := mustRead(t, backPath)[cpc.ModePalOffset : cpc.ModePalOffset+33]
	if block[0] != 0x8C|1 {
		t.Errorf("Plus block mode byte = 0x%02X, want 0x%02X (the .pal's mode)", block[0], 0x8C|1)
	}
	for pen, ink := range inks {
		rgb := cpc.PaletteColor(int(ink), false)
		want, _ := cpc.NearestPlus(uint8(rgb>>16), uint8(rgb>>8), uint8(rgb))
		if got := plusBlockValue(block, pen); got != want {
			t.Errorf("pen %d = 0x%03X, want 0x%03X (from ink %d)", pen, got, want, ink)
		}
	}
}

// TestCLIPackKitToClassicVariant verifies the other cross-conversion: a .kit
// given to a classic "L" variant is reduced to ink indices in its 17-byte header
// field, instead of being refused as an unreadable .pal (a .kit is 160 bytes, so
// the classic loader used to die on a truncated read).
func TestCLIPackKitToClassicVariant(t *testing.T) {
	tempDir := t.TempDir()
	defer resetConvertFlags(t)

	size := cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines)
	scrPath := filepath.Join(tempDir, "screen.scr")
	if err := os.WriteFile(scrPath, cpc.EmbedModePal(mkRawSCR(size), []byte{1, 0, 6, 24, 26}), 0644); err != nil {
		t.Fatalf("write screen failed: %v", err)
	}

	// A .kit holding the exact hardware colours white, red, green and black.
	vals := [16]int{0xFFF, 0x00F, 0xF00, 0x000}
	var words [16]uint16
	for i, v := range vals {
		words[i] = uint16(((v>>8)&0x0F)<<8 | (v&0x0F)<<4 | (v>>4)&0x0F)
	}
	kitPath := writeKITWords(t, filepath.Join(tempDir, "plus.kit"), words)

	pksPath := filepath.Join(tempDir, "out.pks")
	rootCmd.SetArgs([]string{"pack", "-i", scrPath, "-o", pksPath, "--method", "pksl", "--palette", kitPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pack --palette with a .kit failed: %v", err)
	}

	header, err := compress.ParsePKSHeader(mustRead(t, pksPath)[128:])
	if err != nil {
		t.Fatalf("ParsePKSHeader failed: %v", err)
	}
	if header.Variant != compress.PKSL {
		t.Fatalf("variant = %v, want PKSL", header.Variant)
	}
	if header.Palette[0] != 1 {
		t.Errorf("header palette mode byte = %d, want 1 (the screen's mode)", header.Palette[0])
	}
	for pen, v := range vals {
		rgb := cpc.PaletteColor(v, true)
		want, exact := cpc.NearestInk(uint8(rgb>>16), uint8(rgb>>8), uint8(rgb))
		if !exact {
			t.Fatalf("test fixture colour 0x%03X is not a hardware ink", v)
		}
		if got := int(header.Palette[1+pen]); got != want {
			t.Errorf("pen %d ink = %d, want %d (from 0x%03X)", pen, got, want, v)
		}
	}
}

// TestCLIPlusKitRealSamples exercises the real CPC Plus corpus, where each
// raw+kit directory pairs a screen with the .kit holding its palette. Packing
// with --palette must place the .kit's 33-byte block inside the packed dump —
// the Plus variants have no palette field — so unpacking restores exactly that
// block, the 0xFFFF "unused pen" markers included. A sample that still carries
// its own block must round-trip it unchanged, the block the .kit describes must
// be what the screen itself holds, and the .kit reader and writer must be
// inverses of one another.
func TestCLIPlusKitRealSamples(t *testing.T) {
	root := os.Getenv(corpusSamplesEnv)
	if root == "" {
		t.Skipf("set %s to run the corpus-driven tests", corpusSamplesEnv)
	}
	resetConvertFlags(t)
	defer resetConvertFlags(t)

	var pairs []string
	for _, modeDir := range []string{"mode0+", "mode1+", "mode2+"} {
		found, _ := filepath.Glob(filepath.Join(root, modeDir, "standard_*", "raw+kit", "*.KIT"))
		pairs = append(pairs, found...)
	}
	if len(pairs) == 0 {
		t.Skipf("no CPC Plus .kit pairs under %s", root)
	}

	tempDir := t.TempDir()
	for _, kit := range pairs {
		kit := kit
		name := strings.TrimSuffix(filepath.Base(kit), filepath.Ext(kit))
		t.Run(filepath.Base(filepath.Dir(filepath.Dir(kit)))+"/"+name, func(t *testing.T) {
			kitBytes := mustRead(t, kit)
			payload := kitBytes[128:]
			if len(payload) != 32 {
				t.Fatalf(".kit payload is %d bytes, want 32", len(payload))
			}

			// 1) The .kit reader and writer are inverses: every colour of the
			// corpus .kit survives a save/load cycle (the writer used to swap
			// the wrong nibbles, so 0x042F came back as 0x1300). The 0xFFFF
			// "unused pen" word is the single exception: the loader reads it as
			// the 12-bit colour 0xFFF, so a re-saved file marks that pen white.
			var loaded [16]uint16
			if err := fileio.LoadPaletteKit(kit, loaded[:]); err != nil {
				t.Fatalf("LoadPaletteKit failed: %v", err)
			}
			outKIT := filepath.Join(tempDir, name+".kit")
			if err := fileio.SavePaletteKit(outKIT, loaded[:]); err != nil {
				t.Fatalf("SavePaletteKit failed: %v", err)
			}
			var reloaded [16]uint16
			if err := fileio.LoadPaletteKit(outKIT, reloaded[:]); err != nil {
				t.Fatalf("LoadPaletteKit of the re-saved file failed: %v", err)
			}
			if reloaded != loaded {
				t.Errorf("KIT round trip = %v, want %v", reloaded, loaded)
			}
			got := mustRead(t, outKIT)[128:]
			for i := 0; i < 16; i++ {
				wantWord := binary.LittleEndian.Uint16(payload[i*2:])
				if wantWord == 0xFFFF {
					continue // unused pen, not a colour, see above
				}
				if gotWord := binary.LittleEndian.Uint16(got[i*2:]); gotWord != wantWord {
					t.Errorf("pen %d word = 0x%04X, want 0x%04X", i, gotWord, wantWord)
				}
			}

			// 2) A screen carrying its own Plus block describes exactly the
			// palette the .kit holds: 0x8C|mode followed by the payload.
			rawScr := filepath.Join(filepath.Dir(filepath.Dir(kit)), "raw", name+".SCR")
			rawBlock := []byte(nil)
			if _, err := os.Stat(rawScr); err == nil {
				rawBytes := mustRead(t, rawScr)
				rawPayload := rawBytes[128:]
				block := cpc.ExtractModePal(rawPayload)
				if len(block) != 33 {
					t.Fatalf("ExtractModePal(%s) = %d bytes, want the 33-byte Plus block", rawScr, len(block))
				}
				rawBlock = block
				if block[0]&0x80 == 0 {
					t.Errorf("Plus block mode byte = 0x%02X, want bit 7 set", block[0])
				}
				if !bytes.Equal(block[1:], payload) {
					t.Errorf("screen block payload = % x, want the .kit payload % x", block[1:], payload)
				}

				// ...and packing that screen with --palette restores the very
				// same block, so nothing about it is lost on the way.
				pksPath := filepath.Join(tempDir, name+"_raw.pks")
				backPath := filepath.Join(tempDir, name+"_raw.scr")
				rootCmd.SetArgs([]string{"pack", "-i", rawScr, "-o", pksPath, "--method", "pksp", "--palette", kit})
				if err := rootCmd.Execute(); err != nil {
					t.Fatalf("pack %s failed: %v", rawScr, err)
				}
				rootCmd.SetArgs([]string{"unpack", "-i", pksPath, "-o", backPath})
				if err := rootCmd.Execute(); err != nil {
					t.Fatalf("unpack failed: %v", err)
				}
				back := mustRead(t, backPath)
				if got := back[cpc.ModePalOffset : cpc.ModePalOffset+33]; !bytes.Equal(got, rawBlock) {
					t.Errorf("restored block = % x, want the screen's own % x", got, rawBlock)
				}
			}

			// 3) The raw+kit screen has a stripped block, so the mode is unknown
			// and defaults to 0: the .kit payload is still placed verbatim.
			scrPath := filepath.Join(filepath.Dir(kit), name+".SCR")
			if _, err := os.Stat(scrPath); err != nil {
				t.Skipf("screen not available: %s", scrPath)
			}
			pksPath := filepath.Join(tempDir, name+".pks")
			backPath := filepath.Join(tempDir, name+".scr")
			rootCmd.SetArgs([]string{"pack", "-i", scrPath, "-o", pksPath, "--method", "pksp", "--palette", kit})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("pack %s failed: %v", scrPath, err)
			}

			// The packed file keeps the 4-byte header of every Plus variant.
			packed := mustRead(t, pksPath)
			h, err := compress.ParsePKSHeader(packed[128:])
			if err != nil {
				t.Fatalf("ParsePKSHeader failed: %v", err)
			}
			if !h.CpcPlus {
				t.Errorf("packed header does not report CPC Plus (variant %v)", h.Variant)
			}
			if h.DataOffset != 4 {
				t.Errorf("%v payload offset = %d, want 4 (no palette field)", h.Variant, h.DataOffset)
			}

			rootCmd.SetArgs([]string{"unpack", "-i", pksPath, "-o", backPath})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("unpack failed: %v", err)
			}

			want := make([]byte, 33)
			want[0] = 0x8C // stripped block: the mode cannot be recovered, so 0
			copy(want[1:], payload)
			back := mustRead(t, backPath)
			if len(back) != cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines) {
				t.Fatalf("restored dump is %d bytes, want %d", len(back), cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines))
			}
			if got := back[cpc.ModePalOffset : cpc.ModePalOffset+33]; !bytes.Equal(got, want) {
				t.Errorf("restored block = % x, want % x", got, want)
			}
		})
	}
}

// TestCLIUnpackRejectsRenderedOutputFormats verifies that `unpack` refuses to
// write its decompressed bytes into a file whose name announces a format it does
// not produce. A restored PKS payload is raw screen data, so `unpack
// -o screen.png` used to emit 16336 raw bytes, report success with exit 0 and
// leave behind a file no viewer can open — the mistake only surfacing later, in
// whatever read it. Those extensions now fail with a non-zero exit and write
// nothing, while the raw outputs (.scr, .bin) keep working unchanged.
func TestCLIUnpackRejectsRenderedOutputFormats(t *testing.T) {
	tempDir := t.TempDir()

	// A real PKSL payload, produced by the pack command itself.
	srcSCR := filepath.Join(tempDir, "src.scr")
	if err := writeSCR(srcSCR, cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines)); err != nil {
		t.Fatalf("writeSCR failed: %v", err)
	}
	pksPath := filepath.Join(tempDir, "screen.pks")
	rootCmd.SetArgs([]string{"pack", "-i", srcSCR, "-o", pksPath, "--method", "pksl"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pack failed: %v", err)
	}

	// Case is irrelevant, so .PNG must be rejected too.
	for _, ext := range []string{".png", ".PNG", ".jpg", ".jpeg", ".gif", ".bmp", ".asm", ".dsk"} {
		t.Run(ext, func(t *testing.T) {
			outPath := filepath.Join(tempDir, "screen"+ext)
			rootCmd.SetArgs([]string{"unpack", "-i", pksPath, "-o", outPath})
			err := rootCmd.Execute()
			if err == nil {
				t.Fatalf("unpack into %s must fail", ext)
			}
			if !strings.Contains(err.Error(), "not supported") {
				t.Errorf("error should report an unsupported output format, got: %v", err)
			}
			if _, serr := os.Stat(outPath); serr == nil {
				t.Errorf("unpack must not create %s", outPath)
			}
		})
	}

	// The outputs `unpack` does produce are unaffected, and a restored screen is
	// exactly what the other commands expect to read.
	scrOut := filepath.Join(tempDir, "back.scr")
	rootCmd.SetArgs([]string{"unpack", "-i", pksPath, "-o", scrOut})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unpack into .scr failed: %v", err)
	}
	if got := mustRead(t, scrOut); len(got) != cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines) {
		t.Errorf("restored .scr is %d bytes, want %d", len(got), cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines))
	}

	binOut := filepath.Join(tempDir, "back.bin")
	rootCmd.SetArgs([]string{"unpack", "-i", pksPath, "-o", binOut})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unpack into .bin failed: %v", err)
	}
	if !bytes.Equal(mustRead(t, scrOut), mustRead(t, binOut)) {
		t.Error("the .scr and .bin outputs must hold the same bytes")
	}
}

// TestCLIVersionFlag verifies that --version reports the release being built.
// The flag went missing during the import from the 1.2.0-pks tree, and the
// string it reports must name this release (1.2.1), not the unofficial -pks
// build it replaced; the verbose banner carries the same string.
func TestCLIVersionFlag(t *testing.T) {
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	defer rootCmd.SetOut(os.Stdout)

	rootCmd.SetArgs([]string{"--version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("--version failed: %v", err)
	}

	want := "convimgcpc version " + version.Version
	if got := strings.TrimSpace(out.String()); got != want {
		t.Errorf("--version printed %q, want %q", got, want)
	}
	if version.Version != "1.2.1" {
		t.Errorf("version = %q, want the released 1.2.1", version.Version)
	}
	// -v stays the verbose shorthand: cobra drops the -v alias for --version
	// when the shorthand is taken, and a silent `-v` (no banner) would mean the
	// alias had been hijacked.
	if f := rootCmd.Flags().Lookup("version"); f == nil {
		t.Fatal("--version flag is not registered")
	} else if f.Shorthand != "" {
		t.Errorf("--version shorthand = %q, want none (-v is --verbose)", f.Shorthand)
	}
}

// TestCLIConvertRefusesZeroedAMSDOSHeader pins the case that motivated the
// AMSDOS field tests: an all-zero 128-byte prefix satisfies a checksum
// comparison (0 sums to a stored 0), so a file of zeros used to be read as a
// headed screen — the first 128 bytes were taken for a "header" and the
// remainder rendered as a screen sized around them. It must now be refused,
// with a warning that says what the prefix was, and nothing written.
func TestCLIConvertRefusesZeroedAMSDOSHeader(t *testing.T) {
	cases := []struct {
		name string
		size int
	}{
		{"standard-shaped zero file", cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines) + 128},
		{"underscan-shaped zero file", cpc.BitmapSize(cpc.UnderscanCols, cpc.UnderscanLines) + 128},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			inPath := filepath.Join(tempDir, "zeros.scr")
			if err := os.WriteFile(inPath, make([]byte, tc.size), 0644); err != nil {
				t.Fatalf("write fixture failed: %v", err)
			}
			outPath := filepath.Join(tempDir, "zeros.png")

			var stderr bytes.Buffer
			rootCmd.SetErr(&stderr)
			defer rootCmd.SetErr(os.Stderr)
			resetConvertFlags(t)
			defer resetConvertFlags(t)

			rootCmd.SetArgs([]string{"convert", "-i", inPath, "-o", outPath})
			if err := rootCmd.Execute(); err == nil {
				t.Fatal("convert of a zeroed file succeeded, want a refusal")
			}
			if _, serr := os.Stat(outPath); serr == nil {
				t.Error("a PNG was written for a zeroed file")
			}
			if !strings.Contains(stderr.String(), "zeroed AMSDOS header") {
				t.Errorf("stderr does not explain the zeroed prefix: %q", stderr.String())
			}
		})
	}
}

// TestCLIConvertWarnsOnBrokenHeader verifies the lenient reading is kept and
// made visible: a screen whose AMSDOS checksum a writer got wrong still
// converts with a warning naming the problem, while a screen whose header holds
// up converts without one.
func TestCLIConvertWarnsOnBrokenHeader(t *testing.T) {
	payload := mkRawSCR(cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines))
	headed := amsdosWrap(t, payload, 0xC000)
	broken := append([]byte{}, headed...)
	broken[0x43] ^= 0xFF // corrupt the stored checksum only

	cases := []struct {
		name     string
		data     []byte
		wantWarn bool
	}{
		{"corrupt checksum", broken, true},
		{"healthy header", headed, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			inPath := filepath.Join(tempDir, "screen.scr")
			if err := os.WriteFile(inPath, tc.data, 0644); err != nil {
				t.Fatalf("write fixture failed: %v", err)
			}
			outPath := filepath.Join(tempDir, "screen.png")

			var stderr bytes.Buffer
			rootCmd.SetErr(&stderr)
			defer rootCmd.SetErr(os.Stderr)
			resetConvertFlags(t)
			defer resetConvertFlags(t)

			rootCmd.SetArgs([]string{"convert", "-i", inPath, "-o", outPath})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("convert failed: %v", err)
			}
			if _, serr := os.Stat(outPath); serr != nil {
				t.Errorf("no PNG was written: %v", serr)
			}

			warned := strings.Contains(stderr.String(), "AMSDOS header")
			if warned != tc.wantWarn {
				t.Errorf("header warning = %v, want %v (stderr: %q)", warned, tc.wantWarn, stderr.String())
			}
			if tc.wantWarn && !strings.Contains(stderr.String(), "checksum") {
				t.Errorf("warning does not name the checksum: %q", stderr.String())
			}
		})
	}
}

// TestCLIPackKeepsZeroedPrefix verifies the same rule on the packing side: a
// file whose first 128 bytes are zero is compressed as it is — those bytes are
// data, not a header — with a warning, while the same screen behind a real
// header is stripped and compressed as picture data only.
func TestCLIPackKeepsZeroedPrefix(t *testing.T) {
	payload := mkRawSCR(cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines))
	zeroed := append(make([]byte, 128), payload...)
	headed := amsdosWrap(t, payload, 0xC000)

	cases := []struct {
		name     string
		data     []byte
		want     []byte
		wantWarn bool
	}{
		{"zeroed prefix", zeroed, zeroed, true},
		{"real header", headed, payload, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			inPath := filepath.Join(tempDir, "input.bin")
			if err := os.WriteFile(inPath, tc.data, 0644); err != nil {
				t.Fatalf("write fixture failed: %v", err)
			}
			outPath := filepath.Join(tempDir, "out.zx0")

			var stderr bytes.Buffer
			rootCmd.SetErr(&stderr)
			defer rootCmd.SetErr(os.Stderr)

			rootCmd.SetArgs([]string{"pack", "-i", inPath, "-o", outPath, "--method", "zx0"})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("pack failed: %v", err)
			}

			want := make([]byte, len(tc.want)*2+1024)
			n, err := compress.NewCompressor().Pack(tc.want, len(tc.want), want, 0, compress.MethodZX0)
			if err != nil {
				t.Fatalf("reference Pack failed: %v", err)
			}
			if got := mustRead(t, outPath); !bytes.Equal(got, want[:n]) {
				t.Errorf("packed payload is not the compression of the expected %d bytes", len(tc.want))
			}

			warned := strings.Contains(stderr.String(), "zeroed AMSDOS header")
			if warned != tc.wantWarn {
				t.Errorf("zeroed-prefix warning = %v, want %v (stderr: %q)", warned, tc.wantWarn, stderr.String())
			}
		})
	}
}
