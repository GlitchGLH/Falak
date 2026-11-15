package ui

import (
	"fmt"
	"math"
	"testing"
)

func newTestSplit(t *testing.T, rtl bool) *SplitDialog {
	d := NewSplitDialog(1.5, msgTestFonts(t), false, rtl, SplitLabels{
		Title: "Split Video", File: "clip.mp4", Add: "Add Split", Split: "Split",
		Count: func(n int) string { return fmt.Sprintf("%d split points", n) },
	})
	d.SetVideo(10, 0.125)
	return d
}

// TestSplitPoints: a point goes only at the playhead, never closer than
// the gap to the ends or another point; points stay sorted, drag only
// between their neighbours, and removing one keeps the selection right.
func TestSplitPoints(t *testing.T) {
	d := newTestSplit(t, false)
	if d.Add() {
		t.Fatal("a point at 0 s (the start) was added")
	}
	for _, p := range []float64{6, 2} {
		d.SetPlayhead(p)
		if !d.Add() {
			t.Fatalf("a point at %v s was refused", p)
		}
	}
	if d.Sel != 0 || len(d.Points) != 2 || d.Points[0] != 2 || d.Points[1] != 6 {
		t.Fatalf("points %v, selected %d", d.Points, d.Sel)
	}
	d.SetPlayhead(2.1)
	if d.CanAdd() || d.Add() {
		t.Error("a point 0.1 s from another was added (gap 0.125)")
	}
	d.SetPlayhead(9.9)
	if d.CanAdd() {
		t.Error("a point 0.1 s from the end can be added")
	}
	if got := d.MovePoint(0, 7); got != 6-0.125 {
		t.Errorf("point 1 dragged past point 2 went to %v, want %v", got, 6-0.125)
	}
	if got := d.MovePoint(1, 20); got != 10-0.125 {
		t.Errorf("point 2 dragged past the end went to %v", got)
	}
	d.Sel = 1
	d.Remove(0)
	if len(d.Points) != 1 || d.Sel != 0 {
		t.Errorf("after removing point 1: %v, selected %d", d.Points, d.Sel)
	}
	d.Remove(0)
	if len(d.Points) != 0 || d.Sel != -1 {
		t.Errorf("after removing the last point: %v, selected %d", d.Points, d.Sel)
	}
}

// TestSplitHitTest: the × above a point, the point's line, the strip, Add
// Split and Split are found where they are drawn, in both directions; in
// Arabic the time runs from the right and Add Split sits on the right.
func TestSplitHitTest(t *testing.T) {
	for _, rtl := range []bool{false, true} {
		d := newTestSplit(t, rtl)
		d.SetPlayhead(2.5)
		d.Add()
		s := d.Scale
		x := d.px(d.X(2.5))
		if k, i := d.HitTest(x, d.px(splitBadgeY)); k != SplitBadge || i != 0 {
			t.Errorf("rtl %v: on the ×: %d,%d", rtl, k, i)
		}
		if k, i := d.HitTest(x+3*s, d.px(splitStripY+splitStripH/2)); k != SplitPoint || i != 0 {
			t.Errorf("rtl %v: on the point's line: %d,%d", rtl, k, i)
		}
		if k, _ := d.HitTest(d.px(d.X(7)), d.px(splitStripY+splitStripH/2)); k != SplitStrip {
			t.Errorf("rtl %v: on the strip: %d", rtl, k)
		}
		if tm := d.TimeAt(d.px(d.X(7))); math.Abs(tm-7) > 0.01 {
			t.Errorf("rtl %v: TimeAt(X(7)) = %v", rtl, tm)
		}
		a := d.addRect()
		if k, _ := d.HitTest(d.px(a[0]+a[2]/2), d.px(a[1]+a[3]/2)); k != SplitAdd {
			t.Errorf("rtl %v: on Add Split: %d", rtl, k)
		}
		if k, _ := d.HitTest(d.px(vidW/2), d.px(splitBtnY+splitBtnH/2)); k != SplitButton {
			t.Errorf("rtl %v: on Split: %d", rtl, k)
		}
		early, late := d.X(1), d.X(9)
		if !rtl && early >= late || rtl && early <= late {
			t.Errorf("rtl %v: 1 s at x %v, 9 s at x %v", rtl, early, late)
		}
		if mid := vidW / 2; !rtl && a[0] >= mid || rtl && a[0] < mid {
			t.Errorf("rtl %v: Add Split at x %v", rtl, a[0])
		}
		_ = d.Render()
	}
}
