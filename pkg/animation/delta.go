// Package animation provides delta-difference encoding for CPC animation frames.
// This handles frame-to-frame differences for efficient animation storage.
package animation

import (
	"fmt"
	"io"
	"math"
)

// DeltaDiff represents a block of consecutive byte changes at a specific address.
// This is equivalent to the C# DeltaDiff class.
type DeltaDiff struct {
	StartAddress int     // Starting address for this diff block
	LstDelta     []byte  // List of byte values to change
}

// NewDeltaDiff creates a new delta difference block at the specified address.
func NewDeltaDiff(startAddress int) *DeltaDiff {
	return &DeltaDiff{
		StartAddress: startAddress,
		LstDelta:     make([]byte, 0),
	}
}

// AddValue adds a byte value to this delta block.
func (dd *DeltaDiff) AddValue(value byte) {
	dd.LstDelta = append(dd.LstDelta, value)
}

// Length returns the number of bytes in this delta block.
func (dd *DeltaDiff) Length() int {
	return len(dd.LstDelta)
}

// DiffAnim manages delta-difference encoding between animation frames.
// This is equivalent to the C# DiffAnim class.
type DiffAnim struct {
	Delta   []*DeltaDiff // List of delta difference blocks
	LastAdr int          // Last address written (for optimization)
}

// NewDiffAnim creates a new delta animation encoder.
func NewDiffAnim() *DiffAnim {
	return &DiffAnim{
		Delta:   make([]*DeltaDiff, 0),
		LastAdr: 0x10000, // Initialize to invalid address
	}
}

// AddDiff adds a byte difference at the specified address.
// This automatically creates new blocks when addresses are not consecutive.
func (da *DiffAnim) AddDiff(adr int, data byte) {
	// If this address is not consecutive with the last one, start a new block
	if adr != da.LastAdr+1 {
		da.Delta = append(da.Delta, NewDeltaDiff(adr))
	}

	// Add the byte to the current (last) delta block
	if len(da.Delta) > 0 {
		da.Delta[len(da.Delta)-1].AddValue(data)
	}

	da.LastAdr = adr
}

// Clear resets the delta animation to empty state.
func (da *DiffAnim) Clear() {
	da.Delta = make([]*DeltaDiff, 0)
	da.LastAdr = 0x10000
}

// GetBlockCount returns the number of delta blocks.
func (da *DiffAnim) GetBlockCount() int {
	return len(da.Delta)
}

// GetTotalBytes returns the total number of changed bytes across all blocks.
func (da *DiffAnim) GetTotalBytes() int {
	total := 0
	for _, d := range da.Delta {
		total += len(d.LstDelta)
	}
	return total
}

// Save writes the delta animation as Z80 assembly source code.
// This replicates the C# Save method behavior.
func (da *DiffAnim) Save(w io.Writer, nbOctetsLigne int) error {
	for _, d := range da.Delta {
		length := len(d.LstDelta)

		// Write address
		if _, err := fmt.Fprintf(w, "\tDW\t#%04X\t;adresse\n", d.StartAddress); err != nil {
			return err
		}

		// Write byte count
		if _, err := fmt.Fprintf(w, "\tDB\t#%02X\t;nbre octets\n", length); err != nil {
			return err
		}

		// Write data bytes
		nbOctets := 0
		line := "\tDB\t"

		for i, b := range d.LstDelta {
			line += fmt.Sprintf("#%02X,", b)
			nbOctets++

			// Break line when reaching maximum bytes per line or total columns
			maxBytesPerLine := int(math.Min(float64(nbOctetsLigne), float64(80))) // Assuming NumCol = 80
			if nbOctets >= maxBytesPerLine {
				// Remove trailing comma and write line
				if len(line) > 0 && line[len(line)-1] == ',' {
					line = line[:len(line)-1]
				}
				if _, err := fmt.Fprintln(w, line); err != nil {
					return err
				}

				line = "\tDB\t"
				nbOctets = 0

				// Add line break if not the last byte
				if i < length-1 {
					if _, err := fmt.Fprintln(w, ""); err != nil {
						return err
					}
				}
			}
		}

		// Write remaining bytes if any
		if nbOctets > 0 {
			// Remove trailing comma
			if len(line) > 0 && line[len(line)-1] == ',' {
				line = line[:len(line)-1]
			}
			if _, err := fmt.Fprintln(w, line); err != nil {
				return err
			}
		}
	}

	// Write end marker if any blocks were written
	if len(da.Delta) > 0 {
		if _, err := fmt.Fprintf(w, "\tDW\t#0000\t;fin de table\n"); err != nil {
			return err
		}
	}

	return nil
}

// SaveBinary writes the delta animation as binary data.
// Returns the binary representation suitable for direct CPC memory loading.
func (da *DiffAnim) SaveBinary() []byte {
	var result []byte

	for _, d := range da.Delta {
		// Add address (little-endian word)
		result = append(result, byte(d.StartAddress&0xFF))
		result = append(result, byte(d.StartAddress>>8))

		// Add length
		length := len(d.LstDelta)
		result = append(result, byte(length))

		// Add data bytes
		result = append(result, d.LstDelta...)
	}

	// Add end marker if any blocks exist
	if len(da.Delta) > 0 {
		result = append(result, 0x00, 0x00) // Address 0x0000 marks end
	}

	return result
}

// CompareFrames compares two frame buffers and generates delta differences.
// This is used to create delta animations between consecutive frames.
func (da *DiffAnim) CompareFrames(frame1, frame2 []byte, baseAddress int) {
	da.Clear() // Start with clean state

	minLength := len(frame1)
	if len(frame2) < minLength {
		minLength = len(frame2)
	}

	for i := 0; i < minLength; i++ {
		if frame1[i] != frame2[i] {
			da.AddDiff(baseAddress+i, frame2[i])
		}
	}

	// Handle case where frame2 is longer than frame1
	for i := minLength; i < len(frame2); i++ {
		da.AddDiff(baseAddress+i, frame2[i])
	}
}

// ApplyDelta applies delta differences to a frame buffer.
// This reconstructs a frame by applying the stored deltas to a base frame.
func ApplyDelta(baseFrame []byte, deltas *DiffAnim, baseAddress int) []byte {
	// Create a copy of the base frame
	result := make([]byte, len(baseFrame))
	copy(result, baseFrame)

	// Apply each delta block
	for _, d := range deltas.Delta {
		startOffset := d.StartAddress - baseAddress
		if startOffset >= 0 && startOffset < len(result) {
			for i, b := range d.LstDelta {
				offset := startOffset + i
				if offset < len(result) {
					result[offset] = b
				}
			}
		}
	}

	return result
}

// OptimizeDelta optimizes the delta by merging nearby blocks and removing redundant data.
func (da *DiffAnim) OptimizeDelta(maxGap int) {
	if len(da.Delta) <= 1 {
		return
	}

	optimized := []*DeltaDiff{}
	current := da.Delta[0]

	for i := 1; i < len(da.Delta); i++ {
		next := da.Delta[i]

		// Calculate gap between current block end and next block start
		gap := next.StartAddress - (current.StartAddress + len(current.LstDelta))

		// Merge if gap is small enough
		if gap <= maxGap {
			// Fill gap with zeros if needed
			for j := 0; j < gap; j++ {
				current.AddValue(0x00)
			}
			// Append next block's data
			current.LstDelta = append(current.LstDelta, next.LstDelta...)
		} else {
			// Gap too large, start new block
			optimized = append(optimized, current)
			current = next
		}
	}

	// Add the last block
	optimized = append(optimized, current)

	da.Delta = optimized
}

/*
Assembly code template for loading delta animations on CPC:

LD	HL,DiffImage_1
Next:
	CALL	Loop
	LD	A,(HL)
	INC	HL
	OR	(HL)
	DEC	HL
	RET	Z
	JR	Next

Loop:
	LD	E,(HL)      ; Get low byte of address
	INC	HL
	LD	D,(HL)      ; Get high byte of address
	INC	HL
	LD	A,D         ; Check if address is 0000 (end marker)
	OR	E
	RET	Z           ; Return if end of data
	LD	A,(HL)      ; Get byte count
	INC	HL
	EX	DE,HL       ; DE = data pointer, HL = target address
	LD	BC,#C000    ; Add CPC screen base address
	ADD	HL,BC
	EX	DE,HL       ; HL = data pointer, DE = target address
	LD	B,0         ; BC = byte count
	LD	C,A
	LDIR            ; Copy BC bytes from HL to DE
	JR	Loop
*/