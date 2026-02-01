// Package cpc provides CPC+ hardware sprite support.
package cpc

import (
	"fmt"
	"image"
	"image/color"

	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
)

// SpriteSize is the dimension of a CPC+ hardware sprite (16x16 pixels).
const SpriteSize = 16

// SpriteBytesPerRow is the number of bytes per sprite row in ASIC format.
// Each byte holds 2 pixels (4 bits each), so 16 pixels = 8 bytes.
const SpriteBytesPerRow = 8

// SpriteEncodedSize is the total encoded size of one sprite in ASIC RAM.
// 16 rows * 8 bytes per row = 128 bytes.
const SpriteEncodedSize = SpriteSize * SpriteBytesPerRow

// MaxSprites is the number of hardware sprites available on the CPC+.
const MaxSprites = 16

// MaxSpritePalette is the number of colors in the sprite palette.
const MaxSpritePalette = 16

// Sprite represents a single CPC+ hardware sprite.
type Sprite struct {
	// Data holds 16x16 pixel values, each in range 0-15 (pen index).
	// Data[y][x] where y=row, x=column.
	Data [SpriteSize][SpriteSize]byte

	// X position on screen (0-767).
	X int
	// Y position on screen (0-255).
	Y int

	// MagX is horizontal magnification (1, 2, or 4).
	MagX int
	// MagY is vertical magnification (1, 2, or 4).
	MagY int

	// Visible controls whether the sprite is displayed.
	Visible bool
}

// SpriteBank holds all 16 hardware sprites and the shared sprite palette.
type SpriteBank struct {
	Sprites [MaxSprites]Sprite
	// Palette holds 16 CPC+ 12-bit colors in 0x0GBR format
	// (G=green bits 11-8, B=blue bits 7-4, R=red bits 3-0).
	Palette [MaxSpritePalette]int
}

// NewSprite creates a new sprite with default magnification of 1x1.
func NewSprite() Sprite {
	return Sprite{
		MagX:    1,
		MagY:    1,
		Visible: true,
	}
}

// NewSpriteBank creates a new sprite bank with default sprites.
func NewSpriteBank() *SpriteBank {
	sb := &SpriteBank{}
	for i := range sb.Sprites {
		sb.Sprites[i] = NewSprite()
	}
	return sb
}

// Encode converts the sprite pixel data to CPC+ ASIC format.
// Returns 128 bytes: 16 rows of 8 bytes each, with 2 pixels packed per byte.
// In each byte, the high nibble is the left pixel and the low nibble is the right pixel.
func (s *Sprite) Encode() []byte {
	buf := make([]byte, SpriteEncodedSize)
	for y := 0; y < SpriteSize; y++ {
		for x := 0; x < SpriteSize; x += 2 {
			byteIdx := y*SpriteBytesPerRow + x/2
			hi := s.Data[y][x] & 0x0F
			lo := s.Data[y][x+1] & 0x0F
			buf[byteIdx] = (hi << 4) | lo
		}
	}
	return buf
}

// Decode loads sprite pixel data from CPC+ ASIC format (128 bytes).
// Each byte contains two 4-bit pixel values (high nibble = left, low nibble = right).
func (s *Sprite) Decode(data []byte) error {
	if len(data) < SpriteEncodedSize {
		return fmt.Errorf("sprite data too short: got %d bytes, need %d", len(data), SpriteEncodedSize)
	}
	for y := 0; y < SpriteSize; y++ {
		for x := 0; x < SpriteSize; x += 2 {
			byteIdx := y*SpriteBytesPerRow + x/2
			s.Data[y][x] = (data[byteIdx] >> 4) & 0x0F
			s.Data[y][x+1] = data[byteIdx] & 0x0F
		}
	}
	return nil
}

// ToRGBA renders the sprite as an *image.RGBA using the given CPC+ 12-bit palette.
// Pen 0 is rendered as transparent.
func (s *Sprite) ToRGBA(palette [MaxSpritePalette]int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, SpriteSize, SpriteSize))
	for y := 0; y < SpriteSize; y++ {
		for x := 0; x < SpriteSize; x++ {
			pen := s.Data[y][x] & 0x0F
			if pen == 0 {
				// Transparent
				img.Set(x, y, color.RGBA{0, 0, 0, 0})
			} else {
				c := GetColorFromCpcPlus(palette[pen])
				img.Set(x, y, color.RGBA{R: c.R, G: c.V, B: c.B, A: 255})
			}
		}
	}
	return img
}

// ToRGBAMagnified renders the sprite with magnification applied.
// The output size is (16*MagX) x (16*MagY).
func (s *Sprite) ToRGBAMagnified(palette [MaxSpritePalette]int) *image.RGBA {
	magX := s.MagX
	magY := s.MagY
	if magX < 1 {
		magX = 1
	}
	if magY < 1 {
		magY = 1
	}
	w := SpriteSize * magX
	h := SpriteSize * magY
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < SpriteSize; y++ {
		for x := 0; x < SpriteSize; x++ {
			pen := s.Data[y][x] & 0x0F
			var c color.RGBA
			if pen == 0 {
				c = color.RGBA{0, 0, 0, 0}
			} else {
				rgb := GetColorFromCpcPlus(palette[pen])
				c = color.RGBA{R: rgb.R, G: rgb.V, B: rgb.B, A: 255}
			}
			for my := 0; my < magY; my++ {
				for mx := 0; mx < magX; mx++ {
					img.Set(x*magX+mx, y*magY+my, c)
				}
			}
		}
	}
	return img
}

// Import converts a 16x16 region from a DirectBitmap into sprite pixel data.
// srcX, srcY specify the top-left corner of the region in the source bitmap.
// The palette is used to find the closest pen for each pixel.
func (s *Sprite) Import(img *bitmap.DirectBitmap, srcX, srcY int, palette [MaxSpritePalette]int) {
	for y := 0; y < SpriteSize; y++ {
		for x := 0; x < SpriteSize; x++ {
			px := srcX + x
			py := srcY + y
			if px < img.Width() && py < img.Height() {
				pen := GetPenColor(img, px, py, palette, true)
				s.Data[y][x] = byte(pen & 0x0F)
			} else {
				s.Data[y][x] = 0
			}
		}
	}
}

// Clear sets all pixels to pen 0 (transparent).
func (s *Sprite) Clear() {
	s.Data = [SpriteSize][SpriteSize]byte{}
}

// FlipH flips the sprite horizontally (left-right).
func (s *Sprite) FlipH() {
	for y := 0; y < SpriteSize; y++ {
		for x := 0; x < SpriteSize/2; x++ {
			s.Data[y][x], s.Data[y][SpriteSize-1-x] = s.Data[y][SpriteSize-1-x], s.Data[y][x]
		}
	}
}

// FlipV flips the sprite vertically (top-bottom).
func (s *Sprite) FlipV() {
	for y := 0; y < SpriteSize/2; y++ {
		s.Data[y], s.Data[SpriteSize-1-y] = s.Data[SpriteSize-1-y], s.Data[y]
	}
}

// RotateCW rotates the sprite 90 degrees clockwise.
func (s *Sprite) RotateCW() {
	var tmp [SpriteSize][SpriteSize]byte
	for y := 0; y < SpriteSize; y++ {
		for x := 0; x < SpriteSize; x++ {
			tmp[x][SpriteSize-1-y] = s.Data[y][x]
		}
	}
	s.Data = tmp
}

// EncodeBank encodes all sprites in the bank to ASIC format.
// Returns 16 * 128 = 2048 bytes.
func (sb *SpriteBank) EncodeBank() []byte {
	buf := make([]byte, 0, MaxSprites*SpriteEncodedSize)
	for i := 0; i < MaxSprites; i++ {
		buf = append(buf, sb.Sprites[i].Encode()...)
	}
	return buf
}

// DecodeBank loads all 16 sprites from ASIC format data.
func (sb *SpriteBank) DecodeBank(data []byte) error {
	if len(data) < MaxSprites*SpriteEncodedSize {
		return fmt.Errorf("sprite bank data too short: got %d bytes, need %d",
			len(data), MaxSprites*SpriteEncodedSize)
	}
	for i := 0; i < MaxSprites; i++ {
		offset := i * SpriteEncodedSize
		if err := sb.Sprites[i].Decode(data[offset : offset+SpriteEncodedSize]); err != nil {
			return fmt.Errorf("sprite %d: %w", i, err)
		}
	}
	return nil
}

// EncodePalette encodes the sprite palette as 16 little-endian 16-bit words (32 bytes).
// Each color is a CPC+ 12-bit value in 0x0GBR format.
func (sb *SpriteBank) EncodePalette() []byte {
	buf := make([]byte, MaxSpritePalette*2)
	for i := 0; i < MaxSpritePalette; i++ {
		c := sb.Palette[i] & 0x0FFF
		buf[i*2] = byte(c & 0xFF)
		buf[i*2+1] = byte((c >> 8) & 0xFF)
	}
	return buf
}

// DecodePalette loads the sprite palette from 32 bytes of little-endian 16-bit words.
func (sb *SpriteBank) DecodePalette(data []byte) error {
	if len(data) < MaxSpritePalette*2 {
		return fmt.Errorf("palette data too short: got %d bytes, need %d", len(data), MaxSpritePalette*2)
	}
	for i := 0; i < MaxSpritePalette; i++ {
		sb.Palette[i] = int(data[i*2]) | (int(data[i*2+1]) << 8)
	}
	return nil
}
