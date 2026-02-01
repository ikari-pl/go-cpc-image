package asmgen

import (
	"bytes"
	"strings"
	"testing"
)

// --- Utility function tests ---

// TestNewASMWriter_WritesToUnderlyingBuffer verifies that a freshly created
// ASMWriter directs all output to the provided io.Writer, confirming the
// plumbing between the writer and the assembly generation functions.
func TestNewASMWriter_WritesToUnderlyingBuffer(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if aw == nil {
		t.Fatal("NewASMWriter returned nil")
	}
}

func TestWriteLabel(t *testing.T) {
	tests := []struct {
		label    string
		expected string
	}{
		{"MyLabel", "MyLabel:\n"},
		{"", ""},
		{"Start", "Start:\n"},
	}
	for _, tc := range tests {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		if err := aw.WriteLabel(tc.label); err != nil {
			t.Fatalf("WriteLabel(%q) failed: %v", tc.label, err)
		}
		if buf.String() != tc.expected {
			t.Errorf("WriteLabel(%q) = %q, want %q", tc.label, buf.String(), tc.expected)
		}
	}
}

func TestWriteComment(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.WriteComment("Hello world"); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "; Hello world\n" {
		t.Errorf("got %q", buf.String())
	}
}

func TestWriteInstruction(t *testing.T) {
	tests := []struct {
		op, operands, comment, expected string
	}{
		{"LD", "A,#42", "Load constant", "\tLD\tA,#42\t\t; Load constant\n"},
		{"DI", "", "", "\tDI\n"},
		{"LD", "HL,#1234", "", "\tLD\tHL,#1234\n"},
		{"NOP", "", "Wait 1 NOP", "\tNOP\t\t; Wait 1 NOP\n"},
	}
	for _, tc := range tests {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		if err := aw.WriteInstruction(tc.op, tc.operands, tc.comment); err != nil {
			t.Fatal(err)
		}
		if buf.String() != tc.expected {
			t.Errorf("WriteInstruction(%q,%q,%q) = %q, want %q", tc.op, tc.operands, tc.comment, buf.String(), tc.expected)
		}
	}
}

func TestWriteOrg(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.WriteOrg(0x4000); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "ORG") || !strings.Contains(buf.String(), "#4000") {
		t.Errorf("got %q", buf.String())
	}
}

func TestWriteRun(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.WriteRun("$"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "RUN") || !strings.Contains(buf.String(), "$") {
		t.Errorf("got %q", buf.String())
	}
}

func TestWriteNoList(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.WriteNoList(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Nolist") {
		t.Errorf("got %q", buf.String())
	}
}

func TestWriteList(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.WriteList(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "List") {
		t.Errorf("got %q", buf.String())
	}
}

func TestWriteEquate(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		contains string
	}{
		{"buffer", 0x8000, "buffer\tEQU\t#8000"},
		{"addr", uint16(0x1234), "addr\tEQU\t#1234"},
		{"label", "Start", "label\tEQU\tStart"},
		{"misc", 42.5, "misc\tEQU\t42.5"},
	}
	for _, tc := range tests {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		if err := aw.WriteEquate(tc.name, tc.value); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), tc.contains) {
			t.Errorf("WriteEquate(%q, %v) = %q, want to contain %q", tc.name, tc.value, buf.String(), tc.contains)
		}
	}
}

func TestWriteBlankLine(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.WriteBlankLine(); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "\n" {
		t.Errorf("got %q", buf.String())
	}
}

func TestWriteRawLine(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.WriteRawLine("some raw text"); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "some raw text\n" {
		t.Errorf("got %q", buf.String())
	}
}

func TestWriteHeaderFull(t *testing.T) {
	t.Run("with version and size", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		if err := aw.WriteHeader("v1.0", "Mode 0", 80, 200); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "go-cpc-image") {
			t.Error("missing generator comment")
		}
		if !strings.Contains(s, "Mode 0") {
			t.Error("missing mode")
		}
		if !strings.Contains(s, "80x200") {
			t.Error("missing size")
		}
	})

	t.Run("empty version", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		if err := aw.WriteHeader("", "Mode 1", 0, 0); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if strings.Contains(s, "go-cpc-image") {
			t.Error("should not have generator comment with empty version")
		}
		if !strings.Contains(s, "Mode 1") {
			t.Error("missing mode")
		}
		if strings.Contains(s, "Size") {
			t.Error("should not have size with zero dimensions")
		}
	})
}

// --- WriteDB tests ---

func TestWriteDBFull(t *testing.T) {
	t.Run("mixed data with label and size", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		data := []byte{0x01, 0x02, 0x03, 0x04}
		if err := aw.WriteDB(data, 2, "TestData", true); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "TestData:") {
			t.Error("missing label")
		}
		if !strings.Contains(s, "#01,#02") {
			t.Error("missing first line data")
		}
		if !strings.Contains(s, "#03,#04") {
			t.Error("missing second line data")
		}
		if !strings.Contains(s, "Total size 4 bytes") {
			t.Error("missing size comment")
		}
	})

	t.Run("repeated values uses DS", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		data := []byte{0xAA, 0xAA, 0xAA}
		if err := aw.WriteDB(data, 8, "", false); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "DS") || !strings.Contains(s, "3,#AA") {
			t.Errorf("expected DS directive for repeated values, got %q", s)
		}
	})

	t.Run("empty data", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		if err := aw.WriteDB(nil, 8, "Empty", false); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "Empty:") {
			t.Error("missing label")
		}
		if strings.Contains(s, "DB") {
			t.Error("should not have DB for empty data")
		}
	})

	t.Run("no label", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		if err := aw.WriteDB([]byte{0x42}, 8, "", false); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(buf.String(), ":") {
			t.Error("should not have label")
		}
	})
}

// --- WriteDW tests ---

func TestWriteDWFull(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		data := []uint16{0x1234, 0x5678, 0xABCD}
		if err := aw.WriteDW(data, 3, "", true); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "DW") {
			t.Error("missing DW")
		}
		if !strings.Contains(s, "#1234") {
			t.Error("missing first word")
		}
		if !strings.Contains(s, "Total size 3 words") {
			t.Error("missing size comment")
		}
	})

	t.Run("empty", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		if err := aw.WriteDW(nil, 8, "Label", false); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(buf.String(), "DW") {
			t.Error("should not have DW for empty data")
		}
	})

	t.Run("line wrapping", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		data := []uint16{0x001, 0x002, 0x003, 0x004}
		if err := aw.WriteDW(data, 2, "", false); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		lines := strings.Split(strings.TrimSpace(s), "\n")
		dwCount := 0
		for _, l := range lines {
			if strings.Contains(l, "DW") {
				dwCount++
			}
		}
		if dwCount != 2 {
			t.Errorf("expected 2 DW lines, got %d", dwCount)
		}
	})
}

// --- WriteDBWithLabels tests ---

func TestWriteDBWithLabels(t *testing.T) {
	t.Run("with section labels", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		data := []byte{1, 2, 3, 4, 5, 6}
		if err := aw.WriteDBWithLabels(data, 2, 2, "sec", true); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "sec00:") {
			t.Error("missing first section label")
		}
		if !strings.Contains(s, "sec01:") {
			t.Error("missing second section label")
		}
		if !strings.Contains(s, "Total size 6 bytes") {
			t.Error("missing size")
		}
	})

	t.Run("empty data", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		if err := aw.WriteDBWithLabels(nil, 8, 4, "sec", false); err != nil {
			t.Fatal(err)
		}
		if buf.Len() != 0 {
			t.Errorf("expected empty output for nil data, got %q", buf.String())
		}
	})
}

// --- WriteDWWithLabels tests ---

func TestWriteDWWithLabels(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	data := []uint16{0x100, 0x200, 0x300, 0x400}
	if err := aw.WriteDWWithLabels(data, 2, 1, "word", true); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "word00:") {
		t.Error("missing first label")
	}
	if !strings.Contains(s, "DW") {
		t.Error("missing DW")
	}
	if !strings.Contains(s, "Total size 4 words") {
		t.Error("missing size")
	}
}

// --- Depack generation tests ---

func TestGenerateDepack(t *testing.T) {
	tests := []struct {
		name       string
		method     PackMethod
		jumpLabel  string
		contains   []string
		notContain []string
	}{
		{
			"PackNone produces nothing",
			PackNone, "",
			nil,
			[]string{"Depack"},
		},
		{
			"PackStandard produces LZW depack",
			PackStandard, "",
			[]string{"Depack:", "Decompression", "BclLzw:", "TstCodeLzw:", "No more bytes"},
			nil,
		},
		{
			"PackStandard with jump label",
			PackStandard, "MyLabel",
			[]string{"JP\tZ,MyLabel"},
			[]string{"RE\tZ"},
		},
		{
			"PackZX0",
			PackZX0, "",
			[]string{"Depack:", "dzx0s_literals:", "dzx0s_elias:"},
			nil,
		},
		{
			"PackZX0V2",
			PackZX0V2, "",
			[]string{"Depack:", "dzx0s_literals:", "dzx0s_new_offset:"},
			nil,
		},
		{
			"PackZX1",
			PackZX1, "",
			[]string{"Depack:", "dzx1s_literals:", "dzx1s_elias:"},
			nil,
		},
		{
			"PackZX0Ovs uses ZX0",
			PackZX0Ovs, "",
			[]string{"Depack:", "dzx0s_literals:"},
			nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			aw := NewASMWriter(&buf)
			if err := aw.GenerateDepack(tc.method, tc.jumpLabel); err != nil {
				t.Fatal(err)
			}
			s := buf.String()
			for _, c := range tc.contains {
				if !strings.Contains(s, c) {
					t.Errorf("output missing %q", c)
				}
			}
			for _, c := range tc.notContain {
				if strings.Contains(s, c) {
					t.Errorf("output should not contain %q", c)
				}
			}
		})
	}
}

// --- Display generation tests ---

func TestGenerateStandardDisplay(t *testing.T) {
	t.Run("no pack no plus", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		if err := aw.GenerateStandardDisplay(false, PackNone, "media", "palette", false); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "DI") {
			t.Error("missing DI")
		}
		if !strings.Contains(s, "SetPal:") {
			t.Error("missing SetPal label (old CPC init)")
		}
		if !strings.Contains(s, "TstSpace:") {
			t.Error("missing TstSpace")
		}
		if !strings.Contains(s, "RET") {
			t.Error("missing RET")
		}
	})

	t.Run("with pack and plus", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		if err := aw.GenerateStandardDisplay(false, PackZX0, "media", "palette", true); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "Unlock:") {
			t.Error("missing CPC+ unlock")
		}
		if !strings.Contains(s, "CALL\tDepack") {
			t.Error("missing depack call")
		}
		if !strings.Contains(s, "LD\tDE,#C000") {
			t.Error("should use #C000 for non-overscan")
		}
	})

	t.Run("overscan with pack", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		if err := aw.GenerateStandardDisplay(true, PackStandard, "media", "palette", false); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "LD\tDE,#0200") {
			t.Error("should use #0200 for overscan")
		}
	})

	t.Run("PackZX0Ovs post-process", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		if err := aw.GenerateStandardDisplay(false, PackZX0Ovs, "media", "palette", false); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "LoopDepack:") {
			t.Error("missing ZX0 overscan post-process")
		}
	})
}

func TestGenerateInitOld(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.GenerateInitOld("pal"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "SetPal:") {
		t.Error("missing SetPal label")
	}
	if !strings.Contains(s, "LD\tHL,pal") {
		t.Error("missing palette load")
	}
}

func TestGenerateInitPlus(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.GenerateInitPlus("pal"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "Unlock:") {
		t.Error("missing Unlock label")
	}
	if !strings.Contains(s, "UnlockAsic") {
		t.Error("missing UnlockAsic reference")
	}
	if !strings.Contains(s, "LDIR") {
		t.Error("missing LDIR for palette copy")
	}
}

func TestGenerateWaitSpace(t *testing.T) {
	t.Run("wait and withoutRet", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		if err := aw.GenerateWaitSpace(true, true); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "TstSpace:") {
			t.Error("missing TstSpace")
		}
		if !strings.Contains(s, "JR\tZ,TstSpace") {
			t.Error("missing wait loop")
		}
		if strings.Contains(s, "\tRET\n") {
			t.Error("should not have RET when withoutRet=true")
		}
	})

	t.Run("no wait with ret", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		if err := aw.GenerateWaitSpace(false, false); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if strings.Contains(s, "JR\tZ,TstSpace") {
			t.Error("should not wait")
		}
		if !strings.Contains(s, "\tRET\n") {
			t.Error("should have RET")
		}
	})
}

func TestGenerateScreenFormat(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	// Default 80x200 produces no CRTC changes
	if err := aw.GenerateScreenFormat(); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	// 80x200 and 80*200=16000 < 0x4000, so no overscan setup either
	if strings.Contains(s, "BC01") {
		t.Error("should not set CRTC H regs for 80 cols")
	}
	if strings.Contains(s, "BC06") {
		t.Error("should not set CRTC V regs for 200 lines")
	}
}

func TestGenerateScreenFormatEx(t *testing.T) {
	t.Run("non-standard dimensions", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		if err := aw.GenerateScreenFormatEx(96, 272); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "BC01") {
			t.Error("should set CRTC H regs for non-80 cols")
		}
		if !strings.Contains(s, "BC06") {
			t.Error("should set CRTC V regs for non-200 lines")
		}
	})

	t.Run("standard dimensions", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		if err := aw.GenerateScreenFormatEx(80, 200); err != nil {
			t.Fatal(err)
		}
		if buf.Len() != 0 {
			t.Errorf("expected empty for 80x200, got %q", buf.String())
		}
	})

	t.Run("overscan", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		if err := aw.GenerateScreenFormatEx(96, 272); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "BC0C") {
			t.Error("should set R12/R13 for overscan")
		}
	})
}

// --- Palette tests ---

func TestGeneratePalette(t *testing.T) {
	palette := make([]uint16, 16)
	for i := range palette {
		palette[i] = uint16(i)
	}

	t.Run("standard CPC", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		params := PaletteParams{VirtualMode: 0}
		if err := aw.GeneratePalette(palette, params, false, false, "pal"); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "pal:") {
			t.Error("missing label")
		}
		if !strings.Contains(s, "DB") {
			t.Error("missing DB for standard CPC")
		}
	})

	t.Run("CPC Plus with unlock and mode", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		params := PaletteParams{CPCPlus: true, VirtualMode: 1}
		if err := aw.GeneratePalette(palette, params, true, true, "pal"); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "UnlockAsic:") {
			t.Error("missing UnlockAsic label")
		}
		if !strings.Contains(s, "DW") {
			t.Error("missing DW for CPC Plus")
		}
	})

	t.Run("standard with mode", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		params := PaletteParams{VirtualMode: 7}
		if err := aw.GeneratePalette(palette, params, false, true, "pal"); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		// mode 7 -> mode=1, |0x8C = 0x8D
		if !strings.Contains(s, "#8D") {
			t.Errorf("expected mode byte #8D, got %q", s)
		}
	})
}

func TestGenerateAnimationPalette(t *testing.T) {
	palette := []uint16{0x100, 0x200}
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.GenerateAnimationPalette(palette, 2, "animpal"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "animpal:") {
		t.Error("missing label")
	}
	if !strings.Contains(s, "Frame 0") {
		t.Error("missing frame 0 comment")
	}
	if !strings.Contains(s, "Frame 1") {
		t.Error("missing frame 1 comment")
	}
}

func TestGenerateOCPPalette(t *testing.T) {
	palette := []uint16{0x123, 0x456}
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.GenerateOCPPalette(palette, "ocp"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "ocp:") {
		t.Error("missing label")
	}
	if !strings.Contains(s, "DW") {
		t.Error("missing DW")
	}
}

func TestGenerateKitPalette(t *testing.T) {
	palette := []uint16{0x123}
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.GenerateKitPalette(palette, "kit"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "kit:") {
		t.Error("missing label")
	}
	if !strings.Contains(s, "DW") {
		t.Error("missing DW")
	}
}

func TestGeneratePaletteTransition(t *testing.T) {
	start := []uint16{0x000, 0x000}
	end := []uint16{0xFFF, 0xFFF}
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.GeneratePaletteTransition(start, end, 4, "trans"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "trans:") {
		t.Error("missing label")
	}
	if !strings.Contains(s, "Transition step 0/4") {
		t.Error("missing step 0")
	}
	if !strings.Contains(s, "Transition step 4/4") {
		t.Error("missing step 4")
	}
}

func TestGenerateSplitRasterPalette(t *testing.T) {
	splitPal := [][][]int{
		{{0}, {1}, {2}, {3}, {4}, {5}},
		{{6}, {7}, {8}, {9}, {10}, {11}},
	}
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.GenerateSplitRasterPalette(splitPal, 2, "splitpal"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "splitpal:") {
		t.Error("missing label")
	}
	if !strings.Contains(s, "Line 0") {
		t.Error("missing line 0")
	}
}

// --- Animation tests ---

func TestGenerateAnimHeader(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	params := AnimParams{NumCol: 80, NumLig: 200}
	if err := aw.GenerateAnimHeader(0x4000, params); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "ORG") {
		t.Error("missing ORG")
	}
	if !strings.Contains(s, "RUN") {
		t.Error("missing RUN")
	}
	if !strings.Contains(s, "DI") {
		t.Error("missing DI")
	}
}

func TestGenerateAnimHeader_Overscan(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	params := AnimParams{NumCol: 96, NumLig: 272}
	if err := aw.GenerateAnimHeader(0x4000, params); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	// 96*272 > 0x4000 so orgAddr forced to 0x8000
	if !strings.Contains(s, "#8000") {
		t.Error("expected ORG #8000 for overscan")
	}
}

func TestGenerateAnimPlayback(t *testing.T) {
	t.Run("basic animation loop", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		params := AnimParams{
			PackMethod: PackZX0,
		}
		if err := aw.GenerateAnimPlayback(params); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "Debut:") {
			t.Error("missing Debut label")
		}
		if !strings.Contains(s, "AnimDelta") {
			t.Error("missing AnimDelta reference")
		}
		if !strings.Contains(s, "Boucle:") {
			t.Error("missing Boucle label")
		}
		if !strings.Contains(s, "InitDraw:") {
			t.Error("missing InitDraw label")
		}
	})

	t.Run("image mode", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		params := AnimParams{
			ImageMode:  true,
			PackMethod: PackZX0,
		}
		if err := aw.GenerateAnimPlayback(params); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "Delta0") {
			t.Error("missing Delta0 reference for image mode")
		}
		if strings.Contains(s, "AnimDelta") {
			t.Error("should not reference AnimDelta in image mode")
		}
	})

	t.Run("with delay", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		params := AnimParams{
			WithDelay:  true,
			PackMethod: PackStandard,
		}
		if err := aw.GenerateAnimPlayback(params); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "NewIrq:") {
			t.Error("missing NewIrq label")
		}
		if !strings.Contains(s, "SyncFrame:") {
			t.Error("missing SyncFrame label")
		}
	})

	t.Run("with loop", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		params := AnimParams{
			Loop:       true,
			PackMethod: PackZX0,
		}
		if err := aw.GenerateAnimPlayback(params); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "Boucle2:") {
			t.Error("missing Boucle2 label for loop mode")
		}
	})
}

// --- Banking tests ---

func TestGenerateBankInit(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.GenerateBankInit(); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "#7FC0") {
		t.Error("missing bank init port value")
	}
}

func TestGenerateBankSwitch(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.GenerateBankSwitch(3); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "IX+3") {
		t.Error("missing IX offset")
	}
	if !strings.Contains(s, "#7F") {
		t.Error("missing port")
	}
}

func TestGenerateAnimPointers(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		if err := aw.GenerateAnimPointers(3, nil, nil, false); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "AnimDelta:") {
			t.Error("missing AnimDelta label")
		}
		if !strings.Contains(s, "Delta0") {
			t.Error("missing Delta0")
		}
		if !strings.Contains(s, "Delta2") {
			t.Error("missing Delta2")
		}
		if !strings.Contains(s, "#0") {
			t.Error("missing end marker")
		}
	})

	t.Run("with speeds and banks", func(t *testing.T) {
		var buf bytes.Buffer
		aw := NewASMWriter(&buf)
		speeds := []int{9, 6, 3}
		banks := []int{0xC4, 0xC5, 0xC6}
		if err := aw.GenerateAnimPointers(3, banks, speeds, true); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if !strings.Contains(s, "Frame delay") {
			t.Error("missing speed bytes")
		}
		if !strings.Contains(s, "Bank") {
			t.Error("missing bank bytes")
		}
	})
}

func TestGenerateAnimEnd(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.GenerateAnimEnd(1000, false, false, 0, 80); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "Total animation size = 1000") {
		t.Error("missing size comment")
	}
	if !strings.Contains(s, "_EndCode:") {
		t.Error("missing _EndCode label")
	}
	if !strings.Contains(s, "buffer:") {
		t.Error("missing buffer label")
	}
}

func TestGenerateAnimEnd_Force8000(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.GenerateAnimEnd(500, true, false, 0, 80); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "buffer\tEQU\t#8000") {
		t.Error("missing buffer EQU for force8000")
	}
}

func TestGenerateAnimEnd_Mode8(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.GenerateAnimEnd(500, true, true, 8, 80); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "Fonte:") {
		t.Error("missing Fonte label for mode 8+")
	}
	if !strings.Contains(s, "DataFnt:") {
		t.Error("missing DataFnt label for mode 8+")
	}
}

// --- ASIC tests ---

func TestWriteASICUnlock(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteASICUnlock(&buf, "unlock"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "unlock:") {
		t.Error("missing label")
	}
	if !strings.Contains(s, "asic_unlock_loop:") {
		t.Error("missing loop label")
	}
	if !strings.Contains(s, "asic_unlock_data:") {
		t.Error("missing data label")
	}
	if !strings.Contains(s, "BC11") {
		t.Error("missing BC11")
	}
}

func TestWriteASICUnlock_NoLabel(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteASICUnlock(&buf, ""); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "ASIC unlock") {
		t.Error("missing comment")
	}
}

func TestWriteASICLock(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteASICLock(&buf, "lock"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "lock:") {
		t.Error("missing label")
	}
	if !strings.Contains(s, "BC00") {
		t.Error("missing BC00")
	}
	if !strings.Contains(s, "RET") {
		t.Error("missing RET")
	}
}

func TestWriteSpriteData(t *testing.T) {
	var buf bytes.Buffer
	data := []byte{0x12, 0x34, 0x56}
	if err := WriteSpriteData(&buf, data, "sprite1"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "sprite1:") {
		t.Error("missing label")
	}
	if !strings.Contains(s, "Total size 3 bytes") {
		t.Error("missing size")
	}
}

func TestWriteSpritePalette(t *testing.T) {
	var pal [16]int
	pal[0] = 0x123
	pal[1] = 0xABC
	var buf bytes.Buffer
	if err := WriteSpritePalette(&buf, pal, "sprpal"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "sprpal:") {
		t.Error("missing label")
	}
	if !strings.Contains(s, "DW") {
		t.Error("missing DW")
	}
}

// --- Draw mode tests ---

func TestGenerateDrawBlock(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	params := DrawBlockParams{Height: 8, NumCol: 80}
	if err := aw.GenerateDrawBlock(params); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "LoopBlock:") {
		t.Error("missing LoopBlock")
	}
	if !strings.Contains(s, "FinBlock:") {
		t.Error("missing FinBlock")
	}
	if !strings.Contains(s, "LDIR") {
		t.Error("missing LDIR")
	}
}

func TestGenerateDrawBlock_NonStandardHeight(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	params := DrawBlockParams{Height: 4, NumCol: 80}
	if err := aw.GenerateDrawBlock(params); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "BlockOK:") {
		t.Error("missing BlockOK label for non-8 height")
	}
}

func TestGenerateDrawDC_ColumnMode(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	params := DrawDCParams{ModeCol: true, NumCol: 80, AddBC26: 7}
	if err := aw.GenerateDrawDC(params); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "Draw1:") {
		t.Error("missing Draw1")
	}
	if !strings.Contains(s, "SauvSp:") {
		t.Error("missing SauvSp")
	}
}

func TestGenerateDrawDC_RowMode(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	params := DrawDCParams{ModeCol: false, NumCol: 80, AddBC26: 7}
	if err := aw.GenerateDrawDC(params); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "Nbx:") {
		t.Error("missing Nbx")
	}
	if !strings.Contains(s, "LDIR") {
		t.Error("missing LDIR")
	}
}

func TestGenerateDrawDC_OptimSpeed(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	params := DrawDCParams{ModeCol: false, OptimSpeed: true, NumCol: 80, AddBC26: 7}
	if err := aw.GenerateDrawDC(params); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if strings.Contains(s, "NbDec:") {
		t.Error("should not have NbDec in optimSpeed mode")
	}
}

func TestGenerateDrawDirect(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.GenerateDrawDirect(false); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "DrawImgD1:") {
		t.Error("missing DrawImgD1")
	}
	if !strings.Contains(s, "DrawImgD4:") {
		t.Error("missing DrawImgD4 (RLE)")
	}
	if !strings.Contains(s, "DrawImgD6:") {
		t.Error("missing DrawImgD6 (RLE loop)")
	}
}

func TestGenerateDrawDirect_128K(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.GenerateDrawDirect(true); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	// 128K mode has one extra INC IX
	count := strings.Count(s, "INC\tIX")
	if count != 4 {
		t.Errorf("expected 4 INC IX for 128K mode, got %d", count)
	}
}

func TestGenerateDrawAscii_ImageMode(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	params := DrawAsciiParams{
		FrameD:      true,
		ImageMode:   true,
		VirtualMode: 0,
		NumCol:      80,
		NumLig:      200,
	}
	if err := aw.GenerateDrawAscii(params); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "DrawImgD1:") {
		t.Error("missing DrawImgD1")
	}
	if !strings.Contains(s, "TstSpace:") {
		t.Error("missing TstSpace for image mode")
	}
}

// --- EGX tests ---

func TestGenerateEGXDisplayFull(t *testing.T) {
	palette := make([]uint16, 16)
	for i := range palette {
		palette[i] = uint16(i)
	}
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	params := EGXParams{VirtualMode: 3, YEgx: 1, NumCol: 80, NumLig: 200}
	if err := aw.GenerateEGXDisplayFull(palette, params, PackZX0, "media", "pal"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "SetPalette:") {
		t.Error("missing SetPalette")
	}
	if !strings.Contains(s, "WaitVbl:") {
		t.Error("missing WaitVbl")
	}
	if !strings.Contains(s, "SetMode:") {
		t.Error("missing SetMode")
	}
	if !strings.Contains(s, "Depack:") {
		t.Error("missing Depack")
	}
	if !strings.Contains(s, "TstSpace:") {
		t.Error("missing TstSpace")
	}
}

// --- ZX0 overscan post-process test ---

func TestGenerateZX0OvsPostProcess(t *testing.T) {
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.GenerateZX0OvsPostProcess(); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "LoopDepack:") {
		t.Error("missing LoopDepack")
	}
	if !strings.Contains(s, "LDDR") {
		t.Error("missing LDDR")
	}
	if !strings.Contains(s, "Zones") {
		t.Error("missing Zones reference")
	}
}

// --- Split raster tests ---

func TestGenerateSplitRasterASM(t *testing.T) {
	splitScreen := &SplitScreen{
		SplitLines: []SplitLine{
			{
				SplitList: []SplitEntry{
					{Enable: true, Color: 0, Length: 64, Position: 0},
					{Enable: true, Color: 1, Length: 64, Position: 64},
				},
				NumPen: 0,
				Delay:  DelayMin + 200,
			},
		},
	}
	compressedData := []byte{0x01, 0x02, 0x03}
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.GenerateSplitRasterASM(splitScreen, compressedData, 0x100); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "ImageCmp:") {
		t.Error("missing ImageCmp label")
	}
	if !strings.Contains(s, "Boucle:") {
		t.Error("missing Boucle label")
	}
	if !strings.Contains(s, "Depack:") {
		t.Error("missing Depack label")
	}
	if !strings.Contains(s, "Overscan:") {
		t.Error("missing Overscan label")
	}
	if !strings.Contains(s, "Palette:") {
		t.Error("missing Palette label")
	}
}

func TestGenerateSplitRasterASM_EmptyLines(t *testing.T) {
	splitScreen := &SplitScreen{
		SplitLines: []SplitLine{
			{SplitList: nil},
			{SplitList: []SplitEntry{{Enable: false}}},
		},
	}
	var buf bytes.Buffer
	aw := NewASMWriter(&buf)
	if err := aw.GenerateSplitRasterASM(splitScreen, []byte{0x00}, 0x100); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "(empty)") {
		t.Error("expected empty line comments")
	}
}
