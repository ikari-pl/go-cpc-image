package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ikari-pl/go-cpc-image/pkg/cpc"
)

// writeScreenSource writes a deterministic PNG of exactly w x h, so a conversion
// has a picture whose size is the only thing that decides the scan area under
// `--geometry auto`.
func writeScreenSource(t *testing.T, dir string, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := cpc.CpcRgbPalette[(x/16+y/16)%16]
			img.Set(x, y, color.RGBA{R: c.R, G: c.V, B: c.B, A: 255})
		}
	}
	return writeRGBA(t, dir, fmt.Sprintf("source-%dx%d.png", w, h), img)
}

// convertToDump runs `convert` for one source and returns the bytes of the .scr
// it writes, together with what the run printed on stderr.
func convertToDump(t *testing.T, dir string, in string, args ...string) ([]byte, string) {
	t.Helper()
	out := filepath.Join(dir, "screen.scr")
	resetConvertFlags(t)
	defer resetConvertFlags(t)

	var stderr bytes.Buffer
	rootCmd.SetErr(&stderr)
	defer rootCmd.SetErr(os.Stderr)

	rootCmd.SetArgs(append([]string{"convert", "-i", in, "-o", out}, args...))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert %v failed: %v", args, err)
	}
	return mustRead(t, out), stderr.String()
}

// TestCLIConvertGeometryWritesScanArea checks that each --geometry writes a
// screen dump of its own scan area (an overscan dump is four times the standard
// one, an underscan dump smaller) and that the retired --overscan flag still
// means overscan.
func TestCLIConvertGeometryWritesScanArea(t *testing.T) {
	cases := []struct {
		name  string
		args  []string
		cols  int
		lines int
	}{
		{"standard", []string{"--geometry", "standard"}, cpc.StandardCols, cpc.StandardLines},
		{"underscan", []string{"--geometry", "underscan"}, cpc.UnderscanCols, cpc.UnderscanLines},
		{"overscan", []string{"--geometry", "overscan"}, cpc.OverscanCols, cpc.OverscanLines},
		{"alias --overscan", []string{"--overscan"}, cpc.OverscanCols, cpc.OverscanLines},
		{"auto falls back to standard", nil, cpc.StandardCols, cpc.StandardLines},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			in := writeScreenSource(t, dir, 40, 40) // matches no screen grid

			data, _ := convertToDump(t, dir, in, append([]string{"-m", "1"}, tc.args...)...)

			want := 128 + cpc.BitmapSize(tc.cols, tc.lines)
			if len(data) != want {
				t.Errorf("dump is %d bytes, want %d (128-byte header + %d)",
					len(data), want, cpc.BitmapSize(tc.cols, tc.lines))
			}
			if !cpc.CheckAmsdos(data) {
				t.Error("dump carries no valid AMSDOS header")
			}
		})
	}
}

// TestCLIConvertOverscanAliasIsByteIdentical pins the backwards compatibility of
// the boolean: --overscan must write exactly what --geometry overscan writes.
func TestCLIConvertOverscanAliasIsByteIdentical(t *testing.T) {
	srcDir := t.TempDir()
	in := writeScreenSource(t, srcDir, 100, 80)

	aliasDir := t.TempDir()
	alias, _ := convertToDump(t, aliasDir, in, "-m", "1", "--overscan")

	explicitDir := t.TempDir()
	explicit, _ := convertToDump(t, explicitDir, in, "-m", "1", "--geometry", "overscan")

	if !bytes.Equal(alias, explicit) {
		t.Errorf("--overscan and --geometry overscan differ (%d vs %d bytes)", len(alias), len(explicit))
	}
}

// TestCLIConvertGeometryConflict checks the one combination the flags refuse: a
// wrong --overscan in a batch line must not silently win by precedence.
func TestCLIConvertGeometryConflict(t *testing.T) {
	dir := t.TempDir()
	in := writeScreenSource(t, dir, 40, 40)
	out := filepath.Join(dir, "screen.scr")

	resetConvertFlags(t)
	defer resetConvertFlags(t)
	rootCmd.SetArgs([]string{"convert", "-i", in, "-o", out, "--overscan", "--geometry", "underscan"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("--overscan with --geometry underscan was accepted, want an error")
	}
	if !strings.Contains(err.Error(), "overscan") || !strings.Contains(err.Error(), "underscan") {
		t.Errorf("error %q should name both flags", err)
	}

	// Redundant, not conflicting.
	resetConvertFlags(t)
	rootCmd.SetArgs([]string{"convert", "-i", in, "-o", out, "--overscan", "--geometry", "overscan"})
	if err := rootCmd.Execute(); err != nil {
		t.Errorf("--overscan --geometry overscan failed: %v", err)
	}
}

// TestCLIConvertGeometryAutoReadsEmittedGrid converts a picture drawn at the grid
// this tool emits for a scan area and expects `auto` to pick that scan area back,
// which is what makes re-converting a PNG it produced geometry-stable.
func TestCLIConvertGeometryAutoReadsEmittedGrid(t *testing.T) {
	for _, g := range []screenGeometry{geomStandard, geomUnderscan, geomOverscan} {
		t.Run(g.name, func(t *testing.T) {
			w, h := scrOutputSize(g.cols, g.lines, 1)

			srcDir := t.TempDir()
			in := writeScreenSource(t, srcDir, w, h)

			autoDir := t.TempDir()
			auto, _ := convertToDump(t, autoDir, in, "-m", "1")

			explicitDir := t.TempDir()
			explicit, _ := convertToDump(t, explicitDir, in, "-m", "1", "--geometry", g.name)

			if !bytes.Equal(auto, explicit) {
				t.Errorf("auto picked another scan area than --geometry %s for a %dx%d source", g.name, w, h)
			}
		})
	}
}

// TestCLIConvertGeometryAutoReadsCanvasAndNative covers the other two families of
// the rule: a source drawn on the renderer's canvas and one drawn at the screen's
// own native resolution. The mode is what tells mode 0's 160x200 native picture
// apart from the pictures that share a canvas.
func TestCLIConvertGeometryAutoReadsCanvasAndNative(t *testing.T) {
	cases := []struct {
		name  string
		mode  int
		w, h  int
		cols  int
		lines int
	}{
		{"mode 0 native standard", 0, 160, 200, cpc.StandardCols, cpc.StandardLines},
		{"mode 0 native underscan", 0, 128, 192, cpc.UnderscanCols, cpc.UnderscanLines},
		{"mode 0 native overscan", 0, 192, 272, cpc.OverscanCols, cpc.OverscanLines},
		{"mode 1 native underscan", 1, 256, 192, cpc.UnderscanCols, cpc.UnderscanLines},
		{"mode 2 native standard", 2, 640, 200, cpc.StandardCols, cpc.StandardLines},
		{"standard canvas", 1, 640, 400, cpc.StandardCols, cpc.StandardLines},
		{"underscan canvas", 1, 512, 384, cpc.UnderscanCols, cpc.UnderscanLines},
		{"overscan canvas", 1, 768, 544, cpc.OverscanCols, cpc.OverscanLines},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			in := writeScreenSource(t, dir, tc.w, tc.h)

			data, _ := convertToDump(t, dir, in, "-m", fmt.Sprint(tc.mode))

			want := 128 + cpc.BitmapSize(tc.cols, tc.lines)
			if len(data) != want {
				t.Errorf("dump is %d bytes, want %d for a %dx%d source in mode %d",
					len(data), want, tc.w, tc.h, tc.mode)
			}
		})
	}
}

// TestCLIConvertGeometryAutoFallback reports itself: -v says which scan area was
// used and, when nothing matched, that a standard screen was assumed.
func TestCLIConvertGeometryAutoFallback(t *testing.T) {
	dir := t.TempDir()
	in := writeScreenSource(t, dir, 200, 150)

	_, quiet := convertToDump(t, dir, in, "-m", "1")
	if quiet != "" {
		t.Errorf("a plain run printed %q, want nothing", quiet)
	}

	data, verbose := convertToDump(t, dir, in, "-m", "1", "-v")
	if len(data) != 128+cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines) {
		t.Errorf("fallback dump is %d bytes, want the standard one (%d)",
			len(data), 128+cpc.BitmapSize(cpc.StandardCols, cpc.StandardLines))
	}
	if !strings.Contains(verbose, "No screen size matched 200x150") {
		t.Errorf("verbose output %q does not report the fallback", verbose)
	}
	if !strings.Contains(verbose, "standard") {
		t.Errorf("verbose output %q does not name the scan area used", verbose)
	}
}

// TestCLIConvertUnderscanWritesAReadableScreen closes the gap the reading side
// had left open: an underscan dump this tool writes must carry its palette where
// a raw underscan screen keeps it and must render back at the underscan grid.
func TestCLIConvertUnderscanWritesAReadableScreen(t *testing.T) {
	dir := t.TempDir()
	in := writeScreenSource(t, dir, 256, 192) // the grid a mode-1 underscan PNG is emitted at

	dumpPath := filepath.Join(dir, "screen.scr")
	dump, _ := convertToDump(t, dir, in, "-m", "1")
	if want := 128 + cpc.BitmapSize(cpc.UnderscanCols, cpc.UnderscanLines); len(dump) != want {
		t.Fatalf("underscan dump is %d bytes, want %d", len(dump), want)
	}

	// The palette sits where a raw underscan screen keeps it (&17D0, the place
	// the reading side was verified against the corpus).
	block := modePalOf(t, dumpPath)
	nonZero := false
	for _, b := range block {
		if b != 0 {
			nonZero = true
			break
		}
	}
	if !nonZero {
		t.Error("underscan dump carries an all-zero palette block")
	}

	// And it renders back at the emitted underscan grid.
	back := filepath.Join(dir, "back.png")
	resetConvertFlags(t)
	defer resetConvertFlags(t)
	rootCmd.SetArgs([]string{"convert", "-i", dumpPath, "-o", back})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("rendering the underscan dump back failed: %v", err)
	}
	f, err := os.Open(back)
	if err != nil {
		t.Fatalf("open %s: %v", back, err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", back, err)
	}
	if w, h := scrOutputSize(cpc.UnderscanCols, cpc.UnderscanLines, 1); img.Bounds().Dx() != w || img.Bounds().Dy() != h {
		t.Errorf("rendered dump is %dx%d, want %dx%d",
			img.Bounds().Dx(), img.Bounds().Dy(), w, h)
	}
}
