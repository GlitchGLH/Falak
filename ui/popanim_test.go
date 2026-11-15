package ui

import (
	"image"
	"math"
	"testing"
)

// TestPopCurves: the popups use the wheel's numbers.
func TestPopCurves(t *testing.T) {
	sc, a := PopOpen(0)
	if math.Abs(sc-0.4) > 1e-9 || math.Abs(a-0.5) > 1e-9 {
		t.Errorf("first frame: scale %.3f alpha %.3f, want 0.40 and 0.50", sc, a)
	}
	if _, a := PopOpen(1.0 / 30); a < 0.999 {
		t.Errorf("opacity %.3f by the third frame", a)
	}
	if PopOpenDone(0.1) || !PopOpenDone(0.25) {
		t.Error("the entrance should end between 0.1 and 0.25 s")
	}
	if sc, a := PopClose(0); sc != 1 || a != 1 {
		t.Errorf("close start %.3f %.3f", sc, a)
	}
	if sc, a := PopClose(1); math.Abs(sc-0.92) > 1e-9 || a != 0 {
		t.Errorf("close end %.3f %.3f, want 0.92 and 0", sc, a)
	}
	// Panel (the progress window) pops in on the same curve
	src := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for i := range src.Pix {
		src.Pix[i] = 255
	}
	f := PopFrame(nil, src, 1, 1)
	if f.Pix[0] != 255 || f.Rect != src.Rect {
		t.Error("full frame differs from the picture")
	}
	f = PopFrame(f, src, 0.4, 0.5)
	if f.Pix[0] != 0 { // the corner is outside the 40% picture
		t.Error("corner drawn at 40%")
	}
	c := f.PixOffset(20, 10)
	if v := f.Pix[c+3]; v < 120 || v > 135 {
		t.Errorf("centre alpha %d at 50%%", v)
	}
}
