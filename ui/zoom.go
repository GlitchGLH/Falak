package ui

import (
	"image"
	"image/draw"

	xdraw "golang.org/x/image/draw"
)

// ZoomFrame returns src scaled about its center by sc (1 = full size) with
// every premultiplied channel faded by alpha (0..1), on a canvas of src's
// own size. The close animation shrinks and fades the last frame this way.
func ZoomFrame(src *image.RGBA, sc, alpha float64) *image.RGBA {
	w, h := src.Rect.Dx(), src.Rect.Dy()
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	if sc > 0.001 && alpha > 0.001 {
		sw, sh := int(float64(w)*sc+0.5), int(float64(h)*sc+0.5)
		if sw > 0 && sh > 0 {
			zoomed := image.NewRGBA(image.Rect(0, 0, sw, sh))
			xdraw.CatmullRom.Scale(zoomed, zoomed.Bounds(), src, src.Rect, draw.Src, nil)
			dx, dy := (w-sw)/2, (h-sh)/2
			r := image.Rect(0, 0, sw, sh).Add(image.Pt(dx, dy)).Intersect(out.Rect)
			if !r.Empty() {
				draw.Draw(out, r, zoomed, image.Pt(r.Min.X-dx, r.Min.Y-dy), draw.Src)
			}
		}
	}
	if alpha < 1 {
		a := uint32(alpha * 65535)
		for i, p := range out.Pix {
			out.Pix[i] = uint8(uint32(p) * a >> 16)
		}
	}
	return out
}
