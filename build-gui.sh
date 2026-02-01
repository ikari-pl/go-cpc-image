#!/bin/bash

# Build script for ConvImgCpc Fyne GUI
# This script sets up the dependencies and builds the GUI application

echo "ConvImgCpc GUI Build Script"
echo "============================="

# Step 1: Add Fyne dependency
echo "Adding Fyne dependency..."
go get fyne.io/fyne/v2
if [ $? -ne 0 ]; then
    echo "Failed to add Fyne dependency"
    exit 1
fi

# Step 2: Update go.mod and go.sum
echo "Updating dependencies..."
go mod tidy
if [ $? -ne 0 ]; then
    echo "Failed to update dependencies"
    exit 1
fi

# Step 3: Build the GUI application
echo "Building GUI application..."
go build -o convimgcpc-gui ./cmd/convimgcpc-gui/
if [ $? -ne 0 ]; then
    echo "Build failed"
    exit 1
fi

echo "Build successful!"
echo "Run with: ./convimgcpc-gui"

# Optional: Show application info
if command -v ./convimgcpc-gui &> /dev/null; then
    echo ""
    echo "Application features:"
    echo "- Split panel layout (source | CPC preview)"
    echo "- Live conversion with 50ms debouncing"
    echo "- Mode controls (0/1/2), overscan toggle"
    echo "- Dithering controls with all methods"
    echo "- Interactive 16-pen palette editor"
    echo "- File operations (open images, save SCR/ASM/DSK/PNG)"
    echo "- Zoom controls (1x/2x/4x)"
    echo "- Demo mode for testing without images"
fi