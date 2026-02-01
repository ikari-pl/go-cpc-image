// Package asmgen provides Z80 animation playback assembly generation.
package asmgen

import "fmt"

// AnimParams holds parameters for animation playback code generation
type AnimParams struct {
	WithDelay   bool       // Enable frame speed control
	Loop        bool       // Loop animation when finished
	Gest128K    bool       // Enable 128K bank switching
	ImageMode   bool       // Single-image mode (no animation loop)
	VirtualMode int        // CPC display mode (7 = mode 7, >7 = mode 8+)
	NumCol      int        // Number of columns
	NumLig      int        // Number of lines
	PackMethod  PackMethod // Compression method for delta data
}

// GenerateAnimHeader generates the animation header with ORG, DI, and
// screen format setup.
// Ported from SaveAsm.cs GenereEntete.
func (aw *ASMWriter) GenerateAnimHeader(orgAddr int, params AnimParams) error {
	if params.NumLig*params.NumCol > 0x4000 {
		orgAddr = 0x8000
	}

	if err := aw.WriteOrg(uint16(orgAddr)); err != nil {
		return err
	}
	if err := aw.WriteRun("$"); err != nil {
		return err
	}
	if err := aw.WriteBlankLine(); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DI", "", ""); err != nil {
		return err
	}
	return aw.GenerateScreenFormatEx(params.NumCol, params.NumLig)
}

// GenerateAnimPlayback generates the main animation playback loop including
// delta decompression, frame pointer traversal, speed control, and optional
// 128K bank switching.
// Ported from SaveAsm.cs GenereAffichage.
func (aw *ASMWriter) GenerateAnimPlayback(params AnimParams) error {
	// Install custom IRQ handler for speed control
	if params.WithDelay {
		if err := aw.WriteInstruction("LD", "HL,NewIrq", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "(#39),HL", "Install IRQ handler"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("EI", "", ""); err != nil {
			return err
		}
	}

	// Font setup for modes >= 8
	if params.VirtualMode > 7 {
		if err := aw.WriteInstruction("LD", "BC,#F00", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "DE,Fonte", ""); err != nil {
			return err
		}
		if err := aw.WriteLabel("SetFonte"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "HL,DataFnt", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "A,C", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("AND", "B", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("ADD", "A,L", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "L,A", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "A,(HL)", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "(DE),A", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "DE", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "HL,DataFnt", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "A,C", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("RRCA", "", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("RRCA", "", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("RRCA", "", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("RRCA", "", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("AND", "B", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("ADD", "A,L", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "L,A", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "A,(HL)", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "(DE),A", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "DE", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "C", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("JR", "NZ,SetFonte", ""); err != nil {
			return err
		}
	}

	if err := aw.WriteLabel("Debut"); err != nil {
		return err
	}

	if params.ImageMode {
		// Single image: just point HL to Delta0
		if err := aw.WriteInstruction("LD", "HL,Delta0", ""); err != nil {
			return err
		}
	} else {
		// Animation loop: traverse AnimDelta pointer table
		if err := aw.WriteInstruction("LD", "IX,AnimDelta", ""); err != nil {
			return err
		}

		if err := aw.WriteLabel("Boucle"); err != nil {
			return err
		}

		// Bank init at start of each frame
		if params.Gest128K {
			if err := aw.GenerateBankInit(); err != nil {
				return err
			}
		}

		// Load delta address from pointer table
		if err := aw.WriteInstruction("LD", "H,(IX+1)", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "L,(IX+0)", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "A,H", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("OR", "H", "Check for end marker"); err != nil {
			return err
		}

		if params.Loop {
			// Loop mode: restart from beginning
			ixResetOffset := 2
			if params.Gest128K {
				ixResetOffset++
			}
			if params.WithDelay {
				ixResetOffset++
			}
			if err := aw.WriteInstruction("JR", "NZ,Boucle2", ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("LD", fmt.Sprintf("IX,AnimDelta+%d", ixResetOffset), ""); err != nil {
				return err
			}
			if err := aw.WriteInstruction("JR", "Boucle", "Restart animation"); err != nil {
				return err
			}
			if err := aw.WriteLabel("Boucle2"); err != nil {
				return err
			}
		} else {
			// Non-loop: return when done
			if err := aw.WriteInstruction("RET", "Z", ""); err != nil {
				return err
			}
		}
	}

	// Load frame delay
	if params.WithDelay {
		if err := aw.WriteInstruction("LD", "A,(IX+2)", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "(SyncFrame+3),A", "Set frame delay target"); err != nil {
			return err
		}
	}

	// Switch bank for this frame's data
	if params.Gest128K {
		ixOffset := 2
		if params.WithDelay {
			ixOffset = 3
		}
		if err := aw.GenerateBankSwitch(ixOffset); err != nil {
			return err
		}
	}

	// Decompress delta into buffer
	if err := aw.WriteInstruction("LD", "DE,Buffer", ""); err != nil {
		return err
	}
	if err := aw.GenerateDepack(params.PackMethod, "InitDraw"); err != nil {
		return err
	}

	// IRQ handler for frame timing
	if params.WithDelay {
		if err := aw.WriteLabel("NewIrq"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("PUSH", "AF", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "A,(SyncFrame+1)", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("INC", "A", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "(SyncFrame+1),A", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("POP", "AF", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("EI", "", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("RET", "", ""); err != nil {
			return err
		}
	}

	// Frame synchronization
	if err := aw.WriteLabel("InitDraw"); err != nil {
		return err
	}
	if params.WithDelay {
		if err := aw.WriteLabel("SyncFrame"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "A,0", "Frame counter (self-modifying)"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("CP", "1", "Target delay (self-modifying)"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("JR", "C,SyncFrame", "Wait until enough frames elapsed"); err != nil {
			return err
		}
		if err := aw.WriteInstruction("XOR", "A", ""); err != nil {
			return err
		}
		if err := aw.WriteInstruction("LD", "(SyncFrame+1),A", "Reset counter"); err != nil {
			return err
		}
	}

	return nil
}
