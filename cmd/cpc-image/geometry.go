package main

import (
	"fmt"
	"strings"

	"github.com/ikari-pl/go-cpc-image/pkg/cpc"
)

// screenGeometry is the scan area a conversion writes: the bytes of one CRTC
// line and the number of lines. `--geometry` selects one, and `auto` works it out
// from the input picture's pixel size, which is the only signal a bitmap carries.
type screenGeometry struct {
	name  string
	cols  int
	lines int
}

// The three scan areas the CLI knows. The numbers come from pkg/cpc, so a screen
// written here is the same shape the rest of the tree reads, packs and unpacks.
var (
	geomStandard  = screenGeometry{"standard", cpc.StandardCols, cpc.StandardLines}
	geomUnderscan = screenGeometry{"underscan", cpc.UnderscanCols, cpc.UnderscanLines}
	geomOverscan  = screenGeometry{"overscan", cpc.OverscanCols, cpc.OverscanLines}
)

// geometryNames lists the accepted --geometry values, in the order the help and
// the error message print them.
var geometryNames = []string{"auto", "standard", "underscan", "overscan"}

// parseGeometry validates the --geometry value and folds in --overscan, which is
// kept as an alias of `--geometry=overscan` for backwards compatibility. valueSet
// says whether the user passed --geometry at all: the alias alone keeps working,
// an explicitly different geometry is an error (a wrong batch line must not win by
// precedence) and `--overscan --geometry=overscan` is accepted as redundant.
func parseGeometry(value string, valueSet, overscanAlias bool) (string, error) {
	if overscanAlias && !valueSet {
		return "overscan", nil
	}

	want := strings.ToLower(strings.TrimSpace(value))
	switch want {
	case "auto", "standard", "underscan", "overscan":
	default:
		return "", fmt.Errorf("invalid geometry: %s (available: %s)",
			value, strings.Join(geometryNames, ", "))
	}

	if overscanAlias && want != "overscan" {
		return "", fmt.Errorf("--overscan means --geometry=overscan, which conflicts with --geometry=%s", value)
	}
	return want, nil
}

// geometryForSize names the scan area an input of srcW x srcH belongs to, for the
// mode the user gave. Only an exact hit counts, in this order:
//
//  1. the canvas the renderer draws (cols*8 x lines*2), which does not depend on
//     the mode: 640x400 standard, 512x384 underscan, 768x544 overscan;
//  2. the grid this tool emits a screen PNG with (scrOutputSize): 320x200,
//     256x192 and 384x272 for modes 0 and 1, the canvases above for mode 2;
//  3. the screen's own native picture (cols<<(mode+1) x lines): 160x200, 128x192
//     and 192x272 for mode 0; 320x200, 256x192 and 384x272 for mode 1; 640x200,
//     512x192 and 768x272 for mode 2.
//
// The three families never name two different scan areas inside a mode (pinned by
// TestGeometryForSizeRuleTable), so the answer is never ambiguous — which is what
// keeps `auto` from being the kind of guessing this project avoids elsewhere.
// Rules 1-3 are also exactly the sizes this tool writes, so re-converting a PNG
// it produced returns the geometry that PNG came from. A size that matches
// nothing reports matched=false, and the caller falls back to a standard screen.
func geometryForSize(mode, srcW, srcH int) (screenGeometry, bool) {
	if mode < 0 || mode > 2 {
		return screenGeometry{}, false
	}
	all := []screenGeometry{geomStandard, geomUnderscan, geomOverscan}

	for _, g := range all {
		if srcW == g.cols*8 && srcH == g.lines*2 {
			return g, true
		}
	}
	for _, g := range all {
		if w, h := scrOutputSize(g.cols, g.lines, mode); srcW == w && srcH == h {
			return g, true
		}
	}
	for _, g := range all {
		if srcW == g.cols<<(mode+1) && srcH == g.lines {
			return g, true
		}
	}
	return screenGeometry{}, false
}
