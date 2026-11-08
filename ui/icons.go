package ui

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/fogleman/gg"
	xdraw "golang.org/x/image/draw"
)

// IconScale is the size of tool icons relative to the 19 px box the wheel
// asks for; the icons are drawn a little larger and bolder than before,
// like the reference (thick, rounded, modern glyphs).
const IconScale = 1.16

// drawIcon draws a rounded, bold line icon centered at (x, y) inside a box
// of side s. All coordinates below are in units of half the box (-1..1).
func drawIcon(dc *gg.Context, id string, x, y, s float64, c color.Color) {
	// Drawn 4x larger, then scaled down: thin, crisp, smooth strokes from
	// the downscale's anti-aliasing, snapped to whole pixels.
	const ss = 4
	box := s * IconScale
	fw := int(math.Ceil(box*ss)) + 4
	frame := image.NewRGBA(image.Rect(0, 0, fw, fw))
	sdc := gg.NewContextForRGBA(frame)
	drawIconBody(sdc, id, float64(fw)/2, float64(fw)/2, box*ss, c)
	small := image.NewRGBA(image.Rect(0, 0, int(math.Ceil(box))+2, int(math.Ceil(box))+2))
	xdraw.CatmullRom.Scale(small, small.Bounds(), frame, frame.Bounds(), draw.Src, nil)
	dc.DrawImage(small, int(math.Round(x-float64(small.Rect.Dx())/2)), int(math.Round(y-float64(small.Rect.Dy())/2)))
}

// drawIconBody draws the icon's strokes; drawIcon supersamples them and
// scales the result down. The box side s is the final on-screen size.
func drawIconBody(dc *gg.Context, id string, x, y, s float64, c color.Color) {
	dc.Push()
	defer dc.Pop()
	dc.SetColor(c)
	h := s / 2
	lw := s * (1.75 / 24) // about 1.75 px at the 24 px design size
	dc.SetLineWidth(lw)
	dc.SetLineCap(gg.LineCapRound)
	dc.SetLineJoin(gg.LineJoinRound)
	P := func(px, py float64) (float64, float64) { return x + px*h, y + py*h }
	moveTo := func(px, py float64) { dc.MoveTo(P(px, py)) }
	lineTo := func(px, py float64) { dc.LineTo(P(px, py)) }
	line := func(x0, y0, x1, y1 float64) {
		dc.NewSubPath()
		moveTo(x0, y0)
		lineTo(x1, y1)
		dc.Stroke()
	}
	poly := func(closed bool, pts ...[2]float64) {
		dc.NewSubPath()
		moveTo(pts[0][0], pts[0][1])
		for _, p := range pts[1:] {
			lineTo(p[0], p[1])
		}
		if closed {
			dc.ClosePath()
		}
	}
	// chevron head at (x1,y1) for a shaft coming from (x0,y0)
	head := func(x0, y0, x1, y1, l float64) {
		a := math.Atan2(y1-y0, x1-x0)
		poly(false,
			[2]float64{x1 - l*math.Cos(a-0.75), y1 - l*math.Sin(a-0.75)},
			[2]float64{x1, y1},
			[2]float64{x1 - l*math.Cos(a+0.75), y1 - l*math.Sin(a+0.75)})
		dc.Stroke()
	}
	arrow := func(x0, y0, x1, y1 float64) {
		line(x0, y0, x1, y1)
		head(x0, y0, x1, y1, 0.48)
	}
	rrect := func(x0, y0, w, hh, r float64) {
		dc.NewSubPath()
		dc.DrawRoundedRectangle(x+x0*h, y+y0*h, w*h, hh*h, r*h)
	}
	circle := func(cx, cy, r float64) {
		dc.NewSubPath()
		dc.DrawCircle(x+cx*h, y+cy*h, r*h)
	}
	dot := func(cx, cy, r float64) { circle(cx, cy, r); dc.Fill() }
	// rotate a point list by angle (radians) about the origin
	rot := func(a float64, pts ...[2]float64) [][2]float64 {
		out := make([][2]float64, len(pts))
		for i, p := range pts {
			out[i] = [2]float64{p[0]*math.Cos(a) - p[1]*math.Sin(a), p[0]*math.Sin(a) + p[1]*math.Cos(a)}
		}
		return out
	}

	switch id {
	case "compress": // two arrows pointing in to the center
		arrow(-0.92, -0.92, -0.14, -0.14)
		arrow(0.92, 0.92, 0.14, 0.14)
	case "metadata": // tag with a hole, pointing down-left
		pts := rot(-math.Pi/4,
			[2]float64{-1.0, 0}, [2]float64{-0.42, -0.52}, [2]float64{0.92, -0.52},
			[2]float64{0.92, 0.52}, [2]float64{-0.42, 0.52})
		poly(true, pts...)
		dc.Stroke()
		hp := rot(-math.Pi/4, [2]float64{0.5, 0})[0]
		dot(hp[0], hp[1], 0.15)
	case "rotate": // circular arrow
		const r = 0.78
		a0, a1 := -math.Pi*0.30, math.Pi*1.25
		dc.NewSubPath()
		dc.DrawArc(x, y, r*h, a0, a1)
		dc.Stroke()
		ex, ey := r*math.Cos(a0), r*math.Sin(a0)
		poly(false, [2]float64{ex - 0.52, ey - 0.08}, [2]float64{ex, ey}, [2]float64{ex + 0.08, ey + 0.52})
		dc.Stroke()
	case "resize": // make smaller: frame, small filled picture, arrow in
		rrect(-0.88, -0.88, 1.76, 1.76, 0.36)
		dc.Stroke()
		rrect(-0.88, 0.02, 0.86, 0.86, 0.26)
		dc.Fill()
		arrow(0.56, -0.56, 0.14, -0.14)
	case "crop":
		poly(false, [2]float64{-0.5, -1}, [2]float64{-0.5, 0.5}, [2]float64{1, 0.5})
		dc.Stroke()
		poly(false, [2]float64{-1, -0.5}, [2]float64{0.5, -0.5}, [2]float64{0.5, 1})
		dc.Stroke()
	case "edit": // sliders
		for i, k := range []float64{-0.35, 0.4, -0.05} {
			yy := (float64(i) - 1) * 0.66
			line(-0.92, yy, 0.92, yy)
			dot(k, yy, 0.25)
		}
	case "redact": // eye with a slash
		dc.NewSubPath()
		moveTo(-0.95, 0)
		dc.CubicTo(x-0.55*h, y-0.78*h, x+0.55*h, y-0.78*h, x+0.95*h, y)
		dc.CubicTo(x+0.55*h, y+0.78*h, x-0.55*h, y+0.78*h, x-0.95*h, y)
		dc.ClosePath()
		dc.Stroke()
		dot(0, 0, 0.27)
		line(-0.82, -0.82, 0.82, 0.82)
	case "mute": // filled speaker with an x
		poly(true, [2]float64{-0.92, -0.32}, [2]float64{-0.52, -0.32}, [2]float64{-0.08, -0.78},
			[2]float64{-0.08, 0.78}, [2]float64{-0.52, 0.32}, [2]float64{-0.92, 0.32})
		dc.FillPreserve()
		dc.Stroke()
		line(0.32, -0.3, 0.9, 0.3)
		line(0.9, -0.3, 0.32, 0.3)
	case "snapshot": // camera
		dc.NewSubPath()
		moveTo(-0.95, -0.25)
		dc.QuadraticTo(x-0.95*h, y-0.5*h, x-0.7*h, y-0.5*h)
		lineTo(-0.42, -0.5)
		lineTo(-0.26, -0.78)
		lineTo(0.26, -0.78)
		lineTo(0.42, -0.5)
		lineTo(0.7, -0.5)
		dc.QuadraticTo(x+0.95*h, y-0.5*h, x+0.95*h, y-0.25*h)
		lineTo(0.95, 0.6)
		dc.QuadraticTo(x+0.95*h, y+0.85*h, x+0.7*h, y+0.85*h)
		lineTo(-0.7, 0.85)
		dc.QuadraticTo(x-0.95*h, y+0.85*h, x-0.95*h, y+0.6*h)
		dc.ClosePath()
		dc.Stroke()
		circle(0, 0.15, 0.34)
		dc.Stroke()
	case "normalize": // level meter bars
		for i, hh := range []float64{0.35, 0.68, 1.0, 0.68, 0.35} {
			xx := (float64(i) - 2) * 0.44
			line(xx, -hh*0.82, xx, hh*0.82)
		}
	case "visualize": // waveform in a frame
		rrect(-0.95, -0.78, 1.9, 1.56, 0.34)
		dc.Stroke()
		dc.NewSubPath()
		moveTo(-0.62, 0)
		for i, v := range []float64{0, -0.4, 0.38, -0.18, 0.42, 0} {
			lineTo(-0.62+float64(i+1)*0.2, v)
		}
		dc.Stroke()
	case "channels": // two speakers
		for _, xx := range []float64{-0.47, 0.47} {
			rrect(xx-0.36, -0.88, 0.72, 1.76, 0.24)
			dc.Stroke()
			circle(xx, 0.3, 0.17)
			dc.Stroke()
			dot(xx, -0.42, 0.09)
		}
	case "extract": // open box with an arrow out
		rrect(-0.88, -0.86, 1.76, 0.5, 0.18)
		dc.Stroke()
		poly(false, [2]float64{-0.72, -0.36}, [2]float64{-0.72, 0.62}, [2]float64{0.72, 0.62}, [2]float64{0.72, -0.36})
		dc.Stroke()
		arrow(0, 0.36, 0, -0.06)
	case "trim": // scissors
		circle(-0.5, 0.55, 0.27)
		dc.Stroke()
		circle(0.5, 0.55, 0.27)
		dc.Stroke()
		line(-0.32, 0.34, 0.55, -0.88)
		line(0.32, 0.34, -0.55, -0.88)
	case "speed": // gauge with a needle
		dc.NewSubPath()
		dc.DrawArc(x, y+0.14*h, 0.88*h, math.Pi*0.80, math.Pi*2.20)
		dc.Stroke()
		line(0, 0.14, 0.42, -0.36)
		dot(0, 0.14, 0.17)
	case "split": // film strip
		rrect(-0.72, -0.92, 1.44, 1.84, 0.26)
		dc.Stroke()
		line(-0.3, -0.92, -0.3, 0.92)
		line(0.3, -0.92, 0.3, 0.92)
		for _, yy := range []float64{-0.5, 0, 0.5} {
			line(-0.72, yy, -0.3, yy)
			line(0.3, yy, 0.72, yy)
		}
	case "merge": // two arrows joining into one page
		arrow(-0.82, -0.95, -0.24, -0.4)
		arrow(0.82, -0.95, 0.24, -0.4)
		rrect(-0.62, -0.08, 1.24, 1.0, 0.24)
		dc.Stroke()
	case "blackbg": // filled frame with a white mountain line
		rrect(-0.92, -0.78, 1.84, 1.56, 0.34)
		dc.FillPreserve()
		dc.Stroke()
		// the line inside is cut out of the fill: dark on light icons
		if r, g, b, _ := c.RGBA(); r+g+b > 3*0x8000 {
			dc.SetColor(color.NRGBA{30, 28, 27, 235})
		} else {
			dc.SetColor(color.NRGBA{255, 255, 255, 235})
		}
		poly(false, [2]float64{-0.6, 0.42}, [2]float64{-0.2, -0.04}, [2]float64{0.18, 0.3},
			[2]float64{0.42, 0.1}, [2]float64{0.62, 0.32})
		dc.Stroke()
	case "bleep": // waveform with a cut
		dc.NewSubPath()
		moveTo(-0.95, 0)
		for i, v := range []float64{-0.3, 0.5, -0.72, 0.72, -0.5, 0.3, 0} {
			lineTo(-0.95+float64(i+1)*0.27, v)
		}
		dc.Stroke()
	case "addbg": // picture frame
		rrect(-0.92, -0.78, 1.84, 1.56, 0.34)
		dc.Stroke()
		poly(false, [2]float64{-0.62, 0.42}, [2]float64{-0.2, -0.04}, [2]float64{0.18, 0.3},
			[2]float64{0.42, 0.1}, [2]float64{0.62, 0.32})
		dc.Stroke()
		dot(0.38, -0.36, 0.13)
	case "annotate": // pencil
		pts := rot(-math.Pi/4,
			[2]float64{-1.05, 0}, [2]float64{-0.62, -0.3}, [2]float64{0.92, -0.3},
			[2]float64{0.92, 0.3}, [2]float64{-0.62, 0.3})
		poly(true, pts...)
		dc.Stroke()
		a := rot(-math.Pi/4, [2]float64{0.5, -0.3}, [2]float64{0.5, 0.3})
		line(a[0][0], a[0][1], a[1][0], a[1][1])
	}
}

// DrawIcon is drawIcon for the preview tools.
func DrawIcon(dc *gg.Context, id string, x, y, s float64, c color.Color) {
	drawIcon(dc, id, x, y, s, c)
}
