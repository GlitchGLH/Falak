package ui

import (
	"image"
	"image/color"
	"math"
	"sort"

	"github.com/fogleman/gg"
)

// The Split Video window: the preview shows the frame under the playhead,
// the thumbnail strip has the playhead (white) and the split points
// (accent lines, each with a round × above it that removes it). "Add
// Split" adds a point at the playhead; points can be dragged; the counter
// shows how many there are, and Split makes the parts.

const (
	splitH      = 566.0 // card height
	splitBadgeY = 352.0 // centre of the × badges
	splitBadgeR = 9.0
	splitStripY = 372.0
	splitStripH = 50.0
	splitRowY   = 438.0 // "Add Split" and the counter
	splitRowH   = 36.0
	splitAddW   = 220.0
	splitErrY   = 490.0
	splitBtnY   = 506.0
	splitBtnH   = 36.0
	splitGrab   = 6.0 // how far from a point's line a press still takes it
)

// SplitLabels holds the window's texts in the current language; Count
// gives the counter line for n points.
type SplitLabels struct {
	Title, File, Add, Split, Preparing, PreviewFailed string
	Count                                             func(n int) string
}

// Hit-test results of the Split window (with an index for a point).
const (
	SplitNone = iota
	SplitClose
	SplitBadge // the × above point idx
	SplitPoint // the line of point idx
	SplitStrip // the strip elsewhere: moves the playhead
	SplitAdd
	SplitButton
)

// SplitDialog is the Split Video window's picture and state.
type SplitDialog struct {
	videoCard
	Labels SplitLabels

	Duration float64 // seconds (0 until known)
	Gap      float64 // the shortest part: points keep this far from the ends and each other
	Playhead float64
	Points   []float64 // sorted
	Sel      int       // the point last added or dragged (-1: none), drawn filled
	Frame    *image.RGBA
	Thumbs   []*image.RGBA
	Failed   bool
	Progress float64 // FFmpeg download share, -1 when none
	Error    string  // red line above the button
}

func NewSplitDialog(scale float64, fonts *Fonts, dark, rtl bool, labels SplitLabels) *SplitDialog {
	return &SplitDialog{videoCard: newVideoCard(scale, fonts, dark, rtl, splitH), Labels: labels,
		Thumbs: make([]*image.RGBA, StripCells), Progress: -1, Sel: -1}
}

// Ready reports whether the video is known.
func (d *SplitDialog) Ready() bool { return d.Duration > 0 && !d.Failed }

// SetVideo sets the length and the shortest part.
func (d *SplitDialog) SetVideo(sec, gap float64) { d.Duration, d.Gap = sec, gap }

// ThumbSize is the size thumbnails are asked for (window pixels).
func (d *SplitDialog) ThumbSize() (int, int) { return d.thumbSize(splitStripH) }

func (d *SplitDialog) stripR() [4]float64 { return stripRect(splitStripY, splitStripH) }

// X is the logical x of time t on the strip (Arabic runs from the right).
func (d *SplitDialog) X(t float64) float64 {
	r := d.stripR()
	if d.Duration <= 0 {
		return r[0]
	}
	return d.timeX(t/d.Duration, r[0], r[2])
}

// TimeAt is the time under window pixel x on the strip.
func (d *SplitDialog) TimeAt(x float64) float64 {
	r := d.stripR()
	return d.shareAt(x, r[0], r[2]) * d.Duration
}

// fits reports whether a point at t keeps the gap from the ends and from
// every point but skip.
func (d *SplitDialog) fits(t float64, skip int) bool {
	if t < d.Gap-1e-9 || t > d.Duration-d.Gap+1e-9 {
		return false
	}
	for i, p := range d.Points {
		if i != skip && math.Abs(p-t) < d.Gap-1e-9 {
			return false
		}
	}
	return true
}

// CanAdd reports whether a point can go at the playhead.
func (d *SplitDialog) CanAdd() bool { return d.Ready() && d.fits(d.Playhead, -1) }

// Add puts a point at the playhead; false when it can't go there.
func (d *SplitDialog) Add() bool {
	if !d.CanAdd() {
		return false
	}
	d.Points = append(d.Points, d.Playhead)
	sort.Float64s(d.Points)
	d.Sel = sort.SearchFloat64s(d.Points, d.Playhead)
	d.Error = ""
	return true
}

// Remove takes point i away.
func (d *SplitDialog) Remove(i int) {
	if i < 0 || i >= len(d.Points) {
		return
	}
	d.Points = append(d.Points[:i], d.Points[i+1:]...)
	switch {
	case d.Sel == i:
		d.Sel = -1
	case d.Sel > i:
		d.Sel--
	}
}

// MovePoint drags point i to t, kept between its neighbours (the gap
// apart) and inside the video, and returns where it went.
func (d *SplitDialog) MovePoint(i int, t float64) float64 {
	lo, hi := d.Gap, d.Duration-d.Gap
	if i > 0 {
		lo = d.Points[i-1] + d.Gap
	}
	if i < len(d.Points)-1 {
		hi = d.Points[i+1] - d.Gap
	}
	if hi < lo { // no room (cannot happen when the points were valid)
		return d.Points[i]
	}
	d.Points[i] = math.Max(lo, math.Min(hi, t))
	d.Sel = i
	return d.Points[i]
}

// SetPlayhead puts the playhead at t (inside the video).
func (d *SplitDialog) SetPlayhead(t float64) { d.Playhead = math.Max(0, math.Min(d.Duration, t)) }

// addRect is the "Add Split" button, on the side reading starts.
func (d *SplitDialog) addRect() [4]float64 {
	return [4]float64{d.mirror(vidPad, splitAddW), splitRowY, splitAddW, splitRowH}
}

// HitTest says what window pixel (x, y) is on, with the point's index for
// SplitBadge and SplitPoint.
func (d *SplitDialog) HitTest(x, y float64) (int, int) {
	if d.overClose(x, y) {
		return SplitClose, -1
	}
	if !d.Ready() {
		return SplitNone, -1
	}
	s := d.Scale
	for i := len(d.Points) - 1; i >= 0; i-- {
		px := d.px(d.X(d.Points[i]))
		if math.Hypot(x-px, y-d.px(splitBadgeY)) <= (splitBadgeR+3)*s {
			return SplitBadge, i
		}
	}
	r := d.stripR()
	if d.in(x, y, r, 4) {
		best, bi := splitGrab*s, -1
		for i, p := range d.Points {
			if dx := math.Abs(x - d.px(d.X(p))); dx <= best {
				best, bi = dx, i
			}
		}
		if bi >= 0 {
			return SplitPoint, bi
		}
		return SplitStrip, -1
	}
	if d.in(x, y, d.addRect(), 0) {
		return SplitAdd, -1
	}
	if d.in(x, y, [4]float64{vidPad, splitBtnY, vidW - 2*vidPad, splitBtnH}, 0) {
		return SplitButton, -1
	}
	return SplitNone, -1
}

// Render draws one frame.
func (d *SplitDialog) Render() *image.RGBA {
	d.begin(d.Labels.Title, d.Labels.File)
	dc, s := d.dc, d.Scale
	status := d.Labels.Preparing
	if d.Failed {
		status = d.Labels.PreviewFailed
	}
	d.preview(d.Frame, status, d.Progress)
	r := d.stripR()
	d.videoCard.strip(r, d.Thumbs)
	if d.Ready() {
		d.clockBadge(FormatClock(d.Playhead))
		white := color.NRGBA{255, 255, 255, 255}
		for i, p := range d.Points {
			x := d.px(d.X(p))
			// the line across the strip
			dc.DrawRoundedRectangle(x-2*s, d.px(r[1]-3), 4*s, (r[3]+6)*s, 2*s)
			dc.SetColor(Accent)
			dc.Fill()
			// the × that removes it (filled for the point last touched)
			by := d.px(splitBadgeY)
			dc.DrawCircle(x, by, splitBadgeR*s)
			if i == d.Sel {
				dc.SetColor(Accent)
			} else {
				dc.SetColor(color.NRGBA{Accent.R, Accent.G, Accent.B, 150})
			}
			dc.Fill()
			k := 3.2 * s
			dc.SetLineCap(gg.LineCapRound)
			dc.SetLineWidth(1.6 * s)
			dc.SetColor(white)
			dc.DrawLine(x-k, by-k, x+k, by+k)
			dc.DrawLine(x+k, by-k, x-k, by+k)
			dc.Stroke()
		}
		// the playhead: white with a dark edge, a little taller than the strip
		x := d.px(d.X(d.Playhead))
		dc.DrawRoundedRectangle(x-2.5*s, d.px(r[1]-5), 5*s, (r[3]+10)*s, 2.5*s)
		dc.SetColor(color.NRGBA{0, 0, 0, 120})
		dc.Fill()
		dc.DrawRoundedRectangle(x-1.5*s, d.px(r[1]-4), 3*s, (r[3]+8)*s, 1.5*s)
		dc.SetColor(white)
		dc.Fill()
	}

	// "Add Split" (quiet, dimmed while no point can go at the playhead)
	// and the counter on the other side
	a := d.addRect()
	dc.DrawRoundedRectangle(d.px(a[0]), d.px(a[1]), a[2]*s, a[3]*s, a[3]*s/2)
	alpha := uint8(255)
	if !d.CanAdd() {
		alpha = 110
	}
	dc.SetColor(color.NRGBA{Accent.R, Accent.G, Accent.B, alpha / 7})
	dc.Fill()
	d.textCenter(d.Labels.Add, a[0]+a[2]/2, a[1]+a[3]/2, 14, true, color.NRGBA{Accent.R, Accent.G, Accent.B, alpha})
	if d.Ready() && d.Labels.Count != nil {
		d.textEnd(d.Labels.Count(len(d.Points)), splitRowY+splitRowH/2, 14, true, d.textCol())
	}
	if d.Error != "" {
		d.textStart(d.Error, splitErrY, 12, false, errorRed)
	}
	d.button(d.Labels.Split, splitBtnY, splitBtnH, d.Ready())
	return d.work
}

// textEnd draws s ending at the side opposite to reading start.
func (c *videoCard) textEnd(s string, y, size float64, bold bool, col color.Color) {
	f := c.Fonts.Face(math.Round(size*c.Scale*4)/4, bold)
	if f == nil || s == "" {
		return
	}
	c.dc.SetFontFace(f)
	c.dc.SetColor(col)
	t := Visual(s)
	tw, _ := c.dc.MeasureString(t)
	x := c.px(vidW-vidPad) - tw
	if c.RTL {
		x = c.px(vidPad)
	}
	c.dc.DrawString(t, x, c.px(y)+capHeightOf(f)/2)
}

// clockBadge shows a time in the preview's bottom corner on the side
// reading starts, white on a dark rounded box.
func (c *videoCard) clockBadge(t string) {
	f := c.Fonts.Face(math.Round(12*c.Scale*4)/4, true)
	if f == nil {
		return
	}
	dc, s := c.dc, c.Scale
	dc.SetFontFace(f)
	tw, _ := dc.MeasureString(t)
	w, h := tw+12*s, 20*s
	x := c.px(vidPad + 8)
	if c.RTL {
		x = c.px(vidW-vidPad-8) - w
	}
	y := c.px(vidPrevY+vidPrevH-8) - h
	dc.DrawRoundedRectangle(x, y, w, h, 6*s)
	dc.SetColor(color.NRGBA{0, 0, 0, 150})
	dc.Fill()
	dc.SetColor(color.NRGBA{255, 255, 255, 255})
	dc.DrawString(t, x+6*s, y+h/2+capHeightOf(f)/2)
}
