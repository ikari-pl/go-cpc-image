// Package main provides the GUI application entry point for Go CPC image.
// This is the Fyne-based graphical interface matching the original WinForms layout.
package main

import (
	"log"
	"os"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/ikari-pl/go-cpc-image/internal/gui"
	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
)

func main() {
	// Create Fyne application
	a := app.NewWithID("com.ikari.go-cpc-image")

	// Apply compact theme for denser toolbar/buttons
	a.Settings().SetTheme(gui.NewCompactTheme())

	// Create and configure the main window
	w := a.NewWindow("Go CPC image!")
	w.Resize(fyne.NewSize(1400, 900))

	// Initialize the GUI application
	guiApp, err := gui.NewApplication(a, w)
	if err != nil {
		log.Fatalf("Failed to create GUI application: %v", err)
	}

	// Handle CLI arguments: [image_path] [mode]
	args := os.Args[1:]
	if len(args) >= 1 {
		img, err := bitmap.NewDirectBitmapFromFile(args[0])
		if err != nil {
			log.Printf("Failed to load %s: %v", args[0], err)
		} else {
			if len(args) >= 2 {
				if mode, err := strconv.Atoi(args[1]); err == nil {
					guiApp.GetParams().VirtualMode = mode
				}
			}
			guiApp.RefreshControls()
			guiApp.SetSourceImage(img)
		}
	}

	// Set up the main window content and show it
	w.SetContent(guiApp.BuildLayout())
	w.ShowAndRun()
}