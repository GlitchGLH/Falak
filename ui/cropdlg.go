package ui

import (
	"image"
	"image/color"
	"math"
	"strconv"
)

// The Crop Video window: the preview shows a frame with the crop rectangle
// (8 handles; the area outside it dimmed; dragged inside to move it), the
// aspect ratio buttons (Free, 1:1, 16:9, 9:16, 4:3, 3:4), the W and H
// fields in pixels (kept in sync with the rectangle; typed like the
// Compress dialog's size box) and Crop. The picture itself never mirrors;
// the controls do in Arabic.

const (
	cropH       = 530.0 // card height
	cropLabelY  = 350.0
	cropChipY   = 368.0
	cropChipH   = 36.0
	cropChipG   = 6.0
	cropFieldY  = 418.0
	cropFieldH  = 36.0
	cropFieldLW = 70.0 // the room for "W" / "H" before each box
	cropBtnY    = 470.0
	cropBtnH    = 36.0
	cropGrab    = 11.0 // how far from a handle a press still takes it
	cropDigits  = 5
)

// CropRatios are the ratio buttons' width:height (0, 0 = Free).
var CropRatios = [][2]int{{0, 0}, {1, 1}, {16, 9}, {9, 16}, {4, 3}, {3, 4}}

// CropLabels holds the window's texts in the current language; Ratios
// are the buttons' labels in CropRatios' order.
type CropLabels struct {
	Title, File, Ratio, W, H, Crop, Preparing, PreviewFailed string
	Ratios                                                   []string
}

// Hit-test results of the Crop window.
const (
	CropNone = iota
	CropClose
	CropMove   // inside the rectangle
	CropHandle // one of the 8 handles (index: see cropHandles)
	CropChip   // a ratio button (index)
	CropField  // 0 = W, 1 = H
	CropButton
)

// cropHandles are the handles' sides: x and y each -1 (left/top), 0 (the
// middle) or 1 (right/bottom).
var cropHandles = [8][2]int{{-1, -1}, {0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}}

// CropDialog is the Crop Video window's picture and state. The rectangle
// is in pixels of the picture as shown (even numbers).
type CropDialog struct {
	videoCard
	Labels CropLabels

	VW, VH     int // the picture's size (0 until known)
	X, Y, W, H int
	Ratio      int // index into CropRatios
	Frame      *image.RGBA
	Failed     bool
	Progress   float64 // FFmpeg download share, -1 when none

	Field    int    // the field being typed in: -1 none, 0 W, 1 H
	Typed    string // its digits
	selected bool   // the whole number is selected: the next digit replaces it
	CaretOn  bool

	drag     int // CropMove or CropHandle while dragging, else 0
	dragH    int
	dragFrom [2]float64 // the press, in picture pixels
	dragRect [4]float64 // the rectangle at the press
}

func NewCropDialog(scale float64, fonts *Fonts, dark, rtl bool, labels CropLabels) *CropDialog {
	return &CropDialog{videoCard: newVideoCard(scale, fonts, dark, rtl, cropH), Labels: labels, Progress: -1, Field: -1}
}

// Ready reports whether the picture's size is known.
func (d *CropDialog) Ready() bool { return d.VW > 0 && d.VH > 0 && !d.Failed }

// SetVideo sets the picture's size and selects all of it.
func (d *CropDialog) SetVideo(w, h int) {
	d.VW, d.VH = w, h
	d.X, d.Y, d.W, d.H = 0, 0, w&^1, h&^1
}

// minSide is the smallest side allowed in each direction.
func (d *CropDialog) minSide() (float64, float64) {
	return float64(min(32, d.VW&^1)), float64(min(32, d.VH&^1))
}

// ratio is the chosen width/height (0 = Free).
func (d *CropDialog) ratio() float64 {
	r := CropRatios[d.Ratio]
	if r[1] == 0 {
		return 0
	}
	return float64(r[0]) / float64(r[1])
}

// set stores a rectangle given in float pixels: even numbers, inside the
// picture.
func (d *CropDialog) set(x, y, w, h float64) {
	ev := func(v float64) int { return int(math.Round(v/2)) * 2 }
	W, H := ev(w), ev(h)
	mw, mh := d.minSide()
	W = max(int(mw), min(W, d.VW&^1))
	H = max(int(mh), min(H, d.VH&^1))
	X := max(0, min(ev(x), (d.VW-W)&^1))
	Y := max(0, min(ev(y), (d.VH-H)&^1))
	d.X, d.Y, d.W, d.H = X, Y, W, H
}

// fitRatio gives the largest w×h of ratio r (w/h) not over maxW×maxH and
// not under the smallest side.
func (d *CropDialog) fitRatio(r, w, maxW, maxH float64) (float64, float64) {
	mw, mh := d.minSide()
	w = math.Min(w, math.Min(maxW, maxH*r))
	w = math.Max(w, math.Max(mw, mh*r))
	return w, w / r
}

// SetRatio chooses ratio button i: a fixed ratio makes the largest
// rectangle of that ratio around the current one's centre that fits the
// picture; Free keeps the rectangle.
func (d *CropDialog) SetRatio(i int) {
	d.Ratio = i
	r := d.ratio()
	if r == 0 || !d.Ready() {
		return
	}
	cx, cy := float64(d.X)+float64(d.W)/2, float64(d.Y)+float64(d.H)/2
	w, h := d.fitRatio(r, math.Inf(1), float64(d.VW), float64(d.VH))
	d.set(cx-w/2, cy-h/2, w, h)
}

// SetSize sets the width (field 0) or height (field 1) to v pixels, keeping
// the centre; with a fixed ratio the other side follows it.
func (d *CropDialog) SetSize(field, v int) {
	if !d.Ready() {
		return
	}
	cx, cy := float64(d.X)+float64(d.W)/2, float64(d.Y)+float64(d.H)/2
	w, h := float64(d.W), float64(d.H)
	if field == 0 {
		w = float64(v)
	} else {
		h = float64(v)
	}
	if r := d.ratio(); r > 0 {
		if field == 1 {
			w = h * r
		}
		w, h = d.fitRatio(r, w, float64(d.VW), float64(d.VH))
	}
	d.set(cx-w/2, cy-h/2, w, h)
}

// display is the picture's place in the preview (window pixels) and the
// window pixels per picture pixel; it matches how preview() draws the frame.
func (d *CropDialog) display() (x, y, k float64) {
	s := d.Scale
	px, py, pw, ph := d.px(vidPad), d.px(vidPrevY), (vidW-2*vidPad)*s, vidPrevH*s
	fw, fh := float64(d.VW), float64(d.VH)
	if d.Frame != nil {
		fw, fh = float64(d.Frame.Rect.Dx()), float64(d.Frame.Rect.Dy())
	}
	kf := math.Min(pw/fw, ph/fh)
	dw, dh := fw*kf, fh*kf
	x0, y0 := math.Round(px+(pw-dw)/2), math.Round(py+(ph-dh)/2)
	x1, y1 := math.Round(px+(pw+dw)/2), math.Round(py+(ph+dh)/2)
	return x0, y0, math.Min((x1-x0)/float64(d.VW), (y1-y0)/float64(d.VH))
}

// rectPx is the crop rectangle in window pixels.
func (d *CropDialog) rectPx() (x0, y0, x1, y1 float64) {
	ox, oy, k := d.display()
	return ox + float64(d.X)*k, oy + float64(d.Y)*k, ox + float64(d.X+d.W)*k, oy + float64(d.Y+d.H)*k
}

func (d *CropDialog) handlePx(i int) (float64, float64) {
	x0, y0, x1, y1 := d.rectPx()
	hx, hy := cropHandles[i][0], cropHandles[i][1]
	return x0 + (x1-x0)*float64(hx+1)/2, y0 + (y1-y0)*float64(hy+1)/2
}

// ChipRect is ratio button i in logical card coordinates (mirrored in Arabic).
func (d *CropDialog) ChipRect(i int) [4]float64 {
	n := float64(len(CropRatios))
	w := (vidW - 2*vidPad - (n-1)*cropChipG) / n
	return [4]float64{d.mirror(vidPad+float64(i)*(w+cropChipG), w), cropChipY, w, cropChipH}
}

// fieldRects are field f's label room and box (logical; W on the side
// reading starts).
func (d *CropDialog) fieldRects(f int) (label, box [4]float64) {
	half := (vidW - 2*vidPad - 16) / 2
	x := vidPad + float64(f)*(half+16)
	label = [4]float64{d.mirror(x, cropFieldLW), cropFieldY, cropFieldLW, cropFieldH}
	box = [4]float64{d.mirror(x+cropFieldLW, half-cropFieldLW), cropFieldY, half - cropFieldLW, cropFieldH}
	return
}

// HitTest says what window pixel (x, y) is on, with an index for a
// handle, a ratio button or a field.
func (d *CropDialog) HitTest(x, y float64) (int, int) {
	if d.overClose(x, y) {
		return CropClose, -1
	}
	if !d.Ready() {
		return CropNone, -1
	}
	if d.Frame != nil {
		best, bi := cropGrab*d.Scale, -1
		for i := range cropHandles {
			hx, hy := d.handlePx(i)
			if dist := math.Hypot(x-hx, y-hy); dist <= best {
				best, bi = dist, i
			}
		}
		if bi >= 0 {
			return CropHandle, bi
		}
		x0, y0, x1, y1 := d.rectPx()
		if x >= x0 && x <= x1 && y >= y0 && y <= y1 {
			return CropMove, -1
		}
	}
	for i := range CropRatios {
		if d.in(x, y, d.ChipRect(i), cropChipG/2) {
			return CropChip, i
		}
	}
	for f := 0; f < 2; f++ {
		if _, box := d.fieldRects(f); d.in(x, y, box, 0) {
			return CropField, f
		}
	}
	if d.in(x, y, [4]float64{vidPad, cropBtnY, vidW - 2*vidPad, cropBtnH}, 0) {
		return CropButton, -1
	}
	return CropNone, -1
}

// toPicture turns a window pixel into picture pixels.
func (d *CropDialog) toPicture(x, y float64) (float64, float64) {
	ox, oy, k := d.display()
	return (x - ox) / k, (y - oy) / k
}

// StartDrag begins moving the rectangle (CropMove) or dragging handle i
// from window pixel (x, y).
func (d *CropDialog) StartDrag(kind, i int, x, y float64) {
	d.drag, d.dragH = kind, i
	px, py := d.toPicture(x, y)
	d.dragFrom = [2]float64{px, py}
	d.dragRect = [4]float64{float64(d.X), float64(d.Y), float64(d.W), float64(d.H)}
}

// Dragging reports whether a drag is going on.
func (d *CropDialog) Dragging() bool { return d.drag != 0 }

// EndDrag ends it.
func (d *CropDialog) EndDrag() { d.drag = 0 }

// DragTo follows the pointer at window pixel (x, y): a move keeps the size
// inside the picture; a handle moves its sides, the opposite ones stay (a
// middle handle keeps the other direction's centre); a fixed ratio holds.
func (d *CropDialog) DragTo(x, y float64) {
	if d.drag == 0 {
		return
	}
	px, py := d.toPicture(x, y)
	dx, dy := px-d.dragFrom[0], py-d.dragFrom[1]
	rx, ry, rw, rh := d.dragRect[0], d.dragRect[1], d.dragRect[2], d.dragRect[3]
	vw, vh := float64(d.VW), float64(d.VH)
	if d.drag == CropMove {
		d.set(math.Max(0, math.Min(rx+dx, vw-rw)), math.Max(0, math.Min(ry+dy, vh-rh)), rw, rh)
		return
	}
	hx, hy := cropHandles[d.dragH][0], cropHandles[d.dragH][1]
	mw, mh := d.minSide()
	// the side that stays (the anchor), the room towards the dragged side
	// and the size the pointer asks for
	axis := func(h int, r0, rs, delta, limit float64) (anchor, room, want float64) {
		switch h {
		case -1:
			anchor = r0 + rs
			return anchor, anchor, anchor - (r0 + delta)
		case 1:
			return r0, limit - r0, rs + delta
		}
		c := r0 + rs/2
		return c, 2 * math.Min(c, limit-c), rs
	}
	ax, roomX, wantW := axis(hx, rx, rw, dx, vw)
	ay, roomY, wantH := axis(hy, ry, rh, dy, vh)
	var w, h float64
	if r := d.ratio(); r > 0 {
		switch {
		case hx != 0 && hy != 0:
			w = math.Max(wantW, wantH*r)
		case hx != 0:
			w = wantW
		default:
			w = wantH * r
		}
		w, h = d.fitRatio(r, w, roomX, roomY)
	} else {
		w = math.Max(mw, math.Min(wantW, roomX))
		h = math.Max(mh, math.Min(wantH, roomY))
	}
	place := func(hs int, anchor, size float64) float64 {
		switch hs {
		case -1:
			return anchor - size
		case 1:
			return anchor
		}
		return anchor - size/2
	}
	d.set(place(hx, ax, w), place(hy, ay, h), w, h)
}

// Focus gives field f (0 W, 1 H) the keyboard: its number is selected, so
// the first digit typed replaces it.
func (d *CropDialog) Focus(f int) {
	if d.Field >= 0 && d.Field != f {
		d.Commit()
	}
	if !d.Ready() {
		return
	}
	d.Field, d.Typed, d.selected, d.CaretOn = f, strconv.Itoa(d.fieldValue(f)), true, true
}

func (d *CropDialog) fieldValue(f int) int {
	if f == 0 {
		return d.W
	}
	return d.H
}

// Commit applies what was typed (an empty field changes nothing) and takes
// the keyboard away from the field.
func (d *CropDialog) Commit() {
	if d.Field < 0 {
		return
	}
	if v, err := strconv.Atoi(d.Typed); err == nil && v > 0 {
		d.SetSize(d.Field, v)
	}
	d.Field, d.Typed, d.selected = -1, "", false
}

// Char edits the field being typed in (digits and backspace; Arabic-Indic
// digits become western digits), at most 5 digits.
func (d *CropDialog) Char(r rune) {
	if d.Field < 0 {
		return
	}
	digit := rune(0)
	switch {
	case r >= '0' && r <= '9':
		digit = r
	case r >= '٠' && r <= '٩':
		digit = r - '٠' + '0'
	}
	switch {
	case r == '\b' || r == 127:
		if d.selected {
			d.Typed = ""
		} else if n := len(d.Typed); n > 0 {
			d.Typed = d.Typed[:n-1]
		}
	case digit != 0:
		if d.selected {
			d.Typed = ""
		}
		if len(d.Typed) < cropDigits {
			d.Typed += string(digit)
		}
	default:
		return
	}
	d.selected, d.CaretOn = false, true
}

// Render draws one frame.
func (d *CropDialog) Render() *image.RGBA {
	d.begin(d.Labels.Title, d.Labels.File)
	dc, s := d.dc, d.Scale
	status := d.Labels.Preparing
	if d.Failed {
		status = d.Labels.PreviewFailed
	}
	d.preview(d.Frame, status, d.Progress)
	if d.Frame != nil && d.Ready() {
		d.overlay()
	}
	d.textStart(d.Labels.Ratio, cropLabelY, 13, true, d.textCol())
	for i, label := range d.Labels.Ratios {
		r := d.ChipRect(i)
		dc.DrawRoundedRectangle(d.px(r[0]), d.px(r[1]), r[2]*s, r[3]*s, r[3]*s/2)
		col := d.textCol()
		switch {
		case i == d.Ratio:
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
	for f, name := range []string{d.Labels.W, d.Labels.H} {
		d.field(f, name)
	}
	d.button(d.Labels.Crop, cropBtnY, cropBtnH, d.Ready())
	return d.work
}

// overlay dims the picture outside the rectangle and draws its frame, the
// thirds and the 8 handles.
func (d *CropDialog) overlay() {
	dc, s := d.dc, d.Scale
	ox, oy, k := d.display()
	ex, ey := ox+float64(d.VW)*k, oy+float64(d.VH)*k
	x0, y0, x1, y1 := d.rectPx()
	dc.SetColor(color.NRGBA{0, 0, 0, 150})
	for _, r := range [][4]float64{{ox, oy, ex, y0}, {ox, y1, ex, ey}, {ox, y0, x0, y1}, {x1, y0, ex, y1}} {
		if r[2] > r[0] && r[3] > r[1] {
			dc.DrawRectangle(r[0], r[1], r[2]-r[0], r[3]-r[1])
			dc.Fill()
		}
	}
	dc.SetLineWidth(1 * s)
	dc.SetColor(color.NRGBA{255, 255, 255, 90})
	for i := 1; i < 3; i++ {
		fx := x0 + (x1-x0)*float64(i)/3
		fy := y0 + (y1-y0)*float64(i)/3
		dc.DrawLine(fx, y0, fx, y1)
		dc.DrawLine(x0, fy, x1, fy)
	}
	dc.Stroke()
	dc.SetLineWidth(1.5 * s)
	dc.SetColor(color.NRGBA{255, 255, 255, 255})
	dc.DrawRectangle(x0, y0, x1-x0, y1-y0)
	dc.Stroke()
	for i := range cropHandles {
		hx, hy := d.handlePx(i)
		dc.DrawCircle(hx, hy+0.6*s, 5.5*s)
		dc.SetColor(color.NRGBA{0, 0, 0, 70})
		dc.Fill()
		dc.DrawCircle(hx, hy, 5*s)
		dc.SetColor(color.NRGBA{255, 255, 255, 255})
		dc.Fill()
		dc.DrawCircle(hx, hy, 5*s)
		dc.SetLineWidth(1.2 * s)
		dc.SetColor(Accent)
		dc.Stroke()
	}
}

// field draws "W" or "H" and its box with the number in the middle (the
// typed digits, selected or with the caret, while it has the keyboard).
func (d *CropDialog) field(f int, name string) {
	dc, s := d.dc, d.Scale
	lr, box := d.fieldRects(f)
	col := d.textCol()
	if !d.Ready() {
		col.A = 110
	}
	lx := lr[0]
	if d.RTL {
		lx = lr[0] + lr[2]
	}
	d.text(name, lx, cropFieldY+cropFieldH/2, 13, true, col)
	bx, by, bw, bh := d.px(box[0]), d.px(box[1]), box[2]*s, box[3]*s
	dc.DrawRoundedRectangle(bx, by, bw, bh, 8*s)
	if d.Dark {
		dc.SetColor(color.NRGBA{255, 255, 255, 22})
	} else {
		dc.SetColor(color.NRGBA{0, 0, 0, 18})
	}
	dc.Fill()
	editing := d.Field == f
	if editing {
		dc.DrawRoundedRectangle(bx, by, bw, bh, 8*s)
		dc.SetLineWidth(2 * s)
		dc.SetColor(Accent)
		dc.Stroke()
	}
	num := ""
	switch {
	case editing:
		num = d.Typed
	case d.Ready():
		num = strconv.Itoa(d.fieldValue(f))
	}
	face := d.Fonts.Face(math.Round(14*s*4)/4, false)
	if face == nil {
		return
	}
	dc.SetFontFace(face)
	nw, _ := dc.MeasureString(num)
	nx := bx + bw/2 - nw/2
	capH := capHeightOf(face)
	base := by + bh/2 + capH/2
	if editing && d.selected && nw > 0 {
		dc.DrawRectangle(nx-2*s, base-capH-4*s, nw+4*s, capH+8*s)
		dc.SetColor(color.NRGBA{Accent.R, Accent.G, Accent.B, 90})
		dc.Fill()
	}
	dc.SetColor(col)
	dc.DrawString(num, nx, base)
	if editing && d.CaretOn {
		cx := nx + nw + 1.5*s
		dc.SetLineWidth(1.5 * s)
		dc.SetColor(d.textCol())
		dc.DrawLine(cx, base-capH-4*s, cx, base+4*s)
		dc.Stroke()
	}
}
