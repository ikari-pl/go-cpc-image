package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// screenPNGSize renders a screen dump back to PNG with the given output knobs and
// returns the size of the picture it wrote.
func screenPNGSize(t *testing.T, dumpPath, dir string, args ...string) (int, int) {
	t.Helper()
	out := filepath.Join(dir, "out.png")
	resetConvertFlags(t)
	defer resetConvertFlags(t)
	rootCmd.SetArgs(append([]string{"convert", "-i", dumpPath, "-o", out}, args...))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("rendering %s with %v failed: %v", dumpPath, args, err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatalf("open %s: %v", out, err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", out, err)
	}
	return img.Bounds().Dx(), img.Bounds().Dy()
}

// screenPNG renders a screen dump back to PNG with the given output knobs and
// returns the decoded picture, so a test can compare two renders pixel by pixel.
func screenPNG(t *testing.T, dumpPath, dir, name string, args ...string) image.Image {
	t.Helper()
	out := filepath.Join(dir, name)
	resetConvertFlags(t)
	defer resetConvertFlags(t)
	rootCmd.SetArgs(append([]string{"convert", "-i", dumpPath, "-o", out}, args...))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("rendering %s with %v failed: %v", dumpPath, args, err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatalf("open %s: %v", out, err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", out, err)
	}
	return img
}

// TestCLIConvertScreenOutputKnobs pins the two output knobs against the sizes the
// design fixes: by default a screen PNG is written on the PAR-pre-compensated
// grid, --native writes the screen's own pixels (no pre-compensation, so a grid
// that carries no aspect correction of its own apart from the density it declares)
// and --scale zooms whatever the previous step produced by a whole factor. The
// expected numbers are the table the design states, written out here rather than
// recomputed.
func TestCLIConvertScreenOutputKnobs(t *testing.T) {
	cases := []struct {
		name       string
		mode       int
		geometry   string
		defW, defH int // the PAR grid: what a screen PNG is written at by default
		natW, natH int // the screen's own pixels
	}{
		{"mode 0 standard", 0, "standard", 320, 200, 160, 200},
		{"mode 0 underscan", 0, "underscan", 256, 192, 128, 192},
		{"mode 1 standard", 1, "standard", 320, 200, 320, 200},
		{"mode 1 overscan", 1, "overscan", 384, 272, 384, 272},
		{"mode 2 overscan", 2, "overscan", 768, 544, 768, 272},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			src := writeScreenSource(t, dir, 40, 40)
			dump := filepath.Join(dir, "screen.scr")

			resetConvertFlags(t)
			rootCmd.SetArgs([]string{"convert", "-i", src, "-o", dump,
				"-m", fmt.Sprint(tc.mode), "--geometry", tc.geometry})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("writing the %s dump failed: %v", tc.geometry, err)
			}
			resetConvertFlags(t)

			knobs := []struct {
				name string
				args []string
				w, h int
			}{
				{"default", nil, tc.defW, tc.defH},
				{"--native", []string{"--native"}, tc.natW, tc.natH},
				{"--scale 3", []string{"--scale", "3"}, tc.defW * 3, tc.defH * 3},
				{"--native --scale 2", []string{"--native", "--scale", "2"}, tc.natW * 2, tc.natH * 2},
			}
			for _, k := range knobs {
				w, h := screenPNGSize(t, dump, dir, k.args...)
				if w != k.w || h != k.h {
					t.Errorf("%s: wrote %dx%d, want %dx%d", k.name, w, h, k.w, k.h)
				}
			}
		})
	}
}

// TestCLIConvertScaleValidation checks the integer zoom refuses a factor that
// would mean nothing: a fractional or zero zoom would mash pixels.
func TestCLIConvertScaleValidation(t *testing.T) {
	dir := t.TempDir()
	src := writeScreenSource(t, dir, 40, 40)
	out := filepath.Join(dir, "screen.scr")

	for _, factor := range []string{"0", "-2"} {
		resetConvertFlags(t)
		rootCmd.SetArgs([]string{"convert", "-i", src, "-o", out, "--scale", factor})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("--scale %s was accepted, want an error", factor)
		}
		if !strings.Contains(err.Error(), "scale") {
			t.Errorf("--scale %s error %q does not mention the scale", factor, err)
		}
	}
	resetConvertFlags(t)
}

// TestCLIConvertEmitsTheNativeGridMagnified pins what the output stage does and
// nothing else: a screen PNG holds the screen's own grid with whole pixels
// duplicated — the mode's PAR pair — so the default file is exactly the --native
// one magnified by that pair (the factor cancels out of --scale, since both sides
// carry it). Comparing pixels, not just the sizes, is what shows a renderer that
// drew a magnified canvas and had it decimated on the way out.
func TestCLIConvertEmitsTheNativeGridMagnified(t *testing.T) {
	cases := []struct {
		name     string
		mode     int
		geometry string
		scale    int
		magX     int // the duplication the default grid applies to the native one
		magY     int
	}{
		{"mode 0 standard", 0, "standard", 1, 2, 1},
		{"mode 0 standard scaled 2", 0, "standard", 2, 2, 1},
		{"mode 1 underscan", 1, "underscan", 1, 1, 1},
		{"mode 2 standard", 2, "standard", 1, 1, 2},
		{"mode 2 overscan scaled 2", 2, "overscan", 2, 1, 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			src := writeScreenSource(t, dir, 60, 60)
			dump := filepath.Join(dir, "screen.scr")

			resetConvertFlags(t)
			rootCmd.SetArgs([]string{"convert", "-i", src, "-o", dump,
				"-m", fmt.Sprint(tc.mode), "--geometry", tc.geometry})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("writing the %s dump failed: %v", tc.geometry, err)
			}
			resetConvertFlags(t)

			scale := fmt.Sprint(tc.scale)
			native := screenPNG(t, dump, dir, "native.png", "--native", "--scale", scale)
			def := screenPNG(t, dump, dir, "default.png", "--scale", scale)

			nb := native.Bounds()
			db := def.Bounds()
			if db.Dx() != nb.Dx()*tc.magX || db.Dy() != nb.Dy()*tc.magY {
				t.Fatalf("default = %dx%d, want the %dx%d native grid x %d,%d",
					db.Dx(), db.Dy(), nb.Dx(), nb.Dy(), tc.magX, tc.magY)
			}

			for y := 0; y < db.Dy(); y++ {
				for x := 0; x < db.Dx(); x++ {
					got := color.RGBAModel.Convert(def.At(db.Min.X+x, db.Min.Y+y)).(color.RGBA)
					want := color.RGBAModel.Convert(
						native.At(nb.Min.X+x/tc.magX, nb.Min.Y+y/tc.magY)).(color.RGBA)
					if got != want {
						t.Fatalf("default (%d,%d) = %v, want the native pixel (%d,%d) = %v",
							x, y, got, x/tc.magX, y/tc.magY, want)
					}
				}
			}
		})
	}
}
