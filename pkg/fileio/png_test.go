package fileio

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// pngChunks walks a PNG and returns the chunk types in file order, the pHYs pair
// and unit it carries (if any) and the index of that chunk. Every CRC is verified
// on the way: a chunk has to be well formed, not merely present.
func pngChunks(t *testing.T, data []byte) (types []string, ppux, ppuy uint32, unit byte, physIndex int) {
	t.Helper()
	physIndex = -1

	if !bytes.HasPrefix(data, pngSignature) {
		t.Fatalf("not a PNG: the signature is missing")
	}
	for off := pngSignatureLen; off+pngChunkHeader <= len(data); {
		length := int(binary.BigEndian.Uint32(data[off : off+4]))
		if off+pngChunkHeader+length > len(data) {
			t.Fatalf("chunk at offset %d runs past the end of the file", off)
		}
		kind := string(data[off+4 : off+8])
		want := binary.BigEndian.Uint32(data[off+8+length : off+12+length])
		if got := crc32.ChecksumIEEE(data[off+4 : off+8+length]); got != want {
			t.Fatalf("chunk %s has CRC %08x, want %08x", kind, got, want)
		}
		types = append(types, kind)

		if kind == "pHYs" {
			if length != pngPHYsDataLen {
				t.Fatalf("pHYs carries %d bytes, want %d", length, pngPHYsDataLen)
			}
			body := data[off+8 : off+8+length]
			ppux = binary.BigEndian.Uint32(body[0:4])
			ppuy = binary.BigEndian.Uint32(body[4:8])
			unit = body[8]
			physIndex = len(types) - 1
		}
		off += pngChunkHeader + length
	}
	return types, ppux, ppuy, unit, physIndex
}

// testImage builds a small picture whose pixels are all distinct, so a decode that
// loses or reorders anything is caught.
func testImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	for y := 0; y < 3; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 60), G: uint8(y * 80), B: 7, A: 255})
		}
	}
	return img
}

// TestWritePNGWithDensity checks the primitive: the chunk lands immediately after
// IHDR, it declares the pair as a real density (unit 1), the rest of the file is
// byte for byte the encoder's own output, and the picture still decodes as it was.
func TestWritePNGWithDensity(t *testing.T) {
	img := testImage()

	var plain bytes.Buffer
	if err := png.Encode(&plain, img); err != nil {
		t.Fatalf("png.Encode failed: %v", err)
	}

	var with bytes.Buffer
	if err := WritePNGWithDensity(&with, img, 708, 1416); err != nil {
		t.Fatalf("WritePNGWithDensity failed: %v", err)
	}

	types, ppux, ppuy, unit, index := pngChunks(t, with.Bytes())
	if index != 1 {
		t.Errorf("pHYs is chunk %d, want 1 (right after IHDR): %v", index, types)
	}
	if ppux != 708 || ppuy != 1416 {
		t.Errorf("pHYs declares %d x %d pixels per metre, want 708 x 1416", ppux, ppuy)
	}
	if unit != pngPHYsUnitMetre {
		t.Errorf("pHYs unit = %d, want %d (pixels per metre)", unit, pngPHYsUnitMetre)
	}

	// The chunk costs exactly its own 21 bytes and changes nothing else: dropping
	// those bytes from the spliced file has to give the plain encode back, byte for
	// byte. IHDR is 25 bytes on the wire, so the chunk sits at offset 33.
	if got, want := len(with.Bytes()), plain.Len()+pngPHYsChunkLen; got != want {
		t.Errorf("PNG is %d bytes, want %d (the encode's %d plus the chunk)",
			got, want, plain.Len())
	}
	const ihdrEnd = pngSignatureLen + pngChunkHeader + 13
	stripped := append([]byte{}, with.Bytes()[:ihdrEnd]...)
	stripped = append(stripped, with.Bytes()[ihdrEnd+pngPHYsChunkLen:]...)
	if !bytes.Equal(stripped, plain.Bytes()) {
		t.Error("the encoded picture is not byte for byte png.Encode's output")
	}

	back, err := png.Decode(bytes.NewReader(with.Bytes()))
	if err != nil {
		t.Fatalf("the file no longer decodes: %v", err)
	}
	if back.Bounds() != img.Bounds() {
		t.Errorf("decoded size %v, want %v", back.Bounds(), img.Bounds())
	}
	if got := back.At(3, 2); got != img.At(3, 2) {
		t.Errorf("pixel (3,2) = %v, want %v", got, img.At(3, 2))
	}
}

// TestWritePNGWithDensityReplacesExistingChunk re-splices a file that already
// carries a pHYs chunk: the new pair must replace the old one, and the result must
// be identical to a first-time splice, so a re-encode cannot leave two chunks
// behind or leave the file depending on its history.
func TestWritePNGWithDensityReplacesExistingChunk(t *testing.T) {
	img := testImage()

	var first bytes.Buffer
	if err := WritePNGWithDensity(&first, img, 708, 1416); err != nil {
		t.Fatalf("first WritePNGWithDensity failed: %v", err)
	}
	second, err := splicePHYs(first.Bytes(), 2832, 2832)
	if err != nil {
		t.Fatalf("second splice failed: %v", err)
	}

	types, ppux, ppuy, unit, index := pngChunks(t, second)
	if count := len(types); count != 4 {
		t.Errorf("chunks after re-splicing: %v, want 4 (IHDR, pHYs, IDAT, IEND)", types)
	}
	if index != 1 || ppux != 2832 || ppuy != 2832 || unit != pngPHYsUnitMetre {
		t.Errorf("re-spliced pHYs = chunk %d, %d x %d unit %d, want chunk 1, 2832 x 2832 unit %d",
			index, ppux, ppuy, unit, pngPHYsUnitMetre)
	}

	var fresh bytes.Buffer
	if err := WritePNGWithDensity(&fresh, img, 2832, 2832); err != nil {
		t.Fatalf("fresh WritePNGWithDensity failed: %v", err)
	}
	if !bytes.Equal(second, fresh.Bytes()) {
		t.Error("re-splicing a file that already had a pHYs chunk differs from a fresh encode")
	}
}

// TestSplicePHYsRejectsBrokenInput checks that a payload that is not a PNG, or is
// truncated inside a chunk, is refused instead of being copied out as if it were.
func TestSplicePHYsRejectsBrokenInput(t *testing.T) {
	var plain bytes.Buffer
	if err := png.Encode(&plain, testImage()); err != nil {
		t.Fatalf("png.Encode failed: %v", err)
	}

	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"not a PNG", []byte("this is not a PNG at all")},
		{"signature only", pngSignature},
		{"truncated inside IHDR", plain.Bytes()[:20]},
		{"truncated inside IDAT", plain.Bytes()[:len(plain.Bytes())-9]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := splicePHYs(tc.data, 1416, 1416); err == nil {
				t.Error("splicePHYs accepted a broken payload")
			}
		})
	}
}
