package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/fogleman/gg"
)

// The Trim Video window: the preview shows the frame under the handle being
// dragged, the thumbnail strip has two handles, the parts left out are
// dimmed, the selected length is shown, and Trim makes the piece.

const (
	trimH      = 526.0 // card height
	trimHintY  = 348.0
	trimStripY = 364.0
	trimStripH = 50.0
	trimSelY   = 434.0
	trimErrY   = 453.0
	trimBtnY   = 470.0
	trimBtnH   = 36.0
	trimHandle = 12.0 // handle width
)

// TrimLabels holds the window's texts in the current language.
type TrimLabels struct {
	Title, File, Hint, Selected, Trim, Preparing, PreviewFailed string
}

// Hit-test results of the Trim window.
const (
	TrimNone = iota
	TrimClose
	TrimStartHandle
	TrimEndHandle
	TrimButton
	TrimStrip // the strip between or outside the handles
)

// TrimDialog is the Trim Video window's picture and state.
type TrimDialog struct {
	videoCard
	Labels TrimLabels

	Duration   float64 // seconds (0 until known)
	MinGap     float64 // the handles never come closer (one frame)
	Start, End float64 // the selection, seconds
	Frame      *image.RGBA
	Thumbs     []*image.RGBA // StripCells entries, nil until loaded
	Failed     bool          // the preview couldn't be shown
	Progress   float64       // FFmpeg download share, -1 when none
	Error      string        // red line above the button
	Active     int           // the handle being dragged (TrimStartHandle/TrimEndHandle), else 0
}

func NewTrimDialog(scale float64, fonts *Fonts, dark, rtl bool, labels TrimLabels) *TrimDialog {
	return &TrimDialog{videoCard: newVideoCard(scale, fonts, dark, rtl, trimH), Labels: labels,
		Thumbs: make([]*image.RGBA, StripCells), Progress: -1}
}

// Ready reports whether the video is known and can be trimmed.
func (d *TrimDialog) Ready() bool { return d.Duration > 0 && !d.Failed }

// SetDuration sets the length and selects the whole video.
func (d *TrimDialog) SetDuration(sec, minGap float64) {
	d.Duration, d.MinGap = sec, math.Min(minGap, sec)
	d.Start, d.End = 0, sec
}

// ThumbSize is the size thumbnails are asked for (window pixels).
func (d *TrimDialog) ThumbSize() (int, int) { return d.thumbSize(trimStripH) }

func (d *TrimDialog) strip() [4]float64 { return stripRect(trimStripY, trimStripH) }

// handleX is the logical centre x of the handle at time t.
func (d *TrimDialog) handleX(t float64) float64 {
	r := d.strip()
	if d.Duration <= 0 {
		return r[0]
	}
	return d.timeX(t/d.Duration, r[0], r[2])
}

// TimeAt is the time under window pixel x on the strip.
func (d *TrimDialog) TimeAt(x float64) float64 {
	r := d.strip()
	return d.shareAt(x, r[0], r[2]) * d.Duration
}

// Move puts the dragged handle at time t (kept inside the video and at
// least MinGap from the other handle) and returns where it went.
func (d *TrimDialog) Move(handle int, t float64) float64 {
	switch handle {
	case TrimStartHandle:
		d.Start = math.Max(0, math.Min(t, d.End-d.MinGap))
		return d.Start
	case TrimEndHandle:
		d.End = math.Min(d.Duration, math.Max(t, d.Start+d.MinGap))
		return d.End
	}
	return t
}

// Nearest is the handle closer to window pixel x.
func (d *TrimDialog) Nearest(x float64) int {
	t := d.TimeAt(x)
	if math.Abs(t-d.Start) <= math.Abs(t-d.End) {
		return TrimStartHandle
	}
	return TrimEndHandle
}

// HitTest says what window pixel (x, y) is on.
func (d *TrimDialog) HitTest(x, y float64) int {
	if d.overClose(x, y) {
		return TrimClose
	}
	if d.Ready() {
		r := d.strip()
		for _, h := range []int{TrimEndHandle, TrimStartHandle} {
			t := d.Start
			if h == TrimEndHandle {
				t = d.End
			}
			hx := d.handleX(t)
			if d.in(x, y, [4]float64{hx - trimHandle/2, r[1] - 3, trimHandle, r[3] + 6}, 5) {
				return h
			}
		}
		if d.in(x, y, r, 0) {
			return TrimStrip
		}
		if d.in(x, y, [4]float64{vidPad, trimBtnY, vidW - 2*vidPad, trimBtnH}, 0) {
			return TrimButton
		}
	}
	return TrimNone
}

// Render draws one frame.
func (d *TrimDialog) Render() *image.RGBA {
	d.begin(d.Labels.Title, d.Labels.File)
	dc, s := d.dc, d.Scale
	status := d.Labels.Preparing
	if d.Failed {
		status = d.Labels.PreviewFailed
	}
	d.preview(d.Frame, status, d.Progress)
	d.textStart(d.Labels.Hint, trimHintY, 12, false, d.grey())

	r := d.strip()
	d.videoCard.strip(r, d.Thumbs)
	if d.Ready() {
		sx, ex := d.handleX(d.Start), d.handleX(d.End)
		lo, hi := math.Min(sx, ex), math.Max(sx, ex)
		// the parts left out are dimmed
		dim := color.NRGBA{0, 0, 0, 150}
		if lo > r[0] {
			dc.DrawRectangle(d.px(r[0]), d.px(r[1]), (lo-r[0])*s, r[3]*s)
			dc.SetColor(dim)
			dc.Fill()
		}
		if hi < r[0]+r[2] {
			dc.DrawRectangle(d.px(hi), d.px(r[1]), (r[0]+r[2]-hi)*s, r[3]*s)
			dc.SetColor(dim)
			dc.Fill()
		}
		d.roundCorners(d.px(r[0]), d.px(r[1]), r[2]*s, r[3]*s, 6*s)
		// the selection frame: accent lines along the top and bottom
		dc.SetColor(Accent)
		dc.DrawRectangle(d.px(lo), d.px(r[1]-1.5), (hi-lo)*s, 3*s)
		dc.Fill()
		dc.DrawRectangle(d.px(lo), d.px(r[1]+r[3]-1.5), (hi-lo)*s, 3*s)
		dc.Fill()
		for _, hx := range []float64{sx, ex} {
			d.handle(dc, hx, r)
		}
		d.textStart(fmt.Sprintf(d.Labels.Selected, FormatClock(d.End-d.Start)), trimSelY, 14, true, d.textCol())
	}
	if d.Error != "" {
		d.textStart(d.Error, trimErrY, 12, false, errorRed)
	}
	d.button(d.Labels.Trim, trimBtnY, trimBtnH, d.Ready())
	return d.work
}

// handle draws one handle centred on logical x: an accent bar with a
// white grip line, a little taller than the strip.
func (d *TrimDialog) handle(dc *gg.Context, hx float64, r [4]float64) {
	s := d.Scale
	dc.DrawRoundedRectangle(d.px(hx-trimHandle/2), d.px(r[1]-3), trimHandle*s, (r[3]+6)*s, 4*s)
	dc.SetColor(Accent)
	dc.Fill()
	dc.SetLineCap(gg.LineCapRound)
	dc.SetLineWidth(2 * s)
	dc.SetColor(color.NRGBA{255, 255, 255, 230})
	dc.DrawLine(d.px(hx), d.px(r[1]+r[3]/2-8), d.px(hx), d.px(r[1]+r[3]/2+8))
	dc.Stroke()
}
