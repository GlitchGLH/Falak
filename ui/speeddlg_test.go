package ui

import "testing"

func newTestSpeed(t *testing.T, rtl bool) *SpeedDialog {
	d := NewSpeedDialog(1.5, msgTestFonts(t), false, rtl, SpeedLabels{
		Title: "Change Video Speed", File: "clip.mp4", Speed: "Speed", Change: "Change Speed",
		Chips: []string{"0.25×", "0.5×", "0.75×", "1.25×", "1.5×", "2×", "3×", "4×"},
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
