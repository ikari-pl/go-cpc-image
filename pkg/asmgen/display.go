// Package asmgen provides Z80 display routine assembly generation.
package asmgen

import (
	"fmt"
	"strings"
)

// DisplayMode represents different CPC display modes
type DisplayMode int

const (
	DisplayStandard DisplayMode = iota
	DisplayOverscan
	DisplayEGX
	DisplayModeX
)

// PackMethod represents compression methods (imported from fileio)
type PackMethod int

const (
	PackNone PackMethod = iota
	PackStandard
	PackZX0
	PackZX0V2
	PackZX1
	PackZX0Ovs
)

// GenerateStandardDisplay generates standard CPC display routine
func (aw *ASMWriter) GenerateStandardDisplay(overscan bool, packMethod PackMethod, labelMedia, labelPalette string, cpcPlus bool) error {
	if err := aw.WriteInstruction("DI", "", ""); err != nil {
		return err
	}

	if cpcPlus {
		if err := aw.GenerateInitPlus(labelPalette); err != nil {
			return err
		}
	} else {
		if err := aw.GenerateInitOld(labelPalette); err != nil {
			return err
		}
	}

	if err := aw.GenerateScreenFormat(); err != nil {
		return err
	}

	if packMethod != PackNone {
		if err := aw.WriteInstruction("LD", "HL,"+labelMedia, ""); err != nil {
			return err
		}
		if overscan {
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
	}

	if packMethod == PackZX0Ovs {
		if err := aw.GenerateZX0OvsPostProcess(); err != nil {
			return err
		}
	}

	if err := aw.GenerateWaitSpace(true, true); err != nil {
		return err
	}

	// Restore CRTC register 12
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

	return aw.GenerateDepack(packMethod, "")
}

// GenerateEGXDisplay generates EGX mode display routine
func (aw *ASMWriter) GenerateEGXDisplay(palette []uint16, overscan bool, packMethod PackMethod, labelMedia, labelPalette string) error {
	if err := aw.WriteInstruction("LD", "HL,"+labelPalette, ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,(HL)", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "C,B", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("CALL", "#BC38", ""); err != nil {
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
	if err := aw.WriteInstruction("CALL", "#BC32", ""); err != nil {
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

	if err := aw.GenerateScreenFormat(); err != nil {
		return err
	}

	if err := aw.WriteInstruction("LD", "HL,"+labelMedia, ""); err != nil {
		return err
	}
	if overscan {
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

	if err := aw.WriteInstruction("LD", "HL,#012F", ""); err != nil {
		return err
	}

	// Calculate mode value (would need modeVirtuel and yEgx from context)
	mode := 0x8C01 // Default for mode 3
	// if modeVirtuel == 4 { mode = 0x8D03 }
	// if yEgx == 2 { mode += 0x100 }

	if err := aw.WriteInstruction("LD", fmt.Sprintf("DE,#%04X", mode), ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("SetMode"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,#7F", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),D", "First mode for even lines"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,D", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("XOR", "E", "Invert mode for next line"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "D,A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,11", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("WaitNextLine"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DJNZ", "WaitNextLine", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("BIT", "0,(HL)", "+ 4 NOPs"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DEC", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,H", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OR", "L", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NZ,SetMode", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("CALL", "TstSpace", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "Z,WaitVbl", ""); err != nil {
		return err
	}

	// Restore CRTC register 12
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

	if err := aw.GenerateWaitSpace(false, false); err != nil {
		return err
	}

	if err := aw.GenerateDepack(packMethod, ""); err != nil {
		return err
	}

	// Generate palette data
	var line strings.Builder
	line.WriteString("")
	for y := 0; y < 16; y++ {
		if y > 0 {
			line.WriteString(",")
		}
		line.WriteString(fmt.Sprintf("%d", palette[y]))
	}

	if err := aw.WriteBlankLine(); err != nil {
		return err
	}
	if err := aw.WriteLabel(labelPalette); err != nil {
		return err
	}

	return aw.WriteInstruction("DB", line.String(), "")
}

// GenerateModeXDisplay generates Mode X display routine
func (aw *ASMWriter) GenerateModeXDisplay(colMode5 [][]int, overscan bool, packMethod PackMethod, labelMedia string, numLig int) error {
	if err := aw.WriteInstruction("LD", "HL,"+labelMedia, ""); err != nil {
		return err
	}
	if overscan {
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

	if err := aw.WriteInstruction("DI", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "HL,#C9FB", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "(#38),HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("EI", "", ""); err != nil {
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

	if err := aw.WriteInstruction("LD", "HL,Color01", "Initialize fixed colors (0 and 1)"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "BC,#7F00", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),C", "Select pen 0"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUTI", "", "Update color"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "C", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),C", "Select pen 1"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUTI", "", "Update color"); err != nil {
		return err
	}

	if overscan {
		if err := aw.WriteInstruction("LD", "BC,#BC0C", ""); err != nil {
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

	// Wait for VBI
	for i := 0; i < 6; i++ {
		if err := aw.WriteInstruction("HALT", "", ""); err != nil {
			return err
		}
	}

	if err := aw.WriteInstruction("DI", "", ""); err != nil {
		return err
	}

	waitCycles := 634
	if numLig != 200 {
		waitCycles = 268
	}

	if err := aw.WriteInstruction("LD", fmt.Sprintf("BC,%d", waitCycles), ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DEC", "BC", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,B", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OR", "C", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NZ,$-3", "Wait for first visible screen line"); err != nil {
		return err
	}

	if err := aw.WriteLabel("Boucle"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "HL,ColorModeX", "Initialize variable colors"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", fmt.Sprintf("DE,%d", numLig), "per image lines (2 and 3)"); err != nil {
		return err
	}

	if err := aw.WriteLabel("LoopLineX"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "BC,#7F02", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),C", "Select pen 2"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUTI", "", "Update color"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "C", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),C", "Select pen 3"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUTI", "", "Update color"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,8", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("NEG", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DJNZ", "$", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DEC", "DE", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,D", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OR", "E", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NZ,LoopLineX", ""); err != nil {
		return err
	}

	waitCycles2 := 1022
	if numLig != 200 {
		waitCycles2 = 364
	}

	if err := aw.WriteInstruction("LD", fmt.Sprintf("BC,%d", waitCycles2), ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DEC", "BC", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,B", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OR", "C", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NZ,$-3", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("CP", "(HL)", ""); err != nil {
		return err
	}
	if numLig == 200 {
		if err := aw.WriteInstruction("CP", "(HL)", ""); err != nil {
			return err
		}
	}
	if err := aw.WriteInstruction("JR", "Boucle", ""); err != nil {
		return err
	}

	// Generate decompression code
	if err := aw.GenerateDepack(packMethod, ""); err != nil {
		return err
	}

	// Generate color data
	CpcVGA := "TDU\\X]LEMVFW^@_NGORBSZY[JCK"

	if err := aw.WriteLabel("Color01"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DB", fmt.Sprintf("'%c%c'", CpcVGA[colMode5[0][0]], CpcVGA[colMode5[0][1]]), ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("ColorModeX"); err != nil {
		return err
	}

	line := "'"
	nbOctets := 0
	for y := 0; y < 272; y++ {
		line += string(CpcVGA[colMode5[y][2]]) + string(CpcVGA[colMode5[y][3]])
		nbOctets++
		if nbOctets >= 16 {
			if err := aw.WriteInstruction("DB", line+"'", ""); err != nil {
				return err
			}
			line = "'"
			nbOctets = 0
		}
	}
	if nbOctets > 0 {
		if err := aw.WriteInstruction("DB", line+"'", ""); err != nil {
			return err
		}
	}

	return nil
}

// Helper functions

func (aw *ASMWriter) GenerateInitOld(labelPalette string) error {
	if err := aw.WriteInstruction("LD", "HL,"+labelPalette, ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,#7F", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("XOR", "A", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("SetPal"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "B", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUTI", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("CP", "18", ""); err != nil {
		return err
	}

	return aw.WriteInstruction("JR", "C,SetPal", "")
}

func (aw *ASMWriter) GenerateInitPlus(labelPalette string) error {
	if err := aw.WriteInstruction("LD", "BC,#BC11", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "HL,UnlockAsic", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("Unlock"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,(HL)", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DEC", "C", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NZ,Unlock", ""); err != nil {
		return err
	}

	if err := aw.WriteInstruction("LD", "BC,#7FB8", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,("+labelPalette+")", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),C", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "HL,"+labelPalette+"+1", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "DE,#6400", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "BC,#0022", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LDIR", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "BC,#7FA0", ""); err != nil {
		return err
	}

	return aw.WriteInstruction("OUT", "(C),C", "")
}

func (aw *ASMWriter) GenerateScreenFormat() error {
	// This would need access to Cpc.NumCol and Cpc.NumLig from context
	// For now, assume standard 80x200
	numCol := 80
	numLig := 200

	if numCol != 80 {
		regVal := ((numCol + 1) >> 1) + ((26 + (numCol >> 2)) << 8)
		if err := aw.WriteInstruction("LD", fmt.Sprintf("HL,#%04X", regVal), ""); err != nil {
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
		if err := aw.WriteInstruction("LD", fmt.Sprintf("HL,#%04X", regVal), ""); err != nil {
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
		if err := aw.WriteInstruction("LD", "BC,#BC0C", ""); err != nil {
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

func (aw *ASMWriter) GenerateWaitSpace(wait, withoutRet bool) error {
	if err := aw.WriteLabel("TstSpace"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "BC,#F40E", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),C", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "BC,#F6C0", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),C", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("XOR", "A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "BC,#F792", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),C", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "BC,#F645", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),C", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,#F4", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("IN", "A,(C)", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "BC,#F782", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),C", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "BC,#F600", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),C", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "A", ""); err != nil {
		return err
	}

	if wait {
		if err := aw.WriteInstruction("JR", "Z,TstSpace", ""); err != nil {
			return err
		}
	}

	if !withoutRet {
		if err := aw.WriteInstruction("RET", "", ""); err != nil {
			return err
		}
	}

	return nil
}

func (aw *ASMWriter) GenerateZX0OvsPostProcess() error {
	if err := aw.WriteInstruction("EX", "DE,HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DEC", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "IX,Zones", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("LoopDepack"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,(IX+1)", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("AND", "A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "Z,TstSpace", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "D,A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "E,(IX+0)", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,(IX+3)", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "C,(IX+2)", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LDDR", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
		return err
	}

	return aw.WriteInstruction("JR", "LoopDepack", "")
}