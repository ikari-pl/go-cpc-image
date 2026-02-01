// Package asmgen provides Z80 EGX mode display assembly generation.
package asmgen

import "fmt"

// EGXParams holds parameters for EGX mode display generation
type EGXParams struct {
	VirtualMode int  // 3 = EGX mode 0/1, 4 = EGX mode 1/2
	YEgx        int  // 1 or 2 (scanline grouping)
	Overscan    bool // Overscan mode
	NumCol      int  // Number of columns
	NumLig      int  // Number of lines
}

// GenerateEGXDisplayFull generates a complete EGX mode display routine with
// configurable mode and scanline parameters, ported from SaveAsm.cs
// GenereAfficheModeEgx.
func (aw *ASMWriter) GenerateEGXDisplayFull(palette []uint16, params EGXParams, packMethod PackMethod, labelMedia, labelPalette string) error {
	// Set palette via firmware calls
	if err := aw.WriteInstruction("LD", "HL,"+labelPalette, ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,(HL)", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "C,B", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("CALL", "#BC38", "Set border color"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("XOR", "A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "HL,"+labelPalette, ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("SetPalette"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,(HL)", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "C,B", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("PUSH", "AF", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("PUSH", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("CALL", "#BC32", "Set ink"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("POP", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("POP", "AF", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("CP", "#10", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NZ,SetPalette", ""); err != nil {
		return err
	}

	// Set screen format (CRTC registers)
	if err := aw.GenerateScreenFormatEx(params.NumCol, params.NumLig); err != nil {
		return err
	}

	// Decompress image data
	if err := aw.WriteInstruction("LD", "HL,"+labelMedia, ""); err != nil {
		return err
	}
	if params.Overscan {
		if err := aw.WriteInstruction("LD", "DE,#0200", ""); err != nil {
			return err
		}
	} else {
		if err := aw.WriteInstruction("LD", "DE,#C000", ""); err != nil {
			return err
		}
	}
	if err := aw.WriteInstruction("CALL", "Depack", ""); err != nil {
		return err
	}

	// Raster loop: wait for VBL then toggle modes per scanline
	if err := aw.WriteInstruction("DI", "", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("WaitVbl"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,#F5", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("IN", "A,(C)", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("RRA", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NC,WaitVbl", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("WaitEnd"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("IN", "A,(C)", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("RRA", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "C,WaitEnd", ""); err != nil {
		return err
	}

	// Line counter for mode switching loop
	if err := aw.WriteInstruction("LD", "HL,#012F", ""); err != nil {
		return err
	}

	// Calculate mode toggle value based on EGX type
	mode := 0x8C01 // EGX mode 0/1 (virtual mode 3)
	if params.VirtualMode == 4 {
		mode = 0x8D03 // EGX mode 1/2
	}
	if params.YEgx == 2 {
		mode += 0x100
	}

	if err := aw.WriteInstruction("LD", fmt.Sprintf("DE,#%04X", mode), "Mode toggle pair"); err != nil {
		return err
	}

	// Per-scanline mode switching
	if err := aw.WriteLabel("SetMode"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,#7F", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),D", "Set mode for even lines"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,D", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("XOR", "E", "Toggle mode for next line"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "D,A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,11", "Scanline timing delay"); err != nil {
		return err
	}

	if err := aw.WriteLabel("WaitNextLine"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DJNZ", "WaitNextLine", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("BIT", "0,(HL)", "+4 NOPs timing"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DEC", "HL", "Decrement line counter"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,H", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OR", "L", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NZ,SetMode", "Continue for all lines"); err != nil {
		return err
	}

	// Check for keypress to exit
	if err := aw.WriteInstruction("CALL", "TstSpace", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "Z,WaitVbl", "Loop until key pressed"); err != nil {
		return err
	}

	// Restore CRTC register 12 and exit
	if err := aw.WriteInstruction("LD", "BC,#BC0C", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,#30", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),C", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "B", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("EI", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("RET", "", ""); err != nil {
		return err
	}

	// Keypress test routine
	if err := aw.GenerateWaitSpace(false, false); err != nil {
		return err
	}

	// Decompressor
	if err := aw.GenerateDepack(packMethod, ""); err != nil {
		return err
	}

	// Inline palette data
	var line string
	for y := 0; y < 16; y++ {
		if y > 0 {
			line += ","
		}
		line += fmt.Sprintf("%d", palette[y])
	}
	if err := aw.WriteBlankLine(); err != nil {
		return err
	}
	if err := aw.WriteLabel(labelPalette); err != nil {
		return err
	}
	return aw.WriteInstruction("DB", line, "")
}

// GenerateScreenFormatEx generates CRTC register setup for given dimensions.
func (aw *ASMWriter) GenerateScreenFormatEx(numCol, numLig int) error {
	if numCol != 80 {
		regVal := ((numCol + 1) >> 1) + ((26 + (numCol >> 2)) << 8)
		if err := aw.WriteInstruction("LD", fmt.Sprintf("HL,#%04X", regVal), "CRTC H registers"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "BC,#BC01", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("OUT", "(C),C", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "B", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("OUT", "(C),H", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("DEC", "B", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "C", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("OUT", "(C),C", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "B", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("OUT", "(C),L", ""); err != nil {
			return err
		}
	}

	if numLig != 200 {
		regVal := ((numLig + 7) >> 3) + ((18 + (numLig >> 4)) << 8)
		if err := aw.WriteInstruction("LD", fmt.Sprintf("HL,#%04X", regVal), "CRTC V registers"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "BC,#BC06", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("OUT", "(C),C", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "B", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("OUT", "(C),H", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("DEC", "B", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "C", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("OUT", "(C),C", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "B", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("OUT", "(C),L", ""); err != nil {
			return err
		}
	}

	if numLig*numCol > 0x4000 {
		if err := aw.WriteInstruction("LD", "BC,#BC0C", "Overscan: set R12/R13"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("OUT", "(C),C", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "B", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "C", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("OUT", "(C),C", ""); err != nil {
			return err
		}
	}

	return nil
}
