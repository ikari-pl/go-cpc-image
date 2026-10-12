package main

import (
	"testing"

	"github.com/ikari-pl/go-cpc-image/pkg/cpc"
)

// testGeometries are the three scan areas, so a density test can walk all of them
// and show the density does not depend on which one a screen has.
var testGeometries = []struct {
	name        string
	cols, lines int
}{
	{"standard", cpc.StandardCols, cpc.StandardLines},
	{"underscan", cpc.UnderscanCols, cpc.UnderscanLines},
	{"overscan", cpc.OverscanCols, cpc.OverscanLines},
}

// TestPixelDensity pins the pair every mode and every output policy declares, in
// each of the three scan areas: a CPC pixel is one size, so the density is a
// property of the mode and of the grid, never of the border around the picture.
func TestPixelDensity(t *testing.T) {
	for _, tc := range []struct {
		name         string
		mode         int
		native       bool
		scale        int
		wantX, wantY uint32
	}{
		{"mode 0 default (its pixels duplicated across)", 0, false, 1, 1416, 1416},
		{"mode 0 native (the mode's own pair)", 0, true, 1, 708, 1416},
		{"mode 0 default scaled 2x", 0, false, 2, 2832, 2832},
		{"mode 0 native scaled 3x", 0, true, 3, 2124, 4248},
		{"mode 1 default", 1, false, 1, 1416, 1416},
		{"mode 1 native (already the same grid)", 1, true, 1, 1416, 1416},
		{"mode 1 default scaled 2x", 1, false, 2, 2832, 2832},
		{"mode 2 default (whole canvas: two rows per CPC line)", 2, false, 1, 2832, 2832},
		{"mode 2 native (the mode's own pair)", 2, true, 1, 2832, 1416},
		{"mode 2 default scaled 2x", 2, false, 2, 5664, 5664},
		{"mode 2 native scaled 2x", 2, true, 2, 5664, 2832},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, g := range testGeometries {
				gotX, gotY := pixelDensity(g.cols, g.lines, tc.mode, tc.native, tc.scale)
				if gotX != tc.wantX || gotY != tc.wantY {
					t.Errorf("%s: density = %d x %d pixels/meter, want %d x %d",
						g.name, gotX, gotY, tc.wantX, tc.wantY)
				}
			}
		})
	}
}

// TestPixelDensityMatchesEmittedGrid ties the declared pair to the grid the file
// really holds, in both directions: the duplication the density assumes (two file
// pixels per CPC pixel across in mode 0, two rows per CPC line in mode 2) has to be
// the duplication screenOutputSize applies, and the density has to scale with the
// emitted size exactly — one moving without the other cannot slip through.
func TestPixelDensityMatchesEmittedGrid(t *testing.T) {
	for _, g := range testGeometries {
		for mode := 0; mode <= 2; mode++ {
			natW, natH := g.cols<<(mode+1), g.lines
			baseX, baseY := pixelDensityBase(mode)
			for _, native := range []bool{false, true} {
				for _, scale := range []int{1, 2, 3} {
					dupX, dupY := pixelDuplication(mode, native)
					outW, outH := screenOutputSize(g.cols, g.lines, mode, native, scale)
					wantW := natW * int(dupX) * scale
					wantH := natH * int(dupY) * scale
					if outW != wantW || outH != wantH {
						t.Errorf("%s mode %d native=%v scale %d: emitted %dx%d, but the "+
							"density assumes %dx%d", g.name, mode, native, scale,
							outW, outH, wantW, wantH)
						continue
					}

					gotX, gotY := pixelDensity(g.cols, g.lines, mode, native, scale)
					// The same picture, said either way: the file's density times the
					// CPC pixel count is the pixel's density times the file's count.
					if gotX*uint32(natW) != baseX*uint32(outW) ||
						gotY*uint32(natH) != baseY*uint32(outH) {
						t.Errorf("%s mode %d native=%v scale %d: density %d x %d does not "+
							"scale with the %dx%d grid from a %d x %d pixel",
							g.name, mode, native, scale, gotX, gotY, outW, outH, baseX, baseY)
					}
				}
			}
		}
	}
}

// TestDensityInDPI pins the rounding the -v line reports, including the reference
// densities of every mode.
func TestDensityInDPI(t *testing.T) {
	for _, tc := range []struct {
		perMetre, want uint32
	}{
		{708, 18}, {1416, 36}, {2124, 54}, {2832, 72}, {4248, 108}, {5664, 144},
	} {
		if got := densityInDPI(tc.perMetre); got != tc.want {
			t.Errorf("densityInDPI(%d) = %d, want %d", tc.perMetre, got, tc.want)
		}
	}
}
