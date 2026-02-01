// Package fileio provides CPC file format I/O operations including SCR files with AMSDOS headers.
package fileio

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/ikari/go-cpc-image/pkg/compress"
	"github.com/ikari/go-cpc-image/pkg/cpc"
)

// PackMethod represents compression methods for SCR files
type PackMethod int

const (
	PackNone PackMethod = iota
	PackStandard
	PackZX0
	PackZX0V2
	PackZX1
	PackZX0Ovs
)

// OutputFormat represents output file formats
type OutputFormat int

const (
	OutputBinary OutputFormat = iota
	OutputAssembler
	OutputDSK
)

// SCRParams holds parameters for SCR file generation
type SCRParams struct {
	WithPalette bool
	WithCode    bool
	CPCPlus     bool
	VirtualMode int
}

// LoadSCR loads a CPC SCR file and returns the screen data and palette.
func LoadSCR(data []byte) ([]byte, [16]int, error) {
	if len(data) < 16384 {
		return nil, [16]int{}, fmt.Errorf("SCR file too small: %d bytes", len(data))
	}

	// Simple implementation - just extract screen data and return default palette
	scrData := make([]byte, 16384)
	copy(scrData, data[:16384])

	// Return default palette for now
	defaultPalette := [16]int{1, 24, 20, 6, 26, 0, 2, 7, 10, 12, 14, 16, 18, 22, 1, 14}

	return scrData, defaultPalette, nil
}

// SaveSCRSimple saves CPC screen data as a simple SCR file without compression or embedded code.
func SaveSCRSimple(writer interface{}, scrData []byte, palette []int, params interface{}) error {
	if len(scrData) < 16384 {
		return fmt.Errorf("insufficient screen data: %d bytes", len(scrData))
	}

	// Simple implementation - just write the screen data
	// In a full implementation, this would:
	// 1. Create AMSDOS header
	// 2. Add palette data if requested
	// 3. Add display code if requested
	// 4. Apply compression if requested

	if w, ok := writer.(interface{ Write([]byte) (int, error) }); ok {
		_, err := w.Write(scrData[:16384])
		return err
	}

	return fmt.Errorf("invalid writer type")
}

// Z80 embedded code byte arrays - copied exactly from C# SauveImage.cs
var (
	// CodeStd - Standard CPC display routine
	CodeStd = []byte{
		0x3A, 0xD0, 0xD7, // LD A,(#D7D0)
		0xCD, 0x1C, 0xBD, // CALL #BD1C
		0x21, 0xD1, 0xD7, // LD HL, #D7D1
		0x46,             // LD B,(HL)
		0x48,             // LD C,B
		0xCD, 0x38, 0xBC, // CALL #BC38
		0xAF,             // XOR A
		0x21, 0xD1, 0xD7, // LD HL,#D7D1
		0x46,             // LD B,(HL)
		0x48,             // LD C,B
		0xF5,             // PUSH AF
		0xE5,             // PUSH HL
		0xCD, 0x32, 0xBC, // CALL #BC32
		0xE1,             // POP HL
		0xF1,             // POP AF
		0x23,             // INC HL
		0x3C,             // INC A
		0xFE, 0x10,       // CP #10
		0x20, 0xF1,       // JR NZ,BCL
		0xC3, 0x18, 0xBB, // JP #BB18
	}

	// CodeP0 - CPC Plus display routine (part 1)
	CodeP0 = []byte{
		0xF3,             // DI
		0x01, 0x11, 0xBC, // LD BC,#BC11
		0x21, 0xD0, 0xDF, // LD HL,#DFD0
		0x7E,             // LD A,(HL)
		0xED, 0x79,       // OUT (C),A
		0x23,             // INC HL
		0x0D,             // DEC C
		0x20, 0xF9,       // JR NZ,BCL1
		0x01, 0xA0, 0x7F, // LD BC,#7FA0
		0x3A, 0xD0, 0xD7, // LD A,(#D7D0)
		0xED, 0x79,       // OUT (C),A
		0xED, 0x49,       // OUT (C),C
		0x01, 0xB8, 0x7F, // LD BC,#7FB8
		0xED, 0x49,       // OUT (C),C
		0x21, 0xD1, 0xD7, // LD HL,#D7D1
		0x11, 0x00, 0x64, // LD DE,#6400
		0x01, 0x22, 0x00, // LD BC,#0022
		0xED, 0xB0,       // LDIR
		0xCD, 0xD0, 0xCF, // CALL WaitKey
		0x38, 0xFB,       // JR C,BCL2
		0xFB,             // EI
		0xC9,             // RET
	}

	// CodeP1 - CPC Plus display routine (part 2)
	CodeP1 = []byte{
		0x01, 0x0E, 0xF4, // LD BC,#F40E
		0xED, 0x49,       // OUT (C),C
		0x01, 0xC0, 0xF6, // LD BC,#F6C0
		0xED, 0x49,       // OUT (C),C
		0xAF,             // XOR A
		0xED, 0x79,       // OUT (C),A
		0x01, 0x92, 0xF7, // LD BC,#F792
		0xED, 0x49,       // OUT (C),C
		0x01, 0x45, 0xF6, // LD BC,#F645
		0xED, 0x49,       // OUT (C),C
		0x06, 0xF4,       // LD B,#F4
		0xED, 0x78,       // IN A,(C)
		0x01, 0x82, 0xF7, // LD BC,#F782
		0xED, 0x49,       // OUT (C),C
		0x01, 0x00, 0xF6, // LD BC,#F600
		0xED, 0x49,       // OUT (C),C
		0x17,             // RLA
		0xC9,             // RET
	}

	// CodeP3 - ASIC unlock sequence
	CodeP3 = []byte{0xFF, 0x00, 0xFF, 0x77, 0xB3, 0x51, 0xA8, 0xD4, 0x62, 0x39, 0x9C, 0x46, 0x2B, 0x15, 0x8A, 0xCD, 0xEE}

	// CodeOv - Overscan display routine
	CodeOv = []byte{
		0x21, 0x47, 0x08, // LD HL,#847
		0xCD, 0x36, 0x08, // CALL SetRegs
		0x3A, 0x00, 0x08, // LD A,(#0800)
		0xCD, 0x1C, 0xBD, // CALL #BD1C
		0x21, 0x01, 0x08, // LD HL,#0801
		0xAF,             // XOR A
		0x4E,             // LD C,(HL)
		0x41,             // LD B,C
		0xF5,             // PUSH AF
		0xE5,             // PUSH HL
		0xCD, 0x32, 0xBC, // CALL #BC32
		0xE1,             // POP HL
		0xF1,             // POP AF
		0x23,             // INC HL
		0x3C,             // INC A
		0xFE, 0x10,       // CP #10
		0x20, 0xF1,       // JR NZ,SetPal
		0xCD, 0x18, 0xBB, // CALL #BB18
		0x21, 0x57, 0x08, // LD HL,#0857
		0x01, 0x00, 0xBC, // LD BC,#BC00
		0x7E,             // LD A,(HL)
		0xA7,             // AND A
		0xC8,             // RET Z
		0xED, 0x79,       // OUT (C),A
		0x04,             // INC B
		0x23,             // INC HL
		0x7E,             // LD A,(HL)
		0xED, 0x79,       // OUT (C),A
		0x23,             // INC HL
		0x05,             // DEC B
		0x18, 0xF2,       // JR SetRegs1
		0x01, 0x30, 0x02, 0x32, 0x03, 0x89, 0x06, 0x22,
		0x07, 0x23, 0x0C, 0x0D, 0x0D, 0x00, 0x00, 0x00,
		0x01, 0x28, 0x02, 0x2E, 0x03, 0x8E, 0x06, 0x19,
		0x07, 0x1E, 0x0C, 0x30, 0x00,
	}

	// CodeOvP - Overscan display routine for CPC Plus
	CodeOvP = []byte{
		0xF3,             // DI
		0x01, 0x11, 0xBC, // LD BC,#BC11
		0x21, 0x86, 0x08, // LD HL,#0886
		0x04,             // INC B
		0xED, 0xA3,       // OUTI
		0x0D,             // DEC C
		0x20, 0xFA,       // JR NZ,SetAsic
		0x21, 0x97, 0x08, // LD HL,#897
		0xCD, 0x75, 0x08, // CALL SetReg
		0x01, 0xB8, 0x7F, // LD BC,#7FB8
		0x3A, 0x00, 0x08, // LD A,(#0800)
		0xED, 0x49,       // OUT (C),C
		0xED, 0x79,       // OUT (C),A
		0x21, 0x01, 0x08, // LD HL,#0801
		0x11, 0x00, 0x64, // LD DE,#6400
		0x01, 0x20, 0x00, // LD BC,#0020
		0xED, 0xB0,       // LDIR
		0xAF,             // XOR A
		0x01, 0x0E, 0xF4, // LD BC,#F40E
		0xED, 0x49,       // OUT (C),C
		0x01, 0xC0, 0xF6, // LD BC,#F6C0
		0xED, 0x49,       // OUT (C),C
		0xED, 0x79,       // OUT (C),A
		0x01, 0x92, 0xF7, // LD BC,#F792
		0xED, 0x49,       // OUT (C),C
		0x01, 0x45, 0xF6, // LD BC,#F645
		0xED, 0x49,       // OUT (C),C
		0x06, 0xF4,       // LD B,#F4
		0xED, 0x78,       // IN A,(C)
		0x01, 0x82, 0xF7, // LD BC,#F782
		0xED, 0x49,       // OUT (C),C
		0x17,             // RLA
		0x38, 0xDD,       // JR C,WaitKey
		0x01, 0xA0, 0x7F, // LD BC,#7FA0
		0xED, 0x49,       // OUT (C),C
		0xFB,             // EI
		0x21, 0xA5, 0x08, // LD HL,#08A5
		0x01, 0x00, 0xBC, // LD BC,#BC00
		0x7E,             // LD A,(HL)
		0xA7,             // AND A
		0xC8,             // RET Z
		0xED, 0x79,       // OUT (C),A
		0x04,             // INC B
		0x23,             // INC HL
		0x7E,             // LD A,(HL)
		0xED, 0x79,       // OUT (C),A
		0x23,             // INC HL
		0x05,             // DEC B
		0x18, 0xF2,       // JR SetReg1
		0xFF, 0x00, 0xFF, 0x77, 0xB3, 0x51, 0xA8, 0xD4,
		0x62, 0x39, 0x9C, 0x46, 0x2B, 0x15, 0x8A, 0xCD,
		0xEE, 0x01, 0x30, 0x02, 0x32, 0x06, 0x22, 0x07,
		0x23, 0x0C, 0x0D, 0x0D, 0x00, 0x00, 0x00, 0x01,
		0x28, 0x02, 0x2E, 0x06, 0x19, 0x07, 0x1E, 0x0C,
		0x30,
	}

	// codeEgx0 - EGX mode display routine (part 1)
	codeEgx0 = []byte{
		0x21, 0x00, 0x20, // LD HL,#2000
		0x2B,             // DEC HL
		0x7C,             // LD A,H
		0xB5,             // OR L
		0x20, 0xFB,       // JR NZ,Wait0
		0xF3,             // DI
		0x06, 0xF5,       // LD B,#F5
		0xED, 0x78,       // IN A,(C)
		0x1F,             // RRA
		0x30, 0xF9,       // JR NC,WaitVbl
		0x21, 0x2F, 0x01, // LD HL,#012F
		0x11, 0x01, 0x8C, // LD DE,#8C01
		0x06, 0x7F,       // LD B,#7F
		0xED, 0x51,       // OUT (C),D
		0x7A,             // LD A,D
		0xAB,             // XOR E
		0x57,             // LD D,A
		0x06, 0x0B,       // LD B,#0B
		0x10, 0xFE,       // DJNZ WaitNextLine
		0xCB, 0x46,       // BIT 0,(HL)
		0x2B,             // DEC HL
		0x7C,             // LD A,H
		0xB5,             // OR L
		0x20, 0xEE,       // JR NZ,SetMode
		0xCD, 0xD0, 0xFF, // CALL WaitKey
		0x28, 0xDC,       // JR Z,WaitVbl
		0xFB,             // EI
		0xC9,             // RET
	}

	// codeEgx1 - EGX mode display routine (part 2)
	codeEgx1 = []byte{
		0x16, 0x45,       // LD D,#45
		0x01, 0x0E, 0xF4, // LD BC,#F40E
		0xED, 0x49,       // OUT (C),C
		0x01, 0xC0, 0xF6, // LD BC,#F6C0
		0xED, 0x49,       // OUT (C),C
		0xAF,             // XOR A
		0xED, 0x79,       // OUT (C),A
		0x01, 0x92, 0xF7, // LD BC,#F792
		0xED, 0x49,       // OUT (C),C
		0x06, 0xF6,       // LD B,#F6
		0xED, 0x51,       // OUT (C),D
		0x06, 0xF4,       // LD B,#F4
		0xED, 0x78,       // IN A,(C)
		0x01, 0x82, 0xF7, // LD BC,#F782
		0xED, 0x49,       // OUT (C),C
		0x3C,             // INC A
		0x20, 0x07,       // JR NZ,WaitKey2
		0x7A,             // LD A,D
		0x3C,             // INC A
		0x57,             // LD D,A
		0xFE, 0x4A,       // CP #4A
		0x38, 0xD7,       // JR C,WaitKey1
		0xC9,             // RET
	}

	// codeDepack - Standard LZW decompression routine
	codeDepack = []byte{
		0x21, 0x00, 0x00, // LD HL,Source
		0x11, 0x00, 0x00, // LD DE,Dest
		0x7E,             // LD A,(HL)
		0x23,             // INC HL
		0x1F,             // RRA
		0xCB, 0xFF,       // SET 7,A
		0x32, 0xD3, 0xA5, // LD (BclLzw+1),A
		0x38, 0x0D,       // JR C,TstCodeLzw
		0xED, 0xA0,       // LDI
		0x3E, 0x00,       // LD A,0
		0xCB, 0x1F,       // RRA
		0x32, 0xD3, 0xA5, // LD (BclLzw+1),A
		0x30, 0xF5,       // JR NC,CopByteLzw
		0x28, 0xE9,       // JR Z,DepkLzw
		0x7E,             // LD A,(HL)
		0xA7,             // AND A
		0xCA, 0x00, 0x00, // JP Z,AfficheImage
		0x23,             // INC HL
		0x47,             // LD B,A
		0x07,             // RLCA
		0x30, 0x1D,       // JR NC,TstLzw40
		0x07,             // RLCA
		0x07,             // RLCA
		0x07,             // RLCA
		0xE6, 0x07,       // AND #07
		0xC6, 0x03,       // ADD A,#03
		0x4F,             // LD C,A
		0x78,             // LD A,B
		0xE6, 0x0F,       // AND #0F
		0x47,             // LD B,A
		0x79,             // LD A,C
		0x37,             // SCF
		0x4E,             // LD C,(HL)
		0x23,             // INC HL
		0xE5,             // PUSH HL
		0x62,             // LD H,D
		0x6B,             // LD L,E
		0xED, 0x42,       // SBC HL,DE
		0x06, 0x00,       // LD B,#00
		0x4F,             // LD C,A
		0xED, 0xB0,       // LDIR
		0xE1,             // POP HL
		0x18, 0xCE,       // JR BclLzw
		0x07,             // RLCA
		0x30, 0x10,       // JR NC,TstLzw20
		0x48,             // LD C,B
		0xCB, 0xB1,       // RES 6,C
		0x06, 0x00,       // LD B,#00
		0xE5,             // PUSH HL
		0x62,             // LD H,D
		0x6B,             // LD L,E
		0xED, 0x42,       // SBC HL,BC
		0xED, 0xA0,       // LDI
		0xED, 0xA0,       // LDI
		0x18, 0xEA,       // JR CopyBytes3
		0x07,             // RLCA
		0x30, 0x29,       // JR NC,TstLzw10
		0x78,             // LD A,B
		0xC6, 0xE2,       // ADD A,#E2
		0x06, 0x00,       // LD B,#00
		0x18, 0xD4,       // JR CopyBytes0
		0x4E,             // LD C,(HL)
		0xE5,             // PUSH HL
		0x62,             // LD H,D
		0x6B,             // LD L,E
		0xFE, 0xF0,       // CP #F0
		0x20, 0x0B,       // JR NZ,CodeLzw02
		0xAF,             // XOR A
		0x47,             // LD B,A
		0x03,             // INC BC
		0xED, 0x42,       // SBC HL,BC
		0xED, 0xB0,       // LDIR
		0xE1,             // POP HL
		0x23,             // INC HL
		0x18, 0x9E,       // JR BclLzw
		0xFE, 0x20,       // CP #20
		0x38, 0x07,       // JR C,CodeLzw01
		0x48,             // LD C,B
		0x06, 0x00,       // LD B,#00
		0xED, 0x42,       // SBC HL,BC
		0x18, 0xC0,       // JR CopyBytes2
		0xAF,             // XOR A
		0x25,             // DEC H
		0x18, 0xBB,       // JR CopyBytes1
		0x07,             // RLCA
		0x30, 0xDB,       // JR NC,CodeLzw0F
		0xCB, 0xA0,       // RES 4,B
		0x4E,             // LD C,(HL)
		0x23,             // INC HL
		0x7E,             // LD A,(HL)
		0x23,             // INC HL
		0xE5,             // PUSH HL
		0x62,             // LD H,D
		0x6B,             // LD L,E
		0xED, 0x42,       // SBC HL,BC
		0x06, 0x00,       // LD B,#00
		0x4F,             // LD C,A
		0x03,             // INC BC
		0x18, 0xA8,       // JR CopyBytes2
	}

	// codeDZX0 - ZX0 decompression routine
	codeDZX0 = []byte{
		0x21, 0x00, 0x00, // LD HL,Source
		0x11, 0x00, 0x00, // LD DE,Dest
		0x01, 0xFF, 0xFF, // LD BC,#FFFF
		0xC5,             // PUSH BC
		0x03,             // INC BC
		0x3E, 0x80,       // LD A,#80
		0xCD, 0x3F, 0xA0, // CALL dzx0s_elias
		0xED, 0xB0,       // LDIR
		0x87,             // ADD A,A
		0x38, 0x0D,       // JR C,dzx0s_new_offset
		0xCD, 0x3F, 0xA0, // CALL dzx0s_elias
		0xE3,             // EX (SP),HL
		0xE5,             // PUSH HL
		0x19,             // ADD HL,DE
		0xED, 0xB0,       // LDIR
		0xE1,             // POP HL
		0xE3,             // EX (SP),HL
		0x87,             // ADD A,A
		0x30, 0xEB,       // JR NC,dxz0s_litterals
		0xCD, 0x3F, 0xA0, // CALL dzx0_elias
		0x47,             // LD B,A
		0xF1,             // POP AF
		0xAF,             // XOR A
		0x91,             // SUB C
		0xCA, 0x00, 0x00, // JP Z,AfficheImage
		0x4F,             // LD C,A
		0x78,             // LD A,B
		0x41,             // LD B,C
		0x4E,             // LD C,(HL)
		0x23,             // INC HL
		0xCB, 0x18,       // RR B
		0xCB, 0x19,       // RR C
		0xC5,             // PUSH BC
		0x01, 0x01, 0x00, // LD BC,#0001
		0xD4, 0x47, 0xA0, // CALL NC,dzx0s_elias_backtrack
		0x03,             // INC BC
		0x18, 0xD9,       // JR dzx0s_copy
		0x0C,             // INC C
		0x87,             // ADD A,A
		0x20, 0x03,       // JR NZ,dzx0s_elias_skip
		0x7E,             // LD A,(HL)
		0x23,             // INC HL
		0x17,             // RLA
		0xD8,             // RET C
		0x87,             // ADD A,A
		0xCB, 0x11,       // RL C
		0xCB, 0x10,       // RL B
		0x18, 0xF2,       // JR dzx0s_elias_loop
	}

	// codeDZX0_V2 - ZX0 decompression routine V2
	codeDZX0_V2 = []byte{
		0x21, 0x00, 0x00, // LD HL,Source
		0x11, 0x00, 0x00, // LD DE,Dest
		0x01, 0xFF, 0xFF, // LD BC,#FFFF
		0xC5,             // PUSH BC
		0x03,             // INC BC
		0x3E, 0x80,       // LD A,#80
		0xCD, 0x3D, 0xA0, // CALL dzx0s_elias
		0xED, 0xB0,       // LDIR
		0x87,             // ADD A,A
		0x38, 0x0D,       // JR C,dzx0s_new_offset
		0xCD, 0x3D, 0xA0, // CALL dzx0s_elias
		0xE3,             // EX (SP),HL
		0xE5,             // PUSH HL
		0x19,             // ADD HL,DE
		0xED, 0xB0,       // LDIR
		0xE1,             // POP HL
		0xE3,             // EX (SP),HL
		0x87,             // ADD A,A
		0x30, 0xEB,       // JR NC,dxz0s_litterals
		0xC1,             // POP BC
		0x0E, 0xFE,       // LD C,#FE
		0xCD, 0x3E, 0xA0, // CALL dzx0_elias_loop
		0x0C,             // INC C
		0xCA, 0x00, 0x00, // JP Z,AfficheImage
		0x41,             // LD B,C
		0x4E,             // LD C,(HL)
		0x23,             // INC HL
		0xCB, 0x18,       // RR B
		0xCB, 0x19,       // RR C
		0xC5,             // PUSH BC
		0x01, 0x01, 0x00, // LD BC,#0001
		0xD4, 0x45, 0xA0, // CALL NC,dzx0s_elias_backtrack
		0x03,             // INC BC
		0x18, 0xDB,       // JR dzx0s_copy
		0x0C,             // INC C
		0x87,             // ADD A,A
		0x20, 0x03,       // JR NZ,dzx0s_elias_skip
		0x7E,             // LD A,(HL)
		0x23,             // INC HL
		0x17,             // RLA
		0xD8,             // RET C
		0x87,             // ADD A,A
		0xCB, 0x11,       // RL C
		0xCB, 0x10,       // RL B
		0x18, 0xF2,       // JR dzx0s_elias_loop
	}

	// codeDZX1 - ZX1 decompression routine
	codeDZX1 = []byte{
		0x21, 0x00, 0x00, // LD HL,Source
		0x11, 0x00, 0x00, // LD DE,Dest
		0x01, 0xFF, 0xFF, // LD BC,#FFFF
		0xC5,             // PUSH BC
		0x3E, 0x80,       // LD A,#80
		0xCD, 0x3D, 0xA0, // CALL dzx1s_elias
		0xED, 0xB0,       // LDIR
		0x87,             // ADD A,A
		0x38, 0x0D,       // JR C,dzx1s_new_offset
		0xCD, 0x3D, 0xA0, // CALL dzx1s_elias
		0xE3,             // EX (SP),HL
		0xE5,             // PUSH HL
		0x19,             // ADD HL,DE
		0xED, 0xB0,       // LDIR
		0xE1,             // POP HL
		0xE3,             // EX (SP),HL
		0x87,             // ADD A,A
		0x30, 0xEB,       // JR NC,dzx1s_literals
		0x33,             // INC SP
		0x33,             // INC SP
		0x05,             // DEC B
		0x4E,             // LD C,(HL)
		0x23,             // INC HL
		0xCB, 0x19,       // RR C
		0x30, 0x0A,       // JR NC,dzx1s_msb_skip
		0x46,             // LD B,(HL)
		0x23,             // INC HL
		0xCB, 0x18,       // RR B
		0x04,             // INC B
		0xCA, 0x00, 0x00, // JP Z,AfficheImage
		0xCB, 0x11,       // RL C
		0xC5,             // PUSH BC
		0xCD, 0x3B, 0xA0, // CALL dzx1s_elias
		0x03,             // INC BC
		0x18, 0xDC,       // JR dzx1s_copy
		0x01, 0x01, 0x00, // LD BC,1
		0x87,             // ADD A,A
		0x20, 0x03,       // JR NZ,dzx1s_elias_skip
		0x7E,             // LD A,(HL)
		0x23,             // INC HL
		0x17,             // RLA
		0xD0,             // RET NC
		0x87,             // ADD A,A
		0xCB, 0x11,       // RL C
		0xCB, 0x10,       // RL B
		0x18, 0xF2,       // JR dzx1s_elias_loop
	}
)

// Poke16 writes a 16-bit value in little-endian format at the specified offset
func Poke16(data []byte, offset int, value uint16) {
	binary.LittleEndian.PutUint16(data[offset:offset+2], value)
}

// SaveSCR saves a CPC bitmap as an SCR file with optional compression and embedded code
func SaveSCR(filename string, bitmap []byte, bitmapSize int, packMethod PackMethod, format OutputFormat, params SCRParams, palette []uint16, mode5Colors [][]int) (int, error) {
	var bufPack [0x8000]byte
	overscan := bitmapSize > 0x3F00

	// Prepare palette data
	var modePal [48]byte
	if params.WithPalette && format != OutputAssembler {
		if params.CPCPlus {
			modePal[0] = byte(params.VirtualMode | 0x8C)
			k := 1
			for i := 0; i < 16; i++ {
				modePal[k] = byte(((palette[i] >> 4) & 0x0F) | (palette[i] << 4))
				k++
				modePal[k] = byte(palette[i] >> 8)
				k++
			}
		} else {
			modePal[0] = byte(params.VirtualMode)
			for i := 0; i < 16; i++ {
				modePal[1+i] = byte(palette[i])
			}
		}
	}

	// Set EGX mode values for codeEgx0 if needed
	if params.VirtualMode == 3 || params.VirtualMode == 4 {
		mode := 0x8C01
		if params.VirtualMode == 4 {
			mode = 0x8D03
		}

		// Adjust for yEgx == 2 (would need this from context)
		// if Cpc.yEgx == 2 {
		//     mode += 0x100
		// }

		codeEgx0[20] = byte(mode & 0xFF)
		codeEgx0[21] = byte(mode >> 8)
	}

	// Copy image data and embed code/palette if needed
	imgCpc := make([]byte, bitmapSize)
	copy(imgCpc, bitmap)

	if !overscan {
		if params.WithPalette && format != OutputAssembler {
			copy(imgCpc[0x17D0:], modePal[:])
		}

		if params.WithCode && format != OutputAssembler {
			if params.CPCPlus {
				copy(imgCpc[0x07D0:], CodeP0)
				copy(imgCpc[0x0FD0:], CodeP1)
				copy(imgCpc[0x1FD0:], CodeP3)
			} else {
				copy(imgCpc[0x07D0:], CodeStd)
			}

			if params.VirtualMode == 3 || params.VirtualMode == 4 {
				copy(imgCpc[0x37D0:], codeEgx0)
				copy(imgCpc[0x2FD0:], codeEgx1)
				imgCpc[0x07F2] = 0xD0
				imgCpc[0x07F3] = 0xF7 // CALL 0xF7D0
				imgCpc[0x37FA] = 0xEF // Call 0xEFD0
			}
		}
	} else {
		if bitmapSize > 0x4000 {
			copy(imgCpc[0x600:], modePal[:])
			if params.WithCode && format != OutputAssembler {
				if params.CPCPlus {
					copy(imgCpc[0x621:], CodeOvP)
				} else {
					copy(imgCpc[0x611:], CodeOv)
				}

				if params.VirtualMode == 3 || params.VirtualMode == 4 {
					copy(imgCpc[0x1600:], codeEgx0)
					copy(imgCpc[0x1640:], codeEgx1)
					if params.CPCPlus {
						imgCpc[0x669] = 0xCD
						Poke16(imgCpc, 0x66A, 0x1800) // CALL #1800
					} else {
						Poke16(imgCpc, 0x631, 0x1800) // CALL #1800
					}

					Poke16(imgCpc, 0x1629, 0x1840) // CALL #1840
				}
			}
		}
	}

	startAdr := uint16(0xC000)
	exec := uint16(0xC7D0)
	if overscan {
		startAdr = 0x200
		if params.CPCPlus {
			exec = 0x821
		} else {
			exec = 0x811
		}
	}

	lg := bitmapSize

	// Handle compression
	if packMethod != PackNone {
		compressor := compress.NewCompressor()
		var err error

		switch packMethod {
		case PackStandard:
			lg, err = compressor.Pack(imgCpc, len(imgCpc), bufPack[:], 0, compress.Standard)
		case PackZX0:
			lg, err = compressor.Pack(imgCpc, len(imgCpc), bufPack[:], 0, compress.MethodZX0)
		case PackZX0V2:
			lg, err = compressor.Pack(imgCpc, len(imgCpc), bufPack[:], 0, compress.MethodZX0V2)
		case PackZX1:
			lg, err = compressor.Pack(imgCpc, len(imgCpc), bufPack[:], 0, compress.MethodZX1)
		case PackZX0Ovs:
			lg, err = compressor.Pack(imgCpc, len(imgCpc), bufPack[:], 0, compress.MethodZX0Ovs)
		}

		if err != nil {
			return 0, fmt.Errorf("compression failed: %w", err)
		}

		// Add decompression code if needed
		if params.WithCode && format != OutputAssembler {
			var newExec uint16

			switch packMethod {
			case PackStandard:
				copy(bufPack[lg:], codeDepack)
				Poke16(bufPack[:], lg+0x04, startAdr)
				startAdr = uint16(0xA657 - (lg + len(codeDepack)))
				Poke16(bufPack[:], lg+0x01, startAdr)
				Poke16(bufPack[:], lg+0x20, exec)
				lg += len(codeDepack)
				exec = uint16(0xA657 - len(codeDepack))

			case PackZX0:
				newExec = uint16(0xA657 - len(codeDZX0))
				copy(bufPack[lg:], codeDZX0)
				Poke16(bufPack[:], lg+0x04, startAdr)
				Poke16(bufPack[:], lg+0x0E, newExec+0x3F)
				Poke16(bufPack[:], lg+0x16, newExec+0x3F)
				Poke16(bufPack[:], lg+0x23, newExec+0x3F)
				Poke16(bufPack[:], lg+0x3A, newExec+0x47)
				startAdr = uint16(0xA657 - (lg + len(codeDZX0)))
				Poke16(bufPack[:], lg+0x01, startAdr)
				Poke16(bufPack[:], lg+0x2A, exec)
				lg += len(codeDZX0)
				exec = newExec

			case PackZX0V2:
				newExec = uint16(0xA657 - len(codeDZX0_V2))
				copy(bufPack[lg:], codeDZX0_V2)
				Poke16(bufPack[:], lg+0x04, startAdr)
				Poke16(bufPack[:], lg+0x0E, newExec+0x3D)
				Poke16(bufPack[:], lg+0x16, newExec+0x3D)
				Poke16(bufPack[:], lg+0x26, newExec+0x3E)
				Poke16(bufPack[:], lg+0x38, newExec+0x45)
				startAdr = uint16(0xA657 - (lg + len(codeDZX0_V2)))
				Poke16(bufPack[:], lg+0x01, startAdr)
				Poke16(bufPack[:], lg+0x2A, exec)
				lg += len(codeDZX0_V2)
				exec = newExec

			case PackZX1:
				newExec = uint16(0xA657 - len(codeDZX1))
				copy(bufPack[lg:], codeDZX1)
				Poke16(bufPack[:], lg+0x04, startAdr)
				Poke16(bufPack[:], lg+0x0D, newExec+0x3B)
				Poke16(bufPack[:], lg+0x15, newExec+0x3B)
				Poke16(bufPack[:], lg+0x36, newExec+0x3B)
				startAdr = uint16(0xA657 - (lg + len(codeDZX1)))
				Poke16(bufPack[:], lg+0x01, startAdr)
				Poke16(bufPack[:], lg+0x30, exec)
				lg += len(codeDZX1)
				exec = newExec
			}
		} else {
			startAdr = uint16(0xA657 - lg)
			exec = 0
		}
	}

	// Save based on format
	switch format {
	case OutputBinary:
		entete := cpc.CreeEntete(filename, startAdr, uint16(lg), exec)
		headerBytes, err := cpc.AmsdosToByte(entete)
		if err != nil {
			return 0, fmt.Errorf("failed to create AMSDOS header: %w", err)
		}

		file, err := os.Create(filename)
		if err != nil {
			return 0, fmt.Errorf("failed to create file: %w", err)
		}
		defer file.Close()

		if _, err := file.Write(headerBytes); err != nil {
			return 0, fmt.Errorf("failed to write header: %w", err)
		}

		dataToWrite := imgCpc
		if packMethod != PackNone {
			dataToWrite = bufPack[:lg]
		}

		if _, err := file.Write(dataToWrite); err != nil {
			return 0, fmt.Errorf("failed to write data: %w", err)
		}

	case OutputAssembler:
		return 0, fmt.Errorf("assembler output not implemented yet")

	case OutputDSK:
		return 0, fmt.Errorf("DSK output not implemented yet")
	}

	return lg, nil
}