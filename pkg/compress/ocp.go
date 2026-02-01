// Package compress provides OCP (OCP Art Studio) compression/decompression using MJH header format.
package compress

import (
	"errors"
)

const (
	// MarkerOCP is the RLE marker byte used in OCP compression
	MarkerOCP byte = 0x01
)

// OCP represents the OCP Art Studio compression engine using MJH headers
type OCP struct{}

// NewOCP creates a new OCP compressor/decompressor
func NewOCP() *OCP {
	return &OCP{}
}

// DepackOCP decompresses OCP (MJH header) compressed data.
// The input data contains one or more MJH blocks. Each block has:
//   - 3 bytes: "MJH" signature
//   - 2 bytes: little-endian output length for this block
//   - compressed data using RLE with marker byte 0x01
//
// Returns the number of bytes written to bufOut.
func (o *OCP) DepackOCP(bufIn []byte, bufOut []byte) (int, error) {
	if len(bufIn) < 5 {
		return 0, errors.New("input too short for OCP data")
	}

	posIn := 0
	posOut := 0
	maxOut := len(bufOut)

	for posOut < maxOut {
		// Check for MJH header
		if posIn+4 >= len(bufIn) {
			break
		}
		if bufIn[posIn] != 'M' || bufIn[posIn+1] != 'J' || bufIn[posIn+2] != 'H' {
			break
		}

		posIn += 3
		lgOut := int(bufIn[posIn]) | (int(bufIn[posIn+1]) << 8)
		posIn += 2

		cntBlock := 0
		for cntBlock < lgOut {
			if posIn >= len(bufIn) {
				break
			}
			// Check for next MJH header (end of current block)
			if posIn+2 < len(bufIn) && bufIn[posIn] == 'M' && bufIn[posIn+1] == 'J' && bufIn[posIn+2] == 'H' {
				break
			}

			a := bufIn[posIn]
			posIn++

			if a == MarkerOCP {
				// RLE: next byte is count, then byte to repeat
				if posIn+1 >= len(bufIn) {
					break
				}
				c := int(bufIn[posIn])
				posIn++
				a = bufIn[posIn]
				posIn++
				if c == 0 {
					c = 0x100
				}

				for i := 0; i < c && cntBlock < lgOut; i++ {
					if posOut >= maxOut {
						break
					}
					bufOut[posOut] = a
					posOut++
					cntBlock++
				}
			} else {
				// Literal byte
				if posOut >= maxOut {
					break
				}
				bufOut[posOut] = a
				posOut++
				cntBlock++
			}
		}
	}

	return posOut, nil
}

// PackOCP compresses data using OCP (MJH header) RLE compression.
// The output contains MJH blocks with RLE-encoded data using marker byte 0x01.
// blockSize controls how large each MJH block's decompressed size is (typically 0x4000).
// Returns the number of bytes written to bufOut.
func (o *OCP) PackOCP(bufIn []byte, lengthIn int, bufOut []byte, blockSize int) (int, error) {
	if lengthIn <= 0 || lengthIn > len(bufIn) {
		return 0, errors.New("invalid input size")
	}
	if blockSize <= 0 {
		blockSize = 0x4000
	}

	posIn := 0
	posOut := 0

	for posIn < lengthIn {
		// Determine block decompressed size
		remaining := lengthIn - posIn
		lgOut := blockSize
		if lgOut > remaining {
			lgOut = remaining
		}

		// Write MJH header
		if posOut+5 > len(bufOut) {
			return 0, errors.New("output buffer too small")
		}
		bufOut[posOut] = 'M'
		bufOut[posOut+1] = 'J'
		bufOut[posOut+2] = 'H'
		bufOut[posOut+3] = byte(lgOut & 0xFF)
		bufOut[posOut+4] = byte((lgOut >> 8) & 0xFF)
		posOut += 5

		// RLE compress block
		blockEnd := posIn + lgOut
		for posIn < blockEnd {
			a := bufIn[posIn]

			// Count consecutive identical bytes
			runLen := 1
			for posIn+runLen < blockEnd && bufIn[posIn+runLen] == a && runLen < 0x100 {
				runLen++
			}

			if a == MarkerOCP || runLen >= 3 {
				// Encode as RLE
				if posOut+3 > len(bufOut) {
					return 0, errors.New("output buffer too small")
				}
				bufOut[posOut] = MarkerOCP
				posOut++
				if runLen == 0x100 {
					bufOut[posOut] = 0 // 0 means 256
				} else {
					bufOut[posOut] = byte(runLen)
				}
				posOut++
				bufOut[posOut] = a
				posOut++
				posIn += runLen
			} else {
				// Literal byte
				if posOut >= len(bufOut) {
					return 0, errors.New("output buffer too small")
				}
				bufOut[posOut] = a
				posOut++
				posIn++
			}
		}
	}

	return posOut, nil
}
