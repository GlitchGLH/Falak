package ui

import "testing"

func newTestSpeed(t *testing.T, rtl bool) *SpeedDialog {
	d := NewSpeedDialog(1.5, msgTestFonts(t), false, rtl, SpeedLabels{
		Title: "Change Video Speed", File: "clip.mp4", Speed: "Speed", Change: "Change Speed",
		Smooth: "Smooth slow motion", SmoothHint: "Takes longer to process",
		Chips:    []string{"0.25×", "0.5×", "0.75×", "1.25×", "1.5×", "2×", "3×", "4×"},
		SmoothOK: []bool{true, true, true, false, false, false, false, false},
	}, 5)
	d.Duration = 10
	return d
}

// TestSpeedChips: the eight speed buttons fit the card in one row without
// overlapping, the slowest on the side reading starts (mirrored in
// Arabic), each found by HitTest where it is drawn; Step stays in the row.
func TestSpeedChips(t *testing.T) {
	for _, rtl := range []bool{false, true} {
		d := newTestSpeed(t, rtl)
		prev := -1.0
		for i := range d.Labels.Chips {
			r := d.ChipRect(i)
			if r[0] < vidPad-0.01 || r[0]+r[2] > vidW-vidPad+0.01 {
				t.Errorf("rtl %v: button %d outside the card: %v", rtl, i, r)
			}
			if i > 0 && (!rtl && r[0] <= prev || rtl && r[0] >= prev) {
				t.Errorf("rtl %v: button %d at x %v after %v", rtl, i, r[0], prev)
			}
			prev = r[0]
			if k, j := d.HitTest(d.px(r[0]+r[2]/2), d.px(r[1]+r[3]/2)); k != SpeedChip || j != i {
				t.Errorf("rtl %v: HitTest on button %d = %d,%d", rtl, i, k, j)
			}
			if e, a := (newTestSpeed(t, false)).ChipRect(i), d.ChipRect(i); rtl && (a[0]-(vidW-e[0]-e[2]) > 0.01 || a[0]-(vidW-e[0]-e[2]) < -0.01) {
				t.Errorf("button %d: Arabic %v is not English %v mirrored", i, a, e)
			}
		}
		if k, _ := d.HitTest(d.px(vidW/2), d.px(speedBtnY+speedBtnH/2)); k != SpeedButton {
			t.Errorf("rtl %v: HitTest on Change Speed = %d", rtl, k)
		}
		if !d.Step(2) || d.Sel != 7 || d.Step(1) || d.Sel != 7 {
			t.Errorf("rtl %v: Step past the end: %d", rtl, d.Sel)
		}
		if !d.Step(-10) || d.Sel != 0 {
			t.Errorf("rtl %v: Step past the start: %d", rtl, d.Sel)
		}
		_ = d.Render()
	}
}

// TestSpeedSmooth: the "Smooth slow motion" switch works only at the slow
// speeds (shown disabled at the others), is off at start, sits on the side
// reading ends (mirrored in Arabic) and is drawn in the accent colour when
// on; the choice is kept but counts only at a slow speed.
func TestSpeedSmooth(t *testing.T) {
	for _, rtl := range []bool{false, true} {
		d := newTestSpeed(t, rtl)
		row := func() int {
			k, _ := d.HitTest(d.px(vidW/2), d.px(speedRowY+speedRowH/2))
			return k
		}
		if d.Smooth || d.SmoothOn() || d.SmoothEnabled() || row() == SpeedSmooth {
			t.Errorf("rtl %v: at 2× the switch works or is on", rtl)
		}
		d.Sel = 1 // 0.5×
		if !d.SmoothEnabled() || row() != SpeedSmooth {
			t.Errorf("rtl %v: at 0.5× the switch doesn't work", rtl)
		}
		d.Smooth = true
		r := d.switchRect()
		if mid := vidW / 2; !rtl && r[0] < mid || rtl && r[0] > mid {
			t.Errorf("rtl %v: switch at x %v", rtl, r[0])
		}
		img := d.Render()
		on := img.RGBAAt(int(d.px(r[0]+r[2]/2)), int(d.px(r[1]+2)))
		if on.R != Accent.R || on.G != Accent.G || on.B != Accent.B {
			t.Errorf("rtl %v: switch on drawn %v, want the accent", rtl, on)
		}
		d.Sel = 3 // 1.25×: kept, but off and disabled
		if !d.Smooth || d.SmoothOn() || row() == SpeedSmooth {
			t.Errorf("rtl %v: at 1.25× smooth counts or can be clicked", rtl)
		}
		img = d.Render()
		if off := img.RGBAAt(int(d.px(r[0]+r[2]/2)), int(d.px(r[1]+2))); off == on {
			t.Errorf("rtl %v: the disabled switch looks on", rtl)
		}
		if k, _ := d.HitTest(d.px(vidW/2), d.px(speedBtnY+speedBtnH/2)); k != SpeedButton {
			t.Errorf("rtl %v: the button moved under the row: %d", rtl, k)
		}
	}
}
