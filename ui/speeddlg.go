package ui

import (
	"image"
	"image/color"
)

// The Change Video Speed window: a silent preview playing at the chosen
// speed, a row of speed buttons (0.25× … 4×; the slowest on the side
// reading starts), the "Smooth slow motion" switch (slow speeds only) and
// Change Speed.

const (
	speedH      = 536.0 // card height
	speedLabelY = 350.0
	speedChipY  = 368.0
	speedChipH  = 36.0
	speedChipG  = 6.0   // gap between the speed buttons
	speedRowY   = 416.0 // the "Smooth slow motion" row
	speedRowH   = 44.0
	speedSwW    = 40.0 // its switch, as in the tray menu
	speedSwH    = 24.0
	speedBtnY   = 476.0
	speedBtnH   = 36.0
)

// SpeedLabels holds the window's texts in the current language; Chips are
// the speeds as shown ("0.5×").
type SpeedLabels struct {
	Title, File, Speed, Change, Preparing, PreviewFailed string
	Smooth, SmoothHint                                   string
	Chips                                                []string
	SmoothOK                                             []bool // per chip: the switch works at that speed
}

// Hit-test results of the Speed window (with the speed's index for SpeedChip).
const (
	SpeedNone = iota
	SpeedClose
	SpeedChip
	SpeedSmooth
	SpeedButton
)

// SpeedDialog is the Change Video Speed window's picture and state.
type SpeedDialog struct {
	videoCard
	Labels SpeedLabels

	Duration float64 // seconds (0 until known)
	Sel      int     // the chosen speed (index into Labels.Chips)
	Smooth   bool    // the user's choice; it counts only where SmoothEnabled
	Frame    *image.RGBA
	Failed   bool
	Progress float64 // FFmpeg download share, -1 when none
}

func NewSpeedDialog(scale float64, fonts *Fonts, dark, rtl bool, labels SpeedLabels, sel int) *SpeedDialog {
	return &SpeedDialog{videoCard: newVideoCard(scale, fonts, dark, rtl, speedH), Labels: labels, Sel: sel, Progress: -1}
}

// Ready reports whether the video is known.
func (d *SpeedDialog) Ready() bool { return d.Duration > 0 && !d.Failed }

// SmoothEnabled reports whether the switch works at the chosen speed (it
// is shown disabled, never hidden, for the others).
func (d *SpeedDialog) SmoothEnabled() bool {
	return d.Ready() && d.Sel >= 0 && d.Sel < len(d.Labels.SmoothOK) && d.Labels.SmoothOK[d.Sel]
}

// SmoothOn reports whether the job makes smooth slow motion.
func (d *SpeedDialog) SmoothOn() bool { return d.Smooth && d.SmoothEnabled() }

// switchRect is the switch in logical card coordinates (on the side
// reading ends, like the tray menu's switches).
func (d *SpeedDialog) switchRect() [4]float64 {
	return [4]float64{d.mirror(vidW-vidPad-speedSwW, speedSwW), speedRowY + (speedRowH-speedSwH)/2, speedSwW, speedSwH}
}

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
	if d.SmoothEnabled() && d.in(x, y, [4]float64{vidPad, speedRowY, vidW - 2*vidPad, speedRowH}, 0) {
		return SpeedSmooth, -1
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
	d.smoothRow()
	d.button(d.Labels.Change, speedBtnY, speedBtnH, d.Ready())
	return d.work
}

// smoothRow draws "Smooth slow motion", its hint under it and the switch
// (dimmed, and off, at a speed it doesn't work at).
func (d *SpeedDialog) smoothRow() {
	dc, s := d.dc, d.Scale
	on, enabled := d.SmoothOn(), d.SmoothEnabled()
	title, hint := d.textCol(), d.grey()
	if !enabled {
		title.A, hint.A = 110, 110
	}
	d.textStart(d.Labels.Smooth, speedRowY+12, 13, true, title)
	d.textStart(d.Labels.SmoothHint, speedRowY+32, 12, false, hint)
	r := d.switchRect()
	x, y, w, h := d.px(r[0]), d.px(r[1]), r[2]*s, r[3]*s
	dc.DrawRoundedRectangle(x, y, w, h, h/2)
	switch {
	case on:
		dc.SetColor(Accent)
	case d.Dark:
		dc.SetColor(color.NRGBA{82, 82, 88, 255})
	default:
		dc.SetColor(color.NRGBA{212, 212, 217, 255})
	}
	dc.Fill()
	kr := h/2 - 2.5*s
	kx := x + h/2 // the knob sits on the start side when off
	if on != d.RTL {
		kx = x + w - h/2
	}
	dc.DrawCircle(kx, y+h/2+0.6*s, kr)
	dc.SetColor(color.NRGBA{0, 0, 0, 40})
	dc.Fill()
	dc.DrawCircle(kx, y+h/2, kr)
	dc.SetColor(color.NRGBA{255, 255, 255, 255})
	dc.Fill()
	if !enabled { // dim the whole switch with the card's colour
		dc.DrawRoundedRectangle(x-1, y-1, w+2, h+2, h/2+1)
		if d.Dark {
			dc.SetColor(color.NRGBA{40, 40, 44, 130})
		} else {
			dc.SetColor(color.NRGBA{255, 255, 255, 130})
		}
		dc.Fill()
	}
}
