// Package compress provides compression and decompression functionality for various formats.
package compress

import (
	"errors"
)

// Compressor provides a unified interface for different compression algorithms
type Compressor struct {
	lzw *LZW
	zx0 *ZX0
	zx1 *ZX1
}

// NewCompressor creates a new unified compressor
func NewCompressor() *Compressor {
	return &Compressor{
		lzw: NewLZW(),
		zx0: NewZX0(),
		zx1: NewZX1(),
	}
}

// Pack compresses data using the specified method
func (c *Compressor) Pack(bufIn []byte, lengthIn int, bufOut []byte, lengthOut int, pkMethod PackMethod) (int, error) {
	switch pkMethod {
	case Standard:
		return c.lzw.PackStd(bufIn, lengthIn, bufOut, lengthOut)

	case MethodZX0:
		return c.zx0.PackZX0(bufIn, lengthIn, bufOut, false)

	case MethodZX0V2:
		return c.zx0.PackZX0(bufIn, lengthIn, bufOut, true)

	case MethodZX1:
		return c.zx1.PackZX1(bufIn, lengthIn, bufOut)

	case MethodZX0Ovs:
		// ZX0 with screen overlays handling
		bufTmp := make([]byte, 0x8000)
		posIn := 0
		posOut := 0

		for i := 0; i < 15; i++ {
			var lIn int
			if i < 7 {
				lIn = 0x600
			} else if i == 7 {
				lIn = 0xCC0
			} else {
				lIn = 0x6C0
			}

			if posIn+lIn <= lengthIn && posOut+lIn <= len(bufTmp) {
				copy(bufTmp[posOut:posOut+lIn], bufIn[posIn:posIn+lIn])
			}

			if i != 7 {
				posIn += 0x800
			} else {
				posIn += 0xE00
			}
			posOut += lIn
		}

		return c.zx0.PackZX0(bufTmp, posOut, bufOut, false)

	default:
		return 0, errors.New("unsupported compression method")
	}
}

// Depack decompresses data using the specified method
func (c *Compressor) Depack(bufIn []byte, startIn int, bufOut []byte, pkMethod PackMethod) (int, error) {
	switch pkMethod {
	case Standard:
		return c.lzw.DepackStd(bufIn, startIn, bufOut)

	default:
		return 0, errors.New("unsupported decompression method")
	}
}