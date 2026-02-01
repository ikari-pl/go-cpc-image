// Package fileio provides project save/load functionality using the .gocpcimg format.
// A .gocpcimg file is a ZIP archive containing settings.json, palette.json, and source.png.
package fileio

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"

	"github.com/ikari-pl/go-cpc-image/pkg/convert"
)

// SaveProject saves a complete project state to a .gocpcimg file.
// The file is a ZIP archive containing:
//   - settings.json: all conversion parameters
//   - palette.json: the 16-entry palette array
//   - source.png: the source image encoded as PNG
func SaveProject(filename string, params *convert.Settings, sourceImg *image.RGBA) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create project file: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	// Write settings.json
	settingsData, err := json.MarshalIndent(params, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	w, err := zw.Create("settings.json")
	if err != nil {
		return fmt.Errorf("create settings.json in zip: %w", err)
	}
	if _, err := w.Write(settingsData); err != nil {
		return fmt.Errorf("write settings.json: %w", err)
	}

	// Write palette.json
	paletteData, err := json.MarshalIndent(params.Palette, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal palette: %w", err)
	}
	w, err = zw.Create("palette.json")
	if err != nil {
		return fmt.Errorf("create palette.json in zip: %w", err)
	}
	if _, err := w.Write(paletteData); err != nil {
		return fmt.Errorf("write palette.json: %w", err)
	}

	// Write source.png
	w, err = zw.Create("source.png")
	if err != nil {
		return fmt.Errorf("create source.png in zip: %w", err)
	}
	if err := png.Encode(w, sourceImg); err != nil {
		return fmt.Errorf("encode source.png: %w", err)
	}

	return nil
}

// LoadProject loads a project state from a .gocpcimg file.
// Returns the conversion parameters and source image.
func LoadProject(filename string) (*convert.Settings, *image.RGBA, error) {
	zr, err := zip.OpenReader(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("open project file: %w", err)
	}
	defer zr.Close()

	var params *convert.Settings
	var sourceImg *image.RGBA

	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, nil, fmt.Errorf("open %s in zip: %w", f.Name, err)
		}

		switch f.Name {
		case "settings.json":
			data, err := io.ReadAll(rc)
			if err != nil {
				rc.Close()
				return nil, nil, fmt.Errorf("read settings.json: %w", err)
			}
			params = &convert.Settings{}
			if err := json.Unmarshal(data, params); err != nil {
				rc.Close()
				return nil, nil, fmt.Errorf("unmarshal settings.json: %w", err)
			}

		case "source.png":
			data, err := io.ReadAll(rc)
			if err != nil {
				rc.Close()
				return nil, nil, fmt.Errorf("read source.png: %w", err)
			}
			img, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				rc.Close()
				return nil, nil, fmt.Errorf("decode source.png: %w", err)
			}
			// Convert to *image.RGBA if needed
			if rgba, ok := img.(*image.RGBA); ok {
				sourceImg = rgba
			} else {
				bounds := img.Bounds()
				sourceImg = image.NewRGBA(bounds)
				for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
					for x := bounds.Min.X; x < bounds.Max.X; x++ {
						sourceImg.Set(x, y, img.At(x, y))
					}
				}
			}
		}

		rc.Close()
	}

	if params == nil {
		return nil, nil, fmt.Errorf("settings.json not found in project file")
	}
	if sourceImg == nil {
		return nil, nil, fmt.Errorf("source.png not found in project file")
	}

	return params, sourceImg, nil
}
