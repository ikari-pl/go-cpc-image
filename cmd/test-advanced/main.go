// Test program for advanced CPC conversion modes
package main

import (
	"fmt"

	"github.com/ikari-pl/go-cpc-image/pkg/animation"
	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
	"github.com/ikari-pl/go-cpc-image/pkg/convert"
	"github.com/ikari-pl/go-cpc-image/pkg/splitscreen"
)

func main() {
	fmt.Println("Testing advanced CPC conversion modes...")

	// Test Mode X converter
	fmt.Println("Testing Mode X converter...")
	params := convert.NewDefaultParam()
	_ = convert.NewModeXConverter(params, false)
	fmt.Printf("Mode X converter created with CPC+ mode: %t\n", false)

	// Test Split-screen converter
	fmt.Println("Testing Split converter...")
	_ = convert.NewSplitConverter(params, false)
	fmt.Printf("Split converter created with CPC+ mode: %t\n", false)

	// Test ASCII converter
	fmt.Println("Testing ASCII converter...")
	_ = convert.NewAsciiConverter(params, false)
	fmt.Printf("ASCII converter created\n")

	// Test split-screen structures
	fmt.Println("Testing split-screen structures...")
	splitEcran := splitscreen.NewSplitScreen()
	splitEcran.InitializeLines()
	fmt.Printf("Split screen initialized with %d lines\n", len(splitEcran.SplitLines))

	bitmapCpc := splitscreen.NewBitmapCpc(320, 200, 1)
	fmt.Printf("CPC bitmap created: %dx%d, mode %d\n", bitmapCpc.Width, bitmapCpc.Height, 1)

	// Test animation structures
	fmt.Println("Testing animation structures...")
	diffAnim := animation.NewDiffAnim()
	diffAnim.AddDiff(0xC000, 0x55)
	diffAnim.AddDiff(0xC001, 0xAA)
	fmt.Printf("Delta animation created with %d blocks\n", diffAnim.GetBlockCount())

	imageSource := animation.NewImageSource()
	imageSource.Init()
	fmt.Printf("Image source created with %d frames\n", imageSource.NbImg())

	// Test PatternM1 structures
	fmt.Println("Testing PatternM1 structures...")
	trame := convert.NewPatternM1()
	trame.SetPix(0, 0, 1)
	trame.SetPix(1, 1, 2)
	pen := trame.GetPix(0, 0)
	fmt.Printf("PatternM1 created, pixel (0,0) = %d\n", pen)

	// Test with a small bitmap
	fmt.Println("Testing with bitmap...")
	bmp := bitmap.NewDirectBitmap(16, 16)
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			color := bitmap.NewRgbColor(uint8(x*16), uint8(y*16), 128)
			bmp.SetPixelColor(x, y, color)
		}
	}

	fmt.Printf("Test bitmap created: %dx%d\n", bmp.Width(), bmp.Height())

	// Test conversion structures
	dest := convert.NewSimpleCpcImage(16, 16, 1)
	fmt.Printf("CPC image destination created: %dx%d, mode %d\n", dest.Width, dest.Height, dest.Mode)

	fmt.Println("All tests completed successfully!")
}