package main

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"time"

	"raha/conv"
	"raha/ui"
)

func main() {
	fonts := ui.LoadFonts([]string{"/usr/share/fonts/opentype/inter/Inter-Bold.otf"}, []string{"/usr/share/fonts/opentype/inter/Inter-SemiBold.otf"})
	f, _ := os.Open("/mnt/user-data/uploads/6bfb5124-image.png")
	src, _ := png.Decode(f)
	files := []string{"a.pdf", "a.png", "a.jpg", "a.mp4"}
	sheet := image.NewRGBA(image.Rect(0, 0, 540*len(files), 540))
	for k, name := range files {
		n := 540
		bg := image.NewRGBA(image.Rect(0, 0, n, n))
		draw.Draw(bg, bg.Rect, src, image.Pt(900, 300), draw.Src)
		t := time.Now()
		w := ui.New(1.5, n, n, 270, 270, ui.FrostBackground(bg, 21, 1.25), false, fonts, t)
		w.Sharp = ui.FrostBackground(bg, 4.5, 1.1)
		w.Reveal(t)
		var items []ui.Item
		for _, it := range conv.ToolItemsFor([]string{name, "b" + name[1:]}) {
			items = append(items, ui.Item{ID: it.ID, Label: it.Label, Icon: it.Icon, Enabled: it.Enabled})
		}
		w.SetItems(items, false, t)
		img := w.Render(t.Add(2 * time.Second))
		out := image.NewRGBA(bg.Rect)
		draw.Draw(out, out.Rect, bg, image.Point{}, draw.Src)
		draw.Draw(out, out.Rect, img, image.Point{}, draw.Over)
		draw.Draw(sheet, image.Rect(540*k, 0, 540*k+540, 540), out, image.Point{}, draw.Src)
	}
	o, _ := os.Create("/tmp/tools.png")
	png.Encode(o, sheet)
	o.Close()
}
