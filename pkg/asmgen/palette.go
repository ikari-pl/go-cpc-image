// Package asmgen provides Z80 palette data assembly generation.
package asmgen

import (
	"fmt"
	"strings"
)

// PaletteParams holds palette generation parameters
type PaletteParams struct {
	DisableState []int // Array indicating which palette entries are disabled
	CPCPlus      bool  // Whether targeting CPC Plus
	VirtualMode  int   // Virtual mode number
}

// GeneratePalette generates palette data in assembly format
func (aw *ASMWriter) GeneratePalette(palette []uint16, params PaletteParams, withUnlockAsic, withMode bool, label string) error {
	if params.CPCPlus {
		if withUnlockAsic {
			if err := aw.WriteLabel("UnlockAsic"); err != nil {
				return err
			}
			if err := aw.WriteInstruction("DB", "#FF,#00,#FF,#77,#B3,#51,#A8,#D4,#62,#39,#9C,#46,#2B,#15,#8A,#CD,#EE", ""); err != nil {
				return err
			}
		}

		if label != "" {
			if err := aw.WriteLabel(label); err != nil {
				return err
			}
		}

		if withMode {
			mode := params.VirtualMode
			if mode == 7 {
				mode = 1
			} else {
				mode = mode & 3
			}
			mode |= 0x8C
			if err := aw.WriteInstruction("DB", fmt.Sprintf("#%02X", mode), ""); err != nil {
				return err
			}
		}

		var line strings.Builder
		line.WriteString("")

		for i := 0; i < 17; i++ {
			if i > 0 {
				line.WriteString(",")
			}

			var c uint16 = 0xFFFF
			if i < 16 {
				if len(params.DisableState) == 0 || i >= len(params.DisableState) || params.DisableState[i] == 0 {
					// Use actual palette value
					if i < len(palette) {
						c = palette[i]
					} else if len(palette) > 0 {
						c = palette[0] // Default to first color
					}
				}
			} else {
				// 17th entry (border)
				if len(palette) > 0 {
					c = palette[0]
				}
			}

			if c == 0xFFFF {
				line.WriteString("#F0F") // Disabled color
			} else {
				// CPC Plus format: GGGRRRBBBB -> #GBRR
				high := byte(c >> 8)
				low := byte(((c >> 4) & 0x0F) | (c << 4))
				line.WriteString(fmt.Sprintf("#%1X%02X", high, low))
			}
		}

		return aw.WriteInstruction("DW", line.String(), "")

	} else {
		// Standard CPC
		if label != "" {
			if err := aw.WriteLabel(label); err != nil {
				return err
			}
		}

		var line strings.Builder
		line.WriteString("")

		// CpcVGA lookup table
		CpcVGA := "TDU\\X]LEMVFW^@_NGORBSZY[JCK"

		for i := 0; i < 17; i++ {
			if i > 0 {
				line.WriteString(",")
			}

			k := -1
			if i < 16 {
				if len(params.DisableState) == 0 || i >= len(params.DisableState) || params.DisableState[i] == 0 {
					if i < len(palette) {
						k = int(palette[i])
					} else if len(palette) > 0 {
						k = int(palette[0])
					}
				}
			} else {
				// 17th entry (border)
				if len(palette) > 0 {
					k = int(palette[0])
				}
			}

			if k == -1 {
				line.WriteString("#FF")
			} else {
				if k >= 0 && k < len(CpcVGA) {
					line.WriteString(fmt.Sprintf("#%02X", int(CpcVGA[k])))
				} else {
					line.WriteString("#00")
				}
			}
		}

		if withMode {
			line.WriteString(",")
			mode := params.VirtualMode
			if mode == 7 {
				mode = 1
			} else {
				mode = mode & 3
			}
			mode |= 0x8C
			line.WriteString(fmt.Sprintf("#%02X", mode))
		}

		return aw.WriteInstruction("DB", line.String(), "")
	}
}

// GenerateAnimationPalette generates palette data for animations
func (aw *ASMWriter) GenerateAnimationPalette(palette []uint16, frameCount int, label string) error {
	if label != "" {
		if err := aw.WriteLabel(label); err != nil {
			return err
		}
	}

	// For animations, generate frame-by-frame palette data
	for frame := 0; frame < frameCount; frame++ {
		if err := aw.WriteComment(fmt.Sprintf("Frame %d palette", frame)); err != nil {
			return err
		}

		var line strings.Builder
		line.WriteString("")

		for i := 0; i < 16; i++ {
			if i > 0 {
				line.WriteString(",")
			}

			// In animations, palette might change per frame
			// For now, use the same palette for all frames
			colorIndex := i
			if colorIndex < len(palette) {
				line.WriteString(fmt.Sprintf("#%03X", palette[colorIndex]))
			} else {
				line.WriteString("#000")
			}
		}

		if err := aw.WriteInstruction("DW", line.String(), ""); err != nil {
			return err
		}
	}

	return nil
}

// GenerateSplitRasterPalette generates palette data for split-raster effects
func (aw *ASMWriter) GenerateSplitRasterPalette(splitPalette [][][]int, lineCount int, label string) error {
	if label != "" {
		if err := aw.WriteLabel(label); err != nil {
			return err
		}
	}

	CpcVGA := "TDU\\X]LEMVFW^@_NGORBSZY[JCK"

	// Generate palette data for each scanline
	for line := 0; line < lineCount; line++ {
		if line < len(splitPalette) {
			if err := aw.WriteComment(fmt.Sprintf("Line %d colors", line)); err != nil {
				return err
			}

			var lineData strings.Builder
			lineData.WriteString("")

			// Each line can have up to 6 color changes
			for colorSlot := 0; colorSlot < 6; colorSlot++ {
				if colorSlot > 0 {
					lineData.WriteString(",")
				}

				if colorSlot < len(splitPalette[line]) && len(splitPalette[line][colorSlot]) > 0 {
					colorIndex := splitPalette[line][colorSlot][0]
					if colorIndex >= 0 && colorIndex < len(CpcVGA) {
						lineData.WriteString(fmt.Sprintf("#%02X", int(CpcVGA[colorIndex])))
					} else {
						lineData.WriteString("#00")
					}
				} else {
					lineData.WriteString("#00")
				}
			}

			if err := aw.WriteInstruction("DB", lineData.String(), ""); err != nil {
				return err
			}
		}
	}

	return nil
}

// GenerateOCPPalette generates OCP+ palette format
func (aw *ASMWriter) GenerateOCPPalette(palette []uint16, label string) error {
	if label != "" {
		if err := aw.WriteLabel(label); err != nil {
			return err
		}
	}

	// OCP+ format uses hardware palette values directly
	var line strings.Builder
	line.WriteString("")

	for i := 0; i < 16; i++ {
		if i > 0 {
			line.WriteString(",")
		}

		if i < len(palette) {
			// Convert from internal format to CPC Plus hardware format
			color := palette[i]
			// Format: GGGRRRBBBB (12-bit) -> #GBRR (byte + byte)
			g := (color >> 8) & 0x0F
			r := (color >> 4) & 0x0F
			b := color & 0x0F

			// CPC Plus hardware format
			hwValue := (g << 8) | (b << 4) | r
			line.WriteString(fmt.Sprintf("#%03X", hwValue))
		} else {
			line.WriteString("#000")
		}
	}

	return aw.WriteInstruction("DW", line.String(), "")
}

// GenerateKitPalette generates KIT format palette data
func (aw *ASMWriter) GenerateKitPalette(palette []uint16, label string) error {
	if label != "" {
		if err := aw.WriteLabel(label); err != nil {
			return err
		}
	}

	// KIT format stores palette as 16-bit words in a specific format
	var line strings.Builder
	line.WriteString("")

	for i := 0; i < 16; i++ {
		if i > 0 {
			line.WriteString(",")
		}

		if i < len(palette) {
			color := palette[i]
			// KIT format: rearrange bits differently than OCP+
			kitValue := ((color & 0xF00)) + ((color & 0xF0) << 4) + ((color & 0x0F) >> 4)
			line.WriteString(fmt.Sprintf("#%03X", kitValue))
		} else {
			line.WriteString("#000")
		}
	}

	return aw.WriteInstruction("DW", line.String(), "")
}

// GeneratePaletteTransition generates palette transition animation
func (aw *ASMWriter) GeneratePaletteTransition(startPalette, endPalette []uint16, steps int, label string) error {
	if label != "" {
		if err := aw.WriteLabel(label); err != nil {
			return err
		}
	}

	// Generate interpolated palette steps
	for step := 0; step <= steps; step++ {
		if err := aw.WriteComment(fmt.Sprintf("Transition step %d/%d", step, steps)); err != nil {
			return err
		}

		var line strings.Builder
		line.WriteString("")

		for i := 0; i < 16; i++ {
			if i > 0 {
				line.WriteString(",")
			}

			if i < len(startPalette) && i < len(endPalette) {
				// Linear interpolation between start and end colors
				startColor := int(startPalette[i])
				endColor := int(endPalette[i])

				// Interpolate each component separately
				startR := (startColor >> 8) & 0x0F
				startG := (startColor >> 4) & 0x0F
				startB := startColor & 0x0F

				endR := (endColor >> 8) & 0x0F
				endG := (endColor >> 4) & 0x0F
				endB := endColor & 0x0F

				interpR := startR + ((endR - startR) * step) / steps
				interpG := startG + ((endG - startG) * step) / steps
				interpB := startB + ((endB - startB) * step) / steps

				interpColor := (interpR << 8) | (interpG << 4) | interpB
				line.WriteString(fmt.Sprintf("#%03X", interpColor))
			} else {
				line.WriteString("#000")
			}
		}

		if err := aw.WriteInstruction("DW", line.String(), ""); err != nil {
			return err
		}
	}

	return nil
}