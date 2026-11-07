package main

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"time"

	"raha/ui"
)

func main() {
	fonts := ui.LoadFonts([]string{"" + os.Getenv("PF") + "/usr/share/fonts/opentype/inter/Inter-Bold.otf"}, []string{os.Getenv("PF2"), "/usr/share/fonts/opentype/inter/Inter-SemiBold.otf"})
	f, _ := os.Open(os.Args[1])
	src, _ := png.Decode(f)
	scale := 1.5
	w, h := ui.PanelSize(scale)
	bg := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(bg, bg.Rect, src, image.Pt(40, 60), draw.Src)
	t0 := time.Now()
	p := ui.NewPanel(scale, bg, fonts, os.Args[3], os.Args[4], t0)
	println("build", time.Since(t0).String())
	p.SetProgress(0.37)
	var img *image.RGBA
	for i := 0; i < 40; i++ {
		img = p.Render(t0.Add(time.Duration(i*16) * time.Millisecond))
	}
	t1 := time.Now()
	img = p.Render(t0.Add(2 * time.Second))
	println("frame", time.Since(t1).String())
	out := image.NewRGBA(img.Rect)
	draw.Draw(out, out.Rect, bg, image.Point{}, draw.Src)
	draw.Draw(out, out.Rect, img, image.Point{}, draw.Over)
	o, _ := os.Create(os.Args[2])
	png.Encode(o, out)
	o.Close()
}
