package ui

import (
	"os"
	"path/filepath"
	"testing"
)

// TestHelpCardOKButton: the OK button is drawn (the body's clip must not
// hide it) exactly where HitTest answers 2, in both directions and themes.
func TestHelpCardOKButton(t *testing.T) {
	font := ""
	for _, f := range []string{filepath.Join(os.Getenv("WINDIR"), "Fonts", "segoeui.ttf"), "/usr/share/fonts/opentype/inter/Inter-SemiBold.otf"} {
		if _, err := os.Stat(f); err == nil {
			font = f
			break
		}
	}
	if font == "" {
		t.Skip("no font")
	}
	fonts := LoadFonts([]string{font}, []string{font})
	body := "line\n"
	for i := 0; i < 6; i++ {
		body += body // long enough to scroll, like the real text
	}
	for _, scale := range []float64{1, 2} {
		for _, rtl := range []bool{false, true} {
			for _, dark := range []bool{false, true} {
				h := NewHelpCard(scale, fonts, dark, rtl, "How to use", body, "OK")
				img := h.Render()
				s, m := scale, helpMargin*scale
				bx, by := m+(helpW-helpOKW)/2*s, m+(helpH-helpPad-helpOKH)*s
				// a point on the button, left of its label
				x, y := bx+helpOKH*s/2, by+helpOKH*s/2
				c := img.RGBAAt(int(x), int(y))
				if c.R != Accent.R || c.G != Accent.G || c.B != Accent.B || c.A != 255 {
					t.Errorf("scale %v rtl %v dark %v: OK button not drawn at (%.0f,%.0f): %v", scale, rtl, dark, x, y, c)
				}
				if got := h.HitTest(x, y); got != 2 {
					t.Errorf("scale %v rtl %v dark %v: HitTest on the button = %d, want 2", scale, rtl, dark, got)
				}
				if got := h.HitTest(bx-3*s, y); got == 2 {
					t.Errorf("scale %v rtl %v dark %v: HitTest left of the button = 2", scale, rtl, dark)
				}
			}
		}
	}
}
