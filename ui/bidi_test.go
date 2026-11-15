package ui

import (
	"strings"
	"testing"
	"time"
)

// TestSubLineArabic: in the Arabic progress window the file name comes
// first (on the right) and the percent follows on its left.
func TestSubLineArabic(t *testing.T) {
	if got := subLine("Holiday.mp4", " · 45%", false); got != "Holiday.mp4 · 45%" {
		t.Errorf("English: %q", got)
	}
	if got := subLine("Holiday.mp4", " · 45%", true); got != "45% · Holiday.mp4" {
		t.Errorf("Arabic: %q, want the name on the right", got)
	}
	if got := subLine("Holiday.mp4", "", true); got != "Holiday.mp4" {
		t.Errorf("Arabic without a percent: %q", got)
	}
	// drawn: the name's ink is at the right end of the line, by the bar's end
	fonts := msgTestFonts(t)
	t0 := time.Unix(1000, 0)
	p := NewPanelDir(2, nil, fonts, "Converting to MP3", "Holiday.mp4", false, true, t0)
	p.SetProgress(0.45)
	img := p.Render(t0.Add(5 * time.Second))
	m := PanelMargin * 2
	y0, y1 := int(m+(subY-8)*2), int(m+(subY+8)*2)
	right := int(m + (barX+barW)*2)
	last := -1
	for y := y0; y < y1; y++ {
		for x := int(m); x < right+4; x++ {
			if c := img.RGBAAt(x, y); c.A == 255 && c.R < 160 && x > last {
				last = x
			}
		}
	}
	if last < right-6 {
		t.Errorf("the Arabic line ends at x %d, want it at the bar's end %d", last, right)
	}
}

// TestCardFileNameFirstArabic: a "<file name>: <Arabic message>" line in
// an Arabic card has the file name at the right end (read first), with the
// colon on its left and the Arabic sentence after it.
func TestCardFileNameFirstArabic(t *testing.T) {
	msg := "هذا الخيار لا ينطبق على الملفات المحددة" // notApplicable() in Arabic
	v := Visual("notes.txt: " + msg)
	if !strings.HasSuffix(v, " :notes.txt") {
		t.Errorf("drawn left to right as %q: want it to end with \" :notes.txt\"", v)
	}
	fonts := msgTestFonts(t)
	full := NewMessageCard(2, fonts, false, true, MsgError, "فلك", "notes.txt: "+msg, []string{"موافق"})
	name := NewMessageCard(2, fonts, false, true, MsgError, "فلك", "notes.txt", []string{"موافق"})
	fi, ni := full.Render(), name.Render()
	y0, y1 := int(full.px(msgBodyY)), int(full.px(msgBodyY+msgLine))
	minX, maxX, ink := 1<<30, -1, 0
	for y := y0; y < y1; y++ {
		for x := 0; x < ni.Rect.Dx(); x++ {
			if c := ni.RGBAAt(x, y); c.A == 255 && c.R < 100 {
				minX, maxX, ink = min(minX, x), max(maxX, x), ink+1
			}
		}
	}
	bad := 1 << 30 // allow a sub-pixel shift of the glyphs
	for sh := -2; sh <= 2; sh++ {
		d := 0
		for y := y0; y < y1; y++ {
			for x := minX; x <= maxX; x++ {
				if (fi.RGBAAt(x+sh, y).R < 100) != (ni.RGBAAt(x, y).R < 100) {
					d++
				}
			}
		}
		bad = min(bad, d)
	}
	if maxX < 0 || bad*5 > ink {
		t.Errorf("the right end of the line is not the file name (%d of %d ink pixels differ in x %d..%d)", bad, ink, minX, maxX)
	}
}

// TestVisualSpeedTitle: in «جار تغيير السرعة إلى %s» the speed reads as
// "0.5×" (the × right after the number), on the left end where the
// Arabic sentence finishes; the Split counter's number stays first.
func TestVisualSpeedTitle(t *testing.T) {
	for _, sp := range []string{"0.25×", "0.5×", "2×", "4×"} {
		if v := Visual("جار تغيير السرعة إلى " + sp); !strings.HasPrefix(v, sp+" ") {
			t.Errorf("%s: drawn as %q, want it to start with %q", sp, v, sp+" ")
		}
	}
	if v := Visual("3 نقاط تقسيم"); !strings.HasSuffix(v, " 3") {
		t.Errorf("the counter is drawn as %q, want the number at the right end", v)
	}
	if v := Visual("11 نقطة تقسيم"); !strings.HasSuffix(v, " 11") {
		t.Errorf("the counter is drawn as %q, want 11 at the right end", v)
	}
}
