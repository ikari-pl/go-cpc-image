package fileio

import (
	"fmt"
	"os"

	"github.com/ikari-pl/go-cpc-image/pkg/render"
)

const (
	snaHeaderSize  = 256
	snaSignature   = "MV - SNA"
	snaMinFileSize = snaHeaderSize + 0xC000 + 0x4000 // header + memory up to and including 16KB screen at 0xC000

	// Header offsets
	snaCRTCReg1 = 0x1B // Horizontal displayed characters (NumCol)
	snaCRTCReg6 = 0x1D // Vertical displayed character rows (NumLig)
	snaGAConfig = 0x25 // Gate Array ROM config / mode (bits 0-1)
	snaPalette  = 0x2F // 17 palette entries (16 pens + border)

	snaScreenBase = 0xC000 // Screen memory starts here in CPC address space
)

// buildHardwareToFirmware builds a reverse lookup from CpcVGA:
// given a hardware color byte value, return the firmware color index (0-26).
func buildHardwareToFirmware() map[byte]int {
	m := make(map[byte]int, 27)
	for i := 0; i < len(CpcVGA); i++ {
		m[CpcVGA[i]] = i
	}
	return m
}

// LoadSNA reads an SNA snapshot file and extracts screen data, palette, and mode
// into a BitmapCpc suitable for rendering.
func LoadSNA(filename string) (*render.BitmapCpc, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading SNA file: %w", err)
	}

	if len(data) < snaMinFileSize {
		return nil, fmt.Errorf("SNA file too small: %d bytes (need at least %d)", len(data), snaMinFileSize)
	}

	// Verify signature
	sig := string(data[0:8])
	if sig != snaSignature {
		return nil, fmt.Errorf("invalid SNA signature: %q", sig)
	}

	// Extract screen mode from GA config (bits 0-1)
	mode := int(data[snaGAConfig]) & 0x03

	// Extract CRTC values
	numCol := int(data[snaCRTCReg1])
	numLig := int(data[snaCRTCReg6])

	// Extract palette: map hardware values to firmware color indices (0-26)
	hwToFw := buildHardwareToFirmware()
	var palette [16]int
	for i := 0; i < 16; i++ {
		hwVal := data[snaPalette+i]
		if fw, ok := hwToFw[hwVal]; ok {
			palette[i] = fw
		}
		// else stays 0 (black)
	}

	// Create BitmapCpc
	bmp := render.NewBitmapCpcWithParams(numCol, numLig, false)
	bmp.VirtualMode = mode
	bmp.Palette = palette

	// Set mode table for all lines
	for y := 0; y < 272; y++ {
		bmp.ModeTable[y] = mode
	}

	// Copy 16KB screen data from memory dump
	screenOffset := snaHeaderSize + snaScreenBase
	screenEnd := screenOffset + 0x4000
	if screenEnd > len(data) {
		screenEnd = len(data)
	}
	copy(bmp.ScreenData[:], data[screenOffset:screenEnd])

	return bmp, nil
}
