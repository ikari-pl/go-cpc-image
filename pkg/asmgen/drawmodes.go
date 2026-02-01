// Package asmgen provides Z80 draw mode assembly generation for
// block-based, character-based, and direct screen update routines.
package asmgen

import "fmt"

// DrawBlockParams holds parameters for block-based drawing
type DrawBlockParams struct {
	Height     int  // Block height in scanlines (typically 8)
	WithDelay  bool // Frame speed control enabled
	Gest128K   bool // 128K bank switching enabled
	NumCol     int  // Number of columns (for BC26 wrap)
}

// GenerateDrawBlock generates the block-based screen update routine.
// Delta data contains a stream of (address_hi, address_lo) pairs followed
// by column data for each block, terminated by a zero byte.
// Ported from SaveAsm.cs GenereDrawBlock.
func (aw *ASMWriter) GenerateDrawBlock(params DrawBlockParams) error {
	if err := aw.WriteInstruction("LD", "HL,buffer", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,(HL)", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "(TailleBlock1+1),A", "Block width"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "C,A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("XOR", "A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("SUB", "C", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "(TailleBlock2+1),A", "Negative width for address adjust"); err != nil {
		return err
	}

	if err := aw.WriteLabel("LoopBlock"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,(HL)", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("AND", "A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "Z,FinBlock", "Zero = end of block list"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "D,A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "E,(HL)", "Screen address"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", fmt.Sprintf("A,%d", params.Height), ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("TailleBlock1"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "BC,4", "Block width (self-modifying)"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LDIR", "", "Copy one scanline of block"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("EX", "DE,HL", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("TailleBlock2"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "BC,#7FF", "Address adjust (self-modifying)"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ADD", "HL,BC", "Next scanline"); err != nil {
		return err
	}

	if params.Height != 8 {
		if err := aw.WriteInstruction("JR", "NC,BlockOK", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", fmt.Sprintf("BC,#C0%02X", params.NumCol), "Wrap to next character row"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("ADD", "HL,BC", ""); err != nil {
			return err
		}
		if err := aw.WriteLabel("BlockOK"); err != nil {
			return err
		}
	}

	if err := aw.WriteInstruction("EX", "DE,HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DEC", "A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NZ,TailleBlock1", "Next scanline of block"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "LoopBlock", "Next block"); err != nil {
		return err
	}

	if err := aw.WriteLabel("FinBlock"); err != nil {
		return err
	}
	// Advance IX past this frame's pointer entry
	if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
		return err
	}
	if params.WithDelay {
		if err := aw.WriteInstruction("INC", "IX", "Skip speed byte"); err != nil {
			return err
		}
	}
	if params.Gest128K {
		if err := aw.WriteInstruction("INC", "IX", "Skip bank byte"); err != nil {
			return err
		}
	}
	if err := aw.WriteInstruction("JP", "Boucle", "Next frame"); err != nil {
		return err
	}
	return aw.WriteNoList()
}

// DrawDCParams holds parameters for column/row-based drawing
type DrawDCParams struct {
	WithDelay  bool // Frame speed control
	ModeCol    bool // Column-oriented draw (true) vs row-oriented (false)
	Gest128K   bool // 128K bank switching
	AddBC26    int  // Scanline address increment (depends on screen width)
	OptimSpeed bool // Optimized speed mode (skips BC26 adjustment)
	NumCol     int  // Number of columns
}

// GenerateDrawDC generates the column/row based screen update routine.
// Ported from SaveAsm.cs GenereDrawDC.
func (aw *ASMWriter) GenerateDrawDC(params DrawDCParams) error {
	if err := aw.WriteInstruction("LD", "HL,buffer", ""); err != nil {
		return err
	}

	if params.ModeCol {
		// Column-oriented mode: reads width, height, screen address, then
		// draws columns top-to-bottom for each X position.
		if err := aw.WriteInstruction("LD", "B,(HL)", "Width in X"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "A,(HL)", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "(NbLig+1),A", "Height in Y"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "E,(HL)", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "D,(HL)", "Screen address"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("EX", "DE,HL", ""); err != nil {
			return err
		}
		if params.WithDelay {
			if err := aw.WriteInstruction("DI", "", ""); err != nil {
				return err
			}
		}
		if err := aw.WriteInstruction("LD", "(SauvSp+1),SP", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", fmt.Sprintf("SP,#C0%02X", params.NumCol), ""); err != nil {
			return err
		}

		if err := aw.WriteLabel("Draw1"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "(RecupHL+1),HL", ""); err != nil {
			return err
		}
		if err := aw.WriteLabel("NbLig"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "C,0", "Height (self-modifying)"); err != nil {
			return err
		}

		if err := aw.WriteLabel("Draw2"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "A,(DE)", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "DE", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "(HL),A", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "A,H", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("ADD", fmt.Sprintf("A,#%02X", params.AddBC26+1), "Next scanline"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "H,A", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("JR", "NC,Draw3", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("ADD", "HL,SP", "Wrap to next char row"); err != nil {
			return err
		}

		if err := aw.WriteLabel("Draw3"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("DEC", "C", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("JR", "NZ,Draw2", ""); err != nil {
			return err
		}

		if err := aw.WriteLabel("RecupHL"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "HL,0", "Restore column X (self-modifying)"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "HL", "Next column"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("DJNZ", "Draw1", ""); err != nil {
			return err
		}

		if err := aw.WriteLabel("SauvSp"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "SP,0", "Restore SP (self-modifying)"); err != nil {
			return err
		}
		if params.WithDelay {
			if err := aw.WriteInstruction("EI", "", ""); err != nil {
				return err
			}
		}
	} else {
		// Row-oriented mode: uses LDIR to copy rows then adjusts address
		if err := aw.WriteInstruction("LD", "A,(HL)", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "(Nbx+1),A", "Width in X"); err != nil {
			return err
		}

		if !params.OptimSpeed {
			if err := aw.WriteInstruction("NEG", "", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("LD", "(NbDec+1),A", "Adjustment for address wrap"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("LD", "A,(HL)", "Height in Y"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("LD", "E,(HL)", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("LD", "D,(HL)", "Screen address"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
				return err
			}
		} else {
			if err := aw.WriteInstruction("LD", "A,(HL)", "Height in Y"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
				return err
			}
		}

		if err := aw.WriteLabel("Nbx"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "BC,0", "Width (self-modifying)"); err != nil {
			return err
		}

		if params.OptimSpeed {
			if err := aw.WriteInstruction("LD", "E,(HL)", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("LD", "D,(HL)", "Screen address"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
				return err
			}
		}

		if err := aw.WriteInstruction("LDIR", "", "Copy one row"); err != nil {
			return err
		}

		if !params.OptimSpeed {
			if err := aw.WriteInstruction("EX", "DE,HL", ""); err != nil {
				return err
			}
			if err := aw.WriteLabel("NbDec"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("LD", fmt.Sprintf("BC,#%02XFF", params.AddBC26), "Address wrap"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("ADD", "HL,BC", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("JR", "NC,Suite", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("LD", fmt.Sprintf("BC,#C0%02X", params.NumCol), ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("ADD", "HL,BC", ""); err != nil {
				return err
			}
			if err := aw.WriteLabel("Suite"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("EX", "DE,HL", ""); err != nil {
				return err
			}
		}

		if err := aw.WriteInstruction("DEC", "A", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("JR", "NZ,Nbx", "Next row"); err != nil {
			return err
		}
	}

	// Advance IX and loop
	if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
		return err
	}
	if params.WithDelay {
		if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
			return err
		}
	}
	if params.Gest128K {
		if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
			return err
		}
	}
	if err := aw.WriteInstruction("JP", "Boucle", "Next frame"); err != nil {
		return err
	}
	return aw.WriteNoList()
}

// GenerateDrawDirect generates the direct delta screen update routine.
// Delta data encodes byte-level changes with displacement and optional RLE.
// Ported from SaveAsm.cs GenereDrawDirect.
func (aw *ASMWriter) GenerateDrawDirect(gest128K bool) error {
	if err := aw.WriteInstruction("LD", "HL,buffer", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "DE,#C000", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "C,(HL)", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,(HL)", "BC = byte count"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("DrawImgD1"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,(HL)", "Displacement"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("BIT", "7,A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NZ,DrawImgD4", "RLE if bit 7 set"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ADD", "A,E", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NC,DrawImgD2", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "D", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("DrawImgD2"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "E,A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LDI", "", "Copy single byte"); err != nil {
		return err
	}

	if err := aw.WriteLabel("DrawImgD3"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JP", "PE,DrawImgD1", "Continue if BC != 0"); err != nil {
		return err
	}
	// Frame done: advance IX and loop
	if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
		return err
	}
	if gest128K {
		if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
			return err
		}
	}
	if err := aw.WriteInstruction("JP", "Boucle", "Next frame"); err != nil {
		return err
	}

	// RLE block: bit 7 was set
	if err := aw.WriteLabel("DrawImgD4"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("AND", "#7F", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ADD", "A,E", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NC,DrawImgD5", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "D", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("DrawImgD5"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "E,A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("PUSH", "BC", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,(HL)", "RLE length"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,(HL)", "RLE value"); err != nil {
		return err
	}

	if err := aw.WriteLabel("DrawImgD6"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "(DE),A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "DE", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DJNZ", "DrawImgD6", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("POP", "BC", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("CPI", "", "Adjust BC and check"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "DrawImgD3", ""); err != nil {
		return err
	}
	return aw.WriteNoList()
}

// DrawAsciiParams holds parameters for ASCII/character-based drawing
type DrawAsciiParams struct {
	FrameFull   bool // Full frame (contains both O and D types)
	FrameO      bool // Ordered sequential frame
	FrameD      bool // Delta frame with displacements
	Gest128K    bool // 128K bank switching
	ImageMode   bool // Single image (no animation loop)
	WithSpeed   bool // Speed byte in pointer table
	VirtualMode int  // Display mode
	NumCol      int  // Number of columns
	NumLig      int  // Number of lines
}

// GenerateDrawAscii generates the ASCII/character-based screen update routine.
// This draws 8-scanline character blocks using a font table lookup.
// Ported from SaveAsm.cs GenereDrawAscii.
func (aw *ASMWriter) GenerateDrawAscii(params DrawAsciiParams) error {
	overscan := params.NumLig*params.NumCol > 0x4000

	if params.FrameFull && !params.ImageMode {
		if err := aw.WriteInstruction("LD", "HL,Buffer", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "A,(HL)", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("CP", "'D'", "Delta frame?"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("JR", "Z,DrawImgD", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("CP", "'I'", "Ignore frame?"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("JR", "Z,DrawImgI", ""); err != nil {
			return err
		}
	} else if params.FrameO {
		if err := aw.WriteInstruction("LD", "HL,Buffer", ""); err != nil {
			return err
		}
	}

	// Sequential ordered frame drawing (FrameO / full frame)
	if !params.FrameD {
		screenAddr := "#C000"
		if overscan {
			screenAddr = "#0200"
		}
		if err := aw.WriteInstruction("LD", "BC,"+screenAddr, ""); err != nil {
			return err
		}
		if params.VirtualMode == 7 {
			if err := aw.WriteInstruction("LD", "D,Fonte/256", ""); err != nil {
				return err
			}
		}

		if err := aw.WriteLabel("DrawImgO"); err != nil {
			return err
		}

		if params.VirtualMode > 7 {
			// Mode 8+: 2-byte font lookup
			if err := aw.WriteInstruction("LD", "D,Fonte/512", ""); err != nil {
				return err
			}
			if !params.FrameO {
				if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
					return err
				}
			}
			if err := aw.WriteInstruction("LD", "E,(HL)", "ASCII code"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("EX", "DE,HL", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("ADD", "HL,HL", ""); err != nil {
				return err
			}
		} else {
			// Mode 0-7: nibble-based font lookup
			if !params.FrameO {
				if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
					return err
				}
			}
			if err := aw.WriteInstruction("LD", "A,(HL)", "ASCII code"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("AND", "#0F", "Low nibble"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("RLCA", "", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("RLCA", "", "*4 for font offset"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("LD", "E,A", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("EX", "DE,HL", ""); err != nil {
				return err
			}
		}

		// Draw 8 scanlines of the character cell
		// Lines 0,1,2,3 (using SET/RES on bit 3,4,5 of H/B)
		if err := aw.WriteInstruction("LD", "A,(HL)", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "L", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "(BC),A", "Scanline 0"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("SET", "3,B", ""); err != nil {
			return err
		}
		if params.VirtualMode == 7 {
			if err := aw.WriteInstruction("LD", "A,(HL)", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("INC", "L", ""); err != nil {
				return err
			}
		}
		if err := aw.WriteInstruction("LD", "(BC),A", "Scanline 1"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("SET", "4,B", ""); err != nil {
			return err
		}
		if params.VirtualMode == 7 {
			if err := aw.WriteInstruction("LD", "A,(HL)", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("INC", "L", ""); err != nil {
				return err
			}
		}
		if err := aw.WriteInstruction("LD", "(BC),A", "Scanline 2"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("RES", "3,B", ""); err != nil {
			return err
		}
		if params.VirtualMode == 7 {
			if err := aw.WriteInstruction("LD", "A,(HL)", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("INC", "L", ""); err != nil {
				return err
			}
		}
		if err := aw.WriteInstruction("LD", "(BC),A", "Scanline 3"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("SET", "5,B", ""); err != nil {
			return err
		}

		// Upper nibble font for mode 7
		if params.VirtualMode == 7 {
			if err := aw.WriteInstruction("LD", "A,(DE)", "ASCII code"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("AND", "#F0", "High nibble"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("RRCA", "", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("RRCA", "", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("ADD", "A,3", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("LD", "L,A", ""); err != nil {
				return err
			}
		}

		// Lines 4,5,6,7
		if err := aw.WriteInstruction("LD", "A,(HL)", ""); err != nil {
			return err
		}
		if params.VirtualMode == 7 {
			if err := aw.WriteInstruction("DEC", "L", ""); err != nil {
				return err
			}
		} else {
			if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
				return err
			}
		}
		if err := aw.WriteInstruction("LD", "(BC),A", "Scanline 4"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("SET", "3,B", ""); err != nil {
			return err
		}
		if params.VirtualMode == 7 {
			if err := aw.WriteInstruction("LD", "A,(HL)", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("DEC", "L", ""); err != nil {
				return err
			}
		}
		if err := aw.WriteInstruction("LD", "(BC),A", "Scanline 5"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("RES", "4,B", ""); err != nil {
			return err
		}
		if params.VirtualMode == 7 {
			if err := aw.WriteInstruction("LD", "A,(HL)", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("DEC", "L", ""); err != nil {
				return err
			}
		}
		if err := aw.WriteInstruction("LD", "(BC),A", "Scanline 6"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("RES", "3,B", ""); err != nil {
			return err
		}
		if params.VirtualMode == 7 {
			if err := aw.WriteInstruction("LD", "A,(HL)", ""); err != nil {
				return err
			}
		}
		if err := aw.WriteInstruction("LD", "(BC),A", "Scanline 7"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("RES", "5,B", ""); err != nil {
			return err
		}

		if err := aw.WriteInstruction("EX", "DE,HL", ""); err != nil {
			return err
		}
		if params.FrameO {
			if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
				return err
			}
		}
		if err := aw.WriteInstruction("INC", "BC", "Next character position"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("BIT", "3,B", "End of row?"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("JR", "Z,DrawImgO", ""); err != nil {
			return err
		}

		if overscan {
			if err := aw.WriteInstruction("LD", "A,B", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("ADD", "A,#40", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("LD", "B,A", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("JP", "P,DrawImgO", "Continue for overscan"); err != nil {
				return err
			}
		}
	}

	// End of sequential frame: advance IX and loop
	if !params.FrameD {
		if !params.ImageMode {
			if err := aw.WriteLabel("DrawImgI"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
				return err
			}
			if params.WithSpeed {
				if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
					return err
				}
			}
			if params.Gest128K {
				if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
					return err
				}
			}
			if err := aw.WriteInstruction("JP", "Boucle", ""); err != nil {
				return err
			}
		} else {
			if err := aw.GenerateWaitSpace(true, false); err != nil {
				return err
			}
		}
	}

	// Delta frame drawing
	if !params.FrameO {
		if !params.FrameD {
			if err := aw.WriteLabel("DrawImgD"); err != nil {
				return err
			}
		}

		if err := aw.WriteInstruction("LD", "HL,#C000", ""); err != nil {
			return err
		}
		if params.FrameD {
			if err := aw.WriteInstruction("LD", "IY,Buffer", ""); err != nil {
				return err
			}
		} else {
			if err := aw.WriteInstruction("LD", "IY,Buffer+1", ""); err != nil {
				return err
			}
		}

		if err := aw.WriteInstruction("LD", "C,(IY+0)", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "B,(IY+1)", "BC = change count"); err != nil {
			return err
		}

		if params.FrameD {
			if err := aw.WriteInstruction("LD", "A,B", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("OR", "C", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("JR", "Z,EndDraw", "Empty delta"); err != nil {
				return err
			}
		}

		if err := aw.WriteLabel("DrawImgD1"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "D,0", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "E,(IY+2)", "Displacement"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("ADD", "HL,DE", "Update screen address"); err != nil {
			return err
		}

		if params.VirtualMode > 7 {
			// Mode 8+: 2-byte font
			if err := aw.WriteInstruction("EX", "DE,HL", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("LD", "H,Fonte/512", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("LD", "L,(IY+3)", "ASCII code"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("ADD", "HL,HL", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("EX", "DE,HL", ""); err != nil {
				return err
			}
		} else {
			if err := aw.WriteInstruction("LD", "D,Fonte/256", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("LD", "A,(IY+3)", "ASCII code"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("AND", "#0F", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("RLCA", "", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("RLCA", "", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("LD", "E,A", ""); err != nil {
				return err
			}
		}

		// Draw 8 scanlines using H register bit manipulation
		if err := aw.WriteInstruction("LD", "A,(DE)", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "E", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "(HL),A", "Scanline 0"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("SET", "3,H", ""); err != nil {
			return err
		}
		if params.VirtualMode == 7 {
			if err := aw.WriteInstruction("LD", "A,(DE)", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("INC", "E", ""); err != nil {
				return err
			}
		}
		if err := aw.WriteInstruction("LD", "(HL),A", "Scanline 1"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("SET", "4,H", ""); err != nil {
			return err
		}
		if params.VirtualMode == 7 {
			if err := aw.WriteInstruction("LD", "A,(DE)", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("INC", "E", ""); err != nil {
				return err
			}
		}
		if err := aw.WriteInstruction("LD", "(HL),A", "Scanline 2"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("RES", "3,H", ""); err != nil {
			return err
		}
		if params.VirtualMode == 7 {
			if err := aw.WriteInstruction("LD", "A,(DE)", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("INC", "E", ""); err != nil {
				return err
			}
		}
		if err := aw.WriteInstruction("LD", "(HL),A", "Scanline 3"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("SET", "5,H", ""); err != nil {
			return err
		}

		// Upper nibble for mode 7
		if params.VirtualMode == 7 {
			if err := aw.WriteInstruction("LD", "A,(IY+3)", "ASCII code"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("AND", "#F0", "High nibble"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("RRCA", "", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("RRCA", "", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("ADD", "A,3", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("LD", "E,A", ""); err != nil {
				return err
			}
		}

		if err := aw.WriteInstruction("LD", "A,(DE)", ""); err != nil {
			return err
		}
		if params.VirtualMode == 7 {
			if err := aw.WriteInstruction("DEC", "E", ""); err != nil {
				return err
			}
		}
		if err := aw.WriteInstruction("LD", "(HL),A", "Scanline 4"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("SET", "3,H", ""); err != nil {
			return err
		}
		if params.VirtualMode == 7 {
			if err := aw.WriteInstruction("LD", "A,(DE)", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("DEC", "E", ""); err != nil {
				return err
			}
		}
		if err := aw.WriteInstruction("LD", "(HL),A", "Scanline 5"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("RES", "4,H", ""); err != nil {
			return err
		}
		if params.VirtualMode == 7 {
			if err := aw.WriteInstruction("LD", "A,(DE)", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("DEC", "E", ""); err != nil {
				return err
			}
		}
		if err := aw.WriteInstruction("LD", "(HL),A", "Scanline 6"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("RES", "3,H", ""); err != nil {
			return err
		}
		if params.VirtualMode == 7 {
			if err := aw.WriteInstruction("LD", "A,(DE)", ""); err != nil {
				return err
			}
		}
		if err := aw.WriteInstruction("LD", "(HL),A", "Scanline 7"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("RES", "5,H", ""); err != nil {
			return err
		}

		// Advance IY and check counter
		if err := aw.WriteInstruction("INC", "IY", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "IY", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("CPI", "", "Decrement BC, advance HL"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("JP", "PE,DrawImgD1", "Continue if BC != 0"); err != nil {
			return err
		}

		if params.FrameD {
			if !params.ImageMode {
				if err := aw.WriteLabel("EndDraw"); err != nil {
					return err
				}
				if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
					return err
				}
				if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
					return err
				}
				if params.WithSpeed {
					if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
						return err
					}
				}
				if params.Gest128K {
					if err := aw.WriteInstruction("INC", "IX", ""); err != nil {
						return err
					}
				}
				if err := aw.WriteInstruction("JP", "Boucle", ""); err != nil {
					return err
				}
			} else {
				if err := aw.GenerateWaitSpace(true, false); err != nil {
					return err
				}
			}
		} else {
			if err := aw.WriteInstruction("JR", "DrawImgI", ""); err != nil {
				return err
			}
		}
	}

	return aw.WriteNoList()
}
