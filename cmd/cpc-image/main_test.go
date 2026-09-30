// Package main provides tests for the cpc-image command-line tool
// (cmd/cpc-image/main.go).
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
				if !bytes.Equal(cpc.ScreenToColumnMajor(restored), cpc.ScreenToColumnMajor(testData)) {
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
	if back.Bounds().Dx() != 640 || back.Bounds().Dy() != 400 {
		t.Fatalf("back PNG dims = %dx%d, want 640x400", back.Bounds().Dx(), back.Bounds().Dy())
	}

	mismatches := 0
	for y := 0; y < 400 && mismatches < 5; y++ {
		for x := 0; x < 640 && mismatches < 5; x++ {
			c1 := color.RGBAModel.Convert(src.At(x, y)).(color.RGBA)
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

// TestCLIConvertRejectsPKSInput verifies that convert refuses a PKS-compressed
// screen ("PKxx" signature) instead of feeding the compressed bytes to the
// screen renderer, which would silently emit a corrupt PNG.
func TestCLIConvertRejectsPKSInput(t *testing.T) {
	tempDir := t.TempDir()

	// A real PKSL payload produced by the compressor, plus a bare "PKUL"
	// (underscan) signature: every variant must be rejected, including those
	// that ParsePKSHeader does not recognize.
	raw := mkRawSCR(cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines))
	pksBuf := make([]byte, len(raw)*2+1024)
	packedSize, perr := compress.NewCompressor().PKS().PackPKS(raw, len(raw), pksBuf, compress.PKSL, nil)
	if perr != nil {
		t.Fatalf("PackPKS failed: %v", perr)
	}
	pkul := append([]byte("PKUL"), raw...)

	cases := []struct {
		name string
		data []byte
	}{
		{"PKSL", pksBuf[:packedSize]},
		{"PKUL", pkul},
	}

	// Keep the .scr extension: that is the route a user would actually take
	// when pointing convert at a compressed screen dump.
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inPath := filepath.Join(tempDir, tc.name+".scr")
			if err := os.WriteFile(inPath, tc.data, 0644); err != nil {
				t.Fatalf("write fixture failed: %v", err)
			}

			outPath := filepath.Join(tempDir, tc.name+".png")
			rootCmd.SetArgs([]string{"convert", "-i", inPath, "-o", outPath})
			err := rootCmd.Execute()
			if err == nil {
				t.Fatal("convert on a PKS-compressed screen should fail")
			}
			if !strings.Contains(err.Error(), "PKS") {
				t.Errorf("error should mention PKS, got: %v", err)
			}
			if _, serr := os.Stat(outPath); serr == nil {
				t.Error("convert must not write an output file for rejected PKS input")
			}
		})
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
