package main

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ikari-pl/go-cpc-image/pkg/cpc"
)

// pngPHYsOf reads the pHYs chunk of the PNG at path with an independent walk of the
// file — checking every CRC on the way, so the chunk has to be well formed and not
// merely present — and returns the pair, its unit and the index of the chunk. ok is
// false when the file declares no density at all.
func pngPHYsOf(t *testing.T, path string) (ppux, ppuy uint32, unit byte, index int, ok bool) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s failed: %v", path, err)
	}
	const pngSignature = 8
	const pngChunkHeader = 12

	if !bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		t.Fatalf("%s is not a PNG: the signature is missing", path)
	}

	index = -1
	for off, i := pngSignature, 0; off+pngChunkHeader <= len(data); i++ {
		length := int(binary.BigEndian.Uint32(data[off : off+4]))
		if off+pngChunkHeader+length > len(data) {
			t.Fatalf("%s: a chunk runs past the end of the file", path)
		}
		kind := string(data[off+4 : off+8])
		want := binary.BigEndian.Uint32(data[off+8+length : off+12+length])
		if got := crc32.ChecksumIEEE(data[off+4 : off+8+length]); got != want {
			t.Fatalf("%s: chunk %s has CRC %08x, want %08x", path, kind, got, want)
		}
		if kind == "pHYs" {
			if length != 9 {
				t.Fatalf("%s: pHYs carries %d bytes, want 9", path, length)
			}
			body := data[off+8 : off+8+length]
			ppux = binary.BigEndian.Uint32(body[0:4])
			ppuy = binary.BigEndian.Uint32(body[4:8])
			unit = body[8]
			index = i
			ok = true
		}
		off += pngChunkHeader + length
	}
	return ppux, ppuy, unit, index, ok
}

// convertScreenToPNG runs the real SCR -> PNG path for one synthetic screen and
// returns the path of the PNG it wrote, so a test can then read the file back.
func convertScreenToPNG(t *testing.T, numCol, numLig, mode int, args ...string) string {
	t.Helper()
	dir := t.TempDir()
	inPath := filepath.Join(dir, "screen.scr")
	if err := os.WriteFile(inPath, syntheticSCR(numCol, numLig, mode), 0644); err != nil {
		t.Fatalf("write screen failed: %v", err)
	}

	outPath := filepath.Join(dir, "screen.png")
	rootCmd.SetArgs(append([]string{"convert", "-i", inPath, "-o", outPath}, args...))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert SCR -> PNG failed: %v", err)
	}
	return outPath
}

// TestCLIConvertSCRToPNGDeclaresDensity checks the whole path — a real dump, the
// real writer, the file on disk — for every mode, scan area and output knob: the
// density the file declares, the size of the grid it was written at, and the fact
// that the chunk sits right after IHDR and means pixels per metre.
func TestCLIConvertSCRToPNGDeclaresDensity(t *testing.T) {
	for _, tc := range []struct {
		name           string
		numCol, numLig int
		mode           int
		args           []string
		wantW, wantH   int
		wantX, wantY   uint32
	}{
		{"mode 1 standard", cpc.StandardCols, cpc.StandardLines, 1, nil, 320, 200, 1416, 1416},
		{"mode 1 underscan", cpc.UnderscanCols, cpc.UnderscanLines, 1, nil, 256, 192, 1416, 1416},
		{"mode 1 overscan", cpc.OverscanCols, cpc.OverscanLines, 1, nil, 384, 272, 1416, 1416},
		{"mode 0 standard (duplicated across)", cpc.StandardCols, cpc.StandardLines, 0, nil, 320, 200, 1416, 1416},
		{"mode 0 underscan", cpc.UnderscanCols, cpc.UnderscanLines, 0, nil, 256, 192, 1416, 1416},
		{"mode 0 overscan", cpc.OverscanCols, cpc.OverscanLines, 0, nil, 384, 272, 1416, 1416},
		{"mode 0 native standard", cpc.StandardCols, cpc.StandardLines, 0,
			[]string{"--native"}, 160, 200, 708, 1416},
		{"mode 0 native scaled 2x", cpc.StandardCols, cpc.StandardLines, 0,
			[]string{"--native", "--scale", "2"}, 320, 400, 1416, 2832},
		{"mode 1 standard scaled 2x", cpc.StandardCols, cpc.StandardLines, 1,
			[]string{"--scale", "2"}, 640, 400, 2832, 2832},
		{"mode 2 standard (whole canvas)", cpc.StandardCols, cpc.StandardLines, 2, nil, 640, 400, 2832, 2832},
		{"mode 2 underscan", cpc.UnderscanCols, cpc.UnderscanLines, 2, nil, 512, 384, 2832, 2832},
		{"mode 2 overscan", cpc.OverscanCols, cpc.OverscanLines, 2, nil, 768, 544, 2832, 2832},
		{"mode 2 native standard", cpc.StandardCols, cpc.StandardLines, 2,
			[]string{"--native"}, 640, 200, 2832, 1416},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConvertFlags(t)
			defer resetConvertFlags(t)

			outPath := convertScreenToPNG(t, tc.numCol, tc.numLig, tc.mode, tc.args...)

			// The grid the chunk describes: checked here on its own, so a density
			// that matched a wrongly sized file could not pass.
			img := decodeIndexedPNG(t, outPath)
			if got := img.Bounds(); got.Dx() != tc.wantW || got.Dy() != tc.wantH {
				t.Errorf("PNG is %dx%d, want %dx%d", got.Dx(), got.Dy(), tc.wantW, tc.wantH)
			}

			ppux, ppuy, unit, index, ok := pngPHYsOf(t, outPath)
			if !ok {
				t.Fatal("the PNG declares no pixel density")
			}
			if index != 1 {
				t.Errorf("pHYs is chunk %d, want 1 (right after IHDR)", index)
			}
			if unit != 1 {
				t.Errorf("pHYs unit = %d, want 1 (pixels per metre)", unit)
			}
			if ppux != tc.wantX || ppuy != tc.wantY {
				t.Errorf("density = %d x %d pixels/meter, want %d x %d",
					ppux, ppuy, tc.wantX, tc.wantY)
			}
		})
	}
}

// TestCLIConvertSCRToPNGDensityIgnoresResize pins that the density describes the
// grid the screen is written at and not the source picture: how the source was
// fitted onto the screen cannot change how big a CPC pixel is.
func TestCLIConvertSCRToPNGDensityIgnoresResize(t *testing.T) {
	for _, policy := range []string{"fit", "keepsmaller", "keeplarger", "origin"} {
		t.Run(policy, func(t *testing.T) {
			resetConvertFlags(t)
			defer resetConvertFlags(t)

			outPath := convertScreenToPNG(t, cpc.StandardCols, cpc.StandardLines, 1,
				"--resize", policy)

			ppux, ppuy, _, _, ok := pngPHYsOf(t, outPath)
			if !ok {
				t.Fatal("the PNG declares no pixel density")
			}
			if ppux != 1416 || ppuy != 1416 {
				t.Errorf("density = %d x %d pixels/meter, want 1416 x 1416 whatever --resize did",
					ppux, ppuy)
			}
		})
	}
}

// TestCLIConvertSCRToPNGVerboseReportsDensity pins the -v line: it is how a user
// sees what the file declares without opening it, and it reports the density in both
// units, metres being the one the chunk carries.
func TestCLIConvertSCRToPNGVerboseReportsDensity(t *testing.T) {
	resetConvertFlags(t)
	defer resetConvertFlags(t)

	var stderr bytes.Buffer
	rootCmd.SetErr(&stderr)
	defer rootCmd.SetErr(os.Stderr)

	// A mode-0 native grid is the interesting one to report: its two values differ.
	convertScreenToPNG(t, cpc.StandardCols, cpc.StandardLines, 0, "--native", "-v")

	if want := "Pixel density: 708 x 1416 pixels/meter (18 x 36 dpi)"; !strings.Contains(stderr.String(), want) {
		t.Errorf("stderr does not carry %q:\n%s", want, stderr.String())
	}
}
