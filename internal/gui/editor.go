// Package gui provides drawing tools that operate directly on CPC screen data.
package gui

import (
	"github.com/ikari-pl/go-cpc-image/pkg/cpc"
	"github.com/ikari-pl/go-cpc-image/pkg/render"
)

// DrawTool identifies the active drawing tool.
type DrawTool int

const (
	ToolPencil    DrawTool = iota
	ToolLine
	ToolRectangle // filled
	ToolFloodFill
)

// Editor handles mouse-driven drawing on the CPC screen buffer.
type Editor struct {
	app       *Application
	History   *UndoStack
	Tool      DrawTool
	Pen       int // current pen index 0-15
	drawing   bool
	startX    int // start CPC pixel coords (for line/rect)
	startY    int
	lastX     int
	lastY     int
}

// NewEditor creates a new editor wired to the application.
func NewEditor(app *Application) *Editor {
	return &Editor{
		app:     app,
		History: NewUndoStack(),
		Tool:    ToolPencil,
		Pen:     1,
	}
}

// bitmapCpc is a convenience accessor.
func (e *Editor) bitmapCpc() *render.BitmapCpc {
	return e.app.cpcImage.BitmapCpc
}

// mode returns the current CPC mode (0, 1, or 2).
func (e *Editor) mode() int {
	m := e.bitmapCpc().VirtualMode
	if m > 2 {
		m = 1 // fall back to mode 1 for EGX/split/etc.
	}
	return m
}

// pixelWidth returns the CPC pixel width in byte-pixels (mode0=4, mode1=2, mode2=1).
func (e *Editor) pixelWidth() int {
	switch e.mode() {
	case 0:
		return 4
	case 1:
		return 2
	default:
		return 1
	}
}

// maxX returns the CPC screen width in CPC pixel coordinates.
func (e *Editor) maxX() int {
	return (e.bitmapCpc().NumCol * 8) / e.pixelWidth()
}

// maxY returns the CPC screen height in CPC pixel coordinates (each CPC line = 2 display rows).
func (e *Editor) maxY() int {
	return e.bitmapCpc().NumLig * 2
}

// ScreenToCpc converts display pixel coordinates to CPC pixel coordinates.
// displayW/displayH are the actual widget dimensions in display pixels.
func (e *Editor) ScreenToCpc(displayX, displayY float32, displayW, displayH float32) (int, int) {
	b := e.bitmapCpc()
	// The rendered image is NumCol*8 wide, NumLig*2 tall in raw pixels,
	// then displayed with 4:3 aspect scaling. For simplicity we assume
	// the raster fills the widget 1:1 after aspect correction done by the
	// preview widget. Map proportionally.
	rawW := float32(b.NumCol * 8)
	rawH := float32(b.NumLig * 2)

	// Compute the letterboxed 4:3 rectangle within the widget
	aspect := float32(4.0 / 3.0)
	widgetAspect := displayW / displayH
	var scaledW, scaledH, offsetX, offsetY float32
	if aspect > widgetAspect {
		scaledW = displayW
		scaledH = displayW / aspect
	} else {
		scaledH = displayH
		scaledW = displayH * aspect
	}
	offsetX = (displayW - scaledW) / 2
	offsetY = (displayH - scaledH) / 2

	// Position within the scaled CPC image
	relX := (displayX - offsetX) / scaledW
	relY := (displayY - offsetY) / scaledH
	if relX < 0 || relX >= 1 || relY < 0 || relY >= 1 {
		return -1, -1
	}

	// Convert to raw pixel coords, then to CPC pixel coords
	rawPx := int(relX * rawW)
	rawPy := int(relY * rawH)

	cpcX := rawPx / e.pixelWidth()
	cpcY := rawPy

	// Clamp
	if cpcX >= e.maxX() {
		cpcX = e.maxX() - 1
	}
	if cpcY >= e.maxY() {
		cpcY = e.maxY() - 1
	}
	return cpcX, cpcY
}

// SetCpcPixel writes a pen value into the CPC screen data at CPC pixel coordinates.
func (e *Editor) SetCpcPixel(x, y, pen int) {
	b := e.bitmapCpc()
	pw := e.pixelWidth()
	mode := e.mode()

	// Convert CPC pixel x to raw byte-pixel x
	rawX := x * pw
	col := rawX / 8 // byte column
	pixInByte := rawX % 8 // pixel offset within byte

	// CPC address for this scanline
	addr := cpc.CpcAddress(y, b.NumCol, b.NumLig) + col
	if addr < 0 || addr >= len(b.ScreenData) {
		return
	}

	octet := b.ScreenData[addr]

	switch mode {
	case 0: // 2 pixels per byte, each pixel = 4 raw pixels
		// pixel 0 is bits 7,5,3,1; pixel 1 is bits 6,4,2,0
		slot := pixInByte / 4 // 0 or 1
		if pen > 15 {
			pen = 15
		}
		// Clear the slot and set new pen
		if slot == 0 {
			octet &= 0x55 // clear pixel 0 bits (7,5,3,1)
			octet |= byte(cpc.TabOctetMode[pen])
		} else {
			octet &= 0xAA // clear pixel 1 bits (6,4,2,0)
			octet |= byte(cpc.TabOctetMode[pen] >> 1)
		}

	case 1: // 4 pixels per byte, each pixel = 2 raw pixels
		slot := pixInByte / 2 // 0..3
		if pen > 3 {
			pen = 3
		}
		// TabOctetMode encodes pen for slot 0; shift right by slot for other slots
		// Clear the slot mask and set
		// Mode 1 bit layout per slot:
		//   slot 0: bits 7, 3
		//   slot 1: bits 6, 2
		//   slot 2: bits 5, 1
		//   slot 3: bits 4, 0
		mask := byte(0x88 >> uint(slot))
		octet &^= mask
		octet |= byte(cpc.TabOctetMode[pen] >> uint(slot))

	case 2: // 8 pixels per byte, each pixel = 1 raw pixel
		if pen > 1 {
			pen = 1
		}
		bit := uint(7 - pixInByte)
		if pen == 1 {
			octet |= 1 << bit
		} else {
			octet &^= 1 << bit
		}
	}

	b.ScreenData[addr] = octet
}

// GetCpcPixel reads the pen value from CPC screen data at CPC pixel coordinates.
func (e *Editor) GetCpcPixel(x, y int) int {
	b := e.bitmapCpc()
	pw := e.pixelWidth()
	mode := e.mode()

	rawX := x * pw
	col := rawX / 8
	pixInByte := rawX % 8

	addr := cpc.CpcAddress(y, b.NumCol, b.NumLig) + col
	if addr < 0 || addr >= len(b.ScreenData) {
		return 0
	}

	octet := b.ScreenData[addr]

	switch mode {
	case 0:
		slot := pixInByte / 4
		if slot == 0 {
			return int(octet>>7) + int((octet&0x20)>>3) + int((octet&0x08)>>2) + int((octet&0x02)<<2)
		}
		return int((octet&0x40)>>6) + int((octet&0x10)>>2) + int((octet&0x04)>>1) + int((octet&0x01)<<3)

	case 1:
		slot := pixInByte / 2
		switch slot {
		case 0:
			return int((octet>>7)&1) + int((octet>>2)&2)
		case 1:
			return int((octet>>6)&1) + int((octet>>1)&2)
		case 2:
			return int((octet>>5)&1) + int((octet>>0)&2)
		case 3:
			return int((octet>>4)&1) + int((octet<<1)&2)
		}

	case 2:
		bit := uint(7 - pixInByte)
		return int((octet >> bit) & 1)
	}
	return 0
}

// pushUndo saves the current state before a drawing operation.
func (e *Editor) pushUndo() {
	e.History.Push(CaptureSnapshot(e.bitmapCpc()))
}

// refreshDisplay re-renders the CPC screen and refreshes the preview.
func (e *Editor) refreshDisplay() {
	e.app.cpcDisplay.Store(e.app.renderCpcToRGBA())
	e.app.preview.RefreshCpc()
}

// Undo reverts to the previous state.
func (e *Editor) Undo() {
	s := e.History.Undo()
	if s == nil {
		return
	}
	RestoreSnapshot(e.bitmapCpc(), s)
	e.refreshDisplay()
}

// Redo re-applies the next state.
func (e *Editor) Redo() {
	s := e.History.Redo()
	if s == nil {
		return
	}
	RestoreSnapshot(e.bitmapCpc(), s)
	e.refreshDisplay()
}

// --- Drawing operations ---

// OnMouseDown is called when the mouse button is pressed on the CPC preview.
func (e *Editor) OnMouseDown(cpcX, cpcY int) {
	if cpcX < 0 || cpcY < 0 {
		return
	}
	e.pushUndo()
	e.drawing = true
	e.startX = cpcX
	e.startY = cpcY
	e.lastX = cpcX
	e.lastY = cpcY

	switch e.Tool {
	case ToolPencil:
		e.SetCpcPixel(cpcX, cpcY, e.Pen)
		e.refreshDisplay()
	case ToolFloodFill:
		e.floodFill(cpcX, cpcY, e.Pen)
		e.drawing = false
		e.refreshDisplay()
	case ToolLine, ToolRectangle:
		// wait for mouse up
	}
}

// OnMouseDrag is called when the mouse moves while pressed.
func (e *Editor) OnMouseDrag(cpcX, cpcY int) {
	if !e.drawing || cpcX < 0 || cpcY < 0 {
		return
	}

	switch e.Tool {
	case ToolPencil:
		e.drawLine(e.lastX, e.lastY, cpcX, cpcY, e.Pen)
		e.lastX = cpcX
		e.lastY = cpcY
		e.refreshDisplay()
	case ToolLine, ToolRectangle:
		// Could show preview; for now just track endpoint
		e.lastX = cpcX
		e.lastY = cpcY
	}
}

// OnMouseUp is called when the mouse button is released.
func (e *Editor) OnMouseUp(cpcX, cpcY int) {
	if !e.drawing {
		return
	}
	e.drawing = false

	if cpcX < 0 {
		cpcX = e.lastX
	}
	if cpcY < 0 {
		cpcY = e.lastY
	}

	switch e.Tool {
	case ToolLine:
		e.drawLine(e.startX, e.startY, cpcX, cpcY, e.Pen)
		e.refreshDisplay()
	case ToolRectangle:
		e.drawFilledRect(e.startX, e.startY, cpcX, cpcY, e.Pen)
		e.refreshDisplay()
	}
}

// drawLine uses Bresenham's algorithm.
func (e *Editor) drawLine(x0, y0, x1, y1, pen int) {
	dx := x1 - x0
	dy := y1 - y0
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}

	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}

	err := dx - dy

	for {
		e.SetCpcPixel(x0, y0, pen)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

// drawFilledRect draws a filled rectangle.
func (e *Editor) drawFilledRect(x0, y0, x1, y1, pen int) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			e.SetCpcPixel(x, y, pen)
		}
	}
}

// floodFill fills a contiguous region with the given pen using a stack-based approach.
func (e *Editor) floodFill(x, y, pen int) {
	target := e.GetCpcPixel(x, y)
	if target == pen {
		return // already the right color
	}

	maxW := e.maxX()
	maxH := e.maxY()

	type point struct{ x, y int }
	stack := []point{{x, y}}

	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if p.x < 0 || p.x >= maxW || p.y < 0 || p.y >= maxH {
			continue
		}
		if e.GetCpcPixel(p.x, p.y) != target {
			continue
		}

		e.SetCpcPixel(p.x, p.y, pen)
		stack = append(stack, point{p.x + 1, p.y})
		stack = append(stack, point{p.x - 1, p.y})
		stack = append(stack, point{p.x, p.y + 1})
		stack = append(stack, point{p.x, p.y - 1})
	}
}
