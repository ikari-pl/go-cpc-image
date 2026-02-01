// Package asmgen provides Z80 128K bank switching assembly generation.
package asmgen

import "fmt"

// BankingParams holds parameters for 128K memory banking
type BankingParams struct {
	NumCol int // Number of columns (for buffer EQU)
}

// GenerateBankInit generates the bank initialization code that selects
// the default bank (C0 = RAM configuration 0).
// Ported from SaveAsm.cs GenereAffichage 128K handling.
func (aw *ASMWriter) GenerateBankInit() error {
	if err := aw.WriteComment("Select default 128K bank"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "BC,#7FC0", ""); err != nil {
		return err
	}
	return aw.WriteInstruction("OUT", "(C),C", "")
}

// GenerateBankSwitch generates code to switch the active 128K memory bank
// using the value pointed to by IX at the given offset.
// The bank value is written to port #7F.
func (aw *ASMWriter) GenerateBankSwitch(ixOffset int) error {
	if err := aw.WriteInstruction("LD", "B,#7F", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", fmt.Sprintf("A,(IX+%d)", ixOffset), "Bank selection byte"); err != nil {
		return err
	}
	return aw.WriteInstruction("OUT", "(C),A", "Switch bank")
}

// GenerateAnimPointers generates the AnimDelta pointer table used by the
// animation playback loop. Each entry contains a word address for the delta
// data, optionally a speed byte and a bank byte.
// Ported from SaveAsm.cs GenerePointeurs.
func (aw *ASMWriter) GenerateAnimPointers(nbImages int, banks []int, speeds []int, gest128K bool) error {
	if gest128K {
		if err := aw.WriteInstruction("ORG", "EndBank0", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("Write", "Direct -1,-1,#C0", ""); err != nil {
			return err
		}
	}

	if err := aw.WriteLabel("AnimDelta"); err != nil {
		return err
	}

	for i := 0; i < nbImages; i++ {
		if err := aw.WriteInstruction("DW", fmt.Sprintf("Delta%d", i), ""); err != nil {
			return err
		}
		if speeds != nil && i < len(speeds) {
			speed := speeds[i] / 3
			if speed < 1 {
				speed = 1
			}
			if speed > 255 {
				speed = 255
			}
			if err := aw.WriteInstruction("DB", fmt.Sprintf("#%02X", speed), "Frame delay"); err != nil {
				return err
			}
		}
		if gest128K && banks != nil && i < len(banks) {
			if err := aw.WriteInstruction("DB", fmt.Sprintf("#%02X", banks[i]), "Bank"); err != nil {
				return err
			}
		}
	}

	// End marker
	return aw.WriteInstruction("DW", "#0", "End of animation")
}

// GenerateAnimEnd generates the animation end section including total size
// comment, font tables for mode 7+, and the buffer label.
// Ported from SaveAsm.cs GenereFin.
func (aw *ASMWriter) GenerateAnimEnd(totalSize int, force8000 bool, withCode bool, virtualMode int, numCol int) error {
	if err := aw.WriteComment(fmt.Sprintf("Total animation size = %d (#%04X)", totalSize, totalSize)); err != nil {
		return err
	}
	if err := aw.WriteLabel("_EndCode"); err != nil {
		return err
	}
	if err := aw.WriteList(); err != nil {
		return err
	}

	if virtualMode >= 7 {
		align := 256
		if virtualMode > 7 {
			align = 512
		}
		if err := aw.WriteInstruction("Align", fmt.Sprintf("%d", align),
			fmt.Sprintf("Must be a multiple of %d", align)); err != nil {
			return err
		}
		if err := aw.WriteLabel("Fonte"); err != nil {
			return err
		}
		if virtualMode > 7 {
			if err := aw.WriteInstruction("DS", fmt.Sprintf("%d", align), ""); err != nil {
				return err
			}
		}
		// Mode 7 font data would be generated here from trameM1 table
	}

	if withCode && virtualMode >= 7 && force8000 {
		if err := aw.WriteInstruction("Org", "#8000", ""); err != nil {
			return err
		}
	}

	if force8000 {
		if err := aw.WriteEquate("buffer", "#8000"); err != nil {
			return err
		}
	} else {
		if err := aw.WriteLabel("buffer"); err != nil {
			return err
		}
	}

	if withCode && virtualMode > 7 {
		if err := aw.WriteLabel("DataFnt"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("DB", "#00,#C0,#0C,#CC,#30,#F0,#3C,#FC,#03,#C3,#0F,#CF,#33,#F3,#3F,#FF", ""); err != nil {
			return err
		}
	}

	return nil
}
