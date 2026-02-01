package splitscreen

import (
	"testing"
)

// --- Split tests ---

func TestNewSplit(t *testing.T) {
	s := NewSplit()
	if s.Length != 32 {
		t.Errorf("expected Length 32, got %d", s.Length)
	}
	if s.Color != 0 {
		t.Errorf("expected Color 0, got %d", s.Color)
	}
	if s.Enable != false {
		t.Error("expected Enable false")
	}
}

// --- SplitLine tests ---

func TestNewSplitLine(t *testing.T) {
	sl := NewSplitLine()
	if len(sl.SplitList) != MaxSplits {
		t.Errorf("expected %d splits, got %d", MaxSplits, len(sl.SplitList))
	}
	if sl.NumPen != 0 {
		t.Errorf("expected NumPen 0, got %d", sl.NumPen)
	}
	if sl.Delay != DelayMin {
		t.Errorf("expected Delay %d, got %d", DelayMin, sl.Delay)
	}
	if sl.NewMode != 0 {
		t.Errorf("expected NewMode 0, got %d", sl.NewMode)
	}
	if sl.ChangeMode {
		t.Error("expected ChangeMode false")
	}
	// Each split should have defaults
	for i, s := range sl.SplitList {
		if s.Length != 32 {
			t.Errorf("split %d: expected Length 32, got %d", i, s.Length)
		}
	}
}

func TestSplitLine_GetSplit_ExistingIndex(t *testing.T) {
	sl := NewSplitLine()
	s := sl.GetSplit(3)
	if s != sl.SplitList[3] {
		t.Error("GetSplit should return existing split")
	}
}

func TestSplitLine_GetSplit_ExtendsIfNeeded(t *testing.T) {
	sl := NewSplitLine()
	origLen := len(sl.SplitList)
	s := sl.GetSplit(origLen + 2)
	if s == nil {
		t.Fatal("GetSplit returned nil")
	}
	if len(sl.SplitList) != origLen+3 {
		t.Errorf("expected length %d, got %d", origLen+3, len(sl.SplitList))
	}
}

func TestSplitLine_AddSplit(t *testing.T) {
	sl := NewSplitLine()
	before := len(sl.SplitList)
	sl.AddSplit(64, 5, true)
	if len(sl.SplitList) != before+1 {
		t.Error("AddSplit should append")
	}
	last := sl.SplitList[len(sl.SplitList)-1]
	if last.Length != 64 || last.Color != 5 || !last.Enable {
		t.Errorf("unexpected split values: %+v", last)
	}
}

// --- SplitScreen tests ---

func TestNewSplitScreen(t *testing.T) {
	ss := NewSplitScreen()
	if len(ss.SplitLines) != 0 {
		t.Error("expected empty SplitLines initially")
	}
}

func TestSplitScreen_InitializeLines(t *testing.T) {
	ss := NewSplitScreen()
	ss.InitializeLines()
	if len(ss.SplitLines) != MaxLines {
		t.Errorf("expected %d lines, got %d", MaxLines, len(ss.SplitLines))
	}
	for i, l := range ss.SplitLines {
		if l == nil {
			t.Errorf("line %d is nil", i)
		}
	}
}

func TestSplitScreen_GetLine(t *testing.T) {
	ss := NewSplitScreen()
	ss.InitializeLines()

	// Valid index
	if ss.GetLine(0) == nil {
		t.Error("GetLine(0) should not be nil")
	}
	if ss.GetLine(MaxLines-1) == nil {
		t.Error("GetLine(MaxLines-1) should not be nil")
	}

	// Out of bounds
	if ss.GetLine(-1) != nil {
		t.Error("GetLine(-1) should be nil")
	}
	if ss.GetLine(MaxLines) != nil {
		t.Error("GetLine(MaxLines) should be nil")
	}
}

// --- BitmapCpc tests ---

func TestNewBitmapCpc(t *testing.T) {
	bmp := NewBitmapCpc(320, 200, 1)
	if bmp.Width != 320 || bmp.Height != 200 {
		t.Errorf("unexpected dimensions: %dx%d", bmp.Width, bmp.Height)
	}
	if len(bmp.ScreenData) != 0x10000 {
		t.Error("wrong ScreenData size")
	}
	if len(bmp.ImgAscii) != 0x1000 {
		t.Error("wrong ImgAscii size")
	}
	if len(bmp.ModeTable) != MaxLines {
		t.Error("wrong ModeTable size")
	}
	for y := 0; y < MaxLines; y++ {
		if bmp.ModeTable[y] != 1 {
			t.Errorf("ModeTable[%d] expected 1, got %d", y, bmp.ModeTable[y])
			break
		}
	}
	if bmp.IsComputed {
		t.Error("IsComputed should be false initially")
	}
	// Check default palette init
	if bmp.SplitPalette[0][0][5] != 5 {
		t.Error("default pen mapping wrong")
	}
	if bmp.SplitPalette[0][0][16] != 0 {
		t.Error("extra palette slot should be 0")
	}
}

func TestGetSetSplitPalette(t *testing.T) {
	bmp := NewBitmapCpc(320, 200, 0)

	bmp.SetSplitPalette(10, 20, 3, 42)
	if v := bmp.GetSplitPalette(10, 20, 3); v != 42 {
		t.Errorf("expected 42, got %d", v)
	}

	// Out of bounds should return 0 / be no-op
	if bmp.GetSplitPalette(-1, 0, 0) != 0 {
		t.Error("negative x should return 0")
	}
	if bmp.GetSplitPalette(0, -1, 0) != 0 {
		t.Error("negative y should return 0")
	}
	if bmp.GetSplitPalette(0, 0, -1) != 0 {
		t.Error("negative pen should return 0")
	}
	if bmp.GetSplitPalette(MaxCols, 0, 0) != 0 {
		t.Error("x=MaxCols should return 0")
	}
	if bmp.GetSplitPalette(0, MaxLines, 0) != 0 {
		t.Error("y=MaxLines should return 0")
	}
	if bmp.GetSplitPalette(0, 0, 17) != 0 {
		t.Error("pen=17 should return 0")
	}

	// SetSplitPalette out of bounds should not panic
	bmp.SetSplitPalette(-1, 0, 0, 1)
	bmp.SetSplitPalette(MaxCols, 0, 0, 1)
	bmp.SetSplitPalette(0, MaxLines, 0, 1)
	bmp.SetSplitPalette(0, 0, 17, 1)
}

func TestAppliquePalette(t *testing.T) {
	bmp := NewBitmapCpc(320, 200, 0)
	pal := make([]int, 16)
	for i := range pal {
		pal[i] = 100 + i
	}

	ret := bmp.AppliquePalette(0, 0, 5, pal)
	if ret != 5 {
		t.Errorf("expected return 5, got %d", ret)
	}
	for x := 0; x < 5; x++ {
		for pen := 0; pen < 16; pen++ {
			if bmp.SplitPalette[x][0][pen] != 100+pen {
				t.Errorf("palette not applied at x=%d pen=%d", x, pen)
				return
			}
		}
	}
	// x=5 should be unchanged (still default)
	if bmp.SplitPalette[5][0][0] != 0 {
		t.Error("x=5 should be unchanged")
	}
}

func TestAppliquePalette_ClampToMaxCols(t *testing.T) {
	bmp := NewBitmapCpc(320, 200, 0)
	pal := make([]int, 16)
	ret := bmp.AppliquePalette(90, 0, MaxCols+10, pal)
	if ret != MaxCols {
		t.Errorf("expected clamped to %d, got %d", MaxCols, ret)
	}
}

func TestCalcPaletteSplit(t *testing.T) {
	bmp := NewBitmapCpc(320, 100, 0)
	if bmp.IsComputed {
		t.Fatal("should not be computed yet")
	}
	bmp.CalcPaletteSplit()
	if !bmp.IsComputed {
		t.Error("IsComputed should be true after CalcPaletteSplit")
	}
}

func TestCalcPaletteSplit_WithEnabledSplits(t *testing.T) {
	bmp := NewBitmapCpc(320, 100, 0)
	line := bmp.SplitScreen.GetLine(0)
	if line == nil {
		t.Fatal("line 0 is nil")
	}
	// Enable a split
	line.SplitList[0].Enable = true
	line.SplitList[0].Color = 3
	line.SplitList[0].Length = 64

	bmp.CalcPaletteSplit()
	if !bmp.IsComputed {
		t.Error("should be computed")
	}
}

func TestOptimizeSplits(t *testing.T) {
	bmp := NewBitmapCpc(320, 200, 0)
	line := bmp.SplitScreen.GetLine(0)

	// Clear default splits and add custom ones
	line.SplitList = []*Split{
		{Length: 32, Color: 1, Enable: true},
		{Length: 32, Color: 1, Enable: true}, // same color, should merge
		{Length: 16, Color: 2, Enable: true},
		{Length: 0, Color: 3, Enable: true},   // zero length, should be removed
		{Length: 32, Color: 4, Enable: false},  // disabled, should be removed
	}

	bmp.OptimizeSplits()

	splits := line.SplitList
	if len(splits) != 2 {
		t.Fatalf("expected 2 splits after optimize, got %d", len(splits))
	}
	if splits[0].Color != 1 || splits[0].Length != 64 {
		t.Errorf("first split: expected color=1 length=64, got color=%d length=%d", splits[0].Color, splits[0].Length)
	}
	if splits[1].Color != 2 || splits[1].Length != 16 {
		t.Errorf("second split: expected color=2 length=16, got color=%d length=%d", splits[1].Color, splits[1].Length)
	}
}

func TestOptimizeSplits_NilLine(t *testing.T) {
	bmp := NewBitmapCpc(320, 200, 0)
	// Set a line to nil - should not panic
	bmp.SplitScreen.SplitLines[5] = nil
	bmp.OptimizeSplits() // should not panic
}

func TestOptimizeSplits_AllDisabled(t *testing.T) {
	bmp := NewBitmapCpc(320, 200, 0)
	// All default splits are disabled, so after optimize all lines should be empty
	bmp.OptimizeSplits()
	line := bmp.SplitScreen.GetLine(0)
	if len(line.SplitList) != 0 {
		t.Errorf("expected 0 splits, got %d", len(line.SplitList))
	}
}

// --- Constants tests ---

// TestSplitScreenConstants_MatchHardwareLimits checks that the package-level
// timing and dimension constants (DelayMin, MaxSplits, MaxLines, MaxCols)
// match the Amstrad CPC's hardware constraints for split-raster effects.
func TestSplitScreenConstants_MatchHardwareLimits(t *testing.T) {
	if DelayMin != -4 {
		t.Error("DelayMin wrong")
	}
	if MaxSplits != 7 {
		t.Error("MaxSplits wrong")
	}
	if MaxLines != 272 {
		t.Error("MaxLines wrong")
	}
	if MaxCols != 96 {
		t.Error("MaxCols wrong")
	}
}
