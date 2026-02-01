package cpc

import (
	"testing"
)

func TestSpriteEncodeDecodeRoundtrip(t *testing.T) {
	s := NewSprite()
	// Fill with a known pattern
	for y := 0; y < SpriteSize; y++ {
		for x := 0; x < SpriteSize; x++ {
			s.Data[y][x] = byte((x + y) & 0x0F)
		}
	}

	encoded := s.Encode()
	if len(encoded) != SpriteEncodedSize {
		t.Fatalf("encoded size = %d, want %d", len(encoded), SpriteEncodedSize)
	}

	var s2 Sprite
	if err := s2.Decode(encoded); err != nil {
		t.Fatalf("Decode error: %v", err)
	}

	for y := 0; y < SpriteSize; y++ {
		for x := 0; x < SpriteSize; x++ {
			if s.Data[y][x] != s2.Data[y][x] {
				t.Errorf("pixel mismatch at (%d,%d): got %d, want %d", x, y, s2.Data[y][x], s.Data[y][x])
			}
		}
	}
}

func TestSpriteEncodeFormat(t *testing.T) {
	s := NewSprite()
	// Set first row: pixels 0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15
	for x := 0; x < SpriteSize; x++ {
		s.Data[0][x] = byte(x)
	}

	encoded := s.Encode()

	// First row bytes should be: 0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF
	expected := []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF}
	for i, want := range expected {
		if encoded[i] != want {
			t.Errorf("byte %d = 0x%02X, want 0x%02X", i, encoded[i], want)
		}
	}
}

func TestSpriteBankEncodeDecodeRoundtrip(t *testing.T) {
	sb := NewSpriteBank()
	for i := 0; i < MaxSprites; i++ {
		for y := 0; y < SpriteSize; y++ {
			for x := 0; x < SpriteSize; x++ {
				sb.Sprites[i].Data[y][x] = byte((i + x + y) & 0x0F)
			}
		}
	}

	encoded := sb.EncodeBank()
	if len(encoded) != MaxSprites*SpriteEncodedSize {
		t.Fatalf("bank size = %d, want %d", len(encoded), MaxSprites*SpriteEncodedSize)
	}

	sb2 := NewSpriteBank()
	if err := sb2.DecodeBank(encoded); err != nil {
		t.Fatalf("DecodeBank error: %v", err)
	}

	for i := 0; i < MaxSprites; i++ {
		for y := 0; y < SpriteSize; y++ {
			for x := 0; x < SpriteSize; x++ {
				if sb.Sprites[i].Data[y][x] != sb2.Sprites[i].Data[y][x] {
					t.Errorf("sprite %d pixel (%d,%d): got %d, want %d",
						i, x, y, sb2.Sprites[i].Data[y][x], sb.Sprites[i].Data[y][x])
				}
			}
		}
	}
}

func TestSpritePaletteEncodeDecodeRoundtrip(t *testing.T) {
	sb := NewSpriteBank()
	sb.Palette = [16]int{
		0x000, 0xF00, 0x0F0, 0x00F,
		0xFF0, 0xF0F, 0x0FF, 0xFFF,
		0x100, 0x010, 0x001, 0x110,
		0x101, 0x011, 0x111, 0x222,
	}

	encoded := sb.EncodePalette()
	if len(encoded) != 32 {
		t.Fatalf("palette size = %d, want 32", len(encoded))
	}

	sb2 := NewSpriteBank()
	if err := sb2.DecodePalette(encoded); err != nil {
		t.Fatalf("DecodePalette error: %v", err)
	}

	for i := 0; i < MaxSpritePalette; i++ {
		if sb.Palette[i] != sb2.Palette[i] {
			t.Errorf("palette[%d] = 0x%03X, want 0x%03X", i, sb2.Palette[i], sb.Palette[i])
		}
	}
}

func TestSpriteDecodeShortData(t *testing.T) {
	s := NewSprite()
	err := s.Decode(make([]byte, 10))
	if err == nil {
		t.Error("expected error for short data")
	}
}

func TestSpriteToRGBA(t *testing.T) {
	s := NewSprite()
	s.Data[0][0] = 1
	palette := [16]int{0x000, 0xF00} // pen 1 = green=0xF, blue=0, red=0

	img := s.ToRGBA(palette)
	if img.Bounds().Dx() != 16 || img.Bounds().Dy() != 16 {
		t.Fatalf("image size = %dx%d, want 16x16", img.Bounds().Dx(), img.Bounds().Dy())
	}

	// Pen 0 should be transparent
	r, g, b, a := img.At(1, 0).RGBA()
	if a != 0 {
		t.Errorf("pen 0 alpha = %d, want 0 (r=%d g=%d b=%d)", a, r, g, b)
	}

	// Pen 1 (0xF00 = G=0xF, B=0, R=0) should be green=255, red=0, blue=0
	_, g1, _, a1 := img.At(0, 0).RGBA()
	if a1 == 0 {
		t.Error("pen 1 should not be transparent")
	}
	if g1 == 0 {
		t.Error("pen 1 green channel should be non-zero")
	}
}

func TestSpriteFlipH(t *testing.T) {
	s := NewSprite()
	s.Data[0][0] = 5
	s.Data[0][15] = 10
	s.FlipH()
	if s.Data[0][0] != 10 || s.Data[0][15] != 5 {
		t.Errorf("FlipH: got [0][0]=%d [0][15]=%d, want 10,5", s.Data[0][0], s.Data[0][15])
	}
}

func TestSpriteFlipV(t *testing.T) {
	s := NewSprite()
	s.Data[0][0] = 5
	s.Data[15][0] = 10
	s.FlipV()
	if s.Data[0][0] != 10 || s.Data[15][0] != 5 {
		t.Errorf("FlipV: got [0][0]=%d [15][0]=%d, want 10,5", s.Data[0][0], s.Data[15][0])
	}
}

func TestSpriteRotateCW(t *testing.T) {
	s := NewSprite()
	s.Data[0][0] = 1  // top-left -> top-right
	s.Data[0][15] = 2 // top-right -> bottom-right
	s.RotateCW()
	if s.Data[0][15] != 1 {
		t.Errorf("after RotateCW: top-left should move to top-right, got %d", s.Data[0][15])
	}
	if s.Data[15][15] != 2 {
		t.Errorf("after RotateCW: top-right should move to bottom-right, got %d", s.Data[15][15])
	}
}
