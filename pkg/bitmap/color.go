// Package bitmap provides color and bitmap abstractions for CPC image handling.
package bitmap

// RgbColor represents a color with red, green (v=vert), blue components.
// This is equivalent to the C# RgbColor class and uses the CPC naming convention
// where 'v' stands for "vert" (green in French).
type RgbColor struct {
	R, V, B uint8
}

// NewRgbColor creates a new RgbColor from RGB components.
func NewRgbColor(r, v, b uint8) RgbColor {
	return RgbColor{R: r, V: v, B: b}
}

// NewRgbColorFromInt creates a new RgbColor from a packed ARGB integer.
// The alpha channel is ignored.
func NewRgbColorFromInt(value int) RgbColor {
	return RgbColor{
		R: uint8(value >> 16),
		V: uint8(value >> 8),
		B: uint8(value),
	}
}

// GetColor returns the color as a packed RGB integer (no alpha).
func (c RgbColor) GetColor() int {
	return int(c.B) + (int(c.V) << 8) + (int(c.R) << 16)
}

// GetColorArgb returns the color as a packed ARGB integer with alpha=255.
func (c RgbColor) GetColorArgb() int {
	return int(c.B) + (int(c.V) << 8) + (int(c.R) << 16) + (255 << 24)
}

// SetColorArgb sets the color from a packed ARGB integer.
// The alpha channel is ignored.
func (c *RgbColor) SetColorArgb(value int) {
	c.R = uint8(value >> 16)
	c.V = uint8(value >> 8)
	c.B = uint8(value)
}