// Package main provides the GUI application entry point for ConvImgCpc.
// This is the Fyne-based graphical interface matching the original WinForms layout.
package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/ikari/go-cpc-image/internal/gui"
)

func main() {
	// Create Fyne application
	a := app.NewWithID("com.ikari.convimgcpc")

	// Create and configure the main window
	w := a.NewWindow("ConvImgCpc - CPC Graphics Converter")
	w.Resize(fyne.NewSize(1400, 900))

	// Initialize the GUI application
	guiApp, err := gui.NewApplication(a, w)
	if err != nil {
		log.Fatalf("Failed to create GUI application: %v", err)
	}

	// Set up the main window content and show it
	w.SetContent(guiApp.BuildLayout())
	w.ShowAndRun()
}