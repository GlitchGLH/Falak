// helpprev renders the help card as a 2x2 grid (light and dark themes,
// English and Arabic) for a quick visual check.
package main

import (
	"flag"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"raha/ui"
)

const enBody = `Drag a file: the wheel opens around it.
Drop the file on a format to convert it into, or press Shift while dragging to see the tools instead.
While converting, a small panel shows the progress; press its X to stop.
Press the Falak icon next to the clock to open the menu.`

const arBody = `اسحب ملفًا: تُفتح العجلة حوله.
أفلته على صيغة ليُحوَّل إليها، أو اضغط Shift أثناء السحب لتظهر الأدوات بدلًا منها.
أثناء التحويل تظهر لوحة صغيرة فيها التقدّم؛ اضغط X لإيقافه.
اضغط أيقونة فلك بجانب الساعة لفتح القائمة.`

func main() {
	out := flag.String("o", "helpprev.png", "output PNG")
	bold := flag.String("bold", "", "bold font file")
	semi := flag.String("semi", "", "regular font file")
	flag.Parse()
	if *bold == "" || *semi == "" {
		os.Stderr.WriteString("helpprev needs -bold and -semi font files\n")
		os.Exit(2)
	}
	fonts := ui.LoadFonts([]string{*bold}, []string{*semi})
	type cell struct {
		dark, rtl       bool
		title, body, ok string
	}
	cells := []cell{
		{false, false, "How to use", enBody, "OK"},
		{false, true, "طريقة الاستخدام", arBody, "موافق"},
		{true, false, "How to use", enBody, "OK"},
		{true, true, "طريقة الاستخدام", arBody, "موافق"},
	}
	var imgs []*image.RGBA
	for _, c := range cells {
		card := ui.NewHelpCard(1, fonts, c.dark, c.rtl, c.title, c.body, c.ok)
		imgs = append(imgs, card.Render())
	}
	w, h := imgs[0].Rect.Dx(), imgs[0].Rect.Dy()
	gap := 20
	grid := image.NewRGBA(image.Rect(0, 0, w*2+gap*3, h*2+gap*3))
	draw.Draw(grid, grid.Bounds(), &image.Uniform{color.NRGBA{232, 232, 236, 255}}, image.Point{}, draw.Src)
	for i, im := range imgs {
		x, y := gap+(i%2)*(w+gap), gap+(i/2)*(h+gap)
		draw.Draw(grid, image.Rect(x, y, x+w, y+h), im, image.Point{}, draw.Src)
	}
	f, err := os.Create(*out)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, grid); err != nil {
		panic(err)
	}
}
