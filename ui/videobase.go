package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/fogleman/gg"
	xdraw "golang.org/x/image/draw"
)

// The shared base of the video windows (Trim, Snapshots, later Split, Speed
// and Crop): the Compress dialog's solid card, the X and the title in the
// same places, a preview area, a thumbnail strip and an action button. All
// geometry is in logical units; Arabic mirrors everything, the timeline
// included (time runs from the right, like the Arabic progress bar).

const (
	vidW     = 496.0 // card width
	vidPad   = 24.0
	vidPrevY = 76.0
	vidPrevH = 252.0 // 16:9 for the full inner width (448)
)

// videoCard holds what every video window draws with.
type videoCard struct {
	Scale  float64
	W, H   int // window pixels
	Fonts  *Fonts
	Dark   bool
	RTL    bool
	cardH  float64
	base   *image.RGBA
	work   *image.RGBA
	dc     *gg.Context
	strips map[*image.RGBA]*image.RGBA // thumbnails already scaled to their cell
}

func newVideoCard(scale float64, fonts *Fonts, dark, rtl bool, cardH float64) videoCard {
	c := videoCard{Scale: scale, Fonts: fonts, Dark: dark, RTL: rtl, cardH: cardH}
	c.W = int(math.Ceil((vidW + 2*PanelMargin) * scale))
	c.H = int(math.Ceil((cardH + 2*PanelMargin) * scale))
	c.work = image.NewRGBA(image.Rect(0, 0, c.W, c.H))
	c.dc = gg.NewContextForRGBA(c.work)
	c.base = glassBase(c.W, c.H, vidW, cardH, PanelMargin, panelR, scale, nil, dark)
	c.strips = map[*image.RGBA]*image.RGBA{}
	return c
}

// Size is the window size in pixels.
func (c *videoCard) Size() (int, int) { return c.W, c.H }

// px turns a logical card coordinate into a window pixel.
func (c *videoCard) px(v float64) float64 { return (PanelMargin + v) * c.Scale }

// mirror returns the logical x of a box of width w at logical x for the
// current direction (Arabic: measured from the right edge).
func (c *videoCard) mirror(x, w float64) float64 {
	if c.RTL {
		return vidW - x - w
	}
	return x
}

// PreviewSize is the preview area in window pixels: frames are asked for at
// this size.
func (c *videoCard) PreviewSize() (int, int) {
	return int(math.Round((vidW - 2*vidPad) * c.Scale)), int(math.Round(vidPrevH * c.Scale))
}

func (c *videoCard) grey() color.NRGBA {
	if c.Dark {
		return color.NRGBA{214, 214, 220, 255}
	}
	return color.NRGBA{88, 88, 93, 255}
}

func (c *videoCard) textCol() color.NRGBA {
	if c.Dark {
		return color.NRGBA{235, 235, 238, 255}
	}
	return color.NRGBA{40, 40, 45, 255}
}

var errorRed = color.NRGBA{214, 61, 46, 255}

// text draws s with its baseline centred on logical y; logical x is the
// side reading starts on (the left in English, the right in Arabic).
func (c *videoCard) text(s string, x, y, size float64, bold bool, col color.Color) {
	f := c.Fonts.Face(math.Round(size*c.Scale*4)/4, bold)
	if f == nil || s == "" {
		return
	}
	dc := c.dc
	dc.SetFontFace(f)
	dc.SetColor(col)
	t := Visual(s)
	px := c.px(x)
	if c.RTL {
		tw, _ := dc.MeasureString(t)
		px -= tw
	}
	dc.DrawString(t, px, c.px(y)+capHeightOf(f)/2)
}

// textStart draws s at the reading-start padding of the card.
func (c *videoCard) textStart(s string, y, size float64, bold bool, col color.Color) {
	x := vidPad
	if c.RTL {
		x = vidW - vidPad
	}
	c.text(s, x, y, size, bold, col)
}

// textCenter draws s centred on logical x.
func (c *videoCard) textCenter(s string, x, y, size float64, bold bool, col color.Color) {
	f := c.Fonts.Face(math.Round(size*c.Scale*4)/4, bold)
	if f == nil || s == "" {
		return
	}
	dc := c.dc
	dc.SetFontFace(f)
	dc.SetColor(col)
	t := Visual(s)
	tw, _ := dc.MeasureString(t)
	dc.DrawString(t, c.px(x)-tw/2, c.px(y)+capHeightOf(f)/2)
}

// begin starts a frame: the card, the X (top left, as in the Compress
// dialog), the title and the file name under it.
func (c *videoCard) begin(title, file string) {
	copy(c.work.Pix, c.base.Pix)
	dc, s := c.dc, c.Scale
	cx, cy := c.px(cdlgClose[0]), c.px(cdlgClose[1])
	dc.DrawCircle(cx, cy, cdlgClose[2]*s)
	if c.Dark {
		dc.SetColor(color.NRGBA{255, 255, 255, 14})
	} else {
		dc.SetColor(color.NRGBA{0, 0, 0, 8})
	}
	dc.Fill()
	dd := 4.6 * s
	dc.SetLineCap(gg.LineCapRound)
	dc.SetLineWidth(1.7 * s)
	dc.SetColor(c.grey())
	dc.DrawLine(cx-dd, cy-dd, cx+dd, cy+dd)
	dc.DrawLine(cx+dd, cy-dd, cx-dd, cy+dd)
	dc.Stroke()
	tx := 60.0
	if c.RTL {
		tx = vidW - vidPad
	}
	c.text(title, tx, 32.5, 17, true, c.textCol())
	if f := c.Fonts.Face(math.Round(13*s*4)/4, true); f != nil {
		dc.SetFontFace(f)
		c.text(fitText(dc, file, (vidW-60-vidPad)*s), tx, 54, 13, true, Accent)
	}
}

// overClose reports whether window pixel (x, y) is on the X.
func (c *videoCard) overClose(x, y float64) bool {
	return math.Hypot(x-c.px(cdlgClose[0]), y-c.px(cdlgClose[1])) <= (cdlgClose[2]+4)*c.Scale
}

// in reports whether window pixel (x, y) is inside logical rect r (x, y,
// w, h; x already mirrored as needed), grown by pad on every side.
func (c *videoCard) in(x, y float64, r [4]float64, pad float64) bool {
	rx, ry := c.px(r[0]-pad), c.px(r[1]-pad)
	return x >= rx && x <= rx+(r[2]+2*pad)*c.Scale && y >= ry && y <= ry+(r[3]+2*pad)*c.Scale
}

// preview draws the frame (fitted, letterboxed on black) or, while there is
// none, the status line, with a thin bar when progress >= 0.
func (c *videoCard) preview(frame *image.RGBA, status string, progress float64) {
	dc, s := c.dc, c.Scale
	x, y, w, h := c.px(vidPad), c.px(vidPrevY), (vidW-2*vidPad)*s, vidPrevH*s
	dc.DrawRoundedRectangle(x, y, w, h, 10*s)
	dc.SetColor(color.NRGBA{12, 12, 14, 255})
	dc.Fill()
	if frame != nil {
		fw, fh := float64(frame.Rect.Dx()), float64(frame.Rect.Dy())
		k := math.Min(w/fw, h/fh)
		dw, dh := fw*k, fh*k
		dst := image.Rect(int(math.Round(x+(w-dw)/2)), int(math.Round(y+(h-dh)/2)), int(math.Round(x+(w+dw)/2)), int(math.Round(y+(h+dh)/2)))
		if dst.Dx() == frame.Rect.Dx() && dst.Dy() == frame.Rect.Dy() {
			xdraw.Copy(c.work, dst.Min, frame, frame.Rect, xdraw.Over, nil)
		} else {
			xdraw.ApproxBiLinear.Scale(c.work, dst, frame, frame.Rect, xdraw.Over, nil)
		}
		c.roundCorners(x, y, w, h, 10*s)
		return
	}
	light := color.NRGBA{225, 225, 230, 255}
	c.textCenter(status, vidW/2, vidPrevY+vidPrevH/2-(func() float64 {
		if progress >= 0 {
			return 8
		}
		return 0
	})(), 13, false, light)
	if progress >= 0 {
		bw := 160 * s
		bx, by := c.px(vidW/2)-bw/2, c.px(vidPrevY+vidPrevH/2+10)
		dc.DrawRoundedRectangle(bx, by, bw, 4*s, 2*s)
		dc.SetColor(color.NRGBA{255, 255, 255, 40})
		dc.Fill()
		fill := bw * math.Min(1, progress)
		fx := bx
		if c.RTL {
			fx = bx + bw - fill
		}
		if fill > 0 {
			dc.DrawRoundedRectangle(fx, by, fill, 4*s, 2*s)
			dc.SetColor(Accent)
			dc.Fill()
		}
	}
}

// roundCorners paints the card colour back over the four corners outside
// a rounded rect of radius r, so a picture drawn into it has round corners
// (no clip: gg keeps a clip after Pop).
func (c *videoCard) roundCorners(x, y, w, h, r float64) {
	card := color.RGBA{255, 255, 255, 255}
	if c.Dark {
		card = color.RGBA{40, 40, 44, 255}
	}
	for _, corner := range [4][2]float64{{x, y}, {x + w - r, y}, {x, y + h - r}, {x + w - r, y + h - r}} {
		ccx := corner[0] + r
		if corner[0] > x {
			ccx = corner[0]
		}
		ccy := corner[1] + r
		if corner[1] > y {
			ccy = corner[1]
		}
		for py := int(corner[1]); py < int(math.Ceil(corner[1]+r)); py++ {
			for qx := int(corner[0]); qx < int(math.Ceil(corner[0]+r)); qx++ {
				d := math.Hypot(float64(qx)+0.5-ccx, float64(py)+0.5-ccy)
				a := math.Max(0, math.Min(1, d-r+0.5)) // 0 inside, 1 outside
				if a <= 0 || !(image.Point{qx, py}.In(c.work.Rect)) {
					continue
				}
				i := c.work.PixOffset(qx, py)
				p := c.work.Pix[i : i+4]
				p[0] = uint8(float64(p[0])*(1-a) + float64(card.R)*a)
				p[1] = uint8(float64(p[1])*(1-a) + float64(card.G)*a)
				p[2] = uint8(float64(p[2])*(1-a) + float64(card.B)*a)
				p[3] = 255
			}
		}
	}
}

// StripCells is how many thumbnails a strip shows.
const StripCells = 8

// stripRect is the strip's logical rect (x, y, w, h) at logical top y.
func stripRect(y, h float64) [4]float64 { return [4]float64{vidPad, y, vidW - 2*vidPad, h} }

// ThumbSize is the size thumbnails are asked for (window pixels): tall
// enough for a cell, any width (the cell shows the middle of it).
func (c *videoCard) thumbSize(stripH float64) (int, int) {
	h := int(math.Round(stripH * c.Scale))
	return h * 3, h
}

// strip draws the thumbnails in a row (the first at the reading-start side;
// a missing one is a plain cell) inside logical rect r.
func (c *videoCard) strip(r [4]float64, thumbs []*image.RGBA) {
	dc, s := c.dc, c.Scale
	x, y, w, h := c.px(r[0]), c.px(r[1]), r[2]*s, r[3]*s
	dc.DrawRoundedRectangle(x, y, w, h, 6*s)
	if c.Dark {
		dc.SetColor(color.NRGBA{255, 255, 255, 18})
	} else {
		dc.SetColor(color.NRGBA{0, 0, 0, 14})
	}
	dc.Fill()
	n := len(thumbs)
	for i, t := range thumbs {
		if t == nil {
			continue
		}
		cell := i
		if c.RTL {
			cell = n - 1 - i
		}
		x0 := int(math.Round(x + w*float64(cell)/float64(n)))
		x1 := int(math.Round(x + w*float64(cell+1)/float64(n)))
		dst := image.Rect(x0, int(math.Round(y)), x1, int(math.Round(y+h)))
		scaled := c.cover(t, dst.Dx(), dst.Dy())
		xdraw.Copy(c.work, dst.Min, scaled, scaled.Rect, xdraw.Src, nil)
	}
	c.roundCorners(x, y, w, h, 6*s)
}

// cover scales t to fill w×h (cropping the middle), cached per thumbnail.
func (c *videoCard) cover(t *image.RGBA, w, h int) *image.RGBA {
	if m, ok := c.strips[t]; ok && m.Rect.Dx() == w && m.Rect.Dy() == h {
		return m
	}
	tw, th := float64(t.Rect.Dx()), float64(t.Rect.Dy())
	k := math.Max(float64(w)/tw, float64(h)/th)
	sw, sh := float64(w)/k, float64(h)/k
	src := image.Rect(int((tw-sw)/2), int((th-sh)/2), int((tw+sw)/2), int((th+sh)/2)).Add(t.Rect.Min)
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.ApproxBiLinear.Scale(out, out.Rect, t, src, xdraw.Src, nil)
	c.strips[t] = out
	return out
}

// button draws the accent action button (dimmed when not active) at
// logical y with height h, full inner width.
func (c *videoCard) button(label string, y, h float64, active bool) {
	dc, s := c.dc, c.Scale
	dc.DrawRoundedRectangle(c.px(vidPad), c.px(y), (vidW-2*vidPad)*s, h*s, h*s/2)
	if active {
		dc.SetColor(Accent)
	} else {
		dc.SetColor(color.NRGBA{Accent.R, Accent.G, Accent.B, 110})
	}
	dc.Fill()
	c.textCenter(label, vidW/2, y+h/2, 15, true, color.NRGBA{255, 255, 255, 255})
}

// timeX maps a share of the video (0..1) to a logical x on a track from
// logical x0 with width w; Arabic runs from the right.
func (c *videoCard) timeX(f, x0, w float64) float64 {
	f = math.Max(0, math.Min(1, f))
	if c.RTL {
		return x0 + (1-f)*w
	}
	return x0 + f*w
}

// shareAt maps window pixel x to a share of the video (0..1) on a track
// from logical x0 with width w.
func (c *videoCard) shareAt(px, x0, w float64) float64 {
	f := (px/c.Scale - PanelMargin - x0) / w
	if c.RTL {
		f = 1 - f
	}
	return math.Max(0, math.Min(1, f))
}

// FormatClock shows seconds as minutes:seconds.hundredths ("0:04.18"),
// with hours in front from one hour on ("1:02:03.45"). The hundredths are
// cut, never rounded up, so the time never runs ahead of the frame.
func FormatClock(sec float64) string {
	if sec < 0 || math.IsNaN(sec) {
		sec = 0
	}
	cs := int64(math.Floor(sec*100 + 1e-6))
	h, m, s, c := cs/360000, cs/6000%60, cs/100%60, cs%100
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d.%02d", h, m, s, c)
	}
	return fmt.Sprintf("%d:%02d.%02d", m, s, c)
}
