package convert

import (
	"math"
	"testing"

	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
)

// TestComputeMSE_IdenticalImages verifies that two identical images produce zero error.
func TestComputeMSE_IdenticalImages(t *testing.T) {
	a := bitmap.NewDirectBitmap(10, 10)
	b := bitmap.NewDirectBitmap(10, 10)
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			c := bitmap.NewRgbColor(uint8(x*25), uint8(y*25), 128)
			a.SetPixelColor(x, y, c)
			b.SetPixelColor(x, y, c)
		}
	}
	mse := ComputeMSE(a, b)
	if mse != 0 {
		t.Errorf("MSE of identical images = %f, want 0", mse)
	}
}

// TestComputeMSE_DifferentImages verifies that differing images produce non-zero error.
func TestComputeMSE_DifferentImages(t *testing.T) {
	a := bitmap.NewDirectBitmap(2, 2)
	b := bitmap.NewDirectBitmap(2, 2)
	// a is all black, b is all white
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			a.SetPixelColor(x, y, bitmap.NewRgbColor(0, 0, 0))
			b.SetPixelColor(x, y, bitmap.NewRgbColor(255, 255, 255))
		}
	}
	mse := ComputeMSE(a, b)
	expected := 255.0 * 255.0 // (255²+255²+255²) / 3 = 255²
	if math.Abs(mse-expected) > 0.01 {
		t.Errorf("MSE = %f, want %f", mse, expected)
	}
}

// TestComputeMSE_DimensionMismatch returns a large error for mismatched dimensions.
func TestComputeMSE_DimensionMismatch(t *testing.T) {
	a := bitmap.NewDirectBitmap(10, 10)
	b := bitmap.NewDirectBitmap(5, 5)
	mse := ComputeMSE(a, b)
	if mse < 1e17 {
		t.Errorf("MSE of mismatched dimensions = %f, want very large", mse)
	}
}
