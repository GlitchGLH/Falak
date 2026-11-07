// Command entrance renders the first frames of the wheel's pop-open at 60 fps.
package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"time"

	"raha/ui"
)

func main() {
	fonts := ui.LoadFonts([]string{"/usr/share/fonts/opentype/inter/Inter-Bold.otf"}, []string{"/usr/share/fonts/opentype/inter/Inter-SemiBold.otf"})
	scale := 1.0
	n := int(ui.LogicalSize * scale)
	bg := image.NewRGBA(image.Rect(0, 0, n, n))
	draw.Draw(bg, bg.Rect, image.NewUniform(color.RGBA{250, 250, 250, 255}), image.Point{}, draw.Src)
	items := []ui.Item{}
	for _, f := range []string{"MP3", "M4A", "FLAC", "OGG", "OPUS", "AIFF"} {
		items = append(items, ui.Item{ID: f, Label: f, Enabled: true})
	}
	t0 := time.Now()
	w := ui.New(scale, n, n, float64(n)/2, float64(n)/2, ui.FrostBackground(bg, 14, 1.25), false, fonts, t0)
	w.Sharp = bg
	w.Reveal(t0)
	w.SetItems(items, false, t0)
	cols := 9
	sheet := image.NewRGBA(image.Rect(0, 0, cols*n, n))
	for i := 0; i < cols; i++ {
		now := t0.Add(time.Duration(float64(i) * float64(time.Second) / 60))
		img := w.Render(now)
		cell := image.Rect(i*n, 0, (i+1)*n, n)
		draw.Draw(sheet, cell, bg, image.Point{}, draw.Src)
		draw.Draw(sheet, cell, img, image.Point{}, draw.Over)
	}
	f, _ := os.Create(os.Args[1])
	png.Encode(f, sheet)
	f.Close()
}
