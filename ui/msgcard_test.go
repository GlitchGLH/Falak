package ui

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func msgTestFonts(t *testing.T) *Fonts {
	for _, f := range []string{filepath.Join(os.Getenv("WINDIR"), "Fonts", "segoeui.ttf"), "/usr/share/fonts/opentype/inter/Inter-SemiBold.otf"} {
		if _, err := os.Stat(f); err == nil {
			return LoadFonts([]string{f}, []string{f})
		}
	}
	t.Skip("no font")
	return nil
}

// TestMessageCardButtons: every button is drawn where HitTest finds it,
// the first one in the accent color; Arabic mirrors the order (the first
// button on the right) and the X stays where the other Falak windows have it.
func TestMessageCardButtons(t *testing.T) {
	fonts := msgTestFonts(t)
	for _, scale := range []float64{1, 1.5, 2} {
		for _, rtl := range []bool{false, true} {
			for _, dark := range []bool{false, true} {
				for _, btns := range [][]string{{"OK"}, {"Yes", "No"}} {
					c := NewMessageCard(scale, fonts, dark, rtl, MsgQuestion, "Falak", "Start the helper now?", btns)
					img := c.Render()
					name := func() string {
						return strings.Join(btns, "/") + map[bool]string{false: " en", true: " ar"}[rtl] +
							map[bool]string{false: " light", true: " dark"}[dark]
					}
					for i := range btns {
						r := c.ButtonRect(i)
						// a point inside the button, clear of its label
						x, y := c.px(r[0]+r[3]/2), c.px(r[1]+r[3]/2)
						if got := c.HitTest(x, y); got != i {
							t.Errorf("%s scale %v: HitTest on button %d = %d", name(), scale, i, got)
						}
						if i == 0 {
							p := img.RGBAAt(int(x), int(y))
							if p.R != Accent.R || p.G != Accent.G || p.B != Accent.B || p.A != 255 {
								t.Errorf("%s scale %v: first button not drawn in the accent color: %v", name(), scale, p)
							}
						}
						if r[0] < msgPad-0.01 || r[0]+r[2] > msgW-msgPad+0.01 {
							t.Errorf("%s: button %d outside the card's padding: %v", name(), i, r)
						}
					}
					if len(btns) == 2 {
						a, b := c.ButtonRect(0), c.ButtonRect(1)
						if !rtl && a[0] >= b[0] || rtl && a[0] <= b[0] {
							t.Errorf("%s: first button at x %v, second at %v", name(), a[0], b[0])
						}
						if a[0]+a[2] > b[0] && b[0]+b[2] > a[0] {
							t.Errorf("%s: buttons overlap: %v %v", name(), a, b)
						}
					}
					if got := c.HitTest(c.px(cdlgClose[0]), c.px(cdlgClose[1])); got != MsgHitClose {
						t.Errorf("%s: HitTest on the X = %d", name(), got)
					}
					if got := c.HitTest(c.px(msgW/2), c.px(10)); got != MsgHitNone {
						t.Errorf("%s: HitTest on the card's top = %d, want a drag handle", name(), got)
					}
				}
			}
		}
	}
}

// TestMessageCardMirror: in Arabic the buttons are the English ones
// mirrored, and the text starts at the right.
func TestMessageCardMirror(t *testing.T) {
	fonts := msgTestFonts(t)
	en := NewMessageCard(1, fonts, false, false, MsgInfo, "Falak", "How to use", []string{"Yes", "No"})
	ar := NewMessageCard(1, fonts, false, true, MsgInfo, "فلك", "طريقة الاستخدام", []string{"نعم", "لا"})
	for i := 0; i < 2; i++ {
		e, a := en.ButtonRect(i), ar.ButtonRect(i)
		if d := a[0] - (msgW - e[0] - e[2]); d > 0.01 || d < -0.01 || a[1] != e[1] || a[2] != e[2] {
			t.Errorf("button %d: Arabic %v is not English %v mirrored", i, a, e)
		}
	}
	// ink of the one text line: left half in English, right half in Arabic
	ink := func(c *MessageCard) (left, right int) {
		img := c.Render()
		y0, y1 := int(c.px(msgBodyY)), int(c.px(msgBodyY+msgLine))
		for y := y0; y < y1; y++ {
			for x := int(c.px(0)); x < int(c.px(msgW)); x++ {
				if p := img.RGBAAt(x, y); p.R < 128 && p.A == 255 {
					if float64(x) < c.px(msgW/2) {
						left++
					} else {
						right++
					}
				}
			}
		}
		return
	}
	if l, r := ink(en); l == 0 || r != 0 {
		t.Errorf("English text: %d ink pixels on the left, %d on the right", l, r)
	}
	if l, r := ink(ar); r == 0 || l != 0 {
		t.Errorf("Arabic text: %d ink pixels on the left, %d on the right", l, r)
	}
}

// TestMessageCardWrap: long messages wrap inside the card without losing a
// word, a path too long for one line is cut, and a very long message
// scrolls instead of making the card taller than 14 lines.
func TestMessageCardWrap(t *testing.T) {
	fonts := msgTestFonts(t)
	path := `C:\Users\Someone\Documents\` + strings.Repeat("VeryLongFolderName", 8) + `\file.mp4`
	body := "Couldn't save the snapshot\n\n" + path + ": Access is denied."
	for _, rtl := range []bool{false, true} {
		c := NewMessageCard(1.25, fonts, false, rtl, MsgError, "Falak", body, []string{"OK"})
		dc := c.dc
		dc.SetFontFace(fonts.Face(msgTextPx*1.25, false))
		for _, l := range c.Lines() {
			if w, _ := dc.MeasureString(Visual(l)); w > (msgW-2*msgPad)*1.25+0.5 {
				t.Errorf("line wider than the card: %q (%v)", l, w)
			}
		}
		if got, want := strings.Join(c.Lines(), ""), strings.ReplaceAll(strings.ReplaceAll(body, "\n", ""), " ", ""); strings.ReplaceAll(got, " ", "") != want {
			t.Errorf("text lost in wrapping:\n got %q\nwant %q", got, want)
		}
		if c.Lines()[1] != "" {
			t.Errorf("the empty line between the paragraphs is gone: %q", c.Lines())
		}
	}

	long := strings.Repeat("A line of an error message.\n", 40)
	c := NewMessageCard(1, fonts, true, false, MsgError, "Falak", long, []string{"OK"})
	short := NewMessageCard(1, fonts, true, false, MsgError, "Falak", "One line", []string{"OK"})
	if want := short.H + int((msgMaxLines-1)*msgLine); c.H > want+1 || c.H < want-1 {
		t.Errorf("long message: card %d px tall, want %d (14 lines)", c.H, want)
	}
	c.Scroll(1000)
	if c.scrollPx != c.maxPx || c.maxPx <= 0 {
		t.Errorf("scroll to the end: %v of %v", c.scrollPx, c.maxPx)
	}
	c.Scroll(-1000)
	if c.scrollPx != 0 {
		t.Errorf("scroll to the top: %v", c.scrollPx)
	}
	_ = c.Render() // scrolled rendering must not panic
	var _ image.Image = c.Render()
}

// TestVisualGuillemets: the quotation marks around «فلك» in the helper
// messages face the right way once drawn (mirrored like brackets).
func TestVisualGuillemets(t *testing.T) {
	// the helper hint's approved Arabic (helper_windows.go)
	v := []rune(Visual("لا يستطيع «فلك» قراءة زر Shift عند السحب من سطح المكتب"))
	open, close := -1, -1
	for i, r := range v {
		switch r {
		case '«':
			open = i
		case '»':
			close = i
		}
	}
	if open < 0 || close < 0 || open > close {
		t.Errorf("drawn left to right as %q: want « on the left and » on the right", string(v))
	}
}
