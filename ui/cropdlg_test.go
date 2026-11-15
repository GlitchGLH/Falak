package ui

import (
	"image"
	"testing"
)

func newTestCrop(t *testing.T, rtl bool) *CropDialog {
	d := NewCropDialog(1.5, msgTestFonts(t), false, rtl, CropLabels{
		Title: "Crop Video", File: "clip.mp4", Ratio: "Aspect ratio", W: "W", H: "H", Crop: "Crop",
		Ratios: []string{"Free", "1:1", "16:9", "9:16", "4:3", "3:4"},
	})
	d.SetVideo(1920, 1080)
	pw, ph := d.PreviewSize()
	d.Frame = image.NewRGBA(image.Rect(0, 0, pw, ph))
	return d
}

func (d *CropDialog) rect() [4]int { return [4]int{d.X, d.Y, d.W, d.H} }

// dragHandle drags handle i by (dx, dy) picture pixels.
func dragHandle(d *CropDialog, i int, dx, dy float64) {
	x, y := d.handlePx(i)
	_, _, k := d.display()
	d.StartDrag(CropHandle, i, x, y)
	d.DragTo(x+dx*k, y+dy*k)
	d.EndDrag()
}

func evenRect(t *testing.T, d *CropDialog, what string) {
	t.Helper()
	r := d.rect()
	for _, v := range r {
		if v%2 != 0 {
			t.Errorf("%s: odd number in %v", what, r)
		}
	}
	if r[0] < 0 || r[1] < 0 || r[0]+r[2] > d.VW || r[1]+r[3] > d.VH {
		t.Errorf("%s: %v leaves the %dx%d picture", what, r, d.VW, d.VH)
	}
}

// TestCropDrag: the whole picture at start; a corner handle moves only its
// sides; the rectangle moves inside the picture only; a fixed ratio keeps
// its shape on corner and side handles; the size never goes under 32.
func TestCropDrag(t *testing.T) {
	d := newTestCrop(t, false)
	if d.rect() != [4]int{0, 0, 1920, 1080} {
		t.Fatalf("start %v", d.rect())
	}
	dragHandle(d, 4, -400, -300) // bottom right, inwards
	if r := d.rect(); r[0] != 0 || r[1] != 0 || abs(r[2]-1520) > 2 || abs(r[3]-780) > 2 {
		t.Errorf("bottom-right handle: %v", r)
	}
	evenRect(t, d, "corner")
	x0, y0, _, _ := d.rectPx()
	_, _, k := d.display()
	d.StartDrag(CropMove, -1, x0+10, y0+10)
	d.DragTo(x0+10+5000*k, y0+10+5000*k) // far past the corner
	d.EndDrag()
	if r := d.rect(); r[0]+r[2] != 1920 || r[1]+r[3] != 1080 {
		t.Errorf("moved past the corner: %v", r)
	}
	evenRect(t, d, "move")
	dragHandle(d, 0, 5000, 5000) // top left, past the opposite corner
	if r := d.rect(); r[2] != 32 || r[3] != 32 {
		t.Errorf("smallest: %v, want 32×32", r)
	}

	d.SetVideo(1920, 1080)
	d.SetRatio(1) // 1:1
	if d.rect() != [4]int{420, 0, 1080, 1080} {
		t.Errorf("1:1 = %v, want the centred 1080 square", d.rect())
	}
	dragHandle(d, 2, -300, 100) // top right
	if r := d.rect(); r[2] != r[3] || r[0] != 420 || r[1]+r[3] != 1080 {
		t.Errorf("1:1 corner: %v", r)
	}
	d.SetRatio(2) // 16:9
	if d.rect() != [4]int{0, 0, 1920, 1080} {
		t.Errorf("16:9 on a 16:9 video = %v", d.rect())
	}
	dragHandle(d, 3, -640, 0) // the right side
	if r := d.rect(); r[2] != 1280 || r[3] != 720 || r[1] != 180 {
		t.Errorf("16:9 right side: %v, want 1280×720 around the old middle", r)
	}
	d.SetRatio(3) // 9:16 fits the height
	if r := d.rect(); r[3] != 1080 || abs(r[2]-608) > 2 {
		t.Errorf("9:16 = %v", r)
	}
	evenRect(t, d, "9:16")
}

func abs(v int) int { return max(v, -v) }

// TestCropFields: typing in W or H (Arabic-Indic digits too) sets the size
// around the centre when committed, even and inside the picture; with a
// fixed ratio the other side follows.
func TestCropFields(t *testing.T) {
	d := newTestCrop(t, true)
	d.Focus(0)
	for _, r := range "١٢٨٠" {
		d.Char(r)
	}
	if d.Typed != "1280" {
		t.Fatalf("typed %q", d.Typed)
	}
	d.Commit()
	if d.rect() != [4]int{320, 0, 1280, 1080} || d.Field != -1 {
		t.Errorf("W 1280: %v", d.rect())
	}
	d.Focus(1)
	d.Char('7')
	d.Char('2')
	d.Char('1')
	d.Commit()
	if d.H != 720 && d.H != 722 {
		t.Errorf("H 721: %d, want an even number", d.H)
	}
	d.SetRatio(2)
	d.Focus(0)
	for _, r := range "99999" {
		d.Char(r)
	}
	d.Char('9') // a sixth digit is ignored
	d.Commit()
	if d.rect() != [4]int{0, 0, 1920, 1080} {
		t.Errorf("W 99999 at 16:9: %v", d.rect())
	}
	d.Focus(1)
	d.Char('4')
	d.Commit()
	if r := d.rect(); r[3] != 32 || abs(r[2]-56) > 2 {
		t.Errorf("H 4 at 16:9: %v, want the smallest 16:9", r)
	}
	d.Focus(0)
	d.Char('\b')
	d.Commit() // an empty field changes nothing
	if r := d.rect(); r[3] != 32 {
		t.Errorf("empty field changed the size: %v", r)
	}
}

// TestCropHitTest: everything is found where it is drawn in both
// directions; the ratio buttons, W/H and their order mirror in Arabic.
func TestCropHitTest(t *testing.T) {
	for _, rtl := range []bool{false, true} {
		d := newTestCrop(t, rtl)
		d.SetRatio(1)
		for i := range cropHandles {
			x, y := d.handlePx(i)
			if k, j := d.HitTest(x, y); k != CropHandle || j != i {
				t.Errorf("rtl %v: handle %d: %d,%d", rtl, i, k, j)
			}
		}
		x0, y0, x1, y1 := d.rectPx()
		if k, _ := d.HitTest((x0+x1)/2, (y0+y1)/2); k != CropMove {
			t.Errorf("rtl %v: inside: %d", rtl, k)
		}
		for i := range CropRatios {
			r := d.ChipRect(i)
			if k, j := d.HitTest(d.px(r[0]+r[2]/2), d.px(r[1]+r[3]/2)); k != CropChip || j != i {
				t.Errorf("rtl %v: ratio %d: %d,%d", rtl, i, k, j)
			}
		}
		for f := 0; f < 2; f++ {
			_, b := d.fieldRects(f)
			if k, j := d.HitTest(d.px(b[0]+b[2]/2), d.px(b[1]+b[3]/2)); k != CropField || j != f {
				t.Errorf("rtl %v: field %d: %d,%d", rtl, f, k, j)
			}
		}
		_, w := d.fieldRects(0)
		if mid := vidW / 2; !rtl && w[0] > mid || rtl && w[0] < mid {
			t.Errorf("rtl %v: W at x %v", rtl, w[0])
		}
		if first := d.ChipRect(0); !rtl && first[0] > vidW/2 || rtl && first[0] < vidW/2 {
			t.Errorf("rtl %v: Free at x %v", rtl, first[0])
		}
		if k, _ := d.HitTest(d.px(vidW/2), d.px(cropBtnY+cropBtnH/2)); k != CropButton {
			t.Errorf("rtl %v: Crop: %d", rtl, k)
		}
	}
}

// boxPixels copies a logical rect of the picture.
func boxPixels(img *image.RGBA, c *videoCard, r [4]float64) *image.RGBA {
	b := image.Rect(int(c.px(r[0])), int(c.px(r[1])), int(c.px(r[0]+r[2])), int(c.px(r[1]+r[3])))
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		copy(out.Pix[y*out.Stride:], img.Pix[img.PixOffset(b.Min.X, b.Min.Y+y):img.PixOffset(b.Max.X, b.Min.Y+y)])
	}
	return out
}

// pixelDiff is the mean difference per channel between a and b (over
// their shared size), at the best horizontal shift of up to 3 pixels: mirrored places
// can land on a different fraction of a pixel.
func pixelDiff(a, b *image.RGBA) float64 {
	best := 1e9
	w, h := min(a.Rect.Dx(), b.Rect.Dx()), min(a.Rect.Dy(), b.Rect.Dy())
	for sh := -3; sh <= 3; sh++ {
		sum, n := 0.0, 0
		for y := 0; y < h; y++ {
			for x := 3; x < w-3; x++ {
				for c := 0; c < 3; c++ {
					d := int(a.Pix[a.PixOffset(x, y)+c]) - int(b.Pix[b.PixOffset(x+sh, y)+c])
					sum += float64(max(d, -d))
					n++
				}
			}
		}
		best = min(best, sum/float64(n))
	}
	return best
}

// closest is the index of the picture in refs most like img.
func closest(img *image.RGBA, refs []*image.RGBA) (int, []float64) {
	best, diffs := 0, make([]float64, len(refs))
	for j, r := range refs {
		diffs[j] = pixelDiff(img, r)
		if diffs[j] < diffs[best] {
			best = j
		}
	}
	return best, diffs
}

// TestCropNumbersDontFlip: in the Arabic window each ratio label ("16:9",
// never "9:16"), each W/H number and each speed value ("0.5×") looks most
// like the same label in the English window, out of all of them (a
// mirrored place may sit on another fraction of a pixel, so the pictures
// are close, not equal). The strings themselves come out of Visual as they
// go in.
func TestCropNumbersDontFlip(t *testing.T) {
	for _, s := range []string{"16:9", "9:16", "4:3", "3:4", "1:1", "1280", "0.5×", "0.25×"} {
		if v := Visual(s); v != s {
			t.Errorf("Visual(%q) = %q", s, v)
		}
	}
	en, ar := newTestCrop(t, false), newTestCrop(t, true)
	en.SetSize(0, 1280)
	ar.SetSize(0, 1280)
	ie, ia := en.Render(), ar.Render()
	var refs []*image.RGBA
	for i := range CropRatios {
		refs = append(refs, boxPixels(ie, &en.videoCard, en.ChipRect(i)))
	}
	for i := 1; i < len(CropRatios); i++ { // "Free" is a word, not a number
		if j, diffs := closest(boxPixels(ia, &ar.videoCard, ar.ChipRect(i)), refs); j != i {
			t.Errorf("Arabic %s looks like English %s (%v)", en.Labels.Ratios[i], en.Labels.Ratios[j], diffs)
		}
	}
	var fields []*image.RGBA
	for f := 0; f < 2; f++ {
		_, b := en.fieldRects(f)
		fields = append(fields, boxPixels(ie, &en.videoCard, b))
	}
	for f := 0; f < 2; f++ {
		_, b := ar.fieldRects(f)
		if j, diffs := closest(boxPixels(ia, &ar.videoCard, b), fields); j != f {
			t.Errorf("Arabic field %d (%d) looks like English field %d (%v)", f, ar.fieldValue(f), j, diffs)
		}
	}
	se, sa := newTestSpeed(t, false), newTestSpeed(t, true)
	se.Sel, sa.Sel = 1, 1
	je, ja := se.Render(), sa.Render()
	var chips []*image.RGBA
	for i := range se.Labels.Chips {
		chips = append(chips, boxPixels(je, &se.videoCard, se.ChipRect(i)))
	}
	for i := range sa.Labels.Chips {
		if j, diffs := closest(boxPixels(ja, &sa.videoCard, sa.ChipRect(i)), chips); j != i {
			t.Errorf("Arabic %s looks like English %s (%v)", se.Labels.Chips[i], se.Labels.Chips[j], diffs)
		}
	}
}
