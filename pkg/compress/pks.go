// Package compress provides PKS family packed format support.
// PKS variants:
//   - PKSL: 320x200 Standard
//   - PKS3: 320x200 Mode 3
//   - PKSP: 320x200 Plus
//   - PKVL: Overscan Standard
//   - PKVP: Overscan Plus
package compress

// PKSVariant identifies a specific PKS format variant
type PKSVariant int

const (
	// PKSL is 320x200 Standard
	PKSL PKSVariant = iota
	// PKS3 is 320x200 Mode 3
	PKS3
	// PKSP is 320x200 Plus
	PKSP
	// PKVL is Overscan Standard
	PKVL
	// PKVP is Overscan Plus
	PKVP
)

// PKSHeader holds parsed PKS file header information
type PKSHeader struct {
	Variant  PKSVariant
	CpcPlus  bool
	Overscan bool
	// Palette holds up to 17 bytes of palette data (for PKSL: ink values at offset 4..20)
	Palette [17]byte
	// DataOffset is where the compressed data starts in the input buffer
	DataOffset int
}

// PKS represents the PKS family compression engine.
// PKS formats use the Standard (LZW-style) compression from PackModule
// with a 4-byte signature header and optional embedded palette.
type PKS struct {
	lzw *LZW
}

// NewPKS creates a new PKS compressor/decompressor
func NewPKS() *PKS {
	return &PKS{
		lzw: NewLZW(),
	}
}

// ParsePKSHeader reads and interprets a PKS header from the input buffer.
// The first 4 bytes are the signature: "PK" + two variant bytes.
// Returns header info describing the variant and where compressed data begins.
func ParsePKSHeader(bufIn []byte) (*PKSHeader, error) {
	if len(bufIn) < 4 {
		return nil, ErrInputTooShort
	}
	if bufIn[0] != 'P' || bufIn[1] != 'K' {
		return nil, ErrInvalidSignature
	}

	h := &PKSHeader{}

	// Determine variant from bytes 2 and 3
	b2 := bufIn[2]
	b3 := bufIn[3]

	h.CpcPlus = (b3 == 'P') || (b2 == 'O')
	h.Overscan = (b2 == 'V') || (b3 == 'V')

	switch {
	case b2 == 'S' && b3 == 'L':
		h.Variant = PKSL
	case b2 == 'S' && b3 == '3':
		h.Variant = PKS3
	case b2 == 'S' && b3 == 'P':
		h.Variant = PKSP
	case b2 == 'V' && b3 == 'L':
		h.Variant = PKVL
	case b2 == 'V' && b3 == 'P':
		h.Variant = PKVP
	case b2 == 'O':
		// PKO* variants (plus overscan)
		h.Variant = PKVP
		h.CpcPlus = true
		h.Overscan = true
	default:
		return nil, ErrUnknownVariant
	}

	// PKSL has 17 bytes of palette at offset 4, then data at offset 21
	if h.Variant == PKSL {
		if len(bufIn) < 21 {
			return nil, ErrInputTooShort
		}
		copy(h.Palette[:], bufIn[4:21])
		h.DataOffset = 21
	} else {
		h.DataOffset = 4
	}

	return h, nil
}

// DepackPKS decompresses a PKS-family packed file.
// It parses the header, then decompresses the payload using the Standard (LZW) algorithm.
// Returns the decompressed size and parsed header information.
func (p *PKS) DepackPKS(bufIn []byte, bufOut []byte) (int, *PKSHeader, error) {
	header, err := ParsePKSHeader(bufIn)
	if err != nil {
		return 0, nil, err
	}

	n, err := p.lzw.DepackStd(bufIn, header.DataOffset, bufOut)
	if err != nil {
		return 0, header, err
	}

	return n, header, nil
}

// PackPKS compresses data into PKS format with the given variant.
// It writes the appropriate header, optional palette, then Standard-compressed data.
// palette is only used for PKSL variant (17 bytes of ink values).
// Returns the total number of bytes written to bufOut.
func (p *PKS) PackPKS(bufIn []byte, lengthIn int, bufOut []byte, variant PKSVariant, palette []byte) (int, error) {
	if lengthIn <= 0 || lengthIn > len(bufIn) {
		return 0, ErrInvalidInputSize
	}

	pos := 0

	// Write signature
	if pos+4 > len(bufOut) {
		return 0, ErrOutputTooSmall
	}
	bufOut[0] = 'P'
	bufOut[1] = 'K'

	switch variant {
	case PKSL:
		bufOut[2] = 'S'
		bufOut[3] = 'L'
	case PKS3:
		bufOut[2] = 'S'
		bufOut[3] = '3'
	case PKSP:
		bufOut[2] = 'S'
		bufOut[3] = 'P'
	case PKVL:
		bufOut[2] = 'V'
		bufOut[3] = 'L'
	case PKVP:
		bufOut[2] = 'V'
		bufOut[3] = 'P'
	}
	pos = 4

	// PKSL includes 17 bytes of palette
	if variant == PKSL {
		if pos+17 > len(bufOut) {
			return 0, ErrOutputTooSmall
		}
		if palette != nil && len(palette) >= 17 {
			copy(bufOut[pos:pos+17], palette[:17])
		}
		pos += 17
	}

	// Compress using Standard method
	n, err := p.lzw.PackStd(bufIn, lengthIn, bufOut, pos)
	if err != nil {
		return 0, err
	}

	return n, nil
}
