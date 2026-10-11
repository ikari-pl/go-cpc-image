package main

import (
	"fmt"
	"strings"
)

// resizePolicies lists the --resize values, in the order the help and the error
// message print them.
var resizePolicies = []string{"fit", "keepsmaller", "keeplarger", "origin"}

// parseResizePolicy validates the --resize value. Like the other enumerations of
// this CLI it is case-insensitive and refuses anything it does not know, so a
// batch line cannot silently fall back to the default mapping.
func parseResizePolicy(value string) (string, error) {
	want := strings.ToLower(strings.TrimSpace(value))
	for _, p := range resizePolicies {
		if want == p {
			return want, nil
		}
	}
	return "", fmt.Errorf("invalid resize policy: %s (available: %s)",
		value, strings.Join(resizePolicies, ", "))
}

// screenRect returns the rectangle of the screen grid a picture of srcW x srcH
// lands in under a --resize policy: its width and height, and the top-left offset
// it is drawn at (negative when the picture is bigger than the screen and has to
// be cropped). The screen keeps the pen-0 colour wherever the rectangle does not
// cover it.
//
//	fit         both axes fill the screen (the behaviour before --resize existed);
//	keepsmaller one factor for both axes, the smaller one, so the whole picture is
//	            kept, centred, and a band of background is left free;
//	keeplarger  the same, with the larger factor: the screen is filled and the
//	            overflow of the picture is cut off;
//	origin      no scaling: the picture is drawn at 0,0 and clipped.
func screenRect(policy string, srcW, srcH, screenW, screenH int) (dstW, dstH, offX, offY int) {
	switch policy {
	case "origin":
		// No scaling and no centring: pixel (0,0) of the picture is pixel (0,0)
		// of the screen, and whatever does not fit is dropped.
		return srcW, srcH, 0, 0
	case "keepsmaller", "keeplarger":
		// screenW/srcW <= screenH/srcH decides which axis binds, compared by
		// cross-multiplication so the decision stays exact in integers.
		widthBinds := screenW*srcH <= screenH*srcW
		if policy == "keeplarger" {
			widthBinds = !widthBinds
		}
		if widthBinds {
			dstW, dstH = screenW, srcH*screenW/srcW
		} else {
			dstW, dstH = srcW*screenH/srcH, screenH
		}
	default: // fit
		dstW, dstH = screenW, screenH
	}
	offX = (screenW - dstW) / 2
	offY = (screenH - dstH) / 2
	return dstW, dstH, offX, offY
}
