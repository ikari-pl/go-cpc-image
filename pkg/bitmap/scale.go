package bitmap

import "image"

// ScaleImage resamples an RGBA picture to dstW x dstH by nearest neighbour:
// whole pixels are replicated or dropped, never blended, so a pen colour comes
// through exactly and an integer factor stays exact. It is the single place a
// picture gains pixels — the output stage reaches the file's grid through it, and
// so does the display canvas a screen is magnified onto.
func ScaleImage(src *image.RGBA, dstW, dstH int) *image.RGBA {
	if src == nil || dstW <= 0 || dstH <= 0 {
		return src
	}
	b := src.Bounds()
	if b.Dx() == dstW && b.Dy() == dstH {
		return src
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for y := 0; y < dstH; y++ {
		sy := b.Min.Y + y*b.Dy()/dstH
		if sy >= b.Max.Y {
			sy = b.Max.Y - 1
		}
		row := dst.PixOffset(0, y)
		for x := 0; x < dstW; x++ {
			sx := b.Min.X + x*b.Dx()/dstW
			if sx >= b.Max.X {
				sx = b.Max.X - 1
			}
			si := src.PixOffset(sx, sy)
			copy(dst.Pix[row+x*4:], src.Pix[si:si+4])
		}
	}
	return dst
}
