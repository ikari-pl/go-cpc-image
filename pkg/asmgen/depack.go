// Package asmgen provides Z80 decompression routine assembly generation.
package asmgen

import "fmt"

// GenerateDepack generates the appropriate decompression routine
func (aw *ASMWriter) GenerateDepack(packMethod PackMethod, jumpLabel string) error {
	switch packMethod {
	case PackStandard:
		return aw.GenerateLZWDepack(jumpLabel)
	case PackZX0, PackZX0Ovs:
		return aw.GenerateZX0Depack(jumpLabel)
	case PackZX0V2:
		return aw.GenerateZX0V2Depack(jumpLabel)
	case PackZX1:
		return aw.GenerateZX1Depack(jumpLabel)
	default:
		return nil
	}
}

// GenerateLZWDepack generates the standard LZW decompression routine
func (aw *ASMWriter) GenerateLZWDepack(jumpLabel string) error {
	if err := aw.WriteComment("Decompression"); err != nil {
		return err
	}
	if err := aw.WriteLabel("Depack"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,(HL)", "DepackBits = InBuf[ InBytes++ ]"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("RRA", "", "Fast rotation only calculate C flag"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("SET", "7,A", "Set bit 7 keeping C flag"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "(BclLzw+1),A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "C,TstCodeLzw", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("CopByteLzw"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LDI", "", "OutBuf[ OutBytes++ ] = InBuf[ InBytes++ ]"); err != nil {
		return err
	}

	if err := aw.WriteLabel("BclLzw"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,0", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("RR", "A", "Rotation with C and Z flags calculation"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "(BclLzw+1),A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NC,CopByteLzw", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "Z,Depack", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("TstCodeLzw"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,(HL)", "A = InBuf[ InBytes ];"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("AND", "A", ""); err != nil {
		return err
	}

	retInstruction := "RET Z"
	if jumpLabel != "" {
		retInstruction = fmt.Sprintf("JP Z,%s", jumpLabel)
	}
	if err := aw.WriteInstruction(retInstruction[:2], retInstruction[3:], "No more bytes to process = done"); err != nil {
		return err
	}

	if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,A", "B = InBuf[ InBytes ]"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("RLCA", "", "A & #80 ?"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NC,TstLzw40", ""); err != nil {
		return err
	}

	if err := aw.WriteInstruction("RLCA", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("RLCA", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("RLCA", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("AND", "7", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ADD", "A,3", "Length = 3 + ( ( InBuf[ InBytes ] >> 4 ) & 7 );"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "C,A", "C = Length"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,B", "B = InBuf[InBytes]"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("AND", "#0F", "Delta = ( InBuf[ InBytes++ ] & 15 ) << 8"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,A", "B = Delta high"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,C", "A = Length"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("SCF", "", "Reposition C flag (for Delta++)"); err != nil {
		return err
	}

	if err := aw.WriteLabel("CopyBytes0"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "C,(HL)", "C = Delta low (Delta |= InBuf[ InBytes++ ]);"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("PUSH", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "H,D", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "L,E", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("SBC", "HL,BC", "HL=HL-(BC+1)"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,0", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("CopyBytes1"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "C,A", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("CopyBytes2"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LDIR", "", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("CopyBytes3"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("POP", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "BclLzw", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("TstLzw40"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("RLCA", "", "A & #40 ?"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NC,TstLzw20", ""); err != nil {
		return err
	}

	if err := aw.WriteInstruction("LD", "C,B", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("RES", "6,C", "Delta = 1 + InBuf[ InBytes++ ] & #3f;"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,0", "BC = Delta + 1 because C flag = 1"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("PUSH", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "H,D", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "L,E", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("SBC", "HL,BC", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LDI", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LDI", "", "Length = 2"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "CopyBytes3", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("TstLzw20"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("RLCA", "", "A & #20 ?"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NC,TstLzw10", ""); err != nil {
		return err
	}

	if err := aw.WriteInstruction("LD", "A,B", "B between #20 and #3F"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ADD", "A,#E2", "= ( A AND #1F ) + 2, and set carry"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,0", "Length = 2 + ( InBuf[ InBytes++ ] & 31 );"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "CopyBytes0", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("CodeLzw0F"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "C,(HL)", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("PUSH", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "H,D", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "L,E", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("CP", "#F0", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NZ,CodeLzw02", ""); err != nil {
		return err
	}

	if err := aw.WriteInstruction("XOR", "A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "BC", "Length = Delta = InBuf[ InBytes + 1 ] + 1;"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("SBC", "HL,BC", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LDIR", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("POP", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "HL", "Inbytes += 2"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "BclLzw", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("CodeLzw02"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("CP", "#20", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "C,CodeLzw01", ""); err != nil {
		return err
	}

	if err := aw.WriteInstruction("LD", "C,B", "Length = Delta = InBuf[ InBytes ];"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,0", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("SBC", "HL,BC", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "CopyBytes2", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("CodeLzw01"); err != nil {
		return err
	}
	if err := aw.WriteComment("Here, B = 1"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("XOR", "A", "Carry to zero"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DEC", "H", "Length = Delta = 256"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "CopyBytes1", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("TstLzw10"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("RLCA", "", "A & #10 ?"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NC,CodeLzw0F", ""); err != nil {
		return err
	}

	if err := aw.WriteInstruction("RES", "4,B", "B = Delta(high) -> ( InBuf[ InBytes++ ] & 15 ) << 8;"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "C,(HL)", "C = Delta(low)  -> InBuf[ InBytes++ ];"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,(HL)", "A = Length - 1"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("PUSH", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "H,D", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "L,E", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("SBC", "HL,BC", "Flag C=1 -> hl=hl-(bc+1) (Delta+1)"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,0", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "C,A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "BC", "BC =  Length = InBuf[ InBytes++ ] + 1;"); err != nil {
		return err
	}

	return aw.WriteInstruction("JR", "CopyBytes2", "")
}

// GenerateZX0Depack generates the ZX0 decompression routine
func (aw *ASMWriter) GenerateZX0Depack(jumpLabel string) error {
	if err := aw.WriteComment("Decompression"); err != nil {
		return err
	}
	if err := aw.WriteLabel("Depack"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "bc,#ffff", "preserve default offset 1"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("push", "bc", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("inc", "bc", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "a,#80", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx0s_literals"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("call", "dzx0s_elias", "obtain length"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ldir", "", "copy literals"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("add", "a,a", "copy from last offset or new offset?"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("jr", "c,dzx0s_new_offset", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("call", "dzx0s_elias", "obtain length"); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx0s_copy"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ex", "(sp),hl", "preserve source,restore offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("push", "hl", "preserve offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("add", "hl,de", "calculate destination - offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ldir", "", "copy from offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("pop", "hl", "restore offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ex", "(sp),hl", "preserve offset,restore source"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("add", "a,a", "copy from literals or new offset?"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("jr", "nc,dzx0s_literals", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx0s_new_offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("call", "dzx0s_elias", "obtain offset MSB"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "b,a", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("pop", "af", "discard last offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("xor", "a", "adjust for negative offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("sub", "c", ""); err != nil {
		return err
	}

	retInstruction := "RET Z"
	if jumpLabel != "" {
		retInstruction = fmt.Sprintf("JP Z,%s", jumpLabel)
	}
	if err := aw.WriteInstruction(retInstruction[:2], retInstruction[3:], "No more bytes to process = done"); err != nil {
		return err
	}

	if err := aw.WriteInstruction("ld", "c,a", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "a,b", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "b,c", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "c,(hl)", "obtain offset LSB"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("inc", "hl", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("rr", "b", "last offset bit becomes first length bit"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("rr", "c", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("push", "bc", "preserve new offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "bc,1", "obtain length"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("call", "nc,dzx0s_elias_backtrack", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("inc", "bc", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("jr", "dzx0s_copy", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx0s_elias"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("inc", "c", "interlaced Elias gamma coding"); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx0s_elias_loop"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("add", "a,a", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("jr", "nz,dzx0s_elias_skip", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "a,(hl)", "load another group of 8 bits"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("inc", "hl", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("rla", "", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx0s_elias_skip"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ret", "c", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx0s_elias_backtrack"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("add", "a,a", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("rl", "c", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("rl", "b", ""); err != nil {
		return err
	}

	return aw.WriteInstruction("jr", "dzx0s_elias_loop", "")
}

// GenerateZX0V2Depack generates the ZX0 V2 decompression routine
func (aw *ASMWriter) GenerateZX0V2Depack(jumpLabel string) error {
	if err := aw.WriteComment("Decompression"); err != nil {
		return err
	}
	if err := aw.WriteLabel("Depack"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "bc,#ffff", "preserve default offset 1"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("push", "bc", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("inc", "bc", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "a,#80", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx0s_literals"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("call", "dzx0s_elias", "obtain length"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ldir", "", "copy literals"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("add", "a,a", "copy from last offset or new offset?"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("jr", "c,dzx0s_new_offset", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("call", "dzx0s_elias", "obtain length"); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx0s_copy"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ex", "(sp),hl", "preserve source, restore offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("push", "hl", "preserve offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("add", "hl,de", "calculate destination -offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ldir", "", "copy from offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("pop", "hl", "restore offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ex", "(sp),hl", "preserve offset, restore source"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("add", "a,a", "copy from literals or new offset?"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("jr", "nc,dzx0s_literals", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx0s_new_offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("pop", "bc", "discard last offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "c,#fe", "prepare negative offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("call", "dzx0s_elias_loop", "obtain offset MSB"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("inc", "c", ""); err != nil {
		return err
	}

	retInstruction := "RET Z"
	if jumpLabel != "" {
		retInstruction = fmt.Sprintf("JP Z,%s", jumpLabel)
	}
	if err := aw.WriteInstruction(retInstruction[:2], retInstruction[3:], "No more bytes to process = done"); err != nil {
		return err
	}

	if err := aw.WriteInstruction("ld", "b,c", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "c,(hl)", "obtain offset LSB"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("inc", "hl", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("rr", "b", "last offset bit becomes first length bit"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("rr", "c", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("push", "bc", "preserve new offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "bc,1", "obtain length"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("call", "nc,dzx0s_elias_backtrack", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("inc", "bc", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("jr", "dzx0s_copy", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx0s_elias"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("inc", "c", "interlaced Elias gamma coding"); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx0s_elias_loop"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("add", "a,a", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("jr", "nz,dzx0s_elias_skip", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "a,(hl)", "load another group of 8 bits"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("inc", "hl", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("rla", "", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx0s_elias_skip"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ret", "c", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx0s_elias_backtrack"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("add", "a,a", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("rl", "c", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("rl", "b", ""); err != nil {
		return err
	}

	return aw.WriteInstruction("jr", "dzx0s_elias_loop", "")
}

// GenerateZX1Depack generates the ZX1 decompression routine
func (aw *ASMWriter) GenerateZX1Depack(jumpLabel string) error {
	if err := aw.WriteComment("Decompression"); err != nil {
		return err
	}
	if err := aw.WriteLabel("Depack"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "bc,#ffff", "preserve default offset 1"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("push", "bc", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "a,#80", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx1s_literals"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("call", "dzx1s_elias", "obtain length"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ldir", "", "copy literals"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("add", "a,a", "copy from last offset or new offset?"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("jr", "c,dzx1s_new_offset", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("call", "dzx1s_elias", "obtain length"); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx1s_copy"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ex", "(sp),hl", "preserve source,restore offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("push", "hl", "preserve offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("add", "hl,de", "calculate destination - offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ldir", "", "copy from offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("pop", "hl", "restore offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ex", "(sp),hl", "preserve offset,restore source"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("add", "a,a", "copy from literals or new offset?"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("jr", "nc,dzx1s_literals", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx1s_new_offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("inc", "sp", "discard last offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("inc", "sp", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("dec", "b", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "c,(hl)", "obtain offset LSB"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("inc", "hl", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("rr", "c", "single byte offset?"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("jr", "nc,dzx1s_msb_skip", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "b,(hl)", "obtain offset MSB"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("inc", "hl", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("rr", "b", "replace last LSB bit with last MSB bit"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("inc", "b", ""); err != nil {
		return err
	}

	retInstruction := "RET Z"
	if jumpLabel != "" {
		retInstruction = fmt.Sprintf("JP Z,%s", jumpLabel)
	}
	if err := aw.WriteInstruction(retInstruction[:2], retInstruction[3:], "No more bytes to process = done"); err != nil {
		return err
	}

	if err := aw.WriteInstruction("rl", "c", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx1s_msb_skip"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("push", "bc", "preserve new offset"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("call", "dzx1s_elias", "obtain length"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("inc", "bc", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("jr", "dzx1s_copy", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx1s_elias"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "bc,1", "interlaced Elias gamma coding"); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx1s_elias_loop"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("add", "a,a", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("jr", "nz,dzx1s_elias_skip", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ld", "a,(hl)", "load another group of 8 bits"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("inc", "hl", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("rla", "", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("dzx1s_elias_skip"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("ret", "nc", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("add", "a,a", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("rl", "c", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("rl", "b", ""); err != nil {
		return err
	}

	return aw.WriteInstruction("jr", "dzx1s_elias_loop", "")
}