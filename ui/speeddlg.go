package ui

import (
	"image"
	"image/color"
)

// The Change Video Speed window: a silent preview playing at the chosen
// speed, a row of speed buttons (0.25× … 4×; the slowest on the side
// reading starts) and Change Speed.

const (
	speedH      = 488.0 // card height
	speedLabelY = 350.0
	speedChipY  = 368.0
	speedChipH  = 36.0
	speedChipG  = 6.0 // gap between the speed buttons
	speedBtnY   = 428.0
	speedBtnH   = 36.0
)

// SpeedLabels holds the window's texts in the current language; Chips are
// the speeds as shown ("0.5×").
type SpeedLabels struct {
	Title, File, Speed, Change, Preparing, PreviewFailed string
	Chips                                                []string
}

// Hit-test results of the Speed window (with the speed's index for SpeedChip).
const (
	SpeedNone = iota
	SpeedClose
	SpeedChip
	SpeedButton
)

// SpeedDialog is the Change Video Speed window's picture and state.
type SpeedDialog struct {
	videoCard
	Labels SpeedLabels

	Duration float64 // seconds (0 until known)
	Sel      int     // the chosen speed (index into Labels.Chips)
	Frame    *image.RGBA
	Failed   bool
	Progress float64 // FFmpeg download share, -1 when none
}

func NewSpeedDialog(scale float64, fonts *Fonts, dark, rtl bool, labels SpeedLabels, sel int) *SpeedDialog {
	return &SpeedDialog{videoCard: newVideoCard(scale, fonts, dark, rtl, speedH), Labels: labels, Sel: sel, Progress: -1}
}

// Ready reports whether the video is known.
func (d *SpeedDialog) Ready() bool { return d.Duration > 0 && !d.Failed }

// ChipRect is speed button i in logical card coordinates (x, y, w, h),
// mirrored in Arabic.
func (d *SpeedDialog) ChipRect(i int) [4]float64 {
	n := float64(len(d.Labels.Chips))
	w := (vidW - 2*vidPad - (n-1)*speedChipG) / n
	return [4]float64{d.mirror(vidPad+float64(i)*(w+speedChipG), w), speedChipY, w, speedChipH}
}

// Step moves the choice n buttons towards the end of the row (negative:
// back), staying inside it; it reports whether the choice changed.
func (d *SpeedDialog) Step(n int) bool {
	s := max(0, min(len(d.Labels.Chips)-1, d.Sel+n))
	if s == d.Sel {
		return false
	}
	d.Sel = s
	return true
}

// HitTest says what window pixel (x, y) is on, with the speed's index for
// SpeedChip.
func (d *SpeedDialog) HitTest(x, y float64) (int, int) {
	if d.overClose(x, y) {
		return SpeedClose, -1
	}
	if !d.Ready() {
		return SpeedNone, -1
	}
	for i := range d.Labels.Chips {
		if d.in(x, y, d.ChipRect(i), speedChipG/2) {
			return SpeedChip, i
		}
	}
	if d.in(x, y, [4]float64{vidPad, speedBtnY, vidW - 2*vidPad, speedBtnH}, 0) {
		return SpeedButton, -1
	}
	return SpeedNone, -1
}

// Render draws one frame.
func (d *SpeedDialog) Render() *image.RGBA {
	d.begin(d.Labels.Title, d.Labels.File)
	dc, s := d.dc, d.Scale
	status := d.Labels.Preparing
	if d.Failed {
		status = d.Labels.PreviewFailed
	}
	d.preview(d.Frame, status, d.Progress)
	d.textStart(d.Labels.Speed, speedLabelY, 13, true, d.textCol())
	for i, label := range d.Labels.Chips {
		r := d.ChipRect(i)
		dc.DrawRoundedRectangle(d.px(r[0]), d.px(r[1]), r[2]*s, r[3]*s, r[3]*s/2)
		col := d.textCol()
		switch {
		case i == d.Sel:
			dc.SetColor(Accent)
			col = color.NRGBA{255, 255, 255, 255}
		case d.Dark:
			dc.SetColor(color.NRGBA{255, 255, 255, 22})
		default:
			dc.SetColor(color.NRGBA{0, 0, 0, 12})
		}
		if !d.Ready() {
			col.A = 110
		}
		dc.Fill()
		d.textCenter(label, r[0]+r[2]/2, r[1]+r[3]/2, 13, true, col)
	}
	d.button(d.Labels.Change, speedBtnY, speedBtnH, d.Ready())
	return d.work
}
