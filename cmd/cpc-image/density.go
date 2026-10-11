package main

// Every screen PNG this tool writes declares the density of the grid it holds, in
// pixels per metre (the unit the pHYs chunk carries), which is the density of ONE
// PIXEL OF THE FILE — that is what makes one rule correct for every grid and every
// scan area.

// pixelDensityBase returns the density of one CPC pixel of a mode, with a mode-1
// (square, PAL) pixel as the reference: 1416 pixels per metre, ~36 DPI. The modes
// divide the same pixel clock, so a mode-0 pixel is twice as wide as it is tall and
// a mode-2 one half as wide, and since the pair counts pixels per metre in each axis
// the wider pixel is the one with the smaller x.
//
// A mode outside the table — EGX alternates two pixel widths inside one screen, so
// it has no single pixel to describe — keeps the reference pair instead of borrowing
// a mode's own. That fallback is deliberate, not an oversight.
func pixelDensityBase(mode int) (x, y uint32) {
	switch mode {
	case 0:
		return 708, 1416
	case 2:
		return 2832, 1416
	default:
		return 1416, 1416
	}
}

// pixelDuplication returns how many file pixels one CPC pixel is written as in the
// grid screenOutputSize produces, which is what turns the density of a CPC pixel
// into the density of a file. The default grids are the PAR-pre-compensated ones and
// square their pixels by duplicating: mode 0 across, so its 708 is written as 1416,
// and mode 2 down, one row per half of a CPC line, so its 1416 becomes 2832 — while
// mode 1's grid is already 1:1. `--native` drops that pre-compensation and leaves
// the mode's own pair.
func pixelDuplication(mode int, native bool) (x, y uint32) {
	if native {
		return 1, 1
	}
	switch mode {
	case 0:
		return 2, 1
	case 2:
		return 1, 2
	default:
		return 1, 1
	}
}

// pixelDensity returns the density of the grid a screen PNG is written at, as the
// pair the pHYs chunk declares: the density of the mode's own pixel times how many
// file pixels that pixel became, and times --scale, which magnifies the file without
// changing the screen it shows.
//
// It does not depend on the scan area — a CPC pixel is the same size in an underscan,
// a standard and an overscan screen and only the border around the picture changes —
// nor on --resize, which decides what the source was fitted to, not the size of a
// pixel. So: 1416 x 1416 for a mode-1 grid and for a mode-0 default one, 708 x 1416
// for a native mode-0 grid, 2832 x 1416 for a native mode-2 one and 2832 x 2832 for a
// default mode-2 one, whose whole canvas puts every CPC line on two rows.
func pixelDensity(numCol, numLig, mode int, native bool, scale int) (ppux, ppuy uint32) {
	baseX, baseY := pixelDensityBase(mode)
	dupX, dupY := pixelDuplication(mode, native)
	if scale < 1 {
		scale = 1
	}
	return baseX * dupX * uint32(scale), baseY * dupY * uint32(scale)
}

// densityInDPI converts a pixels-per-metre density to the whole number of dots per
// inch the -v line reports: 1416 px/m is 35.97 ppi, printed as 36.
func densityInDPI(pixelsPerMetre uint32) uint32 {
	// One inch is 0.0254 m, so ppi = px/m * 0.0254, rounded to the nearest whole
	// number as (px/m * 127 + 2500) / 5000.
	return (pixelsPerMetre*127 + 2500) / 5000
}
