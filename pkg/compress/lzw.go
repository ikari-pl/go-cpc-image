// Package compress provides compression and decompression algorithms.
package compress

import (
	"errors"
)

const (
	SeekBack  = 0x1000
	MaxString = 256
)

// PackMethod represents the compression method
type PackMethod int

const (
	Standard PackMethod = iota
	MethodZX0
	MethodZX0V2
	MethodZX1
	MethodZX0Ovs
)

// LZW represents the LZW compression/decompression engine
type LZW struct {
	matches     [MaxString]int
	matchtable  [MaxString][SeekBack]int
	outputIndex int
	bitIndex    int
	bitMask     int
	backTrack   bool
}

// NewLZW creates a new LZW compressor/decompressor
func NewLZW() *LZW {
	return &LZW{}
}

// DepackStd decompresses data using the standard LZW algorithm
func (lzw *LZW) DepackStd(bufIn []byte, startIn int, bufOut []byte) (int, error) {
	if startIn >= len(bufIn) {
		return 0, errors.New("startIn index out of range")
	}

	var depackBits byte = 0
	var bit int
	inBytes := startIn
	outBytes := 0

	for {
		bit = int(depackBits & 1)
		depackBits >>= 1

		if depackBits == 0 {
			if inBytes >= len(bufIn) {
				break
			}
			depackBits = bufIn[inBytes]
			inBytes++
			bit = int(depackBits & 1)
			depackBits >>= 1
			depackBits |= 0x80
		}

		if bit == 0 {
			// Literal byte
			if inBytes >= len(bufIn) || outBytes >= len(bufOut) {
				break
			}
			bufOut[outBytes] = bufIn[inBytes]
			outBytes++
			inBytes++
		} else {
			// Match
			if inBytes >= len(bufIn) {
				break
			}

			if bufIn[inBytes] == 0 {
				break // EOF
			}

			a := bufIn[inBytes]
			var length, delta int

			if (a & 0x80) != 0 {
				// Format: 1aaabbbb cccccccc
				length = 3 + int((bufIn[inBytes]>>4)&7)
				delta = int(bufIn[inBytes]&15) << 8
				inBytes++
				if inBytes >= len(bufIn) {
					break
				}
				delta |= int(bufIn[inBytes])
				inBytes++
				delta++
			} else if (a & 0x40) != 0 {
				// Format: 01aaaaaa
				length = 2
				delta = int(bufIn[inBytes] & 0x3f)
				inBytes++
				delta++
			} else if (a & 0x20) != 0 {
				// Format: 001aaaaa bbbbbbbb
				length = 2 + int(bufIn[inBytes]&31)
				inBytes++
				if inBytes >= len(bufIn) {
					break
				}
				delta = int(bufIn[inBytes])
				inBytes++
				delta++
			} else if (a & 0x10) != 0 {
				// Format: 0001aaaa bbbbbbbb cccccccc
				delta = int(bufIn[inBytes]&15) << 8
				inBytes++
				if inBytes >= len(bufIn) {
					break
				}
				delta |= int(bufIn[inBytes])
				inBytes++
				if inBytes >= len(bufIn) {
					break
				}
				length = int(bufIn[inBytes]) + 1
				inBytes++
				delta++
			} else {
				// Special format
				if bufIn[inBytes] == 15 {
					if inBytes+1 >= len(bufIn) {
						break
					}
					length = int(bufIn[inBytes+1]) + 1
					delta = length
					inBytes += 2
				} else {
					if bufIn[inBytes] > 1 {
						length = int(bufIn[inBytes])
						delta = length
					} else {
						length = 256
						delta = 256
					}
					inBytes++
				}
			}

			// Copy match
			for length > 0 {
				if outBytes >= len(bufOut) || outBytes < delta {
					break
				}
				bufOut[outBytes] = bufOut[outBytes-delta]
				outBytes++
				length--
			}
		}
	}

	return outBytes, nil
}

// PackStd compresses data using the standard LZW algorithm
func (lzw *LZW) PackStd(bufIn []byte, lengthIn int, bufOut []byte, lengthOut int) (int, error) {
	if lengthIn > len(bufIn) || lengthOut > len(bufOut) {
		return 0, errors.New("buffer size mismatch")
	}

	codebuffer := make([]byte, 24)
	var bits byte = 0
	count := 0
	bitcount := 0
	codecount := 0
	matchtablestart := 0
	matchtableend := 0

	// Initialize matches array
	for i := range lzw.matches {
		lzw.matches[i] = 0
	}

	for {
		// Update match table
		for c := matchtableend; c < count; c++ {
			if c >= lengthIn {
				break
			}
			b := int(bufIn[c])
			lzw.matchtable[b][lzw.matches[b]] = c
			lzw.matches[b]++
		}
		matchtableend = count

		if count >= 2 && count < lengthIn {
			stlen := 0
			stpos := 0
			stlen2 := 0

			// Search for matches starting at current position
			bb := int(bufIn[count])
			for c := lzw.matches[bb] - 1; c >= 0; c-- {
				start := lzw.matchtable[bb][c]
				end := start + MaxString
				if end > count {
					end = count
				}

				max := end - start
				if max >= stlen {
					var d int
					for d = 1; d < max; d++ {
						if start+d >= lengthIn || count+d >= lengthIn {
							break
						}
						if bufIn[start+d] != bufIn[count+d] {
							break
						}
					}

					if d >= 2 && d > stlen {
						stlen = d
						stpos = count - start
					}
					if d == stlen && count-start < stpos {
						stpos = count - start
					}
				}

				if stlen == MaxString && stpos == stlen {
					break
				}
			}

			// Check next position for better match
			if count+1 < lengthIn {
				bb = int(bufIn[count+1])
				for c := lzw.matches[bb] - 1; c >= 0; c-- {
					start := lzw.matchtable[bb][c]
					end := start + MaxString
					if end > count+1 {
						end = count + 1
					}

					max := end - start
					if max >= stlen2 {
						var d int
						for d = 1; d < max; d++ {
							if start+d >= lengthIn || count+d+1 >= lengthIn {
								break
							}
							if bufIn[start+d] != bufIn[count+d+1] {
								break
							}
						}

						if d >= 2 && d >= stlen2 {
							stlen2 = d
						}
					}
					if stlen2 == MaxString {
						break
					}
				}
				if stlen2-1 > stlen {
					stlen = 0
				}
			}

			if stlen > 1 {
				if stlen == 2 && stpos >= MaxString {
					// Output literal
					if codecount >= len(codebuffer) || count >= lengthIn {
						break
					}
					codebuffer[codecount] = bufIn[count]
					codecount++
					count++
					bitcount++
				} else {
					// Output match
					if stpos == stlen {
						if stlen == MaxString {
							if codecount >= len(codebuffer) {
								break
							}
							codebuffer[codecount] = 0x01
							codecount++
						} else {
							if codecount >= len(codebuffer) {
								break
							}
							if stlen <= 14 {
								codebuffer[codecount] = byte(stlen)
							} else {
								codebuffer[codecount] = 0x0F
								codecount++
								if codecount >= len(codebuffer) {
									break
								}
								codebuffer[codecount] = byte(stlen - 1)
							}
							codecount++
						}
					} else {
						if stlen == 2 && stpos < 65 {
							if codecount >= len(codebuffer) {
								break
							}
							codebuffer[codecount] = byte(0x40 + stpos - 1)
							codecount++
						} else if stlen <= 33 && stpos < 257 {
							if codecount+1 >= len(codebuffer) {
								break
							}
							codebuffer[codecount] = byte(0x20 + stlen - 2)
							codecount++
							codebuffer[codecount] = byte(stpos - 1)
							codecount++
						} else {
							if stlen >= 3 && stlen <= 10 {
								if codecount+1 >= len(codebuffer) {
									break
								}
								codebuffer[codecount] = byte(0x80 + ((stlen-3)<<4) + ((stpos-1)>>8))
								codecount++
								codebuffer[codecount] = byte(stpos - 1)
								codecount++
							} else {
								if codecount+2 >= len(codebuffer) {
									break
								}
								codebuffer[codecount] = byte(0x10 + ((stpos-1)>>8))
								codecount++
								codebuffer[codecount] = byte(stpos - 1)
								codecount++
								codebuffer[codecount] = byte(stlen - 1)
								codecount++
							}
						}
					}
					bits |= 1 << bitcount
					bitcount++
					count += stlen
				}
			} else {
				// Output literal
				if codecount >= len(codebuffer) || count >= lengthIn {
					break
				}
				codebuffer[codecount] = bufIn[count]
				codecount++
				count++
				bitcount++
			}
		} else {
			// Output literal
			if codecount >= len(codebuffer) || count >= lengthIn {
				break
			}
			codebuffer[codecount] = bufIn[count]
			codecount++
			count++
			bitcount++
		}

		// Output buffer when full
		if bitcount == 8 {
			if lengthOut >= len(bufOut) {
				break
			}
			bufOut[lengthOut] = bits
			lengthOut++

			if lengthOut+codecount > len(bufOut) {
				break
			}
			copy(bufOut[lengthOut:], codebuffer[:codecount])
			lengthOut += codecount
			bitcount = 0
			codecount = 0
			bits = 0
		}

		if count >= lengthIn {
			break
		}

		// Manage sliding window
		oldmatchtablestart := matchtablestart
		matchtablestart = count - SeekBack
		if matchtablestart < 0 {
			matchtablestart = 0
		}

		for c := oldmatchtablestart; c < matchtablestart; c++ {
			if c >= lengthIn {
				break
			}
			b := int(bufIn[c])
			var d int
			for d = 0; d < lzw.matches[b]; d++ {
				if lzw.matchtable[b][d] >= matchtablestart {
					copy(lzw.matchtable[b][d:], lzw.matchtable[b][d:lzw.matches[b]])
					break
				}
			}
			lzw.matches[b] -= d
		}
	}

	// Final output
	if codecount < len(codebuffer) {
		codebuffer[codecount] = 0
		codecount++
	}

	if lengthOut < len(bufOut) {
		bufOut[lengthOut] = bits | (1 << bitcount)
		lengthOut++
	}

	if lengthOut+codecount <= len(bufOut) {
		copy(bufOut[lengthOut:], codebuffer[:codecount])
		lengthOut += codecount
	}

	return lengthOut, nil
}

// Depack decompresses data using the specified method
func (lzw *LZW) Depack(bufIn []byte, startIn int, bufOut []byte, pkMethod PackMethod) (int, error) {
	switch pkMethod {
	case Standard:
		return lzw.DepackStd(bufIn, startIn, bufOut)
	default:
		return 0, errors.New("unsupported decompression method")
	}
}

// Pack compresses data using the specified method
func (lzw *LZW) Pack(bufIn []byte, lengthIn int, bufOut []byte, lengthOut int, pkMethod PackMethod) (int, error) {
	switch pkMethod {
	case Standard:
		return lzw.PackStd(bufIn, lengthIn, bufOut, lengthOut)
	default:
		return 0, errors.New("unsupported compression method")
	}
}