package main

import (
	"fmt"
	"github.com/ikari-pl/go-cpc-image/pkg/convert"
	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
	"github.com/ikari-pl/go-cpc-image/pkg/render"
)

func main() {
	fmt.Println("Testing conversion pipeline compilation...")

	// Test creating default parameters
	prm := convert.NewDefaultParam()
	fmt.Printf("Default CPC mode: %d\n", prm.VirtualMode)

	// Test creating a bitmap
	bmp := bitmap.NewDirectBitmap(640, 400)
	fmt.Printf("Created bitmap: %dx%d\n", bmp.Width(), bmp.Height())

	// Test creating ImageCpc
	dest := &convert.ImageCpc{
		BitmapCpc: &render.BitmapCpc{
			CpcPlus: false,
			NumCol: 80,
			NumLig: 200,
		},
	}

	// Test conversion (simplified)
	fmt.Println("Testing conversion pipeline...")
	colors := convert.Convert(bmp, dest, prm, true)
	fmt.Printf("Found %d colors\n", colors)

	fmt.Println("All tests passed!")
}