package ui

import (
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/fogleman/gg"
)

// HelpCard geometry in logical units — the tray menu card's look: a glass
// card with a title, a scrollable body and an OK button.
const (
	helpW      = 372.0
	helpH      = 470.0
	helpMargin = 16.0 // room around the card for its soft shadow
	helpRadius = 12.0
	helpHeader = 52.0
	helpFooter = 56.0
	helpPad    = 18.0
	helpCloseR = 13.0
	helpLine   = 19.0
	helpTextPx = 13.0
	helpOKW    = 112.0
	helpOKH    = 36.0
)

// HelpCard is the "How to use" card, drawn like the tray menu (dark or
// light, Arabic or English) instead of the system message box.
type HelpCard struct {
	Scale float64
	W, H  int // window pixels
	Fonts *Fonts
	Dark  bool
	RTL   bool

	title, body, okText string
	lines               []string
	scrollPx            float64
	maxPx               float64
}

func NewHelpCard(scale float64, fonts *Fonts, dark, rtl bool, title, body, ok string) *HelpCard {
	h := &HelpCard{Scale: scale, Fonts: fonts, Dark: dark, RTL: rtl, title: title, body: body, okText: ok}
	h.W, h.H = h.Size()
	dc := gg.NewContextForRGBA(image.NewRGBA(image.Rect(0, 0, 4, 4)))
	if f := fonts.Face(math.Round(helpTextPx*scale*4)/4, false); f != nil {
		dc.SetFontFace(f)
		h.lines = h.wrap(dc, body, (helpW-2*helpPad)*scale)
	}
	bodyH := (helpH - helpHeader - helpFooter - helpPad) * scale
	h.maxPx = math.Max(0, float64(len(h.lines))*helpLine*scale-bodyH)
	return h
}

func (h *HelpCard) Size() (int, int) {
	return int(math.Ceil((helpW + 2*helpMargin) * h.Scale)), int(math.Ceil((helpH + 2*helpMargin) * h.Scale))
}

// wrap breaks the text into lines of maxW, paragraph by paragraph.
func (h *HelpCard) wrap(dc *gg.Context, text string, maxW float64) []string {
	var out []string
	for _, para := range strings.Split(text, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		cur := words[0]
		for _, w := range words[1:] {
			cand := cur + " " + w
			if mw, _ := dc.MeasureString(Visual(cand)); mw > maxW {
				out = append(out, cur)
				cur = w
			} else {
				cur = cand
			}
		}
		out = append(out, cur)
	}
	return out
}

// Scroll scrolls the body: a positive delta scrolls down (a wheel notch
// moves three lines).
func (h *HelpCard) Scroll(delta float64) {
	h.scrollPx = clampF(h.scrollPx+delta*3*helpLine*h.Scale, 0, h.maxPx)
}

// HitTest: 0 none, 1 the X, 2 the OK button (window pixels).
func (h *HelpCard) HitTest(x, y float64) int {
	s := h.Scale
	m := helpMargin * s
	cx, cy := h.xCenter(), m+helpHeader/2*s
	if math.Hypot(x-cx, y-cy) <= (helpCloseR+4)*s {
		return 1
	}
	bx, by := m+(helpW-helpOKW)/2*s, m+(helpH-helpPad-helpOKH)*s
	if x >= bx && x <= bx+helpOKW*s && y >= by && y <= by+helpOKH*s {
		return 2
	}
	if h.maxPx > 0 { // the scrollbar strip: never a drag handle
		top := m + (helpHeader+helpPad*0.5)*s
		bottom := m + (helpH-helpFooter-helpPad*0.5)*s
		if y >= top && y <= bottom && math.Abs(x-h.sbX()) <= 6*s {
			return 3
		}
	}
	return 0
}

// xCenter is the close X's center: top right in English, top LEFT in
// Arabic (mirrored, like the tray menu), so it never meets the title.
func (h *HelpCard) xCenter() float64 {
	m := helpMargin * h.Scale
	if h.RTL {
		return m + (helpPad+helpCloseR)*h.Scale
	}
	return m + (helpW-helpPad-helpCloseR)*h.Scale
}

// sbX is the scrollbar thumb's center x, on the side reading starts.
func (h *HelpCard) sbX() float64 {
	m := helpMargin * h.Scale
	if h.RTL {
		return m + 8*h.Scale
	}
	return m + (helpW-8)*h.Scale
}

// Render draws the whole card: shadow, glass, title, X, the body clipped
// to its area at the current scroll, and the OK button.
func (h *HelpCard) Render() *image.RGBA {
	s := h.Scale
	out := image.NewRGBA(image.Rect(0, 0, h.W, h.H))
	dc := gg.NewContextForRGBA(out)
	m := helpMargin * s

	// soft shadow
	sh := image.NewRGBA(out.Rect)
	sdc := gg.NewContextForRGBA(sh)
	sdc.DrawRoundedRectangle(m, m+4*s, helpW*s, helpH*s, helpRadius*s)
	sdc.SetColor(color.NRGBA{0, 0, 0, 60})
	sdc.Fill()
	sh = GaussianBlur(sh, 7*s)
	for i := 3; i+3 < len(out.Pix) && i+3 < len(sh.Pix); i += 4 {
		out.Pix[i] = sh.Pix[i]
	}

	// glass body
	dc.DrawRoundedRectangle(m, m, helpW*s, helpH*s, helpRadius*s)
	if h.Dark {
		dc.SetColor(color.NRGBA{darkGlass.R, darkGlass.G, darkGlass.B, 246})
	} else {
		dc.SetColor(color.NRGBA{250, 250, 250, 246})
	}
	dc.Fill()
	dc.SetColor(color.NRGBA{0, 0, 0, 52})
	dc.SetLineWidth(1 * s)
	dc.Stroke()

	grey := color.NRGBA{88, 88, 93, 255}
	if h.Dark {
		grey = color.NRGBA{214, 214, 220, 255}
	}

	// title
	if f := h.Fonts.Face(math.Round(15*s*4)/4, true); f != nil {
		dc.SetFontFace(f)
		dc.SetColor(grey)
		t := Visual(h.title)
		tw, _ := dc.MeasureString(t)
		x := m + helpPad*s
		if h.RTL {
			x = m + (helpW-helpPad)*s - tw
		}
		dc.DrawString(t, x, m+helpHeader/2*s+capHeightOf(f)/2)
	}

	// X (top right in English, top left in Arabic)
	cx, cy := h.xCenter(), m+helpHeader/2*s
	dc.DrawCircle(cx, cy, helpCloseR*s)
	if h.Dark {
		dc.SetColor(color.NRGBA{255, 255, 255, 20})
	} else {
		dc.SetColor(color.NRGBA{0, 0, 0, 14})
	}
	dc.Fill()
	d := 4.2 * s
	dc.SetLineCap(gg.LineCapRound)
	dc.SetLineWidth(1.6 * s)
	dc.SetColor(grey)
	dc.DrawLine(cx-d, cy-d, cx+d, cy+d)
	dc.DrawLine(cx+d, cy-d, cx-d, cy+d)
	dc.Stroke()

	// body, clipped and scrolled
	if f := h.Fonts.Face(math.Round(helpTextPx*s*4)/4, false); f != nil {
		dc.SetFontFace(f)
		textCol := color.NRGBA{64, 64, 70, 255}
		if h.Dark {
			textCol = color.NRGBA{200, 200, 206, 255}
		}
		dc.SetColor(textCol)
		top := m + (helpHeader+helpPad*0.5)*s
		bottom := m + (helpH-helpFooter-helpPad*0.5)*s
		dc.Push()
		dc.DrawRoundedRectangle(m+2, top, helpW*s-4, bottom-top, helpRadius*s)
		dc.Clip()
		lineH := helpLine * s
		start := int(h.scrollPx / lineH)
		off := h.scrollPx - float64(start)*lineH
		capH := capHeightOf(f)
		for i := start; i < len(h.lines); i++ {
			yy := top - off + float64(i-start)*lineH + lineH/2
			if yy > bottom+lineH {
				break
			}
			if yy < top-lineH/2 {
				continue
			}
			t := Visual(h.lines[i])
			x := m + helpPad*s
			if h.RTL {
				tw, _ := dc.MeasureString(t)
				x = m + (helpW-helpPad)*s - tw
			}
			dc.DrawString(t, x, yy+capH/2)
		}
		dc.Pop()
		dc.ResetClip() // gg's Pop keeps the clip: without this the OK button below is never drawn
	}

	// OK button
	bx, by := m+(helpW-helpOKW)/2*s, m+(helpH-helpPad-helpOKH)*s
	dc.DrawRoundedRectangle(bx, by, helpOKW*s, helpOKH*s, helpOKH*s/2)
	dc.SetColor(Accent)
	dc.Fill()
	if f := h.Fonts.Face(math.Round(14*s*4)/4, true); f != nil {
		dc.SetFontFace(f)
		dc.SetColor(color.NRGBA{255, 255, 255, 255})
		t := Visual(h.okText)
		tw, _ := dc.MeasureString(t)
		dc.DrawString(t, bx+helpOKW*s/2-tw/2, by+helpOKH*s/2+capHeightOf(f)/2)
	}

	// scrollbar thumb, only while more text exists below
	if h.maxPx > 0 {
		top := m + (helpHeader+helpPad*0.5)*s
		track := (helpH - helpHeader - helpFooter - helpPad) * s
		total := float64(len(h.lines)) * helpLine * s
		thumbH := math.Max(24*s, track*track/total)
		if thumbH > track {
			thumbH = track
		}
		y0 := top + (h.scrollPx/h.maxPx)*(track-thumbH)
		dc.DrawRoundedRectangle(h.sbX()-1.5*s, y0, 3*s, thumbH, 1.5*s)
		if h.Dark {
			dc.SetColor(color.NRGBA{255, 255, 255, 110})
		} else {
			dc.SetColor(color.NRGBA{0, 0, 0, 90})
		}
		dc.Fill()
	}
	return out
}
