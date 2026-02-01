// Package fileio provides palette file I/O operations for PAL and KIT formats.
package fileio

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/ikari/go-cpc-image/pkg/cpc"
)

// CpcVGA lookup table for palette conversion (from C# SauveImage.cs)
var CpcVGA = "TDU\\X]LEMVFW^@_NGORBSZY[JCK"

// SavePalette saves a CPC palette in PAL format
func SavePalette(filename string, palette []uint16, params SCRParams) error {
	pal := make([]byte, 239)

	pal[0] = byte(params.VirtualMode)
	indexPal := 3

	if params.CPCPlus {
		for i := 0; i < 16; i++ {
			for j := 0; j < 4; j++ {
				pal[indexPal] = byte(CpcVGA[26-((palette[i]>>4)&0x0F)])
				indexPal++
				pal[indexPal] = byte(CpcVGA[26-(palette[i]&0x0F)])
				indexPal++
				pal[indexPal] = byte(CpcVGA[26-((palette[i]>>8)&0x0F)])
				indexPal++
			}
		}
		pal[195] = pal[3]
		pal[196] = pal[4]
		pal[197] = pal[5]
	} else {
		for i := 0; i < 16; i++ {
			for j := 0; j < 12; j++ {
				if palette[i] < uint16(len(CpcVGA)) {
					pal[indexPal] = byte(CpcVGA[palette[i]])
				} else {
					pal[indexPal] = 0
				}
				indexPal++
			}
		}

		for i := 0; i < 12; i++ {
			pal[indexPal] = pal[i+3]
			indexPal++
		}
	}

	entete := cpc.CreeEntete(filename, 0x8789, uint16(len(pal)), 0x8789)
	headerBytes, err := cpc.AmsdosToByte(entete)
	if err != nil {
		return fmt.Errorf("failed to create AMSDOS header: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(headerBytes); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	if _, err := file.Write(pal); err != nil {
		return fmt.Errorf("failed to write palette data: %w", err)
	}

	return nil
}

// LoadPalette loads a CPC palette from PAL format
func LoadPalette(filename string, palette []uint16, params *SCRParams) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	entete := make([]byte, 0x80)
	pal := make([]byte, 239)

	if _, err := io.ReadFull(file, entete); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	if _, err := io.ReadFull(file, pal); err != nil {
		return fmt.Errorf("failed to read palette data: %w", err)
	}

	if !cpc.CheckAmsdos(entete) || pal[0] >= 11 {
		return fmt.Errorf("invalid palette file format")
	}

	// Convert CpcVGA lookup
	for i := 0; i < 16; i++ {
		for j := 0; j < 27; j++ {
			if pal[3+i*12] == byte(CpcVGA[j]) {
				if params.CPCPlus {
					// Extract RGB components and convert to CPC Plus format
					// This would need access to CpcRgbPalette table from C# code
					// For now, just use the index
					palette[i] = uint16(j)
				} else {
					palette[i] = uint16(j)
				}
				break
			}
		}
	}

	return nil
}

// LoadPaletteKit loads a CPC Plus palette from KIT format
func LoadPaletteKit(filename string, palette []uint16) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	fileSize := stat.Size()
	if fileSize != 158 && fileSize != 160 {
		return fmt.Errorf("invalid KIT file size: %d, expected 158 or 160", fileSize)
	}

	tabBytes := make([]byte, fileSize)
	if _, err := io.ReadFull(file, tabBytes); err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if !cpc.CheckAmsdos(tabBytes) {
		return fmt.Errorf("invalid AMSDOS header")
	}

	start := 128
	startIndex := 1
	if fileSize == 160 {
		startIndex = 0
	}

	for i := startIndex; i < 16; i++ {
		kit := binary.LittleEndian.Uint16(tabBytes[start:])
		col := (kit & 0xF00) + ((kit & 0x0F) << 4) + ((kit & 0xF0) >> 4)
		palette[i] = col
		start += 2
	}

	return nil
}

// SavePaletteKit saves a CPC Plus palette in KIT format
func SavePaletteKit(filename string, palette []uint16) error {
	data := make([]byte, 160) // Full size with AMSDOS header + 32 bytes

	// Create AMSDOS header
	entete := cpc.CreeEntete(filename, uint16(0x6400), 32, 0)
	headerBytes, err := cpc.AmsdosToByte(entete)
	if err != nil {
		return fmt.Errorf("failed to create AMSDOS header: %w", err)
	}

	copy(data, headerBytes)

	// Convert palette to KIT format
	start := 128
	for i := 0; i < 16; i++ {
		col := palette[i]
		kit := ((col & 0xF00)) + ((col & 0xF0) << 4) + ((col & 0x0F) >> 4)
		binary.LittleEndian.PutUint16(data[start:], kit)
		start += 2
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}