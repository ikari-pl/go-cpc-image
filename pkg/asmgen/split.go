// Package asmgen provides Z80 split-raster assembly generation with cycle-exact timing.
package asmgen

import (
	"fmt"
	"strings"
)

// SplitEntry represents a color split in a scanline
type SplitEntry struct {
	Enable   bool
	Color    int
	Length   int
	Position int
}

// SplitLine represents a complete scanline with splits
type SplitLine struct {
	SplitList []SplitEntry
	NumPen     int
	Delay     int // Timing delay in pixels
}

// SplitScreen represents the entire split-raster screen data
type SplitScreen struct {
	SplitLines []SplitLine
}

// Timing constants for cycle-exact code generation
const (
	DelayMin      = 15644 // Minimum delay offset
	PixelsPerNOP   = 8     // 1 NOP = 8 pixels
	CyclesPerLine  = 64    // Total cycles per scanline
	NOPsPerLine    = 512   // NOPs per scanline
)

// GenerateSplitRasterASM generates cycle-exact split-raster Z80 assembly
func (aw *ASMWriter) GenerateSplitRasterASM(splitScreen *SplitScreen, compressedData []byte, orgAddress uint16) error {
	CpcVGA := "TDU\\X]LEMVFW^@_NGORBSZY[JCK"

	nbLigneVide := 0
	tpsImage := 3
	reste := 0
	oldc1, oldc2, oldc3, oldc4, oldc5, oldPen := 0, 0, 0, 0, 0, 0
	oldRetPrec := DelayMin + 15644

	// Generate ORG and compressed data
	if err := aw.WriteOrg(orgAddress); err != nil {
		return err
	}
	if err := aw.WriteNoList(); err != nil {
		return err
	}
	if err := aw.WriteLabel("ImageCmp"); err != nil {
		return err
	}
	if err := aw.WriteDB(compressedData, 16, "", true); err != nil {
		return err
	}

	// Generate initialization code
	if err := aw.generateSplitInit(); err != nil {
		return err
	}

	// Process each scanline
	for y := 0; y < len(splitScreen.SplitLines); y++ {
		lSpl := splitScreen.SplitLines[y]

		if len(lSpl.SplitList) > 0 && lSpl.SplitList[0].Enable {
			hlMod := false
			retPrec := oldRetPrec
			oldRetPrec = 0

			if nbLigneVide > 0 || reste > 0 {
				retPrec += (reste << 3) + (nbLigneVide * 512)
				nbLigneVide = 0
			}

			retard := lSpl.Delay - DelayMin
			var err error
			hlMod, err = aw.generateDelay(retard+retPrec, false, true)
			if err != nil {
				return err
			}

			retSameCol := 0
			if err := aw.WriteComment(fmt.Sprintf("Line %d", y)); err != nil {
				return err
			}

			// Handle pen selection optimization
			if reste < 0 && lSpl.NumPen == oldPen {
				hlMod2, err := aw.generateDelay((6+reste)<<3, false, true)
				if err != nil {
					return err
				}
				hlMod = hlMod || hlMod2
			} else {
				if err := aw.WriteInstruction("LD", fmt.Sprintf("C,%d", lSpl.NumPen), "(2 NOPs)"); err != nil {
					return err
				}
				if err := aw.WriteInstruction("OUT", "(C),C", "(4 NOPs)"); err != nil {
					return err
				}
				oldPen = lSpl.NumPen
			}

			// Generate color register loads with optimization
			c0 := int(CpcVGA[lSpl.SplitList[0].Color])
			if err := aw.WriteInstruction("LD", fmt.Sprintf("C,#%02X", c0), "(2 NOPs)"); err != nil {
				return err
			}

			c1, c2 := 0, 0
			if len(lSpl.SplitList) > 1 {
				c1 = int(CpcVGA[lSpl.SplitList[1].Color])
			}
			if len(lSpl.SplitList) > 2 {
				c2 = int(CpcVGA[lSpl.SplitList[2].Color])
			}

			if (c1 != oldc1 || c2 != oldc2) && (len(lSpl.SplitList) > 1 && lSpl.SplitList[1].Enable || len(lSpl.SplitList) > 2 && lSpl.SplitList[2].Enable) {
				if err := aw.WriteInstruction("LD", fmt.Sprintf("DE,#%02X%02X", c1, c2), "(3 NOPs)"); err != nil {
					return err
				}
			} else {
				retSameCol += 24
			}

			c3, c4 := 0, 0
			if len(lSpl.SplitList) > 3 {
				c3 = int(CpcVGA[lSpl.SplitList[3].Color])
			}
			if len(lSpl.SplitList) > 4 {
				c4 = int(CpcVGA[lSpl.SplitList[4].Color])
			}

			if hlMod || ((c3 != oldc3 || c4 != oldc4) && (len(lSpl.SplitList) > 3 && lSpl.SplitList[3].Enable || len(lSpl.SplitList) > 4 && lSpl.SplitList[4].Enable)) {
				if err := aw.WriteInstruction("LD", fmt.Sprintf("HL,#%02X%02X", c3, c4), "(3 NOPs)"); err != nil {
					return err
				}
			} else {
				retSameCol += 24
			}

			c5 := 0
			if len(lSpl.SplitList) > 5 {
				c5 = int(CpcVGA[lSpl.SplitList[5].Color])
			}
			if c5 != oldc5 && len(lSpl.SplitList) > 5 && lSpl.SplitList[5].Enable {
				if err := aw.WriteInstruction("LD", fmt.Sprintf("A,#%02X", c5), "(2 NOPs)"); err != nil {
					return err
				}
			} else {
				retSameCol += 16
			}

			// Generate delay for same color optimization
			if retSameCol > 0 {
				canUseHL := !(len(lSpl.SplitList) > 3 && lSpl.SplitList[3].Enable) && !(len(lSpl.SplitList) > 4 && lSpl.SplitList[4].Enable)
				if _, err := aw.generateDelay(retSameCol, false, canUseHL); err != nil {
					return err
				}
			}

			// First color change (always happens)
			if err := aw.WriteInstruction("OUT", "(C),C", "(4 NOPs)"); err != nil {
				return err
			}

			tpsLine := (retard >> 3) + 20
			lg := 0
			if len(lSpl.SplitList) > 0 {
				lg = lSpl.SplitList[0].Length - 32
			}
			tpsLine += lg >> 3

			// Generate subsequent color changes
			if len(lSpl.SplitList) > 1 && lSpl.SplitList[1].Enable {
				c6 := 0
				if lg > 0 {
					// Handle 6th color if needed
					if len(lSpl.SplitList) > 6 && lSpl.SplitList[6].Enable && lg > 16 {
						c6 = int(CpcVGA[lSpl.SplitList[6].Color])
						if err := aw.WriteInstruction("LD", fmt.Sprintf("C,#%02X", c6), "(2 NOPs)"); err != nil {
							return err
						}
						lg -= 16
					}
					canUseHL := !(len(lSpl.SplitList) > 3 && lSpl.SplitList[3].Enable) && !(len(lSpl.SplitList) > 4 && lSpl.SplitList[4].Enable)
					if _, err := aw.generateDelay(lg, false, canUseHL); err != nil {
						return err
					}
					lg = 0
				}

				if err := aw.WriteInstruction("OUT", "(C),D", "(4 NOPs)"); err != nil {
					return err
				}
				tpsLine += 4

				// Third color
				if len(lSpl.SplitList) > 2 && lSpl.SplitList[2].Enable {
					lgs := lSpl.SplitList[1].Length - 32
					canUseHL := !(len(lSpl.SplitList) > 3 && lSpl.SplitList[3].Enable) && !(len(lSpl.SplitList) > 4 && lSpl.SplitList[4].Enable)
					if _, err := aw.generateDelay(lgs, false, canUseHL); err != nil {
						return err
					}
					if err := aw.WriteInstruction("OUT", "(C),E", "(4 NOPs)"); err != nil {
						return err
					}
					tpsLine += 4 + (lgs >> 3)

					// Fourth color
					if len(lSpl.SplitList) > 3 && lSpl.SplitList[3].Enable {
						lgs = lSpl.SplitList[2].Length - 32
						if _, err := aw.generateDelay(lgs, false, false); err != nil {
							return err
						}
						if err := aw.WriteInstruction("OUT", "(C),H", "(4 NOPs)"); err != nil {
							return err
						}
						tpsLine += 4 + (lgs >> 3)

						// Fifth color
						if len(lSpl.SplitList) > 4 && lSpl.SplitList[4].Enable {
							lgs = lSpl.SplitList[3].Length - 32
							if _, err := aw.generateDelay(lgs, false, false); err != nil {
								return err
							}
							if err := aw.WriteInstruction("OUT", "(C),L", "(4 NOPs)"); err != nil {
								return err
							}
							tpsLine += 4 + (lgs >> 3)

							// Sixth color
							if len(lSpl.SplitList) > 5 && lSpl.SplitList[5].Enable {
								lgs = lSpl.SplitList[4].Length - 32
								if _, err := aw.generateDelay(lgs, false, false); err != nil {
									return err
								}
								if err := aw.WriteInstruction("OUT", "(C),A", "(4 NOPs)"); err != nil {
									return err
								}
								tpsLine += 4 + (lgs >> 3)
							}

							// Seventh color (uses C register loaded earlier)
							if len(lSpl.SplitList) > 6 && lSpl.SplitList[6].Enable && c6 != 0 {
								lgs = lSpl.SplitList[5].Length - 32
								if _, err := aw.generateDelay(lgs, false, false); err != nil {
									return err
								}
								if err := aw.WriteInstruction("OUT", "(C),C", "(4 NOPs)"); err != nil {
									return err
								}
								tpsLine += 4 + (lgs >> 3)
							}
						}
					}
				}
			}

			// Update state for next line
			oldc1 = c1
			oldc2 = c2
			oldc3 = c3
			oldc4 = c4
			oldc5 = c5
			reste = 64 - tpsLine + (lg >> 3)

		} else {
			nbLigneVide++
			if err := aw.WriteComment(fmt.Sprintf("Line %d (empty)", y)); err != nil {
				return err
			}
		}

		tpsImage += 64
	}

	// Generate end code and data
	return aw.generateSplitEnd()
}

// generateDelay generates cycle-exact delay code
func (aw *ASMWriter) generateDelay(nbPixels int, crashBC, canUseHL bool) (bool, error) {
	hlModified := false
	nbNops := nbPixels >> 3 // 1 NOP = 8 pixels

	// Large delays using loop
	if nbNops >= 1024 {
		bc := (nbNops - 4) / 7
		if crashBC {
			bc = (nbNops - 2) / 7
		}
		delai := (bc * 7) + 4
		if crashBC {
			delai = (bc * 7) + 2
		}

		comment := fmt.Sprintf("Wait %d NOPs (7*%d+%d)", delai, bc, 4)
		if crashBC {
			comment = fmt.Sprintf("Wait %d NOPs (7*%d+%d)", delai, bc, 2)
		}

		if err := aw.WriteInstruction("LD", fmt.Sprintf("BC,%d", bc), comment); err != nil {
			return hlModified, err
		}
		if err := aw.WriteInstruction("DEC", "BC", ""); err != nil {
			return hlModified, err
		}
		if err := aw.WriteInstruction("LD", "A,B", ""); err != nil {
			return hlModified, err
		}
		if err := aw.WriteInstruction("OR", "C", ""); err != nil {
			return hlModified, err
		}
		if err := aw.WriteInstruction("JR", "NZ,$-3", ""); err != nil {
			return hlModified, err
		}

		if !crashBC {
			if err := aw.WriteInstruction("LD", "B,#7F", ""); err != nil {
				return hlModified, err
			}
		}

		nbNops -= delai
	}

	// Medium delays using stack operations
	if nbNops > 11 && nbNops < 17 {
		if err := aw.WriteInstruction("EX", "(SP),HL", "Wait 6 NOPs"); err != nil {
			return hlModified, err
		}
		if err := aw.WriteInstruction("EX", "(SP),HL", "Wait 6 NOPs"); err != nil {
			return hlModified, err
		}
		nbNops -= 12
	}

	if nbNops > 8 && nbNops < 17 {
		if err := aw.WriteInstruction("PUSH", "IX", "Wait 5 NOPs"); err != nil {
			return hlModified, err
		}
		if err := aw.WriteInstruction("POP", "IX", "Wait 4 NOPs"); err != nil {
			return hlModified, err
		}
		nbNops -= 9
	}

	// Medium delays using DJNZ
	if nbNops > 7 && nbNops < 1024 {
		b := (nbNops - 3) >> 2
		if crashBC {
			b = (nbNops - 1) >> 2
		}
		delai := (b << 2) + 3
		if crashBC {
			delai = (b << 2) + 1
		}

		comment := fmt.Sprintf("Wait %d NOPs (4*%d+%d)", delai, b, 3)
		if crashBC {
			comment = fmt.Sprintf("Wait %d NOPs (4*%d+%d)", delai, b, 1)
		}

		if err := aw.WriteInstruction("LD", fmt.Sprintf("B,%d", b), comment); err != nil {
			return hlModified, err
		}
		if err := aw.WriteInstruction("DJNZ", "$-0", ""); err != nil {
			return hlModified, err
		}

		if !crashBC {
			if err := aw.WriteInstruction("LD", "B,#7F", ""); err != nil {
				return hlModified, err
			}
		}

		nbNops -= delai
	}

	// Small delays using various instructions
	if nbNops >= 4 {
		if err := aw.WriteInstruction("ADD", "IX,BC", "Wait 4 NOPs"); err != nil {
			return hlModified, err
		}
		nbNops -= 4
	}

	if nbNops >= 3 && canUseHL {
		if err := aw.WriteInstruction("ADD", "HL,BC", "Wait 3 NOPs"); err != nil {
			return hlModified, err
		}
		nbNops -= 3
		hlModified = true
	}

	if nbNops >= 2 {
		if err := aw.WriteInstruction("CP", "(HL)", "Wait 2 NOPs"); err != nil {
			return hlModified, err
		}
		nbNops -= 2
	}

	// Remaining single NOPs
	for nbNops > 0 {
		if err := aw.WriteInstruction("NOP", "", "Wait 1 NOP"); err != nil {
			return hlModified, err
		}
		nbNops--
	}

	return hlModified, nil
}

// generateSplitInit generates initialization code for split-raster
func (aw *ASMWriter) generateSplitInit() error {
	if err := aw.WriteInstruction("RUN", "$", ""); err != nil {
		return err
	}
	if err := aw.WriteBlankLine(); err != nil {
		return err
	}
	if err := aw.WriteNoList(); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DI", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "HL,ImageCmp", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "DE,#0200", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("CALL", "Depack", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "HL,#C9FB", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "(#38),HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "HL,Overscan", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,#BC", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("BclCrtc"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "A,(HL)", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "HL", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("AND", "A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "Z,SetPalette", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "B", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "B", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUTI", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DEC", "B", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "BclCrtc", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("SetPalette"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,#7F", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("BclPalette"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUT", "(C),A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "B", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("OUTI", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("INC", "A", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("CP", "16", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NZ,BclPalette", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("EI", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("HALT", "", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("Boucle"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("LD", "B,#F5", ""); err != nil {
		return err
	}

	if err := aw.WriteLabel("WaitVbl"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("IN", "A,(C)", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("RRA", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("JR", "NC,WaitVbl", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("EI", "", ""); err != nil {
		return err
	}
	if err := aw.WriteInstruction("HALT", "", ""); err != nil {
		return err
	}

	return aw.WriteInstruction("DI", "", "")
}

// generateSplitEnd generates end code and data tables
func (aw *ASMWriter) generateSplitEnd() error {
	if err := aw.WriteInstruction("JP", "Boucle", ""); err != nil {
		return err
	}
	if err := aw.WriteBlankLine(); err != nil {
		return err
	}

	// Generate ZX0 decompression routine
	if err := aw.GenerateZX0Depack(""); err != nil {
		return err
	}

	if err := aw.WriteBlankLine(); err != nil {
		return err
	}
	if err := aw.WriteList(); err != nil {
		return err
	}
	if err := aw.WriteBlankLine(); err != nil {
		return err
	}

	// CRTC overscan values
	if err := aw.WriteLabel("Overscan"); err != nil {
		return err
	}
	if err := aw.WriteInstruction("DB", "1,48,2,50,3,#8E,6,34,7,35,12,13,13,0,0", ""); err != nil {
		return err
	}

	// Default palette
	if err := aw.WriteLabel("Palette"); err != nil {
		return err
	}

	CpcVGA := "TDU\\X]LEMVFW^@_NGORBSZY[JCK"
	var paletteStr strings.Builder
	paletteStr.WriteString("")

	// Generate default palette (black background)
	for i := 0; i < 16; i++ {
		if i > 0 {
			paletteStr.WriteString(",")
		}
		paletteStr.WriteString(fmt.Sprintf("#%02X", int(CpcVGA[0]))) // All black for split-raster
	}

	return aw.WriteInstruction("DB", paletteStr.String(), "")
}