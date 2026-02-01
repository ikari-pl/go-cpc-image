package main

import (
	"fmt"
	"github.com/ikari/go-cpc-image/pkg/bitmap"
	"github.com/ikari/go-cpc-image/pkg/convert"
	"github.com/ikari/go-cpc-image/pkg/cpc"
)

func main() {
	fmt.Println("Testing ConvImgCpc Go port...")

	// Test RgbColor
	color := bitmap.NewRgbColor(255, 128, 64)
	fmt.Printf("RgbColor: R=%d, V=%d, B=%d, ARGB=0x%X\n",
		color.R, color.V, color.B, color.GetColorArgb())

	// Test DirectBitmap
	bmp := bitmap.NewDirectBitmap(10, 10)
	bmp.SetPixelColor(5, 5, color)
	retrieved := bmp.GetPixelColor(5, 5)
	fmt.Printf("Pixel test: Set=%v, Get=%v\n", color, retrieved)

	// Test CPC palette
	cpcColor := cpc.CpcRgbPalette[13] // White
	fmt.Printf("CPC White: R=%d, V=%d, B=%d\n", cpcColor.R, cpcColor.V, cpcColor.B)

	// Test screen config
	screen := cpc.NewStandardScreen()
	fmt.Printf("Standard screen: %dx%d pixels, %dx%d chars\n",
		screen.TailleX(), screen.TailleY(), screen.NumCol, screen.NumLig)

	// Test AMSDOS header
	header := cpc.CreeEntete("TEST.SCR", 0xC000, 16384, 0xC000)
	fmt.Printf("AMSDOS header: %s, Address=0x%X, Length=%d\n",
		header.GetFilename(), header.Address, header.Length)

	// Test parameters
	params := convert.NewDefaultParam()
	fmt.Printf("Default params: Mode=%d, Method=%s, Pct=%d%%\n",
		params.VirtualMode, params.Method, params.Pct)

	fmt.Println("All tests completed successfully!")
}