// Command opacity renders the wheel over a colourful background.
package main

import (
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"
	"strconv"
	"time"

	"raha/ui"
)

func main() {
	if v := os.Getenv("SEGBLUR"); v != "" {
		ui.SegBlur, _ = strconv.ParseFloat(v, 64)
	}
	if v := os.Getenv("SEGSEE"); v != "" {
		ui.SegSeeBlur, _ = strconv.ParseFloat(v, 64)
	}
	if v := os.Getenv("SEGOPA"); v != "" {
		ui.SegOpacity, _ = strconv.ParseFloat(v, 64)
	}
	if v := os.Getenv("SEGVEIL"); v != "" {
		ui.SegVeil, _ = strconv.ParseFloat(v, 64)
	}
	fonts := ui.LoadFonts([]string{"/usr/share/fonts/opentype/inter/Inter-Bold.otf"}, []string{"/usr/share/fonts/opentype/inter/Inter-SemiBold.otf"})
	scale := 1.5
	n := int(ui.LogicalSize * scale)
	bg := image.NewRGBA(image.Rect(0, 0, n, n))
	if len(os.Args) > 2 {
		if f, err := os.Open(os.Args[2]); err == nil {
			src, _, _ := image.Decode(f)
			f.Close()
			draw.Draw(bg, bg.Rect, src, image.Point{}, draw.Src)
		}
	}
	for y := 0; y < n && len(os.Args) <= 2; y++ {
		for x := 0; x < n; x++ {
			fx, fy := float64(x)/float64(n), float64(y)/float64(n)
			r := 120 + 120*math.Sin(fx*7)
			g := 110 + 100*math.Sin(fy*5+1)
			b := 160 + 90*math.Cos((fx+fy)*6)
			bg.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
		}
	}
	items := []ui.Item{}
	for _, f := range []string{"JPG", "WEBP", "HEIC", "AVIF", "TIFF", "BMP", "ICO", "PDF"} {
		items = append(items, ui.Item{ID: f, Label: f, Enabled: true})
	}
	t0 := time.Now().Add(-time.Second)
	w := ui.New(scale, n, n, float64(n)/2, float64(n)/2, ui.FrostBackground(bg, 14*scale, 1.25), false, fonts, t0)
	w.Sharp = ui.FrostBackground(bg, 3*scale, 1.1)
	w.Reveal(t0)
	w.SetItems(items, false, t0)
	img := w.Render(time.Now())
	out := image.NewRGBA(bg.Rect)
	draw.Draw(out, out.Rect, bg, image.Point{}, draw.Src)
	draw.Draw(out, out.Rect, img, image.Point{}, draw.Over)
	f, _ := os.Create(os.Args[1])
	png.Encode(f, out)
	f.Close()
}
