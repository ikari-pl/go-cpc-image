package fileio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/png"
	"io"
)

// PNG layout: an 8-byte signature, then chunks of a 4-byte length, a 4-byte type,
// the data and a 4-byte CRC over the type and the data, all big-endian.
const (
	pngSignatureLen = 8
	pngChunkHeader  = 12
	pngPHYsDataLen  = 9
	pngPHYsChunkLen = pngChunkHeader + pngPHYsDataLen
)

// pngPHYsUnitMetre is the pHYs unit saying the two values are a real density in
// pixels per metre, rather than a bare pixel aspect ratio (unit 0).
const pngPHYsUnitMetre = 1

var (
	pngSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	pngChunkIHDR = []byte("IHDR")
	pngChunkPHYs = []byte("pHYs")
)

// WritePNGWithDensity encodes img as a PNG and splices an ancillary pHYs chunk
// right after IHDR declaring the density of the grid's pixels — ppux x ppuy pixels
// per metre. That is the only way to write it: image/png's Encoder exposes no
// ancillary chunks at all, and a pHYs chunk has to precede the image data.
//
// A pHYs chunk the encoded file already carried is dropped, so a file can never end
// up with two, and the picture comes out exactly as the encoder wrote it.
func WritePNGWithDensity(w io.Writer, img image.Image, ppux, ppuy uint32) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	out, err := splicePHYs(buf.Bytes(), ppux, ppuy)
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}

// splicePHYs returns encoded with a pHYs chunk carrying the given density spliced
// right after IHDR, and with every pHYs it already held removed. The chunk walk is
// strict: a payload that stops inside a chunk header, or that carries a chunk longer
// than the file, is refused instead of being copied out truncated.
func splicePHYs(encoded []byte, ppux, ppuy uint32) ([]byte, error) {
	if !bytes.HasPrefix(encoded, pngSignature) {
		return nil, fmt.Errorf("not a PNG: the signature is missing")
	}

	out := make([]byte, 0, len(encoded)+pngPHYsChunkLen)
	out = append(out, pngSignature...)

	spliced := false
	for off := pngSignatureLen; off < len(encoded); {
		if off+pngChunkHeader > len(encoded) {
			return nil, fmt.Errorf("PNG ends inside the chunk header at offset %d", off)
		}
		length := int(binary.BigEndian.Uint32(encoded[off : off+4]))
		if off+pngChunkHeader+length > len(encoded) {
			return nil, fmt.Errorf("PNG chunk at offset %d runs past the end of the file", off)
		}
		kind := encoded[off+4 : off+8]
		chunk := encoded[off : off+pngChunkHeader+length]
		off += pngChunkHeader + length

		if bytes.Equal(kind, pngChunkPHYs) {
			continue
		}
		out = append(out, chunk...)
		if !spliced && bytes.Equal(kind, pngChunkIHDR) {
			out = append(out, pHYsChunk(ppux, ppuy)...)
			spliced = true
		}
	}
	if !spliced {
		return nil, fmt.Errorf("no IHDR chunk in the encoded PNG")
	}
	return out, nil
}

// pHYsChunk builds a pHYs chunk: its length, its type, the nine data bytes — the
// density in x, the density in y and the unit — and its CRC.
func pHYsChunk(ppux, ppuy uint32) []byte {
	chunk := make([]byte, 0, pngPHYsChunkLen)
	chunk = binary.BigEndian.AppendUint32(chunk, pngPHYsDataLen)
	chunk = append(chunk, pngChunkPHYs...)
	chunk = binary.BigEndian.AppendUint32(chunk, ppux)
	chunk = binary.BigEndian.AppendUint32(chunk, ppuy)
	chunk = append(chunk, pngPHYsUnitMetre)
	chunk = binary.BigEndian.AppendUint32(chunk, crc32.ChecksumIEEE(chunk[4:]))
	return chunk
}
