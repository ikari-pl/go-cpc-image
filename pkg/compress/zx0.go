// Package compress provides ZX0 compression algorithm implementation.
package compress

import (
	"errors"
)

const (
	MaxOffsetZX0 = 32640
	MaxScale     = 50
)

// Block represents a compression block in the optimal parser
type Block struct {
	Chain      *Block
	GhostChain *Block
	Bits       int
	Index      int
	Offset     int
	References int
	Length     int
}

// ZX0 represents the ZX0 compression engine
type ZX0 struct {
	ghostRoot   *Block
	outputIndex int
	bitIndex    int
	bitMask     int
	backTrack   bool
}

// NewZX0 creates a new ZX0 compressor
func NewZX0() *ZX0 {
	return &ZX0{}
}

// allocate allocates a new block with reference counting
func (zx0 *ZX0) allocate(bits, index, offset int, chain *Block) *Block {
	var ptr *Block

	if zx0.ghostRoot == nil {
		ptr = &Block{}
	} else {
		ptr = zx0.ghostRoot
		zx0.ghostRoot = ptr.GhostChain
		if ptr.Chain != nil {
			ptr.Chain.References--
			if ptr.Chain.References == 0 {
				ptr.Chain.GhostChain = zx0.ghostRoot
				zx0.ghostRoot = ptr.Chain
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

// assign assigns a chain with reference counting
func (zx0 *ZX0) assign(ptr **Block, chain *Block) {
	chain.References++
	if *ptr != nil {
		(*ptr).References--
		if (*ptr).References == 0 {
			(*ptr).GhostChain = zx0.ghostRoot
			zx0.ghostRoot = *ptr
		}
	}
	*ptr = chain
}

// eliasGammaBits calculates the number of bits needed for Elias gamma encoding
func (zx0 *ZX0) eliasGammaBits(value int) int {
	bits := 1
	for value >>= 1; value != 0; value >>= 1 {
		bits += 2
	}
	return bits
}

// writeBit writes a single bit to the output
func (zx0 *ZX0) writeBit(value int, outputData []byte) {
	if zx0.backTrack {
		if value != 0 {
			outputData[zx0.outputIndex-1] |= 1
		}
		zx0.backTrack = false
	} else {
		if zx0.bitMask == 0 {
			zx0.bitMask = 128
			zx0.bitIndex = zx0.outputIndex
			if zx0.outputIndex < len(outputData) {
				outputData[zx0.outputIndex] = 0
				zx0.outputIndex++
			}
		}
		if value != 0 && zx0.bitIndex < len(outputData) {
			outputData[zx0.bitIndex] |= byte(zx0.bitMask)
		}
		zx0.bitMask >>= 1
	}
}

// writeInterlacedEliasGamma writes interlaced Elias gamma encoding
func (zx0 *ZX0) writeInterlacedEliasGamma(value int, outputData []byte, invertMode bool) {
	var i int

	for i = 2; i <= value; i <<= 1 {
	}
	i >>= 1

	for i >>= 1; i != 0; i >>= 1 {
		zx0.writeBit(0, outputData)
		if invertMode {
			if (value & i) == 0 {
				zx0.writeBit(1, outputData)
			} else {
				zx0.writeBit(0, outputData)
			}
		} else {
			zx0.writeBit(value&i, outputData)
		}
	}
	zx0.writeBit(1, outputData)
}

// PackZX0 compresses data using the ZX0 algorithm
func (zx0 *ZX0) PackZX0(inputData []byte, inputSize int, outputData []byte, v2 bool) (int, error) {
	if inputSize <= 0 || inputSize > len(inputData) {
		return 0, errors.New("invalid input size")
	}

	maxOffset := inputSize - 1
	if maxOffset > MaxOffsetZX0 {
		maxOffset = MaxOffsetZX0
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
	chain := zx0.allocate(-1, -1, 1, nil)
	chain.References++
	if lastMatch[1] != nil {
		lastMatch[1].References--
		if lastMatch[1].References == 0 {
			lastMatch[1].GhostChain = zx0.ghostRoot
			zx0.ghostRoot = lastMatch[1]
		}
	}
	lastMatch[1] = chain

	// Optimal parsing
	for index := 0; index < inputSize; index++ {
		bestLengthSize := 2
		currentMaxOffset := index
		if currentMaxOffset > MaxOffsetZX0 {
			currentMaxOffset = MaxOffsetZX0
		}
		if currentMaxOffset < 1 {
			currentMaxOffset = 1
		}

		for offset := 1; offset <= currentMaxOffset; offset++ {
			if index != 0 && index >= offset && inputData[index] == inputData[index-offset] {
				if lastLiteral[offset] != nil {
					length := index - lastLiteral[offset].Index
					bits := lastLiteral[offset].Bits + 1 + zx0.eliasGammaBits(length)
					chain := zx0.allocate(bits, index, offset, lastLiteral[offset])
					chain.References++
					if lastMatch[offset] != nil {
						lastMatch[offset].References--
						if lastMatch[offset].References == 0 {
							lastMatch[offset].GhostChain = zx0.ghostRoot
							zx0.ghostRoot = lastMatch[offset]
						}
					}
					lastMatch[offset] = chain

					if tabOptimal[index] == nil || tabOptimal[index].Bits > bits {
						chain = lastMatch[offset]
						chain.References++
						if tabOptimal[index] != nil {
							tabOptimal[index].References--
							if tabOptimal[index].References == 0 {
								tabOptimal[index].GhostChain = zx0.ghostRoot
								zx0.ghostRoot = tabOptimal[index]
							}
						}
						tabOptimal[index] = chain
					}
				}

				matchLength[offset]++
				if matchLength[offset] > 1 {
					if bestLengthSize < matchLength[offset] {
						bits := tabOptimal[index-bestLength[bestLengthSize]].Bits + zx0.eliasGammaBits(bestLength[bestLengthSize]-1)
						for bestLengthSize < matchLength[offset] {
							bestLengthSize++
							bits2 := tabOptimal[index-bestLengthSize].Bits + zx0.eliasGammaBits(bestLengthSize-1)
							if bits2 <= bits {
								bestLength[bestLengthSize] = bestLengthSize
								bits = bits2
							} else {
								bestLength[bestLengthSize] = bestLength[bestLengthSize-1]
							}
						}
					}
					length := bestLength[matchLength[offset]]
					bits := tabOptimal[index-length].Bits + 8 + zx0.eliasGammaBits((offset-1)/128+1) + zx0.eliasGammaBits(length-1)

					if lastMatch[offset] == nil || lastMatch[offset].Index != index || lastMatch[offset].Bits > bits {
						chain := zx0.allocate(bits, index, offset, tabOptimal[index-length])
						chain.References++
						if lastMatch[offset] != nil {
							lastMatch[offset].References--
							if lastMatch[offset].References == 0 {
								lastMatch[offset].GhostChain = zx0.ghostRoot
								zx0.ghostRoot = lastMatch[offset]
							}
						}
						lastMatch[offset] = chain

						if tabOptimal[index] == nil || tabOptimal[index].Bits > bits {
							chain = lastMatch[offset]
							chain.References++
							if tabOptimal[index] != nil {
								tabOptimal[index].References--
								if tabOptimal[index].References == 0 {
									tabOptimal[index].GhostChain = zx0.ghostRoot
									zx0.ghostRoot = tabOptimal[index]
								}
							}
							tabOptimal[index] = chain
						}
					}
				}
			} else {
				matchLength[offset] = 0
				if lastMatch[offset] != nil {
					length := index - lastMatch[offset].Index
					bits := lastMatch[offset].Bits + 1 + zx0.eliasGammaBits(length) + (length << 3)
					chain := zx0.allocate(bits, index, 0, lastMatch[offset])
					chain.References++
					if lastLiteral[offset] != nil {
						lastLiteral[offset].References--
						if lastLiteral[offset].References == 0 {
							lastLiteral[offset].GhostChain = zx0.ghostRoot
							zx0.ghostRoot = lastLiteral[offset]
						}
					}
					lastLiteral[offset] = chain

					if tabOptimal[index] == nil || tabOptimal[index].Bits > bits {
						chain = lastLiteral[offset]
						chain.References++
						if tabOptimal[index] != nil {
							tabOptimal[index].References--
							if tabOptimal[index].References == 0 {
								tabOptimal[index].GhostChain = zx0.ghostRoot
								zx0.ghostRoot = tabOptimal[index]
							}
						}
						tabOptimal[index] = chain
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

	outputSize := (optimal.Bits + 25) / 8

	for optimal != nil {
		next = optimal.Chain
		optimal.Chain = prev
		prev = optimal
		optimal = next
	}

	// Generate output
	zx0.outputIndex = 0
	zx0.bitMask = 0
	lastOffset := 1
	inputIndex := 0
	zx0.backTrack = true

	for optimal = prev.Chain; optimal != nil; prev, optimal = optimal, optimal.Chain {
		length := optimal.Index - prev.Index

		if optimal.Offset == 0 {
			// Literal sequence
			zx0.writeBit(0, outputData)
			zx0.writeInterlacedEliasGamma(length, outputData, false)
			for i := 0; i < length && inputIndex+i < len(inputData) && zx0.outputIndex+i < len(outputData); i++ {
				outputData[zx0.outputIndex+i] = inputData[inputIndex+i]
			}
			zx0.outputIndex += length
			inputIndex += length
		} else if optimal.Offset == lastOffset {
			// Match with same offset
			zx0.writeBit(0, outputData)
			zx0.writeInterlacedEliasGamma(length, outputData, false)
			inputIndex += length
		} else {
			// Match with new offset
			zx0.writeBit(1, outputData)
			zx0.writeInterlacedEliasGamma((optimal.Offset-1)/128+1, outputData, v2)
			if zx0.outputIndex < len(outputData) {
				outputData[zx0.outputIndex] = byte((127 - (optimal.Offset-1)%128) << 1)
				zx0.outputIndex++
			}
			zx0.backTrack = true
			zx0.writeInterlacedEliasGamma(length-1, outputData, false)
			inputIndex += length
			lastOffset = optimal.Offset
		}
	}

	// End marker
	zx0.writeBit(1, outputData)
	zx0.writeInterlacedEliasGamma(256, outputData, v2)

	return outputSize, nil
}