package ui

import (
	"image"
	"math"
	"testing"
	"time"
)

func newTestWheel(dark bool, items []Item) *Wheel {
	s := 2.0
	size := int(LogicalSize * s)
	bg := image.NewRGBA(image.Rect(0, 0, size, size))
	fonts := LoadFonts([]string{"/usr/share/fonts/opentype/fonts-hosny-amiri/Amiri-Bold.ttf"},
		[]string{"/usr/share/fonts/opentype/fonts-hosny-amiri/Amiri-Regular.ttf"})
	wh := New(s, size, size, float64(size)/2, float64(size)/2, bg, dark, fonts, time.Unix(1000, 0))
	wh.Reveal(time.Unix(1000, 0))
	wh.SetItems(items, false, time.Unix(1000, 0))
	wh.Sharp = image.NewRGBA(wh.img.Rect)
	wh.Prepare()
	<-wh.prep
	return wh
}

func formatItems(n int) []Item {
	items := make([]Item, n)
	for i := range items {
		items[i] = Item{ID: "F", Label: "X", Enabled: true}
	}
	return items
}

// advance renders in ~1/120 s steps until ms have passed and returns the
// last frame.
func advance(wh *Wheel, now *time.Time, ms int) *image.RGBA {
	end := now.Add(time.Duration(ms) * time.Millisecond)
	var img *image.RGBA
	for !now.After(end) {
		*now = now.Add(8333 * time.Microsecond)
		img = wh.Render(*now)
	}
	return img
}

// sectorMass is the alpha mass of one segment's angular sector and the
// alpha-weighted mean radius (logical px) of that mass.
func sectorMass(img *image.RGBA, cx, cy, s float64, i, n int) (float64, float64) {
	span := 2 * math.Pi / float64(n)
	mid := -math.Pi/2 + float64(i)*span
	var sum, wsum float64
	for y := 0; y < img.Rect.Dy(); y++ {
		for x := 0; x < img.Rect.Dx(); x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			r := math.Hypot(dx, dy) / s
			if r < 62 || r > 146 {
				continue
			}
			a := math.Atan2(dy, dx)
			da := a - mid
			for da > math.Pi {
				da -= 2 * math.Pi
			}
			for da < -math.Pi {
				da += 2 * math.Pi
			}
			if math.Abs(da) > span/2-0.06 {
				continue
			}
			al := float64(img.Pix[img.PixOffset(x, y)+3])
			sum += al * r
			wsum += al
		}
	}
	if wsum == 0 {
		return 0, 0
	}
	return sum / wsum, wsum
}

func segAngle(i, n int) float64 { return -math.Pi/2 + float64(i)*2*math.Pi/float64(n) }

// TestHoverLiftsSegment: hovering an enabled segment pushes its blue fill
// outward - the furthest blue pixel moves from the resting outer edge
// (147 logical) to ~147*1.055+3.6 = ~158.7 logical.
func TestHoverLiftsSegment(t *testing.T) {
	wh := newTestWheel(false, formatItems(8))
	now := time.Unix(1000, 0)

	a := segAngle(1, 8)
	wh.SetPointer(wh.CX+math.Cos(a)*105*wh.Scale, wh.CY+math.Sin(a)*105*wh.Scale)
	lifted := cloneRGBA(advance(wh, &now, 400))

	span := 2 * math.Pi / 8
	mid := -math.Pi/2 + float64(1)*span
	best := 0.0
	for y := 0; y < lifted.Rect.Dy(); y++ {
		for x := 0; x < lifted.Rect.Dx(); x++ {
			dx, dy := float64(x)+0.5-wh.CX, float64(y)+0.5-wh.CY
			r := math.Hypot(dx, dy) / wh.Scale
			if r < 100 || r > 170 {
				continue
			}
			aa := math.Atan2(dy, dx) - mid
			for aa > math.Pi {
				aa -= 2 * math.Pi
			}
			for aa < -math.Pi {
				aa += 2 * math.Pi
			}
			if math.Abs(aa) > span/2-0.06 {
				continue
			}
			i := lifted.PixOffset(x, y)
			// the hover fill is the only BLUE thing in the sector
			if lifted.Pix[i+3] >= 200 && lifted.Pix[i+2] >= 200 && lifted.Pix[i] <= 120 && r > best {
				best = r
			}
		}
	}
	if best < 150 || best > 165 {
		t.Fatalf("furthest blue fill pixel at %.1f logical px, want ~158.7", best)
	}
}

// TestDisabledDoesNotLift: a disabled segment never animates.
func TestDisabledDoesNotLift(t *testing.T) {
	items := formatItems(8)
	items[1].Enabled = false
	wh := newTestWheel(false, items)
	now := time.Unix(1000, 0)
	rest := cloneRGBA(advance(wh, &now, 500)) // Render reuses its buffer
	r0, _ := sectorMass(rest, wh.CX, wh.CY, wh.Scale, 1, 8)

	a := segAngle(1, 8)
	wh.SetPointer(wh.CX+math.Cos(a)*105*wh.Scale, wh.CY+math.Sin(a)*105*wh.Scale)
	lifted := advance(wh, &now, 400)
	r1, _ := sectorMass(lifted, wh.CX, wh.CY, wh.Scale, 1, 8)
	if d := math.Abs(r1 - r0); d > 0.5 {
		t.Fatalf("disabled segment moved %.2f logical px, want 0", d)
	}
	if wh.liftA[1] > 0.01 {
		t.Fatalf("disabled segment lift = %.3f, want 0", wh.liftA[1])
	}
}

// TestHoverHysteresis: the pointer just past a border keeps the current
// hover; clearly past it, the hover switches.
func TestHoverHysteresis(t *testing.T) {
	wh := newTestWheel(false, formatItems(8))
	now := time.Unix(1000, 0)
	advance(wh, &now, 500)

	span := 2 * math.Pi / 8
	a := segAngle(1, 8)
	wh.SetPointer(wh.CX+math.Cos(a)*105*wh.Scale, wh.CY+math.Sin(a)*105*wh.Scale)
	advance(wh, &now, 60) // a rendered frame adopts the pointer into hoverIdx
	if got := wh.hoverAt(wh.px, wh.py); got != 1 {
		t.Fatalf("hover on segment centre = %d, want 1", got)
	}
	r := 105.0
	a2 := -math.Pi/2 + 1.5*span + 2/r // 2 logical px past the border
	wh.SetPointer(wh.CX+math.Cos(a2)*r*wh.Scale, wh.CY+math.Sin(a2)*r*wh.Scale)
	if got := wh.hoverAt(wh.px, wh.py); got != 1 {
		t.Fatalf("hover 2px past the border = %d, want 1 (hysteresis)", got)
	}
	a3 := -math.Pi/2 + 1.5*span + 10/r // 10 logical px past the border
	wh.SetPointer(wh.CX+math.Cos(a3)*r*wh.Scale, wh.CY+math.Sin(a3)*r*wh.Scale)
	if got := wh.hoverAt(wh.px, wh.py); got != 2 {
		t.Fatalf("hover 10px past the border = %d, want 2", got)
	}
}

// cloneRGBA copies a frame: Render returns its own buffer, which the next
// frame overwrites.
func cloneRGBA(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Rect)
	copy(dst.Pix, src.Pix)
	return dst
}
