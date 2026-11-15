package ui

import (
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/fogleman/gg"
)

// The message card replaces the Windows message box: the Compress dialog's
// solid card with the X top left, an icon and the title, the message text
// (wrapped; a long one scrolls) and one or more buttons along the bottom.
// Arabic mirrors the title, the text and the button order.

const (
	msgW        = 384.0 // card width, the Compress dialog's
	msgPad      = 24.0
	msgBodyY    = 64.0 // top of the text
	msgLine     = 20.0
	msgTextPx   = 13.0
	msgGap      = 20.0 // text to buttons
	msgBtnH     = 36.0
	msgBtnGap   = 12.0
	msgBottom   = 24.0
	msgMaxLines = 14 // more lines than this scroll
	msgIconR    = 11.0
)

// MsgIcon is the symbol beside the title, as the message box showed.
type MsgIcon int

const (
	MsgInfo MsgIcon = iota
	MsgError
	MsgQuestion
)

// HitTest results other than a button index.
const (
	MsgHitNone   = -1
	MsgHitClose  = -2
	MsgHitScroll = -3
)

// MessageCard draws one message. Button 0 is the default (Enter), drawn in
// the accent color; Focus is the button the keyboard is on.
type MessageCard struct {
	Scale float64
	W, H  int // window pixels
	Fonts *Fonts
	Dark  bool
	RTL   bool
	Focus int

	icon     MsgIcon
	title    string
	buttons  []string
	lines    []string
	visible  int // lines shown at once
	cardH    float64
	scrollPx float64
	maxPx    float64
	base     *image.RGBA
	work     *image.RGBA
	dc       *gg.Context
}

func NewMessageCard(scale float64, fonts *Fonts, dark, rtl bool, icon MsgIcon, title, body string, buttons []string) *MessageCard {
	c := &MessageCard{Scale: scale, Fonts: fonts, Dark: dark, RTL: rtl, icon: icon, title: title, buttons: buttons}
	dc := gg.NewContextForRGBA(image.NewRGBA(image.Rect(0, 0, 4, 4)))
	if f := fonts.Face(math.Round(msgTextPx*scale*4)/4, false); f != nil {
		dc.SetFontFace(f)
		c.lines = wrapText(dc, body, (msgW-2*msgPad)*scale)
	}
	c.visible = len(c.lines)
	if c.visible > msgMaxLines {
		c.visible = msgMaxLines
	}
	if c.visible < 1 {
		c.visible = 1
	}
	c.maxPx = math.Max(0, float64(len(c.lines)-c.visible)*msgLine*scale)
	c.cardH = msgBodyY + float64(c.visible)*msgLine + msgGap + msgBtnH + msgBottom
	c.W = int(math.Ceil((msgW + 2*PanelMargin) * scale))
	c.H = int(math.Ceil((c.cardH + 2*PanelMargin) * scale))
	c.work = image.NewRGBA(image.Rect(0, 0, c.W, c.H))
	c.dc = gg.NewContextForRGBA(c.work)
	c.base = glassBase(c.W, c.H, msgW, c.cardH, PanelMargin, panelR, scale, nil, dark)
	return c
}

// Size is the window size in pixels.
func (c *MessageCard) Size() (int, int) { return c.W, c.H }

// Lines is the message as wrapped for the card.
func (c *MessageCard) Lines() []string { return c.lines }

// Buttons is the number of buttons.
func (c *MessageCard) Buttons() int { return len(c.buttons) }

// wrapText breaks text into lines no wider than maxW, paragraph by
// paragraph; a word wider than a whole line (a long path) is cut.
func wrapText(dc *gg.Context, text string, maxW float64) []string {
	fits := func(s string) bool { w, _ := dc.MeasureString(Visual(s)); return w <= maxW }
	var out []string
	for _, para := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		cur := ""
		for _, w := range words {
			cand := w
			if cur != "" {
				cand = cur + " " + w
			}
			if fits(cand) {
				cur = cand
				continue
			}
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			for !fits(w) { // cut a word too long for any line
				r := []rune(w)
				n := len(r) - 1
				for n > 1 && !fits(string(r[:n])) {
					n--
				}
				out = append(out, string(r[:n]))
				w = string(r[n:])
			}
			cur = w
		}
		out = append(out, cur)
	}
	return out
}

// Scroll scrolls the text: a positive delta scrolls down (a wheel notch
// moves three lines).
func (c *MessageCard) Scroll(delta float64) {
	c.scrollPx = clampF(c.scrollPx+delta*3*msgLine*c.Scale, 0, c.maxPx)
}

// ButtonRect is button i in logical card coordinates (x, y, w, h). Button
// 0 is on the side reading starts: the left in English, the right in Arabic.
func (c *MessageCard) ButtonRect(i int) [4]float64 {
	n := float64(len(c.buttons))
	w := (msgW - 2*msgPad - (n-1)*msgBtnGap) / n
	x := msgPad + float64(i)*(w+msgBtnGap)
	if c.RTL {
		x = msgW - x - w
	}
	return [4]float64{x, c.cardH - msgBottom - msgBtnH, w, msgBtnH}
}

func (c *MessageCard) px(v float64) float64 { return (PanelMargin + v) * c.Scale }

// HitTest answers for window pixel (x, y): a button index, the X, the
// scrollbar strip or nothing (the card can be dragged from there).
func (c *MessageCard) HitTest(x, y float64) int {
	s := c.Scale
	if math.Hypot(x-c.px(cdlgClose[0]), y-c.px(cdlgClose[1])) <= (cdlgClose[2]+4)*s {
		return MsgHitClose
	}
	for i := range c.buttons {
		r := c.ButtonRect(i)
		if x >= c.px(r[0]) && x <= c.px(r[0]+r[2]) && y >= c.px(r[1]) && y <= c.px(r[1]+r[3]) {
			return i
		}
	}
	if c.maxPx > 0 && y >= c.px(msgBodyY) && y <= c.px(msgBodyY+float64(c.visible)*msgLine) &&
		math.Abs(x-c.px(c.sbX())) <= 6*s {
		return MsgHitScroll
	}
	return MsgHitNone
}

// sbX is the scrollbar's logical x, at the edge opposite to reading start.
func (c *MessageCard) sbX() float64 {
	if c.RTL {
		return 10
	}
	return msgW - 10
}

func (c *MessageCard) grey() color.NRGBA {
	if c.Dark {
		return color.NRGBA{214, 214, 220, 255}
	}
	return color.NRGBA{88, 88, 93, 255}
}

func (c *MessageCard) textCol() color.NRGBA {
	if c.Dark {
		return color.NRGBA{235, 235, 238, 255}
	}
	return color.NRGBA{40, 40, 45, 255}
}

// text draws s with its baseline centred on window pixel y; x is the side
// reading starts on (the left in English, the right in Arabic).
func (c *MessageCard) text(s string, x, y, size float64, bold bool, col color.Color) {
	f := c.Fonts.Face(math.Round(size*c.Scale*4)/4, bold)
	if f == nil || s == "" {
		return
	}
	c.dc.SetFontFace(f)
	c.dc.SetColor(col)
	t := Visual(s)
	if c.RTL {
		tw, _ := c.dc.MeasureString(t)
		x -= tw
	}
	c.dc.DrawString(t, x, y+capHeightOf(f)/2)
}

func (c *MessageCard) textCenter(s string, x, y, size float64, bold bool, col color.Color) {
	f := c.Fonts.Face(math.Round(size*c.Scale*4)/4, bold)
	if f == nil || s == "" {
		return
	}
	c.dc.SetFontFace(f)
	c.dc.SetColor(col)
	t := Visual(s)
	tw, _ := c.dc.MeasureString(t)
	c.dc.DrawString(t, x-tw/2, y+capHeightOf(f)/2)
}

// Render draws the card.
func (c *MessageCard) Render() *image.RGBA {
	copy(c.work.Pix, c.base.Pix)
	dc, s := c.dc, c.Scale
	grey := c.grey()

	// X, as in the Compress dialog
	cx, cy := c.px(cdlgClose[0]), c.px(cdlgClose[1])
	dc.DrawCircle(cx, cy, cdlgClose[2]*s)
	if c.Dark {
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

	// icon and title, on the side reading starts
	ix, tx := 60+msgIconR, 60+2*msgIconR+8
	if c.RTL {
		ix, tx = msgW-msgPad-msgIconR, msgW-msgPad-2*msgIconR-8
	}
	iconCol, glyph := Accent, "i"
	switch c.icon {
	case MsgError:
		iconCol, glyph = errorRed, "!"
	case MsgQuestion:
		glyph = "?"
	}
	dc.DrawCircle(c.px(ix), c.px(cdlgClose[1]), msgIconR*s)
	dc.SetColor(iconCol)
	dc.Fill()
	c.textCenter(glyph, c.px(ix), c.px(cdlgClose[1]), 14, true, color.NRGBA{255, 255, 255, 255})
	if f := c.Fonts.Face(math.Round(17*s*4)/4, true); f != nil {
		dc.SetFontFace(f)
		avail := (tx - 60) * s
		if !c.RTL {
			avail = (msgW - msgPad - tx) * s
		}
		c.text(fitText(dc, c.title, avail), c.px(tx), c.px(cdlgClose[1]), 17, true, c.textCol())
	}

	// the message, clipped and scrolled when it is long
	top, bottom := c.px(msgBodyY), c.px(msgBodyY+float64(c.visible)*msgLine)
	if c.maxPx > 0 {
		dc.Push()
		dc.DrawRectangle(c.px(0), top, msgW*s, bottom-top)
		dc.Clip()
	}
	sx := msgPad
	if c.RTL {
		sx = msgW - msgPad
	}
	lineH := msgLine * s
	for i, l := range c.lines {
		y := top + (float64(i)+0.5)*lineH - c.scrollPx
		if y < top-lineH || y > bottom+lineH {
			continue
		}
		c.text(l, c.px(sx), y, msgTextPx, false, c.textCol())
	}
	if c.maxPx > 0 {
		dc.Pop()
		dc.ResetClip()
		track := bottom - top
		thumbH := math.Max(24*s, track*track/(float64(len(c.lines))*lineH))
		thumbH = math.Min(thumbH, track)
		y0 := top + (c.scrollPx/c.maxPx)*(track-thumbH)
		dc.DrawRoundedRectangle(c.px(c.sbX())-1.5*s, y0, 3*s, thumbH, 1.5*s)
		if c.Dark {
			dc.SetColor(color.NRGBA{255, 255, 255, 110})
		} else {
			dc.SetColor(color.NRGBA{0, 0, 0, 90})
		}
		dc.Fill()
	}

	// buttons: the first in the accent color, the others quiet; with more
	// than one, a ring shows which one the keyboard is on
	for i, label := range c.buttons {
		r := c.ButtonRect(i)
		x, y, w, h := c.px(r[0]), c.px(r[1]), r[2]*s, r[3]*s
		dc.DrawRoundedRectangle(x, y, w, h, h/2)
		col := c.textCol()
		if i == 0 {
			dc.SetColor(Accent)
			col = color.NRGBA{255, 255, 255, 255}
		} else if c.Dark {
			dc.SetColor(color.NRGBA{255, 255, 255, 24})
		} else {
			dc.SetColor(color.NRGBA{0, 0, 0, 14})
		}
		dc.Fill()
		if len(c.buttons) > 1 && i == c.Focus {
			g := 3 * s
			dc.DrawRoundedRectangle(x-g, y-g, w+2*g, h+2*g, h/2+g)
			dc.SetColor(color.NRGBA{Accent.R, Accent.G, Accent.B, 200})
			dc.SetLineWidth(1.6 * s)
			dc.Stroke()
		}
		c.textCenter(label, x+w/2, y+h/2, 15, true, col)
	}
	return c.work
}
