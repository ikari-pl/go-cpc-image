// Package compress provides ZX1 compression algorithm implementation.
package compress

import (
	"errors"
)

const (
	MaxOffsetZX1 = 32512
)

// ZX1 represents the ZX1 compression engine
type ZX1 struct {
	ghostRoot   *Block
	outputIndex int
	bitIndex    int
	bitMask     int
}

// NewZX1 creates a new ZX1 compressor
func NewZX1() *ZX1 {
	return &ZX1{}
}

// allocate allocates a new block with reference counting (ZX1 specific)
func (zx1 *ZX1) allocate(bits, index, offset int, chain *Block) *Block {
	var ptr *Block

	if zx1.ghostRoot == nil {
		ptr = &Block{}
	} else {
		ptr = zx1.ghostRoot
		zx1.ghostRoot = ptr.GhostChain
		if ptr.Chain != nil {
			ptr.Chain.References--
			if ptr.Chain.References == 0 {
				ptr.Chain.GhostChain = zx1.ghostRoot
				zx1.ghostRoot = ptr.Chain
			}
		}
	}

	ptr.Bits = bits
	ptr.Index = index
	ptr.Offset = offset

	if chain != nil {
		chain.References++
	}

	ptr.Chain = chain
	ptr.References = 0
	return ptr
}

// assign assigns a chain with reference counting (ZX1 specific)
func (zx1 *ZX1) assign(ptr **Block, chain *Block) {
	chain.References++
	if *ptr != nil {
		(*ptr).References--
		if (*ptr).References == 0 {
			(*ptr).GhostChain = zx1.ghostRoot
			zx1.ghostRoot = *ptr
		}
	}
	*ptr = chain
}

// eliasGammaBits calculates the number of bits needed for Elias gamma encoding (ZX1 specific)
func (zx1 *ZX1) eliasGammaBits(value int) int {
	bits := 1
	for value >>= 1; value != 0; value >>= 1 {
		bits += 2
	}
	return bits
}

// writeBit writes a single bit to the output (ZX1 specific)
func (zx1 *ZX1) writeBit(value int, outputData []byte) {
	if zx1.bitMask == 0 {
		zx1.bitMask = 128
		zx1.bitIndex = zx1.outputIndex
		if zx1.outputIndex < len(outputData) {
			outputData[zx1.outputIndex] = 0
			zx1.outputIndex++
		}
	}
	if value != 0 && zx1.bitIndex < len(outputData) {
		outputData[zx1.bitIndex] |= byte(zx1.bitMask)
	}
	zx1.bitMask >>= 1
}

// writeInterlacedEliasGammaZX1 writes interlaced Elias gamma encoding (ZX1 format)
func (zx1 *ZX1) writeInterlacedEliasGammaZX1(value int, outputData []byte) {
	var i int

	for i = 2; i <= value; i <<= 1 {
	}
	i >>= 1

	for i >>= 1; i > 0; i >>= 1 {
		zx1.writeBit(1, outputData)
		zx1.writeBit(value&i, outputData)
	}
	zx1.writeBit(0, outputData)
}

// PackZX1 compresses data using the ZX1 algorithm
func (zx1 *ZX1) PackZX1(inputData []byte, inputSize int, outputData []byte) (int, error) {
	if inputSize <= 0 || inputSize > len(inputData) {
		return 0, errors.New("invalid input size")
	}

	maxOffset := inputSize - 1
	if maxOffset > MaxOffsetZX1 {
		maxOffset = MaxOffsetZX1
	}
	if maxOffset < 1 {
		maxOffset = 1
	}

	lastLiteral := make([]*Block, maxOffset+1)
	lastMatch := make([]*Block, maxOffset+1)
	tabOptimal := make([]*Block, inputSize+1)
	matchLength := make([]int, maxOffset+1)
	bestLength := make([]int, inputSize+2)

	bestLength[2] = 2
	zx1.assign(&lastMatch[1], zx1.allocate(-1, -1, 1, nil))

	// Optimal parsing
	for index := 0; index < inputSize; index++ {
		bestLengthSize := 2
		currentMaxOffset := index
		if currentMaxOffset > MaxOffsetZX1 {
			currentMaxOffset = MaxOffsetZX1
		}
		if currentMaxOffset < 1 {
			currentMaxOffset = 1
		}

		for offset := 1; offset <= currentMaxOffset; offset++ {
			if index != 0 && index >= offset && inputData[index] == inputData[index-offset] {
				if lastLiteral[offset] != nil {
					length := index - lastLiteral[offset].Index
					bits := lastLiteral[offset].Bits + 1 + zx1.eliasGammaBits(length)
					zx1.assign(&lastMatch[offset], zx1.allocate(bits, index, offset, lastLiteral[offset]))

					if tabOptimal[index] == nil || tabOptimal[index].Bits > bits {
						zx1.assign(&tabOptimal[index], lastMatch[offset])
					}
				}

				matchLength[offset]++
				if matchLength[offset] > 1 {
					if bestLengthSize < matchLength[offset] {
						bits := tabOptimal[index-bestLength[bestLengthSize]].Bits + zx1.eliasGammaBits(bestLength[bestLengthSize]-1)
						for bestLengthSize < matchLength[offset] {
							bestLengthSize++
							bits2 := tabOptimal[index-bestLengthSize].Bits + zx1.eliasGammaBits(bestLengthSize-1)
							if bits2 <= bits {
								bestLength[bestLengthSize] = bestLengthSize
								bits = bits2
							} else {
								bestLength[bestLengthSize] = bestLength[bestLengthSize-1]
							}
						}
					}

					length := bestLength[matchLength[offset]]
					offsetBits := 8
					if offset > 128 {
						offsetBits = 16
					}
					bits := tabOptimal[index-length].Bits + 1 + offsetBits + zx1.eliasGammaBits(length-1)

					if lastMatch[offset] == nil || lastMatch[offset].Index != index || lastMatch[offset].Bits > bits {
						zx1.assign(&lastMatch[offset], zx1.allocate(bits, index, offset, tabOptimal[index-length]))
						if tabOptimal[index] == nil || tabOptimal[index].Bits > bits {
							zx1.assign(&tabOptimal[index], lastMatch[offset])
						}
					}
				}
			} else {
				matchLength[offset] = 0
				if lastMatch[offset] != nil {
					length := index - lastMatch[offset].Index
					bits := lastMatch[offset].Bits + 1 + zx1.eliasGammaBits(length) + length*8
					zx1.assign(&lastLiteral[offset], zx1.allocate(bits, index, 0, lastMatch[offset]))

					if tabOptimal[index] == nil || tabOptimal[index].Bits > bits {
						zx1.assign(&tabOptimal[index], lastLiteral[offset])
					}
				}
			}
		}
	}

	// Build output chain
	var prev *Block
	var next *Block
	optimal := tabOptimal[inputSize-1]
	if optimal == nil {
		return 0, errors.New("failed to find optimal solution")
	}

	outputSize := (optimal.Bits + 24) / 8

	for optimal != nil {
		prev = optimal.Chain
		optimal.Chain = next
		next = optimal
		optimal = prev
	}

	// Generate output
	zx1.outputIndex = 0
	zx1.bitMask = 0
	lastOffset := 1
	inputIndex := 0
	first := true

	for optimal = next.Chain; optimal != nil; optimal = optimal.Chain {
		if optimal.Offset == 0 {
			// Literal sequence
			if first {
				first = false
			} else {
				zx1.writeBit(0, outputData)
			}

			zx1.writeInterlacedEliasGammaZX1(optimal.Length, outputData)
			for i := 0; i < optimal.Length && inputIndex+i < len(inputData) && zx1.outputIndex+i < len(outputData); i++ {
				outputData[zx1.outputIndex+i] = inputData[inputIndex+i]
			}
			zx1.outputIndex += optimal.Length
			inputIndex += optimal.Length
		} else if optimal.Offset == lastOffset {
			// Match with same offset
			zx1.writeBit(0, outputData)
			zx1.writeInterlacedEliasGammaZX1(optimal.Length, outputData)
			inputIndex += optimal.Length
		} else {
			// Match with new offset
			zx1.writeBit(1, outputData)

			if optimal.Offset > 128 {
				// 16-bit offset encoding
				if zx1.outputIndex < len(outputData) {
					outputData[zx1.outputIndex] = byte(255 - ((optimal.Offset-1)&254))
					zx1.outputIndex++
				}
				if zx1.outputIndex < len(outputData) {
					outputData[zx1.outputIndex] = byte(252 - (optimal.Offset-1)/256*2 + optimal.Offset%2)
					zx1.outputIndex++
				}
			} else {
				// 8-bit offset encoding
				if zx1.outputIndex < len(outputData) {
					outputData[zx1.outputIndex] = byte(256 - optimal.Offset*2)
					zx1.outputIndex++
				}
			}

			zx1.writeInterlacedEliasGammaZX1(optimal.Length-1, outputData)
			inputIndex += optimal.Length
			lastOffset = optimal.Offset
		}
	}

	// End marker
	zx1.writeBit(1, outputData)
	if zx1.outputIndex < len(outputData) {
		outputData[zx1.outputIndex] = 255
		zx1.outputIndex++
	}
	if zx1.outputIndex < len(outputData) {
		outputData[zx1.outputIndex] = 255
		zx1.outputIndex++
	}

	return outputSize, nil
}