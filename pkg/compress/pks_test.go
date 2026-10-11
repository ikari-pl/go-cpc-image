// Package compress provides PKS family packed format support.
package compress

import (
	"bytes"
	"testing"
)

// TestParsePKSHeaderAllVariants verifies every signature (standard, underscan
// and overscan) is parsed with the right flags, palette field and data offset.
func TestParsePKSHeaderAllVariants(t *testing.T) {
	palette := make([]byte, 17)
	for i := range palette {
		palette[i] = byte(i + 1)
	}

	cases := []struct {
		name         string
		sig          string
		withPal      bool
		wantVariant  PKSVariant
		wantPlus     bool
		wantOverscan bool
	}{
		{"PKSL standard", "PKSL", true, PKSL, false, false},
		{"PKS3 mode 3", "PKS3", false, PKS3, false, false},
		{"PKSP Plus", "PKSP", false, PKSP, true, false},
		{"PKVL overscan", "PKVL", false, PKVL, false, true},
		{"PKVP overscan Plus", "PKVP", false, PKVP, true, true},
		{"PKUL underscan", "PKUL", true, PKUL, false, false},
		{"PKU3 underscan mode 3", "PKU3", false, PKU3, false, false},
		{"PKUP underscan Plus", "PKUP", false, PKUP, true, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := []byte(tc.sig)
			if tc.withPal {
				buf = append(buf, palette...)
			}
			buf = append(buf, 0xAA, 0xBB)

			h, err := ParsePKSHeader(buf)
			if err != nil {
				t.Fatalf("ParsePKSHeader(%s) failed: %v", tc.sig, err)
			}
			if h.Variant != tc.wantVariant {
				t.Errorf("variant = %v, want %v", h.Variant, tc.wantVariant)
			}
			if h.CpcPlus != tc.wantPlus {
				t.Errorf("CpcPlus = %v, want %v", h.CpcPlus, tc.wantPlus)
			}
			if h.Overscan != tc.wantOverscan {
				t.Errorf("Overscan = %v, want %v", h.Overscan, tc.wantOverscan)
			}
			if h.Variant.HasPalette() != tc.withPal {
				t.Errorf("HasPalette() = %v, want %v", h.Variant.HasPalette(), tc.withPal)
			}
			wantOffset := 4
			if tc.withPal {
				wantOffset = 21
			}
			if h.DataOffset != wantOffset {
				t.Errorf("DataOffset = %d, want %d", h.DataOffset, wantOffset)
			}
			if tc.withPal {
				if !bytes.Equal(h.Palette[:], palette) {
					t.Errorf("palette = % x, want % x", h.Palette[:], palette)
				}
			} else if h.Palette != ([17]byte{}) {
				t.Errorf("palette = % x, want empty (no palette field)", h.Palette[:])
			}
		})
	}
}

// TestPackPKSVariantRoundTrip packs and depacks each variant: the signature must
// survive, the palette field must be written only for the variants that define
// it, and the payload must come back byte-for-byte.
func TestPackPKSVariantRoundTrip(t *testing.T) {
	payload := make([]byte, 12288) // one underscan pixel payload
	for i := range payload {
		payload[i] = byte(i*11 + 5)
	}
	palette := make([]byte, 17)
	for i := range palette {
		palette[i] = byte(i)
	}

	for _, tc := range []struct {
		name       string
		variant    PKSVariant
		wantOffset int
	}{
		{"PKSL", PKSL, 21}, {"PKS3", PKS3, 4}, {"PKSP", PKSP, 4},
		{"PKVL", PKVL, 4}, {"PKVP", PKVP, 4},
		{"PKUL", PKUL, 21}, {"PKU3", PKU3, 4}, {"PKUP", PKUP, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buf := make([]byte, len(payload)*2+1024)
			n, err := NewPKS().PackPKS(payload, len(payload), buf, tc.variant, palette)
			if err != nil {
				t.Fatalf("PackPKS(%s) failed: %v", tc.name, err)
			}
			packed := buf[:n]

			h, err := ParsePKSHeader(packed)
			if err != nil {
				t.Fatalf("ParsePKSHeader failed: %v", err)
			}
			if h.Variant != tc.variant {
				t.Errorf("packed variant = %v, want %v", h.Variant, tc.variant)
			}
			if h.DataOffset != tc.wantOffset {
				t.Errorf("packed data offset = %d, want %d", h.DataOffset, tc.wantOffset)
			}
			if tc.variant.HasPalette() {
				if !bytes.Equal(h.Palette[:], palette) {
					t.Errorf("packed palette = % x, want % x", h.Palette[:], palette)
				}
			} else if h.Palette != ([17]byte{}) {
				t.Errorf("packed palette = % x, want empty (no palette field)", h.Palette[:])
			}

			out := make([]byte, len(payload)+1024)
			dn, dh, err := NewPKS().DepackPKS(packed, out)
			if err != nil {
				t.Fatalf("DepackPKS failed: %v", err)
			}
			if dh.Variant != tc.variant {
				t.Errorf("depacked variant = %v, want %v", dh.Variant, tc.variant)
			}
			if dn != len(payload) || !bytes.Equal(out[:dn], payload) {
				t.Errorf("payload mismatch: got %d bytes, want %d", dn, len(payload))
			}
		})
	}
}

// TestPackPKUPKeepsPlusPaletteInPayload pins the PKUP layout: no 17-byte palette
// field (the packed data starts at offset 4) and the 33-byte CPC Plus block
// travelling inside the packed dump, at &17D0, exactly as in a raw Plus screen.
// A Plus palette does not fit in the 17-byte field that only the "L" variants
// define.
func TestPackPKUPKeepsPlusPaletteInPayload(t *testing.T) {
	const (
		modePalOffset = 0x17D0
		underscanSize = 15872 // 64x192 underscan dump
	)

	payload := make([]byte, underscanSize)
	for i := range payload {
		payload[i] = byte(i*7 + 3)
	}
	// Plus block: mode byte &8C|1, then 16 &xRGB ink pairs.
	payload[modePalOffset] = 0x8D
	for i := 0; i < 16; i++ {
		payload[modePalOffset+1+2*i] = 0x40
		payload[modePalOffset+2+2*i] = byte(i)
	}

	buf := make([]byte, len(payload)*2+1024)
	n, err := NewPKS().PackPKS(payload, len(payload), buf, PKUP, nil)
	if err != nil {
		t.Fatalf("PackPKS(PKUP) failed: %v", err)
	}

	h, err := ParsePKSHeader(buf[:n])
	if err != nil {
		t.Fatalf("ParsePKSHeader failed: %v", err)
	}
	if h.DataOffset != 4 {
		t.Errorf("PKUP data offset = %d, want 4 (no palette field)", h.DataOffset)
	}
	if h.Palette != ([17]byte{}) {
		t.Errorf("PKUP palette field = % x, want empty", h.Palette[:])
	}

	out := make([]byte, len(payload)+1024)
	dn, _, err := NewPKS().DepackPKS(buf[:n], out)
	if err != nil {
		t.Fatalf("DepackPKS failed: %v", err)
	}
	if dn != len(payload) || !bytes.Equal(out[:dn], payload) {
		t.Fatalf("payload mismatch after round-trip: %d bytes, want %d", dn, len(payload))
	}
	if out[modePalOffset] != 0x8D {
		t.Errorf("Plus mode byte at &%04X = 0x%02X, want 0x8D (the palette travels inside the payload)",
			modePalOffset, out[modePalOffset])
	}
}

// TestCompressorUnderscanMethods verifies the compressor method constants map to
// the underscan signatures for both packing and depacking.
func TestCompressorUnderscanMethods(t *testing.T) {
	payload := make([]byte, 4096)
	for i := range payload {
		payload[i] = byte(i * 3)
	}

	for _, tc := range []struct {
		name   string
		method PackMethod
		want   string
	}{
		{"MethodPKUL", MethodPKUL, "PKUL"},
		{"MethodPKU3", MethodPKU3, "PKU3"},
		{"MethodPKUP", MethodPKUP, "PKUP"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buf := make([]byte, len(payload)*2+1024)
			n, err := NewCompressor().Pack(payload, len(payload), buf, 0, tc.method)
			if err != nil {
				t.Fatalf("Pack failed: %v", err)
			}
			if got := string(buf[:4]); got != tc.want {
				t.Errorf("signature = %q, want %q", got, tc.want)
			}

			out := make([]byte, len(payload)+1024)
			dn, err := NewCompressor().Depack(buf[:n], 0, out, tc.method)
			if err != nil {
				t.Fatalf("Depack failed: %v", err)
			}
			if dn != len(payload) || !bytes.Equal(out[:dn], payload) {
				t.Errorf("payload mismatch after Depack: %d bytes", dn)
			}
		})
	}
}
