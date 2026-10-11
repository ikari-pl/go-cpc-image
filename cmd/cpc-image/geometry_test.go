package main

import "testing"

// The geometry a PNG->SCR conversion writes is one of three scan areas, and
// `--geometry auto` works it out from the input's pixel size, because that is
// the only signal a bitmap carries. These tests pin the rule the flag's help
// states: for the mode the user gave, a size matches the canvas the renderer
// draws, the grid this tool emits with scrOutputSize, or the screen's own native
// grid — and those sizes never name two different scan areas, so `auto` chooses
// exactly instead of guessing.

// geometryFamilies lists the sizes that identify a scan area in a mode: the
// canvas (cols*8 x lines*2, mode independent), the grid this tool emits and the
// screen's own native picture.
func geometryFamilies(g screenGeometry, mode int) map[string][2]int {
	emittedW, emittedH := scrOutputSize(g.cols, g.lines, mode)
	return map[string][2]int{
		"canvas":  {g.cols * 8, g.lines * 2},
		"emitted": {emittedW, emittedH},
		"native":  {g.cols << (mode + 1), g.lines},
	}
}

func TestGeometryForSizeRuleTable(t *testing.T) {
	all := []screenGeometry{geomStandard, geomUnderscan, geomOverscan}

	for mode := 0; mode <= 2; mode++ {
		named := map[[2]int]string{}
		for _, g := range all {
			for family, size := range geometryFamilies(g, mode) {
				if prev, ok := named[size]; ok && prev != g.name {
					t.Fatalf("mode %d: %dx%d names both %s and %s", mode, size[0], size[1], prev, g.name)
				}
				named[size] = g.name

				got, matched := geometryForSize(mode, size[0], size[1])
				if !matched {
					t.Errorf("mode %d: %s size %dx%d (%s) matched nothing",
						mode, g.name, size[0], size[1], family)
					continue
				}
				if got.name != g.name {
					t.Errorf("mode %d: %s size %dx%d (%s) -> %s",
						mode, g.name, size[0], size[1], family, got.name)
				}
			}
		}
	}

	// A picture that is no screen of any shape must not match: the fallback is
	// what gives it a mapping, and only an exact hit may choose a scan area.
	for _, size := range [][2]int{{0, 0}, {200, 150}, {319, 199}, {63, 63}, {641, 400}} {
		for mode := 0; mode <= 2; mode++ {
			if got, ok := geometryForSize(mode, size[0], size[1]); ok {
				t.Errorf("mode %d: %dx%d matched %s, want no match",
					mode, size[0], size[1], got.name)
			}
		}
	}
}

func TestParseGeometryFlag(t *testing.T) {
	cases := []struct {
		name     string
		value    string
		valueSet bool
		alias    bool
		want     string
		wantErr  bool
	}{
		{"untouched defaults to auto", "auto", false, false, "auto", false},
		{"explicit standard", "standard", true, false, "standard", false},
		{"explicit underscan", "underscan", true, false, "underscan", false},
		{"value is case insensitive", "OverScan", true, false, "overscan", false},
		{"alias alone selects overscan", "auto", false, true, "overscan", false},
		{"alias redundant with overscan", "overscan", true, true, "overscan", false},
		{"alias with another geometry is an error", "underscan", true, true, "", true},
		{"alias with an explicit auto is an error", "auto", true, true, "", true},
		{"unknown value", "bigscreen", true, false, "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseGeometry(tc.value, tc.valueSet, tc.alias)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseGeometry(%q, set=%v, alias=%v) = %q, want an error",
						tc.value, tc.valueSet, tc.alias, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseGeometry(%q, set=%v, alias=%v) failed: %v",
					tc.value, tc.valueSet, tc.alias, err)
			}
			if got != tc.want {
				t.Errorf("parseGeometry(%q, set=%v, alias=%v) = %q, want %q",
					tc.value, tc.valueSet, tc.alias, got, tc.want)
			}
		})
	}
}
