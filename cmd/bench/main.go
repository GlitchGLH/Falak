// Command bench times the first frames of the pop-open at a given scale.
package main

import (
	"fmt"
	"image"
	"math/rand"
	"time"

	"raha/ui"
)

func main() {
	fonts := ui.LoadFonts([]string{"/usr/share/fonts/opentype/inter/Inter-Bold.otf"}, []string{"/usr/share/fonts/opentype/inter/Inter-SemiBold.otf"})
	scale := 1.5
	n := int(ui.LogicalSize * scale)
	bg := image.NewRGBA(image.Rect(0, 0, n, n))
	for i := range bg.Pix {
		bg.Pix[i] = uint8(rand.Intn(255))
	}
	items := []ui.Item{}
	for _, f := range []string{"MP3", "M4A", "WAV", "FLAC", "OGG", "OPUS", "WMA"} {
		items = append(items, ui.Item{ID: f, Label: f, Enabled: true})
	}
	t0 := time.Now()
	w := ui.New(scale, n, n, float64(n)/2, float64(n)/2, ui.FrostBackground(bg, 14*scale, 1.25), false, fonts, t0)
	w.Sharp = ui.FrostBackground(bg, 3*scale, 1.1)
	w.Prepare() // as the app does when Shift is pressed
	w.PrebuildGlass(7, 8, 9)
	time.Sleep(120 * time.Millisecond) // the drag enters the wheel a moment later
	t1 := time.Now()
	w.Reveal(t1)
	w.SetItems(items, false, t1)
	for f := 0; f < 6; f++ {
		s := time.Now()
		w.Render(t1.Add(time.Duration(f) * time.Second / 60))
		fmt.Printf("frame %d: %v\n", f+1, time.Since(s).Round(time.Millisecond/10))
	}
}
