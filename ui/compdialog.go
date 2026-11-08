package ui

import (
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"
	"sync/atomic"

	"golang.org/x/image/font"

	"github.com/fogleman/gg"
)

// Compress dialog geometry in logical units — the progress panel's glass.
const (
	cdlgW = 384.0
	cdlgH = 340.0
)

// control rects in logical card coordinates (x, y, w, h)
var (
	cdlgClose = [3]float64{32.5, 32.5, 14.5}
	cdlgRows  = [2][4]float64{{24, 106, 336, 40}, {24, 150, 336, 40}}
	cdlgCheck = [4]float64{24, 198, 336, 26}
	cdlgBox   = [4]float64{24, 230, 336, 36}
	cdlgStart = [4]float64{24, 288, 336, 36}
)

// CompressLabels holds the dialog's texts (filled in by the app, in the
// current language).
type CompressLabels struct {
	Title, File, SizeLine, Preset, Balanced, Strong, BalancedHint, StrongHint,
	Target, TargetLabel, Start string
	TargetTooBig string
}

// CompressDialog asks how to compress a video before it starts: a quality
// preset, or a target size in megabytes.
type CompressDialog struct {
	Scale float64
	W, H  int // window pixels
	Fonts *Fonts
	Dark  bool
	RTL   bool

	next atomic.Pointer[image.RGBA] // a rebuilt base waiting to be used
	busy atomic.Bool

	// DebugOutline makes Render draw 1 px outlines: red on each option
	// card, green around the measured ink bounds of its text block.
	DebugOutline bool

	Labels        CompressLabels
	Preset        int // 0 balanced, 1 strong
	Target        bool
	TargetMB      string  // what the user typed
	OriginalBytes float64 // the size of the original file, for the smaller-than check
	Error         string  // red line under the box
	Editing       bool    // the size box has the keyboard focus
	CaretOn       bool    // blink phase of the caret while editing
	selected      bool    // the whole number is selected: the next digit replaces it
	sizeText      string
	base          *image.RGBA
	work          *image.RGBA
	dc            *gg.Context
}

func NewCompressDialog(scale float64, bg *image.RGBA, fonts *Fonts, dark, rtl bool, labels CompressLabels, sizeText string) *CompressDialog {
	d := &CompressDialog{Scale: scale, Fonts: fonts, Dark: dark, RTL: rtl, Labels: labels, sizeText: sizeText}
	d.W, d.H = d.Size()
	d.work = image.NewRGBA(image.Rect(0, 0, d.W, d.H))
	d.dc = gg.NewContextForRGBA(d.work)
	d.base = d.buildBase(bg)
	return d
}

func (d *CompressDialog) Size() (int, int) {
	return int(math.Ceil((cdlgW + 2*PanelMargin) * d.Scale)), int(math.Ceil((cdlgH + 2*PanelMargin) * d.Scale))
}

// BytesPerMB is the megabyte the size box means: 1,048,576 bytes, the same
// as File Explorer and the compression itself (conv.TargetTooSmall).
const BytesPerMB = 1 << 20

// MB parses the typed megabytes (0 = not a valid size yet).
func (d *CompressDialog) MB() float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(d.TargetMB), 64)
	if err != nil || v <= 0 {
		return 0
	}
	return v
}

// StartActive is false while the start button can't run yet.
func (d *CompressDialog) StartActive() bool {
	if !d.Target {
		return true
	}
	mb := d.MB()
	if mb <= 0 {
		return false
	}
	if d.OriginalBytes > 0 && mb*BytesPerMB >= d.OriginalBytes {
		return false // the target is the original size or bigger
	}
	return true
}

// DefaultTargetMB is the size offered when "Compress to a set size" is
// turned on: 100 MB, or half the original (whole MB, at least 1) when the
// original is 100 MB or smaller, so the offer is always below the original.
func (d *CompressDialog) DefaultTargetMB() string {
	if d.OriginalBytes > 0 && d.OriginalBytes <= 100*BytesPerMB {
		return strconv.Itoa(max(1, int(d.OriginalBytes/2/BytesPerMB)))
	}
	return "100"
}

// Focus gives the size box the keyboard: the number is shown selected, so
// the first digit typed replaces it, and the caret is shown.
func (d *CompressDialog) Focus() {
	if !d.Target {
		return
	}
	d.Editing, d.selected, d.CaretOn = true, d.TargetMB != "", true
}

// Blur takes the keyboard focus away from the size box.
func (d *CompressDialog) Blur() { d.Editing, d.selected = false, false }

// Char edits the target-size field while it has the focus (digits and
// backspace only; Arabic-Indic digits from an Arabic keyboard become
// western digits). At most 4 digits.
func (d *CompressDialog) Char(r rune) {
	if !d.Target || !d.Editing {
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
			d.TargetMB = ""
		} else if n := len(d.TargetMB); n > 0 {
			d.TargetMB = d.TargetMB[:n-1]
		}
	case digit != 0:
		if d.selected {
			d.TargetMB = ""
		}
		if len(d.TargetMB) < 4 {
			d.TargetMB += string(digit)
		}
	default:
		return // anything else leaves the selection as it was
	}
	d.selected, d.CaretOn = false, true
}

// HitTest: 0 none, 1 the X, 2/3 the presets, 4 the checkbox, 5 the size
// box, 6 the start button (window pixels).
func (d *CompressDialog) HitTest(x, y float64) int {
	s := d.Scale
	inRect := func(r [4]float64) bool {
		rx, ry := (PanelMargin+r[0])*s, (PanelMargin+r[1])*s
		return x >= rx && x <= rx+r[2]*s && y >= ry && y <= ry+r[3]*s
	}
	cx, cy := (PanelMargin+cdlgClose[0])*s, (PanelMargin+cdlgClose[1])*s
	if math.Hypot(x-cx, y-cy) <= (cdlgClose[2]+4)*s {
		return 1
	}
	if inRect(cdlgRows[0]) {
		return 2
	}
	if inRect(cdlgRows[1]) {
		return 3
	}
	chk := cdlgCheck
	if d.RTL { // mirrored
		chk[0] = cdlgW - chk[0] - chk[2]
	}
	if inRect(chk) {
		return 4
	}
	if d.Target && inRect(cdlgBox) {
		return 5
	}
	if d.StartActive() && inRect(cdlgStart) {
		return 6
	}
	return 0
}

func (d *CompressDialog) buildBase(bg *image.RGBA) *image.RGBA {
	return glassBase(d.W, d.H, cdlgW, cdlgH, PanelMargin, panelR, d.Scale, bg, d.Dark)
}

// Rebackground rebuilds the glass over a new desktop capture in the
// background (while the window is dragged); the next frames pick it up.
func (d *CompressDialog) Rebackground(bg *image.RGBA) {
	if !d.busy.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer d.busy.Store(false)
		d.next.Store(d.buildBase(bg))
	}()
}

// Rebuild is Rebackground done synchronously (once, when a drag ends).
func (d *CompressDialog) Rebuild(bg *image.RGBA) { d.base = d.buildBase(bg) }

func (d *CompressDialog) grey() color.NRGBA {
	if d.Dark {
		return color.NRGBA{214, 214, 220, 255}
	}
	return color.NRGBA{88, 88, 93, 255}
}

func (d *CompressDialog) textCol() color.NRGBA {
	if d.Dark {
		return color.NRGBA{235, 235, 238, 255}
	}
	return color.NRGBA{40, 40, 45, 255}
}

// drawText draws s with its baseline centred at (x, y); the x is the side
// reading starts on — the left for English, the right for Arabic.
func (d *CompressDialog) drawText(dc *gg.Context, s string, x, y, px float64, bold bool, c color.Color) {
	if f := d.Fonts.Face(math.Round(px*d.Scale*4)/4, bold); f != nil {
		dc.SetFontFace(f)
		dc.SetColor(c)
		t := Visual(s)
		if d.RTL {
			tw, _ := dc.MeasureString(t)
			x -= tw
		}
		dc.DrawString(t, x, y+capHeightOf(f)/2)
	}
}

// Render draws one frame over the glass base.
func (d *CompressDialog) Render() *image.RGBA {
	if nb := d.next.Swap(nil); nb != nil {
		d.base = nb
	}
	copy(d.work.Pix, d.base.Pix)
	dc := d.dc
	s := d.Scale
	m := PanelMargin * s
	grey := d.grey()

	// X
	cx, cy := m+cdlgClose[0]*s, m+cdlgClose[1]*s
	dc.DrawCircle(cx, cy, cdlgClose[2]*s)
	if d.Dark {
		dc.SetColor(color.NRGBA{255, 255, 255, 14})
	} else {
		dc.SetColor(color.NRGBA{0, 0, 0, 8})
	}
	dc.Fill()
	dd := 4.6 * s
	dc.SetLineCap(gg.LineCapRound)
	dc.SetLineWidth(1.7 * s)
	dc.SetColor(grey)
	dc.DrawLine(cx-dd, cy-dd, cx+dd, cy+dd)
	dc.DrawLine(cx+dd, cy-dd, cx-dd, cy+dd)
	dc.Stroke()

	// title (right-aligned for Arabic, so it never touches the X)
	tx := m + 60*s
	if d.RTL {
		tx = m + (cdlgW-24)*s
	}
	d.drawText(dc, d.Labels.Title, tx, m+32.5*s, 17, true, d.textCol())
	// the file's name under the title, in the accent color
	d.drawText(dc, d.Labels.File, tx, m+54*s, 13, true, Accent)
	// original size + section
	sx := m + 24*s
	if d.RTL {
		sx = m + (cdlgW-24)*s
	}
	d.drawText(dc, d.Labels.SizeLine+" "+d.sizeText, sx, m+74*s, 13, false, grey)
	d.drawText(dc, d.Labels.Preset, sx, m+94*s, 13, true, d.textCol())

	// the two presets: the title+description block is centered by its
	// REAL ink (the faces' own ascent/descent — Arabic glyphs are taller
	// above and below the line than Latin caps), so the padding inside
	// the card is equal above and below.
	names := []string{d.Labels.Balanced, d.Labels.Strong}
	hints := []string{d.Labels.BalancedHint, d.Labels.StrongHint}
	fT := d.Fonts.Face(math.Round(14*s*4)/4, true)
	fH := d.Fonts.Face(math.Round(11*s*4)/4, false)
	ascT, _ := lineAscent(fT)
	ascH, descH := lineAscent(fH)
	gapPx := 4 * s
	blockH := ascT + gapPx + ascH + descH
	drawLine := func(f font.Face, str string, base float64, c color.Color) (float64, float64) {
		dc.SetFontFace(f)
		dc.SetColor(c)
		t := Visual(str)
		tw, _ := dc.MeasureString(t)
		xx := tx
		if d.RTL {
			xx -= tw
		}
		dc.DrawString(t, xx, base)
		return math.Min(xx, xx+tw), math.Max(xx, xx+tw)
	}
	for i, r := range cdlgRows {
		rx, ry, rw, rh := m+r[0]*s, m+r[1]*s, r[2]*s, r[3]*s
		dc.DrawRoundedRectangle(rx, ry, rw, rh, 10*s)
		if d.Dark {
			dc.SetColor(color.NRGBA{255, 255, 255, 14})
		} else {
			dc.SetColor(color.NRGBA{0, 0, 0, 12})
		}
		dc.Fill()
		if d.Preset == i {
			dc.SetLineWidth(2 * s)
			dc.SetColor(Accent)
			dc.Stroke()
		}
		top := ry + (rh-blockH)/2
		titleBase := top + ascT
		hintBase := titleBase + gapPx + ascH
		// dot on the side the text starts at, centered on the title line
		dx := rx + 16*s
		if d.RTL {
			dx = rx + rw - 16*s
		}
		dc.DrawCircle(dx, titleBase-ascT/2, 5*s)
		if d.Preset == i {
			dc.SetColor(Accent)
		} else if d.Dark {
			dc.SetColor(color.NRGBA{255, 255, 255, 60})
		} else {
			dc.SetColor(color.NRGBA{0, 0, 0, 50})
		}
		dc.Fill()
		tx = rx + 28*s
		if d.RTL {
			tx = rx + rw - 28*s
		}
		x0a, x1a := drawLine(fT, names[i], titleBase, d.textCol())
		x0b, x1b := drawLine(fH, hints[i], hintBase, grey)
		if d.DebugOutline {
			dc.SetLineWidth(1)
			dc.SetColor(color.NRGBA{225, 45, 45, 255})
			dc.DrawRectangle(rx+0.5, ry+0.5, rw-1, rh-1)
			dc.Stroke()
			dc.SetColor(color.NRGBA{40, 170, 70, 255})
			dc.DrawRectangle(math.Min(x0a, x0b)+0.5, top+0.5, math.Max(x1a, x1b)-math.Min(x0a, x0b)-1, blockH-1)
			dc.Stroke()
		}
	}
	// target checkbox + label (the square on the right for Arabic)
	cbx := m + 24*s
	if d.RTL {
		cbx = m + (cdlgW-40)*s
	}
	dc.DrawRoundedRectangle(cbx, m+202*s, 16*s, 16*s, 4*s)
	if d.Target {
		dc.SetColor(Accent)
		dc.Fill()
		dc.SetLineCap(gg.LineCapRound)
		dc.SetLineWidth(2.2 * s)
		dc.SetColor(color.NRGBA{255, 255, 255, 255})
		dc.DrawLine(cbx+4*s, m+210*s, cbx+7*s, m+213*s)
		dc.DrawLine(cbx+7*s, m+213*s, cbx+12*s, m+206*s)
		dc.Stroke()
	} else {
		dc.SetLineWidth(1.6 * s)
		dc.SetColor(grey)
		dc.Stroke()
	}
	lx := m + 48*s
	if d.RTL {
		lx = cbx - 8*s
	}
	d.drawText(dc, d.Labels.Target, lx, m+211*s, 13, false, d.textCol())

	// the size box (only when the target is on)
	if d.Target {
		dc.DrawRoundedRectangle(m+cdlgBox[0]*s, m+cdlgBox[1]*s, cdlgBox[2]*s, cdlgBox[3]*s, 8*s)
		if d.Dark {
			dc.SetColor(color.NRGBA{255, 255, 255, 22})
		} else {
			dc.SetColor(color.NRGBA{0, 0, 0, 18})
		}
		dc.Fill()
		if d.Editing { // focus state: an accent border (unfocused: none, as before)
			dc.DrawRoundedRectangle(m+cdlgBox[0]*s, m+cdlgBox[1]*s, cdlgBox[2]*s, cdlgBox[3]*s, 8*s)
			dc.SetLineWidth(2 * s)
			dc.SetColor(Accent)
			dc.Stroke()
		}
		if f := d.Fonts.Face(math.Round(14*s*4)/4, false); f != nil {
			// the typed number at the side reading starts, "MB" beside it
			dc.SetFontFace(f)
			num := d.TargetMB
			numCol := d.textCol()
			if num == "" && !d.Editing {
				num = d.DefaultTargetMB()
				numCol = grey
			}
			vnum := Visual(num)
			t := vnum + " MB"
			tw, _ := dc.MeasureString(t)
			nw, _ := dc.MeasureString(vnum)
			nx := m + (cdlgW-36)*s - tw // right-aligned for English
			if d.RTL {
				nx = m + 36*s
			}
			base := m + (cdlgBox[1]+24)*s + capHeightOf(f)/2
			capH := capHeightOf(f)
			if d.Editing && d.selected && nw > 0 { // the selected number
				dc.DrawRectangle(nx-2*s, base-capH-4*s, nw+4*s, capH+8*s)
				dc.SetColor(color.NRGBA{Accent.R, Accent.G, Accent.B, 90})
				dc.Fill()
			}
			dc.SetColor(numCol)
			dc.DrawString(t, nx, base)
			if d.Editing && d.CaretOn { // the caret after the number
				cx := nx + nw + 1.5*s
				dc.SetLineWidth(1.5 * s)
				dc.SetColor(d.textCol())
				dc.DrawLine(cx, base-capH-4*s, cx, base+4*s)
				dc.Stroke()
			}
		}
		tlx := m + 36*s
		if d.RTL {
			tlx = m + (cdlgW-36)*s
		}
		d.drawText(dc, d.Labels.TargetLabel, tlx, m+(cdlgBox[1]+24)*s, 12, false, grey)
	}

	// the red line under the box: the target-too-big hint, or an error
	msg := d.Error
	if d.Target && d.OriginalBytes > 0 && d.MB() > 0 && float64(d.MB())*BytesPerMB >= d.OriginalBytes {
		msg = d.Labels.TargetTooBig
	}
	if msg != "" {
		ex := m + 36*s
		if d.RTL {
			ex = m + (cdlgW-36)*s
		}
		d.drawText(dc, msg, ex, m+278*s, 12, false, color.NRGBA{214, 61, 46, 255})
	}

	// start button
	active := d.StartActive()
	dc.DrawRoundedRectangle(m+cdlgStart[0]*s, m+cdlgStart[1]*s, cdlgStart[2]*s, cdlgStart[3]*s, cdlgStart[3]*s/2)
	if active {
		dc.SetColor(Accent)
	} else {
		dc.SetColor(color.NRGBA{Accent.R, Accent.G, Accent.B, 110})
	}
	dc.Fill()
	if f := d.Fonts.Face(math.Round(15*s*4)/4, true); f != nil {
		dc.SetFontFace(f)
		dc.SetColor(color.NRGBA{255, 255, 255, 255})
		t := Visual(d.Labels.Start)
		tw, _ := dc.MeasureString(t)
		dc.DrawString(t, m+(cdlgW/2)*s-tw/2, m+(cdlgStart[1]+cdlgStart[3]/2)*s+capHeightOf(f)/2)
	}
	return d.work
}
