// Package cpc provides CPC screen mode handling and pixel encoding.
package cpc

// TabOctetMode contains the pixel bit patterns for CPC screen modes.
// Mode 0: 2 pixels per byte, Mode 1: 4 pixels per byte, Mode 2: 8 pixels per byte.
// The bit patterns are scattered across the byte in CPC hardware format.
var TabOctetMode = [16]int{
	0x00, 0x80, 0x08, 0x88, 0x20, 0xA0, 0x28, 0xA8,
	0x02, 0x82, 0x0A, 0x8A, 0x22, 0xA2, 0x2A, 0xAA,
}

// ModesVirtuels contains the names of all virtual screen modes.
var ModesVirtuels = [12]string{
	"Mode 0",        // 0
	"Mode 1",        // 1
	"Mode 2",        // 2
	"Mode EGX1",     // 3
	"Mode EGX2",     // 4
	"Mode X",        // 5
	"Mode <Split>",  // 6
	"Mode ASC-UT",   // 7
	"Mode ASC0",     // 8
	"Mode ASC1",     // 9
	"Mode ASC2",     // 10
	"Capture Sprites", // 11
}

// PixelWidth calculates the horizontal pixel step for a given mode and scanline.
// Returns pixels per byte for the current mode.
func PixelWidth(modeVirtuel int, y int, yEgx int) int {
	switch {
	case modeVirtuel == 11:
		// Capture Sprites mode
		return 8 >> 2 // = 2
	case modeVirtuel >= 8:
		// ASCII modes ASC0/1/2
		return 8 >> (modeVirtuel - 8) // ASC0=1, ASC1=2, ASC2=4
	case modeVirtuel >= 5:
		// Mode X and Split modes
		return 8 >> 2 // = 2
	case modeVirtuel > 2:
		// EGX modes 3/4 - alternating line modes
		if (y&2) == yEgx {
			return 8 >> (modeVirtuel - 1) // EGX1=4, EGX2=2
		} else {
			return 8 >> (modeVirtuel - 2) // EGX1=8, EGX2=4
		}
	default:
		// Standard modes 0/1/2
		return 8 >> (modeVirtuel + 1) // Mode0=2, Mode1=4, Mode2=8
	}
}

// MaxPen returns the maximum number of pens (colors) for a given mode and scanline.
func MaxPen(modeVirtuel int, y int, yEgx int) int {
	switch modeVirtuel {
	case 0, 1, 2:
		// Standard modes: 2^(4>>mode) colors
		return 1 << (4 >> modeVirtuel) // Mode0=16, Mode1=4, Mode2=2
	case 3, 4:
		// EGX modes - alternating between mode 1 and mode 2
		if (y&2) == yEgx {
			return 1 << (4 >> (modeVirtuel - 2)) // EGX1=4, EGX2=2
		} else {
			return 1 << (4 >> (modeVirtuel - 3)) // EGX1=2, EGX2=4
		}
	case 5:
		// Mode X
		return 4
	case 6:
		// Split mode
		return 16
	case 7:
		// Mode ASC-UT
		return 4
	case 8:
		// Mode ASC0
		return 16
	case 9:
		// Mode ASC1
		return 4
	case 10:
		// Mode ASC2
		return 2
	default:
		return 16
	}
}