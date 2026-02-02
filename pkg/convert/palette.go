// Package convert provides palette optimization functionality for go-cpc-image.
// This file contains the FindBestColors function and k-means QuantizePalette.
package convert

import (
	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
	"github.com/ikari-pl/go-cpc-image/pkg/cpc"
)

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// GetValColor returns a value used for palette sorting.
// For CPC+ mode, returns weighted sum of RGB components.
// For standard CPC, returns the color index directly.
func GetValColor(c int, prm *Settings) int {
	if prm.CpcPlus {
		v := c >> 8
		r := (c >> 4) & 0x0F
		b := c & 0x0F
		return r*r*prm.CoefR + v*v*prm.CoefV + b*b*prm.CoefB
	}
	return c
}

// FindBestColors finds the best colors among the possible ones.
// This fills the palette array with the most frequently used colors.
// maxPen: maximum number of palette entries to fill
// lockState: array indicating which palette entries are locked
// prm: conversion parameters
// state: conversion state containing frequency table
func FindBestColors(maxPen int, lockState [16]int, prm *Settings, palette *[16]int, state *ConversionState) {
	// Mark unlocked entries as unassigned (matching C# behavior: Cpc.Palette[i] = 0xFFFF)
	for i := 0; i < 16; i++ {
		if prm.LockState[i] == 0 && lockState[i] == 0 {
			palette[i] = 0xFFFF
		}
	}

	// Determine search space
	FindMax := 27
	if prm.CpcPlus {
		FindMax = 4096
	}

	// Clear locked colors from frequency table
	for x := 0; x < maxPen; x++ {
		if lockState[x] > 0 {
			for y := 0; y < 272; y++ {
				if palette[x] < 4096 {
					state.ColorFrequency[palette[x]][y] = 0
				}
			}
		}
	}

	usedIdx := 0
	maxFreq := 0

	// Find most used colors
	for usedIdx = 0; usedIdx < maxPen; usedIdx++ {
		maxFreq = 0
		if lockState[usedIdx] == 0 && prm.DisableState[usedIdx] == 0 {
			// Find color with highest frequency
			for i := 0; i < FindMax; i++ {
				freq := 0
				for y := 0; y < 272; y++ {
					freq += state.ColorFrequency[i][y]
				}

				if maxFreq < freq {
					maxFreq = freq
					palette[usedIdx] = i
				}
			}

			// Clear this color from frequency table
			for y := 0; y < 272; y++ {
				if palette[usedIdx] < 0xFFFF {
					state.ColorFrequency[palette[usedIdx]][y] = 0
				}
			}

			// If using new reduction method, stop after first color
			if prm.NewReduc {
				break
			}
		}
	}

	// Alternative method: find most different colors among most used
	if prm.NewReduc {
		firstColor := cpc.GetColor(palette[usedIdx], prm.CpcPlus)
		takeDist := false

		for x := 0; x < maxPen; x++ {
			if takeDist {
				// Find most used remaining color
				maxFreq = 0
				if lockState[x] == 0 {
					for i := 0; i < FindMax; i++ {
						freq := 0
						for y := 0; y < 272; y++ {
							freq += state.ColorFrequency[i][y]
						}

						if maxFreq < freq {
							maxFreq = freq
							palette[x] = i
						}
					}
				}
			} else {
				// Find most different color from first color
				if lockState[x] == 0 && x != usedIdx {
					dist := 0
					oldDist := 0

					// Try multiple search passes with decreasing frequency thresholds
					for pass := 3; pass >= 0; pass-- {
						for i := 0; i < FindMax; i++ {
							freq := 0
							for y := 0; y < 272; y++ {
								freq += state.ColorFrequency[i][y]
							}

							if freq > maxFreq>>uint(pass) {
								c := cpc.GetColor(i, prm.CpcPlus)
								dr := int(c.R) - int(firstColor.R)
								dv := int(c.V) - int(firstColor.V)
								db := int(c.B) - int(firstColor.B)
								dist = dr*dr*prm.CoefR + dv*dv*prm.CoefV + db*db*prm.CoefB

								if dist > oldDist {
									oldDist = dist
									palette[x] = i
								}
							}
						}
					}

					// If no different color found, retry
					if oldDist == 0 {
						x--
					}
				}
			}

			// Clear selected color from frequency table
			for y := 0; y < 272; y++ {
				if palette[x] < 0xFFFF {
					state.ColorFrequency[palette[x]][y] = 0
				}
			}

			takeDist = !takeDist
			usedIdx = x
		}
	}

	// Sort palette if requested
	if (prm.NewSortPal & 1) == 1 {
		ascending := (prm.NewSortPal & 2) == 0

		for x := 0; x < maxPen-1; x++ {
			if lockState[x] == 0 && prm.DisableState[x] == 0 && palette[x] != 0xFFFF {
				for c := x + 1; c < maxPen; c++ {
					if lockState[c] == 0 && prm.DisableState[c] == 0 && palette[c] != 0xFFFF {
						valX := GetValColor(palette[x], prm)
						valC := GetValColor(palette[c], prm)

						if (valX > valC && ascending) || (valX < valC && !ascending) {
							// Swap palette entries
							tmp := palette[x]
							palette[x] = palette[c]
							palette[c] = tmp
						}
					}
				}
			}
		}
	}
}

// KMeansCluster represents a k-means cluster level for palette quantization
type KMeansCluster struct {
	CentroidColor bitmap.RgbColor // Average color of this level
	NbR, NbV, NbB int             // Accumulated RGB values
	NbPixels      int             // Number of pixels in this cluster
}

// NewKMeansCluster creates a new k-means level with initial RGB values
func NewKMeansCluster(r, v, b byte) *KMeansCluster {
	return &KMeansCluster{
		CentroidColor: bitmap.NewRgbColor(r, v, b),
		NbR:           0,
		NbV:           0,
		NbB:           0,
		NbPixels:      0,
	}
}

// Reset clears the accumulated values for this level
func (n *KMeansCluster) Reset() {
	n.NbR = 0
	n.NbV = 0
	n.NbB = 0
	n.NbPixels = 0
}

// UpdateCentroid calculates the average color for this cluster
func (n *KMeansCluster) UpdateCentroid() {
	if n.NbPixels > 0 {
		n.CentroidColor.R = byte(n.NbR / n.NbPixels)
		n.CentroidColor.V = byte(n.NbV / n.NbPixels)
		n.CentroidColor.B = byte(n.NbB / n.NbPixels)
	} else {
		n.CentroidColor.R = 0
		n.CentroidColor.V = 0
		n.CentroidColor.B = 0
	}
}

// FindNearestCluster finds the best matching k-means level for a color
// and accumulates the color values for cluster update
func FindNearestCluster(prm *Settings, color bitmap.RgbColor, clusters []*KMeansCluster) *KMeansCluster {
	minDist := 0x7FFFFFFF
	ret := 0
	r := color.R
	v := color.V
	b := color.B

	// Find closest cluster
	for n := 0; n < len(clusters); n++ {
		cNiv := clusters[n].CentroidColor
		var dist int

		// Calculate distance based on selected metric
		switch Distance(prm.KMeansDist) {
		case DISTANCE_EUCLIDE:
			// Weighted Euclidean distance
			dr := int(r) - int(cNiv.R)
			dv := int(v) - int(cNiv.V)
			db := int(b) - int(cNiv.B)
			dist = dr*dr*prm.CoefR + dv*dv*prm.CoefV + db*db*prm.CoefB
		case DISTANCE_SUP:
			// Maximum (supremum) distance
			dr := abs(int(r) - int(cNiv.R))
			dv := abs(int(v) - int(cNiv.V))
			db := abs(int(b) - int(cNiv.B))
			dist = max(dr, max(dv, db))
		default:
			// Manhattan distance (default)
			dist = abs(int(r)-int(cNiv.R))*prm.CoefR +
				abs(int(v)-int(cNiv.V))*prm.CoefV +
				abs(int(b)-int(cNiv.B))*prm.CoefB
		}

		if dist < minDist {
			ret = n
			minDist = dist
		}
	}

	// Accumulate color values for this cluster
	clusters[ret].NbR += int(r)
	clusters[ret].NbV += int(v)
	clusters[ret].NbB += int(b)
	clusters[ret].NbPixels++

	return clusters[ret]
}

// QuantizePalette applies k-means color quantization to reduce the image palette
// This preprocesses the image before main CPC conversion
func QuantizePalette(img *bitmap.DirectBitmap, prm *Settings) {
	// Initialize k-means levels with grayscale distribution
	clusters := make([]*KMeansCluster, prm.KMeansColor)
	for n := 0; n < prm.KMeansColor; n++ {
		gray := byte(255 * n / (prm.KMeansColor - 1))
		clusters[n] = NewKMeansCluster(gray, gray, gray)
	}

	// Calculate screen dimensions and step
	width := prm.GetScreenWidth()
	height := prm.GetScreenHeight()
	stepY := 2
	if prm.TrueColorDither {
		stepY = 4
	}

	// Run k-means for specified number of passes
	for p := 0; p < prm.KMeansPass; p++ {
		// Reset all clusters
		for n := 0; n < prm.KMeansColor; n++ {
			clusters[n].Reset()
		}

		// Assign pixels to clusters
		for y := 0; y < height; y += stepY {
			Tx := cpc.PixelWidth(prm.VirtualMode, y, 0)
			for x := 0; x < width; x += Tx {
				color := img.GetPixelColor(x, y)
				FindNearestCluster(prm, color, clusters)
			}
		}

		// Update cluster centers
		for n := 0; n < prm.KMeansColor; n++ {
			clusters[n].UpdateCentroid()
		}
	}

	// Apply quantized colors back to image
	for y := 0; y < height; y += stepY {
		Tx := cpc.PixelWidth(prm.VirtualMode, y, 0)
		for x := 0; x < width; x += Tx {
			originalColor := img.GetPixelColor(x, y)
			nearest := FindNearestCluster(prm, originalColor, clusters)
			for dx := 0; dx < Tx && x+dx < width; dx++ {
				img.SetPixelColor(x+dx, y, nearest.CentroidColor)
			}
		}
	}
}