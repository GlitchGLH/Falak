package ui

import (
	"image"
	"testing"
	"time"
)

// TestConcurrentBuild renders frames while glass is prebuilt in the
// background and the panel glass is rebuilt, as the app does.
func TestConcurrentBuild(t *testing.T) {
	fonts := LoadFonts([]string{"/usr/share/fonts/opentype/inter/Inter-Bold.otf"}, []string{"/usr/share/fonts/opentype/inter/Inter-SemiBold.otf"})
	n := 540
	bg := image.NewRGBA(image.Rect(0, 0, n, n))
	for i := range bg.Pix {
		bg.Pix[i] = uint8(i * 13)
	}
	t0 := time.Now()
	w := New(1.5, n, n, 270, 270, FrostBackground(bg, 21, 1.25), false, fonts, t0)
	w.Sharp = FrostBackground(bg, 4.5, 1.1)
	w.Prepare()
	w.PrebuildGlass(7, 8, 9)
	w.Reveal(t0)
	items := []Item{{"a", "MP3", "", true}, {"b", "WAV", "", true}, {"c", "OGG", "", false}, {"d", "FLAC", "", true}, {"e", "M4A", "", true}, {"f", "OPUS", "", true}, {"g", "AAC", "", true}}
	w.SetItems(items, true, t0)
	for i := 0; i < 30; i++ {
		w.SetPointer(float64(200+i), 150)
		w.Render(t0.Add(time.Duration(i*16) * time.Millisecond))
	}
	alt := []Item{{"x", "COMPRESS", "compress", true}, {"y", "ROTATE", "rotate", true}}
	w.Prebuild(alt)
	w.SetItems(alt, true, t0)
	w.Render(t0.Add(time.Second))
	w.Close(t0.Add(time.Second))
	w.Render(t0.Add(1100 * time.Millisecond))

	pw, ph := PanelSize(1.5)
	pbg := image.NewRGBA(image.Rect(0, 0, pw, ph))
	p := NewPanel(1.5, pbg, fonts, "Converting to MP3", "مقطع.wav", t0)
	for i := 0; i < 40; i++ {
		if i%5 == 0 {
			p.Rebackground(pbg)
		}
		p.SetProgress(float64(i) / 40)
		p.SetPointer(30, 30)
		p.Render(t0.Add(time.Duration(i*16) * time.Millisecond))
	}
	time.Sleep(200 * time.Millisecond)
	p.Render(t0.Add(2 * time.Second))
}
