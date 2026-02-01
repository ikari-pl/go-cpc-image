package animation

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"strings"
	"testing"

	"github.com/ikari/go-cpc-image/pkg/bitmap"
)

// ── DeltaDiff tests ──

func TestNewDeltaDiff(t *testing.T) {
	dd := NewDeltaDiff(0x4000)
	if dd.StartAddress != 0x4000 {
		t.Errorf("expected StartAddress 0x4000, got 0x%04X", dd.StartAddress)
	}
	if dd.Length() != 0 {
		t.Errorf("expected empty, got length %d", dd.Length())
	}
}

func TestDeltaDiff_AddValue(t *testing.T) {
	dd := NewDeltaDiff(0)
	dd.AddValue(0xAA)
	dd.AddValue(0xBB)
	if dd.Length() != 2 {
		t.Fatalf("expected length 2, got %d", dd.Length())
	}
	if dd.LstDelta[0] != 0xAA || dd.LstDelta[1] != 0xBB {
		t.Errorf("unexpected values: %v", dd.LstDelta)
	}
}

// ── DiffAnim tests ──

func TestNewDiffAnim(t *testing.T) {
	da := NewDiffAnim()
	if da.GetBlockCount() != 0 {
		t.Errorf("expected 0 blocks, got %d", da.GetBlockCount())
	}
	if da.GetTotalBytes() != 0 {
		t.Errorf("expected 0 total bytes, got %d", da.GetTotalBytes())
	}
}

func TestDiffAnim_AddDiff_Consecutive(t *testing.T) {
	da := NewDiffAnim()
	da.AddDiff(100, 0x01)
	da.AddDiff(101, 0x02)
	da.AddDiff(102, 0x03)

	if da.GetBlockCount() != 1 {
		t.Fatalf("expected 1 block for consecutive addresses, got %d", da.GetBlockCount())
	}
	if da.GetTotalBytes() != 3 {
		t.Errorf("expected 3 total bytes, got %d", da.GetTotalBytes())
	}
}

func TestDiffAnim_AddDiff_NonConsecutive(t *testing.T) {
	da := NewDiffAnim()
	da.AddDiff(100, 0x01)
	da.AddDiff(200, 0x02) // gap

	if da.GetBlockCount() != 2 {
		t.Fatalf("expected 2 blocks, got %d", da.GetBlockCount())
	}
	if da.Delta[0].StartAddress != 100 {
		t.Errorf("block 0 start address: expected 100, got %d", da.Delta[0].StartAddress)
	}
	if da.Delta[1].StartAddress != 200 {
		t.Errorf("block 1 start address: expected 200, got %d", da.Delta[1].StartAddress)
	}
}

func TestDiffAnim_Clear(t *testing.T) {
	da := NewDiffAnim()
	da.AddDiff(0, 0xFF)
	da.Clear()
	if da.GetBlockCount() != 0 {
		t.Errorf("expected 0 blocks after clear, got %d", da.GetBlockCount())
	}
}

func TestDiffAnim_CompareFrames(t *testing.T) {
	frame1 := []byte{0x00, 0x01, 0x02, 0x03}
	frame2 := []byte{0x00, 0xFF, 0x02, 0xEE}

	da := NewDiffAnim()
	da.CompareFrames(frame1, frame2, 0xC000)

	if da.GetTotalBytes() != 2 {
		t.Fatalf("expected 2 changed bytes, got %d", da.GetTotalBytes())
	}
}

func TestDiffAnim_CompareFrames_Frame2Longer(t *testing.T) {
	frame1 := []byte{0x00}
	frame2 := []byte{0x00, 0xAA, 0xBB}

	da := NewDiffAnim()
	da.CompareFrames(frame1, frame2, 0)

	if da.GetTotalBytes() != 2 {
		t.Fatalf("expected 2 extra bytes, got %d", da.GetTotalBytes())
	}
}

func TestDiffAnim_CompareFrames_Identical(t *testing.T) {
	frame := []byte{0x01, 0x02, 0x03}
	da := NewDiffAnim()
	da.CompareFrames(frame, frame, 0)

	if da.GetBlockCount() != 0 {
		t.Errorf("expected 0 blocks for identical frames, got %d", da.GetBlockCount())
	}
}

func TestApplyDelta(t *testing.T) {
	base := []byte{0x00, 0x00, 0x00, 0x00}
	da := NewDiffAnim()
	da.AddDiff(1, 0xAA)
	da.AddDiff(2, 0xBB)

	result := ApplyDelta(base, da, 0)
	expected := []byte{0x00, 0xAA, 0xBB, 0x00}
	if !bytes.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
	// Verify base is unchanged
	if base[1] != 0x00 {
		t.Error("base frame was modified")
	}
}

func TestApplyDelta_OutOfRange(t *testing.T) {
	base := []byte{0x00, 0x00}
	da := NewDiffAnim()
	da.AddDiff(100, 0xFF) // way past base address + len

	result := ApplyDelta(base, da, 0)
	if !bytes.Equal(result, base) {
		t.Errorf("expected unchanged base, got %v", result)
	}
}

func TestDiffAnim_SaveBinary(t *testing.T) {
	da := NewDiffAnim()
	da.AddDiff(0x1234, 0xAA)
	da.AddDiff(0x1235, 0xBB)

	bin := da.SaveBinary()
	// Expected: addr_lo, addr_hi, length, data..., 0x00, 0x00 end marker
	expected := []byte{0x34, 0x12, 0x02, 0xAA, 0xBB, 0x00, 0x00}
	if !bytes.Equal(bin, expected) {
		t.Errorf("expected %v, got %v", expected, bin)
	}
}

func TestDiffAnim_SaveBinary_Empty(t *testing.T) {
	da := NewDiffAnim()
	bin := da.SaveBinary()
	if len(bin) != 0 {
		t.Errorf("expected empty binary for no blocks, got %v", bin)
	}
}

func TestDiffAnim_Save(t *testing.T) {
	da := NewDiffAnim()
	da.AddDiff(0x4000, 0xAA)

	var buf bytes.Buffer
	err := da.Save(&buf, 16)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "#4000") {
		t.Error("output should contain address #4000")
	}
	if !strings.Contains(output, "#AA") {
		t.Error("output should contain data #AA")
	}
	if !strings.Contains(output, "#0000") {
		t.Error("output should contain end marker")
	}
}

func TestDiffAnim_Save_Empty(t *testing.T) {
	da := NewDiffAnim()
	var buf bytes.Buffer
	err := da.Save(&buf, 16)
	if err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}

func TestDiffAnim_Save_MultiplePerLine(t *testing.T) {
	da := NewDiffAnim()
	// Add enough bytes to span multiple lines
	for i := 0; i < 5; i++ {
		da.AddDiff(i, byte(i))
	}

	var buf bytes.Buffer
	err := da.Save(&buf, 3) // 3 bytes per line
	if err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	// Should have at least 2 DB lines for data
	count := strings.Count(output, "\tDB\t")
	if count < 3 { // address DB + at least 2 data DB lines
		t.Errorf("expected at least 3 DB directives, got %d in:\n%s", count, output)
	}
}

func TestDiffAnim_OptimizeDelta(t *testing.T) {
	da := NewDiffAnim()
	da.AddDiff(100, 0x01)
	da.AddDiff(103, 0x02) // gap of 2

	da.OptimizeDelta(5) // maxGap=5, should merge
	if da.GetBlockCount() != 1 {
		t.Fatalf("expected 1 merged block, got %d", da.GetBlockCount())
	}
	// Block should be: 0x01, 0x00, 0x00, 0x02 (gap filled with zeros)
	if da.GetTotalBytes() != 4 {
		t.Errorf("expected 4 total bytes after merge, got %d", da.GetTotalBytes())
	}
}

func TestDiffAnim_OptimizeDelta_NoMerge(t *testing.T) {
	da := NewDiffAnim()
	da.AddDiff(100, 0x01)
	da.AddDiff(200, 0x02)

	da.OptimizeDelta(2) // maxGap=2, too small
	if da.GetBlockCount() != 2 {
		t.Errorf("expected 2 blocks (no merge), got %d", da.GetBlockCount())
	}
}

func TestDiffAnim_OptimizeDelta_Single(t *testing.T) {
	da := NewDiffAnim()
	da.AddDiff(100, 0x01)
	da.OptimizeDelta(5)
	if da.GetBlockCount() != 1 {
		t.Errorf("expected 1 block, got %d", da.GetBlockCount())
	}
}

func TestDiffAnim_OptimizeDelta_Empty(t *testing.T) {
	da := NewDiffAnim()
	da.OptimizeDelta(5) // should not panic
	if da.GetBlockCount() != 0 {
		t.Errorf("expected 0 blocks, got %d", da.GetBlockCount())
	}
}

// ── Roundtrip: CompareFrames -> ApplyDelta ──

func TestCompareAndApplyRoundtrip(t *testing.T) {
	frame1 := []byte{0x10, 0x20, 0x30, 0x40, 0x50}
	frame2 := []byte{0x10, 0xFF, 0x30, 0xEE, 0x50}

	da := NewDiffAnim()
	da.CompareFrames(frame1, frame2, 0)

	result := ApplyDelta(frame1, da, 0)
	if !bytes.Equal(result, frame2) {
		t.Errorf("roundtrip failed: expected %v, got %v", frame2, result)
	}
}

// ── ImageSource tests ──

func makeBitmap(w, h int) *bitmap.DirectBitmap {
	return bitmap.NewDirectBitmap(w, h)
}

func TestNewImageSource(t *testing.T) {
	is := NewImageSource()
	if is.NbImg() != 0 {
		t.Errorf("expected 0 frames, got %d", is.NbImg())
	}
	if is.GetImage() != nil {
		t.Error("expected nil image for empty source")
	}
}

func TestImageSource_Init(t *testing.T) {
	is := NewImageSource()
	is.Init()
	if is.NbImg() != 1 {
		t.Fatalf("expected 1 frame after Init, got %d", is.NbImg())
	}
	w, h := is.GetFrameSize()
	if w != 768 || h != 544 {
		t.Errorf("expected 768x544, got %dx%d", w, h)
	}
}

func TestImageSource_ClearAll(t *testing.T) {
	is := NewImageSource()
	is.Init()
	is.ClearAll()
	if is.NbImg() != 0 {
		t.Errorf("expected 0 after ClearAll, got %d", is.NbImg())
	}
}

func TestImageSource_InitBitmap(t *testing.T) {
	is := NewImageSource()
	bmp := makeBitmap(10, 10)
	is.InitBitmap(bmp)
	if is.NbImg() != 1 {
		t.Fatalf("expected 1 frame, got %d", is.NbImg())
	}
}

func TestImageSource_InitFrameCount(t *testing.T) {
	is := NewImageSource()
	is.InitFrameCount(5)
	if is.NbImg() != 5 {
		t.Errorf("expected 5 frames, got %d", is.NbImg())
	}
}

func TestImageSource_GetImage_Clamped(t *testing.T) {
	is := NewImageSource()
	is.InitFrameCount(2)
	is.imageSel = 99 // out of range
	img := is.GetImage()
	if img == nil {
		t.Error("expected non-nil image (clamped to last)")
	}
}

func TestImageSource_SelectBitmap(t *testing.T) {
	is := NewImageSource()
	is.InitFrameCount(3)
	is.SelectBitmap(2)
	if is.GetSelectedIndex() != 2 {
		t.Errorf("expected selected index 2, got %d", is.GetSelectedIndex())
	}
	is.SelectBitmap(99) // out of range, should not change
	if is.GetSelectedIndex() != 2 {
		t.Errorf("expected selected index still 2, got %d", is.GetSelectedIndex())
	}
	is.SelectBitmap(-1) // negative, should not change
	if is.GetSelectedIndex() != 2 {
		t.Errorf("expected selected index still 2, got %d", is.GetSelectedIndex())
	}
}

func TestImageSource_AddFrame(t *testing.T) {
	is := NewImageSource()
	is.AddFrame(makeBitmap(4, 4))
	is.AddFrame(makeBitmap(4, 4))
	if is.NbImg() != 2 {
		t.Errorf("expected 2 frames, got %d", is.NbImg())
	}
}

func TestImageSource_ImportBitmap(t *testing.T) {
	is := NewImageSource()
	is.InitFrameCount(2)
	bmp := makeBitmap(10, 10)
	err := is.ImportBitmap(bmp, 1)
	if err != nil {
		t.Fatal(err)
	}
	err = is.ImportBitmap(bmp, 5) // out of range
	if err == nil {
		t.Error("expected error for out of range index")
	}
	err = is.ImportBitmap(bmp, -1)
	if err == nil {
		t.Error("expected error for negative index")
	}
}

func TestImageSource_GetBitmap(t *testing.T) {
	is := NewImageSource()
	is.InitFrameCount(3)

	bmp := is.GetBitmap(1)
	if bmp == nil {
		t.Error("expected non-nil bitmap for index 1")
	}

	// Out of range returns last frame
	bmp = is.GetBitmap(99)
	if bmp == nil {
		t.Error("expected non-nil bitmap for out-of-range (returns last)")
	}

	// Empty source
	is2 := NewImageSource()
	if is2.GetBitmap(0) != nil {
		t.Error("expected nil for empty source")
	}
}

func TestImageSource_DeleteImage(t *testing.T) {
	is := NewImageSource()
	is.InitFrameCount(3)
	is.SelectBitmap(2)

	err := is.DeleteImage(2)
	if err != nil {
		t.Fatal(err)
	}
	if is.NbImg() != 2 {
		t.Errorf("expected 2 frames, got %d", is.NbImg())
	}
	// imageSel should be clamped
	if is.GetSelectedIndex() >= is.NbImg() {
		t.Error("selected index should be clamped")
	}

	err = is.DeleteImage(99)
	if err == nil {
		t.Error("expected error for out of range")
	}
}

func TestImageSource_DeleteImage_AllFrames(t *testing.T) {
	is := NewImageSource()
	is.InitFrameCount(1)
	err := is.DeleteImage(0)
	if err != nil {
		t.Fatal(err)
	}
	if is.NbImg() != 0 {
		t.Errorf("expected 0 frames, got %d", is.NbImg())
	}
	if is.GetSelectedIndex() != 0 {
		t.Errorf("expected selected index 0, got %d", is.GetSelectedIndex())
	}
}

func TestImageSource_FrameTime(t *testing.T) {
	is := NewImageSource()
	is.SetFrameTime(50)
	if is.GetFrameTime() != 50 {
		t.Errorf("expected 50, got %d", is.GetFrameTime())
	}
}

func TestImageSource_FrameRate(t *testing.T) {
	is := NewImageSource()
	is.SetFrameRate(10.0) // 10 FPS -> tpsFrame = 10
	rate := is.GetFrameRate()
	if rate < 9.0 || rate > 11.0 {
		t.Errorf("expected ~10 FPS, got %f", rate)
	}

	is.SetFrameRate(0)
	if is.GetFrameRate() != 0 {
		t.Errorf("expected 0 FPS, got %f", is.GetFrameRate())
	}

	is.SetFrameRate(-5)
	if is.GetFrameRate() != 0 {
		t.Errorf("expected 0 FPS for negative input, got %f", is.GetFrameRate())
	}
}

func TestImageSource_GetFrameDelay(t *testing.T) {
	is := NewImageSource()
	is.SetFrameTime(42)
	if is.GetFrameDelay(0) != 42 {
		t.Errorf("expected 42, got %d", is.GetFrameDelay(0))
	}
}

func TestImageSource_GetFrameSize_Empty(t *testing.T) {
	is := NewImageSource()
	w, h := is.GetFrameSize()
	if w != 0 || h != 0 {
		t.Errorf("expected 0x0 for empty, got %dx%d", w, h)
	}
}

func TestImageSource_GetAnimationInfo(t *testing.T) {
	is := NewImageSource()
	is.InitFrameCount(10)
	is.SetFrameRate(10.0)

	fc, fr, dur := is.GetAnimationInfo()
	if fc != 10 {
		t.Errorf("expected 10 frames, got %d", fc)
	}
	if fr < 9.0 || fr > 11.0 {
		t.Errorf("expected ~10 FPS, got %f", fr)
	}
	if dur < 0.9 || dur > 1.1 {
		t.Errorf("expected ~1.0s duration, got %f", dur)
	}
}

func TestImageSource_GetAnimationInfo_ZeroRate(t *testing.T) {
	is := NewImageSource()
	is.InitFrameCount(5)
	_, _, dur := is.GetAnimationInfo()
	if dur != 0 {
		t.Errorf("expected 0 duration with 0 rate, got %f", dur)
	}
}

func TestImageSource_Clone(t *testing.T) {
	is := NewImageSource()
	is.InitFrameCount(2)
	is.SetFrameTime(33)
	is.SelectBitmap(1)

	clone := is.Clone()
	if clone.NbImg() != is.NbImg() {
		t.Error("clone frame count mismatch")
	}
	if clone.GetFrameTime() != is.GetFrameTime() {
		t.Error("clone frame time mismatch")
	}
	if clone.GetSelectedIndex() != is.GetSelectedIndex() {
		t.Error("clone selected index mismatch")
	}

	// Verify it's a deep copy
	clone.SetFrameTime(99)
	if is.GetFrameTime() == 99 {
		t.Error("clone should be independent of original")
	}
}

func TestImageSource_ResizeAllFrames(t *testing.T) {
	is := NewImageSource()
	is.InitFrameCount(2)
	is.ResizeAllFrames(100, 50)

	w, h := is.GetFrameSize()
	if w != 100 || h != 50 {
		t.Errorf("expected 100x50, got %dx%d", w, h)
	}
}

func TestImageSource_SaveAsGIF(t *testing.T) {
	is := NewImageSource()
	var buf bytes.Buffer

	// Empty: should error
	err := is.SaveAsGIF(&buf)
	if err == nil {
		t.Error("expected error for empty source")
	}

	// With frames: should error (not implemented)
	is.InitFrameCount(1)
	err = is.SaveAsGIF(&buf)
	if err == nil {
		t.Error("expected error (not implemented)")
	}
}

// ── InitFromReader / GIF tests ──

func makeTestGIF(frames int, w, h int, delay int) *bytes.Buffer {
	g := &gif.GIF{}
	for i := 0; i < frames; i++ {
		img := image.NewPaletted(image.Rect(0, 0, w, h), color.Palette{
			color.RGBA{0, 0, 0, 255},
			color.RGBA{255, 0, 0, 255},
		})
		// Set some pixels
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				img.SetColorIndex(x, y, uint8((x+y+i)%2))
			}
		}
		g.Image = append(g.Image, img)
		g.Delay = append(g.Delay, delay)
	}
	g.LoopCount = 0

	var buf bytes.Buffer
	gif.EncodeAll(&buf, g)
	return &buf
}

func TestImageSource_InitFromReader_GIF(t *testing.T) {
	buf := makeTestGIF(3, 4, 4, 10)
	is := NewImageSource()
	err := is.InitFromReader(buf)
	if err != nil {
		t.Fatal(err)
	}
	if is.NbImg() != 3 {
		t.Errorf("expected 3 frames, got %d", is.NbImg())
	}
	if is.GetFrameTime() != 100 { // 10 * 10
		t.Errorf("expected frame time 100, got %d", is.GetFrameTime())
	}
}

func TestImageSource_InitFromReader_SingleFrame(t *testing.T) {
	buf := makeTestGIF(1, 2, 2, 0)
	is := NewImageSource()
	err := is.InitFromReader(buf)
	if err != nil {
		t.Fatal(err)
	}
	if is.NbImg() != 1 {
		t.Errorf("expected 1 frame, got %d", is.NbImg())
	}
	if is.GetFrameTime() != 0 {
		t.Errorf("expected 0 frame time for single frame, got %d", is.GetFrameTime())
	}
}

func TestImageSource_InitFromReader_Invalid(t *testing.T) {
	is := NewImageSource()
	err := is.InitFromReader(strings.NewReader("not an image"))
	if err == nil {
		t.Error("expected error for invalid data")
	}
}
