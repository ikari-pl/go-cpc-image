// Package compress provides PKS family packed format support.
// PKS variants:
//   - PKSL: 320x200 Standard
//   - PKS3: 320x200 Mode 3
//   - PKSP: 320x200 Plus
//   - PKVL: Overscan Standard
//   - PKVP: Overscan Plus
//   - PKUL: Underscan Standard
//   - PKU3: Underscan Mode 3
//   - PKUP: Underscan Plus
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
	// PKUL is Underscan Standard
	PKUL
	// PKU3 is Underscan Mode 3
	PKU3
	// PKUP is Underscan Plus
	PKUP
)

// HasPalette reports whether the variant stores the 17-byte ModePal field
// (mode byte + 16 ink values) between its 4-byte signature and the compressed
// data. Only the "L"-flavoured variants do: their payload is column-major pixel
// data, so the palette has nowhere else to live. Every other variant packs the
// screen dump as it is, palette block included - a CPC Plus palette is 33 bytes
// and travels inside that dump, at &17D0 as in the raw saves.
func (v PKSVariant) HasPalette() bool {
	switch v {
	case PKSL, PKUL:
		return true
	}
	return false
}

// PKSHeader holds parsed PKS file header information
type PKSHeader struct {
	Variant  PKSVariant
	CpcPlus  bool
	Overscan bool
	// Palette holds the 17 bytes read from the header field at offset 4; only the
	// "L" variants have that field, so it stays empty for every other variant (a
	// CPC Plus palette lives inside the packed dump instead).
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
	case b2 == 'U' && b3 == 'L':
		h.Variant = PKUL
	case b2 == 'U' && b3 == '3':
		h.Variant = PKU3
	case b2 == 'U' && b3 == 'P':
		h.Variant = PKUP
	case b2 == 'O':
		// PKO* variants (plus overscan)
		h.Variant = PKVP
		h.CpcPlus = true
		h.Overscan = true
	default:
		return nil, ErrUnknownVariant
	}

	// The "L" variants carry 17 bytes of palette at offset 4 and their data
	// starts at offset 21; every other variant starts right after the signature.
	if h.Variant.HasPalette() {
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
// palette is only used for the variants with a palette field (HasPalette), where
// it provides the 17 bytes of ink values.
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
	case PKUL:
		bufOut[2] = 'U'
		bufOut[3] = 'L'
	case PKU3:
		bufOut[2] = 'U'
		bufOut[3] = '3'
	case PKUP:
		bufOut[2] = 'U'
		bufOut[3] = 'P'
	}
	pos = 4

	// The "L" variants include 17 bytes of palette
	if variant.HasPalette() {
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
