package ui

import (
	"image"
	"image/color"
	"math"
)

// The Video Snapshots window: a preview, a scrubber, previous/next frame
// buttons, the current and total time, and Save Snapshot, which saves the
// frame shown. The window stays open for more; each save shows "Saved".

const (
	snapH       = 494.0 // card height
	snapSliderY = 352.0 // the track's centre line
	snapRowY    = 390.0 // the buttons' and the time's centre line
	snapBtnR    = 16.0  // previous/next button radius
	snapSavedY  = 421.0
	snapSaveY   = 438.0
	snapSaveH   = 36.0
	snapKnobR   = 8.0
)

// SnapLabels holds the window's texts in the current language.
type SnapLabels struct {
	Title, File, Save, Saved, Preparing, PreviewFailed string
}

// Hit-test results of the Snapshots window.
const (
	SnapNone = iota
	SnapClose
	SnapSlider
	SnapPrev
	SnapNext
	SnapSave
)

// SnapDialog is the Video Snapshots window's picture and state.
type SnapDialog struct {
	videoCard
	Labels SnapLabels

	Duration float64 // seconds (0 until known)
	FPS      float64
	Pos      int // the frame shown (0 .. Frames()-1)
	Frame    *image.RGBA
	Failed   bool
	Progress float64 // FFmpeg download share, -1 when none
	Saved    bool    // show the "Saved" line
	Dragging bool    // the scrubber is being dragged
}

func NewSnapDialog(scale float64, fonts *Fonts, dark, rtl bool, labels SnapLabels) *SnapDialog {
	return &SnapDialog{videoCard: newVideoCard(scale, fonts, dark, rtl, snapH), Labels: labels, Progress: -1}
}

// SetVideo sets the length and frame rate and shows the first frame.
func (d *SnapDialog) SetVideo(duration, fps float64) {
	d.Duration, d.FPS, d.Pos = duration, fps, 0
}

// Ready reports whether the video is known.
func (d *SnapDialog) Ready() bool { return d.Duration > 0 && d.FPS > 0 && !d.Failed }

// Frames is how many frames the video has (at least 1).
func (d *SnapDialog) Frames() int {
	return max(1, int(math.Floor(d.Duration*d.FPS+1e-6)))
}

// Step moves by n frames (kept inside the video) and reports whether the
// position changed.
func (d *SnapDialog) Step(n int) bool {
	p := max(0, min(d.Frames()-1, d.Pos+n))
	changed := p != d.Pos
	d.Pos = p
	return changed
}

// FrameAtX is the frame under window pixel x on the scrubber.
func (d *SnapDialog) FrameAtX(x float64) int {
	f := d.shareAt(x, vidPad, vidW-2*vidPad)
	return int(math.Round(f * float64(d.Frames()-1)))
}

// share is the position as a share of the scrubber (0..1).
func (d *SnapDialog) share() float64 {
	if d.Frames() <= 1 {
		return 0
	}
	return float64(d.Pos) / float64(d.Frames()-1)
}

// TimeText is "current / total" (0:04.18 / 0:12.15); in Arabic the current
// time is the one on the right, where reading starts.
func (d *SnapDialog) TimeText() string {
	cur, total := FormatClock(float64(d.Pos)/d.FPS), FormatClock(d.Duration)
	if d.RTL {
		return total + " / " + cur
	}
	return cur + " / " + total
}

// buttonX is the logical centre x of the previous (prev=true) or next
// button: previous at the reading-start side.
func (d *SnapDialog) buttonX(prev bool) float64 {
	left := vidPad + snapBtnR
	right := vidW - vidPad - snapBtnR
	if prev != d.RTL {
		return left
	}
	return right
}

// HitTest says what window pixel (x, y) is on.
func (d *SnapDialog) HitTest(x, y float64) int {
	if d.overClose(x, y) {
		return SnapClose
	}
	if !d.Ready() {
		return SnapNone
	}
	if d.in(x, y, [4]float64{vidPad, snapSliderY - 12, vidW - 2*vidPad, 24}, 0) {
		return SnapSlider
	}
	for _, prev := range []bool{true, false} {
		if math.Hypot(x-d.px(d.buttonX(prev)), y-d.px(snapRowY)) <= (snapBtnR+3)*d.Scale {
			if prev {
				return SnapPrev
			}
			return SnapNext
		}
	}
	if d.in(x, y, [4]float64{vidPad, snapSaveY, vidW - 2*vidPad, snapSaveH}, 0) {
		return SnapSave
	}
	return SnapNone
}

// Render draws one frame.
func (d *SnapDialog) Render() *image.RGBA {
	d.begin(d.Labels.Title, d.Labels.File)
	dc, s := d.dc, d.Scale
	status := d.Labels.Preparing
	if d.Failed {
		status = d.Labels.PreviewFailed
	}
	d.preview(d.Frame, status, d.Progress)

	// the scrubber: the played part in the accent colour, from the
	// reading-start side
	tx, tw := vidPad, vidW-2*vidPad
	track := color.NRGBA{0, 0, 0, 28}
	if d.Dark {
		track = color.NRGBA{255, 255, 255, 36}
	}
	dc.DrawRoundedRectangle(d.px(tx), d.px(snapSliderY-2), tw*s, 4*s, 2*s)
	dc.SetColor(track)
	dc.Fill()
	if d.Ready() {
		kx := d.timeX(d.share(), tx, tw)
		lo, hi := tx, kx
		if d.RTL {
			lo, hi = kx, tx+tw
		}
		if hi > lo {
			dc.DrawRoundedRectangle(d.px(lo), d.px(snapSliderY-2), (hi-lo)*s, 4*s, 2*s)
			dc.SetColor(Accent)
			dc.Fill()
		}
		dc.DrawCircle(d.px(kx), d.px(snapSliderY), snapKnobR*s)
		dc.SetColor(color.NRGBA{255, 255, 255, 255})
		dc.Fill()
		dc.DrawCircle(d.px(kx), d.px(snapSliderY), snapKnobR*s)
		dc.SetLineWidth(2 * s)
		dc.SetColor(Accent)
		dc.Stroke()
	}

	// previous / next frame, the time between them
	for _, prev := range []bool{true, false} {
		d.stepButton(d.buttonX(prev), prev)
	}
	if d.Ready() {
		d.textCenter(d.TimeText(), vidW/2, snapRowY, 14, true, d.textCol())
	}
	if d.Saved {
		d.textCenter(d.Labels.Saved, vidW/2, snapSavedY, 13, true, Accent)
	}
	d.button(d.Labels.Save, snapSaveY, snapSaveH, d.Ready())
	return d.work
}

// stepButton draws a round button with a bar and a triangle: |◀ for the
// previous frame, ▶| for the next; mirrored in Arabic, where time runs from
// the right.
func (d *SnapDialog) stepButton(cx float64, prev bool) {
	dc, s := d.dc, d.Scale
	x, y := d.px(cx), d.px(snapRowY)
	dc.DrawCircle(x, y, snapBtnR*s)
	if d.Dark {
		dc.SetColor(color.NRGBA{255, 255, 255, 20})
	} else {
		dc.SetColor(color.NRGBA{0, 0, 0, 12})
	}
	dc.Fill()
	col := d.grey()
	if !d.Ready() {
		col.A = 90
	}
	dir := 1.0 // the triangle points right (next, in English)
	if prev != d.RTL {
		dir = -1
	}
	h := 5.5 * s
	dc.MoveTo(x+dir*4*s, y)
	dc.LineTo(x-dir*3*s, y-h)
	dc.LineTo(x-dir*3*s, y+h)
	dc.ClosePath()
	dc.SetColor(col)
	dc.Fill()
	dc.DrawRoundedRectangle(x+dir*5*s-1*s, y-h, 2*s, 2*h, 1*s)
	dc.Fill()
}
