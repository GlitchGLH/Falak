package ui

import (
	"image"
	"math"
)

// Every popup window opens and closes with the wheel's own curves (see
// Wheel.Render): it pops open from 40% of its size on an exponential curve
// (time constant 35 ms), half opaque on the first frame and fully opaque by
// the third; it closes by fading out while shrinking 8%, ease-out, in
// 160 ms (80 ms for the wheel's quick close).
const (
	PopCloseSecs     = 0.16
	PopCloseFastSecs = 0.08
)

// PopOpen is the entrance's scale and opacity t seconds after it starts.
func PopOpen(t float64) (sc, alpha float64) {
	alpha = math.Min(1, 0.5+t*21)
	sc = 1 - 0.60*math.Exp(-t/0.035)
	if t > 0.3 {
		sc = 1
	}
	return sc, alpha
}

// PopOpenDone reports whether the entrance has reached the final picture.
func PopOpenDone(t float64) bool {
	sc, a := PopOpen(t)
	return a >= 0.999 && sc >= 0.9995
}

// PopClose is the closing's scale and opacity at share (0..1) of its time.
func PopClose(share float64) (sc, alpha float64) {
	e := easeOutCubic(share)
	return 1 - 0.08*e, 1 - e
}

// PopFrame draws src scaled by sc about its centre, at opacity alpha, into
// dst (reused when it has src's size) and returns it.
func PopFrame(dst, src *image.RGBA, sc, alpha float64) *image.RGBA {
	if dst == nil || dst.Rect != src.Rect {
		dst = image.NewRGBA(src.Rect)
	} else {
		clear(dst.Pix)
	}
	drawLayer(dst, src, float64(src.Rect.Dx())/2, float64(src.Rect.Dy())/2, sc, 0, alpha, true)
	return dst
}
