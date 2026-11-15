package ui

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func TestFormatClock(t *testing.T) {
	for sec, want := range map[float64]string{
		0:       "0:00.00",
		4.18:    "0:04.18",
		12.15:   "0:12.15",
		5.419:   "0:05.41", // cut, not rounded up
		59.999:  "0:59.99",
		61.5:    "1:01.50",
		3723.45: "1:02:03.45",
		-3:      "0:00.00",
	} {
		if got := FormatClock(sec); got != want {
			t.Errorf("FormatClock(%v) = %q, want %q", sec, got, want)
		}
	}
}

func testTrim(rtl bool) *TrimDialog {
	d := NewTrimDialog(1, LoadFonts(nil, nil), false, rtl, TrimLabels{
		Title: "Trim Video", File: "clip.mp4", Hint: "Drag the ends to trim", Selected: "%s selected",
		Trim: "Trim", Preparing: "Preparing the video…", PreviewFailed: "Couldn't show the preview"})
	d.SetDuration(20, 1.0/30)
	return d
}

// winX is the window pixel of a logical card x at scale 1.
func winX(x float64) float64 { return PanelMargin + x }
func winY(y float64) float64 { return PanelMargin + y }

// TestTrimHandles: the handles sit at the ends, time runs from the right in
// Arabic, and the handles keep their order and the one-frame gap.
func TestTrimHandles(t *testing.T) {
	r := stripRect(trimStripY, trimStripH)
	midY := winY(r[1] + r[3]/2)
	left, right := winX(r[0]), winX(r[0]+r[2])
	en, ar := testTrim(false), testTrim(true)
	if h := en.HitTest(left, midY); h != TrimStartHandle {
		t.Errorf("English: left end is %d, want the start handle", h)
	}
	if h := en.HitTest(right, midY); h != TrimEndHandle {
		t.Errorf("English: right end is %d, want the end handle", h)
	}
	if h := ar.HitTest(right, midY); h != TrimStartHandle {
		t.Errorf("Arabic: right end is %d, want the start handle", h)
	}
	if h := ar.HitTest(left, midY); h != TrimEndHandle {
		t.Errorf("Arabic: left end is %d, want the end handle", h)
	}
	quarter := winX(r[0] + r[2]/4)
	if got := en.TimeAt(quarter); math.Abs(got-5) > 0.01 {
		t.Errorf("English: a quarter along is %.2f s, want 5", got)
	}
	if got := ar.TimeAt(quarter); math.Abs(got-15) > 0.01 {
		t.Errorf("Arabic: a quarter from the left is %.2f s, want 15", got)
	}
	d := testTrim(false)
	d.Move(TrimStartHandle, 8)
	d.Move(TrimEndHandle, 3) // can't pass the start
	if d.Start != 8 || math.Abs(d.End-(8+1.0/30)) > 1e-9 {
		t.Errorf("end handle passed the start: %v..%v", d.Start, d.End)
	}
	d.Move(TrimStartHandle, -4)
	d.Move(TrimEndHandle, 99)
	if d.Start != 0 || d.End != 20 {
		t.Errorf("handles left the video: %v..%v", d.Start, d.End)
	}
	if h := en.HitTest(winX(vidW/2), winY(trimBtnY+trimBtnH/2)); h != TrimButton {
		t.Errorf("button hit %d", h)
	}
	if h := en.HitTest(winX(cdlgClose[0]), winY(cdlgClose[1])); h != TrimClose {
		t.Errorf("close hit %d", h)
	}
	waiting := NewTrimDialog(1, LoadFonts(nil, nil), false, false, TrimLabels{})
	if waiting.HitTest(winX(vidW/2), winY(trimBtnY+trimBtnH/2)) != TrimNone {
		t.Error("the button works before the video is known")
	}
}

// TestTrimRender: the left-out parts are dimmed and the selection frame is
// drawn in the accent colour; Arabic mirrors it.
func TestTrimRender(t *testing.T) {
	for _, rtl := range []bool{false, true} {
		d := testTrim(rtl)
		thumb := image.NewRGBA(image.Rect(0, 0, 90, 50))
		for i := range thumb.Pix {
			thumb.Pix[i] = 230
		}
		for i := range d.Thumbs {
			d.Thumbs[i] = thumb
		}
		d.Move(TrimStartHandle, 5)
		d.Move(TrimEndHandle, 15)
		img := d.Render()
		r := stripRect(trimStripY, trimStripH)
		y := int(winY(r[1] + r[3]/2))
		early := int(winX(r[0] + r[2]*0.1)) // 2 s: left out
		if rtl {
			early = int(winX(r[0] + r[2]*0.9))
		}
		mid := int(winX(r[0] + r[2]*0.4)) // inside the selection either way
		if c := img.RGBAAt(early, y); c.R > 120 {
			t.Errorf("rtl=%v: the part before the start is not dimmed (%v)", rtl, c)
		}
		if c := img.RGBAAt(mid, y); c.R < 200 {
			t.Errorf("rtl=%v: the selection is dimmed (%v)", rtl, c)
		}
		top := int(winY(r[1]))
		if c := img.RGBAAt(mid, top); c != (color.RGBA{Accent.R, Accent.G, Accent.B, 255}) {
			t.Errorf("rtl=%v: no selection frame at the top (%v)", rtl, c)
		}
	}
}
