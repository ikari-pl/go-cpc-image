// Package asmgen provides Z80 assembly code generation for CPC+ ASIC features.
package asmgen

import "io"

// ASICUnlockSequence is the 17-byte sequence that must be sent to port &BCxx
// to unlock the CPC+ ASIC registers. The sequence is sent as 17 consecutive
// OUT (C),C instructions with BC set to &BC00 + each byte value.
var ASICUnlockSequence = [17]byte{
	0xFF, 0x00, 0xFF, 0x77, 0xB3, 0x51, 0xA8, 0xD4,
	0x62, 0x39, 0x9C, 0x46, 0x2B, 0x15, 0x8A, 0xCD,
	0xEE,
}

// WriteASICUnlock writes Z80 assembly code that unlocks the CPC+ ASIC.
// The unlock sequence sends 17 specific values to I/O port &BCxx.
func WriteASICUnlock(w io.Writer, label string) error {
	aw := NewASMWriter(w)

	if err := aw.WriteComment("CPC+ ASIC unlock sequence"); err != nil {
		return err
	}
	if label != "" {
		if err := aw.WriteLabel(label); err != nil {
			return err
		}
	}

	if err := aw.WriteInstruction("LD", "BC,#BC11", "start with BC=&BC11"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),C", "send &11 to port &BC"); err != nil {
		return err
	}

	// The standard approach: load each value and OUT it
	if err := aw.WriteInstruction("LD", "HL,asic_unlock_data", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "D,17", "17 bytes to send"); err != nil {
		return err
	}
	if err := aw.WriteLabel("asic_unlock_loop"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,(HL)", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),A", "send byte to &BCxx"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DEC", "D", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NZ,asic_unlock_loop", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("RET", "", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("asic_unlock_data"); err != nil {
		return err
	}
	return aw.WriteDB(ASICUnlockSequence[:], 8, "", false)
}

// WriteASICLock writes Z80 assembly to re-lock the ASIC.
func WriteASICLock(w io.Writer, label string) error {
	aw := NewASMWriter(w)

	if err := aw.WriteComment("CPC+ ASIC lock"); err != nil {
		return err
	}
	if label != "" {
		if err := aw.WriteLabel(label); err != nil {
			return err
		}
	}
	if err := aw.WriteInstruction("LD", "BC,#BC00", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),C", "send &00 to lock ASIC"); err != nil {
		return err
	}
	return aw.WriteInstruction("RET", "", "")
}

// WriteSpriteData writes assembly data for sprite pixel data with a label.
func WriteSpriteData(w io.Writer, spriteData []byte, label string) error {
	aw := NewASMWriter(w)
	return aw.WriteDB(spriteData, 8, label, true)
}

// WriteSpritePalette writes assembly data for the sprite palette.
// Colors are output as DW (16-bit words) in CPC+ 0x0GBR format.
func WriteSpritePalette(w io.Writer, palette [16]int, label string) error {
	aw := NewASMWriter(w)
	words := make([]uint16, 16)
	for i, c := range palette {
		words[i] = uint16(c & 0x0FFF)
	}
	return aw.WriteDW(words, 8, label, false)
}
