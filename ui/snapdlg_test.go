package ui

import (
	"testing"
)

func testSnap(rtl bool) *SnapDialog {
	d := NewSnapDialog(1, LoadFonts(nil, nil), false, rtl, SnapLabels{
		Title: "Video Snapshots", File: "clip.mp4", Save: "Save Snapshot", Saved: "Saved",
		Preparing: "Preparing the video…", PreviewFailed: "Couldn't show the preview"})
	d.SetVideo(12.15, 30)
	return d
}

// TestSnapTime: "current / total", with the current time where reading
// starts (on the right in Arabic).
func TestSnapTime(t *testing.T) {
	en, ar := testSnap(false), testSnap(true)
	en.Pos, ar.Pos = 125, 125 // 4.1666 s
	if got := en.TimeText(); got != "0:04.16 / 0:12.15" {
		t.Errorf("English: %q", got)
	}
	if got := ar.TimeText(); got != "0:12.15 / 0:04.16" {
		t.Errorf("Arabic: %q (the current time must be on the right)", got)
	}
}

// TestSnapSteps: frames stay inside the video, the scrubber runs from the
// right in Arabic, and the previous-frame button is at the reading-start
// side.
func TestSnapSteps(t *testing.T) {
	d := testSnap(false)
	if d.Frames() != 364 {
		t.Fatalf("frames %d, want 364", d.Frames())
	}
	if d.Step(-1) || d.Pos != 0 {
		t.Error("stepped before the first frame")
	}
	d.Pos = d.Frames() - 1
	if d.Step(1) || d.Pos != d.Frames()-1 {
		t.Error("stepped past the last frame")
	}
	left, right := winX(vidPad), winX(vidW-vidPad)
	if d.FrameAtX(left) != 0 || d.FrameAtX(right) != d.Frames()-1 {
		t.Errorf("English scrubber: left %d, right %d", d.FrameAtX(left), d.FrameAtX(right))
	}
	ar := testSnap(true)
	if ar.FrameAtX(right) != 0 || ar.FrameAtX(left) != ar.Frames()-1 {
		t.Errorf("Arabic scrubber: right %d, left %d", ar.FrameAtX(right), ar.FrameAtX(left))
	}
	rowY := winY(snapRowY)
	if h := d.HitTest(winX(vidPad+snapBtnR), rowY); h != SnapPrev {
		t.Errorf("English: left button is %d, want previous", h)
	}
	if h := ar.HitTest(winX(vidPad+snapBtnR), rowY); h != SnapNext {
		t.Errorf("Arabic: left button is %d, want next", h)
	}
	if h := d.HitTest(winX(vidW/2), winY(snapSliderY)); h != SnapSlider {
		t.Errorf("scrubber hit %d", h)
	}
	if h := d.HitTest(winX(vidW/2), winY(snapSaveY+snapSaveH/2)); h != SnapSave {
		t.Errorf("save hit %d", h)
	}
	waiting := NewSnapDialog(1, LoadFonts(nil, nil), false, false, SnapLabels{})
	if waiting.HitTest(winX(vidW/2), winY(snapSaveY+snapSaveH/2)) != SnapNone {
		t.Error("Save works before the video is known")
	}
	for _, rtl := range []bool{false, true} {
		x := testSnap(rtl)
		x.Saved = true
		if img := x.Render(); img.Rect.Dx() != x.W {
			t.Fatal("render size")
		}
	}
}
