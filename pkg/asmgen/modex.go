// Package asmgen provides Z80 Mode X display assembly generation.
package asmgen

import "fmt"

// ModeXParams holds parameters for Mode X display generation
type ModeXParams struct {
	Overscan bool // Whether overscan is enabled
	NumLig   int  // Number of visible lines (200 or 272)
}

// GenerateModeXDisplayFull generates a full Mode X display routine with
// VBL synchronization and per-line palette updates.
// Ported from SaveAsm.cs GenereAfficheModeX.
func (aw *ASMWriter) GenerateModeXDisplayFull(colMode5 [][]int, params ModeXParams, packMethod PackMethod, labelMedia string) error {
	// Decompress image
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

	// Set up interrupt handler
	if err := aw.WriteInstruction("DI", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "HL,#C9FB", "EI; RET"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "(#38),HL", "Install minimal ISR"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("EI", "", ""); err != nil {
		return err
	}

	// VBL synchronization
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

	// Initialize fixed colors (pens 0 and 1)
	if err := aw.WriteInstruction("LD", "HL,Color01", "Fixed colors for pens 0 and 1"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "BC,#7F00", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),C", "Select pen 0"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUTI", "", "Set color 0"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "C", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),C", "Select pen 1"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUTI", "", "Set color 1"); err != nil {
		return err
	}

	// Overscan CRTC setup
	if params.Overscan {
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

	// Wait 6 HALTs for VBI timing
	for i := 0; i < 6; i++ {
		if err := aw.WriteInstruction("HALT", "", ""); err != nil {
			return err
		}
	}
	if err := aw.WriteInstruction("DI", "", ""); err != nil {
		return err
	}

	// Initial delay to first visible line
	waitCycles := 634
	if params.NumLig != 200 {
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
	if err := aw.WriteInstruction("JR", "NZ,$-3", "Wait for first visible line"); err != nil {
		return err
	}

	// Per-line color update loop
	if err := aw.WriteLabel("Boucle"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "HL,ColorModeX", "Variable colors (pens 2 and 3)"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", fmt.Sprintf("DE,%d", params.NumLig), "Number of lines"); err != nil {
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
	if err := aw.WriteInstruction("OUTI", "", "Set color 2"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "C", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),C", "Select pen 3"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUTI", "", "Set color 3"); err != nil {
		return err
	}
	// Timing padding
	if err := aw.WriteInstruction("LD", "B,8", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("NEG", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DJNZ", "$", "Timing delay"); err != nil {
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

	// End-of-frame delay
	waitCycles2 := 1022
	if params.NumLig != 200 {
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
	if err := aw.WriteInstruction("CP", "(HL)", "Fine timing"); err != nil {
		return err
	}
	if params.NumLig == 200 {
		if err := aw.WriteInstruction("CP", "(HL)", "Extra fine timing for 200 lines"); err != nil {
			return err
		}
	}
	if err := aw.WriteInstruction("JR", "Boucle", ""); err != nil {
		return err
	}

	// Decompressor
	if err := aw.GenerateDepack(packMethod, ""); err != nil {
		return err
	}

	// Color data tables
	CpcVGA := "TDU\\X]LEMVFW^@_NGORBSZY[JCK"

	if err := aw.WriteLabel("Color01"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DB", fmt.Sprintf("'%c%c'", CpcVGA[colMode5[0][0]], CpcVGA[colMode5[0][1]]), "Fixed colors"); err != nil {
		return err
	}

	if err := aw.WriteLabel("ColorModeX"); err != nil {
		return err
	}

	line := "'"
	nbOctets := 0
	for y := 0; y < 272; y++ {
		if y < len(colMode5) && len(colMode5[y]) >= 4 {
			line += string(CpcVGA[colMode5[y][2]]) + string(CpcVGA[colMode5[y][3]])
		} else {
			line += string(CpcVGA[0]) + string(CpcVGA[0])
		}
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
