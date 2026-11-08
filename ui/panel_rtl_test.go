package ui

import (
	"bytes"
	"image/color"
	"testing"
	"time"
)

// TestPanelDirection: in Arabic the bar fills from the right and the X
// stays (drawn and clickable) at the top left; English is unchanged.
func TestPanelDirection(t *testing.T) {
	fonts := LoadFonts(nil, nil) // the embedded fallback face is enough here
	t0 := time.Unix(1000, 0)
	render := func(p *Panel) []byte {
		p.SetProgress(0.25)
		var img []byte
		for i := 1; i <= 240; i++ { // past the pop-in and the bar's easing
			img = p.Render(t0.Add(time.Duration(i) * 16 * time.Millisecond)).Pix
		}
		return append([]byte(nil), img...)
	}
	for _, s := range []float64{1, 2} {
		// English: NewPanelDir(rtl=false) is exactly NewPanelTheme
		a := render(NewPanelTheme(s, nil, fonts, "Converting to MP3", "Holiday.mp4", false, t0))
		b := render(NewPanelDir(s, nil, fonts, "Converting to MP3", "Holiday.mp4", false, false, t0))
		if !bytes.Equal(a, b) {
			t.Fatalf("scale %v: the English panel changed", s)
		}
		ar := NewPanelDir(s, nil, fonts, "Converting to MP3", "Holiday.mp4", false, true, t0) // the direction comes from rtl, not from the text
		render(ar)
		img := ar.Render(t0.Add(5 * time.Second))
		m := PanelMargin * s
		y := int(m + barY*s)
		accent := color.RGBA{Accent.R, Accent.G, Accent.B, 255}
		left, right := img.RGBAAt(int(m+(barX+10)*s), y), img.RGBAAt(int(m+(barX+barW-10)*s), y)
		if right != accent || left == accent {
			t.Errorf("scale %v: Arabic bar should fill from the right: left %v right %v", s, left, right)
		}
		cx, cy := (PanelMargin+closeCX)*s, (PanelMargin+closeCY)*s
		if !ar.OverClose(cx, cy) || ar.OverClose(float64(ar.W)-cx, cy) {
			t.Errorf("scale %v: the X click area moved", s)
		}
	}
}
